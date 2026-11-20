# Arm Digital Twin — Bridge Server

Sits between the physical arm (ESP32) and the Decentraland scene. Both sides
connect over plain WebSocket and exchange JSON messages ("envelopes").

```
ESP32 <--ws://.../ws/device-->  bridge  <--ws://.../ws/scene--> Decentraland scene
```

## Run

```bash
go build -o bridge-server .
DEVICE_TOKEN=changeme SCENE_TOKEN=changeme2 ./bridge-server
```

Env vars (all optional for local dev, strongly recommended for anything
public-facing):

- `BRIDGE_ADDR` — listen address, default `:8080`
- `DEVICE_TOKEN` — shared secret the ESP32 must send as `?token=` on `/ws/device`
- `SCENE_TOKEN` — shared secret the scene must send as `?token=` on `/ws/scene`

If either token is unset, that endpoint accepts connections with no auth —
fine on localhost, not fine once this is reachable from the internet.

## Endpoints

- `ws://host:8080/ws/device?token=...` — the ESP32 connects here
- `ws://host:8080/ws/scene?token=...` — the Decentraland scene connects here
- `GET /api/state` — last known arm state as JSON (204 if none yet)
- `GET /healthz` — `{"ok":true,"deviceConnected":bool}`

## Message format (both directions, both endpoints)

```json
{
  "type": "telemetry",      // or "command" | "hello" | "ack" | "error"
  "state": {
    "joints": [10, 20, 30, 40],
    "gripper": 0.5
  },
  "id": "optional-correlation-id"
}
```

- **Device → bridge**: send `type: "telemetry"` whenever joint angles change
  (e.g. every 50–100ms, or on-change with a max interval as a heartbeat).
  The bridge stores it and fans it out to every connected scene.
- **Bridge → scene**: forwards `type: "telemetry"` as it arrives, and sends
  one immediately on connect with the last known state.
- **Scene → bridge → device**: send `type: "command"` with target joint
  angles. The bridge relays it to the device verbatim and replies to the
  scene with `type: "ack"` (or `type: "error"` if no device is connected).

`joints` is just an ordered array — agree on the order/count (e.g.
base, shoulder, elbow, wrist) between firmware and scene code and keep it
consistent. Angles are in degrees; `gripper` is 0–1 (closed–open) but that's
a convention, not enforced by the server.

## Notes on this implementation

- Only one device connection is treated as authoritative at a time — if a
  second one connects, the first is dropped. Multiple scene connections are
  fully supported (so several players see the same synced arm).
- WebSocket ping/pong keepalive runs every ~54s to detect dead connections
  through NATs/proxies.
- `CheckOrigin` currently allows all origins — lock this down to your
  deployed scene's origin before going live.
- State lives in memory only; restart the bridge and last-known-state resets
  until the device sends telemetry again.
