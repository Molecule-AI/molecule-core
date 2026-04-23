# Tool Trace + Platform Instructions — Interactive Demo Script
**Issue:** — | **Source:** PR #1686 | **Acceptance:** Working demo + repo link + 1-min screencast

---

## What This Demo Shows

1. Inject a global platform instruction via the admin API
2. See how an agent picks it up at startup
3. Inspect the tool trace in activity logs — what the agent actually called

**Time:** ~60 seconds | **Tools:** curl, jq | **Setup:** `ORG_API_KEY` with AdminAuth

---

## Demo Script

### Step 1: Create a Global Platform Instruction

```bash
curl -s -X POST https://your-deployment.moleculesai.app/instructions \
  -H "Authorization: Bearer $ORG_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "scope": "global",
    "title": "Cost Guardrails",
    "content": "Always estimate token cost before calling the web search tool. Skip if the query is trivial.",
    "priority": 100
  }' | jq .
```

Expected output:
```json
{
  "id": "inst_abc123",
  "scope": "global",
  "title": "Cost Guardrails",
  "priority": 100,
  "enabled": true,
  "created_at": "2026-04-23T00:00:00Z"
}
```

**Narrative:** "Platform operators can now inject rules — like cost guardrails — that apply to every agent across all workspaces, automatically."

---

### Step 2: Resolve Instructions for a Workspace

```bash
curl -s https://your-deployment.moleculesai.app/workspaces/$WORKSPACE_ID/instructions/resolve \
  -H "Authorization: Bearer $WORKSPACE_TOKEN" | jq .
```

Expected output:
```json
{
  "instructions": "## Platform Instructions\n\n## Cost Guardrails\nAlways estimate token cost before calling the web search tool. Skip if the query is trivial.\n"
}
```

**Narrative:** "At startup, the agent fetches its merged instructions and prepends them to its system prompt — highest context precedence."

---

### Step 3: Inspect the Tool Trace in Activity Logs

Tool traces are attached to every A2A response as `metadata.tool_trace`. After an agent run, query activity logs:

```bash
curl -s https://your-deployment.moleculesai.app/activity-logs?workspace_id=$WORKSPACE_ID&limit=1 \
  -H "Authorization: Bearer $ORG_API_KEY" | \
  jq '.logs[0].tool_trace[:3]'
```

Expected output — each entry captures name, input, and a 300-char output preview:
```json
[
  {
    "tool": "bash",
    "input": "find . -name '*.py' | xargs wc -l",
    "output_preview": "42 total src/main.py..."
  },
  {
    "tool": "read_file",
    "input": "src/agent.py",
    "output_preview": "async def run(prompt: str) -> Message:\n    ..."
  },
  {
    "tool": "WebSearch",
    "input": "Claude Code API rate limits",
    "output_preview": "Rate limits are enforced per workspace..."
  }
]
```

**Narrative:** "Every tool call — bash, file reads, web searches — is captured with its input and output preview, stored in activity_logs.tool_trace. Capped at 200 entries to prevent runaway loops."

---

### Step 4: Query Tool Usage Across All Agents

```bash
# Find all agents that called 'bash' — fast GIN-indexed query
curl -s "https://your-deployment.moleculesai.app/activity-logs?tool=bash" \
  -H "Authorization: Bearer $ORG_API_KEY" | jq '.total'
```

**Narrative:** "The GIN index on tool_trace makes this query fast — operators can audit tool usage across the entire platform in seconds."

---

## Screencast Outline (~60s)

| Time | Action |
|------|--------|
| 0–10s | Run POST /instructions curl → show JSON response |
| 10–25s | Run resolve endpoint → show agent's startup payload |
| 25–45s | Run activity-logs query → show tool_trace entries |
| 45–55s | Run tool=bash filter → show cross-agent audit query |
| 55–60s | Point to Canvas UI if activity log viewer is available |

---

## Files

- Demo script: `docs/marketing/devrel/demos/tool-trace-platform-instructions-demo.sh`
- Architecture: `workspace-server/internal/handlers/instructions.go`, `workspace/a2a_executor.py`
- Migrations: `039_tool_trace.jsonb`, `040_platform_instructions`
