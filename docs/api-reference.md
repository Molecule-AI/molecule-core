# API Reference

Platform API server runs on `:8080` by default. All endpoints return JSON.

**Rate limit:** 600 req/min (configurable via `RATE_LIMIT` env var).
**CORS:** `http://localhost:3000`, `http://localhost:3001` by default (configurable via `CORS_ORIGINS`).

---

## REST Endpoints

### Workspaces

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/workspaces` | Create workspace and provision container |
| `GET` | `/workspaces` | List all workspaces |
| `GET` | `/workspaces/:id` | Get single workspace |
| `PATCH` | `/workspaces/:id` | Update workspace fields |
| `DELETE` | `/workspaces/:id` | Delete workspace and remove container |
| `POST` | `/workspaces/:id/restart` | Restart workspace container |
| `POST` | `/workspaces/:id/pause` | Pause workspace (cascades to children) |
| `POST` | `/workspaces/:id/resume` | Resume paused workspace |

#### POST /workspaces

Create a new workspace. Provisions a Docker container automatically.

```json
{
  "name": "Marketing Lead",
  "role": "Manages marketing campaigns",
  "template": "general-assistant",
  "tier": 2,
  "model": "anthropic:claude-sonnet-4-6",
  "runtime": "langgraph",
  "parent_id": "uuid-of-parent",
  "canvas": { "x": 100, "y": 200 }
}
```

Response: workspace object with `id`, `status: "provisioning"`.

---

### A2A Proxy

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/workspaces/:id/a2a` | Proxy A2A JSON-RPC to workspace agent |

Forwards JSON-RPC 2.0 requests to the workspace's agent container. Automatically wraps in JSON-RPC envelope if missing.

**Headers:**
- `X-Workspace-ID` -- set to caller workspace ID for agent-to-agent calls; empty for canvas-initiated

**Timeouts:**
- Canvas-initiated (no X-Workspace-ID): 5 minutes
- Agent-to-agent (X-Workspace-ID set): 30 minutes

**Example -- send message:**
```json
{
  "jsonrpc": "2.0",
  "id": "uuid",
  "method": "message/send",
  "params": {
    "message": {
      "role": "user",
      "parts": [{ "kind": "text", "text": "Hello agent" }]
    }
  }
}
```

On success for canvas-initiated requests, also broadcasts an `A2A_RESPONSE` WebSocket event.

---

### Secrets

Secrets are encrypted with AES-256-GCM at rest. Values are never returned to the client.

#### Global Secrets

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/settings/secrets` | List global secrets (keys only) |
| `PUT` | `/settings/secrets` | Set a global secret |
| `POST` | `/settings/secrets` | Set a global secret (alias) |
| `DELETE` | `/settings/secrets/:key` | Delete a global secret |

Legacy aliases: `GET/POST/DELETE /admin/secrets` (backward compatible).

**PUT /settings/secrets:**
```json
{ "key": "ANTHROPIC_API_KEY", "value": "sk-ant-..." }
```
Response: `{ "status": "saved", "key": "ANTHROPIC_API_KEY", "scope": "global" }`

#### Workspace Secrets

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/workspaces/:id/secrets` | List merged secrets (workspace + inherited global) |
| `PUT` | `/workspaces/:id/secrets` | Set workspace-level secret override |
| `POST` | `/workspaces/:id/secrets` | Set workspace-level secret override (alias) |
| `DELETE` | `/workspaces/:id/secrets/:key` | Delete workspace-level secret |

**GET /workspaces/:id/secrets** returns a merged view:
```json
[
  { "key": "ANTHROPIC_API_KEY", "has_value": true, "scope": "workspace", "created_at": "...", "updated_at": "..." },
  { "key": "OPENAI_API_KEY", "has_value": true, "scope": "global", "created_at": "...", "updated_at": "..." }
]
```

- `scope: "workspace"` -- set directly on this workspace (overrides global)
- `scope: "global"` -- inherited from global secrets (not overridden)

Setting or deleting a workspace secret triggers an automatic container restart.

#### Precedence

When provisioning a container, secrets are loaded: global first, then workspace-specific. Workspace secrets with the same key override globals. The merged set is injected as environment variables.

#### Model Config

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/workspaces/:id/model` | Get current MODEL_PROVIDER config |

---

### Activity Logs

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/workspaces/:id/activity` | List activity logs (`?type=&limit=`) |
| `GET` | `/workspaces/:id/session-search` | Full-text search across activity + memories (`?q=&limit=`) |
| `POST` | `/workspaces/:id/activity` | Agent self-reports activity |
| `POST` | `/workspaces/:id/notify` | Agent pushes a chat message to canvas |

**POST /workspaces/:id/notify:**
```json
{ "message": "I've completed the analysis." }
```
Broadcasts an `AGENT_MESSAGE` WebSocket event. Does not persist to activity_logs.

**POST /workspaces/:id/activity:**
```json
{
  "activity_type": "a2a_send",
  "method": "message/send",
  "summary": "Delegated task to Dev Lead",
  "target_id": "uuid-of-target",
  "status": "ok",
  "duration_ms": 1500,
  "request_body": {},
  "response_body": {}
}
```
Valid activity types: `a2a_send`, `a2a_receive`, `task_update`, `agent_log`, `skill_promotion`, `error`.

---

### Registry (agent-facing)

Used by workspace agents to self-register and maintain liveness.

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/registry/register` | Agent registers on startup |
| `POST` | `/registry/heartbeat` | Agent heartbeat (includes task state) |
| `POST` | `/registry/update-card` | Agent updates its AgentCard |

**POST /registry/register:**
```json
{
  "id": "workspace-uuid",
  "url": "http://hostname:9000",
  "agent_card": { "name": "...", "skills": [...], "capabilities": {...} }
}
```
Transitions workspace status to `online`, broadcasts `WORKSPACE_ONLINE`.

**POST /registry/heartbeat:**
```json
{
  "workspace_id": "uuid",
  "current_task": "Analyzing report...",
  "active_tasks": 2,
  "error_rate": 0.0,
  "uptime_seconds": 3600
}
```
If error_rate > 0.5, broadcasts `WORKSPACE_DEGRADED`. Recovery broadcasts `WORKSPACE_ONLINE`.

---

### Discovery

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/registry/discover/:id` | Discover workspace by ID |
| `GET` | `/registry/:id/peers` | List accessible peer workspaces |
| `POST` | `/registry/check-access` | Check if two workspaces can communicate |

---

### Team Expansion

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/workspaces/:id/expand` | Expand workspace into a sub-team |
| `POST` | `/workspaces/:id/collapse` | Remove all children, collapse back to single workspace |

---

### Agents

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/workspaces/:id/agent` | Assign agent to workspace |
| `PATCH` | `/workspaces/:id/agent` | Replace agent |
| `DELETE` | `/workspaces/:id/agent` | Remove agent |
| `POST` | `/workspaces/:id/agent/move` | Move agent between workspaces |

---

### Config & Memory

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/workspaces/:id/config` | Get workspace config (JSONB) |
| `PATCH` | `/workspaces/:id/config` | Merge-patch config |
| `GET` | `/workspaces/:id/memory` | List KV memory entries |
| `GET` | `/workspaces/:id/memory/:key` | Get single KV entry |
| `POST` | `/workspaces/:id/memory` | Set KV entry (with optional TTL) |
| `DELETE` | `/workspaces/:id/memory/:key` | Delete KV entry |

---

### Agent Memories (HMA)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/workspaces/:id/memories` | Commit a memory (LOCAL, TEAM, or GLOBAL scope) |
| `GET` | `/workspaces/:id/memories` | Search memories |
| `DELETE` | `/workspaces/:id/memories/:memoryId` | Delete a memory |

---

### Approvals

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/approvals/pending` | List all pending approvals (cross-workspace) |
| `POST` | `/workspaces/:id/approvals` | Create approval request |
| `GET` | `/workspaces/:id/approvals` | List workspace approvals |
| `POST` | `/workspaces/:id/approvals/:approvalId/decide` | Approve or reject |

---

### Async Delegation

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/workspaces/:id/delegate` | Fire-and-forget delegation (`{target_id, task}`) |
| `GET` | `/workspaces/:id/delegations` | List delegations with status and results |

---

### Templates & Files

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/templates` | List available workspace templates |
| `POST` | `/templates/import` | Import template from URL |
| `GET` | `/workspaces/:id/shared-context` | Get shared context files |
| `PUT` | `/workspaces/:id/files` | Replace all config files |
| `GET` | `/workspaces/:id/files` | List files (lazy: `?depth=1&path=subdir`) |
| `GET` | `/workspaces/:id/files/*path` | Read a config file |
| `PUT` | `/workspaces/:id/files/*path` | Write a config file |
| `DELETE` | `/workspaces/:id/files/*path` | Delete a config file |

---

### Plugins

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/plugins` | List available plugins (`?runtime=<name>` filters to compatible) |
| `GET` | `/plugins/sources` | List registered install-source schemes (e.g. `github`, `local`) |
| `GET` | `/workspaces/:id/plugins` | List plugins installed in workspace |
| `GET` | `/workspaces/:id/plugins/available` | Plugins filtered to the workspace's runtime |
| `GET` | `/workspaces/:id/plugins/compatibility?runtime=X` | Preflight runtime change |
| `POST` | `/workspaces/:id/plugins` | Install plugin (`{"source":"<scheme>://<spec>"}`, e.g. `local://ecc`, `github://owner/repo#v1.0`) — auto-restarts |
| `DELETE` | `/workspaces/:id/plugins/:name` | Uninstall plugin — auto-restarts |

---

### Bundles

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/bundles/export/:id` | Export workspace as portable bundle |
| `POST` | `/bundles/import` | Import workspace from bundle |

---

### Other

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check (`{"status": "ok"}`) |
| `GET` | `/metrics` | Prometheus metrics |
| `GET` | `/events` | List structure events |
| `GET` | `/events/:workspaceId` | List events for a workspace |
| `GET` | `/workspaces/:id/traces` | Proxy to Langfuse traces |
| `GET` | `/workspaces/:id/terminal` | WebSocket terminal into container |
| `GET` | `/canvas/viewport` | Get saved canvas viewport |
| `PUT` | `/canvas/viewport` | Save canvas viewport |
| `POST` | `/webhooks/github` | GitHub webhook receiver |

---

## WebSocket Events

Connect to `ws://localhost:8080/ws`. All messages use this envelope:

```json
{
  "event": "EVENT_TYPE",
  "workspace_id": "uuid",
  "timestamp": "2024-01-01T00:00:00Z",
  "payload": { ... }
}
```

**Routing:** Canvas clients (no workspace ID) receive all events. Workspace clients receive only events for workspaces they can communicate with (per hierarchy rules).

### Workspace Lifecycle Events

These are persisted to the `structure_events` table.

| Event | Payload | Trigger |
|-------|---------|---------|
| `WORKSPACE_PROVISIONING` | `{name, tier, parent_id?}` | Container creation or restart begins |
| `WORKSPACE_ONLINE` | `{url, agent_card}` | Agent self-registers or recovers from degraded |
| `WORKSPACE_OFFLINE` | `{}` | A2A proxy detects dead container |
| `WORKSPACE_PAUSED` | `{}` | Pause operation completes |
| `WORKSPACE_DEGRADED` | `{error_rate, sample_error}` | Heartbeat reports error_rate > 0.5 |
| `WORKSPACE_REMOVED` | `{name?}` | Workspace deleted |
| `WORKSPACE_PROVISION_FAILED` | `{error}` | Container start failed |
| `WORKSPACE_EXPANDED` | `{children: [ids]}` | Team expansion complete |
| `WORKSPACE_COLLAPSED` | `{children: [ids]}` | Team collapse complete |

### Agent Events

Persisted to `structure_events`.

| Event | Payload | Trigger |
|-------|---------|---------|
| `AGENT_CARD_UPDATED` | `{agent_card}` | Agent updates its discovery card |
| `AGENT_ASSIGNED` | `{agent_id, name}` | Agent assigned to workspace |
| `AGENT_REPLACED` | `{agent_id, name}` | Agent replaced in workspace |
| `AGENT_REMOVED` | `{agent_id}` | Agent removed from workspace |
| `AGENT_MOVED` | `{from, to, agent_id}` | Agent moved (fired on both source and target) |

### Approval Events

Persisted to `structure_events`.

| Event | Payload | Trigger |
|-------|---------|---------|
| `APPROVAL_REQUESTED` | `{approval_id, workspace_id, ...}` | Agent requests human approval |
| `APPROVAL_ESCALATED` | `{approval_id, child_id, ...}` | Approval escalated to parent workspace |

### High-Frequency Events (broadcast only, not persisted)

| Event | Payload | Trigger |
|-------|---------|---------|
| `TASK_UPDATED` | `{current_task, active_tasks}` | Heartbeat includes task state changes |
| `AGENT_MESSAGE` | `{message, workspace_id, name}` | Agent pushes chat message via `POST /notify` |
| `ACTIVITY_LOGGED` | `{activity_type, method, summary, status, source_id, target_id, duration_ms}` | Any activity log insert |
| `A2A_RESPONSE` | `{response_body, method, duration_ms}` | Canvas-initiated A2A proxy returns success |

### Frontend Handling

The canvas (`canvas-events.ts`) handles these events in its Zustand store:

| Event | Frontend Action |
|-------|----------------|
| `WORKSPACE_ONLINE` | Set node status to `"online"` |
| `WORKSPACE_OFFLINE` | Set node status to `"offline"` |
| `WORKSPACE_PAUSED` | Set node status to `"paused"`, clear currentTask |
| `WORKSPACE_DEGRADED` | Set node status to `"degraded"`, store error rate |
| `WORKSPACE_PROVISIONING` | Update existing node or create new node |
| `WORKSPACE_REMOVED` | Remove node, reparent children, clean edges |
| `AGENT_CARD_UPDATED` | Update node's agentCard |
| `TASK_UPDATED` | Update node's currentTask and activeTasks |
| `AGENT_MESSAGE` | Append to chat messages for the workspace |
| `A2A_RESPONSE` | Extract response text, append to chat messages |

---

## A2A JSON-RPC Methods

Workspace agents implement the A2A protocol via the `a2a-sdk`. The Platform A2A proxy forwards these methods transparently.

### message/send

Synchronous message exchange. Blocks until the agent completes processing.

```json
{
  "jsonrpc": "2.0",
  "id": "unique-id",
  "method": "message/send",
  "params": {
    "message": {
      "messageId": "unique-msg-id",
      "role": "user",
      "parts": [
        { "kind": "text", "text": "Analyze the Q4 report" }
      ]
    }
  }
}
```

Response contains the agent's reply message with `parts` (text, data, etc.).

### message/stream

SSE streaming variant of `message/send`. Returns token-level Server-Sent Events as the agent generates its response.

### tasks/get

Poll the status of a previously submitted async task.

```json
{
  "jsonrpc": "2.0",
  "id": "unique-id",
  "method": "tasks/get",
  "params": {
    "id": "task-uuid"
  }
}
```

Returns task state: `submitted`, `working`, `input-required`, `completed`, `failed`, `canceled`.
