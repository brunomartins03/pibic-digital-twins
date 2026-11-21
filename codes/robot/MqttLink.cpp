#include "MqttLink.h"
#include <ArduinoJson.h>

namespace {
constexpr unsigned long RECONNECT_INTERVAL_MS = 5000;
constexpr const char *STATUS_ONLINE = "online";
constexpr const char *STATUS_OFFLINE = "offline";
// Certificates are checked against the clock: before NTP sets it (any time
// earlier than 2023-11-14), a TLS handshake would fail, so it is not tried.
constexpr time_t MIN_VALID_TIME = 1700000000;
} // namespace

void MqttLink::begin(const Config &config) {
  config_ = config;
  stateTopic_ = String(config.topicPrefix) + "/state";
  cmdTopic_ = String(config.topicPrefix) + "/cmd";
  statusTopic_ = String(config.topicPrefix) + "/status";

  // A random suffix keeps two boards (or simulations) from kicking each
  // other off the broker by sharing a client ID.
  clientId_ = String(config.clientId) + "-" + String((uint32_t)random(0x10000), HEX);

  if (config.caCert) {
#if defined(ESP32)
    tlsClient_.setCACert(config.caCert);
#else
    trustAnchors_.append(config.caCert);
    tlsClient_.setTrustAnchors(&trustAnchors_);
#endif
    configTime(0, 0, "pool.ntp.org", "time.nist.gov"); // syncs once WiFi is up
    mqtt_.setClient(tlsClient_);
  } else {
    mqtt_.setClient(plainClient_);
  }
  mqtt_.setServer(config.host, config.port);
  mqtt_.setCallback([this](char *topic, uint8_t *payload, unsigned int length) {
    handleMessage(topic, payload, length);
  });
}

void MqttLink::loop() {
  if (WiFi.status() != WL_CONNECTED) return;

  if (!mqtt_.connected()) {
    unsigned long now = millis();
    if (hasAttempted_ && now - lastAttemptMs_ < RECONNECT_INTERVAL_MS) return;
    hasAttempted_ = true;
    lastAttemptMs_ = now;
    if (!connect()) return;
  }
  mqtt_.loop();
}

bool MqttLink::connect() {
  if (config_.caCert && time(nullptr) < MIN_VALID_TIME) {
    Serial.println("[mqtt] waiting for NTP time to validate the broker certificate");
    return false;
  }
  Serial.printf("[mqtt] connecting to %s:%u%s as %s\n", config_.host, config_.port,
                config_.caCert ? " (TLS)" : "", clientId_.c_str());

  const char *user = (config_.username && *config_.username) ? config_.username : nullptr;
  const char *pass = user ? config_.password : nullptr;

  // Last will: the broker publishes "offline" (retained) if we vanish.
  bool ok = mqtt_.connect(clientId_.c_str(), user, pass,
                          statusTopic_.c_str(), 1, true, STATUS_OFFLINE);
  if (!ok) {
    Serial.printf("[mqtt] connect failed (state %d), retrying in %lus\n",
                  mqtt_.state(), RECONNECT_INTERVAL_MS / 1000);
    return false;
  }

  mqtt_.publish(statusTopic_.c_str(), STATUS_ONLINE, true);
  mqtt_.subscribe(cmdTopic_.c_str());
  Serial.printf("[mqtt] connected, listening on %s\n", cmdTopic_.c_str());
  return true;
}

void MqttLink::handleMessage(char *topic, uint8_t *payload, unsigned int length) {
  if (cmdTopic_ != topic) return;

  JsonDocument doc;
  DeserializationError err = deserializeJson(doc, payload, length);
  if (err) {
    Serial.printf("[mqtt] command JSON error: %s\n", err.c_str());
    return;
  }

  JsonArrayConst joints = doc["joints"];
  if (joints.size() != JOINT_COUNT || !doc["gripper"].is<float>()) {
    Serial.println("[mqtt] command needs joints[3] and gripper, ignoring");
    return;
  }

  ArmState target;
  for (uint8_t i = 0; i < JOINT_COUNT; i++) target.joints[i] = joints[i].as<float>();
  target.gripper = doc["gripper"].as<float>();

  Serial.printf("[mqtt] command: base=%.0f shoulder=%.0f elbow=%.0f gripper=%.0f\n",
                target.joints[JOINT_BASE], target.joints[JOINT_SHOULDER],
                target.joints[JOINT_ELBOW], target.gripper);
  if (commandCallback_) commandCallback_(target);
}

bool MqttLink::publishState(const ArmState &state) {
  if (!mqtt_.connected()) return false;

  JsonDocument doc;
  JsonArray joints = doc["joints"].to<JsonArray>();
  for (uint8_t i = 0; i < JOINT_COUNT; i++) joints.add((int)state.joints[i]);
  doc["gripper"] = (int)state.gripper;

  char buf[96];
  size_t len = serializeJson(doc, buf, sizeof(buf));
  return mqtt_.publish(stateTopic_.c_str(), (const uint8_t *)buf, len, true);
}
