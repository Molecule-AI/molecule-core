# Phase 34 — Platform Instructions Social Copy
**Campaign:** Phase 34 GA (April 30, 2026)
**Owner:** Community Manager (draft) → Social Media Brand (publish)
**Date:** 2026-04-23

---

## Thread overview

5-post thread. Hook → what it is → how to set it → governance angle → CTA.

---

## Post 1 — Hook

Your org just shipped a new compliance policy.

It applies to *one* team.

The rest of your agents need the old rules.

And you can't touch any of their configs without a six-email back-and-forth.

That's the problem Platform Instructions solves.

---

## Post 2 — What it is

Platform Instructions lets org admins set system-level rules that apply to every agent — at startup, before the agent reads its own config.

```
PUT /cp/platform-instructions
{"instructions": "Confirm before running destructive commands in prod."}
```

Org-wide. Enforced by the platform. Not overridable by workspace config.yaml.

---

## Post 3 — How it works

Two scopes:
- **Global** → every workspace in the org
- **Workspace** → one specific workspace (additive to global rules)

Rules prepend to each agent's effective system prompt. Workspace owners can't strip them out.

Use cases: compliance policies, PR etiquette, audit tagging, safety gates.

---

## Post 4 — Why it matters for teams

DevOps team ships a "no direct main commits" rule → every agent in the org enforces it.

Security team sets a "tag all prod operations" policy → agents for the payment pipeline, the data team, and the infra team all comply.

You set it once. It propagates everywhere.

---

## Post 5 — CTA

Platform Instructions is live for all Molecule AI orgs. No upgrade required.

Docs: docs.moleculesai.app/blog/platform-instructions-governance

---

**Delivery instructions for Social Media Brand:**
- Post 1-2 directly, 3-5 reply-chain under Post 1
- Tweet deck format: Thread by @moleculeai
- Alt text: "Platform Instructions API call showing PUT /cp/platform-instructions with org-level rules"
- No design partner names
- Link check: confirm `docs.moleculesai.app/blog/platform-instructions-governance` resolves before posting