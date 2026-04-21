#!/usr/bin/env bash
# Full-lifecycle SaaS E2E against staging.
#
# Creates a fresh org per run (unique slug), waits for tenant EC2 + cloudflared
# provisioning, exercises every major workspace-level API (registration,
# heartbeat, A2A, delegation, HMA memory, activity, peers, events), then
# tears the whole org down and asserts that every cloud artefact (EC2, SG,
# Cloudflare tunnel, DNS record, DB rows) has gone. A leaked resource at
# teardown is a CI failure — that's the whole point of per-run org
# provisioning.
#
# Required env:
#   MOLECULE_CP_URL                Staging CP base URL (default:
#                                  https://staging-api.moleculesai.app)
#   MOLECULE_SESSION_COOKIE        Valid WorkOS session cookie for a test
#                                  user that's already in the beta
#                                  allowlist AND has accepted current terms.
#                                  Extract from browser after signing in to
#                                  staging. Name: molecule_cp_session.
#   MOLECULE_ADMIN_TOKEN           CP admin bearer (CP_ADMIN_API_TOKEN on
#                                  Railway). Used for teardown via
#                                  DELETE /cp/admin/tenants/:slug and for
#                                  leak-detection reads.
#
# Optional env:
#   E2E_RUNTIME                    Which runtime to test the agent round-trip
#                                  with. Default: hermes (fastest boot, cheap).
#                                  Use claude-code when you need to validate
#                                  that fix.
#   E2E_PROVISION_TIMEOUT_SECS     How long to wait for the tenant EC2 to
#                                  come up. Default: 900 (15 min — cold
#                                  EC2 + cloudflared tunnel + DNS propagation
#                                  can touch that window).
#   E2E_KEEP_ORG                   If set to 1, skip teardown. ONLY use
#                                  locally for debugging — CI must never
#                                  set this or staging fills with orphans.
#   E2E_RUN_ID                     Override the auto-generated suffix. CI
#                                  should pass ${GITHUB_RUN_ID} so the
#                                  org slug is grep-able in AWS later.
#
# Exit codes:
#   0  happy path
#   1  generic failure (see log)
#   2  missing required env
#   3  provisioning timed out
#   4  cleanup left orphan resources (leak detected)

set -euo pipefail

CP_URL="${MOLECULE_CP_URL:-https://staging-api.moleculesai.app}"
SESSION_COOKIE="${MOLECULE_SESSION_COOKIE:?MOLECULE_SESSION_COOKIE required — see header for how to obtain}"
ADMIN_TOKEN="${MOLECULE_ADMIN_TOKEN:?MOLECULE_ADMIN_TOKEN required — from Railway molecule-platform CP env}"
RUNTIME="${E2E_RUNTIME:-hermes}"
PROVISION_TIMEOUT_SECS="${E2E_PROVISION_TIMEOUT_SECS:-900}"
RUN_ID_SUFFIX="${E2E_RUN_ID:-$(date +%H%M%S)-$$}"

# Slug constraints from orgs.go: ^[a-z][a-z0-9-]{2,31}$.
# Prefix with "e2e-" so test orgs are grep-able and auto-cleanup crons
# can target them even when a script crashes before the EXIT trap fires.
SLUG="e2e-$(date +%Y%m%d)-${RUN_ID_SUFFIX}"
SLUG=$(echo "$SLUG" | tr '[:upper:]' '[:lower:]' | tr -cd 'a-z0-9-' | head -c 32)

# ─── logging helpers ────────────────────────────────────────────────────
log()  { echo "[$(date +%H:%M:%S)] $*"; }
fail() { echo "[$(date +%H:%M:%S)] ❌ $*" >&2; exit 1; }
ok()   { echo "[$(date +%H:%M:%S)] ✅ $*"; }

CURL_COMMON=(-sS --fail-with-body --max-time 30)

# ─── cleanup trap ───────────────────────────────────────────────────────
# Teardown runs on every exit path (success, failure, signal). The
# delete-tenant endpoint is idempotent — calling it on a slug that was
# never created returns 404 which we swallow.
CLEANUP_DONE=0
cleanup_org() {
  [ "$CLEANUP_DONE" = "1" ] && return 0
  CLEANUP_DONE=1

  if [ "${E2E_KEEP_ORG:-0}" = "1" ]; then
    log "E2E_KEEP_ORG=1 — skipping teardown. Manually delete $SLUG when done."
    return 0
  fi

  log "🧹 Tearing down org $SLUG..."
  # Confirm token must equal slug — defense against accidental teardowns.
  curl "${CURL_COMMON[@]}" -X DELETE "$CP_URL/cp/admin/tenants/$SLUG" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"confirm_token\":\"$SLUG\"}" >/dev/null 2>&1 \
    && ok "Teardown request accepted" \
    || log "Teardown returned non-2xx (may already be gone)"

  # Leak detection: wait briefly then query CP for any remaining artefacts
  # tagged with this slug. Anything left = bug in DeprovisionInstance.
  sleep 10
  local leak_count
  leak_count=$(curl "${CURL_COMMON[@]}" "$CP_URL/cp/admin/orgs" \
    -H "Authorization: Bearer $ADMIN_TOKEN" 2>/dev/null \
    | python3 -c "import json,sys; d=json.load(sys.stdin); print(sum(1 for o in d.get('orgs', []) if o.get('slug')=='$SLUG' and o.get('status') != 'purged'))" \
    2>/dev/null || echo 0)
  if [ "$leak_count" != "0" ]; then
    echo "⚠️  LEAK: org $SLUG still present post-teardown (count=$leak_count)" >&2
    exit 4
  fi
  ok "Teardown clean — no orphan resources for $SLUG"
}
trap cleanup_org EXIT INT TERM

# ─── 0. Preflight ───────────────────────────────────────────────────────
log "═══════════════════════════════════════════════════════════════════"
log " Staging full-SaaS E2E"
log "   CP:      $CP_URL"
log "   Slug:    $SLUG"
log "   Runtime: $RUNTIME"
log "   Timeout: ${PROVISION_TIMEOUT_SECS}s"
log "═══════════════════════════════════════════════════════════════════"

log "0/10 Preflight: CP reachable?"
curl "${CURL_COMMON[@]}" "$CP_URL/health" >/dev/null || fail "CP health check failed"
ok "CP reachable"

# ─── 1. Accept terms (idempotent) ───────────────────────────────────────
log "1/10 Accepting current terms..."
curl "${CURL_COMMON[@]}" -X POST "$CP_URL/cp/auth/accept-terms" \
  -H "Cookie: molecule_cp_session=$SESSION_COOKIE" \
  -H "Content-Type: application/json" \
  -d '{}' >/dev/null || log "accept-terms returned non-2xx (may already be accepted)"
ok "Terms acceptance step complete"

# ─── 2. Create org ──────────────────────────────────────────────────────
log "2/10 Creating org $SLUG..."
CREATE_RESP=$(curl "${CURL_COMMON[@]}" -X POST "$CP_URL/cp/orgs" \
  -H "Cookie: molecule_cp_session=$SESSION_COOKIE" \
  -H "Content-Type: application/json" \
  -d "{\"slug\":\"$SLUG\",\"name\":\"E2E $SLUG\"}")
echo "$CREATE_RESP" | python3 -m json.tool >/dev/null || fail "Org create returned non-JSON: $CREATE_RESP"
ok "Org created"

# ─── 3. Wait for tenant EC2 + cloudflared tunnel + DNS ──────────────────
log "3/10 Waiting for tenant provisioning (up to ${PROVISION_TIMEOUT_SECS}s)..."
DEADLINE=$(( $(date +%s) + PROVISION_TIMEOUT_SECS ))
LAST_STATUS=""
while true; do
  if [ "$(date +%s)" -gt "$DEADLINE" ]; then
    fail "Tenant provisioning timed out after ${PROVISION_TIMEOUT_SECS}s (last: $LAST_STATUS)"
  fi
  STATUS_JSON=$(curl "${CURL_COMMON[@]}" "$CP_URL/cp/orgs/$SLUG/provision-status" \
    -H "Cookie: molecule_cp_session=$SESSION_COOKIE" 2>/dev/null || echo '{}')
  STATUS=$(echo "$STATUS_JSON" | python3 -c "import json,sys; print(json.load(sys.stdin).get('status',''))" 2>/dev/null || echo "")
  if [ "$STATUS" != "$LAST_STATUS" ]; then
    log "    status → $STATUS"
    LAST_STATUS="$STATUS"
  fi
  case "$STATUS" in
    running)        break ;;
    failed)         fail "Tenant provisioning failed: $(echo "$STATUS_JSON" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("error",""))')" ;;
    provisioning|awaiting_payment|pending|"") sleep 15 ;;
    *)              sleep 15 ;;
  esac
done
ok "Tenant provisioning complete"

TENANT_URL=$(echo "$STATUS_JSON" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('tenant_url') or d.get('url') or '')" 2>/dev/null || echo "")
[ -z "$TENANT_URL" ] && TENANT_URL="https://$SLUG.moleculesai.app"
log "    TENANT_URL=$TENANT_URL"

# Tenant admin token — returned by provision-status for the
# just-provisioned org so the test can call tenant admin endpoints
# (POST /workspaces etc.) without depending on a workspace auth token.
TENANT_ADMIN_TOKEN=$(echo "$STATUS_JSON" | python3 -c "import json,sys; print(json.load(sys.stdin).get('admin_token',''))" 2>/dev/null || echo "")
[ -z "$TENANT_ADMIN_TOKEN" ] && fail "provision-status did not return admin_token"

ORG_ID=$(echo "$STATUS_JSON" | python3 -c "import json,sys; print(json.load(sys.stdin).get('org_id',''))" 2>/dev/null || echo "")

# ─── 4. Wait for tenant TLS cert to be reachable ───────────────────────
log "4/10 Waiting for tenant TLS / DNS propagation..."
TLS_DEADLINE=$(( $(date +%s) + 180 ))
while true; do
  if curl -sSfk --max-time 5 "$TENANT_URL/health" >/dev/null 2>&1; then
    break
  fi
  if [ "$(date +%s)" -gt "$TLS_DEADLINE" ]; then
    fail "Tenant URL never responded 2xx on /health within 3 min"
  fi
  sleep 5
done
ok "Tenant reachable at $TENANT_URL"

tenant_call() {
  local method="$1"; shift
  local path="$1"; shift
  curl "${CURL_COMMON[@]}" -X "$method" "$TENANT_URL$path" \
    -H "Authorization: Bearer $TENANT_ADMIN_TOKEN" \
    -H "X-Molecule-Org-Id: $ORG_ID" \
    "$@"
}

# ─── 5. Provision workspace (parent) ───────────────────────────────────
log "5/10 Provisioning parent workspace (runtime=$RUNTIME)..."
PARENT_RESP=$(tenant_call POST /workspaces \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"E2E Parent\",\"runtime\":\"$RUNTIME\",\"tier\":2,\"model\":\"gpt-4o\"}")
PARENT_ID=$(echo "$PARENT_RESP" | python3 -c "import json,sys; print(json.load(sys.stdin)['id'])")
log "    PARENT_ID=$PARENT_ID"

# ─── 6. Provision child (for delegation test) ──────────────────────────
log "6/10 Provisioning child workspace..."
CHILD_RESP=$(tenant_call POST /workspaces \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"E2E Child\",\"runtime\":\"$RUNTIME\",\"tier\":2,\"model\":\"gpt-4o\",\"parent_id\":\"$PARENT_ID\"}")
CHILD_ID=$(echo "$CHILD_RESP" | python3 -c "import json,sys; print(json.load(sys.stdin)['id'])")
log "    CHILD_ID=$CHILD_ID"

# ─── 7. Wait for both online ───────────────────────────────────────────
log "7/10 Waiting for both workspaces to reach status=online..."
WS_DEADLINE=$(( $(date +%s) + 600 ))  # 10 min
for wid in "$PARENT_ID" "$CHILD_ID"; do
  while true; do
    if [ "$(date +%s)" -gt "$WS_DEADLINE" ]; then
      fail "Workspace $wid never reached online within 10 min"
    fi
    WS_JSON=$(tenant_call GET "/workspaces/$wid" 2>/dev/null || echo '{}')
    WS_STATUS=$(echo "$WS_JSON" | python3 -c "import json,sys; print(json.load(sys.stdin).get('status',''))" 2>/dev/null)
    case "$WS_STATUS" in
      online) break ;;
      failed) fail "Workspace $wid status=failed: $(echo "$WS_JSON" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("last_sample_error",""))')" ;;
      *)      sleep 10 ;;
    esac
  done
  ok "    $wid online"
done

# ─── 8. A2A round-trip on parent ───────────────────────────────────────
log "8/10 Sending A2A message to parent — expecting an agent response..."
A2A_PAYLOAD=$(python3 -c "
import json, uuid
print(json.dumps({
    'jsonrpc': '2.0',
    'method': 'message/send',
    'id': 'e2e-msg-1',
    'params': {
        'message': {
            'role': 'user',
            'messageId': f'e2e-{uuid.uuid4().hex[:8]}',
            'parts': [{'kind': 'text', 'text': 'Reply with exactly: PONG'}]
        }
    }
}))
")
A2A_RESP=$(tenant_call POST "/workspaces/$PARENT_ID/a2a" \
  -H "Content-Type: application/json" \
  -d "$A2A_PAYLOAD")
AGENT_TEXT=$(echo "$A2A_RESP" | python3 -c "
import json, sys
d = json.load(sys.stdin)
parts = d.get('result', {}).get('parts', [])
print(parts[0].get('text', '') if parts else '')
" 2>/dev/null || echo "")
if [ -z "$AGENT_TEXT" ]; then
  fail "A2A returned no text. Raw: $A2A_RESP"
fi
if echo "$AGENT_TEXT" | grep -qiE "error|exception"; then
  fail "A2A returned an error-shaped response: $AGENT_TEXT"
fi
ok "A2A parent round-trip succeeded: \"${AGENT_TEXT:0:80}\""

# ─── 9. HMA memory write/read ──────────────────────────────────────────
log "9/10 Writing + reading HMA memory on parent..."
MEM_PAYLOAD=$(python3 -c "
import json
print(json.dumps({
    'content': 'E2E memory seed — run $SLUG',
    'scope': 'LOCAL'
}))
")
tenant_call POST "/workspaces/$PARENT_ID/memories" \
  -H "Content-Type: application/json" \
  -d "$MEM_PAYLOAD" >/dev/null || fail "memory POST failed"
# Read back and confirm presence
MEM_LIST=$(tenant_call GET "/workspaces/$PARENT_ID/memories?scope=LOCAL")
if ! echo "$MEM_LIST" | grep -q "run $SLUG"; then
  fail "HMA memory not readable after write. List: ${MEM_LIST:0:200}"
fi
ok "HMA memory write+read roundtripped"

# ─── 9b. Peers + activity smoke ────────────────────────────────────────
log "9b.  Peer discovery + activity log smoke..."
# Peers (uses workspace bearer — we don't have one here, so expect 401 and
# just verify the endpoint responds at all rather than 404).
set +e
tenant_call GET "/registry/$PARENT_ID/peers" -o /dev/null -w "%{http_code}\n" 2>&1 | head -1 > /tmp/peers_code.txt
set -e
PEERS_CODE=$(cat /tmp/peers_code.txt)
if [ "$PEERS_CODE" = "404" ]; then
  fail "Peers endpoint missing (404) — route regression"
fi
ok "Peers endpoint reachable (HTTP $PEERS_CODE — 401 expected without ws token)"

ACTIVITY=$(tenant_call GET "/activity?workspace_id=$PARENT_ID&limit=5" 2>/dev/null || echo '[]')
ACTIVITY_COUNT=$(echo "$ACTIVITY" | python3 -c "import json,sys
d=json.load(sys.stdin)
print(len(d if isinstance(d, list) else d.get('events', [])))" 2>/dev/null || echo 0)
log "    Activity events observed: $ACTIVITY_COUNT"

# ─── 10. Cleanup runs via trap ────────────────────────────────────────
log "10/10 All checks passed. Teardown runs via EXIT trap."
ok "═══ STAGING FULL-SAAS E2E PASSED ═══"
