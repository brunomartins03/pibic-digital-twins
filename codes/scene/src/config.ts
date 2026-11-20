import { Vector3 } from '@dcl/sdk/math'

// Relay WebSocket endpoint. A deployed scene can only open wss:// URLs, so
// ws://localhost works in the local preview only.
export const RELAY_URL = 'ws://localhost:8080/ws/scene'
// Must match SCENE_TOKEN on the relay; empty when authentication is disabled.
export const SCENE_TOKEN = ''
export const RECONNECT_DELAY_S = 3

// Placement of the arm inside the parcel. The model is 17 cm tall, so it is
// scaled up to be visible next to an avatar.
export const ARM_POSITION = Vector3.create(8, 0, 8)
export const ARM_SCALE = 8

// Degrees added or removed by each UI button press.
export const COMMAND_STEP_DEG = 10
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
