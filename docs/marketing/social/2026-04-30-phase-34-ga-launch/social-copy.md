# Phase 34 GA Launch — Social Copy
**Publish day:** 2026-04-30 (Partner API Keys GA)
**Status:** APPROVED — GA language confirmed per community FAQ and updated positioning brief (2026-04-23)
**Issue refs:** #1829 (Tool Trace/Platform Instructions thread already posted Apr 23)

---

## X / Twitter Thread (5 tweets) — Phase 34 GA

**Tweet 1 — Announcement hook**
```
Partner API Keys are generally available today.

If you're building a marketplace, a CI/CD platform, or any product on top of Molecule AI — you can now programmatically create and manage Molecule AI orgs via API.

No browser session required. No manual setup. API-first from day one. 🧵
```

**Tweet 2 — What it enables**
```
What mol_pk_* keys unlock:

→ POST /cp/admin/partner-keys — provision a Molecule AI org for your customer
→ DELETE /cp/admin/partner-keys/:id — tear it down, billing stops immediately
→ Org-scoped isolation — a compromised key can't escape its org boundary

Ephemeral test orgs per PR. Clean teardown on merge.
```

**Tweet 3 — First-mover claim**
```
We believe Molecule AI is the first agent platform with a first-class partner provisioning API.

LangGraph Cloud: per-seat SaaS licensing.
CrewAI: marketplace listing.
Molecule AI: an API to build either — programmatically, at scale.
```

**Tweet 4 — Phase 34 stack**
```
Phase 34 also shipped this week:

• Tool Trace — execution record in every A2A response
• Platform Instructions — org-level system prompt via API

Observability + governance. In one stack. [⚠️ SaaS Fed v2 — confirm PM before mentioning.]
```

**Tweet 5 — CTA**
```
Partner API Keys: GA today.

If you're a platform builder, marketplace operator, or running CI/CD on Molecule AI — this is your release.

Docs → https://docs.molecule.ai/api/partner-keys
Partner program → #partner-program on Discord
```

---

## LinkedIn Post (~250 words)

**Partner API Keys are generally available today.**

Starting today, any platform, marketplace, or CI/CD pipeline can programmatically create and manage Molecule AI organizations via API — no browser session, no manual setup, no shared credentials.

The core API is straightforward:

- `POST /cp/admin/partner-keys` — provision a new Molecule AI org for your customer or pipeline
- `DELETE /cp/admin/partner-keys/:id` — tear it down when you're done; billing stops immediately
- Keys are org-scoped by design — a compromised `mol_pk_*` key cannot touch resources outside its org

This is infrastructure-first agent orchestration. You provision the platform; your customers use it. The model is closer to Stripe's API or Twilio's account provisioning than to a SaaS seat license.

Phase 34 also delivered Tool Trace (full execution record in every A2A response) and Platform Instructions (org-level system prompt via API). Together, they give platform builders observability and governance as native platform primitives — not bolt-on integrations. [⚠️ SaaS Fed v2 — confirm with PM before referencing in published copy.]

We believe this makes Molecule AI the first agent platform with a first-class partner provisioning API.

If you're building on top of Molecule AI — or evaluating agent infrastructure for your platform — Partner API Keys GA is the milestone to look at.

Docs: https://docs.molecule.ai/api/partner-keys  
Partner program: join `#partner-program` in the Molecule AI Discord

---

## Publish notes
- Schedule for 2026-04-30 09:00 UTC (GA day)
- Pin tweet 1 for 24h after posting
- Cross-post LinkedIn within 1h of X thread
- Tag @MoleculeAI in all posts
