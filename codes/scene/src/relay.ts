import { engine } from '@dcl/sdk/ecs'
import { RECONNECT_DELAY_S, RELAY_URL, SCENE_TOKEN } from './config'

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

let socket: WebSocket | null = null
let reconnectIn = 0
let nextCommandId = 1

function connect() {
  const url = SCENE_TOKEN ? `${RELAY_URL}?token=${encodeURIComponent(SCENE_TOKEN)}` : RELAY_URL
  const ws = new WebSocket(url)
  socket = ws
  ws.onopen = () => {
    relay.connected = true
    relay.lastError = ''
    console.log(`relay: connected to ${RELAY_URL}`)
  }
  ws.onmessage = (event) => handleMessage(String(event.data))
  ws.onclose = () => {
    if (relay.connected) console.log('relay: disconnected')
    relay.connected = false
    relay.deviceOnline = false
    socket = null
    reconnectIn = RECONNECT_DELAY_S
  }
}

function handleMessage(data: string) {
  let msg: Envelope
  try {
    msg = JSON.parse(data)
  } catch {
    console.log(`relay: invalid message ${data}`)
    return
  }
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

// Sends the target pose to the arm. Returns false when not connected.
export function sendCommand(state: ArmState): boolean {
  if (!socket || !relay.connected) return false
  const envelope: Envelope = { type: 'command', id: String(nextCommandId++), state }
  socket.send(JSON.stringify(envelope))
  return true
}

// Connects now and reconnects after RECONNECT_DELAY_S whenever the socket closes.
export function startRelay() {
  connect()
  engine.addSystem((dt) => {
    if (socket) return
    reconnectIn -= dt
    if (reconnectIn <= 0) connect()
  })
}
