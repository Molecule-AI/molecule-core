# Tool Trace + Platform Instructions — Verification Steps

> **Source PRs:** molecule-core#1686 (feat: tool trace + platform instructions)
> **Staged on:** staging | **Demo package:** `docs/devrel/demos/tool-trace-platform-instructions/`
> **Phase:** Phase 34 | **GA Target:** April 30, 2026

---

## Verification Checklist

### 1. Code + Migration Presence

```bash
# Check migrations 039 and 040 are on staging
gh api repos/Molecule-AI/molecule-core/contents/workspace-server/migrations \
  --ref staging --jq '.[].name' 2>/dev/null | grep -E "039|040"

# Check handler files exist
gh api repos/Molecule-AI/molecule-core/contents/workspace-server/internal/handlers \
  --ref staging --jq '.[].name' 2>/dev/null | grep -i "instruct\|activity"

# Check workspace-side tool_trace injection
gh api repos/Molecule-AI/molecule-core/contents/workspace/a2a_executor.py \
  --ref staging --jq '.size' 2>/dev/null
```

**Expected:** Both migration files present. `instructions.go` and `activity.go` handlers present. `a2a_executor.py` exists and contains `_build_tool_trace` or `tool_trace` references.

---

### 2. API Endpoint Smoke Tests

Run against staging platform with admin credentials:

```bash
PLATFORM_URL="https://platform-staging.molecule.ai"
ADMIN_TOKEN="your-admin-token"
WORKSPACE_ID="test-workspace-id"

# Test 1: Create global instruction
RESP=$(curl -s -w "\n%{http_code}" -X POST "$PLATFORM_URL/instructions" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"scope": "global", "title": "test instruction", "content": "test content", "priority": 1}')
echo "$RESP" | tail -1  # expect 201

# Test 2: List global instructions
curl -s "$PLATFORM_URL/instructions?scope=global" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.[0].scope'  # expect "global"

# Test 3: Resolve workspace instructions
curl -s "$PLATFORM_URL/workspaces/$WORKSPACE_ID/instructions/resolve" \
  -H "X-Workspace-ID: $WORKSPACE_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.instructions | startswith("# Platform Instructions")'  # expect true

# Test 4: Query activity logs with tool_trace
curl -s "$PLATFORM_URL/workspaces/$WORKSPACE_ID/activity?limit=5" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq '.[0].tool_trace'  # expect array or null
```

**Expected:** 201 status on create. Instruction list returns entries. Resolve returns markdown-formatted string. Activity log entries contain `tool_trace` key (may be null for new workspaces — run an A2A call first to populate).

---

### 3. Content-Disposition Check (8KB Cap)

```bash
# Test: oversized instruction rejected
RESP=$(curl -s -w "\n%{http_code}" -X POST "$PLATFORM_URL/instructions" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"scope\": \"global\", \"title\": \"test\", \"content\": \"$(printf 'x%.0s' 9000)\", \"priority\": 1}")
echo "$RESP" | tail -1  # expect 400 or 422
```

**Expected:** 400/422 — oversized instruction rejected at DB CHECK constraint level.

---

### 4. Screencast Storyboard Accuracy

Verify `screencast-storyboard.md` matches actual API behavior:
- Command syntax matches actual endpoint paths
- Response shapes match actual API responses
- 5-moment structure covers all key user journeys
- TTS narration script in `narration.txt` covers all 5 moments

```bash
# Quick: verify storyboard scenarios against actual endpoints
# Scenario 1 — global instruction create
gh api repos/Molecule-AI/molecule-core/contents/docs/devrel/demos/tool-trace-platform-instructions/screencast-storyboard.md \
  --ref staging --jq '.content' | base64 -d | grep -c "curl.*POST.*instructions"  # expect ≥1

# Scenario 5 — activity log query
grep -c "activity?limit" docs/devrel/demos/tool-trace-platform-instructions/screencast-storyboard.md  # expect ≥1
```

---

### 5. Demo Package Completeness

| File | Purpose | Status |
|---|---|---|
| `README.md` | Full runnable demo (5 scenarios, API examples, architecture) | ✅ Present |
| `narration.txt` | TTS script for ~90s screencast, 5 moments | ✅ Present |
| `screencast-storyboard.md` | Frame-by-frame storyboard with timings, narration, production notes | ✅ Present (this file) |
| `verification.md` | Verification steps + checklist | ✅ Present (this file) |

---

## Feature Behavior Summary

### Tool Trace

- **Lifecycle:** A2A request → agent executes tools (with shared `run_id`) → `tool_trace` list built in `a2a_executor.py` → included in A2A response `metadata` → stored in `activity_logs.tool_trace` JSONB
- **Entry fields:** `tool` (string), `input` (object, sanitized), `output_preview` (string, max 200 chars)
- **Cap:** 200 entries per A2A turn (prevents unbounded growth in runaway loops)
- **Query:** `GET /workspaces/:id/activity?limit=N` → `tool_trace` key on each log entry

### Platform Instructions

- **Create:** `POST /instructions` (admin token, requires `instructions:create` scope)
- **List:** `GET /instructions?scope=global|workspace` (admin token)
- **Resolve:** `GET /workspaces/:id/instructions/resolve` (workspace token, gated by WorkspaceAuth — no cross-workspace enumeration)
- **Injection:** resolved string prepended as `# Platform Instructions` section → first section of system prompt → highest precedence
- **Refresh:** fetched at workspace boot and periodically during agent runtime
- **Security:** 8KB content cap via `CHECK (length(content) <= 8192)` DB constraint + `maxInstructionContentLen` handler constant

---

## Sign-off Checklist

- [ ] Migrations 039 + 040 applied on staging
- [ ] All 4 API endpoint smoke tests pass
- [ ] 8KB cap enforcement confirmed (oversized instruction rejected)
- [ ] Storyboard commands match actual API paths
- [ ] Screencast narration script covers all 5 moments
- [ ] Demo package complete (4 files: README, narration, storyboard, verification)
- [ ] Phase 34 talk-track updated with Tool Trace talking points (see `marketing/devrel/phase34-talk-track.md`)

---

*Verification completed: 2026-04-23 | DevRel Engineer*