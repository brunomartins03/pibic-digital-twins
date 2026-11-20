package main

import (
	"log"
	"sync"
	"time"
	"github.com/gorilla/websocket"
)

type Hub struct {
	mu sync.RWMutex
	device *websocket.Conn
	deviceWrite sync.Mutex
	scenes map[*websocket.Conn]*sync.Mutex
	lastState ArmState
	haveState bool
}

func NewHub() *Hub {
	return &Hub{
		scenes: make(map[*websocket.Conn]*sync.Mutex),
	}
}

// Device side

func (h *Hub) SetDevice(conn *websocket.Conn) {
	h.mu.Lock()
	old := h.device
	h.device = conn
	h.mu.Unlock()

	if old != nil {
		log.Println("device: replacing existing connection")
		_ = old.Close()
	}
}

func (h *Hub) ClearDevice(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.device == conn {
		h.device = nil
	}
}

func (h *Hub) PublishTelemetry(state ArmState) {
	state.UpdatedAt = time.Now().UTC()
	state.Source = "device"

	h.mu.Lock()
	h.lastState = state
	h.haveState = true
	h.mu.Unlock()

	h.broadcastToScenes(Envelope{Type: "telemetry", State: &state})
}

func (h *Hub) SendCommand(state ArmState) bool {
	h.mu.RLock()
	conn := h.device
	h.mu.RUnlock()

	if conn == nil {
		return false
	}

	env := Envelope{Type: "command", State: &state}

	h.deviceWrite.Lock()
	defer h.deviceWrite.Unlock()
	if err := conn.WriteJSON(env); err != nil {
		log.Printf("device: write error: %v", err)
		return false
	}
	return true
}

// Scene side

func (h *Hub) AddScene(conn *websocket.Conn) {
	h.mu.Lock()
	h.scenes[conn] = &sync.Mutex{}
	state := h.lastState
	have := h.haveState
	h.mu.Unlock()

	if have {
		h.writeToScene(conn, Envelope{Type: "telemetry", State: &state})
	}
}

func (h *Hub) SceneMutex(conn *websocket.Conn) *sync.Mutex {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.scenes[conn]
}

func (h *Hub) RemoveScene(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.scenes, conn)
}

func (h *Hub) DeviceConnected() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.device != nil
}

func (h *Hub) LastState() (ArmState, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.lastState, h.haveState
}

func (h *Hub) broadcastToScenes(env Envelope) {
	h.mu.RLock()
	targets := make(map[*websocket.Conn]*sync.Mutex, len(h.scenes))
	for c, m := range h.scenes {
		targets[c] = m
	}
	h.mu.RUnlock()

	for conn, mu := range targets {
		mu.Lock()
		err := conn.WriteJSON(env)
		mu.Unlock()
		if err != nil {
			log.Printf("scene: write error, dropping connection: %v", err)
			_ = conn.Close()
			h.RemoveScene(conn)
		}
	}
}

func (h *Hub) writeToScene(conn *websocket.Conn, env Envelope) {
	h.mu.RLock()
	mu := h.scenes[conn]
	h.mu.RUnlock()
	if mu == nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	if err := conn.WriteJSON(env); err != nil {
		log.Printf("scene: write error: %v", err)
	}
}
