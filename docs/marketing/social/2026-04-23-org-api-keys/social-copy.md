# Org-Scoped API Keys — Social Copy
Campaign: Org-Scoped API Keys | Source: `docs/blog/2026-04-21-org-scoped-api-keys/`
Publish day: 2026-04-23 (Day 3)
Status: Draft v1 — needs Marketing Lead approval before posting
Blog URL: `https://docs.molecule.ai/blog/org-scoped-api-keys`

---
## Campaign Angle

**Problem:** Multi-agent production systems use a shared ADMIN_TOKEN — single point of failure, no attribution, impossible to rotate without downtime.

**Solution:** Org-scoped API keys — named, revocable, per-key audit trail. One key per integration. Rotate without touching the others.

**Audience:** Platform engineers, DevOps, security-aware teams running multi-agent production systems.

---

## X (Twitter) — 3-post thread

### Post 1 — Problem framing
```
Your AI agent system has one shared API token.
It can't be rotated without downtime.
It can't be revoked for one integration without revoking all of them.
It can't tell you which agent called your API.

That's the ADMIN_TOKEN problem.
```

### Post 2 — What org-scoped keys solve
```
Org-scoped API keys solve three problems at once:

→ Rotate any key without touching the others
→ Attribute every API call to the integration that made it
→ Revoke one compromised key in one request

One key per integration. Full audit trail. Named, not guessed.
```

### Post 3 — Practical example
```
CI pipeline needs API access. Marketing tool needs API access.
Those are two different integrations.

With org-scoped keys: give each its own named credential.
Revoke the marketing one without touching CI.
Audit log shows which key called what — exactly once per request.

Same org. Isolated credentials. Zero downtime on rotation.
```

---

## X (Twitter) — Alternative single post (282 chars)
```
The difference between "works" and "production-ready" API auth:

Shared ADMIN_TOKEN → all-or-nothing
Org-scoped keys → per-integration, named, revocable, audited

Rotate CI credentials. Leave production agents untouched.
Molecule AI org-scoped keys: new standard for multi-agent teams.
```

---

## LinkedIn — Single post (~180 words)

```
If you're running more than one AI agent in production, the shared API token situation eventually becomes a problem.

It's not a theoretical risk. When your CI pipeline, your monitoring tool, and your backup agent all share one credential, rotating that credential means coordinating downtime across every integration simultaneously. That's not a rotation — it's a project.

Org-scoped API keys change this. Each integration gets its own named credential — minted, named, and revocable independently. When a key is compromised, you revoke one key, not your entire system. When you need to rotate, you mint a new one where the old one is still valid — zero downtime.

Every API call is attributed: the audit log shows the key prefix on every request, the created-by field tracks who minted it, and the full provenance is there when you need it.

Molecule AI org-scoped keys ship with Phase 30. Navigate to Settings → Org API Keys in Canvas, or use the REST API.

The token model that works for one agent doesn't scale to twenty. Org-scoped keys do.

#API #Security #AIAgents #DevOps #MoleculeAI
```

---

## Reddit — r/devops / r/AZURE / r/aws

**Title:** How Molecule AI handles API key rotation in multi-agent production systems

**Body:**
When you're running multiple AI agents and integrations — CI pipelines, monitoring tools, backup scripts — they all need API access. The naive approach: one shared token. The problem: rotate it once, coordinate downtime across every integration.

Org-scoped API keys solve this differently. Each integration gets its own named credential. Revoke one without touching the others. Rotate on schedule without coordinating downtime.

Every API call is logged with the key prefix — you can trace any call back to the specific integration that made it.

Live in Molecule AI Phase 30: [blog post with full implementation details](https://docs.molecule.ai/blog/org-scoped-api-keys)

---

## Hacker News — Show HN

**Title:** Org-scoped API keys for multi-agent production systems

```
We shipped org-scoped API keys in Molecule AI Phase 30.

The problem: multi-agent production systems with a shared ADMIN_TOKEN have no attribution, no isolation, and a rotation that requires coordinating downtime across every integration.

The fix: one named key per integration, revocable independently, full audit trail with key prefix on every request.

API:
POST /org/tokens  → mint
GET  /org/tokens  → list (prefixes, timestamps, no plaintext)
DELETE /org/tokens/:id → revoke, takes effect immediately

The audit log: org-token:mole_a1b2 POST /workspaces/ws_abc123/channels 200 12ms

Key prefix visible on every call. Revocation takes effect on the next request.
No grace period. No downtime on rotation.
```

---

## Image suggestions

| Post | Image |
|---|---|
| X Post 1 (Problem) | Dark card: "ADMIN_TOKEN" crossed out, "org-scoped key" below |
| X Post 2 (Solution) | Comparison table: shared token vs org-scoped — rotate / attribute / revoke |
| X Post 3 (Example) | Terminal screenshot: org API keys list view in Canvas |
| LinkedIn | Quote card: "The token model that works for one agent doesn't scale to twenty." |
| Reddit | Diagram: shared token → org-scoped keys (isolation per integration) |
| HN | No image — technical, text-only format |

---

## Coordination notes

- **Day 2 (2026-04-22):** Discord Adapter — Reddit + HN copy from GH #1383
- **Day 3 (2026-04-23):** Org-Scoped API Keys — this file
- **Day 4 (2026-04-24):** EC2 Console Output — approved copy in `docs/marketing/social/2026-04-24-ec2-console-output/social-copy.md`
- **Day 5 (2026-04-25):** Cloudflare Artifacts — approved, coordinate with Marketing Lead

*Draft by Content Marketer 2026-04-22 — awaiting Marketing Lead approval before posting*