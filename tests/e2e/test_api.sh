#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "$0")/_lib.sh"  # sets BASE and ADMIN_BASE defaults
PASS=0
FAIL=0

# Phase 30.1: tokens issued on first /registry/register must be echoed
# back on every subsequent /registry/heartbeat + /registry/update-card
# as `Authorization: Bearer <token>`. Capture them here.
ECHO_TOKEN=""
SUM_TOKEN=""

# AdminAuth-gated calls need a bearer token once any workspace token
# exists in the DB. ADMIN_TOKEN is populated after the first workspace
# create + test-token mint. acurl = "authenticated curl".
ADMIN_TOKEN=""
acurl() {
  if [ -n "$ADMIN_TOKEN" ]; then
    curl -s -H "Authorization: Bearer $ADMIN_TOKEN" "$@"
  else
    curl -s "$@"
  fi
}

# Pre-test cleanup: remove any workspaces left over from prior runs so
# count-based assertions ("empty", "count=2") are reproducible.
e2e_cleanup_all_workspaces

check() {
  local desc="$1"
  local expected="$2"
  local actual="$3"
  if echo "$actual" | grep -qF "$expected"; then
    echo "PASS: $desc"
    PASS=$((PASS + 1))
  else
    echo "FAIL: $desc"
    echo "  expected to contain: $expected"
    echo "  got: $actual"
    FAIL=$((FAIL + 1))
  fi
}

echo "=== API Integration Tests ==="
echo ""

# Test 1: Health
R=$(curl -s "$BASE/health")
check "GET /health" '"status":"ok"' "$R"

# Test 2: Empty list
# fix/issue-684: GET /workspaces is now on ADMIN_BASE (admin router, port 8081)
R=$(acurl "$ADMIN_BASE/workspaces")
check "GET /workspaces (empty)" '[]' "$R"

# Test 3: Create workspace A (AdminAuth fail-open — no tokens exist yet)
# fix/issue-684: POST /workspaces is now on ADMIN_BASE (admin router, port 8081)
R=$(curl -s -X POST "$ADMIN_BASE/workspaces" -H "Content-Type: application/json" -d '{"name":"Echo Agent","tier":1}')
check "POST /workspaces (create echo)" '"status":"provisioning"' "$R"
ECHO_ID=$(echo "$R" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")

# Mint a test token so all subsequent AdminAuth-gated calls succeed.
# fix/issue-684: test-token endpoint is now on ADMIN_BASE (admin router, port 8081).
# AdminAuth is still fail-open on fresh install so this works on first boot.
# Debug: show what the test-token endpoint returns
TEST_TOKEN_RAW=$(curl -s "$ADMIN_BASE/admin/workspaces/$ECHO_ID/test-token")
echo "  test-token response: $TEST_TOKEN_RAW"
ADMIN_TOKEN=$(echo "$TEST_TOKEN_RAW" | python3 -c "import sys,json; print(json.load(sys.stdin).get('auth_token',''))" 2>/dev/null || echo "")
if [ -n "$ADMIN_TOKEN" ]; then
  echo "  (acquired admin token: ${ADMIN_TOKEN:0:8}...)"
else
  echo "  WARNING: no admin token acquired — subsequent AdminAuth calls will fail"
fi

# Test 4: Create workspace B (needs bearer — tokens now exist in DB)
R=$(acurl -X POST "$ADMIN_BASE/workspaces" -H "Content-Type: application/json" -d '{"name":"Summarizer Agent","tier":1}')
check "POST /workspaces (create summarizer)" '"status":"provisioning"' "$R"
SUM_ID=$(echo "$R" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")

# Test 5: List has 2
R=$(acurl "$ADMIN_BASE/workspaces")
COUNT=$(echo "$R" | python3 -c "import sys,json; print(len(json.load(sys.stdin)))")
check "GET /workspaces (count=2)" "2" "$COUNT"

# Test 6: Get single
R=$(acurl "$BASE/workspaces/$ECHO_ID")
check "GET /workspaces/:id" '"name":"Echo Agent"' "$R"
check "GET /workspaces/:id (agent_card null)" '"agent_card":null' "$R"

# Test 7: Register echo — use workspace-specific token (from test-token
# endpoint), not the admin token. C18 requires a token issued TO THIS
# workspace, not just any valid token.
# fix/issue-684: test-token endpoint now on ADMIN_BASE (admin router, port 8081)
ECHO_WS_TOKEN=$(curl -s "$ADMIN_BASE/admin/workspaces/$ECHO_ID/test-token" | python3 -c "import sys,json; print(json.load(sys.stdin).get('auth_token',''))" 2>/dev/null || echo "")
R=$(curl -s -X POST "$BASE/registry/register" -H "Content-Type: application/json" \
  ${ECHO_WS_TOKEN:+-H "Authorization: Bearer $ECHO_WS_TOKEN"} \
  -d "{\"id\":\"$ECHO_ID\",\"url\":\"http://localhost:8001\",\"agent_card\":{\"name\":\"Echo Agent\",\"skills\":[{\"id\":\"echo\",\"name\":\"Echo\"}]}}")
check "POST /registry/register (echo)" '"status":"registered"' "$R"
# Extract token from register response; fall back to the test-token we
# already minted (register may not return a new token on re-registration).
ECHO_TOKEN=$(echo "$R" | e2e_extract_token)
if [ -z "$ECHO_TOKEN" ]; then ECHO_TOKEN="$ECHO_WS_TOKEN"; fi

# Test 8: Register summarizer — same pattern: workspace-specific token
# fix/issue-684: test-token endpoint now on ADMIN_BASE (admin router, port 8081)
SUM_WS_TOKEN=$(curl -s "$ADMIN_BASE/admin/workspaces/$SUM_ID/test-token" | python3 -c "import sys,json; print(json.load(sys.stdin).get('auth_token',''))" 2>/dev/null || echo "")
R=$(curl -s -X POST "$BASE/registry/register" -H "Content-Type: application/json" \
  ${SUM_WS_TOKEN:+-H "Authorization: Bearer $SUM_WS_TOKEN"} \
  -d "{\"id\":\"$SUM_ID\",\"url\":\"http://localhost:8002\",\"agent_card\":{\"name\":\"Summarizer\",\"skills\":[{\"id\":\"summarize\",\"name\":\"Summarize\"}]}}")
check "POST /registry/register (summarizer)" '"status":"registered"' "$R"
SUM_TOKEN=$(echo "$R" | e2e_extract_token)
if [ -z "$SUM_TOKEN" ]; then SUM_TOKEN="$SUM_WS_TOKEN"; fi

# Test 9: Both online
R=$(acurl "$BASE/workspaces/$ECHO_ID")
check "Echo is online" '"status":"online"' "$R"
check "Echo has agent_card" '"skills"' "$R"
check "Echo has url" '"url":"http://localhost:8001"' "$R"

# Test 10: Heartbeat
R=$(curl -s -X POST "$BASE/registry/heartbeat" -H "Content-Type: application/json" -H "Authorization: Bearer $ECHO_TOKEN" \
  -d "{\"workspace_id\":\"$ECHO_ID\",\"error_rate\":0.0,\"sample_error\":\"\",\"active_tasks\":2,\"uptime_seconds\":120}")
check "POST /registry/heartbeat" '"status":"ok"' "$R"

R=$(acurl "$BASE/workspaces/$ECHO_ID")
check "Heartbeat updated active_tasks" '"active_tasks":2' "$R"
check "Heartbeat updated uptime" '"uptime_seconds":120' "$R"

# Test 11: Discover without X-Workspace-ID — Phase 30.6 requires it
R=$(curl -s "$BASE/registry/discover/$ECHO_ID")
check "GET /registry/discover/:id (missing caller rejected)" 'X-Workspace-ID header is required' "$R"

# Test 12: Discover (from sibling — allowed)
R=$(curl -s "$BASE/registry/discover/$ECHO_ID" -H "X-Workspace-ID: $SUM_ID" -H "Authorization: Bearer $SUM_TOKEN")
check "GET /registry/discover/:id (sibling)" '"url"' "$R"

# Test 13: Peers (root siblings see each other)
R=$(curl -s "$BASE/registry/$ECHO_ID/peers" -H "Authorization: Bearer $ECHO_TOKEN")
check "GET /registry/:id/peers (has summarizer)" '"Summarizer' "$R"

R=$(curl -s "$BASE/registry/$SUM_ID/peers" -H "Authorization: Bearer $SUM_TOKEN")
check "GET /registry/:id/peers (has echo)" '"Echo Agent"' "$R"

# Test 14: Check access (root siblings)
R=$(curl -s -X POST "$BASE/registry/check-access" -H "Content-Type: application/json" \
  -d "{\"caller_id\":\"$ECHO_ID\",\"target_id\":\"$SUM_ID\"}")
check "POST /registry/check-access (siblings allowed)" '"allowed":true' "$R"

# Test 15: PATCH workspace (update position)
R=$(acurl -X PATCH "$BASE/workspaces/$ECHO_ID" -H "Content-Type: application/json" -d '{"x":100,"y":200}')
check "PATCH /workspaces/:id (position)" '"status":"updated"' "$R"

R=$(acurl "$BASE/workspaces/$ECHO_ID")
check "Position saved (x=100)" '"x":100' "$R"
check "Position saved (y=200)" '"y":200' "$R"

# Test 16: PATCH workspace (update name)
R=$(acurl -X PATCH "$BASE/workspaces/$ECHO_ID" -H "Content-Type: application/json" -d '{"name":"Echo Agent v2"}')
check "PATCH /workspaces/:id (name)" '"status":"updated"' "$R"

R=$(acurl "$BASE/workspaces/$ECHO_ID")
check "Name updated" '"name":"Echo Agent v2"' "$R"

# Test 17: Events (#165 / PR #167 — admin-gated, bearer required)
# fix/issue-684: /events now on ADMIN_BASE (admin router, port 8081)
R=$(acurl "$ADMIN_BASE/events" -H "Authorization: Bearer $ECHO_TOKEN")
check "GET /events (has events)" 'WORKSPACE_ONLINE' "$R"

R=$(acurl "$ADMIN_BASE/events/$ECHO_ID" -H "Authorization: Bearer $ECHO_TOKEN")
check "GET /events/:id (has events for echo)" 'WORKSPACE_ONLINE' "$R"

# Test 18: Update card
R=$(curl -s -X POST "$BASE/registry/update-card" -H "Content-Type: application/json" -H "Authorization: Bearer $ECHO_TOKEN" \
  -d "{\"workspace_id\":\"$ECHO_ID\",\"agent_card\":{\"name\":\"Echo Agent v2\",\"skills\":[{\"id\":\"echo\",\"name\":\"Echo\"},{\"id\":\"repeat\",\"name\":\"Repeat\"}]}}")
check "POST /registry/update-card" '"status":"updated"' "$R"

# Test 19: Degraded status transition
# First, ensure workspace is online (Redis TTL may have expired during test)
curl -s -X POST "$BASE/registry/heartbeat" -H "Content-Type: application/json" -H "Authorization: Bearer $ECHO_TOKEN" \
  -d "{\"workspace_id\":\"$ECHO_ID\",\"error_rate\":0.0,\"sample_error\":\"\",\"active_tasks\":0,\"uptime_seconds\":180}" > /dev/null

# Re-register to force online status in case liveness expired
curl -s -X POST "$BASE/registry/register" -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ECHO_TOKEN" \
  -d "{\"id\":\"$ECHO_ID\",\"url\":\"http://localhost:8001\",\"agent_card\":{\"name\":\"Echo Agent v2\",\"skills\":[{\"id\":\"echo\",\"name\":\"Echo\"},{\"id\":\"repeat\",\"name\":\"Repeat\"}]}}" > /dev/null

# Now send high error rate to trigger degraded
R=$(curl -s -X POST "$BASE/registry/heartbeat" -H "Content-Type: application/json" -H "Authorization: Bearer $ECHO_TOKEN" \
  -d "{\"workspace_id\":\"$ECHO_ID\",\"error_rate\":0.8,\"sample_error\":\"API rate limit\",\"active_tasks\":0,\"uptime_seconds\":200}")
check "Heartbeat (high error_rate)" '"status":"ok"' "$R"

R=$(acurl "$BASE/workspaces/$ECHO_ID")
check "Status degraded" '"status":"degraded"' "$R"

# Test 20: Recovery
R=$(curl -s -X POST "$BASE/registry/heartbeat" -H "Content-Type: application/json" -H "Authorization: Bearer $ECHO_TOKEN" \
  -d "{\"workspace_id\":\"$ECHO_ID\",\"error_rate\":0.0,\"sample_error\":\"\",\"active_tasks\":0,\"uptime_seconds\":300}")
check "Heartbeat (recovered)" '"status":"ok"' "$R"

R=$(acurl "$BASE/workspaces/$ECHO_ID")
check "Status back online" '"status":"online"' "$R"

# ---------- Activity Log Tests ----------
echo ""
echo "--- Activity Log Tests ---"

# Test: Report activity log
R=$(curl -s -X POST "$BASE/workspaces/$ECHO_ID/activity" -H "Content-Type: application/json" -H "Authorization: Bearer $ECHO_TOKEN" \
  -d '{"activity_type":"agent_log","method":"inference","summary":"Processing user query"}')
check "POST /workspaces/:id/activity (report)" '"status":"logged"' "$R"

# Test: Report A2A activity
R=$(curl -s -X POST "$BASE/workspaces/$ECHO_ID/activity" -H "Content-Type: application/json" -H "Authorization: Bearer $ECHO_TOKEN" \
  -d "{\"activity_type\":\"a2a_send\",\"method\":\"message/send\",\"summary\":\"Sent to summarizer\",\"target_id\":\"$SUM_ID\",\"duration_ms\":150}")
check "POST activity (a2a_send)" '"status":"logged"' "$R"

# Test: Report error activity
R=$(curl -s -X POST "$BASE/workspaces/$ECHO_ID/activity" -H "Content-Type: application/json" -H "Authorization: Bearer $ECHO_TOKEN" \
  -d '{"activity_type":"error","summary":"Connection timeout","status":"error","error_detail":"dial tcp: timeout after 30s"}')
check "POST activity (error)" '"status":"logged"' "$R"

# Test: Report task update
R=$(curl -s -X POST "$BASE/workspaces/$ECHO_ID/activity" -H "Content-Type: application/json" -H "Authorization: Bearer $ECHO_TOKEN" \
  -d '{"activity_type":"task_update","method":"start","summary":"Started data analysis"}')
check "POST activity (task_update)" '"status":"logged"' "$R"

# Test: Invalid activity type rejected
R=$(curl -s -X POST "$BASE/workspaces/$ECHO_ID/activity" -H "Content-Type: application/json" -H "Authorization: Bearer $ECHO_TOKEN" \
  -d '{"activity_type":"bad_type","summary":"test"}')
check "POST activity (invalid type → 400)" 'invalid activity_type' "$R"

# Test: List all activities
R=$(curl -s "$BASE/workspaces/$ECHO_ID/activity" -H "Authorization: Bearer $ECHO_TOKEN")
COUNT=$(echo "$R" | python3 -c "import sys,json; print(len(json.load(sys.stdin)))")
check "GET /workspaces/:id/activity (has entries)" "4" "$COUNT"

# Test: List activities filtered by type
R=$(curl -s "$BASE/workspaces/$ECHO_ID/activity?type=error" -H "Authorization: Bearer $ECHO_TOKEN")
COUNT=$(echo "$R" | python3 -c "import sys,json; print(len(json.load(sys.stdin)))")
check "GET activity?type=error (count=1)" "1" "$COUNT"
check "GET activity?type=error (has error_detail)" 'dial tcp' "$R"

R=$(curl -s "$BASE/workspaces/$ECHO_ID/activity?type=a2a_send" -H "Authorization: Bearer $ECHO_TOKEN")
COUNT=$(echo "$R" | python3 -c "import sys,json; print(len(json.load(sys.stdin)))")
check "GET activity?type=a2a_send (count=1)" "1" "$COUNT"
check "GET activity?type=a2a_send (has target_id)" "$SUM_ID" "$R"

# Test: List with custom limit
R=$(curl -s "$BASE/workspaces/$ECHO_ID/activity?limit=2" -H "Authorization: Bearer $ECHO_TOKEN")
COUNT=$(echo "$R" | python3 -c "import sys,json; print(len(json.load(sys.stdin)))")
check "GET activity?limit=2 (capped)" "2" "$COUNT"

# Test: Empty activity list for other workspace
R=$(curl -s "$BASE/workspaces/$SUM_ID/activity" -H "Authorization: Bearer $SUM_TOKEN")
check "GET activity (empty for summarizer)" '[]' "$R"

# ---------- Current Task Tests ----------
echo ""
echo "--- Current Task Tests ---"

# Test: Heartbeat with current_task
R=$(curl -s -X POST "$BASE/registry/heartbeat" -H "Content-Type: application/json" -H "Authorization: Bearer $ECHO_TOKEN" \
  -d "{\"workspace_id\":\"$ECHO_ID\",\"error_rate\":0.0,\"sample_error\":\"\",\"active_tasks\":1,\"uptime_seconds\":400,\"current_task\":\"Analyzing document\"}")
check "Heartbeat with current_task" '"status":"ok"' "$R"

# Test: Verify current_task in GET /workspaces/:id
R=$(acurl "$BASE/workspaces/$ECHO_ID")
check "current_task visible in workspace" '"current_task":"Analyzing document"' "$R"
check "active_tasks updated" '"active_tasks":1' "$R"

# Test: Clear current_task
R=$(curl -s -X POST "$BASE/registry/heartbeat" -H "Content-Type: application/json" -H "Authorization: Bearer $ECHO_TOKEN" \
  -d "{\"workspace_id\":\"$ECHO_ID\",\"error_rate\":0.0,\"sample_error\":\"\",\"active_tasks\":0,\"uptime_seconds\":500,\"current_task\":\"\"}")
check "Heartbeat clear current_task" '"status":"ok"' "$R"

R=$(acurl "$BASE/workspaces/$ECHO_ID")
check "current_task cleared" '"current_task":""' "$R"

# Test: current_task in workspace list — admin-gated (C1 fix), bearer required.
# fix/issue-684: GET /workspaces now on ADMIN_BASE (admin router, port 8081)
R=$(curl -s "$ADMIN_BASE/workspaces" -H "Authorization: Bearer $ECHO_TOKEN")
check "current_task in list response" '"current_task"' "$R"

# Test 21: Delete
# fix/issue-684: DELETE /workspaces/:id now on ADMIN_BASE (admin router, port 8081)
R=$(acurl -X DELETE "$ADMIN_BASE/workspaces/$ECHO_ID" -H "Authorization: Bearer $ECHO_TOKEN")
check "DELETE /workspaces/:id" '"status":"removed"' "$R"

R=$(curl -s "$ADMIN_BASE/workspaces" -H "Authorization: Bearer $SUM_TOKEN")
COUNT=$(echo "$R" | python3 -c "import sys,json; print(len(json.load(sys.stdin)))")
check "List after delete (count=1)" "1" "$COUNT"

# Test 22: Bundle round-trip — export → delete → import → verify same config
echo ""
echo "--- Bundle Round-Trip Test ---"

# Export the summarizer workspace (#165 / PR #167 — admin-gated)
# fix/issue-684: /bundles/export now on ADMIN_BASE (admin router, port 8081)
BUNDLE=$(curl -s "$ADMIN_BASE/bundles/export/$SUM_ID" -H "Authorization: Bearer $SUM_TOKEN")
check "GET /bundles/export/:id" '"name":"Summarizer Agent"' "$BUNDLE"

# Capture original config for comparison
ORIG_NAME=$(echo "$BUNDLE" | python3 -c "import sys,json; print(json.load(sys.stdin)['name'])")
ORIG_TIER=$(echo "$BUNDLE" | python3 -c "import sys,json; print(json.load(sys.stdin)['tier'])")

# Delete the workspace — use SUM_TOKEN (per-workspace) for WorkspaceAuth
# and ADMIN_TOKEN for the AdminAuth layer.
# fix/issue-684: DELETE /workspaces/:id now on ADMIN_BASE (admin router, port 8081)
R=$(curl -s -X DELETE "$ADMIN_BASE/workspaces/$SUM_ID" -H "Authorization: Bearer $SUM_TOKEN")
check "Delete before re-import" '"status":"removed"' "$R"

# After deleting the last workspace, all per-workspace tokens are revoked.
# But the test-token we minted earlier may still be in the DB as a live
# row (test-token endpoint issues tokens that aren't workspace-scoped
# for revocation). Clear ADMIN_TOKEN so acurl falls back to no-auth,
# which works when HasAnyLiveTokenGlobal = false (fail-open).
ADMIN_TOKEN=""
R=$(acurl "$ADMIN_BASE/workspaces")
COUNT=$(echo "$R" | python3 -c "import sys,json; print(len(json.load(sys.stdin)))")
check "All workspaces deleted (count=0)" "0" "$COUNT"

# Re-import from the exported bundle (AdminAuth fail-open — no live tokens)
# fix/issue-684: /bundles/import now on ADMIN_BASE (admin router, port 8081)
R=$(acurl -X POST "$ADMIN_BASE/bundles/import" -H "Content-Type: application/json" -d "$BUNDLE")
check "POST /bundles/import" '"status":"provisioning"' "$R"
NEW_ID=$(echo "$R" | python3 -c "import sys,json; print(json.load(sys.stdin)['workspace_id'])")

# Verify new ID is different from old
if [ "$NEW_ID" != "$SUM_ID" ]; then
  echo "PASS: New workspace has different ID"
  PASS=$((PASS + 1))
else
  echo "FAIL: New workspace should have a new ID"
  FAIL=$((FAIL + 1))
fi

# Verify re-imported workspace exists by name — status may be "provisioning",
# "online", or "failed" depending on runtime availability in the environment
# (CI has no Docker, so autogen/langgraph containers never come up). The
# round-trip assertion is about bundle fidelity, not provisioning success.
R=$(curl -s "$BASE/workspaces/$NEW_ID")
check "Re-imported workspace exists" "\"id\":\"$NEW_ID\"" "$R"

REIMPORT_NAME=$(echo "$R" | python3 -c "import sys,json; print(json.load(sys.stdin)['name'])")
REIMPORT_TIER=$(echo "$R" | python3 -c "import sys,json; print(json.load(sys.stdin)['tier'])")

if [ "$REIMPORT_NAME" = "$ORIG_NAME" ]; then
  echo "PASS: Name matches after round-trip ($ORIG_NAME)"
  PASS=$((PASS + 1))
else
  echo "FAIL: Name mismatch — expected '$ORIG_NAME', got '$REIMPORT_NAME'"
  FAIL=$((FAIL + 1))
fi

if [ "$REIMPORT_TIER" = "$ORIG_TIER" ]; then
  echo "PASS: Tier matches after round-trip ($ORIG_TIER)"
  PASS=$((PASS + 1))
else
  echo "FAIL: Tier mismatch — expected '$ORIG_TIER', got '$REIMPORT_TIER'"
  FAIL=$((FAIL + 1))
fi

# Register the re-imported workspace to verify agent_card round-trips
R=$(curl -s -X POST "$BASE/registry/register" -H "Content-Type: application/json" \
  -d "{\"id\":\"$NEW_ID\",\"url\":\"http://localhost:8002\",\"agent_card\":{\"name\":\"Summarizer\",\"skills\":[{\"id\":\"summarize\",\"name\":\"Summarize\"}]}}")
check "Register re-imported workspace" '"status":"registered"' "$R"
# Capture the fresh token issued to the re-imported workspace.  SUM_TOKEN was
# revoked when SUM_ID was deleted above — use this one for cleanup instead.
NEW_TOKEN=$(echo "$R" | e2e_extract_token)

# Re-export and verify agent_card survives the round-trip (#165 / PR #167 — admin-gated)
# fix/issue-684: /bundles/export now on ADMIN_BASE (admin router, port 8081)
REBUNDLE=$(curl -s "$ADMIN_BASE/bundles/export/$NEW_ID" -H "Authorization: Bearer $NEW_TOKEN")
check "Re-exported bundle has agent_card" '"agent_card"' "$REBUNDLE"

# Clean up — use the token just issued to the re-imported workspace
# fix/issue-684: DELETE /workspaces/:id now on ADMIN_BASE (admin router, port 8081)
curl -s -X DELETE "$ADMIN_BASE/workspaces/$NEW_ID" -H "Authorization: Bearer $NEW_TOKEN" > /dev/null

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
exit $FAIL
