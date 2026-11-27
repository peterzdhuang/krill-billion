package lobby

import (
	"testing"

	"github.com/peterzdhuang/krill-billion/internal/data"
	"github.com/peterzdhuang/krill-billion/internal/match"
)

type stubConn struct {
	id   string
	msgs []string
}

func (c *stubConn) Send(msgType string, data any) bool {
	c.msgs = append(c.msgs, msgType)
	return true
}
func (c *stubConn) Close()                {}
func (c *stubConn) PlayerID() string      { return c.id }
func (c *stubConn) SetPlayerID(id string) { c.id = id }

type stubPromptStore struct{}

func (stubPromptStore) RandomPrompts(n int, exclude []string) []data.Prompt { return nil }
func (stubPromptStore) Match(string, string) (string, bool)                 { return "", false }
func (stubPromptStore) Entry(string, string) (data.Entry, bool)             { return data.Entry{}, false }

func TestClaimSeatTaken(t *testing.T) {
	reg := NewRegistry(stubPromptStore{}, match.RealClock{})
	id, _, hostTok, err := reg.Create(Mode1v1, "Host")
	if err != nil {
		t.Fatal(err)
	}
	room, ok := reg.Get(id)
	if !ok {
		t.Fatal("room missing")
	}

	c1 := &stubConn{}
	c2 := &stubConn{}
	j1 := room.Join("Host", match.SeatA1, "", hostTok, c1)
	if j1.ErrCode != "" {
		t.Fatalf("host join: %s %s", j1.ErrCode, j1.ErrMessage)
	}
	j2 := room.Join("Guest", "", "", "", c2)
	if j2.ErrCode != "" {
		t.Fatalf("guest join: %s", j2.ErrCode)
	}
	code, msg := room.ClaimSeat(j2.PlayerID, match.SeatA1)
	if code != "seat_taken" {
		t.Fatalf("claim taken seat: code=%q msg=%q, want seat_taken", code, msg)
	}
}
