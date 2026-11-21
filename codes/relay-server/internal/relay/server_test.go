package relay

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type fakePublisher struct {
	mu   sync.Mutex
	sent []ArmState
}

func (f *fakePublisher) PublishCommand(s ArmState) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, s)
	return nil
}

func (f *fakePublisher) commands() []ArmState {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]ArmState(nil), f.sent...)
}

func newTestServer(t *testing.T, token string) (*Hub, *fakePublisher, *httptest.Server) {
	t.Helper()
	hub := NewHub()
	pub := &fakePublisher{}
	ts := httptest.NewServer(NewServer(hub, pub, token).Handler())
	t.Cleanup(ts.Close)
	return hub, pub, ts
}

func dialScene(t *testing.T, ts *httptest.Server, query string) *websocket.Conn {
	t.Helper()
	u := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/scene" + query
	conn, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func readEnvelope(t *testing.T, conn *websocket.Conn) Envelope {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var env Envelope
	if err := conn.ReadJSON(&env); err != nil {
		t.Fatalf("read: %v", err)
	}
	return env
}

func validState() ArmState {
	return ArmState{Joints: []float64{10, 20, 30}, Gripper: 40}
}

func TestArmStateValidate(t *testing.T) {
	tests := []struct {
		name    string
		state   ArmState
		wantErr bool
	}{
		{"valid", validState(), false},
		{"bounds", ArmState{Joints: []float64{0, 180, 90}, Gripper: 180}, false},
		{"too few joints", ArmState{Joints: []float64{1, 2}, Gripper: 0}, true},
		{"joint out of range", ArmState{Joints: []float64{1, 200, 3}, Gripper: 0}, true},
		{"gripper negative", ArmState{Joints: []float64{1, 2, 3}, Gripper: -1}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.state.Validate(); (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSceneReceivesSnapshotOnConnect(t *testing.T) {
	hub, _, ts := newTestServer(t, "")
	hub.SetDeviceOnline(true)
	hub.UpdateState(validState())

	conn := dialScene(t, ts, "")

	if env := readEnvelope(t, conn); env.Type != TypeStatus || env.DeviceOnline == nil || !*env.DeviceOnline {
		t.Fatalf("first message = %+v, want online status", env)
	}
	env := readEnvelope(t, conn)
	if env.Type != TypeTelemetry || env.State == nil || env.State.Joints[2] != 30 {
		t.Fatalf("second message = %+v, want telemetry", env)
	}
}

func TestTelemetryBroadcast(t *testing.T) {
	hub, _, ts := newTestServer(t, "")
	a, b := dialScene(t, ts, ""), dialScene(t, ts, "")
	readEnvelope(t, a) // initial status
	readEnvelope(t, b)

	hub.UpdateState(validState())

	for _, conn := range []*websocket.Conn{a, b} {
		if env := readEnvelope(t, conn); env.Type != TypeTelemetry || env.State.UpdatedAt == nil {
			t.Fatalf("got %+v, want telemetry with timestamp", env)
		}
	}
}

func TestCommandForwarding(t *testing.T) {
	hub, pub, ts := newTestServer(t, "")
	conn := dialScene(t, ts, "")
	readEnvelope(t, conn) // initial status

	send := func(raw string) Envelope {
		t.Helper()
		if err := conn.WriteMessage(websocket.TextMessage, []byte(raw)); err != nil {
			t.Fatalf("write: %v", err)
		}
		return readEnvelope(t, conn)
	}
	cmd := `{"type":"command","id":"1","state":{"joints":[45,120,60],"gripper":10}}`

	if env := send(cmd); env.Type != TypeError || env.Message != "device offline" || env.ID != "1" {
		t.Fatalf("offline device: got %+v", env)
	}

	hub.SetDeviceOnline(true)
	readEnvelope(t, conn) // status broadcast

	if env := send(cmd); env.Type != TypeAck || env.ID != "1" {
		t.Fatalf("online device: got %+v, want ack", env)
	}
	if got := pub.commands(); len(got) != 1 || got[0].Joints[1] != 120 || got[0].Gripper != 10 {
		t.Fatalf("published %+v", got)
	}

	for _, bad := range []string{
		`not json`,
		`{"type":"command","state":{"joints":[1,2],"gripper":0}}`,
		`{"type":"command","state":{"joints":[1,2,999],"gripper":0}}`,
		`{"type":"command"}`,
		`{"type":"telemetry","state":{"joints":[1,2,3],"gripper":0}}`,
	} {
		if env := send(bad); env.Type != TypeError {
			t.Errorf("%s: got %+v, want error", bad, env)
		}
	}
	if n := len(pub.commands()); n != 1 {
		t.Fatalf("invalid commands were published: %d total", n)
	}
}

func TestSceneToken(t *testing.T) {
	_, _, ts := newTestServer(t, "secret")
	u := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/scene"

	if _, resp, err := websocket.DefaultDialer.Dial(u, nil); err == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("missing token: err=%v resp=%v, want 401", err, resp)
	}
	conn := dialScene(t, ts, "?token=secret")
	readEnvelope(t, conn)
}

func TestHTTPEndpoints(t *testing.T) {
	hub, _, ts := newTestServer(t, "")

	resp, err := http.Get(ts.URL + "/api/state")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("/api/state before telemetry: %d, want 204", resp.StatusCode)
	}

	hub.UpdateState(validState())
	hub.SetDeviceOnline(true)

	var state ArmState
	getJSON(t, ts.URL+"/api/state", &state)
	if state.Gripper != 40 {
		t.Fatalf("/api/state = %+v", state)
	}

	var health map[string]bool
	getJSON(t, ts.URL+"/healthz", &health)
	if !health["ok"] || !health["deviceOnline"] {
		t.Fatalf("/healthz = %v", health)
	}
}

func getJSON(t *testing.T, url string, v any) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("decode %s: %v", url, err)
	}
}

func TestHTTPCommand(t *testing.T) {
	hub, pub, ts := newTestServer(t, "secret")
	post := func(query, body string) (int, Envelope) {
		t.Helper()
		resp, err := http.Post(ts.URL+"/api/command"+query, "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var env Envelope
		_ = json.NewDecoder(resp.Body).Decode(&env)
		return resp.StatusCode, env
	}
	cmd := `{"type":"command","id":"7","state":{"joints":[45,120,60],"gripper":10}}`

	if code, _ := post("", cmd); code != http.StatusUnauthorized {
		t.Fatalf("missing token: %d, want 401", code)
	}
	if code, env := post("?token=secret", cmd); code != http.StatusBadRequest || env.Message != "device offline" {
		t.Fatalf("offline device: %d %+v", code, env)
	}
	hub.SetDeviceOnline(true)
	if code, env := post("?token=secret", cmd); code != http.StatusOK || env.Type != TypeAck || env.ID != "7" {
		t.Fatalf("online device: %d %+v, want ack", code, env)
	}
	if code, env := post("?token=secret", `not json`); code != http.StatusBadRequest || env.Type != TypeError {
		t.Fatalf("invalid JSON: %d %+v", code, env)
	}
	if got := pub.commands(); len(got) != 1 || got[0].Joints[0] != 45 {
		t.Fatalf("published %+v", got)
	}
}
