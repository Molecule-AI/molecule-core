# opencode MCP Integration

Connect [opencode](https://opencode.ai) to the Molecule AI platform so your CLI sessions participate in the A2A mesh — delegate tasks to other workspaces, read shared memory, and send real-time messages to the canvas without leaving the terminal.

## How it works

The platform exposes each workspace as a remote MCP server:

```
GET  /workspaces/:id/mcp/stream   — SSE transport (backwards compat)
POST /workspaces/:id/mcp          — Streamable HTTP transport (primary)
```

Both endpoints are protected by the workspace bearer token (same credential as the A2A API). The opencode client sends the token in `Authorization: Bearer <token>` on every request.

## Quick start

### 1. Get your credentials

```bash
# Platform URL (default: http://localhost:8080 for local dev)
export MOLECULE_MCP_URL=http://localhost:8080

# Workspace ID — shown in the Canvas sidebar or via:
curl -s $MOLECULE_MCP_URL/workspaces | jq '.[0].id'

# Bearer token — mint one via:
curl -s -X POST "$MOLECULE_MCP_URL/workspaces/$WORKSPACE_ID/tokens" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq -r '.token'
```

### 2. Configure opencode

Copy `org-templates/molecule-dev/opencode.json` to `~/.config/opencode/config.json`
(or merge it into your existing config) and set the environment variables:

```bash
export MOLECULE_MCP_URL=http://localhost:8080
export WORKSPACE_ID=<your-workspace-id>
export MOLECULE_MCP_TOKEN=<your-bearer-token>
```

Or set them inline in the config (not recommended for tokens):

```json
{
  "mcpServers": {
    "molecule": {
      "type": "remote",
      "url": "http://localhost:8080/workspaces/ws-abc123/mcp",
      "headers": {
        "Authorization": "Bearer msk_live_abc123..."
      }
    }
  }
}
```

### 3. Start opencode

```bash
opencode
```

The `molecule` MCP server is now available. Type `/tools` in opencode to confirm.

## Available tools

| Tool | Description |
|------|-------------|
| `list_peers` | List reachable workspaces (siblings, parent, children) |
| `get_workspace_info` | Get this workspace's ID, name, role, tier, status |
| `delegate_task` | Synchronous task delegation — waits up to 30 s for a response |
| `delegate_task_async` | Fire-and-forget delegation — returns a `task_id` immediately |
| `check_task_status` | Poll an async task's status and result |
| `commit_memory` | Save information to LOCAL or TEAM persistent memory |
| `recall_memory` | Search LOCAL or TEAM memory |
| `send_message_to_user` | Push a message to the canvas chat *(opt-in, see below)* |

## Optional: enable send_message_to_user

`send_message_to_user` is excluded from the tool list by default to prevent
accidental WebSocket pushes from CLI sessions. To opt in, set:

```bash
# In the platform's environment (e.g. .env or fly secrets set):
MOLECULE_MCP_ALLOW_SEND_MESSAGE=true
```

## Rate limiting

The MCP bridge enforces **120 requests / minute / token**. Long-running opencode sessions that issue many tool calls in rapid succession will see `429 Too Many Requests` with a `Retry-After` header. The standard MCP client will back off automatically.

## Security notes

- **Scope isolation**: `commit_memory` and `recall_memory` only accept `LOCAL` and `TEAM` scopes. `GLOBAL` scope is blocked at the MCP layer (use the internal `a2a_mcp_server.py` for GLOBAL writes from within a workspace container).
- **Access control**: `delegate_task` / `delegate_task_async` verify `CanCommunicate(caller, target)` before forwarding any A2A message — the same check the A2A proxy enforces.
- **Token binding**: each bearer token is bound to a single workspace; cross-workspace impersonation is not possible.

## Troubleshooting

| Symptom | Likely cause | Fix |
|---------|-------------|-----|
| `401 Unauthorized` | Missing or expired bearer token | Mint a new token via `POST /workspaces/:id/tokens` |
| `403 Forbidden` on `delegate_task` | Target workspace is not a peer | Use `list_peers` to find valid targets |
| `429 Too Many Requests` | Rate limit exceeded | Wait `Retry-After` seconds; reduce call frequency |
| `delegate_task` hangs | Target workspace is offline / hibernated | Check workspace status in Canvas; wake it if hibernated |
| `send_message_to_user` returns permission error | Opt-in env var not set | Set `MOLECULE_MCP_ALLOW_SEND_MESSAGE=true` on the platform |
