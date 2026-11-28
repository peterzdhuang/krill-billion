package lobby

import (
	"time"

	"github.com/peterzdhuang/krill-billion/internal/data"
	"github.com/peterzdhuang/krill-billion/internal/match"
)

type command func()

// Room owns lobby + match state on a single goroutine.
type Room struct {
	id        string
	mode      Mode
	hostToken string
	link      string
	store     PromptStore
	clock     match.Clock

	inbox chan command
	quit  chan struct{}

	players map[string]*Player // by player id
	byToken map[string]*Player // reconnect token
	hostID  string

	match   *match.Engine
	started bool
	endedAt time.Time
	touchAt time.Time

	dropTimers map[string]match.Timer // playerID -> lobby drop grace
}

func newRoom(id string, mode Mode, hostToken string, store PromptStore, clock match.Clock) *Room {
	r := &Room{
		id:         id,
		mode:       mode,
		hostToken:  hostToken,
		link:       "/play/" + id,
		store:      store,
		inbox:      make(chan command, 128),
		quit:       make(chan struct{}),
		players:    make(map[string]*Player),
		byToken:    make(map[string]*Player),
		dropTimers: make(map[string]match.Timer),
	}
	// Marshal timer callbacks onto the room actor goroutine.
	r.clock = &actorClock{inner: clock, room: r}
	r.touchAt = r.clock.Now()
	return r
}

// actorClock ensures AfterFunc callbacks run on the room goroutine.
type actorClock struct {
	inner match.Clock
	room  *Room
}

func (c *actorClock) Now() time.Time { return c.inner.Now() }

func (c *actorClock) AfterFunc(d time.Duration, f func()) match.Timer {
	return c.inner.AfterFunc(d, func() {
		c.room.do(f)
	})
}

func (r *Room) loop() {
	for {
		select {
		case <-r.quit:
			return
		case fn := <-r.inbox:
			fn()
		}
	}
}

func (r *Room) do(fn command) {
	select {
	case <-r.quit:
	case r.inbox <- fn:
	}
}

func (r *Room) shutdown() {
	select {
	case <-r.quit:
	default:
		close(r.quit)
	}
}

func (r *Room) shouldGC(now time.Time, idle, afterEnd time.Duration) bool {
	res := make(chan bool, 1)
	r.do(func() {
		if !r.endedAt.IsZero() {
			res <- now.Sub(r.endedAt) >= afterEnd
			return
		}
		res <- now.Sub(r.touchAt) >= idle
	})
	select {
	case v := <-res:
		return v
	case <-time.After(2 * time.Second):
		return false
	}
}

func (r *Room) touch() { r.touchAt = r.clock.Now() }

// JoinResult is returned to the WS layer after join.
type JoinResult struct {
	PlayerID       string
	ReconnectToken string
	Seat           string
	IsHost         bool
	ErrCode        string
	ErrMessage     string
}

func (r *Room) Join(displayName, seat, reconnectToken, hostToken string, conn Conn) JoinResult {
	resCh := make(chan JoinResult, 1)
	r.do(func() {
		r.touch()
		resCh <- r.join(displayName, seat, reconnectToken, hostToken, conn)
	})
	return <-resCh
}

func (r *Room) join(displayName, seat, reconnectToken, hostToken string, conn Conn) JoinResult {
	// Reconnect
	if reconnectToken != "" {
		if p, ok := r.byToken[reconnectToken]; ok {
			if t := r.dropTimers[p.ID]; t != nil {
				t.Stop()
				delete(r.dropTimers, p.ID)
			}
			p.Connected = true
			p.DropAt = time.Time{}
			p.Conn = conn
			conn.SetPlayerID(p.ID)
			if displayName != "" {
				p.DisplayName = displayName
			}
			if r.match != nil && !r.match.Ended() {
				if !r.match.OnReconnect(p.ID) {
					// Not paused for this player; still sync room + live prompt.
					r.sendRoomTo(p)
				}
			}
			r.broadcastLobby()
			return JoinResult{
				PlayerID:       p.ID,
				ReconnectToken: p.ReconnectToken,
				Seat:           p.Seat,
				IsHost:         p.IsHost,
			}
		}
	}

	id, err := randomID(12)
	if err != nil {
		return JoinResult{ErrCode: "internal", ErrMessage: "id gen failed"}
	}
	tok, err := randomID(24)
	if err != nil {
		return JoinResult{ErrCode: "internal", ErrMessage: "token gen failed"}
	}
	p := &Player{
		ID:             id,
		DisplayName:    displayName,
		ReconnectToken: tok,
		Connected:      true,
		Conn:           conn,
	}
	isHost := false
	if r.hostID == "" && hostToken != "" && hostToken == r.hostToken {
		isHost = true
		p.IsHost = true
		r.hostID = id
	}

	r.players[id] = p
	r.byToken[tok] = p
	conn.SetPlayerID(id)

	wantSeat := seat
	if wantSeat == "" && isHost {
		wantSeat = match.SeatA1
	}
	if wantSeat != "" {
		if code, msg := r.claimSeat(p, wantSeat); code != "" {
			// stay unseated but joined
			_ = code
			_ = msg
		}
	}

	r.broadcastLobby()
	return JoinResult{
		PlayerID:       id,
		ReconnectToken: tok,
		Seat:           p.Seat,
		IsHost:         p.IsHost,
	}
}

func (r *Room) ClaimSeat(playerID, seat string) (code, message string) {
	res := make(chan [2]string, 1)
	r.do(func() {
		r.touch()
		p := r.players[playerID]
		if p == nil {
			res <- [2]string{"bad_phase", "unknown player"}
			return
		}
		if r.started {
			res <- [2]string{"bad_phase", "match already started"}
			return
		}
		code, msg := r.claimSeat(p, seat)
		if code == "" {
			r.broadcastLobby()
		}
		res <- [2]string{code, msg}
	})
	out := <-res
	return out[0], out[1]
}

func (r *Room) claimSeat(p *Player, seat string) (code, message string) {
	if seat == "" {
		// leave seat
		p.Seat = ""
		p.Ready = false
		return "", ""
	}
	if !ValidSeat(r.mode, seat) {
		return "bad_phase", "invalid seat for mode"
	}
	for _, other := range r.players {
		if other.ID != p.ID && other.Seat == seat {
			return "seat_taken", "seat already claimed"
		}
	}
	if p.Seat != seat {
		p.Ready = false // unready on move
	}
	p.Seat = seat
	return "", ""
}

func (r *Room) Ready(playerID string) (code, message string) {
	res := make(chan [2]string, 1)
	r.do(func() {
		r.touch()
		p := r.players[playerID]
		if p == nil {
			res <- [2]string{"bad_phase", "unknown player"}
			return
		}
		if r.started {
			res <- [2]string{"bad_phase", "match already started"}
			return
		}
		if p.Seat == "" {
			res <- [2]string{"not_ready", "claim a seat first"}
			return
		}
		p.Ready = true
		r.broadcastLobby()
		res <- [2]string{"", ""}
	})
	out := <-res
	return out[0], out[1]
}

func (r *Room) SetMode(playerID string, mode Mode) (code, message string) {
	res := make(chan [2]string, 1)
	r.do(func() {
		r.touch()
		p := r.players[playerID]
		if p == nil || !p.IsHost {
			res <- [2]string{"not_host", "only host can change mode"}
			return
		}
		if r.started {
			res <- [2]string{"bad_phase", "match already started"}
			return
		}
		if mode != Mode1v1 && mode != Mode2v2 {
			res <- [2]string{"bad_phase", "invalid mode"}
			return
		}
		r.mode = mode
		allowed := make(map[string]bool)
		for _, s := range SeatsFor(mode) {
			allowed[s] = true
		}
		for _, pl := range r.players {
			if pl.Seat != "" && !allowed[pl.Seat] {
				pl.Seat = ""
				pl.Ready = false
			}
		}
		r.broadcastLobby()
		res <- [2]string{"", ""}
	})
	out := <-res
	return out[0], out[1]
}

func (r *Room) Start(playerID string) (code, message string) {
	res := make(chan [2]string, 1)
	r.do(func() {
		r.touch()
		p := r.players[playerID]
		if p == nil || !p.IsHost {
			res <- [2]string{"not_host", "only host can start"}
			return
		}
		if r.started {
			res <- [2]string{"bad_phase", "already started"}
			return
		}
		seats := SeatsFor(r.mode)
		filled := make(map[string]*Player)
		for _, pl := range r.players {
			if pl.Seat != "" {
				filled[pl.Seat] = pl
			}
		}
		for _, s := range seats {
			pl := filled[s]
			if pl == nil {
				res <- [2]string{"not_ready", "all seats must be filled"}
				return
			}
			if !pl.Ready {
				res <- [2]string{"not_ready", "all players must be ready"}
				return
			}
		}
		r.started = true
		eng := match.NewEngine(r.clock, r.store, &roomEvents{room: r})
		for _, s := range seats {
			pl := filled[s]
			eng.AddPlayer(match.Player{
				ID:          pl.ID,
				Seat:        pl.Seat,
				DisplayName: pl.DisplayName,
				Side:        SideOf(pl.Seat),
			})
		}
		r.match = eng
		r.broadcastLobby()
		eng.Start()
		res <- [2]string{"", ""}
	})
	out := <-res
	return out[0], out[1]
}

func (r *Room) Lock(playerID, text string) {
	r.do(func() {
		r.touch()
		if r.match == nil {
			r.sendErr(playerID, "bad_phase", "match not started")
			return
		}
		r.match.OnLock(playerID, text)
	})
}

func (r *Room) Disconnect(playerID string) {
	r.do(func() {
		p := r.players[playerID]
		if p == nil {
			return
		}
		p.Connected = false
		p.Conn = nil
		p.DropAt = r.clock.Now()
		r.touch()

		if r.match != nil && !r.match.Ended() {
			r.match.OnDisconnect(playerID)
			r.broadcastLobby()
			return
		}
		// Pre-start: free seat after 15s
		if r.started {
			r.broadcastLobby()
			return
		}
		if t := r.dropTimers[playerID]; t != nil {
			t.Stop()
		}
		pid := playerID
		r.dropTimers[playerID] = r.clock.AfterFunc(match.LobbyDropGrace, func() {
			r.do(func() {
				pl := r.players[pid]
				if pl == nil || pl.Connected {
					return
				}
				delete(r.byToken, pl.ReconnectToken)
				delete(r.players, pid)
				delete(r.dropTimers, pid)
				if r.hostID == pid {
					r.hostID = ""
					// promote another seated player if any
					for _, other := range r.players {
						other.IsHost = true
						r.hostID = other.ID
						break
					}
				}
				r.broadcastLobby()
			})
		})
		r.broadcastLobby()
	})
}

func (r *Room) sendErr(playerID, code, message string) {
	p := r.players[playerID]
	if p == nil || p.Conn == nil {
		return
	}
	p.Conn.Send("error", map[string]string{"code": code, "message": message})
}

func (r *Room) broadcastLobby() {
	data := r.lobbyStateData("")
	for _, p := range r.players {
		if p.Conn == nil {
			continue
		}
		view := r.lobbyStateData(p.ID)
		p.Conn.Send("lobby_state", view)
		_ = data
	}
}

func (r *Room) lobbyStateData(forPlayer string) map[string]any {
	seats := make([]SeatView, 0)
	for _, s := range SeatsFor(r.mode) {
		sv := SeatView{Seat: s}
		for _, p := range r.players {
			if p.Seat == s {
				sv.PlayerID = p.ID
				sv.DisplayName = p.DisplayName
				sv.Ready = p.Ready
				sv.Connected = p.Connected
				break
			}
		}
		seats = append(seats, sv)
	}
	hostName := ""
	if h := r.players[r.hostID]; h != nil {
		hostName = h.DisplayName
	}
	out := map[string]any{
		"seats":          seats,
		"mode":           string(r.mode),
		"link":           r.link,
		"host":           hostName,
		"host_player_id": r.hostID,
		"started":        r.started,
	}
	if forPlayer != "" {
		if p := r.players[forPlayer]; p != nil {
			out["player_id"] = p.ID
			out["reconnect_token"] = p.ReconnectToken
			out["your_seat"] = p.Seat
		}
	}
	return out
}

func (r *Room) sendRoomTo(p *Player) {
	if p == nil || p.Conn == nil || r.match == nil {
		return
	}
	p.Conn.Send("room_state", r.roomStateData())
	if r.match.Phase() == match.PhaseAnswering || r.match.Phase() == match.PhasePaused {
		pr := r.match.Prompt()
		deadline := r.match.Deadline().UnixMilli()
		if r.match.Phase() == match.PhaseAnswering {
			// ensure min 5s on reconnect path handled by engine
			deadline = r.match.ResumeDeadlineMs()
		}
		p.Conn.Send("round_start", map[string]any{
			"prompt": map[string]any{
				"id":       pr.ID,
				"text":     pr.Text,
				"category": pr.Category,
			},
			"deadline_ts":  deadline,
			"round":        r.match.Round(),
			"sudden_death": r.match.SuddenDeath(),
		})
	}
}

func (r *Room) roomStateData() map[string]any {
	players := make([]map[string]any, 0)
	for _, p := range r.players {
		if p.Seat == "" {
			continue
		}
		players = append(players, map[string]any{
			"player_id":    p.ID,
			"display_name": p.DisplayName,
			"seat":         p.Seat,
			"connected":    p.Connected,
		})
	}
	hp := [2]int{0, 0}
	mult := [2]float64{1, 1}
	phase := "lobby"
	round := 0
	clockMs := int64(0)
	if r.match != nil {
		hp = r.match.HP()
		mult = r.match.Mult()
		phase = string(r.match.Phase())
		round = r.match.Round()
		clockMs = r.match.MatchClockRemainingMs()
	}
	return map[string]any{
		"players":      players,
		"hp":           hp,
		"multipliers":  mult,
		"phase":        phase,
		"clock":        clockMs,
		"round":        round,
		"sudden_death": r.match != nil && r.match.SuddenDeath(),
	}
}

func (r *Room) broadcastAll(msgType string, data any) {
	for _, p := range r.players {
		if p.Conn != nil {
			p.Conn.Send(msgType, data)
		}
	}
}

// roomEvents adapts match.Events onto the room actor (callbacks already on room goroutine via clock).
type roomEvents struct{ room *Room }

func (e *roomEvents) OnRoundStart(round int, prompt data.Prompt, deadlineMs int64, suddenDeath bool) {
	e.room.broadcastAll("round_start", map[string]any{
		"prompt": map[string]any{
			"id":       prompt.ID,
			"text":     prompt.Text,
			"category": prompt.Category,
		},
		"deadline_ts":  deadlineMs,
		"round":        round,
		"sudden_death": suddenDeath,
	})
}

func (e *roomEvents) OnLocked(seat string) {
	e.room.broadcastAll("locked", map[string]any{"seat": seat})
}

func (e *roomEvents) OnRoundResult(out match.RoundOutcome) {
	e.room.broadcastAll("round_result", out)
}

func (e *roomEvents) OnMatchEnd(end match.MatchEnd) {
	e.room.endedAt = e.room.clock.Now()
	winner := any(nil)
	draw := end.Draw || end.Winner < 0
	if !draw {
		winner = end.Winner
	}
	e.room.broadcastAll("match_end", map[string]any{
		"winner": winner,
		"draw":   draw,
		"reason": string(end.Reason),
		"recap":  end.Recap,
	})
}

func (e *roomEvents) OnRoomState() {
	data := e.room.roomStateData()
	e.room.broadcastAll("room_state", data)
}

func (e *roomEvents) OnError(playerID, code, message string) {
	e.room.sendErr(playerID, code, message)
}

// ID returns the lobby id (for handlers).
func (r *Room) ID() string { return r.id }

// HostToken exposes the create-time host token (tests / diagnostics).
func (r *Room) HostToken() string { return r.hostToken }

func (r *Room) Mode() Mode {
	ch := make(chan Mode, 1)
	r.do(func() { ch <- r.mode })
	return <-ch
}
