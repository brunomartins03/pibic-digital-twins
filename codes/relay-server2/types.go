package main

import "time"

type ArmState struct {
	Joints    []float64 `json:"joints"`
	Gripper   float64   `json:"gripper"`
	UpdatedAt time.Time `json:"updatedAt"`
	Source    string    `json:"source,omitempty"`   // device or scene
}

type Envelope struct {
	Type    string          `json:"type"`              // telemetry | command | ack | error | hello
	State   *ArmState       `json:"state,omitempty"`
	Message string          `json:"message,omitempty"` // used by error/ack
	ID      string          `json:"id,omitempty"`
}
