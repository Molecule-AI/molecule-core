# Molecule AI MCP Server

MCP server that exposes Molecule AI platform operations as tools for AI coding agents.

## 20 Tools Available

| Tool | Description |
|------|-------------|
| `list_workspaces` | List all workspaces with status and skills |
| `create_workspace` | Create a new workspace (with optional template) |
| `get_workspace` | Get workspace details |
| `delete_workspace` | Delete workspace (cascades to children) |
| `restart_workspace` | Restart offline/failed workspace |
| `chat_with_agent` | Send message and get AI response |
| `assign_agent` | Assign model to workspace |
| `set_secret` | Set API key or env var |
| `list_secrets` | List secret keys (no values) |
| `list_files` | List workspace config files |
| `read_file` | Read a config file |
| `write_file` | Create or update a file |
| `delete_file` | Delete file or folder |
| `commit_memory` | Store fact (LOCAL/TEAM/GLOBAL) |
| `search_memory` | Search workspace memories |
| `list_templates` | List available templates |
| `expand_team` | Expand workspace to team |
| `collapse_team` | Collapse team to single workspace |
| `list_pending_approvals` | List pending approval requests |
| `decide_approval` | Approve or deny a request |

### Phase 30 — Remote agent (SaaS) management

Tools that surface workspaces with `runtime='external'` (agents that run on
machines outside this platform's Docker network and join via HTTP).

| Tool | Description |
|------|-------------|
| `list_remote_agents` | Filter the workspace list to remote agents only — id / status / url / heartbeat |
| `get_remote_agent_state` | Lightweight `{status, paused, deleted}` projection — faster than `get_workspace` when you only need lifecycle |
| `get_remote_agent_setup_command` | Emit a `WORKSPACE_ID=… PLATFORM_URL=… python3 …` bash one-liner an operator can paste into a remote shell |
| `check_remote_agent_freshness` | Compare `last_heartbeat_at` against a threshold (default 90s) — returns `{fresh, seconds_since_heartbeat}` |

## Setup

### Claude Code

Add to your project's `.mcp.json`:

```json
{
  "mcpServers": {
    "molecule": {
      "command": "node",
      "args": ["./mcp-server/dist/index.js"],
      "env": {
        "MOLECULE_URL": "http://localhost:8080"
      }
    }
  }
}
```

### Cursor

Add to `.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "molecule": {
      "command": "node",
      "args": ["./mcp-server/dist/index.js"],
      "env": {
        "MOLECULE_URL": "http://localhost:8080"
      }
    }
  }
}
```

### Codex / OpenCode

```bash
# Run directly
MOLECULE_URL=http://localhost:8080 node mcp-server/dist/index.js
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `MOLECULE_URL` | `http://localhost:8080` | Platform API URL |

## Examples

```
You: "Create an SEO agent workspace using the seo-agent template"
Agent: [calls create_workspace with template="seo-agent"]

You: "Set the OpenRouter API key for the SEO workspace"
Agent: [calls set_secret with key="OPENROUTER_API_KEY"]

You: "Ask the SEO agent to audit my homepage"
Agent: [calls chat_with_agent with message="Audit https://example.com for SEO"]

You: "What skills does the coding agent have?"
Agent: [calls get_workspace, reads agent_card.skills]
```
