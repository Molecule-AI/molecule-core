#!/bin/sh
# dev-start.sh — one-command local development environment.
#
# Starts: Postgres, Redis, Platform (Go :8080), Canvas (Next.js :3000)
# Stops all on Ctrl-C.
#
# Prerequisites:
#   - Docker (for Postgres + Redis)
#   - Go 1.25+ (for platform)
#   - Node.js 20+ (for canvas)
#
# Usage:
#   ./scripts/dev-start.sh
#   # Open http://localhost:3000

set -e

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

cleanup() {
    echo ""
    echo "Shutting down..."
    kill $PLATFORM_PID $CANVAS_PID 2>/dev/null || true
    docker compose -f "$ROOT/docker-compose.infra.yml" down 2>/dev/null || true
    echo "Done."
}
trap cleanup EXIT INT TERM

echo "==> Starting infrastructure (Postgres, Redis)..."
docker compose -f "$ROOT/docker-compose.infra.yml" up -d

echo "==> Waiting for Postgres..."
until docker compose -f "$ROOT/docker-compose.infra.yml" exec -T postgres pg_isready -q 2>/dev/null; do
    sleep 1
done
echo "    Postgres ready."

echo "==> Starting Platform (Go :8080)..."
cd "$ROOT/workspace-server"
go run ./cmd/server &
PLATFORM_PID=$!

echo "==> Waiting for Platform health..."
until curl -sf http://localhost:8080/health >/dev/null 2>&1; do
    sleep 1
done
echo "    Platform ready."

echo "==> Starting Canvas (Next.js :3000)..."
cd "$ROOT/canvas"
if [ ! -d node_modules ]; then
    npm install
fi
npm run dev &
CANVAS_PID=$!

echo ""
echo "============================================"
echo "  Molecule AI dev environment running"
echo ""
echo "  Canvas:   http://localhost:3000"
echo "  Platform: http://localhost:8080"
echo "  Postgres: localhost:5432"
echo "  Redis:    localhost:6379"
echo ""
echo "  Press Ctrl-C to stop all services"
echo "============================================"

wait
