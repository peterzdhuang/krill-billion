package match

import (
	"strings"
	"time"

	"github.com/peterzdhuang/krill-billion/internal/data"
	"github.com/peterzdhuang/krill-billion/internal/game"
)

// Engine runs one authoritative 1v1 match. All methods must be called from
// a single owning goroutine (the lobby room actor).
type Engine struct {
	clock  Clock
	store  PromptStore
	events Events

	players map[string]*Player // playerID -> player
	bySeat  map[string]*Player

	phase        Phase
	round        int // current round number (1-based, includes sudden death)
	scoredRounds int // completed scored rounds (0..MaxScoredRounds)
	suddenDeath  bool
	sdReason     EndReason // ReasonKO or ReasonCap — why SD was entered
	matchStart   time.Time
	deadline     time.Time
	prompt       data.Prompt
	usedPrompts  []string
	drafts       map[string]string // seat -> text
	locked       map[string]bool
	lastDraftAt  map[string]time.Time
	hp           [2]int
	mult         [2]float64
	recap        []RoundOutcome
	ended        bool
	end          MatchEnd

	// reconnect / forfeit
	pausesUsed     map[string]int  // playerID -> pause count
	pendingForfeit map[string]bool // second drop: forfeit after current round
	pausedFor      string          // playerID currently paused for
	pauseDeadline  time.Time
	savedPhase     Phase
	savedRemain    time.Duration

	answeringTimer Timer
	revealTimer    Timer
	pauseTimer     Timer
}

// NewEngine constructs a match engine. Call Start after seating players.
func NewEngine(clock Clock, store PromptStore, events Events) *Engine {
	return &Engine{
		clock:          clock,
		store:          store,
		events:         events,
		players:        make(map[string]*Player),
		bySeat:         make(map[string]*Player),
		phase:          PhaseIdle,
		drafts:         make(map[string]string),
		locked:         make(map[string]bool),
		lastDraftAt:    make(map[string]time.Time),
		pausesUsed:     make(map[string]int),
		pendingForfeit: make(map[string]bool),
	}
}

func (e *Engine) Phase() Phase              { return e.phase }
func (e *Engine) Round() int                { return e.round }
func (e *Engine) ScoredRounds() int         { return e.scoredRounds }
func (e *Engine) SuddenDeath() bool         { return e.suddenDeath }
func (e *Engine) Ended() bool               { return e.ended }
func (e *Engine) End() MatchEnd             { return e.end }
func (e *Engine) HP() [2]int                { return e.hp }
func (e *Engine) Mult() [2]float64          { return e.mult }
func (e *Engine) Prompt() data.Prompt       { return e.prompt }
func (e *Engine) Deadline() time.Time       { return e.deadline }
func (e *Engine) MatchStart() time.Time     { return e.matchStart }
func (e *Engine) Draft(seat string) string  { return e.drafts[seat] }
func (e *Engine) IsLocked(seat string) bool { return e.locked[seat] }

// RemainingMs returns answering time left (unix-style ms remaining), min 0.
func (e *Engine) RemainingMs() int64 {
	rem := e.deadline.Sub(e.clock.Now())
	if rem < 0 {
		return 0
	}
	return rem.Milliseconds()
}

// ResumeDeadlineMs for reconnect: remaining answering time, at least MinResumeTime.
func (e *Engine) ResumeDeadlineMs() int64 {
	rem := e.deadline.Sub(e.clock.Now())
	if rem < MinResumeTime {
		rem = MinResumeTime
	}
	return e.clock.Now().Add(rem).UnixMilli()
}

func (e *Engine) AddPlayer(p Player) {
	cp := p
	e.players[p.ID] = &cp
	e.bySeat[p.Seat] = &cp
}

// Start begins the match (first round_start).
func (e *Engine) Start() {
	e.hp = [2]int{game.HP1v1, game.HP1v1}
	e.mult = [2]float64{1.0, 1.0}
	e.matchStart = e.clock.Now()
	e.beginRound(false)
}

func (e *Engine) beginRound(sudden bool) {
	e.stopTimers()
	e.suddenDeath = sudden
	e.round++
	e.phase = PhaseAnswering
	e.drafts = make(map[string]string)
	e.locked = make(map[string]bool)
	e.lastDraftAt = make(map[string]time.Time)

	prompts := e.store.RandomPrompts(1, e.usedPrompts)
	if len(prompts) == 0 {
		e.finish(MatchEnd{Winner: -1, Draw: true, Reason: ReasonDraw, Recap: e.recap})
		return
	}
	e.prompt = prompts[0]
	e.usedPrompts = append(e.usedPrompts, e.prompt.ID)
	e.deadline = e.clock.Now().Add(AnsweringDur)

	e.events.OnRoundStart(e.round, e.prompt, e.deadline.UnixMilli(), sudden)
	e.events.OnRoomState()

	e.answeringTimer = e.clock.AfterFunc(AnsweringDur, func() {
		e.onAnsweringTimeout()
	})
}

func (e *Engine) stopTimers() {
	if e.answeringTimer != nil {
		e.answeringTimer.Stop()
		e.answeringTimer = nil
	}
	if e.revealTimer != nil {
		e.revealTimer.Stop()
		e.revealTimer = nil
	}
	if e.pauseTimer != nil {
		e.pauseTimer.Stop()
		e.pauseTimer = nil
	}
}

func (e *Engine) activeSeats() []string {
	return []string{SeatA1, SeatB1}
}

func (e *Engine) OnDraft(playerID, text string) {
	if e.ended || e.phase != PhaseAnswering {
		e.events.OnError(playerID, "bad_phase", "not accepting drafts")
		return
	}
	p := e.players[playerID]
	if p == nil {
		return
	}
	if e.locked[p.Seat] {
		return // ignore late draft after lock
	}
	now := e.clock.Now()
	if last, ok := e.lastDraftAt[p.Seat]; ok && now.Sub(last) < 50*time.Millisecond {
		e.events.OnError(playerID, "rate_limited", "draft too fast")
		return
	}
	e.lastDraftAt[p.Seat] = now
	e.drafts[p.Seat] = text
	// teammate_draft only for same side other seats (2v2); 1v1 has none
	for seat, other := range e.bySeat {
		if other == nil || other.ID == playerID {
			continue
		}
		if other.Side == p.Side {
			e.events.OnTeammateDraft(seat, text)
		}
	}
}

func (e *Engine) OnLock(playerID string) {
	if e.ended || e.phase != PhaseAnswering {
		e.events.OnError(playerID, "bad_phase", "not accepting locks")
		return
	}
	p := e.players[playerID]
	if p == nil {
		return
	}
	if e.locked[p.Seat] {
		return
	}
	// An empty lock is a pass. Anything else must be a real entry (or a typo
	// of one). A miss stays unlocked so the player can retype before the buzzer.
	if text := strings.TrimSpace(e.drafts[p.Seat]); text != "" {
		if _, ok := e.store.Match(e.prompt.ID, text); !ok {
			e.events.OnError(playerID, "not_in_bank", "not in this category")
			return
		}
	}
	e.locked[p.Seat] = true
	e.events.OnLocked(p.Seat)
	if e.allLocked() {
		e.resolveAnswering()
	}
}

func (e *Engine) allLocked() bool {
	for _, seat := range e.activeSeats() {
		if !e.locked[seat] {
			return false
		}
	}
	return true
}

func (e *Engine) onAnsweringTimeout() {
	if e.ended || e.phase != PhaseAnswering {
		return
	}
	for _, seat := range e.activeSeats() {
		if !e.locked[seat] {
			e.locked[seat] = true
			e.events.OnLocked(seat)
		}
	}
	e.resolveAnswering()
}

func (e *Engine) resolveAnswering() {
	if e.answeringTimer != nil {
		e.answeringTimer.Stop()
		e.answeringTimer = nil
	}
	e.phase = PhaseReveal

	var sides [2]SideAnswer
	var scores [2]int
	for i, seat := range e.activeSeats() {
		text := e.drafts[seat]
		sa := SideAnswer{Seat: seat, Text: text}
		if text != "" {
			if eid, ok := e.store.Match(e.prompt.ID, text); ok {
				if ent, ok := e.store.Entry(e.prompt.ID, eid); ok {
					sa.Valid = true
					sa.MatchedLabel = ent.Label
					sa.Rarity = ent.Rarity
					sa.Tier = data.TierFor(ent.Rarity).String()
					sa.Score = game.PlayerScore(ent.Rarity)
				}
			}
		}
		if !sa.Valid {
			sa.Rarity = -1
			sa.Score = 0
			sa.Tier = ""
		}
		sides[i] = sa
		scores[i] = sa.Score
	}

	var out RoundOutcome
	out.Round = e.round
	out.SuddenDeath = e.suddenDeath
	out.Sides = sides

	if e.suddenDeath {
		w, draw := game.SuddenDeath(scores[0], scores[1])
		out.HP = e.hp
		out.Mult = e.mult
		out.Winner = w
		out.Tie = draw
		e.recap = append(e.recap, out)
		e.events.OnRoundResult(out)
		if draw {
			e.finish(MatchEnd{Winner: -1, Draw: true, Reason: ReasonDraw, Recap: e.recap})
		} else {
			reason := e.sdReason
			if reason == "" {
				reason = ReasonCap
			}
			e.finish(MatchEnd{Winner: w, Draw: false, Reason: reason, Recap: e.recap})
		}
		return
	}

	res := game.Resolve(
		game.SideState{Score: scores[0], HP: e.hp[0], Mult: e.mult[0]},
		game.SideState{Score: scores[1], HP: e.hp[1], Mult: e.mult[1]},
	)
	e.hp = res.HPAfter
	e.mult = res.MultAfter
	out.Damage = res.Damage
	out.Gap = res.Gap
	out.HP = e.hp
	out.Mult = e.mult
	out.Winner = res.Winner
	out.Tie = res.Tie
	e.scoredRounds++
	e.recap = append(e.recap, out)
	e.events.OnRoundResult(out)
	e.events.OnRoomState()

	// Check pending forfeits after round
	if e.checkPendingForfeits() {
		return
	}

	// KO
	aDead := e.hp[0] <= 0
	bDead := e.hp[1] <= 0
	if aDead && bDead {
		e.scheduleRevealThen(func() {
			e.sdReason = ReasonKO
			e.beginRound(true)
		})
		return
	}
	if aDead {
		e.scheduleRevealThen(func() {
			e.finish(MatchEnd{Winner: 1, Reason: ReasonKO, Recap: e.recap})
		})
		return
	}
	if bDead {
		e.scheduleRevealThen(func() {
			e.finish(MatchEnd{Winner: 0, Reason: ReasonKO, Recap: e.recap})
		})
		return
	}

	// Cap / budget
	if e.scoredRounds >= MaxScoredRounds {
		e.scheduleRevealThen(e.afterCap)
		return
	}
	e.scheduleRevealThen(e.maybeNextScoredRound)
}

func (e *Engine) scheduleRevealThen(next func()) {
	e.phase = PhaseReveal
	e.deadline = e.clock.Now().Add(RevealDur)
	e.revealTimer = e.clock.AfterFunc(RevealDur, func() {
		if e.ended || e.phase != PhaseReveal {
			return
		}
		next()
	})
}

func (e *Engine) afterCap() {
	w, needSD := game.CapWinner(e.hp[0], e.hp[1])
	if needSD {
		e.sdReason = ReasonCap
		e.beginRound(true)
		return
	}
	e.finish(MatchEnd{Winner: w, Reason: ReasonCap, Recap: e.recap})
}

func (e *Engine) maybeNextScoredRound() {
	elapsed := e.clock.Now().Sub(e.matchStart)
	remain := MatchBudget - elapsed
	if remain < RoundTotal {
		// time cap: treat like round cap by HP
		e.afterCap()
		return
	}
	e.beginRound(false)
}

func (e *Engine) checkPendingForfeits() bool {
	for pid, pending := range e.pendingForfeit {
		if !pending {
			continue
		}
		p := e.players[pid]
		if p == nil {
			continue
		}
		winner := 1 - p.Side
		e.scheduleRevealThen(func() {
			e.finish(MatchEnd{Winner: winner, Reason: ReasonForfeit, Recap: e.recap})
		})
		return true
	}
	return false
}

func (e *Engine) finish(end MatchEnd) {
	if e.ended {
		return
	}
	e.stopTimers()
	e.ended = true
	e.phase = PhaseEnded
	e.end = end
	e.events.OnMatchEnd(end)
	e.events.OnRoomState()
}

// OnDisconnect handles mid-match socket drop for playerID.
// Returns whether a pause was started.
func (e *Engine) OnDisconnect(playerID string) {
	if e.ended {
		return
	}
	p := e.players[playerID]
	if p == nil {
		return
	}

	// After lock: answer still counts; no pause (does not consume the one pause).
	if e.phase == PhaseAnswering && e.locked[p.Seat] {
		return
	}

	used := e.pausesUsed[playerID]
	if used >= 1 {
		// Second drop: immediate forfeit after the round
		e.pendingForfeit[playerID] = true
		if e.phase == PhaseAnswering {
			// auto-lock empty/current draft so round can finish
			if !e.locked[p.Seat] {
				e.locked[p.Seat] = true
				e.events.OnLocked(p.Seat)
			}
			if e.allLocked() {
				e.resolveAnswering()
			}
		}
		return
	}

	// First drop before lock: pause up to 15s
	if e.phase != PhaseAnswering && e.phase != PhaseReveal {
		return
	}
	if e.phase == PhaseReveal {
		// pause only if before lock point — reveal is after lock, so no pause
		return
	}

	e.pausesUsed[playerID]++
	e.savedPhase = e.phase
	e.savedRemain = e.deadline.Sub(e.clock.Now())
	if e.answeringTimer != nil {
		e.answeringTimer.Stop()
		e.answeringTimer = nil
	}
	e.phase = PhasePaused
	e.pausedFor = playerID
	e.pauseDeadline = e.clock.Now().Add(ReconnectPause)
	e.events.OnRoomState()
	e.pauseTimer = e.clock.AfterFunc(ReconnectPause, func() {
		e.onPauseExpired(playerID)
	})
}

func (e *Engine) onPauseExpired(playerID string) {
	if e.ended || e.phase != PhasePaused || e.pausedFor != playerID {
		return
	}
	p := e.players[playerID]
	winner := 0
	if p != nil {
		winner = 1 - p.Side
	}
	e.finish(MatchEnd{Winner: winner, Reason: ReasonForfeit, Recap: e.recap})
}

// OnReconnect restores a paused player. Returns true if match was paused for them.
func (e *Engine) OnReconnect(playerID string) bool {
	if e.phase != PhasePaused || e.pausedFor != playerID {
		return false
	}
	if e.pauseTimer != nil {
		e.pauseTimer.Stop()
		e.pauseTimer = nil
	}
	e.pausedFor = ""
	e.phase = PhaseAnswering
	remain := e.savedRemain
	if remain < MinResumeTime {
		remain = MinResumeTime
	}
	e.deadline = e.clock.Now().Add(remain)
	e.events.OnRoomState()
	e.events.OnRoundStart(e.round, e.prompt, e.deadline.UnixMilli(), e.suddenDeath)
	e.answeringTimer = e.clock.AfterFunc(remain, func() {
		e.onAnsweringTimeout()
	})
	return true
}

// ClockMs remaining in the match budget.
func (e *Engine) MatchClockRemainingMs() int64 {
	if e.matchStart.IsZero() {
		return MatchBudget.Milliseconds()
	}
	rem := MatchBudget - e.clock.Now().Sub(e.matchStart)
	if rem < 0 {
		return 0
	}
	return rem.Milliseconds()
}
