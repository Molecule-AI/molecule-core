#!/bin/bash
# Post-rebuild setup — run after docker compose up -d --build
# Inserts global secrets that the provisioner injects into every workspace container.
# Without these, agents can't call MiniMax or push to GitHub.

set -euo pipefail

DB_CONTAINER="${DB_CONTAINER:-molecule-monorepo-postgres-1}"
DB_USER="${DB_USER:-dev}"
DB_NAME="${DB_NAME:-molecule}"
PLATFORM_URL="${PLATFORM_URL:-http://127.0.0.1:8080}"
ADMIN_TOKEN="${ADMIN_TOKEN:-HlgeMb8LjQLXg/B4y8hYzhbCQlg5LNu0oEa4IjShARE=}"

echo "=== Waiting for platform health ==="
until curl -s --max-time 5 "$PLATFORM_URL/health" >/dev/null 2>&1; do
    echo "  waiting..."
    sleep 3
done
echo "  platform up"

echo "=== Inserting global secrets ==="
docker exec "$DB_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -c "
INSERT INTO global_secrets (key, encrypted_value, encryption_version) VALUES
('ANTHROPIC_BASE_URL', 'https://api.minimax.io/anthropic', 0),
('ANTHROPIC_AUTH_TOKEN', '${MINIMAX_API_KEY:-sk-cp-lHt-QFSyZwZxeo_fMbmLUX3VgHOwbKGMXUZb6PS2U15D3fqjDB2qPh1OVEzvfvWs9CgcrUpyU7C682uVT_8GBy9RFLaFzBcdLkKdVcPX4yj9UaXNTH82KVw}', 0),
('ANTHROPIC_MODEL', 'MiniMax-M2.7', 0),
('ANTHROPIC_SMALL_FAST_MODEL', 'MiniMax-M2.7', 0),
('API_TIMEOUT_MS', '3000000', 0),
('CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC', '1', 0),
('GITHUB_TOKEN', '${GITHUB_PAT:-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}', 0)
ON CONFLICT (key) DO UPDATE SET encrypted_value = EXCLUDED.encrypted_value;
"
echo "  7 global secrets set"

echo "=== Importing org template ==="
curl -s --max-time 600 -X POST "$PLATFORM_URL/org/import" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"dir":"molecule-dev"}' | head -1
echo ""
echo "  import complete"

echo "=== Done ==="
echo "Run: http://127.0.0.1:3000 for canvas"
