#!/usr/bin/env bash
# Common E2E helpers. Source this from every tests/e2e/*.sh.
#
# Usage:
#   source "$(dirname "$0")/_lib.sh"
#   e2e_cleanup_all_workspaces   # call at top of script
#   TOKEN=$(echo "$register_response" | e2e_extract_token)
#
# BASE defaults to http://localhost:8080. Set it before sourcing to override.
# ADMIN_BASE defaults to http://localhost:8081 (fix/issue-684-admin-network-isolation).
# Admin routes (GET/POST/DELETE /workspaces, /events, /bundles/*, /settings/secrets,
# /admin/*, /approvals/pending, /channels/discover, etc.) are served ONLY on ADMIN_BASE.
# In docker-compose, ADMIN_PORT is not published to the host — test runner can use
# localhost:8081 in CI because the platform runs directly on the host there.

: "${BASE:=http://localhost:8080}"
export BASE
: "${ADMIN_BASE:=http://localhost:8081}"
export ADMIN_BASE

# Emit the auth_token from a /registry/register response on stdout.
# See _extract_token.py for the exact semantics.
e2e_extract_token() {
  python3 "$(dirname "${BASH_SOURCE[0]}")/_extract_token.py"
}

# Delete every workspace currently on the platform. Use at the top of a
# script so count-based assertions are reproducible across runs.
# Mint a fresh workspace auth token via the admin endpoint (issue #6).
# Use this INSTEAD of racing /registry/register from the test harness —
# GET /admin/workspaces/:id/test-token is deterministic and gated by
# MOLECULE_ENV (off in production, on in dev / CI).
#
# Usage:
#   TOKEN=$(e2e_mint_test_token "$workspace_id") || exit 1
e2e_mint_test_token() {
  local wid="$1"
  if [ -z "$wid" ]; then
    echo "e2e_mint_test_token: workspace id required" >&2
    return 2
  fi
  # fix/issue-684: test-token endpoint moved to ADMIN_BASE (admin port, not published to host)
  local body
  body=$(curl -s -w "\n%{http_code}" "$ADMIN_BASE/admin/workspaces/$wid/test-token")
  local code
  code=$(printf '%s' "$body" | tail -n1)
  local json
  json=$(printf '%s' "$body" | sed '$d')
  if [ "$code" != "200" ]; then
    echo "e2e_mint_test_token: got HTTP $code (is MOLECULE_ENV!=production? is ADMIN_BASE=$ADMIN_BASE reachable?)" >&2
    return 1
  fi
  printf '%s' "$json" | python3 -c "import json,sys; print(json.load(sys.stdin)['auth_token'])"
}

e2e_cleanup_all_workspaces() {
  # fix/issue-684: GET/DELETE /workspaces moved to admin router (ADMIN_BASE).
  for _wid in $(curl -s "$ADMIN_BASE/workspaces" | python3 -c "import json,sys
try:
  [print(w['id']) for w in json.load(sys.stdin)]
except Exception:
  pass" 2>/dev/null); do
    curl -s -X DELETE "$ADMIN_BASE/workspaces/$_wid?confirm=true" > /dev/null || true
  done
}
