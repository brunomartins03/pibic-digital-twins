package relay

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// TestMQTTLinkIntegration exercises MQTTLink against a real broker playing
// the device's role. Set MQTT_TEST_BROKER (e.g. tcp://localhost:1883) to run.
func TestMQTTLinkIntegration(t *testing.T) {
	broker := os.Getenv("MQTT_TEST_BROKER")
	if broker == "" {
		t.Skip("MQTT_TEST_BROKER not set")
	}
	prefix := fmt.Sprintf("relay-test-%d", time.Now().UnixNano())

	device := mqtt.NewClient(mqtt.NewClientOptions().AddBroker(broker).SetClientID(prefix + "-device"))
	mustToken(t, device.Connect())
	defer device.Disconnect(100)

	commands := make(chan ArmState, 1)
	mustToken(t, device.Subscribe(prefix+topicCmd, 1, func(_ mqtt.Client, m mqtt.Message) {
		var s ArmState
		if err := json.Unmarshal(m.Payload(), &s); err == nil {
			commands <- s
		}
	}))

	// Retained messages published before the relay connects must be
	// delivered on subscribe.
	mustToken(t, device.Publish(prefix+topicStatus, 1, true, statusOnline))
	mustToken(t, device.Publish(prefix+topicState, 1, true, `{"joints":[11,22,33],"gripper":44}`))
	t.Cleanup(func() { // clear retained messages from the broker
		device.Publish(prefix+topicStatus, 1, true, "").Wait()
		device.Publish(prefix+topicState, 1, true, "").Wait()
	})

	hub := NewHub()
	link := ConnectMQTT(MQTTConfig{BrokerURL: broker, ClientID: prefix + "-relay", TopicPrefix: prefix}, hub)
	defer link.Close()

	waitFor(t, "retained state", func() bool {
		s, have, online := hub.Snapshot()
		return have && online && s.Joints[2] == 33 && s.Gripper == 44
	})

	if err := link.PublishCommand(validState()); err != nil {
		t.Fatalf("PublishCommand: %v", err)
	}
	select {
	case got := <-commands:
		if got.Joints[0] != 10 || got.UpdatedAt != nil {
			t.Fatalf("device received %+v", got)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("device did not receive command")
	}

	mustToken(t, device.Publish(prefix+topicStatus, 1, true, "offline"))
	waitFor(t, "offline status", func() bool {
		_, _, online := hub.Snapshot()
		return !online
	})
}

func mustToken(t *testing.T, tok mqtt.Token) {
	t.Helper()
	if !tok.WaitTimeout(10*time.Second) || tok.Error() != nil {
		t.Fatalf("mqtt: %v", tok.Error())
	}
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
