import { Color4 } from '@dcl/sdk/math'
import ReactEcs, { Button, Label, ReactEcsRenderer, UiEntity } from '@dcl/sdk/react-ecs'
import { COMMAND_STEP_DEG, TARGET_HOLD_S } from './config'
import { ArmState, relay, sendCommand } from './relay'

const CONTROLS = ['Base', 'Shoulder', 'Elbow', 'Gripper']
const HOME: ArmState = { joints: [90, 90, 90], gripper: 90 }

const clampAngle = (v: number) => Math.max(0, Math.min(180, v))

// Last pose commanded from this panel. While the arm is still moving toward
// it, new commands build on it instead of on the telemetry, so a click on one
// joint does not stop the motion of the others.
let lastTarget: ArmState | null = null
let lastTargetAt = 0

function command(target: ArmState) {
  if (sendCommand(target)) {
    lastTarget = target
    lastTargetAt = Date.now()
  }
}

function nudge(control: number, delta: number) {
  const recent = lastTarget && Date.now() - lastTargetAt < TARGET_HOLD_S * 1000
  const base = recent ? lastTarget : relay.state
  if (!base) return
  const joints = [...base.joints]
  let gripper = base.gripper
  if (control < 3) joints[control] = clampAngle(joints[control] + delta)
  else gripper = clampAngle(gripper + delta)
  command({ joints, gripper })
}

const angleOf = (control: number) => {
  if (!relay.state) return '--'
  return `${Math.round(control < 3 ? relay.state.joints[control] : relay.state.gripper)}°`
}

const StepButton = (props: { control: number; delta: number }) => (
  <Button
    value={props.delta > 0 ? '+' : '-'}
    variant="secondary"
    fontSize={18}
    uiTransform={{ width: 40, height: 32, margin: '0 4px' }}
    onMouseDown={() => nudge(props.control, props.delta)}
  />
)

const ControlRow = (props: { control: number }) => (
  <UiEntity uiTransform={{ flexDirection: 'row', alignItems: 'center', height: 40 }}>
    <Label value={CONTROLS[props.control]} fontSize={16} uiTransform={{ width: 90 }} textAlign="middle-left" />
    <StepButton control={props.control} delta={-COMMAND_STEP_DEG} />
    <Label value={angleOf(props.control)} fontSize={16} uiTransform={{ width: 60 }} />
    <StepButton control={props.control} delta={COMMAND_STEP_DEG} />
  </UiEntity>
)

const statusText = () => {
  if (!relay.connected) return 'Relay: disconnected'
  return `Relay: connected | Arm: ${relay.deviceOnline ? 'online' : 'offline'}`
}

const ArmPanel = () => (
  <UiEntity
    uiTransform={{
      positionType: 'absolute',
      position: { right: 16, top: 120 },
      width: 280,
      flexDirection: 'column',
      alignItems: 'center',
      padding: 12
    }}
    uiBackground={{ color: Color4.create(0.1, 0.1, 0.12, 0.85) }}
  >
    <Label value="Robotic arm" fontSize={20} uiTransform={{ height: 32 }} />
    <Label value={statusText()} fontSize={14} uiTransform={{ height: 24 }} />
    {CONTROLS.map((_, i) => (
      <ControlRow control={i} />
    ))}
    <Button
      value="Home"
      variant="primary"
      fontSize={16}
      uiTransform={{ width: 120, height: 34, margin: '8px 0 0 0' }}
      onMouseDown={() => command(HOME)}
    />
    {relay.lastError ? (
      <Label value={`Error: ${relay.lastError}`} fontSize={13} color={Color4.Red()} uiTransform={{ height: 24 }} />
    ) : null}
  </UiEntity>
)

export function setupUi() {
  ReactEcsRenderer.setUiRenderer(ArmPanel)
}
