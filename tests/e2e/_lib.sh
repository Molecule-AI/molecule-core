#!/usr/bin/env bash
# Common E2E helpers. Source this from every tests/e2e/*.sh.
#
# Usage:
#   source "$(dirname "$0")/_lib.sh"
#   e2e_base="http://localhost:8080"
#   e2e_cleanup_all_workspaces   # call at top of script
#   token=$(e2e_register "$ID" "$URL" "$CARD_JSON")
#   # then use -H "Authorization: Bearer $token" on heartbeat/update-card

# Emit the auth_token from a /registry/register response. Prints empty
# string (not an error) when no token was issued so callers can still
# exercise the grandfather path.
e2e_extract_token() {
  python3 -c "import sys,json; print(json.load(sys.stdin).get('auth_token',''))" 2>/dev/null || true
}

# Register a workspace and echo the bearer token on stdout.
# Args: $1 workspace_id  $2 url  $3 agent_card JSON
e2e_register() {
  curl -s -X POST "$e2e_base/registry/register" \
    -H "Content-Type: application/json" \
    -d "{\"id\":\"$1\",\"url\":\"$2\",\"agent_card\":$3}" \
    | e2e_extract_token
}

# Heartbeat with bearer auth.
# Args: $1 workspace_id  $2 token  $3 payload_json (without the id)
e2e_heartbeat() {
  curl -s -X POST "$e2e_base/registry/heartbeat" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $2" \
    -d "$3"
}

# Delete every workspace currently on the platform. Use at the top of a
# script so count-based assertions are reproducible across runs.
e2e_cleanup_all_workspaces() {
  for _wid in $(curl -s "$e2e_base/workspaces" | python3 -c "import json,sys
try:
  [print(w['id']) for w in json.load(sys.stdin)]
except Exception:
  pass" 2>/dev/null); do
    curl -s -X DELETE "$e2e_base/workspaces/$_wid?confirm=true" > /dev/null || true
  done
}
