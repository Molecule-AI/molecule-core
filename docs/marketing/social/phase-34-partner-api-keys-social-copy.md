# Phase 34 — Partner API Keys Social Copy
**Campaign:** Phase 34 GA (April 30, 2026)
**Owner:** Community Manager (draft) → Social Media Brand (publish)
**Date:** 2026-04-23

---

## Thread overview

5-post thread. Hook → what it is → how it works → why it matters → CTA.

---

## Post 1 — Hook

You're building a CI/CD pipeline.

You need a test Molecule AI org — spin up, run your test suite, tear it down.

Today that means: create account, verify email, set up org, run tests, manually delete.

Partner API Keys let you do it in a curl call.

---

## Post 2 — What it is

Partner API Keys (`mol_pk_*`) let you programmatically create and manage Molecule AI orgs via API — no browser, no manual handoff.

```
POST /cp/orgs
DELETE /cp/orgs/:slug
```

Scoped keys that can't escape their boundary. Revocable. Rate-limited (60 req/min default).

---

## Post 3 — How it works

```bash
# Spin up ephemeral test org
curl -X POST https://api.moleculesai.app/cp/orgs \
  -H "Authorization: Bearer mol_pk_live_YOUR_KEY" \
  -d '{"slug": "ci-test-$(date +%s)", "plan": "starter"}'

# Run tests → billing stops when you DELETE
curl -X DELETE https://api.moleculesai.app/cp/orgs/ci-test-1234567890 \
  -H "Authorization: Bearer mol_pk_live_YOUR_KEY"
```

No email verification. No browser session. Just API calls.

---

## Post 4 — What you can build

- **Marketplace integrations** — provision Molecule AI orgs from your platform's admin dashboard
- **CI/CD ephemeral test orgs** — one org per pipeline run, full teardown
- **Internal tooling** — spin up orgs for new products or teams without IT involvement

Keys are scoped to the orgs they're authorized for. Revocation is immediate — no grace period.

---

## Post 5 — CTA

Partner API Keys (`mol_pk_*`) GA April 30, 2026.

Early access for partners with a concrete integration use case — reach out via GitHub Discussions or Discord.

Docs: docs.moleculesai.app/blog/partner-api-keys

---

**Delivery instructions for Social Media Brand:**
- Post 1-2 directly, 3-5 reply-chain under Post 1
- Tweet deck format: Thread by @moleculeai
- Alt text: "cURL snippet showing Partner API Key org creation and deletion"
- Do not say "available now" — GA April 30
- No design partner names
- Link check: confirm `docs.moleculesai.app/blog/partner-api-keys` resolves before posting