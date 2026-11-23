# One-shot job that builds codes/scene (campusFGA with the robotic arm twin)
# and publishes it to the Catalyst. Build context "scene" (see compose.yaml)
# -> codes/scene.
# Node 22+ is required by @dcl/sdk-commands (fs.globSync).
FROM node:22.12.0-bookworm-slim

WORKDIR /work/robot-scene
COPY --from=scene package.json package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY --from=scene scene.json tsconfig.json .dclignore ./
COPY --from=scene src ./src
COPY --from=scene assets ./assets

COPY deploy.js /work/deploy.js

CMD ["node", "/work/deploy.js"]
