# Fly.io Deploy Anywhere — Social Campaign Decision Brief
**Owner:** PMM + Marketing Lead | **Status:** DECISION REQUIRED
**Campaign:** Fly.io Deploy Anywhere social | **Post Day:** T+3 (2026-04-23+) | **Blocked on:** credentials + post date decision

---

## Context

Chrome DevTools MCP Day 1 (2026-04-21) is blocked on social API credentials. If credentials aren't provisioned today, Day 1 slides. The Fly.io Deploy Anywhere campaign was planned for Day 3+ (2026-04-23+).

Three decisions needed:
1. **Post date** — when to publish Fly.io thread
2. **Leading angle** — which dimension leads the thread
3. **Day 5 follow-up** — org-scoped API keys campaign or stay silent

---

## Decision 1: Post Date

| Option | Date | Rationale |
|--------|------|----------|
| **A** | 2026-04-23 (Day 3) | Maintains planned cadence. Tight if Chrome DevTools is late. |
| **B** | 2026-04-25 (Day 5) | Breathing room. Realistic if credentials land mid-week. |
| **C** | 2026-04-28 | Wait for Phase 32 narrative. Too late — momentum gap. |

**Recommendation: Option B (2026-04-25).** Credible if Chrome DevTools posts today or tomorrow. Maintains launch momentum without forcing a bad handoff.

---

## Decision 2: Leading Angle

| Option | Hook | Best for |
|--------|------|----------|
| **A** | Infrastructure freedom — "Three backends, one config" | Broad reach, developer audience |
| **B** | Security — "Your Fly API token never touches the tenant" | SaaS builders, enterprise |
| **C** | Indie dev — "Fly.io user? Three env vars and you're on" | Fly.io existing users |

**Recommendation: Option A (Infrastructure freedom).** Widest hook, sets up B (security) and C (indie) as follow-on posts in the thread. Anchors the campaign in the most universally relevant differentiator.

---

## Decision 3: Day 5 Follow-Up

| Option | Action |
|--------|--------|
| **A** | Post org-scoped API keys social campaign on Day 5 |
| **B** | Skip Day 5 — rest the audience, start fresh next week |
| **C** | Condense to a single LinkedIn post on Day 5, org-scoped keys later |

**Recommendation: Option A.** Org-scoped API keys social copy doesn't exist yet. Write it this week so it's ready for Day 5. The security narrative (Fly.io Day 3) sets up org API keys (Day 5) naturally — both are about credential governance.

---

## Campaign Thread Outline (Option B/Recommendation)

**Post 1 — Hook (A: Infrastructure freedom)**
> Your infrastructure choice just got decoupled from your agent platform.

**Post 2 — What's new (A: 3 backends)**
> Docker. Fly.io Machines. Control Plane API. Same agent code.

**Post 3 — Security (B: Fly.io token isolation)**
> If you're building on Fly.io, CONTAINER_BACKEND=controlplane keeps your token off the tenant.

**Post 4 — Indie dev (C: Fly.io existing users)**
> Already on Fly.io? Three env vars.

**Post 5 — CTA (A: comparison table)**
> Self-hosted → Docker. On Fly → flyio. SaaS → controlplane.

**LinkedIn — Enterprise angle (B + A)**
> Infrastructure flexibility meets enterprise security.

---

## Credentials Dependency

Both Chrome DevTools MCP (Day 1) and Fly.io (Day 3/5) require X API v2 + LinkedIn credentials. See `marketing/pmm/gh-issue-blocked-social-credentials.md`. If credentials land 2026-04-22, Day 3 is possible. If not, Day 5 is the floor.

---

*Decision brief by PMM 2026-04-21. Defaulting to B/B/A per Marketing Lead recommendation. PMM to confirm.*
