#include "BridgeClient.h"
#include <ArduinoJson.h>

BridgeClient *BridgeClient::instance_ = nullptr;

void BridgeClient::begin(const char *host, uint16_t port, const char *path, const char *token) {
  instance_ = this;

  String fullPath = String(path);
  if (token != nullptr && strlen(token) > 0) {
    fullPath += "?token=";
    fullPath += token;
  }

  ws_.begin(host, port, fullPath);
  ws_.onEvent(eventTrampoline);
  ws_.setReconnectInterval(5000);
}

void BridgeClient::loop() {
  ws_.loop();
}

void BridgeClient::eventTrampoline(WStype_t type, uint8_t *payload, size_t length) {
  if (instance_) instance_->handleEvent(type, payload, length);
}

void BridgeClient::handleEvent(WStype_t type, uint8_t *payload, size_t length) {
  switch (type) {
    case WStype_CONNECTED:
      connected_ = true;
      Serial.println("[bridge] connected");
      ws_.sendTXT("{\"type\":\"hello\",\"message\":\"esp32-arm online\"}");
      break;

    case WStype_DISCONNECTED:
      connected_ = false;
      Serial.println("[bridge] disconnected, will auto-retry");
      break;

    case WStype_TEXT:
      handleMessage(payload, length);
      break;

    case WStype_ERROR:
      Serial.println("[bridge] websocket error");
      break;

    default:
      break;
  }
}

void BridgeClient::handleMessage(uint8_t *payload, size_t length) {
  JsonDocument doc;
  DeserializationError err = deserializeJson(doc, payload, length);
  if (err) {
    Serial.print("[bridge] JSON parse error: ");
    Serial.println(err.c_str());
    return;
  }

  const char *type = doc["type"] | "";

  if (strcmp(type, "command") == 0) {
    if (!commandCallback_) return;

    JsonArray jointsArr = doc["state"]["joints"];
    if (jointsArr.isNull() || jointsArr.size() < JOINT_COUNT) {
      Serial.println("[bridge] command missing/short joints array, ignoring");
      return;
    }

    ArmState s;
    for (uint8_t i = 0; i < JOINT_COUNT; i++) {
      s.joints[i] = jointsArr[i].as<float>();
    }
    s.gripper = doc["state"]["gripper"] | 90.0f;

    commandCallback_(s);

  } else if (strcmp(type, "error") == 0) {
    Serial.print("[bridge] server error: ");
    Serial.println((const char *)(doc["message"] | ""));
  }
  // "ack" and anything else: nothing to do, just informational.
}

void BridgeClient::sendTelemetry(const ArmState &state) {
  if (!connected_) return;

  JsonDocument doc;
  doc["type"] = "telemetry";
  JsonObject st = doc["state"].to<JsonObject>();
  JsonArray joints = st["joints"].to<JsonArray>();
  for (uint8_t i = 0; i < JOINT_COUNT; i++) joints.add(state.joints[i]);
  st["gripper"] = state.gripper;

  String out;
  serializeJson(doc, out);
  ws_.sendTXT(out);
}
