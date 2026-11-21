#pragma once

#include <Arduino.h>
#include <PubSubClient.h>
#include <functional>

#if defined(ESP32)
#include <WiFi.h>
#else
#include <ESP8266WiFi.h>
#endif
#include <WiFiClientSecure.h>

#include "ArmController.h"

// MQTT connection to the broker shared with the relay server. Topics, under
// a configurable prefix (e.g. "robotarm"):
//
//   <prefix>/state   publish, retained   {"joints":[b,s,e],"gripper":g}
//   <prefix>/cmd     subscribe           same payload, target angles
//   <prefix>/status  publish, retained   "online" | "offline" (last will)
//
// The class knows nothing about servos: it parses commands and hands them
// to the callback registered with onCommand().
class MqttLink {
public:
  using CommandCallback = std::function<void(const ArmState &)>;

  struct Config {
    const char *host;
    uint16_t port;
    const char *clientId;
    const char *username; // empty for anonymous
    const char *password;
    const char *topicPrefix;
    const char *caCert; // PEM CA to connect over TLS; nullptr for plain TCP
  };

  void begin(const Config &config);

  // Call every loop() iteration: keeps the connection alive and reconnects
  // (without blocking) when WiFi is up but the broker connection dropped.
  void loop();

  // Publishes the state as a retained message. Returns false if offline.
  bool publishState(const ArmState &state);

  void onCommand(CommandCallback cb) { commandCallback_ = cb; }
  bool isConnected() { return mqtt_.connected(); }

private:
  WiFiClient plainClient_;
  WiFiClientSecure tlsClient_;
#if !defined(ESP32)
  BearSSL::X509List trustAnchors_;
#endif
  PubSubClient mqtt_;
  Config config_{};
  String clientId_;
  String stateTopic_, cmdTopic_, statusTopic_;
  CommandCallback commandCallback_;
  unsigned long lastAttemptMs_ = 0;
  bool hasAttempted_ = false;

  bool connect();
  void handleMessage(char *topic, uint8_t *payload, unsigned int length);
};
