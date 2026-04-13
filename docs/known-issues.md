# Known Issues

Issues identified in source but not yet filed as GitHub issues (GH_TOKEN unavailable in
automated agent contexts). Each entry has: location, symptom, impact, suggested fix.

---

## KI-001 — Telegram channel `kicked` event does not persist disabled state to DB

**File:** `platform/internal/channels/telegram.go:596`  
**Status:** TODO comment in source, unimplemented  
**Severity:** Medium

### Symptom
When the Molecule AI bot is removed from a Telegram chat (`left` or `kicked` event), the handler
logs the event but does not update the `workspace_channels` row to mark the channel as
`enabled: false`. On the next scheduled outbound message or webhook trigger, the platform
attempts to send to a chat the bot no longer belongs to, receives a Telegram 403 error, and
logs an error — but keeps retrying on every subsequent trigger indefinitely.

### Code pointer
```go
// telegram.go:594-596
case "left", "kicked":
    log.Printf("Channels: Telegram bot removed from chat %d (%s)", chat.ID, chat.Title)
    // TODO: mark channel disabled in DB
```

### Suggested fix
After the `log.Printf`, call the channel manager's update method to set `enabled = false`
on the matching `workspace_channels` row (look up by `config->>'chat_id'`). Requires
injecting a DB handle or update callback into the Telegram handler — same pattern used
by `manager.go`'s `clearChatHistory` callback at line 603.

---

## KI-002 — Delegation system has no idempotency guard against duplicate execution on container-restart race

**File:** `platform/internal/handlers/delegation.go` (see also `delegationRetryDelay`)  
**Status:** Identified in `docs/ecosystem-watch.md` (Trigger.dev section); no fix yet  
**Severity:** Medium

### Symptom
When a workspace container restarts mid-delegation (e.g. Redis TTL expires, liveness monitor
triggers restart), the `POST /workspaces/:id/delegate` call may fire again on the next agent
boot before the first delegation's result is stored. The target workspace executes the same
task twice, potentially producing duplicate side-effects (double commits, double API calls,
double Telegram messages).

### Code pointer
`delegation.go` stores delegations in the DB but uses no idempotency key. The caller
(workspace agent) has no way to detect that a delegation was already accepted; it simply
retries if the HTTP call times out.

### Suggested fix
Accept an optional `idempotency_key` field in the `POST /workspaces/:id/delegate` request
body. On receipt, check for an existing delegation row with the same `(workspace_id,
idempotency_key)` pair. If found and not failed, return the existing delegation ID (HTTP 200)
rather than creating a new row. Agents should pass `idempotency_key = sha256(task_text +
timestamp_minute)` to scope deduplication to a natural retry window.

---

## KI-003 — `commit_memory` MCP tool calls are not surfaced in `activity_logs`

**File:** `workspace-template/builtin_tools/memory.py` + `platform/internal/handlers/activity.go`  
**Status:** Identified in `docs/ecosystem-watch.md` (Letta section); no fix yet  
**Severity:** Low (visibility / debugging quality)

### Symptom
When an agent calls `commit_memory`, the write succeeds and is persisted to the
`agent_memories` table, but no `activity_log` row is created. Operators inspecting the
Canvas chat "Agent Comms" tab cannot see that a memory write occurred, making it hard to
audit what an agent chose to remember during a task.

### Suggested fix
In the MCP server's `commit_memory` handler (or in the platform's `POST /workspaces/:id/memories`
handler), emit an `activity_log` entry of type `tool_call` with `method = "commit_memory"`,
`request = {key, content_length}`, and `duration_ms`. This matches the Letta pattern of
making memory operations first-class visible tool calls in the trace timeline.
