# Phase 34 — Reddit Launch Post
**Campaign:** Phase 34 GA (April 30, 2026)
**Owner:** Community Manager
**Target:** r/MachineLearning
**Date:** 2026-04-23
**Status:** READY — post April 30 ~09:00 PT / 16:00 UTC

---

## Title options

1. "Molecule AI Phase 34: built-in agent execution tracing and governance — no SDK required" (recommended)
2. "How are you handling agent observability in production? We just shipped a platform-native option"
3. "Built agent execution tracing into the platform — no SDK, no sidecar, no sampling"

---

## Body

We've been thinking about a specific gap in agent frameworks: when something goes wrong in a multi-step agent task, you often have no record of *what the agent actually did* — only what it returned.

Phase 34 ships two features aimed at that gap.

**Tool Trace** is a structured execution record embedded in every A2A response. For each tool your agent calls, you get the tool name, input params, and a short output preview — in order, with parallel calls paired via `run_id`. It's in the response envelope. No SDK, no sidecar, no sampling.

```python
response = agent.send(task="deploy to staging")
for entry in response.metadata.tool_trace:
    print(f"{entry['tool']}: {entry['input']} → {entry['output_preview']}")
```

Stored in `activity_logs` as JSONB — queryable and auditable. 200-entry cap prevents runaway loops. Works with custom MCP tools and built-in tools identically.

**Platform Instructions** lets org admins set system-level rules that apply to every agent in the org — at startup, before the agent reads its own config. Think of it as a system prompt for your whole org. "Confirm before prod writes." "Tag all audit-sensitive operations." Rules are injected by the platform — workspace users can't remove them by editing config.yaml.

Both are platform-native, not plugins. They live in the workspace startup path and the A2A response envelope — nothing to install, no external vendor relationship.

**Partner API Keys** (`mol_pk_*`) are GA April 30 — programmatic org provisioning for CI/CD pipelines, marketplace integrations, internal tooling. Ephemeral test orgs per pipeline run, billed by the second, fully revocable.

Curious what others are using for agent observability today — Langfuse, Helicone, custom logging? What are the pain points?

---

## Posting notes

- **Timing:** April 30, ~09:00 PT / 16:00 UTC
- **Engagement SLA:** reply to top-level comments within 30 min
- **Do not name** any design partners
- **Partner API Keys:** frame as "ships April 30" — not available now
- **Tool Trace + Platform Instructions:** available now for all plans
- **SaaS Fed v2:** SaaS-only, affects multi-org operators — not relevant for most r/ML readers, skip unless asked

**CTA:** Discord: discord.gg/moleculeai · Docs: docs.moleculesai.app/blog/tool-trace-platform-instructions