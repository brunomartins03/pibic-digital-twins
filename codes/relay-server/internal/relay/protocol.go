// Package relay bridges the robotic arm (MQTT) and the Decentraland scene
// (WebSocket), keeping the last known arm state in memory.
package relay

import (
	"fmt"
	"time"
)

// JointCount is the number of joints in ArmState.Joints, ordered as
// [base, shoulder, elbow]. It must match the firmware's JOINT_COUNT.
const JointCount = 3

// Valid servo angle range, in degrees.
const (
	MinAngle = 0.0
	MaxAngle = 180.0
)

// ArmState is the pose of the arm. It is the payload of the MQTT state and
// command topics and is embedded in WebSocket envelopes.
type ArmState struct {
	Joints    []float64  `json:"joints"`
	Gripper   float64    `json:"gripper"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

// Validate reports whether the state has the expected joint count and all
// angles inside [MinAngle, MaxAngle].
func (s ArmState) Validate() error {
	if len(s.Joints) != JointCount {
		return fmt.Errorf("expected %d joints, got %d", JointCount, len(s.Joints))
	}
	for i, a := range s.Joints {
		if a < MinAngle || a > MaxAngle {
			return fmt.Errorf("joint %d angle %.1f out of range [%g, %g]", i, a, MinAngle, MaxAngle)
		}
	}
	if s.Gripper < MinAngle || s.Gripper > MaxAngle {
		return fmt.Errorf("gripper angle %.1f out of range [%g, %g]", s.Gripper, MinAngle, MaxAngle)
	}
	return nil
}

// WebSocket envelope types.
const (
	TypeTelemetry = "telemetry" // relay -> scene: current arm state
	TypeStatus    = "status"    // relay -> scene: device online/offline
	TypeCommand   = "command"   // scene -> relay: target arm state
	TypeAck       = "ack"       // relay -> scene: command forwarded
	TypeError     = "error"     // relay -> scene: command rejected
)

// Envelope is every message exchanged on the scene WebSocket.
type Envelope struct {
	Type         string    `json:"type"`
	ID           string    `json:"id,omitempty"`
	State        *ArmState `json:"state,omitempty"`
	DeviceOnline *bool     `json:"deviceOnline,omitempty"`
	Message      string    `json:"message,omitempty"`
}
