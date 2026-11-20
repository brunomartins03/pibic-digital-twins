package relay

import (
	"encoding/json"
	"log"
	"sync"
	"time"
)

// sendBufferSize is how many outgoing messages a scene client may have
// queued before it is considered too slow and dropped.
const sendBufferSize = 32

// sceneClient is one connected Decentraland scene. Only its write pump
// reads from send; the hub owns closing it.
type sceneClient struct {
	send chan []byte
}

// Hub holds the last known arm state and device status and fans updates
// out to every connected scene.
type Hub struct {
	mu           sync.Mutex
	clients      map[*sceneClient]struct{}
	state        ArmState
	haveState    bool
	deviceOnline bool
}

// NewHub returns an empty hub with no known state and the device offline.
func NewHub() *Hub {
	return &Hub{clients: make(map[*sceneClient]struct{})}
}

// UpdateState stores a state reported by the device and broadcasts it.
func (h *Hub) UpdateState(s ArmState) {
	now := time.Now().UTC()
	s.UpdatedAt = &now

	h.mu.Lock()
	defer h.mu.Unlock()
	h.state = s
	h.haveState = true
	h.broadcastLocked(Envelope{Type: TypeTelemetry, State: &s})
}

// SetDeviceOnline records the device's connection status and broadcasts it
// when it changes.
func (h *Hub) SetDeviceOnline(online bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.deviceOnline == online {
		return
	}
	h.deviceOnline = online
	h.broadcastLocked(Envelope{Type: TypeStatus, DeviceOnline: &online})
}

// Snapshot returns the last known state, whether any state was received,
// and whether the device is online.
func (h *Hub) Snapshot() (state ArmState, haveState, deviceOnline bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.state, h.haveState, h.deviceOnline
}

// register adds a client and queues the current status and state, so a
// scene that just connected is immediately in sync.
func (h *Hub) register() *sceneClient {
	c := &sceneClient{send: make(chan []byte, sendBufferSize)}

	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c] = struct{}{}

	online := h.deviceOnline
	h.enqueueLocked(c, Envelope{Type: TypeStatus, DeviceOnline: &online})
	if h.haveState {
		s := h.state
		h.enqueueLocked(c, Envelope{Type: TypeTelemetry, State: &s})
	}
	return c
}

// unregister removes a client and closes its send channel. Safe to call
// more than once.
func (h *Hub) unregister(c *sceneClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.removeLocked(c)
}

// sendTo queues a message for a single client, if still registered.
func (h *Hub) sendTo(c *sceneClient, env Envelope) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.enqueueLocked(c, env)
}

func (h *Hub) broadcastLocked(env Envelope) {
	for c := range h.clients {
		h.enqueueLocked(c, env)
	}
}

func (h *Hub) enqueueLocked(c *sceneClient, env Envelope) {
	if _, ok := h.clients[c]; !ok {
		return
	}
	payload, err := json.Marshal(env)
	if err != nil {
		log.Printf("hub: marshal %s: %v", env.Type, err)
		return
	}
	select {
	case c.send <- payload:
	default:
		log.Println("hub: scene too slow, dropping connection")
		h.removeLocked(c)
	}
}

func (h *Hub) removeLocked(c *sceneClient) {
	if _, ok := h.clients[c]; ok {
		delete(h.clients, c)
		close(c.send)
	}
}
