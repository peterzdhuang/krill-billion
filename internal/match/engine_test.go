package match

import (
	"testing"
	"time"

	"github.com/peterzdhuang/krill-billion/internal/data"
)

// tieStore always matches any non-empty guess to the same rarity so rounds tie.
type tieStore struct {
	n int
}

func (s *tieStore) RandomPrompts(n int, exclude []string) []data.Prompt {
	out := make([]data.Prompt, n)
	for i := 0; i < n; i++ {
		s.n++
		id := "p"
		for _, e := range exclude {
			_ = e
		}
		id = "p-" + itoa(s.n)
		out[i] = data.Prompt{
			ID:   id,
			Text: "A thing",
			Entries: []data.Entry{
				{ID: "e1", Label: "Alpha", Rarity: 40},
			},
		}
	}
	return out
}

func (s *tieStore) Match(promptID, guess string) (string, bool) {
	if guess == "" {
		return "", false
	}
	return "e1", true
}

func (s *tieStore) Entry(promptID, entryID string) (data.Entry, bool) {
	if entryID == "e1" {
		return data.Entry{ID: "e1", Label: "Alpha", Rarity: 40}, true
	}
	return data.Entry{}, false
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var d []byte
	for n > 0 {
		d = append([]byte{byte('0' + n%10)}, d...)
		n /= 10
	}
	return string(d)
}

type recordingEvents struct {
	ends     []MatchEnd
	results  []RoundOutcome
	errors   []string
	rounds   int
	maxRound int
}

// selectiveStore matches any non-empty guess except "xyz".
type selectiveStore struct{ tieStore }

func (s *selectiveStore) Match(promptID, guess string) (string, bool) {
	if guess == "" || guess == "xyz" {
		return "", false
	}
	return "e1", true
}

func (r *recordingEvents) OnRoundStart(round int, prompt data.Prompt, deadlineMs int64, suddenDeath bool) {
	r.rounds++
	if round > r.maxRound {
		r.maxRound = round
	}
}
func (r *recordingEvents) OnLocked(seat string)                   {}
func (r *recordingEvents) OnRoundResult(out RoundOutcome)         { r.results = append(r.results, out) }
func (r *recordingEvents) OnMatchEnd(end MatchEnd)                { r.ends = append(r.ends, end) }
func (r *recordingEvents) OnRoomState()                           {}
func (r *recordingEvents) OnError(playerID, code, message string) { r.errors = append(r.errors, code) }

func TestCapEqualHPSuddenDeathDraw(t *testing.T) {
	clock := NewFakeClock(time.Unix(0, 0).UTC())
	ev := &recordingEvents{}
	eng := NewEngine(clock, &tieStore{}, ev)
	eng.AddPlayer(Player{ID: "a", Seat: SeatA1, DisplayName: "A", Side: 0})
	eng.AddPlayer(Player{ID: "b", Seat: SeatB1, DisplayName: "B", Side: 1})
	eng.Start()

	for i := 0; i < MaxScoredRounds; i++ {
		if eng.Phase() != PhaseAnswering {
			t.Fatalf("round %d: want answering, got %s", i+1, eng.Phase())
		}
		eng.OnLock("a", "Alpha")
		eng.OnLock("b", "Alpha")
		if eng.Ended() {
			t.Fatalf("ended early at scored round %d", eng.ScoredRounds())
		}
		if eng.Phase() != PhaseReveal {
			t.Fatalf("round %d: want reveal after locks, got %s", i+1, eng.Phase())
		}
		clock.Advance(RevealDur)
	}

	if eng.ScoredRounds() != MaxScoredRounds {
		t.Fatalf("scored rounds = %d, want %d", eng.ScoredRounds(), MaxScoredRounds)
	}
	if !eng.SuddenDeath() || eng.Phase() != PhaseAnswering {
		t.Fatalf("expected sudden death answering, sudden=%v phase=%s", eng.SuddenDeath(), eng.Phase())
	}
	if eng.Round() > MaxScoredRounds+1 {
		t.Fatalf("round counter %d exceeds %d+1", eng.Round(), MaxScoredRounds)
	}

	// Tied sudden death (same answer) -> draw
	eng.OnLock("a", "Alpha")
	eng.OnLock("b", "Alpha")

	if !eng.Ended() {
		t.Fatal("expected match end after sudden death")
	}
	end := eng.End()
	if !end.Draw || end.Reason != ReasonDraw {
		t.Fatalf("end = %+v, want draw", end)
	}
	if eng.Round() > MaxScoredRounds+1 {
		t.Fatalf("round counter %d exceeds %d+1", eng.Round(), MaxScoredRounds)
	}
	if ev.maxRound > MaxScoredRounds+1 {
		t.Fatalf("observed max round %d exceeds cap+1", ev.maxRound)
	}
}

func TestRoundResultIncludesPrompt(t *testing.T) {
	clock := NewFakeClock(time.Unix(0, 0).UTC())
	ev := &recordingEvents{}
	eng := NewEngine(clock, &tieStore{}, ev)
	eng.AddPlayer(Player{ID: "a", Seat: SeatA1, DisplayName: "A", Side: 0})
	eng.AddPlayer(Player{ID: "b", Seat: SeatB1, DisplayName: "B", Side: 1})
	eng.Start()

	prompt := eng.Prompt()
	eng.OnLock("a", "Alpha")
	eng.OnLock("b", "Alpha")

	if len(ev.results) != 1 {
		t.Fatalf("results = %d, want 1", len(ev.results))
	}
	got := ev.results[0].Prompt
	if got.ID != prompt.ID || got.Text != prompt.Text || got.Category != prompt.Category {
		t.Fatalf("prompt = %+v, want id/text/category from %+v", got, prompt)
	}
}

func TestLockCarriesText(t *testing.T) {
	clock := NewFakeClock(time.Unix(0, 0).UTC())
	ev := &recordingEvents{}
	eng := NewEngine(clock, &tieStore{}, ev)
	eng.AddPlayer(Player{ID: "a", Seat: SeatA1, DisplayName: "A", Side: 0})
	eng.AddPlayer(Player{ID: "b", Seat: SeatB1, DisplayName: "B", Side: 1})
	eng.Start()

	eng.OnLock("a", "Alpha")
	eng.OnLock("b", "Alpha")

	if eng.Draft(SeatA1) != "Alpha" || eng.Draft(SeatB1) != "Alpha" {
		t.Fatalf("drafts = %q/%q, want Alpha/Alpha", eng.Draft(SeatA1), eng.Draft(SeatB1))
	}
	if len(ev.results) != 1 {
		t.Fatalf("results = %d, want 1", len(ev.results))
	}
	for _, side := range ev.results[0].Sides {
		if !side.Valid || side.Text != "Alpha" {
			t.Fatalf("side = %+v, want valid text=Alpha", side)
		}
	}
}

func TestLockRejectsUnknown(t *testing.T) {
	clock := NewFakeClock(time.Unix(0, 0).UTC())
	ev := &recordingEvents{}
	eng := NewEngine(clock, &selectiveStore{}, ev)
	eng.AddPlayer(Player{ID: "a", Seat: SeatA1, DisplayName: "A", Side: 0})
	eng.AddPlayer(Player{ID: "b", Seat: SeatB1, DisplayName: "B", Side: 1})
	eng.Start()

	eng.OnLock("a", "xyz")
	if eng.IsLocked(SeatA1) {
		t.Fatal("xyz locked")
	}
	if len(ev.errors) != 1 || ev.errors[0] != "not_in_bank" {
		t.Fatalf("errors = %v, want [not_in_bank]", ev.errors)
	}
	if eng.Phase() != PhaseAnswering {
		t.Fatalf("phase = %s, want answering", eng.Phase())
	}

	clock.Advance(AnsweringDur + SubmitGrace)
	if len(ev.results) != 1 {
		t.Fatalf("results = %d, want 1", len(ev.results))
	}
	for _, side := range ev.results[0].Sides {
		if side.Seat == SeatA1 && (side.Valid || side.Score != 0 || side.Text != "") {
			t.Fatalf("timeout answer = %+v, want empty invalid score 0", side)
		}
	}
}

func TestLockKeepsRawGuessForReveal(t *testing.T) {
	clock := NewFakeClock(time.Unix(0, 0).UTC())
	ev := &recordingEvents{}
	eng := NewEngine(clock, &tieStore{}, ev)
	eng.AddPlayer(Player{ID: "a", Seat: SeatA1, DisplayName: "A", Side: 0})
	eng.AddPlayer(Player{ID: "b", Seat: SeatB1, DisplayName: "B", Side: 1})
	eng.Start()

	eng.OnLock("a", "alfa")
	eng.OnLock("b", "Alpha")

	if len(ev.results) != 1 {
		t.Fatalf("results = %d, want 1", len(ev.results))
	}
	a := ev.results[0].Sides[0]
	if !a.Valid || a.Text != "alfa" || a.MatchedLabel != "Alpha" {
		t.Fatalf("side A = %+v, want text=alfa matched_label=Alpha", a)
	}
}

func TestTimeoutGraceAcceptsFinalText(t *testing.T) {
	clock := NewFakeClock(time.Unix(0, 0).UTC())
	ev := &recordingEvents{}
	eng := NewEngine(clock, &tieStore{}, ev)
	eng.AddPlayer(Player{ID: "a", Seat: SeatA1, DisplayName: "A", Side: 0})
	eng.AddPlayer(Player{ID: "b", Seat: SeatB1, DisplayName: "B", Side: 1})
	eng.Start()

	// Deadline fires: enter grace, do not resolve yet.
	clock.Advance(AnsweringDur)
	if eng.Phase() != PhaseAnswering {
		t.Fatalf("phase = %s, want answering during grace", eng.Phase())
	}
	if len(ev.results) != 0 {
		t.Fatalf("resolved during grace: %d results", len(ev.results))
	}

	// Final submit arrives within the grace window.
	eng.OnLock("a", "Alpha")
	eng.OnLock("b", "Alpha")

	if len(ev.results) != 1 {
		t.Fatalf("results = %d, want 1 after grace locks", len(ev.results))
	}
	for _, side := range ev.results[0].Sides {
		if !side.Valid || side.Text != "Alpha" {
			t.Fatalf("side = %+v, want valid Alpha from final submit", side)
		}
	}
}

func TestTimeoutGraceFallsBackToEmpty(t *testing.T) {
	clock := NewFakeClock(time.Unix(0, 0).UTC())
	ev := &recordingEvents{}
	eng := NewEngine(clock, &tieStore{}, ev)
	eng.AddPlayer(Player{ID: "a", Seat: SeatA1, DisplayName: "A", Side: 0})
	eng.AddPlayer(Player{ID: "b", Seat: SeatB1, DisplayName: "B", Side: 1})
	eng.Start()

	clock.Advance(AnsweringDur)
	if len(ev.results) != 0 {
		t.Fatal("resolved at deadline without grace")
	}

	clock.Advance(SubmitGrace)
	if len(ev.results) != 1 {
		t.Fatalf("results = %d, want 1 after grace expiry", len(ev.results))
	}
	for _, side := range ev.results[0].Sides {
		if side.Valid || side.Text != "" || side.Score != 0 {
			t.Fatalf("side = %+v, want empty invalid after grace", side)
		}
	}
}

func TestTimeoutGraceAcceptsInvalidText(t *testing.T) {
	clock := NewFakeClock(time.Unix(0, 0).UTC())
	ev := &recordingEvents{}
	eng := NewEngine(clock, &selectiveStore{}, ev)
	eng.AddPlayer(Player{ID: "a", Seat: SeatA1, DisplayName: "A", Side: 0})
	eng.AddPlayer(Player{ID: "b", Seat: SeatB1, DisplayName: "B", Side: 1})
	eng.Start()

	clock.Advance(AnsweringDur)
	// Auto-submit whatever was typed, even if not in bank.
	eng.OnLock("a", "xyz")
	eng.OnLock("b", "")

	if len(ev.errors) != 0 {
		t.Fatalf("grace locks should not reject: %v", ev.errors)
	}
	if len(ev.results) != 1 {
		t.Fatalf("results = %d, want 1", len(ev.results))
	}
	a := ev.results[0].Sides[0]
	if a.Valid || a.Text != "xyz" || a.Score != 0 {
		t.Fatalf("side A = %+v, want invalid text=xyz", a)
	}
}
