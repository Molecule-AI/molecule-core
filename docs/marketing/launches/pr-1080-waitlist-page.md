# Launch Brief: Waitlist Page with Contact Form
**PR:** [#1080](https://github.com/Molecule-AI/molecule-core/pull/1080) — `feat(canvas): /waitlist page with contact form`
**Merged:** 2026-04-20T16:47:35Z
**Owner:** PMM
**Status:** DRAFT

---

## Problem

Users whose email isn't on the beta allowlist hit a dead end after WorkOS auth redirect — no capture mechanism, no explanation, no next step. The loop wasn't closed on the unauthenticated user experience.

---

## Solution

A dedicated `/waitlist` page that captures waitlist interest with email + optional name + use-case. Soft dedup prevents spam. Privacy guard ensures client never auto-pre-fills email from URL params (regression test included).

---

## 3 Core Claims

1. **No more dead ends.** Email not on allowlist → friendly waitlist page with context, not a broken auth redirect.
2. **Capture + qualify.** Name + use-case fields let the team segment and prioritize inbound interest.
3. **Privacy by design.** Client-side privacy test ensures email is never auto-pre-filled from URL params — compliance-adjacent and trust-building.

---

## Target Developer

- Developers evaluating Molecule AI who hit the beta wall
- Indie devs and teams wanting early access
- PM/sales for waitlist segmentation

---

## CTA

"Join the waitlist → [form]" — Captures warm inbound interest for future GA outreach.

---

## Positioning Alignment

- Low-key feature, not a core positioning angle
- Secondary signal: demonstrates product care (privacy regression test = security-minded team)
- Useful as a "we're growing responsibly" proof point in growth metrics

---

## Open Questions

- Is this waitlist for self-hosted users, SaaS users, or both?
- Is there a CRM integration for the captured leads?
- Does this need a blog post or is it an infra/UX maintenance item?

---

*Not high priority for launch brief promotion. Monitor for CRM workflow integration.*
