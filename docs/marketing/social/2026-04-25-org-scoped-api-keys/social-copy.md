# Org-Scoped API Keys — Social Copy
Campaign: org-scoped-api-keys | Source: PR #1105
Publish day: 2026-04-25 (Day 5)
Status: ✅ Approved by Marketing Lead — 2026-04-21

---

## Feature summary (source: PR #1105)
- Org-scoped API keys: named, revocable, audited credentials replacing the shared ADMIN_TOKEN
- Mint from Canvas UI or `POST /org/tokens`
- sha256 hash stored server-side, plaintext shown once on creation
- Prefix visible in every audit log line
- Immediate revocation — next request, key is dead
- Works across all workspaces AND workspace sub-routes
- Scoped roles (read-only, workspace-write) on the roadmap

**Angle:** "Your AI agent now has its own org-admin identity — named, revokable, audited. No more shared ADMIN_TOKEN."

---

## X (Twitter) — Primary thread (5 posts)

### Post 1 — Hook
You have 20 agents running in production.

One of them is making calls you can't trace.

That's not a hypothetical. That's what happens when you scale past
"one ADMIN_TOKEN works fine" — and it usually happens the week before
a compliance review.

Molecule AI org-scoped API keys: named, revocable, audit-attributable
credentials for every integration.

→ [blog post link]

---

### Post 2 — Problem framing
ADMIN_TOKEN works great — until it doesn't.

→ Can't rotate without downtime (10 agents use it simultaneously)
→ Can't attribute which integration made a call (no prefix in logs)
→ Can't revoke just one (one compromised token compromises everything)

Org-scoped API keys fix all three.

→ [blog post link]

---

### Post 3 — How it works (the product)
Molecule AI org API keys:

→ Mint via Canvas UI or POST /org/tokens
→ sha256 hash stored server-side, plaintext shown once
→ Prefix visible in every audit log line
→ Immediate revocation — next request, key is dead
→ Works across all workspaces AND workspace sub-routes

Rotate without downtime. Attribute every call. Revoke instantly.

→ [blog post link]

---

### Post 4 — Compliance angle
"We need to know which integration called that API endpoint."

Org-scoped API keys: every call tagged with the key's display prefix
in the audit log. Full provenance in `created_by` — which admin minted
the key, when, what it's been calling.

That's the answer your compliance team needs.

→ [blog post link]

---

### Post 5 — CTA
Org-scoped API keys are live on all Molecule AI deployments.

If you're running multi-agent infrastructure and still using a single
ADMIN_TOKEN — fix that.

→ [org API keys docs link]

---

## LinkedIn — Single post

**Title:** One ADMIN_TOKEN across your whole agent fleet is a compliance risk, not a convenience

**Body:**

At two agents, one ADMIN_TOKEN feels fine.

At twenty agents, it's a single point of failure that you can't rotate,
can't audit, and can't compartmentalize.

Molecule AI's org-scoped API keys change the model:

→ One credential per integration — "ci-deploy-bot", "devops-rev-proxy",
  not "the ADMIN_TOKEN"

→ Every API call tagged with the key's prefix in your audit logs

→ Instant revocation — one key compromised, one key revoked,
  zero downtime for other integrations

→ `created_by` provenance on every key — which admin created it,
  when, and what it can reach

The keys work across every workspace in your org — including workspace
sub-routes, not just admin endpoints.

This is the credential model that makes multi-agent infrastructure
defensible at scale.

Org-scoped API keys are available now on all Molecule AI deployments.

→ [org API keys docs link]

UTM: `?utm_source=linkedin&utm_medium=social&utm_campaign=org-scoped-api-keys`

---

## Visual Asset Requirements

1. **Canvas UI screenshot** — Org API Keys tab showing key list
   (name, prefix, created date, last used)
2. **Before/after credential model** — "ADMIN_TOKEN (single, shared,
   un-auditable)" vs "Org-scoped API keys (one per integration,
   named, revocable, attributed)"
3. **Audit log terminal output** — key prefix, workspace ID, timestamp
   in every line

---

## Campaign Notes

- **Publish day:** 2026-04-25 (Day 5)
- **Hashtags:** #AIAgents #MoleculeAI #DevOps #PlatformEngineering
- **X platform tone:** Lead with attribution — "which agent made that call?"
  resonates with developer/DevOps audience
- **LinkedIn platform tone:** Lead with compliance/risk — "one ADMIN_TOKEN
  is a single point of failure" resonates with enterprise audience
- **Key naming examples:** `ci-deploy-bot`, `devops-rev-proxy` — concrete,
  relatable for target audience
- **Self-review applied:** no timeline claims, no person names, no benchmarks
- **CTA links:** org API keys docs page — pending live URL

---

*Source: Molecule-AI/internal `marketing/devrel/social/gh-issue-pr1105-org-api-keys-launch.md`*
*Status: ✅ Approved by Marketing Lead 2026-04-21 — ready for Social Media Brand to publish once credentials are provisioned — Marketing Lead approval required before publish*
