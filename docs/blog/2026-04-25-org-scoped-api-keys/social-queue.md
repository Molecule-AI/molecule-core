# Org-Scoped API Keys — Social Queue
Campaign: org-api-keys-announcement | Blog: `docs/blog/2026-04-25-org-scoped-api-keys/index.md`
Publish day: 2026-04-25
Status: Draft

---

## Post Metadata
- **Blog post:** `docs/blog/2026-04-25-org-scoped-api-keys/index.md`
- **Live URL:** `https://molecule.ai/blog/org-scoped-api-keys`
- **Publish date:** 2026-04-25
- **Author:** Molecule AI
- **Tags:** security, api-keys, governance, enterprise, agentic-ai
- **PR:** molecule-core#1105 (Phase 30)

---

## X / Twitter — Thread Version (5 posts)

### Post 1 — Hook (P0: `org-scoped API keys`)
One shared ADMIN_TOKEN across your whole AI agent fleet.

Every call looks identical in your audit logs.
You can't rotate it without taking down every agent.
If one integration is compromised, every agent is compromised.

That's not a security posture. That's a blast radius waiting to happen.

Org-scoped API keys: named, revocable, audit-attributable credentials for every integration.

→ [link: blog post]

---

### Post 2 — The rotation problem (P0: `API key rotation`)
The reason most teams don't rotate secrets isn't negligence.

It's that rotating a shared ADMIN_TOKEN requires updating every agent simultaneously. In practice, keys age out.

Org-scoped API keys fix this:
→ Mint a new key
→ Update one integration
→ Revoke the old key

The other nineteen agents keep running. Rotation becomes routine.

→ [link: blog post]

---

### Post 3 — What audit attribution actually looks like
Every API call with an org-scoped key is logged with:
→ The key's name (`ci-deploy-bot`, `monitoring-agent`)
→ The workspace it ran from
→ A timestamp
→ The endpoint called

No more "which integration made that call?" questions with no answer.

→ [link: blog post]

---

### Post 4 — Key naming conventions
The audit log is only as useful as the names on the keys.

Good: `ci-deploy-bot`, `monitoring-agent`, `slack-alerts-agent`
Bad: `johnsmith`, `prod-key`, `token-v2`

Name keys after the integration. Not the person, not the team. That way the log stays readable as the team grows.

→ [link: blog post]

---

### Post 5 — CTA
One shared ADMIN_TOKEN across your whole agent fleet is a compliance risk you don't need to carry.

Org-scoped API keys: named, revocable, audit-attributable. Live now.

→ [link: blog post]

---

## X / Twitter — Single Post Version

> One shared ADMIN_TOKEN across a 20-agent fleet is a single point of failure you can't rotate without downtime and can't audit without guesswork. Org-scoped API keys: named, revocable, every call logged. Live now. https://molecule.ai/blog/org-scoped-api-keys

**Hashtags:** #AISecurity #AgenticAI #MoleculeAI #PlatformEngineering #DevOps

---

## LinkedIn — Single Post

**Title:** The credential model that makes multi-agent infrastructure survivable

**Body:**

When you run two agents, one shared ADMIN_TOKEN is fine. You know it, you can rotate it.

When you run twenty — across multiple workspaces, teams, and integrations — that same token is a liability.

You can't attribute which integration made a call. You can't rotate it without taking down every agent simultaneously. And if one integration is compromised, every agent is compromised.

Org-scoped API keys change the credential model.

Every key is named. Every call is logged. Every revocation is instant — no redeployment, no downtime for other integrations.

Here's what that looks like in practice:

**Attribution:**
```
[2026-04-25T10:42:01Z] ci-deploy-bot @ ws-staging → POST /artifacts
[2026-04-25T10:42:08Z] monitoring-agent @ ws-prod → GET /memory
```

**Rotation without downtime:**
1. Mint a new key
2. Update one integration
3. Revoke the old key
→ Other nineteen agents keep running

**Instant revocation:**
Delete a key. The next request fails. Other integrations unaffected.

Org-scoped API keys are live on all Molecule AI deployments. Open Org Settings → API Keys in Canvas.

→ Read the full post: https://molecule.ai/blog/org-scoped-api-keys

**Tags:** #AISecurity #AgenticAI #MoleculeAI #EnterpriseAI #PlatformEngineering #DevOps

---

## Campaign Notes

**Audience:** DevOps/platform engineers (X), Enterprise security/IT (LinkedIn)
**Tone:** Practical, operationally grounded. Lead with the problem (shared token risk), not the feature.
**Differentiation:** Instant revocation without redeployment + per-key audit attribution — this is what separates org-scoped keys from "named tokens."
**Use case pairings:** X → rotation problem + audit log (developer pain), LinkedIn → compliance angle + enterprise buyer concern
**Coordination:**
- Coordinate with Discord adapter announcement if same publish week
- Post on same day as blog goes live
- Monitor for Hacker News / security community threads on API key management

---

## UTM Parameters

| Source | Medium | Campaign | Content |
|---|---|---|---|
| linkedin | social | org-api-keys-announcement | post-1 |
| twitter | social | org-api-keys-announcement | thread |
| direct | organic-search | org-api-keys-announcement | (blank) |

---

## Assets Needed
- OG image: 1200×630px, dark brand theme
- Headline: "One ADMIN_TOKEN across your whole agent fleet is a compliance risk"
- Assign to: Social Media Brand

---

*Draft — 2026-04-22*
