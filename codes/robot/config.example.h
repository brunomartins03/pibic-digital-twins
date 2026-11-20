#pragma once

// Copy this file to config.h and fill in your values. config.h is
// gitignored: never commit real WiFi or broker credentials.

#define WIFI_SSID     "your-wifi-name"
#define WIFI_PASSWORD "your-wifi-password"

// MQTT broker shared with the relay server (usually the relay's machine).
#define MQTT_HOST         "192.168.1.100"
#define MQTT_PORT         1883
#define MQTT_USERNAME     ""          // empty for anonymous access
#define MQTT_PASSWORD     ""
#define MQTT_CLIENT_ID    "robotarm"  // a random suffix is appended
#define MQTT_TOPIC_PREFIX "robotarm"  // must match the relay's MQTT_TOPIC_PREFIX

// Servo GPIOs for a NodeMCU / Wemos D1 mini: D1, D2, D5, D6. These avoid
// the boot strapping pins (GPIO 0, 2, 15) and GPIO 16 (no PWM).
#define PIN_BASE     5   // D1
#define PIN_SHOULDER 4   // D2
#define PIN_ELBOW    14  // D5
#define PIN_GRIPPER  12  // D6

#define STEP_DEG                5      // degrees per local key press
#define MOVE_STEP_DEG           2.0f   // max degrees per motion tick
#define MOVE_TICK_MS            20     // motion tick (2 deg / 20 ms = 100 deg/s)
#define TELEMETRY_MIN_INTERVAL_MS 100  // rate limit for state publishing
