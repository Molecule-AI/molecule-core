# Phase 34 — Partner API Keys Social Copy
**Source:** Phase 34 (GA 2026-04-23)
**Feature:** Partner API Keys (`mol_pk_*`) — programmatic org provisioning for CI/CD, marketplace resellers, and automation platforms
**Blog:** `docs/blog/2026-04-23-partner-api-keys.md`
**Status:** Ready for Social Media Brand — coordinated with Phase 34 launch

---

## X (Twitter) — Primary thread (4 posts)

### Post 1 — Hook (CI/CD / automation angle)
Your CI pipeline shouldn't need a human with a browser to create a test org.

Partner API Keys: programmatic org creation, management, and revocation — via API.

No browser session. No manual handoff.

### Post 2 — What the key does
A `mol_pk_*` key is scoped, rate-limited, and revocable in one call.

Create orgs → poll status → provision workspaces → revoke when done.

One API call replaces the entire admin dashboard flow.

### Post 3 — Marketplace reseller angle
Marketplace listing for Molecule AI? Your buyers expect one-click deploy.

Partner API Keys: automated provisioning from click to running org in under 60 seconds.

Billing through your marketplace. Org management through ours. Same API.

### Post 4 — Security / revocation angle
`mol_pk_*` keys are org-scoped. A compromised key cannot escape its org boundary.

Revoke in one call: `DELETE /cp/admin/partner-keys/:id`

That single action closes the entire automation path. No session to wait for.

### Post 5 — CTA
Partner API Keys: built for CI/CD pipelines, marketplace resellers, and automation platforms.

Provision orgs programmatically. Scope each key to exactly what the integration needs. Revoke instantly.

→ [partner API keys blog post link]

---

## LinkedIn — Marketplace + CI/CD angle

**Title:** Programmatic org provisioning for the platforms building on Molecule AI

When your platform needs to create a Molecule AI org — for a new customer, a CI environment, or a marketplace resale — the last thing you want is to hand that flow to a human with a browser. Neither does your partner.

Partner API Keys give CI/CD pipelines, marketplace resellers, and automation platforms a programmatic way to create and manage Molecule AI orgs. No browser session. No admin dashboard. Just an API call.

**The integration flow:**

→ Create a scoped key with exactly the capabilities the integration needs (`orgs:create`, `orgs:list`, `orgs:delete`)
→ Call `POST /cp/orgs` to provision an org on behalf of your customer
→ Poll `GET /cp/orgs/{id}/status` until the org is ready
→ Redirect the customer to their dashboard
→ Revoke the key when the relationship ends

**Three channels where this matters:**

**CI/CD pipelines** — Spin up a clean test org per PR, run integration tests, delete the org when done. Each run gets a fresh environment. No shared state. No manual cleanup. No browser required.

**Marketplace resellers** — Fully automated provisioning through marketplace billing APIs. A buyer clicks "Deploy", the marketplace calls the Partner API to provision an org, charges begin on the marketplace invoice, and the buyer lands in a fully configured dashboard.

**Partner platforms** — Provision a white-labeled org for every new customer automatically. Scope each integration to exactly what the partner tier allows. Revoke cleanly when the relationship ends.

**The security model:**

`mol_pk_*` keys are org-scoped. A compromised key cannot access other tenants or the platform's own infrastructure. Rate limits are enforced per key, independently of session limits — a misbehaving integration hits its own ceiling without affecting other partners.

Every call is audited: the activity log records which Partner API Key was used, when, and what it did.

Partner API Keys are available on Partner and Enterprise plans. Contact your account team to request issuance.

→ [partner API keys blog post link]

UTM: `?utm_source=linkedin&utm_medium=social&utm_campaign=phase34-partner-api-keys-launch`

---

## Publishing notes

- **Publish day:** 2026-04-24 (Day 1 post-launch) or stagger with Tool Trace / Platform Instructions
- **Audience:** Platform engineers, DevOps, CI/CD teams, marketplace listing owners (X + LinkedIn)
- **Tone:** Operational. Concrete flows. Lead with "what the API call looks like" — resonates with developer audience
- **Angle:** The "no browser required" differentiator is the core message — every post should make that explicit
- **Coordinate with:** Phase 34 Tool Trace (observability) and Platform Instructions (governance) social copy — same launch, different angle
- **Hashtags:** #MoleculeAI #API #CIPipeline #Marketplace #Automation #DevOps #PlatformEngineering #AgenticAI