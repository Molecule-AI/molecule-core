# Launch Brief: PR #1686 — Tool Trace + Platform Instructions

**PR:** #1686 `feat: tool trace + platform instructions`
**Merged:** 2026-04-23T02:43:27Z
**Status:** GA-ready (passed code review, IDOR and DoS fixes verified)
**Brief owner:** PMM

---

## Problem

A2A agentic workflows are opaque to platform teams. When an agent runs inside a workspace, operators have:
- **Zero observability** into which tools were called, what inputs were used, or what outputs were returned
- **No governance layer** — compliance requirements, cost controls, and security guardrails cannot be enforced at the platform level without modifying agent code

This blocks enterprise adoption: platform engineers need to audit agent behavior, and security/compliance teams need to enforce guardrails before agents touch production systems.

---

## Solution

Two independent features shipping in the same PR:

1. **Tool Trace** — Every A2A response metadata (`Message.metadata.tool_trace`) now includes a list of `{tool_name, input, output_preview}` entries. Pairs start/end events via `run_id` so parallel tool calls are correctly scoped. Capped at 200 entries to prevent runaway-loop bloat.

2. **Platform Instructions** — Workspace-scoped configuration rules injected into the system prompt at startup. Supports global and per-workspace scope. Includes a CRUD API and `/workspaces/:id/instructions/resolve` endpoint (IDOR-protected under `wsAuth`). CHECK constraints enforce an 8KB content cap to prevent token-budget DoS.

---

## 3 Claims

1. **Full tool-level visibility in every A2A response** — Platform teams can now see exactly what tools an agent called, with inputs and output previews, without adding instrumentation to the agent itself.

2. **Governance without agent code changes** — Platform Instructions let compliance and security teams enforce guardrails (compliance requirements, cost limits, security policies) at the workspace level, prepended to the agent's system prompt automatically at startup.

3. **Enterprise-ready by default** — IDOR vulnerability in the resolve endpoint fixed pre-merge; 8KB content cap prevents token-budget DoS; `run_id`-scoped parallel tool call pairing ensures correct attribution under concurrent execution.

---

## Target Developer

- **Primary:** Platform engineers and DevOps teams deploying A2A agentic workflows in production
- **Secondary:** Enterprise security/compliance teams requiring audit trails and governance controls before approving agent deployments

---

## CTA

Log into the workspace and enable Platform Instructions via the CRUD API, then deploy an A2A agent — the governance rules are automatically prepended to the system prompt at startup. Tool traces appear in `Message.metadata` on every A2A response with zero agent-side changes.
