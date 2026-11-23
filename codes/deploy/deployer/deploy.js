// Publishes the environment's scenes to the Catalyst, signed by a throwaway
// wallet (the Catalyst runs with IGNORE_BLOCKCHAIN_ACCESS_CHECKS=true, so any
// address may deploy to any parcel). A scene whose files and scene.json match
// the entity already active at its parcels is skipped, so restarts are cheap.
//
// The libraries are the ones the robot scene's @dcl/sdk installs, and the
// files are selected with the same function `sdk-commands deploy` uses.
const { execFileSync } = require('child_process')
const fs = require('fs')
const path = require('path')
const { createRequire } = require('module')

const ROBOT_DIR = '/work/robot-scene'
const SCENES = [ROBOT_DIR, '/work/campusFGA']
const req = createRequire(path.join(ROBOT_DIR, 'package.json'))
const { DeploymentBuilder, createContentClient } = req('dcl-catalyst-client')
const { Authenticator } = req('@dcl/crypto')
const { ethSign, createUnsafeIdentity } = req('@dcl/crypto/dist/crypto')
const { hexToBytes } = req('eth-connect')
const { getPublishableFiles } = req('@dcl/sdk-commands/dist/logic/project-files')

const CONTENT_URL = process.env.CONTENT_URL.replace(/\/$/, '')

// A fresh content server first bootstraps from the public Catalysts, and only
// accepts deployments once its state is "Syncing".
async function waitForCatalyst() {
  let state = 'unreachable'
  for (let i = 0; i < 360; i++) {
    try {
      const status = await (await fetch(`${CONTENT_URL}/status`)).json()
      state = status.synchronizationStatus.synchronizationState
      if (state === 'Syncing') return
    } catch (e) {}
    if (i % 12 === 0) console.log(`waiting for the Catalyst (${state})`)
    await new Promise((r) => setTimeout(r, 5000))
  }
  throw new Error(`Catalyst at ${CONTENT_URL} still ${state} after 30 min`)
}

// The relay URL and token are compiled into the scene, so they are written
// into src/config.ts before building.
function buildRobotScene() {
  const file = path.join(ROBOT_DIR, 'src/config.ts')
  let src = fs.readFileSync(file, 'utf8')
  for (const [name, value] of [
    ['RELAY_URL', process.env.RELAY_URL],
    ['SCENE_TOKEN', process.env.SCENE_TOKEN || '']
  ]) {
    // Any value: a restarted container finds the file already rewritten.
    const re = new RegExp(`^export const ${name} = .*$`, 'm')
    if (!re.test(src)) throw new Error(`${name} not found in ${file}`)
    src = src.replace(re, `export const ${name} = ${JSON.stringify(value)}`)
  }
  fs.writeFileSync(file, src)
  console.log(`robot scene: RELAY_URL=${process.env.RELAY_URL}`)
  execFileSync('npx', ['sdk-commands', 'build', '--production'], { cwd: ROBOT_DIR, stdio: 'inherit' })
}

const fsComponent = { fs: { readFile: fs.promises.readFile } }

// Entity metadata as `sdk-commands deploy` writes it: scene.json plus the SDK
// version. A snapshot carries the metadata it was exported with.
function readMetadata(dir) {
  const exported = path.join(dir, '.entity-metadata.json')
  if (fs.existsSync(exported)) return JSON.parse(fs.readFileSync(exported, 'utf8'))
  const sceneJson = JSON.parse(fs.readFileSync(path.join(dir, 'scene.json'), 'utf8'))
  return { sdkVersion: req('@dcl/sdk/package.json').version, ...sceneJson }
}

async function deployScene(dir, identity, client) {
  const sceneJson = readMetadata(dir)
  const title = sceneJson.display.title
  const files = new Map()
  for (const file of await getPublishableFiles(fsComponent, dir)) {
    files.set(file.replace(/\\/g, '/'), new Uint8Array(fs.readFileSync(path.join(dir, file))))
  }
  const { entityId, files: entityFiles } = await DeploymentBuilder.buildEntity({
    type: 'scene',
    pointers: sceneJson.scene.parcels,
    files,
    metadata: sceneJson
  })

  const entity = JSON.parse(Buffer.from(entityFiles.get(entityId)).toString())
  const res = await fetch(`${CONTENT_URL}/entities/active`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ pointers: [sceneJson.scene.base] })
  })
  const [active] = await res.json()
  if (active && sameDeployment(active, entity)) {
    console.log(`${title}: up to date (${active.id})`)
    return
  }

  const authChain = Authenticator.createSimpleAuthChain(
    entityId,
    identity.address,
    ethSign(hexToBytes(identity.privateKey), entityId)
  )
  const response = await client.deploy({ entityId, files: entityFiles, authChain }, { timeout: 600000 })
  if (response.status !== 200) throw new Error(`${title}: deploy failed (${response.status}) ${await response.text()}`)
  console.log(`${title}: deployed ${entityId} to ${sceneJson.scene.parcels.length} parcel(s) from ${sceneJson.scene.base}`)
}

function sameDeployment(a, b) {
  const content = (e) => JSON.stringify(e.content.map((c) => `${c.file}:${c.hash}`).sort())
  const pointers = (e) => JSON.stringify([...e.pointers].sort())
  return (
    content(a) === content(b) &&
    pointers(a) === pointers(b) &&
    JSON.stringify(a.metadata) === JSON.stringify(b.metadata)
  )
}

async function main() {
  await waitForCatalyst()
  buildRobotScene()
  const identity = createUnsafeIdentity()
  const client = createContentClient({
    url: CONTENT_URL,
    fetcher: { fetch: (url, { timeout, ...init } = {}) => fetch(url, init) }
  })
  for (const dir of SCENES) await deployScene(dir, identity, client)
}

main().catch((e) => {
  console.error(e)
  process.exit(1)
})
