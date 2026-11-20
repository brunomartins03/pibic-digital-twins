#pragma once

#include <Arduino.h>
#include <WebSocketsClient.h>
#include <functional>
#include "ArmController.h"

// Wraps the WebSocket connection to the bridge server: sends telemetry,
// receives commands, and handles reconnection. This class doesn't know
// anything about servos — it just speaks the bridge's JSON protocol and
// hands parsed commands to whoever registered a callback via onCommand().
class BridgeClient {
public:
  using CommandCallback = std::function<void(const ArmState &)>;

  void begin(const char *host, uint16_t port, const char *path, const char *token);
  void loop();
  void sendTelemetry(const ArmState &state);
  void onCommand(CommandCallback cb) { commandCallback_ = cb; }
  bool isConnected() const { return connected_; }

private:
  WebSocketsClient ws_;
  CommandCallback commandCallback_;
  bool connected_ = false;

  void handleEvent(WStype_t type, uint8_t *payload, size_t length);
  void handleMessage(uint8_t *payload, size_t length);

  // WebSocketsClient's onEvent takes a plain function pointer, not a member
  // function, so we route through a static trampoline back to the instance.
  static BridgeClient *instance_;
  static void eventTrampoline(WStype_t type, uint8_t *payload, size_t length);
};
