---
title: "Secure by Design — Molecule AI's Beta Auth Hardening Push"
date: 2026-04-20
slug: beta-auth-hardening
description: "Today's launch hardens Molecule AI's multi-tenant architecture across four dimensions: org-scoped API keys, browser session auth, tenant provisioning security, and a waitlist gate. Here's what changed and why."
tags: [security, platform, multi-tenant, auth, launch]
---

# Secure by Design — Molecule AI's Beta Auth Hardening Push

Four PR chains merged today. Together they close a week's worth of security gaps, eliminate shared secret sprawl, and put Molecule AI's beta on a production-grade auth footing. This post explains each piece and what it means for you.

---

## 1. Org-scoped API keys — full admin access without a browser

The biggest user-facing change: every Molecule AI org can now mint named, revocable bearer tokens from the Canvas Settings panel. No more copying the bootstrap `ADMIN_TOKEN` into scripts, CI pipelines, or Zapier integrations.

**What you get:**
- One key per integration — `zapier-integration`, `github-actions-deploy`, `my-claude-agent`
- Revocation is immediate: `DELETE /org/tokens/:id` returns 401 on the next request
- Every action is audited: server logs, DB `created_by`, and activity log entries carry the 8-character key prefix (`org-token:<prefix>`)
- Org keys reach every workspace in your org, including workspace sub-routes: `/workspaces`, `/workspaces/:id/channels`, `/workspaces/:id/audit`
- 10 mints per hour per IP rate limit on `POST /org/tokens` — a compromised key can't mint a flood

**The visual proof point:** Unlike CrewAI and Hermes (user-prefixed keys), a Molecule org key shows `org:abc123XY` in the admin UI — the org prefix is visible in server logs, every audit row, and the token list. Trivial correlation, full auditability.

→ [User guide: Organization API Keys](/docs/guides/org-api-keys.md)
→ [Architecture: Org API Keys](/docs/architecture/org-api-keys.md)
→ PRs: [#1105](https://github.com/Molecule-AI/molecule-core/pull/1105), [#1107](https://github.com/Molecule-AI/molecule-core/pull/1107), [#1109](https://github.com/Molecule-AI/molecule-core/pull/1109), [#1110](https://github.com/Molecule-AI/molecule-core/pull/1110)

---

## 2. Browser session auth — Canvas admins don't need bearer tokens

Canvas admins can sign in with their browser session instead of managing a bearer token. The platform verifies each Canvas admin session against your identity provider before granting access — no tokens to copy, no secrets to rotate.

When you're logged into Canvas, your session is recognized automatically across all admin-routed features. Sessions are scoped to your org: a user from a different org can't access your workspace data.

**Revocation:** session changes propagate within seconds. If an admin account is revoked in your identity provider, Canvas admin access is gone on the next page load.

**Self-hosted / local dev:** browser session auth is a SaaS feature. In self-hosted deployments, the platform uses workspace-scoped bearer tokens as the sole auth mechanism — behaviour is unchanged.

→ [Guide: Same-Origin Canvas Fetches & Session Auth](/docs/guides/same-origin-canvas-fetches.md)
→ PRs: [#1099](https://github.com/Molecule-AI/molecule-core/pull/1099), [#1100](https://github.com/Molecule-AI/molecule-core/pull/1100)

---

## 3. Tenant provisioning security — structural fixes, not policy patches

The tenant provisioning work closed several credential and isolation gaps that existed in the multi-tenant bootstrap path:

**Secrets manager:** `PutSecret` now creates the secret before any update, fixing a race where a failed intermediate step left a partial credential state.

**Boot observability:** Platform operators have improved visibility into workspace startup, making it easier to diagnose provisioning issues without external tooling.

**Cross-tenant isolation:** Tenant data boundaries are now structurally enforced at the platform level. Each org's workspaces and secrets are isolated regardless of configuration — no shared surface that could allow one tenant to read another's data.

→ Architecture docs in the control plane repo

---

## 4. Same-origin canvas fetches — simplified browser configuration

Canvas needs to reach the platform API and the admin backend to display your workspace data, org settings, and billing information. Previously this required two separate browser-configured URLs, CORS setup, and domain-level cookie configuration.

The platform now handles all Canvas requests through a single domain. You configure `NEXT_PUBLIC_PLATFORM_URL` pointing to your tenant, and Canvas reaches everything it needs from there — no additional domains, no CORS preflight requests from the browser, no extra cookie domains.

---

## 5. Beta gate + waitlist — controlled rollout for the waitlist cohort

Beta access is invitation-only. New visitors to Canvas are verified as members of an approved org before any workspace data is served — non-members see a waitlist contact form instead.

The waitlist itself is a Canvas-administered list with email hashing in audit logs (compliant with EU AI Act record-keeping requirements). Admins triage signups from an internal UI.

This is the operational surface that makes the above security work matter: the beta is invitation-only, credentials are scoped, and every admin action is auditable.

→ Control plane PRs [#145](https://github.com/Molecule-AI/molecule-controlplane/pull/145), [#148](https://github.com/Molecule-AI/molecule-controlplane/pull/148), [#150](https://github.com/Molecule-AI/molecule-controlplane/pull/150)

---

## What this means in practice

If you're already using Molecule AI as a self-hosted deployment, nothing changes today — the auth tier improvements are SaaS-only until you opt into multi-tenant mode.

If you're on the beta waitlist, you'll receive an invite. Once onboarded, your Canvas session is your admin credential. Mint org API keys for your scripts and integrations. Revoke them if anything looks wrong.

If you're evaluating Molecule AI: this launch marks the point where the platform's security posture is intentional and documented, not accumulated accident. Org keys, session auth, and tenant isolation are all covered in the architecture docs — not just the marketing claims.

→ [Quickstart](/docs/quickstart)
→ [Architecture overview](/docs/architecture/architecture)
→ [Platform API reference](/docs/api-reference)

---

*PRs #1075–#1083, #1085–#1100 (monorepo), #145–#150, #153–#169, #172–#173 (controlplane), #12 (molecule-app). Production rollout on 2026-04-20.*
