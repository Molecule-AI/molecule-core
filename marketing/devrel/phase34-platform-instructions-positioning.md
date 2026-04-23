# Platform Instructions — Positioning Brief
## Phase 34 | GA: April 30, 2026

> **Status:** READY — blocks Social Media Brand social copy | **Owner:** DevRel
> **Source:** PR #1686 (`molecule-core`) | **Conflicting PRs:** LangGraph A2A (open, moat intact — verified by PMM)
> **Conflicting narrative risk:** A2A v1 narrative (issue #1286) — pending Content Marketer assignment

---

## What It Is

Platform Instructions let admins define org-wide or workspace-scoped AI governance rules that are automatically injected into every agent's system prompt — at workspace boot and on periodic refresh.

Rules are stored in `platform_instructions` table, resolved via `GET /workspaces/:id/instructions/resolve` (WorkspaceAuth-gated), and prepended as the `# Platform Instructions` section — the first section of the system prompt, highest precedence. 8KB content cap enforced by DB CHECK constraint.

---

## Positioning Statement

> *"AI governance that travels with the platform, not the prompt."*

Platform Instructions solve the problem of "we have AI governance rules, but they only exist if every developer remembers to include them." Rules are defined once by an admin, stored in the platform, and enforced at the infrastructure layer — not the prompting layer.

---

## Target Audiences (priority order)

| Audience | Why they care |
|---|---|
| **VP Engineering / CTO** | AI governance without engineering-wide discipline |
| **Platform admins** | Per-workspace rules for multi-tenant platforms |
| **Compliance teams** | Enforceable, auditable, centrally defined rules |
| **Security engineers** | "No shell commands for users" enforcement at platform level |
| **Platform teams building on Molecule AI** | White-label AI platform with customizable per-tenant behavior |

---

## Competitive Differentiation

| Alternative | What they do | Platform Instructions advantage |
|---|---|---|
| **Per-developer system prompt engineering** | Developers add rules to their agent prompts manually | Rules defined once by admin, enforced for every agent automatically |
| **Prompt injection at runtime** | Manual middleware that prepends prompts | Platform-native — stored in DB, resolved at boot, 8KB capped |
| **Anthropic Moderation API** | Content filtering after the fact | Platform Instructions prevent unwanted behavior proactively — not filter it reactively |
| **LangChain LCEL + runnable config** | Per-chain instruction override | Works at the chain level; Platform Instructions work at the platform level, across every agent |

**Key differentiator:** Platform Instructions are enforced at the infrastructure layer — not the prompting layer. Agents receive the resolved system prompt with rules prepended. They cannot override them by editing their own prompt.

---

## Messaging Pillars

1. **"Define once, inherit everywhere"** — One admin sets the rule. Every agent in the org inherits it. No per-developer discipline required.

2. **"Platform-level, not prompt-level"** — Rules are stored in the platform database, resolved at boot, and injected as the first section of the system prompt. Agents can't override them.

3. **"Per-workspace governance for multi-tenant platforms"** — Tenant A gets "no external file writes without confirmation." Tenant B gets "summarize in 3 sentences." Same platform, different rules per workspace.

4. **"8KB cap — enforced by the database"** — A DB CHECK constraint prevents oversized instructions from being prepended. Token budget DoS isn't possible. This is not a code check — it's a database rule.

5. **"Compliance-ready"** — Rules are auditable (stored in `platform_instructions` table), scoped (global or workspace), and instantly revocable (disable or delete the instruction).

---

## Key Copy Angles (for social)

### Angle 1: Compliance / governance (primary)
```
Your AI agents need guardrails. But how do you enforce them at scale?

Platform Instructions: define org-wide rules once. Every agent in your org inherits them automatically.

Enforced at the platform layer. Not prompt-level. Not per-developer discipline.
→ moleculesai.app/docs/platform-instructions
```

### Angle 2: Multi-tenant AI policy
```
Every tenant on your platform has different AI behavior requirements.

With Platform Instructions:
- Tenant A agents: "No external file writes without confirmation"
- Tenant B agents: "Summarize every API response before responding"
- Tenant C: no restrictions

One platform. Per-workspace governance. No code changes.
→ moleculesai.app/docs/platform-instructions
```

### Angle 3: Platform teams building on Molecule AI
```
You're building a white-label AI platform.

With Molecule AI Platform Instructions, your customers' agents inherit their own AI governance rules — automatically, per workspace.

You define the platform. Your customers define the rules.
→ moleculesai.app/docs/platform-instructions
```

---

## Use Cases (for blog / longer-form content)

1. **Compliance: "No PII in external API calls"** — Global instruction across the org. Every agent flags PII before external calls. No prompting discipline required.

2. **Security: "No shell commands for user-facing agents"** — Workspace-scoped to production. Agents can still use shell commands in internal provisioning scripts. Governance enforced at the right scope.

3. **Multi-tenant SaaS: per-tenant AI behavior** — Each tenant has different rules. Tenant A agents are conservative; Tenant B agents are more permissive. Same platform, different resolved instructions.

4. **Onboarding: "Default to dark theme"** — A workspace-scoped instruction that applies to UI-generating agents only. Clean way to enforce UI conventions without touching agent code.

5. **Cost control: "Summarize responses in 5 sentences or fewer"** — Global instruction to reduce token usage. Enforced across the org without per-agent prompting.

---

## Objection Handlers

**"Can't developers just override the system prompt?"**
> The resolved instruction string is injected by the runtime at boot — agents receive their system prompt with Platform Instructions as the first section. They cannot edit the received string. This is enforced at the infrastructure layer, not the prompting layer.

**"Why not just use system prompt engineering?"**
> Prompting is per-developer, per-agent, and relies on discipline. Platform Instructions are enforced at the platform level — every agent in a workspace inherits them automatically. Define once, enforce everywhere. No "we should add this rule to all our agents" problem.

**"How is this different from Anthropic's prompt management?"**
> Anthropic's prompt management is per-project in their dashboard. Platform Instructions work across every agent on the platform, are scoped per-workspace for multi-tenant use cases, and are enforced at boot — not injected as a separate prompt layer that the LLM might override.

**"What if the instruction is too long?"**
> The 8KB content cap is enforced by a DB CHECK constraint. If you try to create an instruction larger than 8KB, the DB rejects it. This is not a code check — it cannot be bypassed.

**"How do I know the agent actually follows the instruction?"**
> Platform Instructions are prepended as the first section of the system prompt with highest precedence. The LLM processes them before any other content. Compliance enforcement is a prompt architecture decision — for hard enforcement (can't be overridden by any prompt), a runtime guard would be needed.

---

## Content Status

| Asset | Status | Notes |
|---|---|---|
| Talk-track (this feature) | ✅ Done | `phase34-talk-track.md` |
| Social copy | ✅ Done | `phase34-social-copy.md` X-B1, X-B2 |
| Screencast storyboard | ✅ Done | PR #1878 |
| TTS narration | ✅ Done | `narration.txt` |
| **Positioning brief (this doc)** | ✅ Done | Blocks Social Media Brand |
| **Blog post (AI governance)** | ⏳ PMM blocked | In PR #1799 — PM owns |
| **X launch post** | ⏳ Brief done, awaiting Social Media Brand | Ready to execute |
| LinkedIn post | ⏳ Brief done, awaiting Social Media Brand | Ready to execute |

---

*Ready for Social Media Brand execution. GA: April 30, 2026.*