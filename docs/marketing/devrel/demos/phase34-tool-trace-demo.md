# Phase 34 DevRel Demo Brief: Tool Trace
**Date:** 2026-04-24
**Owner:** DevRel Engineer
**Priority:** High — publish alongside Apr 28 social post
**Source:** PR #1686, blog post `docs/marketing/blog/2026-04-23-tool-trace-observability.md`
**Status:** Brief ready — awaiting DevRel execution

---

## Demo Goal

Show a developer how to inspect `tool_trace` in an A2A response — zero setup, works by default. The "aha moment" is seeing the execution tree of a parallel tool call appear automatically in the response metadata.

---

## Target Audience

Platform engineers, DevOps / MLOps, enterprise developers who need production observability for AI agents.

---

## Demo Script (3–5 minutes)

### Step 1: Context (30 seconds)
> "Before Tool Trace, debugging an AI agent meant reconstructing what happened from its output. You'd ask it what tools it ran — and hope its self-report was accurate. Tool Trace solves this. Here's how."

### Step 2: Send a task that triggers multiple tool calls (~1 minute)

```python
import os
from molecule_core import MoleculeClient

client = MoleculeClient(
    org="acme-corp",
    token=os.environ["MOLECULE_ORG_TOKEN"]
)

response = client.messages.send(
    workspace_id="ws-abc123",
    text="Find all documentation mentioning API keys and give me a summary of each file."
)
```

> "I'm asking the agent to search for files AND read multiple of them. That means parallel tool calls."

### Step 3: Inspect the trace (~2 minutes)

```python
last_message = response.messages[-1]
trace = last_message.metadata.get("tool_trace", [])

print(f"Tools called: {len(trace)}")
for entry in trace:
    print(f"\n[{entry['tool_name']}]")
    print(f"  Input:   {entry['input']}")
    print(f"  Output:  {entry['output_preview']}")
    if 'run_id' in entry:
        print(f"  run_id:  {entry['run_id']}")
```

**Expected output:**
```
Tools called: 4

[mcp-code-search]
  Input:   {'query': 'API keys documentation'}
  Output:  found 3 files in 0.12s
  run_id:  run-001

[mcp-file-read]
  Input:   {'path': 'docs/api-keys.md'}
  Output:  ...64 lines, 2 code blocks...
  run_id:  run-002

[mcp-file-read]
  Input:   {'path': 'docs/org-tokens.md'}
  Output:  ...41 lines, 1 code block...
  run_id:  run-003

[mcp-file-read]
  Input:   {'path': 'docs/partner-keys.md'}
  Output:  ...28 lines...
  run_id:  run-004
```

> "Each tool call has a `run_id` linking its start and end events. When calls overlap in time — parallel execution — you can still reconstruct the full execution tree from the run IDs. No guesswork."

### Step 4: Show the "no config" angle (30 seconds)
> "This required zero changes to my code. No SDK plugin, no sidecar agent, no sampling flag. The trace is on every response by default."

### Step 5: Point to persistence (~30 seconds)
> "And it's not just in-memory. The trace persists to `activity_logs.tool_trace` in the platform database — queryable via the standard activity log API. Post-incident review, compliance audits, cost attribution — all available, all in one place."

---

## Key Demo Points (must land)

1. **Zero config** — works by default, no opt-in required
2. **Parallel call support** — `run_id` pairing is the differentiator vs. flat logging
3. **Persistent** — not ephemeral; lives in `activity_logs`
4. **Production use cases** — post-incident review, compliance, SLA verification

---

## Talking Points (avoid)

- Do NOT claim this replaces Langfuse/Datadog for multi-source observability — Tool Trace is A2A-level only
- Do NOT say "GA" — use "live now / beta"
- Do NOT show the raw JSONB column query — too much implementation detail for a demo

---

## Assets Needed

- [ ] Screencast / GIF: the 3-step code flow above (send → inspect trace → show run_id pairing)
- [ ] Optional: split-terminal view showing two tool calls overlapping in time, then the run_id linking them

---

## Publish Checklist

- [ ] Code sample tested against current platform build
- [ ] Screencast < 90 seconds for X post embed
- [ ] Blog post link: `docs/marketing/blog/2026-04-23-tool-trace-observability.md`
- [ ] Social copy aligned: `docs/marketing/social/2026-04-28-tool-trace/social-copy.md`
