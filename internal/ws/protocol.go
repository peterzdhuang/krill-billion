package ws

// Client and server WebSocket message types (mirrored in web/lib/protocol.ts).

const (
	// Client → server
	TypeJoin      = "join"
	TypeClaimSeat = "claim_seat"
	TypeReady     = "ready"
	TypeStart     = "start"
	TypeSetMode   = "set_mode"
	TypeDraft     = "draft"
	TypeLock      = "lock"
	TypePing      = "ping"

	// Server → client
	TypeLobbyState    = "lobby_state"
	TypeRoomState     = "room_state"
	TypeRoundStart    = "round_start"
	TypeTeammateDraft = "teammate_draft"
	TypeLocked        = "locked"
	TypeRoundResult   = "round_result"
	TypeMatchEnd      = "match_end"
	TypeError         = "error"
	TypePong          = "pong"
)

// Envelope is the wire format for every WS message.
type Envelope struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

// --- Client payloads ---

type JoinData struct {
	LobbyID        string `json:"lobby_id"`
	DisplayName    string `json:"display_name"`
	Seat           string `json:"seat,omitempty"`
	ReconnectToken string `json:"reconnect_token,omitempty"`
	HostToken      string `json:"host_token,omitempty"`
}

type ClaimSeatData struct {
	Seat string `json:"seat"`
}

type ReadyData struct{}

type StartData struct{}

type SetModeData struct {
	Mode string `json:"mode"`
}

type DraftData struct {
	Text string `json:"text"`
}

type LockData struct{}

type PingData struct{}

// --- Server payloads ---

type SeatState struct {
	Seat        string `json:"seat"`
	PlayerID    string `json:"player_id,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	Ready       bool   `json:"ready"`
	Connected   bool   `json:"connected"`
}

type LobbyStateData struct {
	Seats          []SeatState `json:"seats"`
	Mode           string      `json:"mode"`
	Link           string      `json:"link"`
	Host           string      `json:"host"`
	HostPlayerID   string      `json:"host_player_id,omitempty"`
	Started        bool        `json:"started"`
	PlayerID       string      `json:"player_id,omitempty"`
	ReconnectToken string      `json:"reconnect_token,omitempty"`
	YourSeat       string      `json:"your_seat,omitempty"`
}

type RoomPlayer struct {
	PlayerID    string `json:"player_id"`
	DisplayName string `json:"display_name"`
	Seat        string `json:"seat"`
	Connected   bool   `json:"connected"`
}

type RoomStateData struct {
	Players     []RoomPlayer `json:"players"`
	HP          [2]int       `json:"hp"`
	Multipliers [2]float64   `json:"multipliers"`
	Phase       string       `json:"phase"`
	Clock       int64        `json:"clock"` // match budget remaining ms
	Round       int          `json:"round"`
	SuddenDeath bool         `json:"sudden_death"`
}

type PromptInfo struct {
	ID       string `json:"id"`
	Text     string `json:"text"`
	Category string `json:"category"`
}

type RoundStartData struct {
	Prompt      PromptInfo `json:"prompt"`
	DeadlineTS  int64      `json:"deadline_ts"` // unix ms
	Round       int        `json:"round"`
	SuddenDeath bool       `json:"sudden_death"`
}

type TeammateDraftData struct {
	Seat string `json:"seat"`
	Text string `json:"text"`
}

type LockedData struct {
	Seat string `json:"seat"`
}

type SideAnswerData struct {
	Seat         string `json:"seat"`
	Text         string `json:"text"`
	MatchedLabel string `json:"matched_label"`
	Rarity       int    `json:"rarity"`
	Tier         string `json:"tier"`
	Score        int    `json:"score"`
	Valid        bool   `json:"valid"`
	Streak       int    `json:"streak"`
}

type RoundResultData struct {
	Round       int               `json:"round"`
	SuddenDeath bool              `json:"sudden_death"`
	Prompt      PromptInfo        `json:"prompt"`
	Sides       [2]SideAnswerData `json:"sides"`
	Damage      int               `json:"damage"`
	Gap         int               `json:"gap"`
	HP          [2]int            `json:"hp"`
	Multipliers [2]float64        `json:"multipliers"`
	Winner      int               `json:"winner"`
	Tie         bool              `json:"tie"`
}

type MatchEndData struct {
	Winner any    `json:"winner"` // 0 | 1 | null
	Draw   bool   `json:"draw"`
	Reason string `json:"reason"` // ko | forfeit | cap | draw
	Recap  any    `json:"recap"`
}

type ErrorData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error codes
const (
	ErrSeatTaken       = "seat_taken"
	ErrNotHost         = "not_host"
	ErrNotReady        = "not_ready"
	ErrBadPhase        = "bad_phase"
	ErrDuplicateAnswer = "duplicate_answer"
	ErrRateLimited     = "rate_limited"
	ErrNotInBank       = "not_in_bank"
)
