# Phase 34 — HackerNews Show HN Post
**Campaign:** Phase 34 GA (April 30, 2026)
**Owner:** Community Manager
**Date:** 2026-04-23
**Status:** READY — post April 30 ~09:00 PT / 16:00 UTC

---

## Title

Show HN: Molecule AI — agent platform with built-in execution tracing in every A2A response

**Alternatives:**
1. "Show HN: Built execution tracing into the A2A protocol — no SDK required"
2. "Show HN: Molecule AI Phase 34 — platform-native agent observability + org governance"

---

## Body

Hey HN — Molecule AI (YC W23). Phase 34 ships two features for teams running agents in production.

**Tool Trace** — every A2A response includes a full execution record:

```python
response = agent.send(task="deploy to staging")
for entry in response.metadata.tool_trace:
    print(f"{entry['tool']}: {entry['input']} → {entry['output_preview']}")
```

No SDK. No sidecar. No sampling. It's in the response envelope by default.

The trace records: tool name, input params, output preview (~200 chars), run_id pairing for parallel calls. Capped at 200 entries per task. Stored as JSONB in `activity_logs`. Works with custom MCP tools and built-in tools.

**Platform Instructions** — org admins set system-level rules that apply to every agent at startup. "Never commit directly to main." "Confirm before prod writes." "Tag all audit-sensitive ops." Rules are injected before the agent reads its own config — workspace users can't override them by editing config.yaml.

Both are platform-native. No external SDK, no plugin, no vendor dependency.

**Partner API Keys** (`mol_pk_*`) GA April 30: programmatic org provisioning for CI/CD, marketplaces, internal tooling. Ephemeral test orgs per pipeline run — spin up, test, teardown. Billing stops when you delete.

Docs: docs.moleculesai.app
GitHub: github.com/Molecule-AI/molecule-core

---

## Honest about what's not done

- Audit trail panel (Canvas UI for traces): in progress, not shipped yet. Data queryable via API.
- SaaS Fed v2: SaaS-only. Self-hosted users unaffected.
- Partner API Keys: not live until April 30.

---

## Anticipated HN objections — FAQ

**"How is this different from Langfuse/Helicone?"**
Tool Trace captures A2A-level agent behavior — tool calls, inputs, output previews. Langfuse/Helicone capture LLM API calls. Different layers. If you're running agents on Molecule, Tool Trace is zero-config and free. If you need cross-platform multi-model observability, Langfuse is still a great complement.

**"Is this open source?"**
Workspace-server is open source at github.com/Molecule-AI/molecule-core. The SaaS control plane (orgs, billing, provisioning) is not.

**"What's the pricing?"**
Tool Trace: all plans, no tier restriction. Platform Instructions: all plans. Partner API Keys: GA April 30 — see docs for rate limits.

**"Why ship four things at once?"**
They're built to work together. Partner API Keys provisions orgs. Platform Instructions governs them. Tool Trace shows what agents did. SaaS Fed v2 keeps it isolated. Shipping together means you get a coherent system from day one, not four features with interdependency debt.

---

## Posting notes

- **Timing:** April 30, ~09:00 PT / 16:00 UTC
- **First reply (pinned):** post the tool_trace code snippet with a one-line intro
- **Monitor:** 3h, reply to every top-level comment within 30 min
- **Do not name** any design partners
- **LangGraph/CrewAI comparisons:** frame as additive, different layer — not competitive
- **Pricing questions:** redirect to docs.moleculesai.app or "contact us"