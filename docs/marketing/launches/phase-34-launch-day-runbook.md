# Phase 34 — April 30, 2026 Launch Day Runbook

**GA Date:** April 30, 2026  
**Owner:** Marketing Lead + Community Manager  
**Status:** DRAFT — PM/PMM must confirm GA vs Beta label before final go/no-go

---

## Pre-Launch Gate (April 28–29)

All items must be ✅ before posting anything on April 30.

### Must-Have (blocking)

- [ ] **GA vs Beta label confirmed by PM/PMM** — `docs/marketing/briefs/phase34-ga-vs-beta-conflict.md`. If Beta: update HN post, Reddit post, GA-day social copy (Apr 30), community announcement, partner-api-keys blog, and partner onboarding guide before any posting.
- [ ] **`partner-api-keys` og_image present** — `docs/blog/2026-04-23-partner-api-keys/` must have a resolved og_image before April 30 (SEO P0 blocker — Social Media Brand assigned).
- [ ] **PR #1860 closed** — `docs/phase34-community-launch` branch has wrong schema/alpha label in Reddit/HN posts. Must be closed or blocked before any Phase 34 content goes live.
- [ ] **`marketing/phase-34-launch-prep` PR merged to staging** — 35+ commits of Phase 34 launch content must be in staging before April 30.
- [ ] **Partner key rate limits confirmed** — `[RATE LIMIT TBD]` placeholder in `docs/marketing/launches/partner-onboarding-guide.md` must be replaced with actual value before the guide goes public.
- [ ] **X API credentials provisioned** — `X_API_KEY` + `X_API_SECRET` from mol-ops (#1865). Social Media Brand cannot post to X without these.

### Should-Have (non-blocking but fix before publish)

- [ ] SaaS Fed v2 battlecard — PM to confirm what shipped; parked at `docs/marketing/briefs/saas-fed-v2-what-shipped.md`
- [ ] Design partner name — "Acme Corp" placeholder in DevRel demo at `docs/devrel/phase-34-partner-api-keys-demo.md`

---

## April 30 — Hour-by-Hour Launch Sequence

### T-2h (pre-launch prep)

1. **Confirm APIs are live** — `curl -s https://api.molecule.ai/cp/platform-instructions -H "Authorization: Bearer $TEST_TOKEN"` should return 200. `POST /cp/admin/partner-keys` should be accepting requests.
2. **Confirm blog posts are indexed** — verify the four Phase 34 canonical posts resolve at docs.molecule.ai:
   - `/blog/ai-agent-observability-without-overhead`
   - `/blog/tool-trace-platform-instructions`
   - `/blog/partner-api-keys`
   - `/blog/platform-instructions-governance`
3. **Discord pre-warm** — pin Phase 34 FAQ in `#faq`: `docs/marketing/launches/phase-34-community-faq.md`
4. **#partner-program channel** — post early-access note for Partner API Keys waitlist

### T-0 (launch posts, in order)

**Step 1 — Community announcement** (Discord `#announcements` + GitHub Discussions)  
Source: `docs/marketing/launches/phase-34-community-announcement.md`

**Step 2 — Reddit posts** (r/MachineLearning, r/LocalLLaMA, r/artificial)  
Source: `docs/marketing/launches/phase-34-reddit-post.md`  
⚠️ Use the version in `docs/marketing/launches/` — NOT anything from `docs/phase34-community-launch` branch (wrong schema)

**Step 3 — HN Show HN**  
Source: `docs/marketing/launches/phase-34-hn-show-hn.md`  
Post manually at news.ycombinator.com — title: "Show HN: Molecule AI – every agent tool call now logged in A2A response (no SDK, GA today)"

**Step 4 — X/Twitter thread** (4 posts)  
Source: `docs/marketing/social/2026-04-30-phase-34-ga-launch/social-copy.md` — X thread section  
⚠️ Requires X_API_KEY + X_API_SECRET (mol-ops #1865)

**Step 5 — LinkedIn post**  
Source: `docs/marketing/social/2026-04-30-phase-34-ga-launch/social-copy.md` — LinkedIn section

**Step 6 — Partner onboarding guide live** — ensure `docs/marketing/launches/partner-onboarding-guide.md` is published/linked from docs.molecule.ai/api/partner-keys

### T+1h (monitoring)

- Monitor `#bug-reports` — canned responses at `docs/marketing/launches/phase-34-community-response-queue.md`
- Monitor Reddit/HN comments — HN objection FAQ is in `docs/marketing/launches/phase-34-hn-show-hn.md` (objections section)
- Monitor `#partner-program` — first Partner API Key questions expected within hours

### T+24h

- DevRel talk-track performance: did the `#show-your-work` demo land? Source: `docs/devrel/talks/tool-trace-platform-instructions-talk-track.md`
- Pull any new GitHub issues or Discussions opened against Phase 34 features — route to DevRel

---

## Key File Index

| Asset | File | Status |
|-------|------|--------|
| GA-day social copy | `docs/marketing/social/2026-04-30-phase-34-ga-launch/social-copy.md` | APPROVED |
| HN Show HN | `docs/marketing/launches/phase-34-hn-show-hn.md` | APPROVED |
| Reddit posts | `docs/marketing/launches/phase-34-reddit-post.md` | APPROVED |
| Community announcement | `docs/marketing/launches/phase-34-community-announcement.md` | APPROVED |
| Community FAQ | `docs/marketing/launches/phase-34-community-faq.md` | APPROVED |
| Discord runbook | `docs/marketing/launches/phase-34-discord-runbook.md` | APPROVED |
| Community response queue | `docs/marketing/launches/phase-34-community-response-queue.md` | APPROVED |
| Partner onboarding guide | `docs/marketing/launches/partner-onboarding-guide.md` | DRAFT — rate limits TBD |
| DevRel talk-track | `docs/devrel/talks/tool-trace-platform-instructions-talk-track.md` | APPROVED |
| Partner API Keys demo | `docs/devrel/phase-34-partner-api-keys-demo.md` | APPROVED — partner name TBD |
| Competitive battlecard | `docs/marketing/battlecard/phase-34-partner-api-keys-battlecard.md` | APPROVED — marketplace billing TBD |

---

## Open Blockers Summary (as of 2026-04-24)

| Blocker | Owner | Urgency |
|---------|-------|---------|
| GA vs Beta label decision | PM / PMM | 🔴 Must resolve by Apr 28 |
| partner-api-keys og_image | Social Media Brand | 🔴 Must resolve by Apr 29 |
| PR marketing/phase-34-launch-prep → staging | Social Media Brand (run script) | 🔴 Must merge before Apr 30 |
| PR #1860 close | Anyone with GH access | 🔴 Must close before Apr 30 |
| Partner key rate limits | PM (molecule-controlplane) | 🟡 Must fill placeholder before partner guide goes public |
| X_API_KEY + X_API_SECRET | mol-ops (#1865) | 🔴 Must provision before Apr 30 X posting |
| SaaS Fed v2 battlecard | PM | 🟡 Non-blocking for Apr 30 core launch |
| Design partner name | PM | 🟡 Non-blocking — remove "Acme Corp" if not confirmed |

---

*Marketing Lead 2026-04-24. All Phase 34 launch content complete on `marketing/phase-34-launch-prep`. Launch is go pending blockers above.*
