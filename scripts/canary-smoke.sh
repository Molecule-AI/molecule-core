#!/bin/bash
# canary-smoke.sh — runs the post-deploy smoke suite against the
# staging canary tenant fleet. Called by the canary-verify.yml GitHub
# Actions workflow after a new workspace-server image gets pushed to
# GHCR; exits non-zero on any failure so the workflow can skip the
# :staging-sha → :latest retag that would otherwise release broken
# code to the prod tenant fleet.
#
# Environment:
#   CANARY_TENANT_URLS       space-sep list of canary tenant base URLs
#                            (e.g. "https://canary-pm.staging.moleculesai.app
#                                   https://canary-mcp.staging.moleculesai.app")
#   CANARY_ADMIN_TOKENS      space-sep list of ADMIN_TOKENs, positionally
#                            matched to CANARY_TENANT_URLS. Canary tenants
#                            are provisioned with known ADMIN_TOKENs so CI
#                            can hit their admin-gated endpoints.
#   CANARY_CP_BASE_URL       CP base URL the canaries call back to
#                            (https://staging-api.moleculesai.app)
#   CANARY_CP_SHARED_SECRET  matches CP's PROVISION_SHARED_SECRET so this
#                            script can also exercise /cp/workspaces/* via
#                            the canary's own CPProvisioner identity.
#
# Exit codes: 0 = all green, 1 = assertion failure, 2 = setup/env problem.

set -euo pipefail

# ── Setup ────────────────────────────────────────────────────────────────

: "${CANARY_TENANT_URLS:?space-sep list of canary base URLs required}"
: "${CANARY_ADMIN_TOKENS:?space-sep list of ADMIN_TOKENs required, same order as URLs}"
: "${CANARY_CP_BASE_URL:?CP base URL required}"

read -r -a URLS <<< "$CANARY_TENANT_URLS"
read -r -a TOKENS <<< "$CANARY_ADMIN_TOKENS"

if [ "${#URLS[@]}" -ne "${#TOKENS[@]}" ]; then
  echo "ERROR: URLS(${#URLS[@]}) and TOKENS(${#TOKENS[@]}) length mismatch" >&2
  exit 2
fi
if [ "${#URLS[@]}" -eq 0 ]; then
  echo "ERROR: no canary URLs configured" >&2
  exit 2
fi

PASS=0
FAIL=0

# ── Helpers ──────────────────────────────────────────────────────────────

check() {
  local desc="$1" expected="$2" actual="$3"
  if echo "$actual" | grep -qF "$expected"; then
    printf "  PASS %s\n" "$desc"
    PASS=$((PASS + 1))
  else
    printf "  FAIL %s\n    expected to contain: %s\n    got: %s\n" "$desc" "$expected" "$actual" >&2
    FAIL=$((FAIL + 1))
  fi
}

# acurl does an admin-authenticated GET/POST/etc. against a canary tenant.
# Takes +BASE_URL +ADMIN_TOKEN as its first two positional args; the rest
# are passed through to curl. Keeps the two values paired so the wrong
# tenant never gets the wrong token.
acurl() {
  local base="$1" token="$2"; shift 2
  curl -sS --max-time 20 -H "Authorization: Bearer $token" "$@" -- "$base${CANARY_ACURL_PATH:-}"
}

# ── Checks (run per canary tenant) ───────────────────────────────────────

for i in "${!URLS[@]}"; do
  base="${URLS[$i]}"
  token="${TOKENS[$i]}"
  printf "\n── %s ──\n" "$base"

  # 1. Liveness — the tenant is up and responding to admin auth.
  CANARY_ACURL_PATH="/admin/liveness" resp=$(acurl "$base" "$token" || true)
  check "liveness returns a subsystems map" '"subsystems"' "$resp"

  # 2. CP env refresh — the workspace-server fetched MOLECULE_CP_SHARED_SECRET
  # from CP on startup. We can't read env directly, but we can assert the
  # liveness + workspace list both work, which together imply the binary
  # booted without crashing on the refresh call. A startup failure in
  # refreshEnvFromCP logs but still boots (best-effort semantics), so
  # this is a sanity check, not a proof.
  CANARY_ACURL_PATH="/workspaces" resp=$(acurl "$base" "$token" || true)
  check "workspace list is JSON array" "[" "$resp"

  # 3. Memory commit round-trip — scope=LOCAL so test data stays on this
  # tenant. Verifies encryption + scrubber + retrieval end-to-end.
  probe_id="canary-smoke-$(date +%s)-$i"
  body=$(printf '{"scope":"LOCAL","namespace":"canary-smoke","content":"probe-%s"}' "$probe_id")
  CANARY_ACURL_PATH="/memories/commit" resp=$(curl -sS --max-time 20 \
    -X POST -H "Content-Type: application/json" -H "Authorization: Bearer $token" \
    --data "$body" "$base/memories/commit" || true)
  check "memory commit accepted" '"id"' "$resp"

  CANARY_ACURL_PATH="/memories/search?query=probe-${probe_id}" \
    resp=$(curl -sS --max-time 20 -H "Authorization: Bearer $token" \
    "$base/memories/search?query=probe-${probe_id}" || true)
  check "memory search finds the probe" "probe-${probe_id}" "$resp"

  # 4. Events admin read — AdminAuth path (C4 fail-closed proof on SaaS).
  CANARY_ACURL_PATH="/events" resp=$(acurl "$base" "$token" || true)
  check "events endpoint returns JSON" "[" "$resp"

  # 5. Negative: unauth'd admin call must 401 (C4 regression gate).
  unauth_code=$(curl -sS -o /dev/null -w '%{http_code}' --max-time 10 "$base/admin/liveness" || echo "000")
  check "unauth'd /admin/liveness returns 401" "401" "$unauth_code"
done

# ── Summary ──────────────────────────────────────────────────────────────

printf "\n=== CANARY SMOKE RESULTS ===\n"
printf "  PASS: %d\n  FAIL: %d\n" "$PASS" "$FAIL"

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
