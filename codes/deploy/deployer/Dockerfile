# One-shot job that builds the robot scene and publishes it, together with
# the campusFGA snapshot, to the Catalyst. Build contexts (see compose.yaml):
#   scene    -> codes/scene
#   campusfga -> codes/deploy/scenes/campusFGA
# Node 22+ is required by @dcl/sdk-commands (fs.globSync).
FROM node:22.12.0-bookworm-slim

WORKDIR /work/robot-scene
COPY --from=scene package.json package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY --from=scene scene.json tsconfig.json .dclignore ./
COPY --from=scene src ./src
COPY --from=scene assets ./assets

COPY --from=campusfga . /work/campusFGA
COPY deploy.js /work/deploy.js

CMD ["node", "/work/deploy.js"]
