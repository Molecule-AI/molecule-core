# Partner API Keys GA Preview — Social Copy
**Publish day:** 2026-04-27
**Status:** APPROVED — Marketing Lead 2026-04-23
**Campaign:** Phase 34 launch run-up

---

## X / Twitter Thread (4 tweets)

---

**Tweet 1 — Tease**

Partner API Keys (`mol_pk_*`) go GA on April 30.

Here's what you'll be able to build with them. 🧵

---

**Tweet 2 — Partner Platform Channel**

Building a platform that needs agent orchestration as a feature?

With `mol_pk_*` you can provision a full Molecule AI org for each of your customers via a single API call — no browser session, no UI fragility, no per-seat licensing negotiation.

`POST /cp/admin/partner-keys` → org is live. Hand your customer a dashboard that's already theirs, already wired up, already running agents.

Zero browser dependency. Every provisioning action is an API call.

---

**Tweet 3 — CI/CD Channel**

Running integration tests against a shared staging org? That's a shared-state problem waiting to bite you.

With Partner API Keys:
→ `POST` to create a fresh org per PR
→ Run your tests in full isolation
→ `DELETE` to tear down and stop billing

Each pipeline run gets a clean org. No cross-contamination. No manual cleanup. CI/CD-native from day one.

---

**Tweet 4 — CTA**

April 30. Partner API Keys go GA.

Docs + partner onboarding guide: docs.molecule.ai/docs/guides/partner-onboarding

Building something? Come find us in the partner Discord before launch day.

---

## LinkedIn Post (~150 words)

**Partner API Keys (`mol_pk_*`) go GA on April 30 — here's what builders should know before then.**

If you're building a platform that embeds agent orchestration as a feature, or running CI/CD pipelines that need clean test environments for every PR, Partner API Keys are the primitive you've been waiting for.

Two use cases that become straightforward on April 30:

**Partner platforms.** Provision a Molecule AI org for each of your customers via `POST /cp/admin/partner-keys` — no browser session required. Org-scoped by design, revocable in one API call. Your integration doesn't break when a UI changes.

**CI/CD automation.** Spin up an ephemeral org per pipeline run, test against real infrastructure in full isolation, and `DELETE` to tear down and stop billing when the run completes. No shared staging org. No contaminated state.

Molecule AI is the first agent platform with a first-class partner provisioning API. Docs and partner onboarding guide go live April 30.

Questions before then? Partner Discord is open now.
