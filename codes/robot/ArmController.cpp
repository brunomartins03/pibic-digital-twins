#include "ArmController.h"
#include <math.h>

float ArmController::clampAngle(float v) {
  if (v < 0) return 0;
  if (v > 180) return 180;
  return v;
}

void ArmController::begin(uint8_t basePin, uint8_t shoulderPin, uint8_t elbowPin,
                           uint8_t gripperPin, uint8_t stepDeg, float moveSpeedDegPerTick) {
  stepDeg_ = stepDeg;
  moveSpeed_ = moveSpeedDegPerTick;

  for (uint8_t i = 0; i < JOINT_COUNT; i++) {
    current_[i] = 90;
    target_[i] = 90;
  }
  currentGripper_ = 90;
  targetGripper_ = 90;

  // Standard hobby servos run at 50Hz; ESP32Servo needs this set explicitly
  // before attach() since the underlying LEDC timer is shared/configurable.
  // servoBase_.setPeriodHertz(50);
  // servoShoulder_.setPeriodHertz(50);
  // servoElbow_.setPeriodHertz(50);
  // servoGripper_.setPeriodHertz(50);

  // 500-2400us is the common full-range pulse width for SG90-class servos.
  // Narrow this if your specific servos buzz/strain at the endpoints.
  servoBase_.attach(basePin, 500, 2400);
  servoShoulder_.attach(shoulderPin, 500, 2400);
  servoElbow_.attach(elbowPin, 500, 2400);
  servoGripper_.attach(gripperPin, 500, 2400);

  writeServos();
}

void ArmController::update() {
  bool moved = false;

  for (uint8_t i = 0; i < JOINT_COUNT; i++) {
    float diff = target_[i] - current_[i];
    if (fabsf(diff) < 0.01f) continue;
    current_[i] += constrain(diff, -moveSpeed_, moveSpeed_);
    moved = true;
  }

  float gDiff = targetGripper_ - currentGripper_;
  if (fabsf(gDiff) >= 0.01f) {
    currentGripper_ += constrain(gDiff, -moveSpeed_, moveSpeed_);
    moved = true;
  }

  if (moved) writeServos();
}

bool ArmController::applyKey(char key) {
  switch (key) {
    case 'q': target_[JOINT_BASE] = clampAngle(target_[JOINT_BASE] + stepDeg_); break;
    case 'a': target_[JOINT_BASE] = clampAngle(target_[JOINT_BASE] - stepDeg_); break;

    case 'w': target_[JOINT_SHOULDER] = clampAngle(target_[JOINT_SHOULDER] + stepDeg_); break;
    case 's': target_[JOINT_SHOULDER] = clampAngle(target_[JOINT_SHOULDER] - stepDeg_); break;

    case 'e': target_[JOINT_ELBOW] = clampAngle(target_[JOINT_ELBOW] + stepDeg_); break;
    case 'd': target_[JOINT_ELBOW] = clampAngle(target_[JOINT_ELBOW] - stepDeg_); break;

    case 'r': targetGripper_ = clampAngle(targetGripper_ + stepDeg_); break;
    case 'f': targetGripper_ = clampAngle(targetGripper_ - stepDeg_); break;

    default:
      return false;
  }
  return true;
}

void ArmController::setTargets(const float joints[JOINT_COUNT], float gripper) {
  for (uint8_t i = 0; i < JOINT_COUNT; i++) {
    target_[i] = clampAngle(joints[i]);
  }
  targetGripper_ = clampAngle(gripper);
}

ArmState ArmController::getState() const {
  ArmState s;
  for (uint8_t i = 0; i < JOINT_COUNT; i++) s.joints[i] = current_[i];
  s.gripper = currentGripper_;
  return s;
}

void ArmController::writeServos() {
  servoBase_.write((int)roundf(current_[JOINT_BASE]));
  servoShoulder_.write((int)roundf(current_[JOINT_SHOULDER]));
  servoElbow_.write((int)roundf(current_[JOINT_ELBOW]));
  servoGripper_.write((int)roundf(currentGripper_));
}
