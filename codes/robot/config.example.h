#pragma once

// Copy this file to config.h and fill in your real values.
// config.h is gitignored on purpose — never commit real WiFi credentials or
// the device token.

#define WIFI_SSID     "your-wifi-name"
#define WIFI_PASSWORD "your-wifi-password"

#define BRIDGE_HOST  "192.168.1.100"  // bridge server's IP or hostname
#define BRIDGE_PORT  8080
#define BRIDGE_PATH  "/ws/device"
#define DEVICE_TOKEN "changeme"        // must match DEVICE_TOKEN on the bridge server

// Servo GPIO pins. Picked to avoid ESP32 strapping pins (0, 2, 12, 15) and
// input-only pins (34-39) on a typical DevKitC board — adjust to match your
// wiring/board if different.
#define PIN_BASE     13
#define PIN_SHOULDER 14
#define PIN_ELBOW    27
#define PIN_GRIPPER  26

#define STEP_DEG               5     // degrees per local key press
#define MOVE_SPEED_DEG         4.0f  // max degrees moved per update() tick
#define TELEMETRY_HEARTBEAT_MS 500   // send telemetry at least this often even if idle
