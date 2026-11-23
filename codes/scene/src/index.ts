import { createArm } from './arm'
import { startRelay } from './relay'
import { setupUi } from './ui'

export function main() {
  createArm()
  startRelay()
  setupUi()
}
