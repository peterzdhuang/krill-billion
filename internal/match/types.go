package match

import (
	"time"

	"github.com/peterzdhuang/krill-billion/internal/data"
)

const (
	AnsweringDur    = 25 * time.Second
	RevealDur       = 10 * time.Second
	RoundTotal      = AnsweringDur + RevealDur // 35s
	MatchBudget     = 600 * time.Second
	MaxScoredRounds = 15
	ReconnectPause  = 15 * time.Second
	MinResumeTime   = 5 * time.Second
	LobbyDropGrace  = 15 * time.Second
	// SubmitGrace is how long after the answering deadline the engine still
	// accepts a final lock-with-text from the client before forcing empty.
	SubmitGrace = 500 * time.Millisecond
)

// Phase is the authoritative match phase.
type Phase string

const (
	PhaseIdle      Phase = "idle"
	PhaseAnswering Phase = "answering"
	PhaseReveal    Phase = "reveal"
	PhasePaused    Phase = "paused"
	PhaseEnded     Phase = "ended"
)

// Seat IDs for 1v1 / 2v2.
const (
	SeatA1 = "A1"
	SeatA2 = "A2"
	SeatB1 = "B1"
	SeatB2 = "B2"
)

// PromptStore is the subset of data.Store the match loop needs.
type PromptStore interface {
	RandomPrompts(n int, exclude []string) []data.Prompt
	Match(promptID, guess string) (entryID string, ok bool)
	Entry(promptID, entryID string) (data.Entry, bool)
}

// Player is a seated competitor.
type Player struct {
	ID          string
	Seat        string
	DisplayName string
	Side        int // 0 = A, 1 = B
}

// SideAnswer is one side's revealed answer for round_result.
type SideAnswer struct {
	Seat         string `json:"seat"`
	Text         string `json:"text"`
	MatchedLabel string `json:"matched_label"`
	Rarity       int    `json:"rarity"`
	Tier         string `json:"tier"`
	Score        int    `json:"score"`
	Valid        bool   `json:"valid"`
	Streak       int    `json:"streak"`
}

// RoundPrompt is the prompt identity carried on round_result / match_end.recap
// so clients can show a GeoGuessr-style reveal and end-of-game summary without
// stashing round_start locally.
type RoundPrompt struct {
	ID       string `json:"id"`
	Text     string `json:"text"`
	Category string `json:"category"`
}

// RoundOutcome is emitted after resolve.
type RoundOutcome struct {
	Round       int           `json:"round"`
	SuddenDeath bool          `json:"sudden_death"`
	Prompt      RoundPrompt   `json:"prompt"`
	Sides       [2]SideAnswer `json:"sides"`
	Damage      int           `json:"damage"`
	Gap         int           `json:"gap"`
	HP          [2]int        `json:"hp"`
	Mult        [2]float64    `json:"multipliers"`
	Winner      int           `json:"winner"` // 0, 1, or -1
	Tie         bool          `json:"tie"`
}

// EndReason for match_end.
type EndReason string

const (
	ReasonKO      EndReason = "ko"
	ReasonForfeit EndReason = "forfeit"
	ReasonCap     EndReason = "cap"
	ReasonDraw    EndReason = "draw"
)

// MatchEnd is the terminal event.
type MatchEnd struct {
	Winner int            `json:"winner"` // 0, 1, or -1 for draw
	Draw   bool           `json:"draw"`
	Reason EndReason      `json:"reason"`
	Recap  []RoundOutcome `json:"recap"`
}

// Events are callbacks the room uses to fan out WS messages.
type Events interface {
	OnRoundStart(round int, prompt data.Prompt, deadlineMs int64, suddenDeath bool)
	OnLocked(seat string)
	OnRoundResult(out RoundOutcome)
	OnMatchEnd(end MatchEnd)
	OnRoomState()
	OnError(playerID, code, message string)
}
