#pragma once

#include <Arduino.h>

// The ESP8266 core ships a working Servo library; the ESP32 needs the
// LEDC-based ESP32Servo replacement (used by the Wokwi simulation).
#if defined(ESP32)
#include <ESP32Servo.h>
#else
#include <Servo.h>
#endif

// Indices into ArmState::joints. The order must match the relay server and
// the Decentraland scene: [base, shoulder, elbow].
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

bool operator==(const ArmState &a, const ArmState &b);
inline bool operator!=(const ArmState &a, const ArmState &b) { return !(a == b); }

// Owns the four servos and the arm's current and target angles. Both local
// (Serial keys) and remote (MQTT) control only set targets; update() moves
// the servos toward them at a bounded speed, so motion is smooth whichever
// path issued the command.
class ArmController {
public:
  void begin(const uint8_t jointPins[JOINT_COUNT], uint8_t gripperPin,
             uint8_t stepDeg, float maxStepPerTick, uint16_t tickMs);

  // Call every loop() iteration. Every tickMs it moves each servo at most
  // maxStepPerTick degrees toward its target.
  void update();

  // Local control: Q/A base, W/S shoulder, E/D elbow, R/F gripper
  // (case-insensitive). Returns true if the key was recognized.
  bool applyKey(char key);

  // Remote control: absolute target angles, clamped to [0, 180].
  void setTargets(const ArmState &target);

  // Angles currently written to the servos, rounded to whole degrees.
  ArmState getState() const;

private:
  Servo servos_[JOINT_COUNT + 1]; // joints, then gripper
  float current_[JOINT_COUNT + 1];
  float target_[JOINT_COUNT + 1];

  uint8_t stepDeg_ = 5;
  float maxStepPerTick_ = 2.0f;
  uint16_t tickMs_ = 20;
  unsigned long lastTickMs_ = 0;

  void nudge(uint8_t index, int delta);
  void writeServos();
  static float clampAngle(float v);
};
