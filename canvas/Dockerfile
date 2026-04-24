FROM node:22-alpine AS builder
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

FROM node:22-alpine
WORKDIR /app
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static
COPY --from=builder /app/public ./public

EXPOSE 3000
ENV PORT=3000
ENV HOSTNAME="0.0.0.0"
# Non-root runtime — use addgroup/adduser without fixed GID/UID to avoid conflicts with base image
RUN addgroup canvas 2>/dev/null || true && adduser -G canvas -s /bin/sh -D canvas 2>/dev/null || true
USER canvas
CMD ["node", "server.js"]
