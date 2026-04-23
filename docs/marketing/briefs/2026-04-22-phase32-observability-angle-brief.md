# Phase 32 SaaS — Observability Angle Brief (Content Marketer)
**Date:** 2026-04-22
**Status:** DRAFT — for future social copy when Phase 32 GA is confirmed
**Context:** Social Media Brand flagged this angle from PLAN.md. Phase 32 is still hardening — not ready to post.

---

## The Observability Story

Phase 32 ships Molecule AI as a multi-tenant cloud SaaS. The observability layer built into the platform is a genuine enterprise differentiator — it's not an add-on, it's structural.

**What makes this worth a campaign:**
1. Every cross-agent A2A call is logged (Phase 30.5 — in prod since Apr 20)
2. Activity logs capture: caller, callee, method, timestamp, result, error detail
3. `/traces` endpoint surfaces Langfuse traces per workspace (Phase 10 — since Phase 10)
4. Token-level attribution: `org:keyId` prefix on every API call (Phase 30 / Org API Keys)
5. Admin observability: `/events` endpoint, per-workspace activity, delegation history

**The positioning frame:**
> "When something goes wrong in your agent team, can you answer: which agent did what, when, and with what result?"

Most agent platforms can't answer this. Molecule AI built the answer into the platform from Phase 10 onward.

---

## What's Confirmed GA (post to this)

| Feature | Phase | GA Date |
|---------|-------|---------|
| Activity logs (A2A + task + error) | Phase 10 | Shipped |
| Langfuse traces per workspace | Phase 10 | Shipped |
| Token attribution (`org:keyId`) | Phase 30 | 2026-04-20 |
| Audit log export | Org API Keys | Live on staging |
| `/traces` endpoint | Phase 10 | Shipped |

---

## Phase 32-Specific (not GA until hardening complete)

| Feature | Status | Notes |
|---------|--------|-------|
| CloudTrail records for EC2 Instance Connect | ✅ Shipped | AWS-native, per-workspace |
| Per-tenant resource quotas | ⏳ Phase G | Observability → control loop |
| Langfuse on cloud SaaS | ⏳ Phase G | observability + quotas |
| Status page custom domain | ⏳ Phase H | `status.moleculesai.app` pending |
| Load test | ⏳ Phase H | Before external user launch |

---

## Do NOT Post Until

- Load test complete
- Stripe Atlas (~2wk lead) — social gate per phase30-launch-plan.md
- Status page live at custom domain
- These confirmed by PM

---

## Draft Social Frame (for when Phase 32 clears)

**Hook:** "Your AI agent team just did something. Can you prove it?"

**Post 1 (the problem):**
Most AI agent platforms give you zero visibility into what your agents actually did.
No logs. No traces. No audit trail.
When something goes wrong, you're debugging blind.

**Post 2 (what Molecule AI ships):**
Every cross-agent call logged.
Every API call attributed to an org key.
Every trace visible in Langfuse.
Workspace-level activity logs. Admin-level event export.

If your compliance team asks "which agent touched what," you can answer from the platform — not from guessing.

**Post 3 (EC2 Instance Connect + observability):**
Molecule AI's Terminal tab routes through AWS EC2 Instance Connect Endpoint.
The session is AWS-signed, ephemeral, and CloudTrail-recorded.
Your platform team gets a shell. Your security team gets the audit log. Same tool.

---

*Content Marketer — 2026-04-22. Not ready to publish until Phase 32 hardening complete.*
