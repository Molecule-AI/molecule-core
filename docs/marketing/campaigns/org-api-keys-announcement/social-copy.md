# Org-Scoped API Keys — Social Copy
Campaign: org-api-keys | Blog: `docs/blog/2026-04-21-org-scoped-api-keys/`
Slug: `org-scoped-api-keys`
Publish day: Day 3 (2026-04-23) — coordinate with Social Media Brand
Assets: OG image at `docs/assets/blog/org-scoped-api-keys-og.png` (1200x630)

---
**NOTE:** Copy ready for human social media execution. X credentials blocked in all agent workspaces.

---

## X (Twitter) — Primary thread (5 posts)

### Post 1 — Hook
Your AI platform has one shared API key.

Every CI pipeline, every webhook integration, every automation tool — all of them using the same credential.

Rotate it: coordinate with every team. Don't rotate it: one leaked key is a full breach.

Org-scoped API keys solve this. Each integration gets its own named key. Revoke one without touching the rest.

→ https://docs.molecule.ai/blog/org-scoped-api-keys

---

### Post 2 — The ADMIN_TOKEN problem
The standard AI platform setup:

One `ADMIN_TOKEN` in an env var. Shared across integrations. Nobody's sure who has it. Rotation requires coordinating every team that uses it.

This is the production security gap nobody talks about.

Org-scoped API keys: every integration gets a named, scoped key. One revoke command. No coordination required.

→ https://docs.molecule.ai/blog/org-scoped-api-keys

---

### Post 3 — Audit trail
Every org key request carries its key prefix in the audit log.

`org:keyId` on every call. Which integration. Which call. What result.

When your compliance team asks "which pipeline touched what," you can answer from the log — not by guessing.

→ https://docs.molecule.ai/blog/org-scoped-api-keys

---

### Post 4 — How it works
Molecule AI org keys:

→ Mint from Canvas (Settings → Org API Keys) or via API
→ Label every key — know what *zapier* is doing vs *ci-bot*
→ Revoke individually, instantly
→ Full org scope — all workspaces, channels, secrets, templates
→ Rate-limited minting (10/hr/IP)

No shared `ADMIN_TOKEN`. No blast-radius on rotation. Audit trail on every call.

→ https://docs.molecule.ai/blog/org-scoped-api-keys

---

### Post 5 — CTA
Every integration should have its own key.

Your CI pipeline. Your Zapier hook. Your monitoring pipeline. Your Slack bot.

Org-scoped API keys ship today. Each integration gets a named key. Revocation is instant. The audit trail shows what each key did.

→ https://docs.molecule.ai/blog/org-scoped-api-keys

---

## LinkedIn — Single post

**Title:** The single-token API security model doesn't scale. Here's what to do instead.

**Body:**

Every AI agent platform starts with the same credential model: one shared `ADMIN_TOKEN` in an environment variable. It's the fastest way to get up and running. It's also the fastest way to create a single point of failure that nobody wants to touch.

Here's the problem: when that one token is shared across every integration — CI pipelines, webhook handlers, automation tools, monitoring scripts — rotation becomes a coordination event. Teams delay it. The token stays live. And if it leaks, every integration is exposed simultaneously.

The org-scoped API keys model inverts this. Instead of one shared token, every integration gets its own named key, scoped to the org, revocable individually.

What this looks like in practice:

→ **Named keys** — label every key by integration. You can tell *ci-bot* from *zapier* from *monitoring* at a glance.
→ **Instant revocation** — revoke one compromised key without touching any other integration.
→ **Audit trail** — every request carries `org:keyId` prefix. Which integration. Which call. What it did. All visible in the log.
→ **Full org scope** — keys manage all workspaces, channels, secrets, templates, and approvals.
→ **Rate-limited minting** — 10 mints per hour per IP, so a compromised session can't mint unlimited keys.

Org-scoped API keys are live now. If you're running any AI agent platform with a single shared token, this is the upgrade worth making before you have a rotation you can't coordinate.

→ [Read the docs](https://docs.molecule.ai/blog/org-scoped-api-keys)
→ [Org API Keys setup guide](https://docs.molecule.ai/docs/guides/org-api-keys)

---

## Campaign notes

**Audience:** Platform engineers (X), DevOps / security leads (LinkedIn)
**Tone:** Direct, operational. Lead with the single-token risk — every platform engineer with a production deployment will feel this pain.
**Differentiation:** Named keys + instant revocation + audit trail. Not just "more keys" — the governance layer around them.
**Suggested image:** `docs/assets/blog/org-scoped-api-keys-og.png` (1200x630, green accent on dark theme)
**Hashtags:** #AIAgents #AgenticAI #MoleculeAI #PlatformEngineering #DevOps #Security
**Coordination:** Day 3 of Phase 30 launch (after Chrome DevTools MCP Day 1 and Discord Adapter Day 2). Coordinate with Social Media Brand queue.
**Social Media Brand status:** Copy ready for manual execution by a human with X/LinkedIn access.
