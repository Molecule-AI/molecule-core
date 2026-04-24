# Tool Trace Demo — PR #1686

> **Source:** `molecule-core` PR #1686 (`feat: tool trace + platform instructions`)
> **API:** `GET /workspaces/:id/activity` → `tool_trace[]` in each log entry
> **Storage:** `activity_logs.tool_trace` JSONB column + GIN index
> **Prerequisite:** A running Molecule AI platform with at least one active workspace

---

## What This Demo Shows

1. **How tool_trace appears in an A2A response** — the `metadata.tool_trace` list your integration receives when the agent returns
2. **How to read historical tool traces** — query `GET /workspaces/:id/activity` for past agent runs
3. **How run_id pairs parallel tool calls** — LangGraph agents can call multiple tools concurrently; `run_id` ensures each `on_tool_end` event pairs with the right `on_tool_start`
4. **Agent Activity Report** — a clean printed output showing each tool, inputs, and outputs

---

## Files

```
docs/marketing/devrel/demos/tool-trace-demo/
├── demo.py          — Runnable Python script (works offline with simulated data)
├── README.md        — This file
└── narration.txt    — TTS narration script (~60s)
```

---

## Quick Start

### Option A: Run with simulated data (no platform required)

```bash
cd docs/marketing/devrel/demos/tool-trace-demo
pip install requests  # a2a SDK optional — demo uses requests
python demo.py
```

Output:
```
[DEMO 1] Reading tool_trace from an A2A response metadata object
════════════════════════════════════════════════════════════
  ▸ web_search
    Input   : {"query": "Molecule AI agent platform observability", ...}
    Output  : [{'title': 'A2A Protocol Spec', 'url': 'https://a2a.chat', ...}]

  ▸ summarize_text
    Input   : {"text": "[full search results...", "max_bullets": 3}
    Output  : • A2A enables direct workspace-to-workspace communication...

  ▸ write_to_file
    Input   : {"path": "/tmp/agent-report.md", ...}
    Output  : File written: /tmp/agent-report.md (847 bytes)
```

### Option B: Run against a live platform

```bash
export PLATFORM_URL=https://your-deployment.moleculesai.app
export WORKSPACE_TOKEN=your-workspace-token
python demo.py
```

The script will query `GET /workspaces/:id/activity` for the last 3 log entries and print an activity report for each.

---

## Key Design Decisions

### Event-based collection

The tool trace is built from two LangGraph streaming events:

| Event | Captures | Limit |
|---|---|---|
| `on_tool_start` | `tool` name + `input` (first 500 chars) | 200 entries max |
| `on_tool_end` | `output_preview` (first 300 chars) | — |

The platform pairs start → end events via `run_id`. If `run_id` is empty, the tools ran sequentially and the most recent `on_tool_end` updates the last entry.

### 200-entry cap

`MAX_TOOL_TRACE = 200` in `a2a_executor.py` prevents runaway agent loops from generating unbounded JSONB payloads. The cap is enforced in Python — the database column stores whatever the runtime writes.

### Where it's stored

```sql
-- Migration: 039_activity_tool_trace.up.sql
ALTER TABLE activity_logs ADD COLUMN tool_trace JSONB;
CREATE INDEX activity_logs_tool_trace ON activity_logs USING GIN (tool_trace);
```

Query:
```bash
curl -s "$PLATFORM_URL/workspaces/$WORKSPACE_ID/activity?limit=5" \
  -H "Authorization: Bearer $WORKSPACE_TOKEN" \
  | jq '.[].tool_trace[:3]'
```

### Metadata path

In the A2A response your integration receives:
```json
{
  "result": { ... },
  "metadata": {
    "tool_trace": [
      {"tool": "bash", "input": "...", "output_preview": "..."},
      ...
    ]
  }
}
```

---

## Demo Scenarios

### 1 — Sequential tool calls (web search → summarize → write)

```
on_tool_start → {"tool": "web_search", "input": "...", "run_id": "run-001"}
on_tool_end   → {"tool": "web_search", "output_preview": "..."}  ← pairs by run_id
on_tool_start → {"tool": "summarize_text", "input": "...", "run_id": "run-002"}
on_tool_end   → {"tool": "summarize_text", "output_preview": "..."}
on_tool_start → {"tool": "write_to_file", "input": "...", "run_id": "run-003"}
on_tool_end   → {"tool": "write_to_file", "output_preview": "File written..."}
```

Output:
```
  ▸ web_search
    Input   : {"query": "Molecule AI tool trace docs", "top_k": 5}
    Output  : [{'title': 'A2A Protocol Spec', 'url': 'https://a2a.chat', ...}]

  ▸ summarize_text
    Input   : {"text": "...[search results]...", "max_bullets": 3}
    Output  : • A2A enables direct workspace-to-workspace communication...

  ▸ write_to_file
    Input   : {"path": "/tmp/agent-report.md", ...}
    Output  : File written: /tmp/agent-report.md (847 bytes)
```

### 2 — Parallel tool calls (LangGraph concurrent invocation)

```
on_tool_start → {"tool": "web_search",  "run_id": "run-parallel-a"}
on_tool_start → {"tool": "read_file",   "run_id": "run-parallel-b"}
on_tool_end   → {"tool": "web_search",  "output_preview": "found 8 results in 142ms",  "run_id": "run-parallel-a"}
on_tool_end   → {"tool": "read_file",   "output_preview": "database_url: postgresql://...", "run_id": "run-parallel-b"}
```

The `AgentActivityReport` class groups by `run_id` and labels parallel groups:
```
  ┌─ Parallel call group [run-paral...]
  │  ▸ web_search
  │    Input   : {"query": "Molecule AI tool trace docs"}
  │    Output  : [found 8 results in 142ms]
  │  ▸ read_file
  │    Input   : {"path": "/workspace/config.yaml"}
  │    Output  : database_url: postgresql://...\nmodel: claude-sonnet-4
  └─
```

### 3 — Query activity logs for audit

```python
from demo import MoleculeAIClient, AgentActivityReport

client = MoleculeAIClient(PLATFORM_URL, WORKSPACE_TOKEN)
logs = client.get_activity_logs("ws-abc123", limit=10)

for entry in logs:
    if entry.get("tool_trace"):
        report = AgentActivityReport.from_activity_log("ws-abc123", entry)
        report.print_report(title=f"Log {entry['id']}")
        # Export to SIEM:
        print(json.dumps(report.as_dict()))
```

---

## Integrating Into Your Code

### Import the report class

```python
from demo import AgentActivityReport

# From A2A response
resp = your_a2a_client.send_task(task="...")
metadata = resp.get("metadata", {})
report = AgentActivityReport.from_response("ws-abc", metadata)
report.print_report()  # human-readable

# Export for audit
audit_event = report.as_dict()
# → send to your SIEM / audit store / webhook
```

### Customize the output

Subclass `AgentActivityReport` and override `_print_tool_call()` for different formatting (JSON, CSV, Slack messages, etc.).

---

## Reference

| Component | File |
|---|---|
| Tool trace collection | `workspace/a2a_executor.py` (`MAX_TOOL_TRACE`, `on_tool_start`/`on_tool_end` events) |
| Metadata attachment | `workspace/a2a_executor.py` (`msg.metadata = {"tool_trace": tool_trace}`) |
| DB schema | `workspace-server/migrations/039_activity_tool_trace.up.sql` |
| Activity handler | `workspace-server/internal/handlers/activity.go` |
| API endpoint | `GET /workspaces/:id/activity` |

---

## Status

| Check | Pass? |
|---|---|
| Simulated demo runs | ✅ |
| Live platform code provided | ✅ (env vars required) |
| Parallel calls with run_id | ✅ |
| JSON export for SIEM | ✅ |
| README walkthrough | ✅ |
| TTS narration script | ✅ (`narration.txt`) |

---

*Demo: `docs/marketing/devrel/demos/tool-trace-demo/` | Source: `molecule-core` PR #1686*