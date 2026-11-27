package lobby

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/peterzdhuang/krill-billion/internal/match"
)

// PromptStore is the data dependency for matches started from lobbies.
type PromptStore = match.PromptStore

// Registry holds lobbies behind a mutex; each lobby runs its own goroutine.
type Registry struct {
	mu     sync.Mutex
	rooms  map[string]*Room
	store  PromptStore
	clock  match.Clock
	nowFn  func() time.Time
	idleGC time.Duration
	endGC  time.Duration
}

func NewRegistry(store PromptStore, clock match.Clock) *Registry {
	if clock == nil {
		clock = match.RealClock{}
	}
	r := &Registry{
		rooms:  make(map[string]*Room),
		store:  store,
		clock:  clock,
		nowFn:  clock.Now,
		idleGC: 30 * time.Minute,
		endGC:  5 * time.Minute,
	}
	go r.gcLoop()
	return r
}

// Create opens a new lobby. Host sits after WS join with host_token.
func (r *Registry) Create(mode Mode, displayName string) (lobbyID, link, hostToken string, err error) {
	if mode != Mode1v1 && mode != Mode2v2 {
		return "", "", "", fmt.Errorf("invalid mode")
	}
	if displayName == "" {
		return "", "", "", fmt.Errorf("display_name required")
	}
	id, err := randomID(8)
	if err != nil {
		return "", "", "", err
	}
	token, err := randomID(16)
	if err != nil {
		return "", "", "", err
	}
	room := newRoom(id, mode, token, r.store, r.clock)
	r.mu.Lock()
	r.rooms[id] = room
	r.mu.Unlock()
	go room.loop()
	return id, "/play/" + id, token, nil
}

func (r *Registry) Get(id string) (*Room, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	room, ok := r.rooms[id]
	return room, ok
}

func (r *Registry) remove(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.rooms, id)
}

func (r *Registry) gcLoop() {
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for range t.C {
		r.mu.Lock()
		now := r.nowFn()
		var dead []*Room
		for _, room := range r.rooms {
			if room.shouldGC(now, r.idleGC, r.endGC) {
				dead = append(dead, room)
			}
		}
		r.mu.Unlock()
		for _, room := range dead {
			room.shutdown()
			r.remove(room.id)
		}
	}
}

func randomID(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	// URL-safe, truncate to n chars
	s := base64.RawURLEncoding.EncodeToString(b)
	if len(s) > n {
		s = s[:n]
	}
	return s, nil
}
