# Phase 34 — Partner API Keys + Governance + Tool Trace — Social Copy

> **Status:** DRAFT — awaiting Phase 34 GA confirmation
> **Features:** Partner API keys (org provisioning), platform instructions (AI governance), tool trace (observability)
> **Owner:** DevRel Engineer
> **Target platforms:** X, LinkedIn

---

## Phase 34 Feature Summary

Phase 34 ships three interconnected capabilities:

| Feature | What it does | Who it's for |
|---|---|---|
| **Partner API Keys** (`mol_pk_*`) | Programmatic org provisioning via hashed, scoped, rate-limited API keys | Partners, CI/CD, marketplace resellers |
| **Platform Instructions** | Org/workspace-scoped AI rules prepended to every agent prompt | Platform admins, compliance teams |
| **Tool Trace** | Every A2A response includes a `tool_trace` array — who called what, with what input, what came back | Developers, DevOps, auditors |

---

## X — Partner API Keys Angle

### X-A1: Developer / CI angle

```
Your CI pipeline needed a test org. You spun one up with an API key.

No browser. No manual clicks. No WorkOS session.

POST /cp/orgs → org created → tests run → org deleted.
10 seconds. Repeatable. Audit-logged.

Partner API keys for Molecule AI:
→ moleculesai.app/docs/partner-api-keys
```

**Tags:** `#DevOps` `#CIautomation` `#MoleculeAI`

---

### X-A2: Platform engineer / infra angle

```
Most SaaS platforms require a browser session to provision a customer org.

Molecule AI's partner API keys change that:
- SHA-256 hashed, never stored in plaintext
- 8 granular scopes (orgs:create, orgs:read, ...)
- Per-key rate limiting
- Revocation = instant 401

Your partner platform, your billing integration, your CI —
all via API key. All auditable.
→ moleculesai.app/docs/partner-api-keys
```

**Tags:** `#PlatformEngineering` `#SaaS` `#MoleculeAI`

---

### X-A3: Marketplace reseller angle

```
You run a marketplace. You want each customer on their own Molecule AI org.

With a partner API key, you automate the full lifecycle:
1. Customer signs up on your platform
2. You POST /cp/orgs → org provisioned instantly
3. Customer lands on their canvas, ready to go
4. You manage billing, they manage agents

White-label AI agent platform, zero manual ops.
→ moleculesai.app/partner
```

**Tags:** `#Marketplace` `#Resellers` `#AIaaS` `#MoleculeAI`

---

## X — Platform Instructions (AI Governance) Angle

### X-B1: Compliance / governance angle

```
Your AI agents need guardrails. Platform Instructions make them stick.

Set org-wide rules: "Never commit to main", "Always summarize in 3 sentences",
"Flag sensitive data before external API calls."

Every agent in your org starts with those rules prepended to their system prompt.
No per-agent config. No relying on prompting discipline.

AI governance that travels with the platform, not the prompt.
→ moleculesai.app/docs/platform-instructions
```

**Tags:** `#AIGovernance` `#Compliance` `#EnterpriseAI` `#MoleculeAI`

---

### X-B2: Multi-tenant AI policy angle

```
Every tenant on your platform has different AI behavior requirements.

With Molecule AI's workspace-scoped Platform Instructions:
- Tenant A agents: "No external file writes without user confirmation"
- Tenant B agents: "Summarize every API response before responding"
- Tenant C: no restrictions

One platform. Per-workspace governance rules.
No code changes. No agent redeployment.
→ moleculesai.app/docs/platform-instructions
```

**Tags:** `#MultiTenant` `#EnterpriseAI` `#SaaS` `#MoleculeAI`

---

## X — Tool Trace (Observability) Angle

### X-C1: Developer / debugging angle

```
Your agent did something unexpected.

With Molecule AI's Tool Trace, you see exactly what happened:
- Which tools were called
- What inputs were passed
- What outputs came back
- In what order

No more reading tea leaves in LLM output. Full instrumented trace.

Tool Trace: built into every A2A response.
→ moleculesai.app/docs/tool-trace
```

**Tags:** `#AgenticAI` `#Observability` `#Debugging` `#MoleculeAI`

---

### X-C2: DevOps / audit angle

```
Regulators want to know: what did your AI agents do, and why?

Molecule AI Tool Trace gives you:
- Tool-by-tool call history per agent session
- Input/output logging (with configurable redaction)
- run_id-paired start/end events
- Stored in your org's activity_logs

Audit-ready agent behavior records.
→ moleculesai.app/docs/tool-trace
```

**Tags:** `#AICompliance` `#Observability` `#DevOps` `#MoleculeAI`

---

## X — Combined Phase 34 Angle (launch announcement)

### X-D1: "Three things" launch post

```
Phase 34 shipped on Molecule AI:

1/ Partner API keys — provision orgs from code, not a browser
   → mol_pk_* scoped keys, SHA-256 hashed, rate-limited

2/ Platform Instructions — org/workspace AI governance that travels with every agent
   → rules prepended to system prompt, 8KB cap, DB-enforced

3/ Tool Trace — instrumented A2A responses, every tool call logged
   → run_id-paired, 200-entry cap, audit-ready

All three shipped today.
→ moleculesai.app
```

---

## LinkedIn Posts

### LinkedIn-E1: Partner API Keys — CTO / Platform Engineering buyer

```
The last thing your platform team wants is a manual "create org" workflow every time a new customer signs up.

With Molecule AI's Partner API Keys, your integration layer handles it:

→ POST /cp/orgs with your scoped API key
→ Org provisioned in seconds — canvas, database branch, agent fleet ready
→ Customer redirected to their org
→ Audit log records every org created, by whom, when

What you get:
- SHA-256 hashed keys, never stored in plaintext
- 8 granular scopes (least-privilege by design)
- Per-key rate limiting (prevents runaway provisioning)
- Instant revocation = immediate 401

Your partner platform. Your billing system. Your CI.
All integrated. All auditable.

Learn more → moleculesai.app/docs/partner-api-keys
```

---

### LinkedIn-E2: Platform Instructions — VP Engineering / Compliance

```
Your AI agents need guardrails. But how do you enforce them at scale?

Molecule AI's Platform Instructions let you define org-wide or workspace-scoped rules
that are prepended to every agent's system prompt — automatically, consistently,
without relying on prompting discipline.

Examples:
- "Never commit to main without a PR review"
- "Flag any PII before external API calls"
- "Summarize responses in 3 sentences or fewer"
- "Require user confirmation before file writes"

Rules are:
→ Stored in your org database (not in agent code)
→ Enforced at platform level (not prompt-level)
→ Enforceable per-workspace (multi-tenant governance)
→ Bounded at 8KB (DB CHECK constraint prevents token budget abuse)

Your platform. Your AI policy. Enforced at the infrastructure layer.

Learn more → moleculesai.app/docs/platform-instructions
```

---

### LinkedIn-E3: Tool Trace — DevOps / Platform Engineering

```
When an AI agent does something unexpected, how do you debug it?

Traditional approach: re-read the prompt, look at the final output, guess.

Molecule AI Tool Trace: every A2A response includes a full instrumented log.

What you get per tool call:
- Tool name
- Input (what was passed)
- Output preview (what came back)
- run_id pairing (handles parallel calls correctly)
- Stored in your org's activity_logs

Capped at 200 entries to prevent runaway loops.

Use cases:
- Debug unexpected agent behavior without re-running
- Audit what decisions an agent made and why
- Track tool usage patterns across your org
- Build dashboards on top of agent activity

Observability that matches how AI agents actually work.

Learn more → moleculesai.app/docs/tool-trace
```

---

## Thread Template (launch day)

**Tweet 1 (hook):**
```
Three things that shipped on @MoleculeAI today — and why each one unlocks a whole new use case.
```

**Tweet 2 (partner keys):**
```
1/ Partner API keys

Your CI pipeline, billing system, or partner platform can now provision a full Molecule AI org via API.

No browser. No manual clicks. Just a scoped, hashed, rate-limited API key.

POST /cp/orgs → org live → audit logged.
```

**Tweet 3 (platform instructions):**
```
2/ Platform Instructions

Define org-wide AI governance rules once. Every agent in your org inherits them automatically.

"No external file writes without confirmation." "Flag PII before API calls." "Summarize in 3 sentences."

Rules enforced at the platform layer. No prompting discipline required.
```

**Tweet 4 (tool trace):**
```
3/ Tool Trace

Every A2A response now includes a full instrumented call log: which tools ran, with what inputs, what came back.

Debug your agents the way you debug your code.
```

**Tweet 5 (CTA):**
```
All three shipped today.

→ moleculesai.app
```

---

## UTM Conventions

| Platform | Campaign | Medium |
|---|---|---|
| X | `phase34-partner-api-keys` | `social` |
| LinkedIn | `phase34-partner-api-keys` | `social` |
| Email | `phase34-partner-api-keys` | `email` |
| Direct | `phase34-partner-api-keys` | `direct` |

Landing URL: `https://moleculesai.app/partner-api-keys`
Docs URL: `https://moleculesai.app/docs/partner-api-keys`
