package relay

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = pongWait * 9 / 10
	maxMessageSize = 4 << 10 // 4 KiB is far above any valid envelope
)

// CommandPublisher delivers a target state to the physical arm.
type CommandPublisher interface {
	PublishCommand(ArmState) error
}

// Server exposes the scene WebSocket and the HTTP helper endpoints.
type Server struct {
	hub        *Hub
	commands   CommandPublisher
	sceneToken string
	upgrader   websocket.Upgrader
}

// NewServer builds a Server. An empty sceneToken disables authentication.
func NewServer(hub *Hub, commands CommandPublisher, sceneToken string) *Server {
	return &Server{
		hub:        hub,
		commands:   commands,
		sceneToken: sceneToken,
		upgrader: websocket.Upgrader{
			// Decentraland scenes run from arbitrary origins; access is
			// controlled by sceneToken instead.
			CheckOrigin: func(*http.Request) bool { return true },
		},
	}
}

// Handler returns the HTTP routes:
//
//	/ws/scene     WebSocket for Decentraland scenes
//	/api/command  the same command as over WebSocket, for scenes that poll
//	/api/state    last known arm state (204 if none yet)
//	/healthz      liveness and device status
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ws/scene", s.handleScene)
	mux.HandleFunc("POST /api/command", s.handleCommandHTTP)
	mux.HandleFunc("GET /api/state", s.handleState)
	mux.HandleFunc("GET /healthz", s.handleHealth)
	return withCORS(mux)
}

func (s *Server) authorized(w http.ResponseWriter, r *http.Request) bool {
	if s.sceneToken != "" && r.URL.Query().Get("token") != s.sceneToken {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return false
	}
	return true
}

func (s *Server) handleScene(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(w, r) {
		return
	}
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("scene: upgrade failed: %v", err)
		return
	}
	log.Printf("scene: connected from %s", r.RemoteAddr)

	c := s.hub.register()
	go writePump(conn, c)
	s.readPump(conn, c)

	s.hub.unregister(c)
	log.Printf("scene: disconnected from %s", r.RemoteAddr)
}

// readPump processes incoming commands until the connection fails. It
// never writes to conn; replies go through the client's send queue.
func (s *Server) readPump(conn *websocket.Conn, c *sceneClient) {
	defer conn.Close()
	conn.SetReadLimit(maxMessageSize)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("scene: read error: %v", err)
			}
			return
		}
		var env Envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			s.hub.sendTo(c, Envelope{Type: TypeError, Message: "invalid JSON: " + err.Error()})
			continue
		}
		s.hub.sendTo(c, s.handleCommand(env))
	}
}

// handleCommand validates and forwards a scene command, returning the
// reply envelope.
func (s *Server) handleCommand(env Envelope) Envelope {
	reply := func(msg string) Envelope {
		return Envelope{Type: TypeError, ID: env.ID, Message: msg}
	}
	if env.Type != TypeCommand {
		return reply("unsupported message type " + `"` + env.Type + `"`)
	}
	if env.State == nil {
		return reply("command without state")
	}
	cmd := ArmState{Joints: env.State.Joints, Gripper: env.State.Gripper}
	if err := cmd.Validate(); err != nil {
		return reply(err.Error())
	}
	if _, _, online := s.hub.Snapshot(); !online {
		return reply("device offline")
	}
	if err := s.commands.PublishCommand(cmd); err != nil {
		log.Printf("scene: publish command: %v", err)
		return reply("failed to forward command")
	}
	return Envelope{Type: TypeAck, ID: env.ID}
}

// writePump is the only goroutine that writes to conn. It exits when the
// hub closes the send channel or a write fails.
func writePump(conn *websocket.Conn, c *sceneClient) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = conn.WriteMessage(websocket.CloseMessage, nil)
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeWait)); err != nil {
				return
			}
		}
	}
}

// handleCommandHTTP accepts a command envelope in the request body and
// answers with the ack or error envelope (status 200 or 400). It serves
// scenes behind proxies that do not forward WebSocket upgrades.
func (s *Server) handleCommandHTTP(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(w, r) {
		return
	}
	var env, reply Envelope
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxMessageSize)).Decode(&env); err != nil {
		reply = Envelope{Type: TypeError, Message: "invalid JSON: " + err.Error()}
	} else {
		reply = s.handleCommand(env)
	}
	if reply.Type == TypeError {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(reply)
		return
	}
	writeJSON(w, reply)
}

func (s *Server) handleState(w http.ResponseWriter, _ *http.Request) {
	state, have, _ := s.hub.Snapshot()
	if !have {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	writeJSON(w, state)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	_, _, online := s.hub.Snapshot()
	writeJSON(w, map[string]bool{"ok": true, "deviceOnline": online})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("http: encode response: %v", err)
	}
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
