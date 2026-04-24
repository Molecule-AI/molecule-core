# Social Queue — 2026-04-25 (Friday)

## Queue Status: ALL QUEUED — Phase 30 continuation (Day 5 of 5)

---

### Post 1 — Hook
**Platform:** X/Twitter
**Status:** QUEUED
**Post Text:**
> You have 20 agents running in production.
>
> One of them is making calls you can't trace.
>
> That's not a hypothetical. That's what happens when you scale past "one ADMIN_TOKEN works fine" — and it usually happens the week before a compliance review.
>
> #AIagents #MoleculeAI #DevOps #A2A

**Hashtags:** #AIagents #MoleculeAI #DevOps #A2A #AgenticAI #PlatformEngineering

**Media Notes:** Simple text card — dark background, high contrast. Lead with the number, not the product.

**Optimal Posting Time:** 15:00 UTC (11 AM ET / 8 AM PT — US afternoon)

**Campaign Tag:** phase-30-org-api-keys

---

### Post 2 — Problem framing
**Platform:** X/Twitter
**Status:** QUEUED
**Post Text:**
> ADMIN_TOKEN works great — until it doesn't.
>
> → Can't rotate without downtime (10 agents use it simultaneously)
> → Can't attribute which integration made a call (no prefix in logs)
> → Can't revoke just one (one compromised key compromises everything)
>
> Org-scoped API keys fix all three.
>
> #DevOps #MoleculeAI #AIagents #Security

**Hashtags:** #DevOps #MoleculeAI #AIagents #Security #AgenticAI #PlatformEngineering

**Media Notes:** Three-item problem card — clean dark theme, bullet list format.

**Optimal Posting Time:** 16:00 UTC (12 PM ET / 9 AM PT — same day stagger)

**Campaign Tag:** phase-30-org-api-keys

---

### Post 3 — How it works
**Platform:** X/Twitter
**Status:** QUEUED
**Post Text:**
> Molecule AI org API keys:
>
> → Mint via Canvas UI or POST /org/tokens
> → sha256 hash stored server-side, plaintext shown once
> → Prefix visible in every audit log line
> → Immediate revocation — next request, key is dead
> → Works across all workspaces AND workspace sub-routes
>
> Rotate without downtime. Attribute every call. Revoke instantly.
>
> #MoleculeAI #DevOps #AIagents

**Hashtags:** #MoleculeAI #DevOps #AIagents #AgenticAI #PlatformEngineering

**Media Notes:** Feature list card — dark background, check marks, brand accent color.

**Optimal Posting Time:** 18:00 UTC (2 PM ET / 11 AM PT — EU evening peak)

**Campaign Tag:** phase-30-org-api-keys

---

### Post 4 — Compliance angle
**Platform:** X/Twitter
**Status:** QUEUED
**Post Text:**
> "We need to know which integration called that API endpoint."
>
> Org-scoped API keys: every call tagged with the key's display prefix in the audit log. Full provenance in created_by — which admin minted the key, when, what it's been calling.
>
> That's the answer your compliance team needs.
>
> #AIagents #MoleculeAI #Compliance #EnterpriseSecurity

**Hashtags:** #AIagents #MoleculeAI #Compliance #EnterpriseSecurity #DevOps #PlatformEngineering

**Media Notes:** Quote card — pull the compliance question as the headline, answer as the body.

**Optimal Posting Time:** 17:00 UTC (1 PM ET / 10 AM PT — same day stagger)

**Campaign Tag:** phase-30-org-api-keys

---

### Post 5 — CTA / LinkedIn
**Platform:** LinkedIn
**Status:** QUEUED
**Post Text:**
> At two agents, one ADMIN_TOKEN feels fine.
>
> At twenty agents, it's a single point of failure that you can't rotate, can't audit, and can't compartmentalize.
>
> Molecule AI's org-scoped API keys change the model:
>
> → One credential per integration — "ci-deploy-bot", "devops-rev-proxy", not "the ADMIN_TOKEN"
>
> → Every API call tagged with the key's prefix in your audit logs
>
> → Instant revocation — one key compromised, one key revoked, zero downtime for other integrations
>
> → created_by provenance on every key — which admin created it, when, and what it can reach
>
> The keys work across every workspace in your org — including workspace sub-routes, not just admin endpoints.
>
> This is the credential model that makes multi-agent infrastructure defensible at scale.
>
> Org-scoped API keys are available now on all Molecule AI deployments.
>
> → docs.molecule.ai/blog/org-scoped-api-keys
>
> #AIagents #MoleculeAI #DevOps #Compliance #EnterpriseSecurity #PlatformEngineering

**Hashtags:** #AIagents #MoleculeAI #DevOps #Compliance #EnterpriseSecurity #PlatformEngineering

**Media Notes:** Professional post card — lead with the compliance/risk framing on LinkedIn. Clean dark theme.

**Optimal Posting Time:** 15:00 UTC (11 AM ET / 8 AM PT — LinkedIn peak engagement)

**Campaign Tag:** phase-30-org-api-keys

---

*Generated: 2026-04-24 | Content Marketer | Phase 30 Day 5*
*Source: PR #1105 | Org-scoped API Keys in Canvas*
*Blog: docs.molecule.ai/blog/org-scoped-api-keys*
*Previous Day 4: EC2 Console Output | Phase 30 complete after today*
