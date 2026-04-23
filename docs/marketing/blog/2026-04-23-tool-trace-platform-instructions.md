---
title: "Agent Observability Built In: Tool Trace + Platform Instructions"
slug: agent-observability-tool-trace-platform-instructions
date: 2026-04-23
authors: [molecule-ai]
tags: [platform, observability, governance, phase-34]
description: "Molecule AI now records every tool call your agents make — name, input, output preview — with zero SDK setup. Plus org-level Platform Instructions."
og_image: /assets/blog/2026-04-23-tool-trace/og.png
---

# Agent Observability Built In: Tool Trace + Platform Instructions

You can now see exactly what your agents did — every tool call, every input, every output preview — without wiring up a third-party observability pipeline.

Phase 34 ships two platform-level features that answer the two questions every production agent team eventually asks: *What did the agent actually do?* And *what should it be allowed to do?*

**Tool Trace** answers the first. **Platform Instructions** answers the second.

---

## Tool Trace: execution record in every A2A response

Every A2A response from a Molecule AI agent now includes a `tool_trace` field in `Message.metadata`. It's a structured list of every tool the agent called during that task — what tool it used, what input it sent, and a preview of what came back.

```json
{
  "metadata": {
    "tool_trace": [
      {
        "tool_name": "web_search",
        "input": { "query": "molecule ai agent platform benchmarks" },
        "output_preview": "Molecule AI ranked #1 in agent coordination latency..."
      },
      {
        "tool_name": "write_file",
        "input": { "path": "research/benchmarks.md", "content": "..." },
        "output_preview": "File written successfully (2,847 bytes)"
      },
      {
        "tool_name": "bash",
        "input": { "command": "python analyze.py research/benchmarks.md" },
        "output_preview": "Analysis complete. 3 insights extracted."
      }
    ]
  }
}
```

No extra API calls. No SDK to install. No separate observability pipeline to configure. The trace is in the response, every time, for every agent on every plan.

### Parallel tool calls and run_id pairing

Agents that call tools in parallel — firing multiple MCP tools concurrently — are handled correctly. Each tool call includes a `run_id` that pairs the start event with its corresponding end event, so concurrent calls don't get interleaved in the trace.

```json
{
  "tool_name": "grep",
  "input": { "pattern": "TODO", "path": "src/" },
  "output_preview": "47 matches found across 12 files",
  "run_id": "a3f9b2c1"
}
```

### Stored and queryable

The full trace is persisted to `activity_logs.tool_trace` (JSONB column). You can query it, export it, or build audit tooling on top of it. The trace is capped at 200 entries per response to prevent runaway loops from bloating your logs.

### Why this matters for production

When something goes wrong in a multi-agent workflow, the question is always the same: *what did the agent actually do?* 

Most platforms give you the output. Molecule AI now gives you the trace. For teams running agents against code repositories, data pipelines, external APIs, or customer data — that trace is the difference between a five-minute diagnosis and a two-hour investigation.

---

## Platform Instructions: system prompt for your whole org

Platform Instructions lets workspace admins configure system-level instructions that apply across every agent in the org — set once via API, enforced before every agent turn.

```http
PUT /cp/platform-instructions
Authorization: Bearer mol_ws_your_token

{
  "instructions": "Always respond in English. Tag every response with the originating workspace ID. Do not execute destructive operations (DELETE, DROP, rm -rf) without explicit confirmation."
}
```

Every agent in your org inherits these instructions at startup. No touching individual workspace configs. No redeployment. The rule is part of what the agent is instructed to do from the first token — not a filter applied after.

### Global and workspace-scoped

Platform Instructions supports two scopes:

- **Global** (`PUT /cp/platform-instructions`): applies to every workspace in the org
- **Workspace-scoped** (via workspace config): per-team or per-project overrides on top of the global baseline

This lets you set org-wide compliance rules at the global level, then allow individual teams to add their own context on top.

### The governance use case

Policy-as-code tools like OPA or Sentinel enforce runtime *resource access* — what the agent can call, what APIs it can hit. Platform Instructions enforces *behavioral guardrails* — what the agent is instructed to do before it reasons about anything.

They're complementary. Platform Instructions is earlier in the chain: the rule is part of the system prompt, not a check applied after the agent has already decided what to do.

For compliance teams, this architecture matters. A behavioral rule that lives in the system prompt has no lag between "policy updated" and "policy in effect" — the next agent turn runs with the new rule. No deployment cycle required.

---

## Tool Trace + Platform Instructions together

The two features form a complete observability and governance loop:

**Platform Instructions** sets what your agents know and are instructed to do going in.  
**Tool Trace** proves what they actually did coming out.

```
[Platform Instructions] → agent turn → [Tool Trace]
  "don't run destructive ops"    "bash: rm -rf → blocked, 0 bytes deleted"
  "tag responses with workspace ID"  "write_file: tagged ✓"
```

Write the policy once. Enforce it everywhere. Trace every execution.

For platform teams managing agent fleets at scale — especially in compliance-sensitive environments — this is the observability and governance stack that was previously only available by integrating third-party tooling. It now ships as part of the platform.

---

## Getting started

**Tool Trace** requires no configuration — it's in every A2A response today. Check `message.metadata.tool_trace` in your next agent run.

**Platform Instructions** is available via the API:

```bash
# Set org-wide instructions
curl -X PUT https://api.molecule.ai/cp/platform-instructions \
  -H "Authorization: Bearer $MOL_WS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"instructions": "Your org-wide instructions here."}'

# Read current instructions
curl https://api.molecule.ai/cp/platform-instructions \
  -H "Authorization: Bearer $MOL_WS_TOKEN"
```

Both features are live as part of Phase 34. Partner API Keys (`mol_pk_*`) — the programmatic org provisioning API — reaches GA on April 30.

→ [Docs: Tool Trace](https://docs.molecule.ai/platform/tool-trace)  
→ [Docs: Platform Instructions](https://docs.molecule.ai/platform/platform-instructions)  
→ [Phase 34 release notes](https://docs.molecule.ai/changelog/phase-34)

---

*Phase 34 also includes Partner API Keys (GA April 30) and SaaS Fed v2. See the [full Phase 34 announcement](https://docs.molecule.ai/blog/phase-34-community-announcement) for the complete picture.*
