# Phase 34 — Tool Trace Social Copy
**Campaign:** Phase 34 GA (April 30, 2026)
**Owner:** Community Manager (draft) → Social Media Brand (publish)
**Date:** 2026-04-23

---

## Thread overview

5-post thread. Hook → what it is → how it works → why it matters → CTA.

---

## Post 1 — Hook

🐛 Your agent just did *something* — and now you have to figure out what.

No logs. No trace. Just a final output with no explanation.

That's the gap Tool Trace fills.

---

## Post 2 — What it is

Every A2A response from Molecule AI now includes a full execution record:

```
tool: "web_search"
input: {"query": "molecule ai pricing"}
output_preview: "Molecule AI offers a free tier with 10k tokens..."

tool: "write_file"
input: {"path": "summary.md"}
output_preview: "File written (412 bytes)"
```

No SDK. No sidecar. No sampling. It's in the response envelope.

---

## Post 3 — How it works

Tool Trace records every tool call — built-in or custom MCP — with:
- Tool name + input params
- Output preview (~200 chars)
- `run_id` pairing for parallel calls
- 200-entry cap (prevents runaway loops)

Data goes to `activity_logs` as JSONB — queryable, auditable, persistent.

---

## Post 4 — Why it matters

When something goes wrong in a multi-agent workflow, you used to have no visibility into *what the agent actually did* — just what it returned.

Tool Trace gives you the full tool call history.

Debug production incidents without guessing.

---

## Post 5 — CTA

Tool Trace is live now on every Molecule AI workspace. Zero config. No tier restrictions.

Explore: docs.moleculesai.app/blog/ai-agent-observability-without-overhead

---

**Delivery instructions for Social Media Brand:**
- Post 1-2 directly, 3-5 reply-chain under Post 1
- Tweet deck format: Thread by @moleculeai
- Alt text for any card images: "Tool Trace execution record example showing tool name, input params, and output preview"
- No design partner names
- Link check: confirm `docs.moleculesai.app/blog/ai-agent-observability-without-overhead` resolves before posting