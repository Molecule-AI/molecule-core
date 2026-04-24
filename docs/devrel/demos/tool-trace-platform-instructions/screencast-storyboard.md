# Screencast Storyboard — Tool Trace + Platform Instructions Demo
> **Source:** PR #1686 (molecule-core) | **Feature:** Tool Trace + Platform Instructions
> **Duration:** ~90s | **Format:** API terminal + Canvas UI, dark zinc theme, blue/purple accents
> **Reference code:** `workspace-server/internal/handlers/instructions.go`, `workspace/a2a_executor.py`
> **Reference API:** `POST /instructions`, `GET /instructions`, `GET /workspaces/:id/instructions/resolve`, `GET /workspaces/:id/activity`

---

## What PR #1686 Ships

Two platform-level observability + control features:

**Tool Trace** — Every A2A response includes `tool_trace` in `Message.metadata`. Entries: tool name, input args (sanitized), output preview (200-char cap). Stored in `activity_logs.tool_trace` (JSONB + GIN index). `run_id` pairs parallel tool calls. 200-entry cap per A2A turn prevents runaway loops.

**Platform Instructions** — Admin-defined AI governance rules (global or workspace-scoped) injected as the first section of every agent's system prompt at boot and on periodic refresh. 8KB content cap enforced by DB CHECK constraint. Resolved via `GET /workspaces/:id/instructions/resolve` (workspace-token gated).

---

## Pre-roll (0:00–0:08)

**Terminal — clean prompt, PLATFORM_URL + ADMIN_TOKEN set.**

```
$ export PLATFORM_URL=https://platform.molecule.ai
$ export ADMIN_TOKEN=mol_sk_...
$ export WORKSPACE_ID=ws-abc123
$
```

Narration (0:00–0:07):
> "Two features shipping together in Phase 34. Tool Trace — every tool call an agent makes, logged and queryable. Platform Instructions — AI governance rules enforced at the platform layer, not the prompt layer. Let's walk through both."

**Camera:** Static. 6-second hold. Clean terminal frame.

---

## Moment 1 — Admin creates a global instruction (0:08–0:25)

**Terminal — cursor at `$`.**

Type (shown on screen):
```bash
curl -s -X POST "$PLATFORM_URL/instructions" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "scope": "global",
    "title": "No shell commands in user-facing agents",
    "content": "Agents must NOT execute shell commands for users. Use file read/write tools or MCP tools only.",
    "priority": 10
  }' | jq .
```

Press Enter.

**Camera:** Hold on the command for 2s. Output appears:

```json
{
  "id": "inst-01",
  "scope": "global",
  "title": "No shell commands in user-facing agents",
  "content": "Agents must NOT execute shell commands...",
  "priority": 10,
  "enabled": true,
  "created_at": "2026-04-23T12:00:00Z"
}
```

Narration (0:10–0:23):
> "Admins create platform-wide instructions in seconds. Scope 'global' applies to every workspace automatically. The priority field controls merge order. The content is capped at 8KB by the database — token budget DoS isn't possible."

---

## Moment 2 — Admin targets a specific workspace (0:25–0:40)

**Terminal — cursor at `$`.**

Type:
```bash
curl -s -X POST "$PLATFORM_URL/instructions" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"scope\": \"workspace\",
    \"scope_target\": \"$WORKSPACE_ID\",
    \"title\": \"Use dark theme by default\",
    \"content\": \"When generating UI components, default to the dark theme unless the user explicitly requests light mode.\",
    \"priority\": 5
  }" | jq .
```

Press Enter.

Output:
```json
{
  "id": "inst-02",
  "scope": "workspace",
  "scope_target": "ws-abc123",
  "title": "Use dark theme by default",
  "priority": 5,
  "enabled": true,
  ...
}
```

Narration (0:27–0:38):
> "Or target a specific workspace with a workspace-scoped instruction. Great for onboarding rules, per-project policies, or defaulting a workspace to a specific configuration. The scope_target ties the instruction to a specific workspace ID."

---

## Moment 3 — Workspace boots, instructions resolved (0:40–0:55)

**Terminal — cursor at `$`. Switch to second pane: Canvas workspace booting.**

Canvas: workspace starting up, spinner visible.

Type:
```bash
curl -s "$PLATFORM_URL/workspaces/$WORKSPACE_ID/instructions/resolve" \
  -H "X-Workspace-ID: $WORKSPACE_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq .
```

Output:
```json
{
  "workspace_id": "ws-abc123",
  "instructions": "# Platform Instructions\n\n> No shell commands in user-facing agents\n...\n> Use dark theme by default\n..."
}
```

Canvas: workspace ready, agent responding.

Narration (0:42–0:53):
> "When a workspace boots, it calls the resolve endpoint using its workspace token. Global and workspace-scoped instructions are merged into one string. The call is gated by WorkspaceAuth — no cross-workspace enumeration. A platform outage never blocks agent startup."

---

## Moment 4 — System prompt injection (0:55–1:08)

**Terminal — two-pane layout.**

Left pane — workspace boot log:
```
[BOOT] Fetching platform instructions...
[BOOT] Merged 2 instructions (global + workspace)
[BOOT] Prepending to system prompt as "# Platform Instructions"
[BOOT] Ready. Agent responding.
```

Right pane — partial system prompt (truncated after `# Platform Instructions`):
```
# Platform Instructions

> No shell commands in user-facing agents. Use file read/write tools or MCP tools only.

> When generating UI components, default to the dark theme...

[... agent prompt continues ...]
```

Narration (0:56–1:06):
> "That resolved string is injected as the first section of the agent's system prompt — so it has highest precedence. Agents receive these instructions at boot and on every periodic refresh. They cannot override them. This is enforced at the runtime layer, not the prompting layer."

---

## Moment 5 — Query activity logs, see tool_trace (1:08–1:28)

**Terminal — cursor at `$`.**

Type:
```bash
curl -s "$PLATFORM_URL/workspaces/$WORKSPACE_ID/activity?limit=3" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  | jq '.[] | {id, activity_type, created_at, tool_trace}'
```

Output:
```json
[
  {
    "id": "log-abc123",
    "activity_type": "a2a_call",
    "created_at": "2026-04-23T12:01:00Z",
    "tool_trace": [
      {
        "tool": "mcp__files__read",
        "input": {"path": "config.yaml"},
        "output_preview": "api_version: v2, region: us-east-1, ..."
      },
      {
        "tool": "mcp__httpx__get",
        "input": {"url": "https://api.example.com/status"},
        "output_preview": "{\"status\": \"ok\", \"latency_ms\": 42}"
      }
    ]
  }
]
```

Narration (0:56–1:06):
> "After every A2A call, the platform stores which tools were actually invoked in the activity log. Admins can query the activity endpoint and see the full tool trace — tool name, sanitized input, and a 200-character output preview. Parallel tool calls are captured correctly via shared run ID."

---

## Close (1:26–1:30)

**Terminal — clean frame, cursor at `$`.**

Narration (1:26–1:28):
> "Tool Trace and Platform Instructions. Observability and control — built into the platform, not bolted on after."

**End card:**
```
Tool Trace — every tool call, logged. Queryable.
Platform Instructions — AI governance at runtime, not prompting.
Phase 34 — moleculesai.app/phase-34
molecule-core#1686
```

**Fade to black.**

---

## Production Notes

- **Theme:** dark zinc (#09090B), terminal SF Mono 12pt, Canvas UI with blue (#3B82F6) accents
- **Recording:** 1440×900 record → 1080p export. Command entry animation: ~3s per command (type + execute)
- **Two-pane layout** for Moment 4 — use a split terminal or overlay cards
- **VO:** consistent professional voice across all Phase 34 demo assets
- **Credentials:** Use env vars (`PLATFORM_URL`, `ADMIN_TOKEN`, `WORKSPACE_ID`) throughout — no hardcoded values on screen
- **Verification:** Before recording, confirm staging has migrations 039 (tool_trace) and 040 (platform_instructions) applied

## Reference Code Locations

| Component | File |
|---|---|
| Tool trace store + return | `workspace-server/internal/handlers/activity.go` |
| Tool trace A2A metadata | `workspace/a2a_executor.py` (calls `_build_tool_trace`) |
| Platform Instructions CRUD | `workspace-server/internal/handlers/instructions.go` |
| Instruction resolve endpoint | `GET /workspaces/:id/instructions/resolve` (gated) |
| System prompt injection | `workspace/prompt.py` (`get_platform_instructions()` → `build_system_prompt`) |
| DB migration (tool_trace) | `workspace-server/migrations/039_activity_tool_trace.up.sql` |
| DB migration (instr.) | `workspace-server/migrations/040_platform_instructions.up.sql` |