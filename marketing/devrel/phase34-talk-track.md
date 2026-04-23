# Phase 34 Talk-Track — Tool Trace, Platform Instructions, Partner API Keys

> **Purpose:** One-pager for sales, DevRel, and PMM to explain Phase 34 capabilities in conversation
> **Audience:** DevOps leads, platform engineers, VP Engineering, CI/CD teams, compliance officers
> **Format:** 5-minute ready / 30-second elevator / objection handles
> **Owner:** DevRel
> **Source:** PR #1686 (molecule-core), Phase 34 positioning briefs

---

## The Hook

*"Molecule AI Phase 34 is about closing the gap between 'agent works in demo' and 'agent works in production.' Three capabilities: Tool Trace, Platform Instructions, and Partner API Keys."*

---

## Capability 1 — Tool Trace (30 seconds)

**What it is:** Every tool call an agent makes is logged in your org's activity_logs. Tool name, inputs, sanitized output preview. Queryable via API.

**Why it matters:** You can now verify what your agent actually did — not just what it said it did. Before Tool Trace, debugging meant replaying conversations and hoping the agent mentioned its actions in output. With Tool Trace, you query the activity log.

**For DevOps:** "Your on-call engineer can see exactly which tools ran before a workspace crashed — without needing to reproduce the issue."

**For compliance:** "Every tool call is stored in your org's database. Audit trails that don't require reconstructing LLM output."

**Objection handle:** *"Isn't this just LangSmith?"*
> No — LangChain callbacks log to a third-party SaaS. Tool Trace is built into the A2A protocol layer and stored in YOUR org's activity_logs. No SDK changes, no third-party dependency, no data leaves your infrastructure.

---

## Capability 2 — Platform Instructions (30 seconds)

**What it is:** Admins define AI governance rules once in the platform. Every agent in the org inherits them — at boot and on periodic refresh. Rules are injected as the first section of the system prompt (highest precedence). 8KB cap enforced by DB constraint.

**Why it matters:** "Define once, inherit everywhere." No per-developer discipline, no version-controlled prompt files that drift out of sync, no agent that can accidentally override a governance rule.

**For VP Engineering:** "Your AI governance policy is enforced at the infrastructure layer — not the prompting layer. Agents can't override it."

**For platform teams:** "Running a multi-tenant platform? Tenant A gets 'no external file writes without confirmation.' Tenant B gets 'summarize in 3 sentences.' Same platform, different rules per workspace."

**For compliance:** "Centrally-defined, auditable rules that apply to every agent — current and future."

**Objection handle:** *"What stops a developer from just editing their system prompt to override the rule?"*
> The rule is injected at runtime from the platform database. It's the first thing in the system prompt, and agents receive it fresh on every periodic refresh. It's infrastructure-enforced, not convention-enforced.

**Objection handle:** *"Can this be abused to inject malicious instructions into agents?"*
> Only admins with AdminAuth can create instructions. The 8KB content cap prevents token-budget DoS. Instructions are stored in your org's database — they don't come from an external source.

---

## Capability 3 — Partner API Keys (30 seconds)

**What it is:** Programmatic org provisioning via SHA-256 hashed, scoped, rate-limited API keys. Org-scoped — can't access other orgs. Immediate revocation. Full audit trail.

**Why it matters:** "Your CI pipeline can spin up a test org, run integration tests, and delete it — without a human, without a browser session."

**For CI/CD teams:** "Reproducible test org lifecycle. Create → test → delete. Automated, auditable, repeatable."

**For partner platforms:** "Marketplace resellers can provision a white-labeled org for each new customer, linked to their billing system."

**For internal ops:** "Secrets automation tools and CI pipelines can manage org provisioning without needing a human admin session."

**Demo line:**
> "Here's a curl command that creates a test org, waits 5 seconds, polls for status, runs tests, and deletes the org. No browser. No WorkOS session. 30 seconds end-to-end."

```bash
ORG_KEY="mol_pk_..."
curl -X POST "$CP_URL/cp/orgs" \
  -H "Authorization: Bearer $ORG_KEY" \
  -d '{"slug": "ci-test-001", "name": "CI Test"}'
# → poll → test → DELETE
```

**Objection handle:** *"Are these keys as secure as human-admin sessions?"*
> Keys are SHA-256 hashed in the database — never stored in plaintext. They're scoped to the creating org and can't access other tenants. Revocation is immediate — 401 on next request, no cache to wait out. Per-key rate limiting prevents abuse.

---

## Putting It Together (20 seconds)

*"These three capabilities are layers of a production-ready platform:*
- *Partner API Keys — the entry point for automation and partners*
- *Platform Instructions — the control plane for AI governance*
- *Tool Trace — the observability layer to verify what agents did*

*Together they answer the three questions production teams ask: Can I provision without a human? Can I enforce rules centrally? Can I verify what my agents did?"*

---

## Competitive Frame

| Competitor | Gap | Molecule AI advantage |
|---|---|---|
| **OpenAI Agents SDK** | No built-in instruction-management API or tool-call audit trail | Platform Instructions + Tool Trace built into the platform layer |
| **CrewAI** | Rules are template-only, not runtime-configurable | Platform Instructions injected at boot, refreshed periodically |
| **LangChain / LangSmith** | Third-party SaaS dependency for observability | Tool Trace stored in your org's DB — no third-party dependency |
| **Build-your-own** | Custom instrumentation required per agent | Tool Trace ships on by default — no SDK changes, no custom code |

---

## Pricing / Packaging

*PMM to fill based on Phase 34 pricing confirmed — this section gated on PMM review.*

---

## Objection Matrix

| Objection | Response |
|---|---|
| "This sounds like monitoring — we already have Datadog" | Tool Trace records tool calls — not server metrics. Datadog tells you the server was slow. Tool Trace tells you which tool the agent called and what came back. Different layer. |
| "Our agents already have governance via system prompts" | Platform Instructions are enforced at the infrastructure layer — stored in the DB, injected at boot, refreshed periodically. Agents can't override them. Prompt files can drift. Platform Instructions can't. |
| "We don't need partner API keys — we manage orgs manually" | Fine for one org. Harder at 10. Impossible at 100. Partner API Keys make org management programmatic at any scale. |
| "Can this be used to spy on our agents?" | Tool Trace is scoped to your own org's activity_logs. You can only query workspaces in your tenant. Cross-org observation is not possible — WorkspaceAuth gates every request. |
| "What happens if the platform has a bug in instruction resolution?" | The 8KB cap is enforced at the DB level (CHECK constraint), not in application code. A code bug can't bypass it. |

---

## Call-to-Action

*"Phase 34 is available today. Tool Trace and Platform Instructions are in GA. Partner API Keys are in GA. Documentation with runnable API demos is at moleculesai.app/docs/devrel/demos/tool-trace-platform-instructions. For a live demo, your SE can run the staging environment in under 5 minutes."*

---

## Quick Reference

| Feature | API | Who uses it |
|---|---|---|
| Tool Trace | `GET /workspaces/:id/activity` → `tool_trace[]` | DevOps, compliance, platform engineers |
| Platform Instructions | `POST /instructions`, `GET /workspaces/:id/instructions/resolve` | VP Eng, platform admins, compliance |
| Partner API Keys | `POST /cp/orgs`, `GET /cp/orgs/:slug`, `DELETE /cp/orgs/:id` | CI/CD, partner platforms, internal ops |

---

*Owner: DevRel | Review: PMM before external use | Last updated: 2026-04-23*