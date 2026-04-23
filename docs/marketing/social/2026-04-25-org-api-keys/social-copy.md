# Social Copy — Org API Keys Launch
## 2026-04-25

**Owner:** DevRel | **Blog:** `docs/blog/2026-04-25-org-api-keys/index.md`
**Status:** ✅ Ready — publish day-of, coordinated with blog post
**Gated on:** Blog post live + CM review

---

## X / Twitter — Thread

**Post 1 — The problem (hook):**
> If your AI agent's API call fails, do you know which credential it used?

> With a shared admin token: no. One token, every integration, no attribution.
> Org API keys change that.

---

**Post 2 — What they are:**
> Every Molecule AI org key is named, scoped, and tied to an actor in the audit trail.

> Mint one for your CI pipeline. One for your ops bot. One for your agent.
> Revoke the compromised one. Leave the others running.

---

**Post 3 — The revocation speed:**
> Revoking an org key in Molecule AI marks it dead in the database.
> The partial index handles the rest — 401 on the next request, no cache TTL to wait through.

> Immediate isolation. Not eventually-consistent isolation.

---

**Post 4 — Audit trail:**
> Every key mint, every API call, every revocation — all in the audit log with the key name and actor.

> Not "some token called the API." "github-actions-deploy called DELETE /org/tokens/ki_abc123 — revoked by alice@molecule.ai at 2026-04-25T09:00:00Z."

---

**Post 5 — CTA:**
> Org API keys are live on Molecule AI Cloud.
> Settings → API Keys → New Key.
> → [moleculesai.app](https://moleculesai.app)

---

## LinkedIn — Post

**Hook:**
> Your AI agent just called your platform API. Which credential did it use?

**Body:**
> If you're using a shared admin token — you don't know. One token, every integration, no attribution, no selective revocation.

> We just shipped Org API Keys for Molecule AI Cloud. Every key is:
> - **Named** by the person who minted it (github-actions-deploy, ops-bot-prod, pm-agent)
> - **Scoped** to a specific actor in the audit trail
> - **Immediately revocable** — partial index, not cache TTL. Dead on the next request.

> Rotate one integration without rotating everything. Know exactly which key made which call. Revoke the compromised key in seconds and nothing else stops working.

> If you're running AI agents in production and using a shared token — this is the upgrade.

**CTA:** Get started at moleculesai.app

---

## Reddit — r/localllama

**Title:** Molecule AI just shipped Org API Keys — named tokens with immediate revocation and a full audit trail

```
We just shipped Org API Keys for Molecule AI Cloud.

The core problem it solves: shared admin tokens in AI agent platforms have no attribution, no selective revocation, and no lifetime management.

Every org key is:
- Named by the person who minted it
- Tied to an actor in the audit trail
- Immediately revocable — not eventually consistent

Mint one for your CI pipeline, one for your ops bot, one for your agent. Revoke the compromised one. Leave the others running.

If you're running AI agents in production and your platform uses a shared admin token — this is the upgrade you're probably building yourself and probably shouldn't have to.

Docs: moleculesai.app/blog/org-api-keys
```

---

## Hacker News

**Title:** Molecule AI Org API Keys — named, revocable, audited tokens for production AI agent platforms

```
Shipped: Org API Keys for Molecule AI Cloud.

The problem: shared admin tokens give you one credential for every integration, script, and agent — no attribution, no selective revocation, no audit trail beyond server logs.

Org API keys solve it at the platform level:
- Named keys (github-actions-deploy, ops-bot-prod, pm-agent)
- Immediate revocation via partial index — 401 on next request, no cache TTL
- Full audit trail: key name, actor, action, timestamp

If you've built something like this yourself, curious how you handled the revocation latency problem. We went with a partial index on (org_id, revoked_at) and it's been clean.
```

---

## Social Queue Status

| Channel | Status | Notes |
|---------|--------|-------|
| X thread | ✅ Ready | 5 posts, publish with blog |
| LinkedIn | ✅ Ready | Single post, full narrative |
| Reddit r/localllama | ✅ Ready | Copy-paste |
| Hacker News | ✅ Ready | Ask about revocation approach |
| Discord (team channel) | ⏳ Pending | DM to #announcements |

---
*Generated 2026-04-23 | DevRel*