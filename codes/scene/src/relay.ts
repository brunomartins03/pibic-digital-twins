import { engine, executeTask } from '@dcl/sdk/ecs'
import { POLL_INTERVAL_S, RECONNECT_DELAY_S, RELAY_TRANSPORT, RELAY_URL, SCENE_TOKEN } from './config'

// Joint angles in degrees: [base, shoulder, elbow] and the gripper, 0-180.
export type ArmState = { joints: number[]; gripper: number }

type Envelope = {
  type: 'telemetry' | 'status' | 'ack' | 'error' | 'command'
  id?: string
  state?: ArmState
  deviceOnline?: boolean
  message?: string
}

// Connection status shown by the UI; `state` is the last telemetry received.
export const relay = {
  connected: false,
  deviceOnline: false,
  state: null as ArmState | null,
  lastError: ''
}

const query = SCENE_TOKEN ? `?token=${encodeURIComponent(SCENE_TOKEN)}` : ''
let nextCommandId = 1

function handleMessage(msg: Envelope) {
  switch (msg.type) {
    case 'telemetry':
      if (msg.state) relay.state = msg.state
      break
    case 'status':
      relay.deviceOnline = msg.deviceOnline === true
      break
    case 'ack':
      relay.lastError = ''
      break
    case 'error':
      relay.lastError = msg.message ?? 'unknown error'
      console.log(`relay: command ${msg.id} rejected: ${relay.lastError}`)
      break
  }
}

function setConnected(connected: boolean) {
  if (connected !== relay.connected) console.log(`relay: ${connected ? 'connected to' : 'disconnected from'} ${RELAY_URL}`)
  relay.connected = connected
  if (!connected) relay.deviceOnline = false
}

// --- WebSocket transport: updates are pushed by the relay -------------------

let socket: WebSocket | null = null

function connectWebSocket() {
  const ws = new WebSocket(`${RELAY_URL.replace(/^http/, 'ws')}/ws/scene${query}`)
  socket = ws
  ws.onopen = () => setConnected(true)
  ws.onmessage = (event) => {
    try {
      handleMessage(JSON.parse(String(event.data)))
    } catch {
      console.log(`relay: invalid message ${event.data}`)
    }
  }
  ws.onclose = () => {
    setConnected(false)
    socket = null
  }
}

function startWebSocket() {
  let reconnectIn = 0
  engine.addSystem((dt) => {
    if (socket) return
    reconnectIn -= dt
    if (reconnectIn <= 0) {
      reconnectIn = RECONNECT_DELAY_S
      connectWebSocket()
    }
  })
}

// --- HTTP transport: the scene polls /healthz and /api/state ----------------

async function poll() {
  try {
    const [health, state] = await Promise.all([fetch(`${RELAY_URL}/healthz`), fetch(`${RELAY_URL}/api/state`)])
    if (!health.ok || !state.ok) throw new Error(`HTTP ${health.status}/${state.status}`)
    setConnected(true)
    handleMessage({ type: 'status', deviceOnline: (await health.json()).deviceOnline })
    if (state.status !== 204) handleMessage({ type: 'telemetry', state: await state.json() })
    return true
  } catch {
    setConnected(false)
    return false
  }
}

function startPolling() {
  let nextPollIn = 0
  let polling = false
  engine.addSystem((dt) => {
    nextPollIn -= dt
    if (polling || nextPollIn > 0) return
    polling = true
    executeTask(async () => {
      nextPollIn = (await poll()) ? POLL_INTERVAL_S : RECONNECT_DELAY_S
      polling = false
    })
  })
}

async function postCommand(envelope: Envelope) {
  try {
    const response = await fetch(`${RELAY_URL}/api/command${query}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(envelope)
    })
    handleMessage(await response.json())
  } catch {
    handleMessage({ type: 'error', id: envelope.id, message: 'failed to reach the relay' })
  }
}

// --- Public API -------------------------------------------------------------

// Sends the target pose to the arm. Returns false when not connected.
export function sendCommand(state: ArmState): boolean {
  if (!relay.connected) return false
  const envelope: Envelope = { type: 'command', id: String(nextCommandId++), state }
  if (RELAY_TRANSPORT === 'http') executeTask(() => postCommand(envelope))
  else socket?.send(JSON.stringify(envelope))
  return true
}

// Starts following the relay with the configured transport, retrying every
// RECONNECT_DELAY_S seconds while it is unreachable.
export function startRelay() {
  if (RELAY_TRANSPORT === 'http') startPolling()
  else startWebSocket()
}
