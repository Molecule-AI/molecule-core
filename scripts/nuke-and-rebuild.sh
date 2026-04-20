#!/bin/bash
# Full nuke + rebuild — one command to reset everything
# Usage: bash scripts/nuke-and-rebuild.sh
set -euo pipefail

echo "=== NUKE ==="
docker compose down -v 2>/dev/null || true
docker ps -a --format "{{.Names}}" | grep "^ws-" | xargs -r docker rm -f 2>/dev/null || true
docker volume ls --format "{{.Name}}" | grep "^ws-" | xargs -r docker volume rm 2>/dev/null || true
docker network rm molecule-monorepo-net 2>/dev/null || true
echo "  cleaned"

echo "=== REBUILD ==="
docker compose up -d --build
echo "  platform + canvas up"

echo "=== POST-REBUILD SETUP ==="
bash scripts/post-rebuild-setup.sh
