// Robotic arm digital twin firmware (ESP8266; ESP32 for the Wokwi simulation).
//
// Local Serial keys and remote MQTT commands both set target angles on the
// ArmController; whenever the real servo angles change, the new state is
// published so the Decentraland twin follows the physical arm.

#if defined(WOKWI_SIM)
#include "config.wokwi.h"
#else
#include "config.h"
#endif

#include "ArmController.h"
#include "MqttLink.h"

#if MQTT_TLS
#include "mqtt_ca.h"
#define MQTT_CA MQTT_CA_CERT
#else
#define MQTT_CA nullptr
#endif

namespace {
constexpr unsigned long WIFI_SETUP_TIMEOUT_MS = 15000;

ArmController arm;
MqttLink mqtt;

ArmState lastPublished;
bool mustPublish = true; // publish on boot and after every (re)connect
bool wasConnected = false;
unsigned long lastPublishMs = 0;
} // namespace

void connectWiFi() {
  Serial.printf("[wifi] connecting to %s", WIFI_SSID);
  WiFi.mode(WIFI_STA);
  WiFi.setAutoReconnect(true);
  WiFi.begin(WIFI_SSID, WIFI_PASSWORD);

  // Bounded wait: local Serial control must keep working without WiFi.
  unsigned long start = millis();
  while (WiFi.status() != WL_CONNECTED && millis() - start < WIFI_SETUP_TIMEOUT_MS) {
    delay(300);
    Serial.print(".");
  }
  if (WiFi.status() == WL_CONNECTED) {
    Serial.printf("\n[wifi] connected, IP: %s\n", WiFi.localIP().toString().c_str());
  } else {
    Serial.println("\n[wifi] not connected yet, retrying in background");
  }
}

void publishStateIfNeeded() {
  bool connected = mqtt.isConnected();
  if (connected && !wasConnected) mustPublish = true; // resync after reconnect
  wasConnected = connected;
  if (!connected) return;

  ArmState current = arm.getState();
  unsigned long now = millis();
  if (!mustPublish && (current == lastPublished || now - lastPublishMs < TELEMETRY_MIN_INTERVAL_MS)) {
    return;
  }
  if (mqtt.publishState(current)) {
    lastPublished = current;
    lastPublishMs = now;
    mustPublish = false;
  }
}

void handleSerialInput() {
  while (Serial.available() > 0) {
    arm.applyKey((char)Serial.read()); // newlines and unknown keys are ignored
  }
}

void printHelp() {
  Serial.println("--- Local controls (Serial, 115200 baud) ---");
  Serial.println("Base     : Q (+) | A (-)");
  Serial.println("Shoulder : W (+) | S (-)");
  Serial.println("Elbow    : E (+) | D (-)");
  Serial.println("Gripper  : R (+) | F (-)");
}

void setup() {
  Serial.begin(115200);
  delay(200);
  Serial.println("\nRobotic arm - digital twin firmware");

  const uint8_t jointPins[JOINT_COUNT] = {PIN_BASE, PIN_SHOULDER, PIN_ELBOW};
  arm.begin(jointPins, PIN_GRIPPER, STEP_DEG, MOVE_STEP_DEG, MOVE_TICK_MS);

  connectWiFi();

  mqtt.onCommand([](const ArmState &target) { arm.setTargets(target); });
  mqtt.begin({MQTT_HOST, MQTT_PORT, MQTT_CLIENT_ID, MQTT_USERNAME, MQTT_PASSWORD, MQTT_TOPIC_PREFIX, MQTT_CA});

  printHelp();
}

void loop() {
  mqtt.loop();
  handleSerialInput();
  arm.update();
  publishStateIfNeeded();
}
