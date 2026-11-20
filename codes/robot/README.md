# Robotic Arm Firmware (ESP32)

Drives 4 servos (base, shoulder, elbow, gripper) and keeps them in sync with
the [bridge server](../bridge-server) — and through it, the Decentraland
digital twin — in both directions.

## How the two control paths work together

- **Local**: same key interface as before — type `Q/A/W/S/E/D/R/F` + Enter
  over USB Serial (115200 baud) to nudge a joint by `STEP_DEG` degrees.
- **Remote**: the bridge relays `command` messages from the Decentraland
  scene as absolute target angles.

Both paths just set a *target* on the same `ArmController`; the servo motion
itself is applied smoothly in `update()` regardless of which path issued it.
Every time the arm's actual position changes (or at least every
`TELEMETRY_HEARTBEAT_MS`), the firmware publishes its current angles back to
the bridge, which is what keeps the physical arm and the digital twin
converged whichever one moved first.

**Don't manually move the servos by hand while powered.** They actively hold
position; forcing them fights that (risking stripped gears) and will desync
the twin until the next command overwrites the position. Use Serial keys or
the digital twin instead — which was the point of building this.

## Wiring

Default pins (see `config.example.h` to change):

| Joint    | GPIO |
|----------|------|
| Base     | 13   |
| Shoulder | 14   |
| Elbow    | 27   |
| Gripper  | 26   |

Picked to avoid ESP32 strapping pins (0, 2, 12, 15) and input-only pins
(34–39) on a typical DevKitC. Power the servos from a separate 5V supply
sized for stall current, not from the ESP32's onboard regulator — share
ground between the two supplies.

## Libraries (Arduino Library Manager)

- **ESP32Servo** (Kevin Harrington / madhephaestus) — `Servo.h` from AVR/ESP8266
  boards doesn't drive ESP32 hardware PWM correctly; this is the standard
  replacement.
- **WebSockets** (Markus Sattler / Links2004) — WebSocket client used to talk
  to the bridge server.
- **ArduinoJson** (Benoit Blanchon), **v7 or later** — the code uses the v7
  `JsonDocument` API.

Board support: install "esp32" by Espressif Systems via Boards Manager if
you haven't already.

## Setup

1. Copy `config.example.h` to `config.h` in this same folder.
2. Fill in `WIFI_SSID`, `WIFI_PASSWORD`, `BRIDGE_HOST` (your bridge server's
   LAN IP), `BRIDGE_PORT`, and `DEVICE_TOKEN` (must match the bridge's
   `DEVICE_TOKEN` env var).
3. Adjust pins in `config.h` if your wiring differs from the table above.
4. Open `robotic_arm.ino` in the Arduino IDE, select your ESP32 board, and
   upload.
5. Open the Serial Monitor at 115200 baud to watch WiFi/bridge connection
   status and use local key controls.

`config.h` is gitignored — it holds real credentials, don't commit it.

## Files

- `robotic_arm.ino` — setup/loop, wires local Serial input and remote bridge
  commands into `ArmController`.
- `ArmController.h/.cpp` — owns the 4 servos, smooth motion, key handling.
- `BridgeClient.h/.cpp` — WebSocket client for the bridge's `/ws/device`
  endpoint: sends telemetry, receives and parses commands.
- `config.example.h` — template for your local `config.h`.

## Protocol

Matches the bridge server's format exactly (see the bridge's README):

```json
{"type": "telemetry", "state": {"joints": [90, 90, 90], "gripper": 90}}
{"type": "command",   "state": {"joints": [45, 120, 60], "gripper": 0}}
```

`joints` is `[base, shoulder, elbow]` in degrees (0–180); `gripper` is also
in degrees here (0–180), not the 0–1 convention mentioned in the bridge
README's generic example — pick whichever convention you want, just keep
firmware and the Decentraland scene using the same one.

## A note on testing

This was written carefully against the documented APIs of ESP32Servo,
WebSocketsClient, and ArduinoJson v7, but **it hasn't been compiled** — this
environment doesn't have network access to the ESP32 toolchain/package
sources. Build it in the Arduino IDE (or PlatformIO) before flashing, and
expect to iron out minor issues like exact library method signatures if
your installed library versions differ from what's assumed here.
