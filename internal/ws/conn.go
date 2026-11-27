package ws

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
)

const (
	writeBufSize   = 32
	pingInterval   = 15 * time.Second
	pongWait       = 2 * pingInterval // drop after 2 missed server pings
	maxMessageSize = 32 << 10
)

// Conn wraps a websocket with a bounded write pump.
type Conn struct {
	ws       *websocket.Conn
	send     chan []byte
	playerID atomic.Value // string
	closed   atomic.Bool
	missed   atomic.Int32
	onClose  func()
	mu       sync.Mutex
}

func newConn(ws *websocket.Conn, onClose func()) *Conn {
	c := &Conn{
		ws:      ws,
		send:    make(chan []byte, writeBufSize),
		onClose: onClose,
	}
	c.playerID.Store("")
	return c
}

func (c *Conn) PlayerID() string {
	v, _ := c.playerID.Load().(string)
	return v
}

func (c *Conn) SetPlayerID(id string) { c.playerID.Store(id) }

func (c *Conn) Send(msgType string, data any) bool {
	if c.closed.Load() {
		return false
	}
	b, err := json.Marshal(Envelope{Type: msgType, Data: data})
	if err != nil {
		return false
	}
	select {
	case c.send <- b:
		return true
	default:
		// slow client — drop connection
		c.Close()
		return false
	}
}

func (c *Conn) Close() {
	if c.closed.Swap(true) {
		return
	}
	close(c.send)
	_ = c.ws.Close(websocket.StatusNormalClosure, "")
	if c.onClose != nil {
		c.onClose()
	}
}

func (c *Conn) writePump(ctx context.Context) {
	defer c.Close()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-c.send:
			if !ok {
				return
			}
			wctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := c.ws.Write(wctx, websocket.MessageText, msg)
			cancel()
			if err != nil {
				return
			}
		}
	}
}

func (c *Conn) heartbeat(ctx context.Context) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if c.closed.Load() {
				return
			}
			// Ping waits for a pong; consecutive failures count as missed heartbeats.
			pctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := c.ws.Ping(pctx)
			cancel()
			if err != nil {
				if c.missed.Add(1) >= 2 {
					c.Close()
					return
				}
				continue
			}
			c.missed.Store(0)
		}
	}
}

func (c *Conn) noteActivity() { c.missed.Store(0) }
