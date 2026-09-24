# Three stages: install deps once, build with them, then ship only the
# standalone server output — the runtime image never contains node_modules
# or source, just what `next build` traced as actually needed.

FROM node:24-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci

FROM node:24-alpine AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
ENV NEXT_TELEMETRY_DISABLED=1

# NEXT_PUBLIC_* is inlined into the client bundle at build time, not read from
# the environment at runtime — so a key supplied only to the running container
# arrives too late and is simply absent. Without it usePush has nothing to
# subscribe with, finds no key, and quietly never subscribes: notifications
# are then switched on, permitted by the browser, and still never arrive.
#
# Empty is allowed so a build without push still works; it is the deployment
# that has to remember, which is why docker-compose passes it through.
ARG NEXT_PUBLIC_VAPID_PUBLIC_KEY=""
ENV NEXT_PUBLIC_VAPID_PUBLIC_KEY=$NEXT_PUBLIC_VAPID_PUBLIC_KEY
RUN npm run build

FROM node:24-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production PORT=3000 HOSTNAME=0.0.0.0
# node:alpine already ships a low-privilege "node" user — no need to add one.
COPY --from=builder /app/public ./public
COPY --from=builder --chown=node:node /app/.next/standalone ./
COPY --from=builder --chown=node:node /app/.next/static ./.next/static
USER node
EXPOSE 3000
CMD ["node", "server.js"]
