#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "==> Ensuring shared docker network exists..."
docker network create molecule-monorepo-net 2>/dev/null || true

echo "==> Starting infrastructure..."
docker compose -f "$ROOT_DIR/docker-compose.infra.yml" up -d

echo "==> Waiting for Postgres..."
until docker compose -f "$ROOT_DIR/docker-compose.infra.yml" exec -T postgres pg_isready -U "${POSTGRES_USER:-dev}" 2>/dev/null; do
  sleep 1
done
echo "    Postgres is ready."

echo "==> Waiting for Redis..."
until docker compose -f "$ROOT_DIR/docker-compose.infra.yml" exec -T redis redis-cli ping 2>/dev/null | grep -q PONG; do
  sleep 1
done
echo "    Redis is ready."

echo "==> Verifying Redis KEA config..."
KEA=$(docker compose -f "$ROOT_DIR/docker-compose.infra.yml" exec -T redis redis-cli config get notify-keyspace-events | tail -1)
echo "    notify-keyspace-events = $KEA"

echo "==> Running migrations..."
MIGRATIONS_DIR="$ROOT_DIR/platform/migrations"
if [ -d "$MIGRATIONS_DIR" ]; then
  for f in "$MIGRATIONS_DIR"/*.sql; do
    echo "    Applying $(basename "$f")..."
    docker compose -f "$ROOT_DIR/docker-compose.infra.yml" exec -T postgres \
      psql -U "${POSTGRES_USER:-dev}" -d "${POSTGRES_DB:-molecule}" -f - < "$f"
  done
  echo "    Migrations complete."
else
  echo "    No migrations directory found, skipping."
fi

echo "==> Infrastructure ready!"
echo "    Postgres: localhost:5432"
echo "    Redis:    localhost:6379"
echo "    Langfuse: localhost:3001"
