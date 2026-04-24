# Org-Scoped API Keys — Social Copy
Campaign: org-api-keys | Blog PR: docs#51 | Issue: molecule-ai/internal#6
Publish day: TBD — post Phase 30 blog launch (do not same-day as chrome-devtools-mcp-seo)
Status: Ready for scheduling — post Chrome DevTools MCP launch (Day 2 slot)

---

## X (Twitter) — Primary thread (5 posts)

### Post 1 — Hook

Your CI pipeline calls your agent API.
Your Zapier integration calls your agent API.
Your monitoring tool calls your agent API.

They all use the same token.
That's a problem — before you even ship.

Molecule AI org-scoped API keys: one credential per integration. Named, revocable, audit-attributable.

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
→ 8-character prefix visible in every audit log line
→ Immediate revocation — next request, key is dead
→ Works across all workspaces AND workspace sub-routes

Rotate without downtime. Attribute every call. Revoke instantly.

---

### Post 4 — Compliance angle

"We need to know which integration called that API endpoint."

Org-scoped API keys: every call tagged with the key's 8-char prefix in the audit log.
Full provenance in `created_by` — which admin minted the key, when, what it's been calling.

That's the answer your compliance team needs.

---

### Post 5 — CTA

Org-scoped API keys ship today on all Molecule AI deployments.

If you're running multi-agent infrastructure and you have a single ADMIN_TOKEN —
today is a good day to fix that.

→ https://docs.moleculeai.ai/blog/org-scoped-api-keys
→ https://canvas.moleculeai.ai (Settings → Org API Keys)

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

→ https://docs.moleculeai.ai/blog/org-scoped-api-keys

---

## Campaign notes

**Audience:** Platform engineers / DevOps (X), Security / compliance / engineering leadership (LinkedIn)
**Tone:** Direct problem-solution. No fluff. Platform engineers respond to specificity.
**Differentiation:** The rotation-without-downtime story is the primary hook — it's the most visceral ADMIN_TOKEN pain.
**Use case pairings:** X → rotation + attribution (DevOps pain), LinkedIn → compliance + provenance (security team concern)
**Hashtags:** #AgenticAI #MoleculeAI #DevOps #PlatformEngineering #Security
**Coordination:** Do NOT post on same day as chrome-devtools-mcp-seo. Suggested spacing: Chrome DevTools MCP Day 1, Org API Keys Day 2, Fly Day 3–5.
**CTA links (confirmed 2026-04-21):**
- Docs: https://docs.moleculeai.ai/blog/org-scoped-api-keys
- Canvas → Org API Keys: https://canvas.moleculeai.ai (Settings → Org API Keys)
**Visual assets (✅ generated 2026-04-21):**
- `org-api-keys-canvas-ui.png` — Canvas Org API Keys UI mockup
- `org-api-keys-before-after.png` — Before/after credential model
- `org-api-keys-audit-log.png` — Audit log terminal output
