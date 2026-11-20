#pragma once

#include <Arduino.h>
#include <Servo.h>

// Indices into the joint array / telemetry payload. This order must match
// what the bridge server and Decentraland scene expect — see the bridge's
// README for the wire protocol.
enum JointIndex : uint8_t {
  JOINT_BASE = 0,
  JOINT_SHOULDER = 1,
  JOINT_ELBOW = 2,
  JOINT_COUNT = 3
};

struct ArmState {
  float joints[JOINT_COUNT]; // degrees, 0-180: [base, shoulder, elbow]
  float gripper;             // degrees, 0-180
};

// Owns the four servos and the arm's current/target angles. Motion is
// applied gradually in update() rather than snapped instantly, so commands
// arriving from the digital twin produce a real, watchable movement instead
// of a jerk — and so a local key press and a remote command can't stomp on
// each other mid-frame.
class ArmController {
public:
  void begin(uint8_t basePin, uint8_t shoulderPin, uint8_t elbowPin,
             uint8_t gripperPin, uint8_t stepDeg, float moveSpeedDegPerTick);

  // Call every loop() iteration. Steps current angles toward target angles
  // by at most moveSpeedDegPerTick per call. Cheap no-op once at target.
  void update();

  // Local control: mirrors the original key-based interface
  // (Q/A/W/S/E/D/R/F, lowercase). Returns true if the key was recognized.
  bool applyKey(char key);

  // Remote control: absolute target angles, e.g. relayed from the bridge
  // server / Decentraland scene. Values are clamped to [0, 180].
  void setTargets(const float joints[JOINT_COUNT], float gripper);

  // Current (not target) angles — what the servos are actually doing right
  // now. This is what gets published as telemetry.
  ArmState getState() const;

private:
  Servo servoBase_, servoShoulder_, servoElbow_, servoGripper_;

  float current_[JOINT_COUNT];
  float target_[JOINT_COUNT];
  float currentGripper_ = 90;
  float targetGripper_ = 90;

  uint8_t stepDeg_ = 5;
  float moveSpeed_ = 4.0f;

  void writeServos();
  static float clampAngle(float v);
};
