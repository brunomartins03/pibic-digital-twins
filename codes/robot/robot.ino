#include <ESP8266WiFi.h>
#include "config.h"
#include "ArmController.h"
#include "BridgeClient.h"

ArmController arm;
BridgeClient bridge;

ArmState lastSent;
unsigned long lastSendMs = 0;
bool haveSent = false;

void connectWiFi() {
  Serial.printf("[wifi] connecting to %s", WIFI_SSID);
  WiFi.mode(WIFI_STA);
  WiFi.begin(WIFI_SSID, WIFI_PASSWORD);

  unsigned long start = millis();
  while (WiFi.status() != WL_CONNECTED) {
    delay(300);
    Serial.print(".");
    // Don't block setup() forever if WiFi is briefly unavailable — keep
    // retrying from loop() instead so local Serial control still works.
    if (millis() - start > 15000) {
      Serial.println("\n[wifi] not connected yet, will keep retrying in background");
      return;
    }
  }
  Serial.print("\n[wifi] connected, IP: ");
  Serial.println(WiFi.localIP());
}

bool stateChanged(const ArmState &a, const ArmState &b) {
  for (uint8_t i = 0; i < JOINT_COUNT; i++) {
    if (a.joints[i] != b.joints[i]) return true;
  }
  return a.gripper != b.gripper;
}

void publishTelemetryIfNeeded() {
  ArmState current = arm.getState();
  unsigned long now = millis();
  bool dueForHeartbeat = (now - lastSendMs) >= TELEMETRY_HEARTBEAT_MS;

  if (!haveSent || stateChanged(current, lastSent) || dueForHeartbeat) {
    bridge.sendTelemetry(current);
    lastSent = current;
    lastSendMs = now;
    haveSent = true;
  }
}

void onRemoteCommand(const ArmState &target) {
  arm.setTargets(target.joints, target.gripper);
}

void handleSerialInput() {
  if (Serial.available() <= 0) return;
  char key = tolower(Serial.read());
  arm.applyKey(key);
}

void setup() {
  Serial.begin(115200);
  delay(200);
  Serial.println("\nRobotic Arm — digital twin firmware");

  arm.begin(PIN_BASE, PIN_SHOULDER, PIN_ELBOW, PIN_GRIPPER, STEP_DEG, MOVE_SPEED_DEG);

  connectWiFi();

  bridge.onCommand(onRemoteCommand);
  bridge.begin(BRIDGE_HOST, BRIDGE_PORT, BRIDGE_PATH, DEVICE_TOKEN);

  Serial.println("--- Local controls (Serial) ---");
  Serial.println("Base     : Q (Left)  | A (Right)");
  Serial.println("Shoulder : W (Up)    | S (Down)");
  Serial.println("Elbow    : E (Up)    | D (Down)");
  Serial.println("Gripper  : R (Open)  | F (Close)");
  Serial.println("Type a key and press Enter to move.");
}

void loop() {
  if (WiFi.status() != WL_CONNECTED) {
    static unsigned long lastRetry = 0;
    if (millis() - lastRetry > 10000) {
      lastRetry = millis();
      Serial.println("[wifi] reconnecting...");
      WiFi.reconnect();
    }
  }

  bridge.loop();
  handleSerialInput();
  arm.update();
  publishTelemetryIfNeeded();
}
