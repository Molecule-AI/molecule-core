#!/usr/bin/env bash
# Common E2E helpers. Source this from every tests/e2e/*.sh.
#
# Usage:
#   source "$(dirname "$0")/_lib.sh"
#   e2e_cleanup_all_workspaces   # call at top of script
#   TOKEN=$(echo "$register_response" | e2e_extract_token)
#
# BASE defaults to http://localhost:8080. Set it before sourcing to override.

: "${BASE:=http://localhost:8080}"
export BASE

# Emit the auth_token from a /registry/register response on stdout.
# See _extract_token.py for the exact semantics.
e2e_extract_token() {
  python3 "$(dirname "${BASH_SOURCE[0]}")/_extract_token.py"
}

# Delete every workspace currently on the platform. Use at the top of a
# script so count-based assertions are reproducible across runs.
e2e_cleanup_all_workspaces() {
  for _wid in $(curl -s "$BASE/workspaces" | python3 -c "import json,sys
try:
  [print(w['id']) for w in json.load(sys.stdin)]
except Exception:
  pass" 2>/dev/null); do
    curl -s -X DELETE "$BASE/workspaces/$_wid?confirm=true" > /dev/null || true
  done
}
