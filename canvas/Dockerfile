FROM node:20-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm install
COPY . .
ARG NEXT_PUBLIC_PLATFORM_URL=http://localhost:8080
ARG NEXT_PUBLIC_WS_URL=ws://localhost:8080/ws
ARG NEXT_PUBLIC_ADMIN_TOKEN=
ENV NEXT_PUBLIC_PLATFORM_URL=$NEXT_PUBLIC_PLATFORM_URL
ENV NEXT_PUBLIC_WS_URL=$NEXT_PUBLIC_WS_URL
ENV NEXT_PUBLIC_ADMIN_TOKEN=$NEXT_PUBLIC_ADMIN_TOKEN
RUN npm run build

FROM node:20-alpine
WORKDIR /app
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static
COPY --from=builder /app/public ./public

EXPOSE 3000
ENV PORT=3000
ENV HOSTNAME="0.0.0.0"
# Non-root runtime — node image defaults to root, explicitly drop.
# node:20-alpine ships with a `node` user at uid/gid 1000; remove it before
# claiming 1000 for `canvas` so `addgroup -g 1000` doesn't collide.
RUN deluser --remove-home node 2>/dev/null || true; \
    delgroup node 2>/dev/null || true; \
    addgroup -g 1000 canvas && adduser -u 1000 -G canvas -s /bin/sh -D canvas
USER canvas
CMD ["node", "server.js"]
