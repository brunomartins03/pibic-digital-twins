// Splits model2.glb ("4-Axis Arm Assembly" by samoubala, CC-BY-4.0) into one
// GLB per moving part, each with its origin on the joint pivot, so the
// Decentraland scene can rebuild the kinematic chain with parented entities.
//
// Output (in ../parts/):
//   <part>.glb      one rigid group per joint
//   manifest.json   parent, pivot, offset from parent and rotation axis
//
// All output coordinates use a canonical frame: the base stands on y = 0,
// the turntable axis is the y axis, and the arm reaches toward +z, so the
// shoulder and elbow rotate about the x axis.
//
// Usage: npm run split [-- --input ../model2.glb --out ../parts --max-triangles 10000]

import fs from 'node:fs';
import path from 'node:path';
import { parseArgs } from 'node:util';
import { Logger, NodeIO, getBounds } from '@gltf-transform/core';
import { cloneDocument, dedup, prune, simplify, weld } from '@gltf-transform/functions';
import { MeshoptSimplifier } from 'meshoptimizer';

// --- Model description ------------------------------------------------------
// Parts are the "Object_N" nodes of model2.glb (each wraps one mesh).
// Pivots were measured from the servo output shafts, in model coordinates
// (meters), and are reused here unchanged.

const PITCH_AXIS = [0.659, 0, -0.752]; // shared axis of shoulder, elbow and wrist servos

const PARTS = [
  {
    name: 'base',
    parent: null,
    relayField: null,
    // Stationary box plus the turntable bearings and axles.
    objects: ['Object_30', 'Object_15', 'Object_24', 'Object_39', 'Object_48',
              'Object_21', 'Object_51', 'Object_57', 'Object_60'],
    pivot: 'floor', // base center on the ground
  },
  {
    name: 'turntable',
    parent: 'base',
    relayField: 'joints[0]',
    objects: ['Object_54', 'Object_6', 'Object_9'], // plate, bracket, shoulder servo
    pivot: 'yaw', // turntable center, at the plate's lower face
    axis: 'yaw',
  },
  {
    name: 'upper_arm',
    parent: 'turntable',
    relayField: 'joints[1]',
    objects: ['Object_63', 'Object_18'], // arm, elbow servo
    pivot: [0.0775, 0.202, -0.0618], // shoulder servo shaft
    axis: 'pitch',
  },
  {
    name: 'forearm',
    parent: 'upper_arm',
    relayField: 'joints[2]',
    // Arm, wrist servo, claw body, idler pin, gripper servo. The wrist joint
    // has no servo on the physical arm, so it is kept rigid here.
    objects: ['Object_12', 'Object_33', 'Object_36', 'Object_42', 'Object_27'],
    pivot: [0.0341, 0.2584, -0.0817], // elbow servo shaft
    axis: 'pitch',
  },
  {
    name: 'jaw_driven',
    parent: 'forearm',
    relayField: 'gripper',
    objects: ['Object_3'], // gear on the gripper servo shaft
    pivot: [-0.0519, 0.3009, -0.1798], // gripper servo shaft
    axis: 'yaw',
  },
  {
    name: 'jaw_idler',
    parent: 'forearm',
    relayField: 'gripper (mirrored)',
    objects: ['Object_45'], // gear meshing with the driven one
    pivot: 'idler', // center of the idler pin (Object_42)
    axis: 'yaw',
  },
];

// --- Small column-major 4x4 matrix helpers (glTF convention) -----------------

const mul = (a, b) => {
  const out = new Array(16).fill(0);
  for (let c = 0; c < 4; c++)
    for (let r = 0; r < 4; r++)
      for (let k = 0; k < 4; k++) out[c * 4 + r] += a[k * 4 + r] * b[c * 4 + k];
  return out;
};
const translation = ([x, y, z]) => [1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, x, y, z, 1];
const rotationY = (rad) => {
  const c = Math.cos(rad), s = Math.sin(rad);
  return [c, 0, -s, 0, 0, 1, 0, 0, s, 0, c, 0, 0, 0, 0, 1];
};
const applyPoint = (m, [x, y, z]) => [0, 1, 2].map((r) => m[r] * x + m[4 + r] * y + m[8 + r] * z + m[12 + r]);
const applyDir = (m, [x, y, z]) => [0, 1, 2].map((r) => m[r] * x + m[4 + r] * y + m[8 + r] * z);
const sub = (a, b) => a.map((v, i) => v - b[i]);
const round = (v, digits = 4) => v.map((x) => +x.toFixed(digits) + 0); // + 0 drops -0

// --- Helpers ----------------------------------------------------------------

function objectNode(doc, name) {
  const node = doc.getRoot().listNodes().find((n) => n.getName() === name);
  if (!node) throw new Error(`node ${name} not found`);
  return node;
}

function triangleCount(doc) {
  let total = 0;
  for (const node of doc.getRoot().listNodes()) {
    for (const prim of node.getMesh()?.listPrimitives() ?? []) {
      total += (prim.getIndices()?.getCount() ?? prim.getAttribute('POSITION').getCount()) / 3;
    }
  }
  return total;
}

// Canonical frame: turntable center to the origin, floor to y = 0, and a yaw
// that puts the pitch axis exactly on the x axis with the arm reaching toward
// +z (the wrist servo only disambiguates the direction).
function canonicalFrame(doc) {
  const floorY = getBounds(objectNode(doc, 'Object_30')).min[1];
  const plate = getBounds(objectNode(doc, 'Object_54'));
  const center = [(plate.min[0] + plate.max[0]) / 2, 0, (plate.min[2] + plate.max[2]) / 2];
  const wrist = getBounds(objectNode(doc, 'Object_33'));
  const reach = [(wrist.min[0] + wrist.max[0]) / 2 - center[0], (wrist.min[2] + wrist.max[2]) / 2 - center[2]];
  // Horizontal direction perpendicular to the pitch axis, on the reach side.
  let forward = [-PITCH_AXIS[2], PITCH_AXIS[0]];
  if (forward[0] * reach[0] + forward[1] * reach[1] < 0) forward = forward.map((v) => -v);
  const yaw = Math.atan2(forward[0], forward[1]); // angle from +z toward +x
  const frame = mul(rotationY(-yaw), translation([-center[0], -floorY, -center[2]]));
  return { frame, plate };
}

function resolvePivot(part, doc, frame, plate) {
  if (part.pivot === 'floor') return [0, 0, 0];
  if (part.pivot === 'yaw') return [0, applyPoint(frame, plate.min)[1], 0];
  if (part.pivot === 'idler') {
    const pin = getBounds(objectNode(doc, 'Object_42'));
    const c = [(pin.min[0] + pin.max[0]) / 2, (pin.min[1] + pin.max[1]) / 2, (pin.min[2] + pin.max[2]) / 2];
    return applyPoint(frame, c);
  }
  return applyPoint(frame, part.pivot);
}

// Builds one GLB containing only the part's meshes, re-expressed relative to
// the pivot: a single root node at the origin whose children keep their
// original relative placement.
function extractPart(src, part, frame, pivot) {
  const doc = cloneDocument(src).setLogger(new Logger(Logger.Verbosity.WARN));
  const root = doc.getRoot();
  const toPivot = mul(translation(pivot.map((v) => -v)), frame);

  const partRoot = doc.createNode(part.name);
  for (const objName of part.objects) {
    const obj = objectNode(doc, objName);
    obj.traverse((node) => {
      if (!node.getMesh()) return;
      const child = doc.createNode(objName).setMesh(node.getMesh());
      child.setMatrix(mul(toPivot, node.getWorldMatrix()));
      partRoot.addChild(child);
    });
  }

  const keep = new Set([partRoot, ...partRoot.listChildren()]);
  for (const node of root.listNodes()) if (!keep.has(node)) node.dispose();
  for (const scene of root.listScenes()) scene.dispose();
  doc.createScene(part.name).addChild(partRoot);
  root.setDefaultScene(root.listScenes()[0]);
  return doc;
}

// --- Main -------------------------------------------------------------------

const { values: args } = parseArgs({
  options: {
    input: { type: 'string', default: '../model2.glb' },
    out: { type: 'string', default: '../parts' },
    'max-triangles': { type: 'string', default: '10000' },
  },
});
const maxTriangles = Number(args['max-triangles']);

await MeshoptSimplifier.ready;
const io = new NodeIO();
const src = await io.read(args.input);

const assigned = PARTS.flatMap((p) => p.objects);
const allObjects = src.getRoot().listNodes().map((n) => n.getName()).filter((n) => /^Object_\d+$/.test(n));
const missing = allObjects.filter((n) => !assigned.includes(n));
const duplicated = assigned.filter((n, i) => assigned.indexOf(n) !== i);
if (missing.length || duplicated.length) {
  throw new Error(`part assignment error: missing=${missing} duplicated=${duplicated}`);
}

const { frame, plate } = canonicalFrame(src);
const pitchAxis = round(applyDir(frame, PITCH_AXIS), 3);
const axes = { yaw: [0, 1, 0], pitch: pitchAxis };

const parts = PARTS.map((part) => {
  const pivot = resolvePivot(part, src, frame, plate);
  return { part, pivot, doc: extractPart(src, part, frame, pivot) };
});

// Simplify every part with the same vertex ratio, always starting from the
// welded originals, lowering the ratio until the total fits the budget.
const countAll = () => parts.reduce((sum, p) => sum + triangleCount(p.doc), 0);
const original = countAll();
for (const p of parts) await p.doc.transform(weld(), dedup());
const welded = parts.map((p) => p.doc);
let ratio = 1;
let total = countAll();
while (total > maxTriangles && ratio > 0.05) {
  ratio *= Math.max(0.5, (maxTriangles / total) * 0.95);
  for (const [i, p] of parts.entries()) {
    p.doc = cloneDocument(welded[i]).setLogger(new Logger(Logger.Verbosity.WARN));
    await p.doc.transform(simplify({ simplifier: MeshoptSimplifier, ratio, error: 0.01 }));
  }
  total = countAll();
}

fs.mkdirSync(args.out, { recursive: true });
const manifest = {
  source: '4-Axis Arm Assembly by samoubala, CC-BY-4.0, https://sketchfab.com/3d-models/4-axis-arm-assembly-c0a748faf37644288c1271e9a6f51057',
  units: 'meters',
  frame: 'y up; arm reaches toward +z; base center on the origin',
  note: 'Rest pose is the pose of the original model. Angles are rotations about "axis" (right-hand rule) relative to that pose; calibrate offsets against servo angles.',
  triangles: { original, final: total },
  parts: [],
};
const pivotOf = Object.fromEntries(parts.map((p) => [p.part.name, p.pivot]));
for (const { part, pivot, doc } of parts) {
  await doc.transform(prune());
  const file = `${part.name}.glb`;
  await io.write(path.join(args.out, file), doc);
  manifest.parts.push({
    name: part.name,
    file,
    parent: part.parent,
    relayField: part.relayField,
    axis: part.axis ? axes[part.axis] : null,
    pivot: round(pivot),
    offsetFromParent: round(part.parent ? sub(pivot, pivotOf[part.parent]) : pivot),
    triangles: triangleCount(doc),
  });
}
fs.writeFileSync(path.join(args.out, 'manifest.json'), JSON.stringify(manifest, null, 2) + '\n');

console.log(`triangles: ${original} -> ${total} (vertex ratio ${ratio.toFixed(3)})`);
console.table(manifest.parts.map(({ name, parent, axis, offsetFromParent, triangles }) => (
  { name, parent, axis: JSON.stringify(axis), offset: JSON.stringify(offsetFromParent), triangles })));
