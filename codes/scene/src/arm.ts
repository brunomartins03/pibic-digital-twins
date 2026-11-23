import { engine, Entity, GltfContainer, TextShape, Transform } from '@dcl/sdk/ecs'
import { Quaternion, Vector3 } from '@dcl/sdk/math'
import manifest from '../assets/models/arm/manifest.json'
import {
  ARM_POSITION,
  ARM_ROTATION_Y_DEG,
  ARM_SCALE,
  FOLLOW_SPEED_DEG_S,
  GRIPPER_CALIBRATION,
  GRIPPER_LIMITS,
  JOINT_CALIBRATION
} from './config'
import { ArmState, relay } from './relay'

const MODEL_DIR = 'assets/models/arm'

// manifest.json uses glTF coordinates (right-handed). The explorer is
// left-handed and mirrors imported models on x, so positions flip x and
// rotation axes flip y and z, which keeps the manifest's angle signs.
const toScenePoint = ([x, y, z]: number[]) => Vector3.create(-x, y, z)
const toSceneAxis = ([x, y, z]: number[]) => Vector3.create(x, -y, -z)

type Joint = { entity: Entity; axis: Vector3; field: string }

const joints: Joint[] = []
// Pose currently displayed; moves toward relay.state at FOLLOW_SPEED_DEG_S.
let shown: ArmState | null = null

function modelAngle(field: string, state: ArmState): number {
  const gripper = () => {
    const angle = GRIPPER_CALIBRATION.sign * (state.gripper - GRIPPER_CALIBRATION.rest)
    return Math.min(GRIPPER_LIMITS.max, Math.max(GRIPPER_LIMITS.min, angle))
  }
  switch (field) {
    case 'gripper':
      return gripper()
    case 'gripper (mirrored)':
      return -gripper()
    default: {
      const i = Number(/^joints\[(\d)\]$/.exec(field)?.[1])
      const { rest, sign } = JOINT_CALIBRATION[i]
      return sign * (state.joints[i] - rest)
    }
  }
}

const approach = (from: number, to: number, maxStep: number) =>
  from + Math.max(-maxStep, Math.min(maxStep, to - from))

function followSystem(dt: number) {
  const target = relay.state
  if (!target) return
  const step = FOLLOW_SPEED_DEG_S * dt
  shown = shown
    ? {
        joints: shown.joints.map((angle, i) => approach(angle, target.joints[i], step)),
        gripper: approach(shown.gripper, target.gripper, step)
      }
    : { joints: [...target.joints], gripper: target.gripper }
  for (const joint of joints) {
    Transform.getMutable(joint.entity).rotation = Quaternion.fromAngleAxis(modelAngle(joint.field, shown), joint.axis)
  }
}

// Builds the kinematic chain described in manifest.json: one entity per
// part, parented to the part it is mounted on, with its origin on the pivot.
export function createArm() {
  const root = engine.addEntity()
  const rotation = Quaternion.fromEulerDegrees(0, ARM_ROTATION_Y_DEG, 0)
  Transform.create(root, {
    position: ARM_POSITION,
    rotation,
    scale: Vector3.create(ARM_SCALE, ARM_SCALE, ARM_SCALE)
  })

  const entities = new Map<string, Entity>()
  for (const part of manifest.parts) {
    const entity = engine.addEntity()
    const position = toScenePoint(part.offsetFromParent)
    Transform.create(entity, { position, parent: part.parent ? entities.get(part.parent) : root })
    GltfContainer.create(entity, { src: `${MODEL_DIR}/${part.file}` })
    entities.set(part.name, entity)
    if (part.axis && part.relayField) {
      joints.push({ entity, axis: toSceneAxis(part.axis), field: part.relayField })
    }
  }

  const credit = engine.addEntity()
  // Behind the base, facing the same way as the arm.
  const creditOffset = Vector3.rotate(Vector3.create(0, 0.3, -0.8), rotation)
  Transform.create(credit, { position: Vector3.add(ARM_POSITION, creditOffset), rotation })
  TextShape.create(credit, {
    text: '"4-Axis Arm Assembly" by samoubala\nCC-BY-4.0',
    fontSize: 1
  })

  engine.addSystem(followSystem)
}
