# PMM Positioning Brief: Platform Instructions

**Date:** 2026-04-23
**Owner:** PMM
**Status:** APPROVED — ready for Social Media Brand and Content Marketer use
**Phase:** 34

---

## Core Positioning

**One-liner:** The only agent platform where governance is enforced before the first token is generated — not after.

**Longer form:** Platform Instructions lets enterprise IT and platform teams enforce org-wide policy rules at the system prompt level. Rules are prepended to the agent's system prompt automatically at workspace startup. No SDK changes. No code deploys. No agent-side integration work.

---

## Key Differentiation

Most AI governance tools work by **filtering outputs** after the agent has already decided what to do. That's governance as audit — useful for compliance reports, too late to prevent harm.

Platform Instructions enforces governance **at the source**: before the first token is generated. The agent is instructed what to do — and what not to do — from its very first turn. That's governance as architecture.

**Competitor gap:** No other A2A-native agent platform surfaces a governance control that takes effect before the agent can act. Cloud providers (AWS Agentic, GCP Vertex AI) offer post-hoc logging and blocking, but not pre-prompt injection.

---

## Target Audience

- **Primary:** Enterprise platform engineers and DevOps teams responsible for AI deployment governance
- **Secondary:** Security/compliance teams requiring audit-ready control surfaces before approving production agents

---

## 3 Core Claims (for social copy)

1. **"Before the first turn, not after"** — Platform Instructions governance is architectural, not advisory. Rules are prepended to the system prompt at startup. The agent can't act outside those rules because it was never instructed to.

2. **"Global and workspace-scoped, zero SDK required"** — One rule applies to every workspace in the org. Another rule applies only to a specific workspace. Both enforced automatically at startup. No agent-side code changes.

3. **"Enterprise-ready: IDOR-protected, DoS-hardened"** — The resolve endpoint is IDOR-protected under wsAuth. An 8KB content cap prevents token-budget DoS. `run_id`-scoped parallel tool call pairing ensures correct attribution under concurrent execution.

---

## What NOT to Say

- Do not frame Platform Instructions as "prompt engineering" — it's governance architecture, not content tuning
- Do not compare directly to AWS/GCP — they don't have this capability, so direct comparison amplifies their feature gap
- Do not imply the governance can be bypassed — the architecture prevents bypass at the source, not just reports it

---

## CTA

Enable Platform Instructions via the CRUD API → governance rules auto-inject into the system prompt at workspace startup → deploy your A2A agent with zero code changes.

---

## Assets

- Blog post: `docs/blog/2026-04-23-platform-instructions-governance/index.md`
- Launch brief (PR #1686): `docs/marketing/launches/pr-1686-tool-trace-platform-instructions.md`
- Phase 34 messaging matrix: `docs/marketing/phase34-messaging-matrix.md`
