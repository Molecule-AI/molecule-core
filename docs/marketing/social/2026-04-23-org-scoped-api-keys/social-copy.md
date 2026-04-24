# Social Queue — 2026-04-23
# Campaign: Org-Scoped API Keys launch
# Owner: Community Manager (Reddit + HN)
# Social Media Brand: X thread + LinkedIn

## STATUS
- :x: BLOG URL 404 — using GitHub source as primary link
- Reddit r/LocalLLaMA: COPY READY ✅ — BLOCKED (no Reddit credentials)
- Reddit r/MachineLearning: COPY READY ✅ — BLOCKED (no Reddit credentials)
- Hacker News: COPY READY ✅ — BLOCKED (no browser capability)
- X thread (5 posts): COPY READY ✅ — PARTIALLY BLOCKED (user tokens present; X_API_KEY + X_API_SECRET still needed per issue #1865)
- LinkedIn: COPY READY ✅ — BLOCKED (no LinkedIn OAuth access token)
- Blog URL (PRIMARY — GitHub source): https://github.com/Molecule-AI/molecule-core/tree/main/docs/blog/2026-04-21-org-scoped-api-keys
- moleculesai.app/blog/org-scoped-api-keys: returns 404 (not live)

---

## Reddit r/LocalLLaMA

**Title:** Org-scoped API keys for Molecule AI — named tokens you can revoke instantly without downtime

**Body:**
Every production multi-agent system eventually hits the same wall: one shared `ADMIN_TOKEN` that you can't rotate without coordinating downtime across every agent and integration.

Molecule AI just shipped org-scoped API keys. Key properties:

→ Named at creation (e.g. `ci-deploy-bot`, `devops-rev-proxy`) — you know exactly what each key is for
→ SHA-256 hash stored server-side — plaintext shown once, never again
→ Works across every workspace in your org, including sub-routes like `/workspaces/:id/secrets`
→ Immediate revocation — revoke a key and it's dead on the next request, no grace period
→ Full audit trail: every call logged with the key prefix so you know exactly which integration made it

The rotation story: mint a new key, update your integration, revoke the old one. Both keys are valid simultaneously during the window — zero downtime.

Real-world use case: your CI pipeline gets `ci-deploy-bot`, your observability stack gets `devops-rev-proxy`. Compromise one, revoke one. The other keeps running.

Docs and API reference: https://github.com/Molecule-AI/molecule-core/tree/main/docs/blog/2026-04-21-org-scoped-api-keys

*Note: moleculesai.app/blog/org-scoped-api-keys returns 404 — GitHub source is canonical until blog is live.*

---

## Reddit r/MachineLearning

**Title:** Molecule AI org API keys: named, revocable tokens for multi-agent production systems

**Body:**
The gap between "works in prototype" and "safe in production" for AI agent platforms is usually credential management.

Molecule AI just shipped org-scoped API keys — long-lived credentials that live at the organization level and reach every workspace. Each key is:

- Named (you pick the identifier: `data-pipeline`, `backup-script`, `monitoring-integration`)
- Revocable with zero downtime (mint new key → update integration → revoke old; both valid during the window)
- Auditable (every request logged with the key prefix + created-by attribution)

This is what makes it safe to give each integration its own credential. One compromised key doesn't cascade. One rotation doesn't require coordinating with every other agent in the system.

Authentication priority tier: org API keys sit above workspace tokens and `ADMIN_TOKEN` in the auth hierarchy — they're the primary path for service integrations.

API example (mint via REST):

```bash
curl -X POST https://platform.moleculeai.ai/org/tokens \
  -H "Authorization: Bearer <session-token>" \
  -d '{"name": "ci-deploy-bot"}'
```

Docs: https://github.com/Molecule-AI/molecule-core/tree/main/docs/blog/2026-04-21-org-scoped-api-keys

---

## Hacker News

**Title:** Molecule AI — org-scoped API keys for multi-agent production systems

**Body:**
Molecule AI just shipped org-scoped API keys — a credential management layer for teams running multiple AI agents in production.

The core problem: a single shared `ADMIN_TOKEN` scales poorly. Rotation requires downtime across every agent using it. There's no call attribution. A compromise is total.

Org-scoped keys solve this at the org level:

→ Each key is named at creation (e.g. `ci-deploy-bot`) — scoped to exactly what it's for
→ Immediate revocation — delete a key and it's dead on the next request, no grace period
→ Works across all workspaces in the org, including sub-routes like `/workspaces/:id/secrets`
→ Full audit trail: key prefix logged on every call, created-by stored at mint time

Rotation without downtime: mint a new key, update your integration, revoke the old one. Both valid simultaneously during the transition window.

The keys sit above `ADMIN_TOKEN` in the auth priority hierarchy — primary path for service integrations, `ADMIN_TOKEN` becomes break-glass.

Code and API docs: https://github.com/Molecule-AI/molecule-core/tree/main/docs/blog/2026-04-21-org-scoped-api-keys

---

## X Thread — Social Media Brand

Post 1 (Hook — the problem):
Your AI platform has one API token.
Ten agents use it.
To rotate it, you need to coordinate all ten at once — or accept a window where some of them can't authenticate.
Org-scoped keys fix this.

Post 2 (The solution — what keys are):
Molecule AI just shipped named, revocable API keys at the org level.
Each integration gets its own credential.
ci-deploy-bot. devops-rev-proxy. data-pipeline.
One compromised → revoke one. Zero cascade.

Post 3 (Rotation without downtime):
The rotation workflow:
1. Mint a new key
2. Update your integration
3. Revoke the old key
Both keys valid simultaneously during step 2.
Zero downtime. Full audit trail from day one.

Post 4 (Audit trail):
Every request authenticated with an org API key is logged with the key prefix:
org-token:mole_a1b2 POST /workspaces/ws_abc123/channels 200 12ms
You know exactly which integration made every call.
Revoke a compromised key → it's dead on the next request.

Post 5 (CTA):
Org-scoped API keys for Molecule AI are live.
Each key: named, revocable, auditable, works across every workspace in your org.
Rotate without downtime. Trace every call back to the key that made it.
Start at Settings → Org API Keys in Canvas.
https://github.com/Molecule-AI/molecule-core/tree/main/docs/blog/2026-04-21-org-scoped-api-keys

---

## LinkedIn — Social Media Brand

**Title:** We gave every AI agent in our org its own API key. Here's what changed.

When you run one AI agent, credential management is simple: one token, one place to rotate it.

When you run ten — CI pipelines, monitoring integrations, backup scripts, devops proxies — a single shared token becomes a production liability. Rotate it and you coordinate downtime across every integration simultaneously. One compromise and everything is exposed.

We just shipped org-scoped API keys for Molecule AI. Each key is:

→ **Named at creation.** `ci-deploy-bot`, `devops-rev-proxy`, `data-pipeline` — each integration has a clear identity from day one.

→ **Revocable immediately.** One API call and the key is dead on the very next request. No grace period, no cooldown. A compromised CI pipeline key doesn't cascade to your observability stack.

→ **Rotatable without downtime.** Mint a new key, update your integration, revoke the old one. Both keys valid simultaneously during the window. No coordination required.

→ **Auditable.** Every call is logged with the key prefix. You know exactly which integration made every request — and who created that key.

→ **Org-wide.** Works across every workspace in your org, including sub-routes like `/workspaces/:id/secrets` — not just admin-surface endpoints.

The auth priority hierarchy: org API keys sit above workspace tokens and `ADMIN_TOKEN`. They're the primary path for service integrations. `ADMIN_TOKEN` becomes break-glass for operators and CLI tooling.

If you're running AI agents in production, credential hygiene matters as much as it does for any other service. Org-scoped keys make that tractable at scale.

→ [Canvas → Settings → Org API Keys](https://canvas.moleculeai.ai)
→ [Docs](https://github.com/Molecule-AI/molecule-core/tree/main/docs/blog/2026-04-21-org-scoped-api-keys)

#AIAgents #PlatformEngineering #APISecurity #MoleculeAI #DevOps

---

*Queue file prepared by Community Manager — 2026-04-23. All copy self-reviewed. Credential-blocked pending infra restore.*
