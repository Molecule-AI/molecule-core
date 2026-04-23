# Phase 34 — DevRel + Support Pre-Brief: Top-10 Community Questions
**Campaign:** Phase 34 GA Launch (April 30, 2026)
**Owner:** Community Manager
**Use:** DevRel + Support pre-briefing before launch day
**Date:** 2026-04-23
**Status:** DRAFT — for DevRel + Support pre-briefing

> These are the questions most likely to come up on Reddit, HN, Discord, and Slack on launch day. DevRel and Support should have answers ready before the announcement goes live.

---

## Partner API Keys (`mol_pk_*`)

### Q1: "Is Partner API Keys available now or is it still in beta?"

**Short answer:** Ships April 30, 2026. If you're reading this before that date, it's not yet generally available.

**Full answer:** Partner API Keys are part of the Phase 34 release. The `mol_pk_*` key format and `POST /cp/admin/partner-keys` endpoint will be live at GA. Key creation is admin-gated — you need to be an org admin to generate partner keys.

**Don't say:** "It's in beta" unless PM has confirmed a specific beta program. Don't name specific enterprise partners in community-facing copy.

---

### Q2: "Can I use Partner API Keys to create ephemeral test orgs for CI/CD?"

**Short answer:** Yes — that's the primary use case. Create org → run tests → delete org → billing stops.

**Full answer:** With a partner key scoped to `orgs:create` and `orgs:delete`, a CI pipeline can spin up a temporary org, run your test suite against a fresh Molecule workspace, and tear it down in one pipeline run. Rate limits apply (default 60 req/min per key), so for high-frequency CI you'll want a dedicated key per pipeline job or a test org pool.

**Technical note:** Org creation is async — `POST /cp/orgs` returns `202 Accepted` with `{status: "provisioning"}`. Poll `GET /cp/orgs/:slug` for `"status": "active"` before running tests.

---

### Q3: "Can partner keys access other orgs' data? Can I be locked out of my own org?"

**Short answer:** Scoped keys can only access the orgs they're authorized for.

**Full answer:** Keys have explicit `scopes` and optionally `org_id` bindings. A key created with `orgs:read` + `org_id=abc` cannot read data from org `xyz`. Revocation is immediate — `DELETE /cp/admin/partner-keys/:id` invalidates the key on the next request. There is no grace period.

---

### Q4: "What's the rate limit? Will it throttle my CI pipeline?"

**Short answer:** Default is 60 requests/minute per key. Configurable per key at creation time.

**Full answer:** Ephemeral test orgs (create → test → delete) typically run 5–10 API calls per org lifecycle, so 60 req/min handles ~6 concurrent ephemeral orgs comfortably. For high-frequency CI use a dedicated key per pipeline runner.

---

## Tool Trace

### Q5: "How is Tool Trace different from Langfuse or Helicone?"

**Short answer:** Tool Trace captures A2A-level agent behavior. Langfuse/Helicone capture LLM API calls. Different layers.

**Full answer:** Langfuse/Helicone instrument the LLM API (prompt → response). Tool Trace instruments the agent behavior layer — the sequence of tool calls, inputs, and output previews that produced the LLM call. If your agent calls `bash git diff`, `mcp__github__create_issue`, and `write_to_file`, Tool Trace records all three with their inputs/outputs. Langfuse records the model inference. They complement each other. Tool Trace is zero-config and platform-native — no SDK, no API key, no sidecar. It's in every A2A `Message.metadata` and stored in `activity_logs.tool_trace` as JSONB.

---

### Q6: "Does Tool Trace slow down my agents?"

**Short answer:** No — Tool Trace is a passive append to the A2A response and a fire-and-forget write to `activity_logs`.

**Full answer:** Tool Trace is added to `Message.metadata` in-memory after each tool call and does not block the response. The `activity_logs` write is async and capped at 200 entries per task to prevent unbounded growth. Entries are dropped from the end of the trace for tasks exceeding 200 tool calls — this is by design.

---

### Q7: "How do I actually read the tool trace? Is there a UI?"

**Short answer:** Raw trace is in `Message.metadata.tool_trace[]` on every A2A response. Canvas UI for traces is in progress (issue #759).

**Full answer:** If you're consuming A2A responses programmatically, the trace is a JSON array in the response envelope. From the Canvas chat, an activity log panel is in progress (issue #759) that will surface traces visually. The `activity_logs` table stores `tool_trace` as JSONB so you can query it directly via the API (`GET /workspaces/:id/activity?type=tool_call`).

---

## Platform Instructions

### Q8: "How is this different from system prompt configuration? Aren't I already doing this in my config.yaml?"

**Short answer:** Platform Instructions operate at the platform level, outside the workspace config. Governance happens before the agent reads its own config.

**Full answer:** `config.yaml` sets the workspace's system prompt. Platform Instructions are additional rules that the platform injects into the workspace's effective system prompt at startup — workspace admins with platform-level access can enforce rules that workspace users can't override or delete by editing `config.yaml`. The platform prepends its own rules above whatever the workspace has configured. Use case: enterprise IT sets "never commit to main without a code review" — this rule applies to every agent in the org regardless of individual workspace config.

---

### Q9: "Can Platform Instructions leak my sensitive system prompt to other orgs?"

**Short answer:** No — Platform Instructions are scoped to the workspace that created them. No cross-workspace enumeration is possible.

**Full answer:** `GET /workspaces/:id/instructions/resolve` requires workspace authentication (`wsAuth`) — a workspace can only read its own instructions. Org admins can only set instructions for workspaces they own. The instructions API has an 8KB content cap per instruction, preventing token-budget DoS from oversized rules.

---

## SaaS Federation v2

### Q10: "What's different in SaaS Federation v2? Is this just a name change?"

**Short answer:** No — it's the multi-tenant control plane architecture. New orgs get isolated, billing-ready tenant environments as a default.

**Full answer:** SaaS Federation v2 enables isolated orgs under a centralized control plane. Org data is tenant-isolated (separate Neon branch per org), billing is centralized (Stripe-backed per-org subscriptions), and cross-tenant guardrails prevent data leakage between orgs. For self-hosted users, nothing changes — the architecture is additive for SaaS tenants only.

---

## DevRel + Support Pre-Brief Summary

| Feature | Primary question | What to emphasize | What to soft-pedal |
|---------|-----------------|-------------------|-------------------|
| Partner API Keys | "Is it available?" | Ephemeral CI orgs use case | Specific partner names, pricing tiers |
| Tool Trace | "How does it compare to Langfuse?" | A2A-level vs. LLM-level; zero-config | Integration complexity |
| Platform Instructions | "Can I override it?" | Org-level governance vs. workspace config | Specific compliance frameworks |
| SaaS Fed v2 | "Is this just a rebrand?" | Multi-tenant isolation architecture | Internal implementation details |

---

*Last updated: 2026-04-23*