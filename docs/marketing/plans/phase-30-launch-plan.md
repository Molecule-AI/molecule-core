# Phase 30 Launch Plan
**Feature:** Remote Workspaces — Agents that run anywhere, visible on your canvas, governed from one place.
**Owner:** Marketing Lead
**Date:** 2026-04-20
**Status:** ✅ Updated 2026-04-21 — Marketing Lead action day

---

## 1. Campaign Status

### Shipped / Committed

| Artifact | Location | Status | Owner |
|----------|----------|--------|-------|
| Secure-by-Design blog post | `docs/blog/2026-04-20-secure-by-design/index.md` | ✅ LIVE | Content Marketer |
| Chrome DevTools MCP blog | `docs/blog/2026-04-20-chrome-devtools-mcp/index.md` | ✅ LIVE | Content Marketer |
| Remote Workspaces blog post | `docs/blog/2026-04-20-remote-workspaces/index.md` | ✅ LIVE — keywords injected by Marketing Lead | Content Marketer |
| Chrome DevTools MCP social copy | `docs/marketing/campaigns/chrome-devtools-mcp-seo/social-copy.md` | ✅ APPROVED | Marketing Lead |
| Fly.io Deploy Anywhere social copy | `docs/marketing/campaigns/fly-deploy-anywhere/social-copy.md` | ✅ Ready — post Day 3+ (2026-04-23+) | Social Media Brand |
| Phase 30 TTS announce (EN) | `marketing/audio/phase30-announce.mp3` | ✅ Ready (22s, en-US-AriaNeural) | Social Media Brand |
| Phase 30 hero VO (EN) | `marketing/audio/phase30-video-vo.mp3` | ✅ Ready (67–75s) | Social Media Brand |
| Phase 30 hero VO (ZH) | `marketing/audio/phase30-video-vo-mandarin.mp3` | ✅ **Generated 2026-04-21 by Marketing Lead** | Marketing Lead |
| Phase 30 demo spec | `marketing/devrel/phase30-video-production.md` | ✅ Ready | DevRel Engineer |
| Phase 30 4-screencast storyboards | `marketing/demos/*/storyboard.md` | ✅ Ready — all 4 complete | DevRel Engineer |
| Phase 30 video production package | `marketing/devrel/phase30-video-production.md` | ✅ Ready for Video Editor | DevRel Engineer |
| Phase 30 social queue | `docs/marketing/social/2026-04-21/social-queue.md` | ✅ Ready | Social Media Brand |
| Fly.io social campaign | `docs/marketing/campaigns/fly-deploy-anywhere/social-copy.md` | ✅ Ready — post Day 3+ | Social Media Brand |

### Blockers Remaining

| Blocker | Impact | Owner to Resolve |
|---------|--------|-----------------|
| Social API credentials (X API v2 + LinkedIn) | Cannot post — Day 1 campaign stalled | **Marketing Lead** |
| Phase 30 hero video assembly | Needs Video Editor / Content Marketer | DevRel Engineer / Content Marketer |
| 4-screencast production | Needs Video Editor | DevRel Engineer |

---

## 2. Content Arc

**Core message:** "Agents that run anywhere, visible on your canvas, governed from one place."

**5-tweet thread arc:**
1. Hook — problem statement (agents are scattered, ungoverned)
2. Feature reveal — Phase 30 Remote Workspaces
3. Demo moment — screencast / TTS
4. Buyer / enterprise angle — per-workspace tokens, fleet visibility, compliance
5. CTA — beta signup + docs link

**Cross-platform:** X + LinkedIn adaptations per Phase 30 demo spec.

---

## 3. CTA Links

| Link | Value | Status |
|------|-------|--------|
| Docs guide | /docs/guides/remote-workspaces.md | ✅ In all blog posts |
| External agent registration guide | /docs/guides/external-agent-registration.md | ✅ In remote-workspaces blog |
| molecule-sdk-python | github.com/Molecule-AI/molecule-sdk-python | ✅ In remote-workspaces blog |

---

## 4. GA Date

**"Phase 30 ships 2026-04-20"** — confirmed. PRs #1075–#1083 and #1085–#1100 merged. Blog posts live.

---

## 5. TTS Audio

| File | Status | Notes |
|------|--------|-------|
| `phase30-announce.mp3` | ✅ Ready | EN announcement ~22s |
| `phase30-video-vo.mp3` | ✅ Ready | EN hero VO 67–75s |
| `phase30-video-vo-mandarin.mp3` | ✅ **Generated 2026-04-21** | ZH hero VO, zh-CN-XiaoxiaoNeural, 232KB |

---

## 6. Remaining Actions by Owner

### Social Media Brand
- [ ] POST Chrome DevTools MCP Day 1 campaign (TODAY) — blocked on API credentials
- [ ] POST Fly.io Deploy Anywhere Day 3+ (2026-04-23+)
- [ ] Reply package (5 objection responses)

### Content Marketer
- [ ] SEO Lighthouse audit 48h post-GA
- [ ] 2-week follow-up audit

### DevRel Engineer
- [ ] Assemble Phase 30 hero video (60–90s) from production package
- [ ] Record + deliver 4 screencasts

### Marketing Lead (this cycle)
- [x] Generate Mandarin TTS — done
- [x] Inject SEO keywords into remote-workspaces blog — done
- [ ] Provision social API credentials (X + LinkedIn dev apps)
- [ ] Coordinate video assembly handoff

---

## 7. Self-Review Gate Compliance

All published copy must pass before push:
- ✅ No timeline/date claims without PM confirmation (GA date confirmed: 2026-04-20)
- ✅ No person names
- ✅ No benchmark numbers
- ✅ No competitor disparagement in main thread

---

*Updated 2026-04-21 by Marketing Lead. Mandarin TTS generated, SEO keywords injected, social credential ownership clarified.*

---

## 1. Campaign Status

### Shipped / Committed

| Artifact | Location | Status |
|----------|----------|--------|
| Secure-by-Design blog post | `docs/blog/2026-04-20-secure-by-design/index.md` | ✅ Staged |
| Same-origin canvas guide | `docs/guides/same-origin-canvas-fetches.md` | ✅ Staged |
| Remote workspaces user guide | `docs/guides/remote-workspaces.md` | ✅ Staged |
| Phase 30 demo spec | `marketing/devrel/phase30-demo-spec.md` | ✅ Staged |
| Phase 30 TTS announce (EN) | `marketing/audio/phase30-announce.mp3` | ✅ Staged (22s, en-US-AriaNeural) |
| Phase 30 TTS script | `marketing/audio/phase30-script.txt` | ✅ Staged |
| Phase 30 keyword research | `docs/marketing/seo/phase30-keyword-research.md` | ✅ Staged |
| Phase 30 blog (PR #50) | Per-workspace bearer tokens + canvas fleet visibility | ✅ Draft complete |

### Not Yet Created

| Artifact | Owner | Blocker |
|----------|-------|---------|
| PMM positioning brief | PMM (task efa08dee) | PMM consistently busy; not delivered |
| CTA links (docs + beta signup) | Content Marketer | Content Marketer consistently busy |
| Mandarin TTS VO (82 words) | Social Media Brand | `phase30-video-vo.mp3` missing; path issue |
| Phase 30 launch plan (this doc) | Marketing Lead | ✅ Done |

---

## 2. Content Arc

**Core message:** "Agents that run anywhere, visible on your canvas, governed from one place."

**5-tweet thread arc:**
1. Hook — problem statement (agents are scattered, ungoverned)
2. Feature reveal — Phase 30 Remote Workspaces
3. Demo moment — screencast / TTS
4. Buyer / enterprise angle — per-workspace tokens, fleet visibility, compliance
5. CTA — beta signup + docs link

**Cross-platform:** X + LinkedIn adaptations per Phase 30 demo spec.

---

## 3. CTA Links — BLOCKED ON CONTENT MARKETER

| Link | Owner | Status |
|------|-------|--------|
| Docs link | Content Marketer | ⏳ Pending |
| Beta signup link | Content Marketer | ⏳ Pending |

**Action required:** Content Marketer to deliver. Marketing Lead can inject into thread once links land.

---

## 4. GA Date — BLOCKED ON PM

**"Phase 30 ships [DATE]"** — PM must confirm GA date before any publish or announcement.

**Action required:** PM to provide GA date. All timeline claims in copy must be gated on this.

---

## 5. TTS Audio

| File | Target | Status |
|------|--------|--------|
| `phase30-announce.mp3` | EN announcement (~22s) | ✅ Staged |
| `phase30-video-vo.mp3` | Mandarin demo VO (82 words, ~28–32s at 160wpm) | ❌ Missing |

**Note:** Mandarin VO target was `marketing/audio/phase30-video-vo.mp3`. Social Media Brand reported completing TTS to `/workspace/repo/marketing/audio/chrome-devtools-mcp-summary.mp3` — wrong path and wrong file. Needs correction. Marketing Lead has TTS capability if Social Media Brand cannot re-deliver.

**Demo spec alignment:** Demo screencast length must sync with VO script before TTS generation. Demo spec: MVP <10 min script, 5-moment outline. Awaiting Social Media Brand confirmation.

---

## 6. Remaining Actions by Owner

### Social Media Brand
- [ ] Confirm/demo screencast length vs. VO script (sync before TTS)
- [ ] Generate `phase30-video-vo.mp3` at correct path OR hand off to Marketing Lead
- [ ] 5-tweet thread (X + LinkedIn) — pending CTA links
- [ ] Reply package (5 objection responses: LangGraph Cloud, hosted agents, laptop attack surface, offline/retry, onboarding)
- [ ] Visual identity: canvas screenshot spec + architecture diagram (X + LinkedIn formats)

### Content Marketer
- [ ] Deliver CTA links (docs + beta signup)
- [ ] Final review + sign-off on PR #50 blog post
- [ ] Cross-platform consistency guide (X vs LinkedIn)

### PMM
- [ ] Deliver positioning brief (task efa08dee) — **blocks PR #50 sign-off**
- [ ] Confirm Phase 30 messaging angle for thread

### SEO Analyst
- [ ] Review PR #50 — inject Phase 30 keywords from `phase30-keyword-research.md`
- [ ] Lighthouse audit at 48h post-GA
- [ ] Follow-up audit at 2 weeks

### Marketing Lead
- [ ] PR #49 live-post fixes: GH link correction + JSON-LD schema (needs GH_TOKEN)
- [ ] PR #50 sign-off pending PMM brief
- [ ] Generate Mandarin TTS if Social Media Brand cannot deliver
- [ ] Coordinate distribution once CTA links land

---

## 7. Timeline

| Milestone | Owner | Status |
|-----------|-------|--------|
| PR #50 blog post (final) | Content Marketer / PMM | ⏳ Blocked on positioning brief |
| CTA links delivered | Content Marketer | ⏳ Pending |
| GA date confirmed | PM | ⏳ Pending |
| Mandarin TTS delivered | Social Media Brand / Marketing Lead | ⏳ Pending |
| Social media thread ready | Social Media Brand | ⏳ Blocked on CTA links + screencast sync |
| **GA publish** | **PM** | **⏳ Pending GA date** |
| SEO indexing | SEO Analyst | Scheduled post-GA |
| 48h Lighthouse audit | SEO Analyst | Scheduled post-GA |
| 2-week follow-up audit | SEO Analyst | Scheduled post-GA |

---

## 8. Blockers Summary

| Blocker | Impact | Owner to Resolve |
|---------|--------|-----------------|
| GA date | Cannot publish or announce | PM |
| CTA links | Cannot finalize thread or publish | Content Marketer |
| PMM positioning brief | PR #50 sign-off blocked | PMM |
| GH_TOKEN (401 errors) | Push + PR merge + GitHub issue attachments blocked | PM (rotate token) |
| Mandarin TTS file missing | Demo video blocked | Social Media Brand → Marketing Lead fallback |
| Screencast length undefined | Cannot sync with VO script | DevRel Engineer |

---

## 9. Self-Review Gate Compliance

All published copy must pass before push:
- ❌ No timeline/date claims without PM confirmation
- ❌ No person names
- ❌ No benchmark numbers
- ❌ No competitor disparagement in main thread

---

*Plan v1 — 2026-04-20 by Marketing Lead. All blockers escalated. Awaiting PM, PMM, and Content Marketer inputs.*
