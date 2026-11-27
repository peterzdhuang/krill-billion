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
	ends   []MatchEnd
	rounds int
	maxRound int
}

func (r *recordingEvents) OnRoundStart(round int, prompt data.Prompt, deadlineMs int64, suddenDeath bool) {
	r.rounds++
	if round > r.maxRound {
		r.maxRound = round
	}
}
func (r *recordingEvents) OnLocked(seat string)                          {}
func (r *recordingEvents) OnTeammateDraft(seat, text string)             {}
func (r *recordingEvents) OnRoundResult(out RoundOutcome)                {}
func (r *recordingEvents) OnMatchEnd(end MatchEnd)                       { r.ends = append(r.ends, end) }
func (r *recordingEvents) OnRoomState()                                  {}
func (r *recordingEvents) OnError(playerID, code, message string)        {}

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
		eng.OnDraft("a", "Alpha")
		eng.OnDraft("b", "Alpha")
		eng.OnLock("a")
		eng.OnLock("b")
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
	eng.OnDraft("a", "Alpha")
	eng.OnDraft("b", "Alpha")
	eng.OnLock("a")
	eng.OnLock("b")

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
