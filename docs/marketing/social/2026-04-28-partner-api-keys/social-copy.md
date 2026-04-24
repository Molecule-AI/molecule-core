# Phase 34 — Partner API Keys Social Copy
Campaign: partner-api-keys | Blog: `docs/marketing/blog/2026-04-23-partner-api-keys/index.md`
Slug: partner-api-keys
Publish day: 2026-04-28 (Day 8, Phase 34)
Assets: OG image at `docs/assets/blog/2026-04-23-partner-api-keys-og.png`
Hashtags: #AgenticAI #MoleculeAI #PlatformEngineering #Marketplace #CI/CD #API
UTM: `?utm_source=twitter&utm_medium=social&utm_campaign=partner-api-keys`

---
## X Thread — 5 posts

### Post 1 — Hook (programmatic org management gap)
Your marketplace needs to provision a new customer org.
Your CI pipeline needs a clean test environment.
Your partner portal needs to spin up agent infrastructure.

None of them should need a human with a browser.

Partner API Keys: programmatic org management via `mol_pk_*`.
→ https://docs.molecule.ai/blog/partner-api-keys

---

### Post 2 — What it actually does (the API surface)
`mol_pk_*` keys let you:

→ `POST /cp/orgs` — provision an org
→ `GET /cp/orgs/:id/status` — poll until ready
→ `DELETE /cp/admin/partner-keys/:id` — revoke cleanly

No browser. No admin dashboard. Just the API.

Scoped to exactly what each integration needs.

→ https://docs.molecule.ai/blog/partner-api-keys

---

### Post 3 — CI/CD use case
Your test suite needs a clean Molecule AI org per run.

Old flow: spin up a test environment → pray nothing's shared → manual cleanup.

New flow:
1. `POST /cp/orgs` (via `mol_pk_*` key)
2. Run integration tests
3. `DELETE /cp/admin/partner-keys/:id`
4. Org is gone. Run is clean.

No shared state. No test pollution. No manual cleanup.

→ https://docs.molecule.ai/blog/partner-api-keys

---

### Post 4 — Marketplace reseller angle
You run a marketplace. You resell AI agent infrastructure to your customers.

Your customer signs up on your platform.
You call `POST /cp/orgs` with their details.
They're redirected to their Molecule AI tenant.

The reseller controls the billing relationship.
Molecule AI handles the infrastructure.

You never hand a credential to a human.

→ https://docs.molecule.ai/blog/partner-api-keys

---

### Post 5 — Scoping + revocation story
A partner key is only as powerful as you make it.

`orgs:create` → can create orgs, nothing else
`orgs:list` → can read org status, nothing else
`billing:read` → can see subscription, nothing else

Revoke a key → the integration stops working immediately.
The org stays. The path closes.

That's the security model.

→ https://docs.molecule.ai/blog/partner-api-keys

---

## LinkedIn — Single Post

**Title:** The difference between "agents can talk to each other" and "your platform can own the agent infrastructure your customers use"

**Body:**

When you're building on top of an AI agent platform, the last thing you want is for every customer onboarding to go through a human with an admin dashboard.

That's the problem Partner API Keys solve.

`mol_pk_*` keys let marketplace resellers, CI/CD pipelines, and automation platforms create and manage Molecule AI orgs programmatically — no browser, no admin session, just an API call.

Here's what a customer onboarding flow looks like:

1. Customer signs up on your marketplace
2. You call `POST /cp/orgs` with their details
3. You poll `GET /cp/orgs/:id/status` until provisioning completes
4. You redirect the customer to their Molecule AI tenant

Your customer gets their agent infrastructure. You control the billing relationship. Molecule AI handles the platform.

The same pattern works for CI/CD: create a temporary org per test run, run integration tests, delete the org and revoke the key. Each run is fully isolated. No shared state. No manual cleanup.

What makes this production-ready versus a prototype:

→ **Scoped by default** — a CI pipeline key gets `orgs:create` + `orgs:list`. A marketplace key gets `orgs:create` + `orgs:delete` + `workspaces:create`. Each key does exactly what its integration needs.

→ **Immediate revocation** — `DELETE /cp/admin/partner-keys/:id` closes the integration path instantly. The org stays; the automation is dead.

→ **Full audit trail** — every API call attributed to the key that made it. You can trace a provisioning event back to the integration that triggered it.

→ **Rate-limited per key** — a misbehaving integration hits its own ceiling without affecting other partners or organic traffic.

Partner API Keys are available on Partner and Enterprise plans. Programmatic org management — not a browser dependency in sight.

→ [Read the integration guide](https://docs.molecule.ai/blog/partner-api-keys)
→ [Phase 34 Plan](https://docs.molecule.ai/blog/molecule-ai-build-plan) — all eight steps shipped

**Hashtags:** #AgenticAI #MoleculeAI #PlatformEngineering #Marketplace #CICDPipeline #API
**CTA:** Bookmark for your next platform integration build. Programmatic org management is live.

---

## Campaign notes

**Audience:** Platform engineers / API-first builders (X), Marketplace operators / CI/CD teams / enterprise DevOps (LinkedIn)
**Tone:** Concrete and API-first. Show the flow, not the concept. The CI/CD ephemeral org use case is the most visceral example — use it.
**Differentiation:** No browser dependency is the core differentiator vs. every other org management flow in the market today.
**Coordination:** Publish Day 8 (2026-04-28). No same-day conflicts identified in the current queue.
**Status:** Draft — pending PM confirmation of pricing tier language (Partner/Enterprise plans vs. open beta)
**Self-review applied:** No timeline claims, no person names, no benchmarks. Pricing tier language flagged above.