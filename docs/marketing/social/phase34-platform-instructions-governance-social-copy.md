# Platform Instructions — Governance Social Copy
**Feature:** Platform Instructions (Phase 34 — Enterprise)
**Blog:** `docs/blog/2026-04-23-platform-instructions-governance/index.md`
**Status:** Ready for Social Media Brand — coordinated with Phase 34 launch

---

## X (Twitter) — Standalone governance framing (4 posts)

### Post 1 — Hook / problem framing
Most AI agent platforms enforce policy *after* the agent decides what to do.

Platform Instructions: governance at the system prompt level.
Rules are prepended as core instructions — before the first token generates.

Enterprise governance, shipped.

### Post 2 — What makes it different
A post-hoc filter evaluates after the decision.
Platform Instructions govern before the decision.

Rules sit in the system prompt — shaping reasoning from the ground up.
Not a gate. A context.

### Post 3 — Two scopes
Platform Instructions supports two scopes:

→ Global — applied to every workspace in your org
→ Workspace — applied to a specific workspace only

One rule or fine-grained control. Both enforced before the first turn.

### Post 4 — CTA
Platform Instructions: write a governance policy once, enforce it everywhere.

Global or workspace-scoped rules, injected into the system prompt at startup.
Enterprise-only. Available now.

→ [governance blog post link]

---

## LinkedIn — Governance lead

**Title:** Policy enforcement at the system prompt level — not after the agent decides

The moment an AI agent goes into production, the governance question stops being theoretical. Which tools can it call? What data can it write to? Are there constraints that apply to every turn?

Most platforms answer these questions with post-hoc filtering — a rule that evaluates after the agent has already decided what to do.

Platform Instructions takes a different approach: governance at the source, before the first token is generated. Rules are prepended to the system prompt at workspace startup. The agent doesn't receive these rules as a filter — it receives them as part of its core instruction set.

A filter can be worked around. A system prompt instruction shapes the agent's reasoning from the ground up.

**Two scopes, one governance plane:**

→ Global — applied to every workspace in your org automatically
→ Workspace — applied to a specific workspace only

When a workspace starts, Molecule AI resolves all applicable instructions — global rules combined with workspace-specific ones — and prepends them to the agent's system prompt. The resolved set is fetched once at startup and cached, so governance is enforced without per-turn latency overhead.

**Enterprise-grade access control:**

→ Global instructions managed by org admins
→ Workspace instructions managed by workspace admins within their own scope
→ Resolve endpoint gated by Workspace Auth — workspaces cannot retrieve each other's instructions

For compliance teams: every instruction change is audited. Which admin created it, when, what scope, and what it contains — without requiring external logging infrastructure.

Platform Instructions shipped in Phase 34. Enterprise plans include org-scoped governance, wsAuth-gated resolve endpoints, and full instruction audit logs.

→ [governance blog post link]

UTM: `?utm_source=linkedin&utm_medium=social&utm_campaign=phase34-platform-instructions-launch`

---

## Publishing notes

- **Platform:** X + LinkedIn
- **Audience:** IT governance teams, security engineers, enterprise platform leads
- **Tone:** Precise. Governance-first. Lead with the architectural distinction (prepended vs. post-hoc), not feature specs.
- **Angle:** The differentiator vs. post-hoc filtering is the key message — "before the first token" resonates with IT governance buyers who have been burned by output-scanning guardrails.
- **Coordinate with:** Tool Trace launch (same phase, different angle — observability vs. governance)
- **Hashtags:** #MoleculeAI #AIGovernance #EnterpriseSecurity #PlatformEngineering #AgenticAI