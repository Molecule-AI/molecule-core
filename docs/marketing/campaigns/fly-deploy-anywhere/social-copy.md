# Fly Deploy Anywhere — Social Copy
Campaign: fly-deploy-anywhere | Blog PR: docs#51 (org-scoped API keys)
Publish day: TBD (Day 3–5, separate from chrome-devtools-mcp-seo)
Status: Draft — pending Marketing Lead approval

---

## X (Twitter) — Primary thread (5 posts)

### Post 1 — Hook
You have 20 agents running in production.

One of them is making calls you can't trace.

That's not a hypothetical. That's what happens when you scale past "one ADMIN_TOKEN works fine" —
and it usually happens the week before a compliance review.

Molecule AI org-scoped API keys: named, revocable, audit-attributable credentials for every integration.

---

### Post 2 — Problem framing
ADMIN_TOKEN works great — until it doesn't.

→ Can't rotate without downtime (10 agents use it simultaneously)
→ Can't attribute which integration made a call (no prefix in logs)
→ Can't revoke just one (one compromised token compromises everything)

Org-scoped API keys fix all three.

---

### Post 3 — How it works (the product)
Molecule AI org API keys:

→ Mint via Canvas UI or POST /org/tokens
→ sha256 hash stored server-side, plaintext shown once
→ Prefix visible in every audit log line
→ Immediate revocation — next request, key is dead
→ Works across all workspaces AND workspace sub-routes

Rotate without downtime. Attribute every call. Revoke instantly.

---

### Post 4 — Compliance angle
"We need to know which integration called that API endpoint."

Org-scoped API keys: every call tagged with the key's display prefix in the audit log.
Full provenance in `created_by` — which admin minted the key, when, what it's been calling.

That's the answer your compliance team needs.

---

### Post 5 — CTA
Org-scoped API keys ship today on all Molecule AI deployments.

If you're running multi-agent infrastructure and you have a single ADMIN_TOKEN —
today is a good day to fix that.

→ [link: docs blog post]

---

## LinkedIn — Single post

**Title:** One ADMIN_TOKEN across your whole agent fleet is a compliance risk, not a convenience

**Body:**

At two agents, one ADMIN_TOKEN feels fine.

At twenty agents, it's a single point of failure that you can't rotate, can't audit,
and can't compartmentalize.

Molecule AI's org-scoped API keys change the model:

→ One credential per integration — "ci-deploy-bot", "devops-rev-proxy", not "the ADMIN_TOKEN"
→ Every API call tagged with the key's prefix in your audit logs
→ Instant revocation — one key compromised, one key revoked, zero downtime for other integrations
→ `created_by` provenance on every key — which admin created it, when, and what it can reach

The keys work across every workspace in your org — including workspace sub-routes,
not just admin endpoints.

This is the credential model that makes multi-agent infrastructure defensible at scale.

Org-scoped API keys are available now on all Molecule AI deployments.

→ [link: docs blog post]

---

## Campaign notes

**Audience:** Platform engineers / DevOps (X), Security / compliance / engineering leadership (LinkedIn)
**Tone:** Direct problem-solution. No fluff. Platform engineers respond to specificity.
**Differentiation:** The rotation-without-downtime story is the primary hook — it's the most visceral ADMIN_TOKEN pain.
**Use case pairings:** X → rotation + attribution (DevOps pain), LinkedIn → compliance + provenance (security team concern)
**Hashtags:** #AgenticAI #MoleculeAI #DevOps #PlatformEngineering
**Coordination:** Do NOT post on same day as chrome-devtools-mcp-seo. Suggested spacing: Chrome DevTools MCP Day 1, Fly Day 3–5.
**Cross-link opportunity:** Chrome DevTools MCP post mentions org API keys for audit attribution — these two campaigns reinforce each other. Consider a "Part 1 / Part 2" LinkedIn thread if posted in sequence.
