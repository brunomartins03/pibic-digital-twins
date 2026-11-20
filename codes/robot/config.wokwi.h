#pragma once

// Configuration for the Wokwi simulation (ESP32 DevKit), selected by
// building with -DWOKWI_SIM (see Makefile). Wokwi runs in the cloud, so the
// broker must be reachable from the internet; the relay must use the same
// broker and topic prefix. Only dummy angles travel over this public broker.

#define WIFI_SSID     "Wokwi-GUEST"
#define WIFI_PASSWORD ""

#define MQTT_HOST         "broker.emqx.io"
#define MQTT_PORT         1883
#define MQTT_USERNAME     ""
#define MQTT_PASSWORD     ""
#define MQTT_CLIENT_ID    "robotarm-sim"
#define MQTT_TOPIC_PREFIX "dt-sim-7f3a9c"  // random, to avoid clashes on a public broker

// Same GPIO numbers as the real board, wired in simulation/diagram.json.
#define PIN_BASE     5
#define PIN_SHOULDER 4
#define PIN_ELBOW    14
#define PIN_GRIPPER  12

#define STEP_DEG                  5
#define MOVE_STEP_DEG             2.0f
#define MOVE_TICK_MS              20
#define TELEMETRY_MIN_INTERVAL_MS 100
