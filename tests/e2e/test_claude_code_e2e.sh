#!/bin/bash
# Full E2E test for Claude Code workspace runtime
# Run from repo root after: docker compose up -d && docker build -t workspace-template:latest workspace-template/
#
# Prerequisites:
#   - Platform running on localhost:8080
#   - workspace-template:latest image built
#   - .auth-token in workspace-configs-templates/claude-code-default/

set -euo pipefail

PLATFORM="http://localhost:8080"
PASS=0
FAIL=0
ERRORS=""

pass() { echo "PASS: $1"; PASS=$((PASS+1)); }
fail() { echo "FAIL: $1"; echo "  expected: $2"; echo "  got: $3"; FAIL=$((FAIL+1)); ERRORS="$ERRORS\n  - $1"; }

check_contains() {
  if echo "$3" | grep -qi "$2"; then pass "$1"; else fail "$1" "contains '$2'" "$3"; fi
}

# --- Health Check ---
echo "=== Claude Code E2E Tests ==="
echo ""

HEALTH=$(curl -s $PLATFORM/health)
check_contains "Platform healthy" '"status":"ok"' "$HEALTH"

# --- Verify auth token exists ---
if [ -f workspace-configs-templates/claude-code-default/.auth-token ]; then
  pass "Auth token file exists"
else
  fail "Auth token file exists" "file present" "missing"
  echo "FATAL: No .auth-token. Write your Claude Code OAuth token to workspace-configs-templates/claude-code-default/.auth-token"
  exit 1
fi

# --- Clean existing workspaces ---
for id in $(curl -s $PLATFORM/workspaces | python3 -c "import sys,json; [print(w['id']) for w in json.load(sys.stdin)]" 2>/dev/null); do
  curl -s -X DELETE "$PLATFORM/workspaces/$id" > /dev/null
done
docker stop $(docker ps -q --filter "name=ws-") 2>/dev/null || true
docker rm $(docker ps -aq --filter "name=ws-") 2>/dev/null || true

# --- Create Org Chart ---
echo ""
echo "--- Create Workspaces ---"

ROOT=$(curl -s -X POST $PLATFORM/workspaces -H "Content-Type: application/json" \
  -d '{"name":"Root Agent","role":"Company coordinator","runtime":"claude-code","tier":3}' \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
check_contains "Create root workspace" "-" "$ROOT"

CHILD=$(curl -s -X POST $PLATFORM/workspaces -H "Content-Type: application/json" \
  -d "{\"name\":\"Child Agent\",\"role\":\"Sub-team member\",\"runtime\":\"claude-code\",\"tier\":2,\"parent_id\":\"$ROOT\"}" \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
check_contains "Create child workspace" "-" "$CHILD"

# --- Wait for online ---
echo ""
echo "--- Wait for provisioning (40s) ---"
sleep 40

ROOT_STATUS=$(curl -s "$PLATFORM/workspaces/$ROOT" | python3 -c "import sys,json; print(json.load(sys.stdin)['status'])")
check_contains "Root is online" "online" "$ROOT_STATUS"

CHILD_STATUS=$(curl -s "$PLATFORM/workspaces/$CHILD" | python3 -c "import sys,json; print(json.load(sys.stdin)['status'])")
check_contains "Child is online" "online" "$CHILD_STATUS"

# --- Containers running ---
CONTAINER_COUNT=$(docker ps --filter "name=ws-" -q | wc -l | tr -d ' ')
if [ "$CONTAINER_COUNT" -eq 2 ]; then pass "2 containers running"; else fail "2 containers running" "2" "$CONTAINER_COUNT"; fi

# --- Upload system prompts ---
echo ""
echo "--- Upload System Prompts ---"

ROOT_UPLOAD=$(curl -s -X PUT "$PLATFORM/workspaces/$ROOT/files" \
  -H "Content-Type: application/json" \
  -d '{"files":{"system-prompt.md":"You are the Root Agent. You coordinate sub-teams. The company is called TestCorp."}}')
check_contains "Upload root prompt" "replaced" "$ROOT_UPLOAD"

CHILD_UPLOAD=$(curl -s -X PUT "$PLATFORM/workspaces/$CHILD/files" \
  -H "Content-Type: application/json" \
  -d '{"files":{"system-prompt.md":"You are a Child Agent under Root. You specialize in data analysis for TestCorp."}}')
check_contains "Upload child prompt" "replaced" "$CHILD_UPLOAD"

# Verify prompts in containers
sleep 2
ROOT_CONTAINER=$(docker ps --filter "name=ws-${ROOT:0:12}" -q | head -1)
CHILD_CONTAINER=$(docker ps --filter "name=ws-${CHILD:0:12}" -q | head -1)

ROOT_HAS_PROMPT=$(docker exec $ROOT_CONTAINER cat /configs/system-prompt.md 2>/dev/null | head -1)
check_contains "Root container has prompt" "Root Agent" "$ROOT_HAS_PROMPT"

CHILD_HAS_PROMPT=$(docker exec $CHILD_CONTAINER cat /configs/system-prompt.md 2>/dev/null | head -1)
check_contains "Child container has prompt" "Child Agent" "$CHILD_HAS_PROMPT"

# --- A2A Tests ---
echo ""
echo "--- A2A Communication ---"

ROOT_REPLY=$(curl -s -X POST "$PLATFORM/workspaces/$ROOT/a2a" \
  -H "Content-Type: application/json" --max-time 90 \
  -d '{"jsonrpc":"2.0","id":"t1","method":"message/send","params":{"message":{"role":"user","parts":[{"type":"text","text":"What company do you work for? One word."}]}}}')
ROOT_TEXT=$(echo "$ROOT_REPLY" | python3 -c "import sys,json; d=json.load(sys.stdin); p=d.get('result',{}).get('parts',[]); print(p[0]['text'] if p else d.get('error',{}).get('message','EMPTY'))" 2>/dev/null)
check_contains "Root knows company name" "TestCorp" "$ROOT_TEXT"

CHILD_REPLY=$(curl -s -X POST "$PLATFORM/workspaces/$CHILD/a2a" \
  -H "Content-Type: application/json" --max-time 90 \
  -d '{"jsonrpc":"2.0","id":"t2","method":"message/send","params":{"message":{"role":"user","parts":[{"type":"text","text":"What do you specialize in? One phrase."}]}}}')
CHILD_TEXT=$(echo "$CHILD_REPLY" | python3 -c "import sys,json; d=json.load(sys.stdin); p=d.get('result',{}).get('parts',[]); print(p[0]['text'] if p else d.get('error',{}).get('message','EMPTY'))" 2>/dev/null)
check_contains "Child knows its specialty" "data" "$CHILD_TEXT"

# --- Access Control ---
echo ""
echo "--- Access Control ---"

PARENT_CHILD=$(curl -s -X POST $PLATFORM/registry/check-access -H "Content-Type: application/json" \
  -d "{\"caller_id\":\"$ROOT\",\"target_id\":\"$CHILD\"}" | python3 -c "import sys,json; print(json.load(sys.stdin).get('allowed','?'))")
check_contains "Parent→Child allowed" "True" "$PARENT_CHILD"

CHILD_PARENT=$(curl -s -X POST $PLATFORM/registry/check-access -H "Content-Type: application/json" \
  -d "{\"caller_id\":\"$CHILD\",\"target_id\":\"$ROOT\"}" | python3 -c "import sys,json; print(json.load(sys.stdin).get('allowed','?'))")
check_contains "Child→Parent allowed" "True" "$CHILD_PARENT"

# --- Canvas ---
echo ""
echo "--- Canvas ---"
CANVAS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:3000)
check_contains "Canvas returns 200" "200" "$CANVAS"

# --- Summary ---
echo ""
echo "==============================="
echo "  PASS: $PASS  FAIL: $FAIL"
echo "==============================="
if [ $FAIL -gt 0 ]; then
  echo -e "\nFailed tests:$ERRORS"
  exit 1
fi
