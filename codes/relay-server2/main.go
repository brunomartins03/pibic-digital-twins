package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 1 << 16 // 64KB
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool { return true },
}

func main() {
	addr := envOr("BRIDGE_ADDR", ":8080")
	deviceToken := os.Getenv("DEVICE_TOKEN")
	sceneToken := os.Getenv("SCENE_TOKEN")

	if deviceToken == "" || sceneToken == "" {
		log.Println("WARNING: DEVICE_TOKEN and/or SCENE_TOKEN not set — running with NO auth. Fine for local dev, unsafe for anything public.")
	}

	hub := NewHub()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws/device", deviceHandler(hub, deviceToken))
	mux.HandleFunc("/ws/scene", sceneHandler(hub, sceneToken))
	mux.HandleFunc("/api/state", stateHandler(hub))
	mux.HandleFunc("/healthz", healthHandler(hub))

	srv := &http.Server{
		Addr:         addr,
		Handler:      withCORS(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("bridge server listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

// --- /ws/device: the ESP32 connects here ---------------------------------

func deviceHandler(hub *Hub, token string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if token != "" && r.URL.Query().Get("token") != token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("device: upgrade failed: %v", err)
			return
		}
		log.Println("device: connected")
		hub.SetDevice(conn)

		conn.SetReadLimit(maxMessageSize)
		conn.SetReadDeadline(time.Now().Add(pongWait))
		conn.SetPongHandler(func(string) error {
			conn.SetReadDeadline(time.Now().Add(pongWait))
			return nil
		})

		done := make(chan struct{})
		go pingLoop(conn, done, &hub.deviceWrite)

		defer func() {
			close(done)
			hub.ClearDevice(conn)
			_ = conn.Close()
			log.Println("device: disconnected")
		}()

		for {
			var env Envelope
			if err := conn.ReadJSON(&env); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					log.Printf("device: read error: %v", err)
				}
				return
			}

			switch env.Type {
			case "telemetry":
				if env.State != nil {
					hub.PublishTelemetry(*env.State)
				}
			case "hello":
				log.Printf("device: hello (%s)", env.Message)
			default:
				log.Printf("device: unknown message type %q", env.Type)
			}
		}
	}
}

// --- /ws/scene: the Decentraland scene connects here -----------------------

func sceneHandler(hub *Hub, token string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if token != "" && r.URL.Query().Get("token") != token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("scene: upgrade failed: %v", err)
			return
		}
		log.Println("scene: connected")
		hub.AddScene(conn)

		conn.SetReadLimit(maxMessageSize)
		conn.SetReadDeadline(time.Now().Add(pongWait))
		conn.SetPongHandler(func(string) error {
			conn.SetReadDeadline(time.Now().Add(pongWait))
			return nil
		})

		writeMu := hub.SceneMutex(conn)
		done := make(chan struct{})
		go pingLoop(conn, done, writeMu)

		defer func() {
			close(done)
			hub.RemoveScene(conn)
			_ = conn.Close()
			log.Println("scene: disconnected")
		}()

		for {
			var env Envelope
			if err := conn.ReadJSON(&env); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					log.Printf("scene: read error: %v", err)
				}
				return
			}

			if env.Type != "command" || env.State == nil {
				log.Printf("scene: unknown/empty message type %q", env.Type)
				continue
			}

			cmd := *env.State
			cmd.Source = "scene"

			if !hub.SendCommand(cmd) {
				h := Envelope{Type: "error", Message: "no device connected", ID: env.ID}
				_ = conn.WriteJSON(h)
			} else {
				ack := Envelope{Type: "ack", ID: env.ID}
				_ = conn.WriteJSON(ack)
			}
		}
	}
}

// --- REST helpers -----------------------------------------------------

func stateHandler(hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state, ok := hub.LastState()
		w.Header().Set("Content-Type", "application/json")
		if !ok {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_ = json.NewEncoder(w).Encode(state)
	}
}

func healthHandler(hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":              true,
			"deviceConnected": hub.DeviceConnected(),
		})
	}
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// pingLoop sends a WebSocket ping on an interval to keep the connection
// alive through proxies/NATs and to detect dead peers quickly. Writes are
// serialized against mu, the same mutex the hub uses to send real messages
// on this connection, since gorilla/websocket conns aren't safe for
// concurrent writes from multiple goroutines.
func pingLoop(conn *websocket.Conn, done <-chan struct{}, mu *sync.Mutex) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			mu.Lock()
			err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeWait))
			mu.Unlock()
			if err != nil {
				return
			}
		case <-done:
			return
		}
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
