package lobby

import "time"

// Player is a lobby participant (may or may not be seated).
type Player struct {
	ID             string
	DisplayName    string
	ReconnectToken string
	Seat           string
	Ready          bool
	IsHost         bool
	Connected      bool
	Conn           Conn
	DropAt         time.Time // zero if connected
}

// Conn is the outbound WS sink for one player (defined here to avoid import cycles).
type Conn interface {
	Send(msgType string, data any) bool
	Close()
	PlayerID() string
	SetPlayerID(id string)
}

// SeatView is the public seat snapshot for lobby_state.
type SeatView struct {
	Seat        string `json:"seat"`
	PlayerID    string `json:"player_id,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	Ready       bool   `json:"ready"`
	Connected   bool   `json:"connected"`
}
