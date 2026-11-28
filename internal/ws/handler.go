package ws

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/coder/websocket"
	"github.com/peterzdhuang/krill-billion/internal/lobby"
)

// originHosts extracts host[:port] patterns for coder/websocket OriginPatterns.
// FRONTEND_ORIGIN may be a full URL (https://app.vercel.app) or a bare host.
func originHosts(allowed string) (hosts []string, insecure bool) {
	allowed = strings.TrimSpace(allowed)
	if allowed == "" || allowed == "*" {
		return nil, true
	}
	for _, a := range strings.Split(allowed, ",") {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		if a == "*" {
			return nil, true
		}
		if strings.Contains(a, "://") {
			if u, err := url.Parse(a); err == nil && u.Host != "" {
				hosts = append(hosts, u.Host)
				continue
			}
		}
		hosts = append(hosts, a)
	}
	if len(hosts) == 0 {
		return nil, true
	}
	return hosts, false
}

// Handler upgrades WebSocket connections and routes protocol messages.
type Handler struct {
	Registry      *lobby.Registry
	AllowedOrigin string // FRONTEND_ORIGIN; empty allows local / same-origin
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	lobbyID := r.URL.Query().Get("lobby")
	if lobbyID == "" {
		http.Error(w, "lobby query required", http.StatusBadRequest)
		return
	}
	room, ok := h.Registry.Get(lobbyID)
	if !ok {
		http.Error(w, "lobby not found", http.StatusNotFound)
		return
	}

	hosts, insecure := originHosts(h.AllowedOrigin)
	wsConn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: insecure,
		OriginPatterns:     hosts,
	})
	if err != nil {
		return
	}
	wsConn.SetReadLimit(maxMessageSize)

	ctx, cancel := context.WithCancel(r.Context())
	var conn *Conn
	conn = newConn(wsConn, func() {
		cancel()
		if pid := conn.PlayerID(); pid != "" {
			room.Disconnect(pid)
		}
	})

	go conn.writePump(ctx)
	go conn.heartbeat(ctx)
	h.readLoop(ctx, conn, room, lobbyID)
}

func (h *Handler) readLoop(ctx context.Context, conn *Conn, room *lobby.Room, lobbyID string) {
	defer conn.Close()
	joined := false
	for {
		_, data, err := conn.ws.Read(ctx)
		if err != nil {
			return
		}
		conn.noteActivity()

		var env Envelope
		if err := json.Unmarshal(data, &env); err != nil {
			conn.Send(TypeError, ErrorData{Code: ErrBadPhase, Message: "invalid json"})
			continue
		}

		raw, _ := json.Marshal(env.Data)
		switch env.Type {
		case TypePing:
			conn.Send(TypePong, map[string]any{})
		case TypeJoin:
			var d JoinData
			if err := json.Unmarshal(raw, &d); err != nil {
				conn.Send(TypeError, ErrorData{Code: ErrBadPhase, Message: "bad join"})
				continue
			}
			if d.LobbyID != "" && d.LobbyID != lobbyID {
				conn.Send(TypeError, ErrorData{Code: ErrBadPhase, Message: "lobby mismatch"})
				continue
			}
			res := room.Join(d.DisplayName, d.Seat, d.ReconnectToken, d.HostToken, conn)
			if res.ErrCode != "" {
				conn.Send(TypeError, ErrorData{Code: res.ErrCode, Message: res.ErrMessage})
				continue
			}
			joined = true
			// lobby_state already broadcast by Join (includes player_id / token)
		case TypeClaimSeat:
			if !joined {
				continue
			}
			var d ClaimSeatData
			_ = json.Unmarshal(raw, &d)
			if code, msg := room.ClaimSeat(conn.PlayerID(), d.Seat); code != "" {
				conn.Send(TypeError, ErrorData{Code: code, Message: msg})
			}
		case TypeReady:
			if !joined {
				continue
			}
			if code, msg := room.Ready(conn.PlayerID()); code != "" {
				conn.Send(TypeError, ErrorData{Code: code, Message: msg})
			}
		case TypeSetMode:
			if !joined {
				continue
			}
			var d SetModeData
			_ = json.Unmarshal(raw, &d)
			if code, msg := room.SetMode(conn.PlayerID(), lobby.Mode(d.Mode)); code != "" {
				conn.Send(TypeError, ErrorData{Code: code, Message: msg})
			}
		case TypeStart:
			if !joined {
				continue
			}
			if code, msg := room.Start(conn.PlayerID()); code != "" {
				conn.Send(TypeError, ErrorData{Code: code, Message: msg})
			}
		case TypeLock:
			if !joined {
				continue
			}
			var d LockData
			_ = json.Unmarshal(raw, &d)
			room.Lock(conn.PlayerID(), d.Text)
		default:
			// ignore unknown / illegal
		}
	}
}

// CreateLobbyHTTP handles POST /api/lobbies.
func CreateLobbyHTTP(reg *lobby.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Mode        string `json:"mode"`
			DisplayName string `json:"display_name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		mode := lobby.Mode(req.Mode)
		if mode == "" {
			mode = lobby.Mode1v1
		}
		id, link, token, err := reg.Create(mode, req.DisplayName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"lobby_id":   id,
			"link":       link,
			"host_token": token,
		})
	}
}
