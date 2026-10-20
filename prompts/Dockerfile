FROM node:22-alpine@sha256:0a7108bf6c7bf5de370ffb1a3ed6be93d405b43ff159f681a8d18c0e2bc2e402
LABEL org.opencontainers.image.title="InkRoom Prompts" \
      org.opencontainers.image.version="1.0.3" \
      org.opencontainers.image.source="https://github.com/InkRoom-AI/inkroom-client" \
      org.opencontainers.image.licenses="MIT" \
      io.modelcontextprotocol.server.name="io.github.InkRoomAI/inkroom-prompts"
ENV NODE_ENV=production
WORKDIR /opt/inkroom-prompts
COPY package.json package-lock.json ./
RUN npm ci --omit=dev --ignore-scripts --no-audit --no-fund
COPY src ./src
COPY skills/inkroom-prompts/scripts ./skills/inkroom-prompts/scripts
COPY LICENSE ./LICENSE
USER node
ENTRYPOINT ["node", "src/cli.mjs"]
