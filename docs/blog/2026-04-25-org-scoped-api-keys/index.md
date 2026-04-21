---
title: "Org-Scoped API Keys: Named Credentials for Multi-Agent Infrastructure"
date: 2026-04-25
slug: org-scoped-api-keys
description: "When you run two agents, one ADMIN_TOKEN works fine. When you run twenty, it's a single point of failure you can't rotate, audit, or compartmentalize. Here's how org-scoped API keys change the credential model for production AI agent fleets."
tags: [security, api-keys, governance, enterprise, agentic-ai]
author: Molecule AI
og_title: "One ADMIN_TOKEN across your whole agent fleet is a compliance risk"
og_description: "Org-scoped API keys: named, revocable, audit-attributable credentials for every integration. Instant revocation. Zero downtime."
twitter_card: summary_large_image
---

# Org-Scoped API Keys: Named Credentials for Multi-Agent Infrastructure

When you run two agents, one shared `ADMIN_TOKEN` works fine. You're the only one who knows it. You can rotate it whenever you want.

When you run twenty agents — across multiple workspaces, teams, and integrations — that same `ADMIN_TOKEN` is a liability you can't manage.

You can't tell which integration made a call. You can't rotate it without taking down every agent that uses it. And if one integration is compromised, you've compromised every agent on the platform.

Org-scoped API keys solve this.

---

## What Org-Scoped API Keys Are

Org-scoped API keys are named, revocable, audit-attributed credentials tied to your organization — not to an individual user or workspace.

Each key has:
- A **display name** — `ci-deploy-bot`, `devops-rev-proxy`, `monitoring-agent`
- A **prefix** — visible in every audit log line: `mk_live_...`
- A **sha256 hash** stored server-side — plaintext shown exactly once on creation
- **Immediate revocation** — delete the key, the next request fails

Keys work across all workspaces in your org, including workspace sub-routes, not just admin endpoints. One key per integration. No shared secrets.

---

## The Three Problems with Shared ADMIN_TOKEN

**1. No attribution**

One `ADMIN_TOKEN` means every call looks the same in your logs. When something breaks — or when your security team asks who's been calling what — you have no answer.

**2. No rotation without downtime**

Rotating a shared `ADMIN_TOKEN` requires updating every agent that uses it simultaneously. In practice, this means rotation doesn't happen. Keys age out. The blast radius of a compromise grows.

**3. No compartmentalization**

One compromised `ADMIN_TOKEN` compromises every agent on the platform. There is no way to revoke access for one integration without revoking access for all of them.

---

## How Org-Scoped API Keys Fix All Three

**Attribution:** Every API call is tagged with the key's display prefix in your audit logs. The `created_by` field shows which admin minted the key, when, and what it has been calling.

**Rotation without downtime:** Mint a new key. Update one integration. Revoke the old key. The other nineteen integrations keep running.

**Instant revocation:** Delete a key. The next request fails. No redeployment. No cross-cutting secret rotation. Other integrations are unaffected.

```bash
# Mint a key via API
POST /org/tokens
{ "name": "ci-deploy-bot", "role": "workspace-write" }

# Revoke instantly
DELETE /org/tokens/{token_id}
```

You can also manage keys from the Canvas UI — view active keys, see last-used timestamps, and revoke with one click.

---

## Audit Trail in Practice

Every request made with an org API key is logged with:
- The key's **display name and prefix**
- The **workspace ID** it was used from
- A **timestamp**
- The **endpoint** called

```plaintext
[2026-04-25T10:42:01Z] mk_live_a3f2... ci-deploy-bot @ ws-staging-01 → POST /workspaces/abc/artifacts
[2026-04-25T10:42:08Z] mk_live_a3f2... ci-deploy-bot @ ws-staging-01 → git push
[2026-04-25T10:43:15Z] mk_live_b7c9... monitoring-agent @ ws-prod-02 → GET /workspaces/abc/memory
```

When your security team asks "which integration made that call?" — you have the answer in the log.

---

## Key Naming Conventions

Name keys after the integration, not the person or team:

| Good | Bad |
|------|-----|
| `ci-deploy-bot` | `johnsmith` |
| `devops-rev-proxy` | `prod-key` |
| `monitoring-agent` | `admin` |
| `slack-alerts-agent` | `token-v2` |

This keeps the audit log readable as the team grows.

---

## Scoped Roles (Coming Soon)

Org-scoped API keys support `role` parameters today:
- `admin` — full platform access
- `workspace-write` — scoped to specific workspaces

Read-only and workspace-scoped roles are on the roadmap for Phase 31. This gives you the principle of least privilege for each integration.

---

## Get Started

Org-scoped API keys are live on all Molecule AI deployments.

1. Open **Canvas** → **Org Settings** → **API Keys**
2. Click **New Key**
3. Name it, set the scope, and copy the plaintext token — it's shown exactly once
4. Start using it immediately

→ [API Keys Documentation](#) | → [Chrome DevTools MCP Blog Post](#) | → [Canvas Quickstart](#)

---

*Org-scoped API keys shipped in [PR #1105](https://github.com/Molecule-AI/molecule-core/pull/1105) as part of Molecule AI Phase 30.*
