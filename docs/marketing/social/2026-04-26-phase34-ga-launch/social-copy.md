# Phase 34 GA Launch — Social Copy
**Campaign:** Phase 34 GA | **Features:** Tool Trace + Platform Instructions
**Publish day:** 2026-04-26 (Day 6 of Phase 30 social campaign)
**Status:** APPROVED — Marketing Lead 2026-04-23. Plan gating corrected: Platform Instructions is all-plans (verified instructions.go). Ready for Social Brand to publish Apr 26.
**Source:** PRs #1686 + #1824 + blog posts `docs/blog/2026-04-23-tool-trace-*` and `docs/blog/2026-04-23-platform-instructions-governance`
**Owner:** PMM → Social Media Brand | **Canonical:** `https://doc.moleculesai.app/docs/development/observability`

---

## Angle: "See what your agent did. Enforce what it should do."

Two separate product capabilities. One narrative:
- Tool Trace → answer the retrospective question "what did my agent actually do?"
- Platform Instructions → answer the proactive question "what should my agents be allowed to do?"
- Together → complete observability + governance loop for enterprise AI fleets

**Lead with Tool Trace** (accessible to all audiences, available on all plans).
**Pull in Platform Instructions** (governance angle, also available on all plans — don't lead with this on X).

---

## X (Twitter) — Primary thread (6 posts)

### Post 1 — Hook (observability: the gap)
Your AI agent just ran for 20 minutes.
It returned a result.
You have no idea what it actually did in there.

That's not a debugging failure. That's a product gap.

Tool Trace: every tool call, every input, every output — in the response.

→ https://doc.moleculesai.app/docs/development/observability

---

### Post 2 — What Tool Trace captures (product detail)
Each A2A response from a Molecule AI agent now carries a structured tool trace:

→ Which tools were called (Write, Bash, Grep, MCP tools)
→ What inputs were passed (file paths, commands, prompts)
→ What came back (output preview, ~200 chars)
→ Which calls ran in parallel (run_id pairing)

Same level of detail as a debugger trace. Embedded in the response. No extra API calls.

→ https://doc.moleculesai.app/docs/development/observability

---

### Post 3 — Why it matters for production
When something goes wrong in an AI agent run, the question is always the same:
"What did the agent actually do?"

Most platforms give you the result. Molecule AI gives you the trace.

If you're running agents in production — especially anything touching code, data, or external APIs — you need this visibility before something goes wrong.

Tool Trace is live on all Molecule AI plans.

→ https://doc.moleculesai.app/docs/development/observability

---

### Post 4 — The governance side (enterprise angle)
Here's the other half of Phase 34:

Platform Instructions — governance rules enforced at the system prompt level, before every agent turn.

No post-hoc filtering. The rule is part of what the agent is instructed to do from the first token.

Two scopes: global (every workspace in your org) or workspace-scoped (one team, one set of rules).

Security teams notice this architecture.

→ https://doc.moleculesai.app/docs/development/observability

---

### Post 5 — The combination (for enterprise audience)
Tool Trace tells you what the agent did.
Platform Instructions tell it what to do before it does it.

Run them together: write the policy once, enforce it everywhere, trace every execution.

That's the observability + governance loop enterprise AI teams need.

Tool Trace: all plans.
Platform Instructions: all plans.

→ https://doc.moleculesai.app/docs/development/observability

---

### Post 6 — CTA + Phase 34 reference
Phase 34 shipped today: Tool Trace + Platform Instructions.

Tool Trace — every tool call, every input, every output — in every A2A response.

Platform Instructions — org-wide and workspace-scoped governance at the system prompt level.

If you're running AI agents in production and don't know what they're doing inside a turn — fix that.

→ https://doc.moleculesai.app/docs/development/observability

---

## LinkedIn — Single post

**Title:** Two things enterprise AI teams need before they trust a production agent platform

When you're running an AI agent fleet in production — touching code, data pipelines, customer data, or external APIs — there are two questions that come up before the first compliance review:

1. **What did the agent actually do?** Not just the output. The full sequence of tool calls, inputs, and results. If something goes wrong, you need to reconstruct what happened.

2. **Can we enforce what the agent should do at the platform level?** Before the first turn executes. Not a filter — a governance rule baked into the agent's instruction set.

Most platforms answer neither question well. Some answer one. Phase 34 from Molecule AI ships both:

**Tool Trace** — embedded in every A2A response. Every tool call, input, output preview, parallel call grouping, and timing metadata. The full trace without an extra API call. Available on all plans.

**Platform Instructions** — configurable rules scoped globally or per-workspace. Enforced before every agent turn. The rule is part of the system prompt, not a filter applied after. Available on all plans.

Together: write the policy once, enforce it everywhere, trace every execution.

If you're scaling AI agents in production and don't have this — it's the gap worth closing.

→ https://doc.moleculesai.app/docs/development/observability

#MoleculeAI #AIAgents #AgentPlatform #EnterpriseAI #AIGovernance #DevOps

---

## Visual Asset Requirements

1. **Tool Trace screenshot** — A2A response payload showing `Message.metadata.tool_trace` array. Clean, dark theme. Show 3-4 entries with tool name + output preview visible.
2. **Platform Instructions diagram** — System prompt structure: global instructions + workspace instructions → prepended to system prompt → agent reasoning. Clean architecture diagram, not a screenshot.
3. **LinkedIn cover** — Split card: left side "What did the agent do?" with trace snippet / right side "What should it do?" with instruction snippet. Dark mode, molecule navy.

---

## Campaign notes

**Audience:** DevOps + platform engineers (X primary), enterprise IT/security (LinkedIn primary)
**Tone:** Concrete + practical — don't announce, show the output
**Angle:** Lead with observability (Tool Trace) — accessible to all audiences. Platform Instructions as the enterprise pull-through.
**Differentiation:** Tool Trace is embedded in every A2A response — no extra polling, no separate observability stack to integrate.
**CTA:** `https://doc.moleculesai.app/docs/development/observability`
**Coordinate with:** Phase 30 social campaign Day 6. Tool Trace is the natural continuation of the observability story from EC2 Console Output (Day 4) → Org API Keys (Day 5) → Tool Trace + Platform Instructions (Day 6).

---

*PMM drafted 2026-04-23 — Phase 34 GA launch social. Approved by Marketing Lead 2026-04-23. Plan gating note: see `docs/marketing/briefs/platform-instructions-plan-gating-note.md`.*