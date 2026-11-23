import { Vector3 } from '@dcl/sdk/math'

// Relay base URL. The Catalyst nginx publishes the relay under /robotarm/; for
// a fully local preview use http://localhost:8080. A deployed scene can only
// open https:// and wss:// URLs.
export const RELAY_URL = 'https://kiriland.unb.br/robotarm'
// 'http' polls the relay and posts commands, for networks whose proxies do not
// forward WebSocket upgrades (the case of the project's server); 'websocket'
// keeps a connection open and receives updates as they happen.
export const RELAY_TRANSPORT: 'http' | 'websocket' = 'http'
export const POLL_INTERVAL_S = 0.2
// Must match SCENE_TOKEN on the relay; empty when authentication is disabled.
export const SCENE_TOKEN = ''
export const RECONNECT_DELAY_S = 3

// Placement of the arm in scene coordinates (metres from the south-west corner
// of parcel 0,0). Placeholder: the spot of the static arm model the twin
// replaced in campusFGA, on parcel 12,13.
export const ARM_POSITION = Vector3.create(205.12, 2.29, 208.82)
// Rotation around the vertical axis, in degrees. At 0 the arm reaches north (+z).
export const ARM_ROTATION_Y_DEG = -120
// The model is 17.2 cm tall; 5.4 gives the 0.94 m of the replaced model.
export const ARM_SCALE = 5.4

// Degrees added or removed by each UI button press.
export const COMMAND_STEP_DEG = 10
// Seconds during which UI commands build on the last commanded pose instead
// of the telemetry. The real arm crosses 180 degrees in under 2 s; the Wokwi
// simulation runs several times slower.
export const TARGET_HOLD_S = 10
// Speed at which the virtual joints follow the telemetry, in degrees/second.
export const FOLLOW_SPEED_DEG_S = 180

// Servo angle (0-180) to model angle: model = sign * (servo - rest).
// Placeholder values: calibrate against the physical arm.
export const JOINT_CALIBRATION = [
  { rest: 90, sign: 1 }, // base
  { rest: 90, sign: 1 }, // shoulder
  { rest: 90, sign: 1 } // elbow
]
export const GRIPPER_CALIBRATION = { rest: 90, sign: 1 }
// Jaw rotation range in model degrees; the jaws overlap below -15.
export const GRIPPER_LIMITS = { min: -15, max: 45 }
