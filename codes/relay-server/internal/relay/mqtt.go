package relay

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// MQTT topic suffixes, appended to MQTTConfig.TopicPrefix. They must match
// the firmware's MqttLink.
const (
	topicState  = "/state"  // device -> relay, retained ArmState JSON
	topicCmd    = "/cmd"    // relay -> device, ArmState JSON
	topicStatus = "/status" // device -> relay, retained "online" | "offline" (LWT)
)

const (
	statusOnline  = "online"
	publishWait   = 5 * time.Second
	qosAtMostOnce = 0
	qosAtLeastOne = 1
)

// MQTTConfig holds the broker connection settings.
type MQTTConfig struct {
	BrokerURL   string // e.g. tcp://localhost:1883
	ClientID    string
	Username    string
	Password    string
	TopicPrefix string // e.g. robotarm
}

// MQTTLink connects the hub to the device through an MQTT broker. It
// implements CommandPublisher.
type MQTTLink struct {
	client   mqtt.Client
	cmdTopic string
}

// ConnectMQTT starts connecting to the broker and returns immediately; it
// keeps retrying in the background and re-subscribes on every reconnect.
func ConnectMQTT(cfg MQTTConfig, hub *Hub) *MQTTLink {
	stateTopic := cfg.TopicPrefix + topicState
	statusTopic := cfg.TopicPrefix + topicStatus

	onState := func(_ mqtt.Client, msg mqtt.Message) {
		var s ArmState
		if err := json.Unmarshal(msg.Payload(), &s); err != nil {
			log.Printf("mqtt: bad state payload %q: %v", msg.Payload(), err)
			return
		}
		if err := s.Validate(); err != nil {
			log.Printf("mqtt: invalid state: %v", err)
			return
		}
		hub.UpdateState(s)
	}
	onStatus := func(_ mqtt.Client, msg mqtt.Message) {
		online := string(msg.Payload()) == statusOnline
		log.Printf("mqtt: device status %q", msg.Payload())
		hub.SetDeviceOnline(online)
	}

	opts := mqtt.NewClientOptions().
		AddBroker(cfg.BrokerURL).
		SetClientID(cfg.ClientID).
		SetUsername(cfg.Username).
		SetPassword(cfg.Password).
		SetCleanSession(true).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectRetryInterval(5 * time.Second).
		SetConnectionAttemptHandler(func(broker *url.URL, cfg *tls.Config) *tls.Config {
			log.Printf("mqtt: connecting to %s", broker.Redacted())
			return cfg
		}).
		SetOnConnectHandler(func(c mqtt.Client) {
			log.Printf("mqtt: connected to %s", cfg.BrokerURL)
			subs := map[string]byte{stateTopic: qosAtLeastOne, statusTopic: qosAtLeastOne}
			token := c.SubscribeMultiple(subs, func(c mqtt.Client, msg mqtt.Message) {
				if msg.Topic() == statusTopic {
					onStatus(c, msg)
				} else {
					onState(c, msg)
				}
			})
			if token.Wait() && token.Error() != nil {
				log.Printf("mqtt: subscribe failed: %v", token.Error())
			}
		}).
		SetConnectionLostHandler(func(_ mqtt.Client, err error) {
			log.Printf("mqtt: connection lost: %v", err)
		})

	client := mqtt.NewClient(opts)
	client.Connect() // with ConnectRetry, failures are retried in the background
	return &MQTTLink{client: client, cmdTopic: cfg.TopicPrefix + topicCmd}
}

// PublishCommand sends a target state to the device. Commands are not
// retained: a stale command must never move the arm after a reboot.
func (m *MQTTLink) PublishCommand(s ArmState) error {
	payload, err := json.Marshal(ArmState{Joints: s.Joints, Gripper: s.Gripper})
	if err != nil {
		return err
	}
	token := m.client.Publish(m.cmdTopic, qosAtMostOnce, false, payload)
	if !token.WaitTimeout(publishWait) {
		return fmt.Errorf("publish to %s timed out", m.cmdTopic)
	}
	return token.Error()
}

// Close disconnects from the broker, allowing in-flight work to finish.
func (m *MQTTLink) Close() {
	m.client.Disconnect(250)
}
