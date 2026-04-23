---
title: "Tool Trace + Platform Instructions: See Every Tool Call. Govern Every Agent."
date: 2026-04-23
slug: tool-trace-platform-instructions
description: "Tool Trace surfaces every tool call in A2A responses — inputs, output previews, run_id-paired parallel traces. Platform Instructions lets you enforce org-wide governance at the system prompt level. Two enterprise-grade features, shipped together."
og_title: "Tool Trace + Platform Instructions: See Every Tool Call. Govern Every Agent."
og_description: "Full tool-call observability in every A2A response. Org-wide governance at the system prompt level. Two enterprise features, one release."
tags: [tool-trace, observability, platform-instructions, governance, enterprise, debugging, a2a, phase-34]
keywords: [AI agent debugging, tool trace observability, agent governance, platform instructions, enterprise AI audit, system prompt governance, agent observability]
canonical: https://docs.molecule.ai/blog/tool-trace-platform-instructions
tts_audio: /docs/devrel/demos/tool-trace-platform-instructions/intro-narration.mp3
tts_script: /docs/devrel/demos/tool-trace-platform-instructions/narration.txt
---

<script type="application/ld+json">
{
  "@context": "https://schema.org",
  "@type": "TechArticle",
  "headline": "Tool Trace + Platform Instructions: See Every Tool Call. Govern Every Agent.",
  "description": "Tool Trace surfaces every tool call in A2A responses. Platform Instructions enforces org-wide governance at the system prompt level. Two enterprise features, one release.",
  "datePublished": "2026-04-23",
  "author": { "@type": "Organization", "name": "Molecule AI", "url": "https://molecule.ai" },
  "publisher": {
    "@type": "Organization",
    "name": "Molecule AI",
    "logo": { "@type": "ImageObject", "url": "https://molecule.ai/logo.png" }
  },
  "about": {
    "@type": "Thing",
    "name": "AI Agent Observability and Governance",
    "description": "Tool Trace provides full visibility into every tool call an AI agent makes. Platform Instructions enforces governance rules at the system prompt level across an entire organization."
  },
  "proficiencyLevel": "Intermediate",
  "genre": ["technical documentation", "product announcement"]
}
</script>

# Tool Trace + Platform Instructions: See Every Tool Call. Govern Every Agent.

When an AI agent runs in production, most platforms tell you one thing: what the agent said. They don't tell you *what it actually did* — which tools it called, with what inputs, what they returned, and whether a parallel call happened at the same time. That's the gap where debugging ends and guessing begins.

Today Molecule AI ships two features that close it. **Tool Trace** and **Platform Instructions** ship together in Phase 34, and they're designed to answer the two questions enterprise teams ask first when evaluating an AI agent platform: *what did the agent actually do?* and *can we enforce policy at the platform level?*

---

## Tool Trace: Every Tool Call, Exactly What It Did

Tool Trace embeds a structured record of every tool call in every A2A response. It's in `Message.metadata.tool_trace` — no sampling, no sidecar collector, no separate log aggregator to configure. It's there by default on every A2A call.

Each entry records:

- **`tool`** — the name of the tool invoked
- **`input`** — the arguments passed to it (structurally present, not redacted)
- **`output_preview`** — the first ~200 characters of the result, so the trace stays readable
- **`run_id`** — a UUID that pairs the start and end of the same tool call

The `run_id` field handles the parallel tool call problem correctly. When an agent fires multiple tool calls simultaneously in a single turn — say, a `Grep` across three files, or a `Bash` command alongside a `Write` — each entry carries the same `run_id`. You can trace them independently rather than seeing them collapsed into a single ambiguous log line.

The trace is capped at 200 entries per response to prevent runaway loops from bloating payloads.

Here's what it looks like in practice:

```json
{
  "jsonrpc": "2.0",
  "result": {
    "message": {
      "role": "agent",
      "parts": [{ "type": "text", "text": "Build complete. All tests passing." }],
      "metadata": {
        "tool_trace": [
          {
            "tool": "Grep",
            "input": {
              "pattern": "TODO.*governance",
              "glob": "**/*.go",
              "path": "/workspace/src",
              "output_mode": "content"
            },
            "output_preview": "auth.go:14 // TODO: governance layer check\nauth_test.go:8 // TODO: add governance test",
            "run_id": "01HXKM3TWQP7RVZN4G9E6J8BF"
          },
          {
            "tool": "Bash",
            "input": {
              "command": "go build ./...",
              "description": "Verify package compiles after auth changes"
            },
            "output_preview": "Build succeeded. 0 errors, 0 warnings.",
            "run_id": "01HXKM3TWQP7RVZN4G9E6J8BF"
          },
          {
            "tool": "Write",
            "input": {
              "file_path": "/workspace/src/auth_test.go",
              "content": "package auth\n\nfunc TestGovernance... [truncated]"
            },
            "output_preview": "Wrote 1,204 bytes to /workspace/src/auth_test.go",
            "run_id": "01HXKM3TWQP7RVZN4G9E6J8BF"
          }
        ],
        "run_id": "01HXKM3TWQP7RVZN4G9E6J8BF"
      }
    }
  }
}
```

In this example, the agent ran a `Grep` to find governance TODOs, verified the build compiled with `Bash`, then wrote a test file. Tool Trace captured all three — with their actual inputs and output previews — as a single structured unit.

### Querying tool traces

Tool traces are stored in `activity_logs.tool_trace` as a JSONB column. You can query them directly:

```sql
SELECT
  workspace_id,
  created_at,
  tool_trace
FROM activity_logs
WHERE tool_trace IS NOT NULL
  AND created_at > NOW() - INTERVAL '1 hour'
ORDER BY created_at DESC
LIMIT 50;
```

This gives you a live window into what your agents are actually doing — not what they reported doing, but what the tools recorded.

### What Tool Trace is good for

- **Debugging** — pinpoint which tool failed in a parallel call without guessing
- **Compliance auditing** — reconstruct the complete tool chain for any A2A session
- **Prompt engineering** — see what tools your agent is actually reaching for, vs. what you expected
- **Regression detection** — flag when an agent starts calling unexpected tools

---

## Platform Instructions: Governance at the Source

Tool Trace tells you what the agent *did*. Platform Instructions let you define what it *should* do — before it does it.

Platform Instructions are configurable rules with two scopes:

- **Global** — applied to every workspace in your organization automatically
- **Workspace** — applied to a specific workspace only

They are injected into the agent's system prompt at workspace startup, appearing before all other content. Because they go first, they have highest precedence in the prompt. Agents receive these instructions at boot and on every periodic refresh — they cannot be overridden by a downstream prompt.

A simple CRUD API manages the full lifecycle:

```bash
# Create a global instruction — applies to every workspace in your org
curl -X POST https://platform.molecule.ai/cp/platform-instructions \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "All agents must redact PII before writing to any external tool. "
               "Do not write raw email addresses, phone numbers, or national IDs "
               "to any file or API endpoint.",
    "scope": "global"
  }'

# Create a workspace-scoped instruction
curl -X POST https://platform.molecule.ai/cp/platform-instructions \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Default to read-only mode for new integrations. "
               "Require explicit user confirmation before any write operation.",
    "scope": "workspace",
    "workspace_id": "ws_01HXKM3TWQP7RVZN4G9E6J8BF"
  }'

# List all instructions (admin only)
curl https://platform.molecule.ai/cp/platform-instructions \
  -H "Authorization: Bearer <admin-token>"

# Update an instruction
curl -X PUT https://platform.molecule.ai/cp/platform-instructions/pi_01HXK \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{ "content": "Updated: PII redaction now includes credit card numbers." }'

# Delete
curl -X DELETE https://platform.molecule.ai/cp/platform-instructions/pi_01HXK \
  -H "Authorization: Bearer <admin-token>"
```

When a workspace boots, it fetches its resolved instructions via the resolve endpoint. The call is gated by `wsAuth` — the workspace's own token — so there is no cross-workspace enumeration. A workspace can only retrieve its own instructions. The content cap is 8KB per instruction, and a short timeout on the resolve call means a platform outage never blocks agent startup.

### A governance pattern worth naming

Platform Instructions are prepended to the system prompt, not applied as a post-hoc filter. This distinction matters. A filter evaluates after the agent has decided what to do. A system prompt instruction shapes the agent's reasoning *before* the first token is generated. A filter can be worked around. A system prompt instruction is part of what the agent is told to do.

For regulated industries, this means your security team defines data handling rules once — and they apply automatically, on every agent turn, without a code deploy.

---

## The Governance Loop: Write Policy. See Execution.

Tool Trace and Platform Instructions are designed to work together. The loop looks like this:

1. Platform team writes a Platform Instruction (e.g., "no raw PII to external tools")
2. Every workspace boots with that instruction prepended to the system prompt
3. Tool Trace records every tool call — with inputs and output previews — in `activity_logs`
4. Compliance team queries `activity_logs.tool_trace` to verify agents followed the policy
5. If something drifted, the trace is there to reconstruct exactly what happened

Paired with the org API key attribution from [Phase 30's per-workspace bearer tokens](/blog/remote-workspaces), you can trace the complete chain: *which org key, which workspace, which agent, which tools, in what order.*

---

## What's Available and When

| Feature | Availability | Where to find it |
|---------|--------------|-------------------|
| Tool Trace | All plans | `Message.metadata.tool_trace` in every A2A response |
| Tool Trace storage | All plans | `activity_logs.tool_trace` JSONB column |
| Platform Instructions | Enterprise plans only | `/cp/platform-instructions` API endpoints |

If you're on an Enterprise plan and don't see Platform Instructions in your admin settings, contact your account team or check the [Platform Instructions documentation](/docs/guides/platform-instructions).

---

## Get Started

- **Tool Trace** is enabled by default. Check `Message.metadata.tool_trace` in your next A2A response.
- **Platform Instructions** require an Enterprise plan. Once enabled, navigate to your org settings or use the API directly.
- Full protocol details are in the [A2A protocol documentation](/docs/api-protocol/a2a-protocol.md).
- Screencast walkthrough: [Tool Trace + Platform Instructions demo](/docs/devrel/demos/tool-trace-platform-instructions).

---

*Tool Trace and Platform Instructions shipped in Phase 34 (2026-04-23). Molecule AI is open source — contributions welcome.*
