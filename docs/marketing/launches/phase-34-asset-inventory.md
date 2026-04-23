# Phase 34 — Launch Asset Inventory
**Campaign:** Phase 34 GA (April 30, 2026)
**Owner:** Community Manager
**Date:** 2026-04-23
**Status:** IN PROGRESS — git push blocked (token issues)

---

## Discord / Community

| Asset | File | Branch | Status | Notes |
|-------|------|--------|--------|-------|
| Community announcement | `phase-34-community-announcement.md` | `docs/phase34-community-launch` | ✅ Pushed | PR #1860 → staging, auto-merge pending |
| Community FAQ | `phase-34-community-faq.md` | `marketing/phase-34-launch-prep` | ✅ Pushed | 15+ Q&As, 5 sections |
| Discord runbook | `phase-34-discord-runbook.md` | `marketing/phase-34-launch-prep` | ✅ Pushed | 223 lines, timing table, escalation matrix |
| Reddit post (Day 2) | `phase-34-reddit-post.md` | `marketing/phase-34-launch-prep` | 🔄 Local commit (bb21fed0) | 130 lines, r/MachineLearning, not pushed |
| HN Show HN post (Day 2) | `phase-34-hn-show-hn.md` | `marketing/phase-34-launch-prep` | 🔄 Local commit (bb21fed0) | 130 lines, 4-question FAQ, not pushed |

---

## Social Media

| Asset | File | Branch | Status | Notes |
|-------|------|--------|--------|-------|
| Tool Trace social copy | `phase-34-tool-trace-social-copy.md` | `marketing/phase-34-launch-prep` | 🔄 Local commit (026931cc) | 5-post thread draft, NOT pushed |
| Platform Instructions social copy | `phase-34-platform-instructions-social-copy.md` | `marketing/phase-34-launch-prep` | 🔄 Local commit (026931cc) | 5-post thread draft, NOT pushed |
| EC2 Instance Connect social | `ec2-instance-connect-ssh-social-copy.md` | `social-ec2-instance-connect` | ✅ Pushed | Publish-ready, CTA live |
| Partner API Keys social | `phase-34-partner-api-keys-social-copy.md` | `marketing/phase-34-launch-prep` | 🔄 Local commit (8cec7888) | GA April 30 — curl examples, 5-post thread |
| Combined overview social | `tool-trace-platform-instructions-social-copy.md` | `marketing/phase-34-launch-prep` | ✅ Pushed (b40d001b) | Existing file on branch |

**X credentials:** ❌ BLOCKED — Issue #1865, no mol-ops response
Social posts cannot go out until X credentials resolved or Day 2 without social.

---

## Blog Posts (staging, ?ref=staging)

| Post | URL | Status |
|------|-----|--------|
| Tool Trace + Platform Instructions overview | `docs.moleculesai.app/blog/tool-trace-platform-instructions` | ✅ Live |
| Tool Trace deep-dive | `docs.moleculesai.app/blog/ai-agent-observability-without-overhead` | ✅ Live |
| Platform Instructions | `docs.moleculesai.app/blog/platform-instructions-governance` | ✅ Live |
| Partner API Keys | `docs.moleculesai.app/blog/partner-api-keys` | ✅ Live (shows "coming soon" until Apr 30) |

---

## Docs

| File | Status |
|------|--------|
| `docs/architecture/partner-api-keys.md` | ✅ Updated — `mol_pk_*` key format + scopes |
| `docs/api-protocol/a2a-protocol.md` | ✅ Updated — `tool_trace` in `Message.metadata` |
| `docs/guides/external-workspace-quickstart.md` | ✅ Updated — SaaS Fed v2 changes |
| `docs/infra/workspace-terminal.md` | ✅ Shipped in PR #1533 (EC2 Instance Connect) |

---

## DevRel Assets

| Asset | Status | Notes |
|-------|--------|-------|
| Phase 34 Partner API Keys screencast TTS | ✅ Pushed | 579 KB, ~65s narration |
| Phase 34 Partner API Keys runnable demo | ✅ Pushed | Package on marketing branch |
| Cloudflare Artifacts TTS narration | ✅ Pushed | 60s talk-track |
| Phase 30 Remote Workspaces battlecard | ✅ Pushed | `docs/marketing/social/2026-04-22-ec2-instance-connect-ssh/` |

---

## Blockers

| Blocker | Impact | Resolution |
|---------|--------|------------|
| Git push blocked (token) | Social copy drafts can't reach remote | Wait for ops token fix |
| X credentials (issue #1865) | No external social on Apr 30 | mol-ops to provide `X_API_KEY` + `X_API_SECRET` |
| GitHub API 401 | Can't check PR merge state | Token issue, ops-level fix |
| **All assets written** | ✅ Phase 34 complete | None — pending push + publish |

---

## Day 2 Plan (April 30 ~16:00 UTC)

1. Reddit r/MachineLearning: `phase-34-reddit-post.md`
2. HackerNews Show HN: `phase-34-hn-show-hn.md` + pinned tool_trace code sample
3. Monitor 2h (Reddit) / 3h (HN), 30-min reply SLA

---

*Last updated: 2026-04-23 18:00 UTC*