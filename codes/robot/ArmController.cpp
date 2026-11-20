#include "ArmController.h"
#include <math.h>

namespace {
constexpr uint8_t GRIPPER = JOINT_COUNT; // index of the gripper servo
constexpr float HOME_DEG = 90.0f;

// 500-2400us is the usual full-range pulse width for SG90-class servos.
// Narrow it if a servo buzzes or strains at the endpoints.
constexpr int PULSE_MIN_US = 500;
constexpr int PULSE_MAX_US = 2400;
} // namespace

bool operator==(const ArmState &a, const ArmState &b) {
  for (uint8_t i = 0; i < JOINT_COUNT; i++) {
    if (a.joints[i] != b.joints[i]) return false;
  }
  return a.gripper == b.gripper;
}

float ArmController::clampAngle(float v) {
  if (v < 0) return 0;
  if (v > 180) return 180;
  return v;
}

void ArmController::begin(const uint8_t jointPins[JOINT_COUNT], uint8_t gripperPin,
                          uint8_t stepDeg, float maxStepPerTick, uint16_t tickMs) {
  stepDeg_ = stepDeg;
  maxStepPerTick_ = maxStepPerTick;
  tickMs_ = tickMs;

  for (uint8_t i = 0; i <= GRIPPER; i++) {
    current_[i] = HOME_DEG;
    target_[i] = HOME_DEG;
    uint8_t pin = (i == GRIPPER) ? gripperPin : jointPins[i];
#if defined(ESP32)
    servos_[i].setPeriodHertz(50); // standard hobby servo frequency
#endif
    servos_[i].attach(pin, PULSE_MIN_US, PULSE_MAX_US);
  }
  writeServos();
  lastTickMs_ = millis();
}

void ArmController::update() {
  unsigned long now = millis();
  if (now - lastTickMs_ < tickMs_) return;
  lastTickMs_ = now;

  bool moved = false;
  for (uint8_t i = 0; i <= GRIPPER; i++) {
    float diff = target_[i] - current_[i];
    if (fabsf(diff) < 0.01f) continue;
    current_[i] += constrain(diff, -maxStepPerTick_, maxStepPerTick_);
    moved = true;
  }
  if (moved) writeServos();
}

void ArmController::nudge(uint8_t index, int delta) {
  target_[index] = clampAngle(target_[index] + delta);
}

bool ArmController::applyKey(char key) {
  switch (tolower(key)) {
    case 'q': nudge(JOINT_BASE, +stepDeg_); break;
    case 'a': nudge(JOINT_BASE, -stepDeg_); break;
    case 'w': nudge(JOINT_SHOULDER, +stepDeg_); break;
    case 's': nudge(JOINT_SHOULDER, -stepDeg_); break;
    case 'e': nudge(JOINT_ELBOW, +stepDeg_); break;
    case 'd': nudge(JOINT_ELBOW, -stepDeg_); break;
    case 'r': nudge(GRIPPER, +stepDeg_); break;
    case 'f': nudge(GRIPPER, -stepDeg_); break;
    default: return false;
  }
  return true;
}

void ArmController::setTargets(const ArmState &target) {
  for (uint8_t i = 0; i < JOINT_COUNT; i++) target_[i] = clampAngle(target.joints[i]);
  target_[GRIPPER] = clampAngle(target.gripper);
}

ArmState ArmController::getState() const {
  ArmState s;
  for (uint8_t i = 0; i < JOINT_COUNT; i++) s.joints[i] = roundf(current_[i]);
  s.gripper = roundf(current_[GRIPPER]);
  return s;
}

void ArmController::writeServos() {
  for (uint8_t i = 0; i <= GRIPPER; i++) servos_[i].write((int)roundf(current_[i]));
}
