---
title: "See Exactly What Your AI Agent Did: Tool Trace is Live"
date: 2026-04-23
slug: ai-agent-tool-trace
description: "Tool Trace gives every A2A response a structured record of every tool call — inputs, output previews, run_id-paired parallel traces. Built in. Not bolt-on."
og_title: "See Exactly What Your AI Agent Did: Tool Trace is Live"
og_description: "Every tool call, every input, every output — in every A2A response metadata. Tool Trace is live. No sidecar, no sampling, no guesswork."
og_image: /docs/assets/blog/2026-04-23-tool-trace-observability-og.png
tags: [observability, tool-trace, debugging, devops, platform-engineering, a2a]
keywords: [AI agent observability, tool trace debugging, Claude agent debugging, agent audit trail, parallel tool call trace, run_id pairing, AI agent monitoring]
canonical: https://docs.molecule.ai/blog/ai-agent-tool-trace
---

<script type="application/ld+json">
{
  "@context": "https://schema.org",
  "@type": "Article",
  "headline": "See Exactly What Your AI Agent Did: Tool Trace is Live",
  "description": "Tool Trace gives every A2A response a structured record of every tool call — inputs, output previews, run_id-paired parallel traces.",
  "author": { "@type": "Organization", "name": "Molecule AI" },
  "datePublished": "2026-04-23",
  "publisher": { "@type": "Organization", "name": "Molecule AI", "logo": { "@type": "ImageObject", "url": "https://molecule.ai/logo.png" } }
}
</script>

# See Exactly What Your AI Agent Did: Tool Trace is Live

When your AI agent runs in production, most platforms give you the final output. You don't get the tool calls that produced it — the Write that saved the file, the Bash command that ran the tests, the Grep search that found the bug. When something breaks, you're debugging blind, working backward from the symptom instead of forward from the cause.

Tool Trace ends that. Every A2A response from Molecule AI now includes a structured tool_trace array in Message.metadata — a complete, chronological record of every tool the agent called, the inputs it passed, and a preview of what it returned. No sampling. No sidecar collector. No guesswork. Just the data you need to see exactly what your agent did.

## What Tool Trace Captures

Each trace entry contains four fields:

- **tool** — the tool name (Write, Bash, Grep, Read, Edit, etc.)
- **input** — the exact parameters passed to the call
- **output_preview** — the first ~200 characters of the result, keeping traces readable at scale
- **run_id** — links start and end events so concurrent calls are traced independently, not collapsed into a single merged line

Entries are stored in activity_logs.tool_trace as JSONB, so you can query historical traces directly in your existing log infrastructure.

```json
{
  "metadata": {
    "tool_trace": [
      {
        "tool": "Bash",
        "input": {
          "command": "go build ./... && go test ./...",
          "description": "Build and test full Go project"
        },
        "output_preview": "ok      auth    0.314s\nok      config  0.201s\nok      server  0.487s\n--- PASS: TestIntegration (12.3s)"
      },
      {
        "tool": "Write",
        "input": {
          "file_path": "/workspace/coverage/report.json",
          "content": "{\"total\": 94.2, \"files\": {...}}"
        },
        "output_preview": "Wrote 2.1 KB to /workspace/coverage/report.json"
      },
      {
        "tool": "Read",
        "input": {
          "file_path": "/workspace/coverage/report.json"
        },
        "output_preview": "Read 2.1 KB from /workspace/coverage/report.json"
      }
    ],
    "run_id": "01HXKM3T8PRQN4ZW7XYVD2EJ5A"
  }
}
```

The trace is capped at 200 entries per response — more than enough for most turns, and a hard ceiling that keeps payload sizes predictable at scale.

## Parallel Calls: Traced Correctly, Not Merged Away

The hardest part of agent observability isn't capturing one tool call — it's capturing several that fired at the same time without losing track of which did what.

Tool Trace handles this with run_id pairing. When the agent fires Bash and Write concurrently in a single turn, each entry carries the same run_id, with start and end events matched by that identifier. The result is an unambiguous, independent trace per concurrent call — not a merged log line that tells you "something ran" without telling you which call returned what.

This is the difference between "the agent called two tools and one failed" and "Bash called go build and returned exit code 1 because auth_test.go had a governance assertion that fired first."

## Built In, Not Bolt-On

Most observability solutions for AI agents require instrumentation — a tracing SDK, a sidecar collector, a log aggregation pipeline you have to operate. Tool Trace ships in the A2A response itself. If you're already receiving A2A responses from your agent, you already have the trace. No new infrastructure. No sampling configuration. No agent restart.

For platform engineering teams managing a fleet of agents, Tool Trace gives you the observability signal you need — which tools are being called, which inputs are being passed, which outputs are being produced — without adding operational overhead.

## Enterprise Debugging: From Org Key to Tool Call

Combined with the org-scoped API key audit trail from Phase 30, Tool Trace closes the final gap in agent observability. You can trace a production incident from the org API key that authorized the call, through the workspace and agent that executed it, to every tool that ran and what it returned. That level of detail turns "something went wrong" into "Bash called git push with a stale credential and got permission denied, triggered by a delegation from ci-agent using org key mole_a1b2 at 14:23 UTC."

Tool Trace is available on all Molecule AI plans. It is enabled by default — inspect Message.metadata.tool_trace in any A2A response right now.

## Get Started

- Inspect Message.metadata.tool_trace in any A2A response
- Query activity_logs.tool_trace JSONB for historical traces
- Combine with org API key attribution for complete fleet observability
- Read the A2A protocol documentation

---

*Molecule AI is open source. Tool Trace shipped in Phase 34 (2026-04-23).*
