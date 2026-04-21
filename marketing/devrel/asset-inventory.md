# Phase 30 Launch — DevRel Asset Inventory

> **Cycle:** Marketing work cycle — offline asset tracking
> **Status:** Master list, update as content ships
> **Branch:** `content/blog/memory-backup-restore` (9 commits ahead of main; push blocked on GH_TOKEN)

Use this as the source of truth for what DevRel has produced this cycle, what's pending review, what's staged, and what's blocked.

---

## How to Read This Doc

- **✅ LIVE** — published to docs site or social channels
- **🔍 REVIEW** — written, needs eyes from Marketing Lead / Doc Specialist / Support
- **📦 STAGED** — committed to `content/blog/memory-backup-restore`, ready to push
- **🔒 BLOCKED** — requires action (GH_TOKEN refresh, design team screenshot, PMM asset)

---

## Blog Posts

| File | Status | Owner | Needs |
|---|---|---|---|
| `docs/blog/2026-04-20-remote-workspaces/index.md` | 📦 STAGED | DevRel | Marketing Lead final read |
| `docs/blog/2026-04-20-chrome-devtools-mcp/index.md` | 📦 STAGED | DevRel | Technical accuracy check |
| `docs/blog/2026-04-20-container-vs-remote/index.md` | 📦 STAGED | DevRel | Marketing Lead voice review |
| `docs/blog/2026-04-20-secure-by-design/index.md` | 📦 STAGED | DevRel | Security Lead accuracy review |
| `docs/blog/2026-04-17-deploy-anywhere/index.md` | ✅ pre-existing | — | — |

---

## Docs & Guides

| File | Status | Owner | Needs |
|---|---|---|---|
| `docs/guides/remote-workspaces.md` | 📦 STAGED | DevRel | Doc Specialist final review |
| `docs/guides/same-origin-canvas-fetches.md` | 📦 STAGED | DevRel | Security Lead sign-off on `/cp/*` allowlist section |
| `docs/guides/remote-workspaces-faq.md` | 📦 STAGED | DevRel | Marketing Lead (voice), Doc Specialist (technical), Support (troubleshooting) |
| `docs/marketing/seo/keywords.md` | 🔍 REVIEW | SEO Analyst | SEO Analyst to surface and publish |

---

## Marketing / Social Copy

| File | Status | Owner | Needs |
|---|---|---|---|
| `marketing/devrel/phase30-social-copy.md` | 📦 STAGED | DevRel | PMM or CM to schedule posts (X all 4 versions, LinkedIn) |
| `marketing/devrel/chrome-devtools-mcp-social-copy.md` | 📦 STAGED | DevRel | CM to schedule alongside blog post |
| `marketing/copy/phase30-landing-copy.md` | 📦 STAGED | DevRel | Marketing Lead brand voice review |

---

## Demos — Working Demos + Screencasts

### Demo 1: AGENTS.md Auto-Generation (#1172, PR #763)

| Asset | Status | Notes |
|---|---|---|
| `marketing/demos/agents-md-auto-generation/README.md` | 📦 STAGED | 4 scenario working demo + 1-min screencast outline + TTS script |
| `marketing/demos/agents-md-auto-generation/storyboard.md` | 📦 STAGED | Full production storyboard (camera, VO pacing, highlights, 4 moments) |
| `marketing/demos/agents-md-auto-generation/narration.mp3` | 📦 STAGED | 30s TTS (en-US-AriaNeural) |
| Repo link | 📦 STAGED | `workspace/agents_md.py` on `molecule-core` main |
| **GitHub issue comment** | 🔒 BLOCKED | `comment-1172.json` staged; `post-issue-comments.sh` ready; GH_TOKEN must refresh |
| ASSET: Canvas screenshot (pm-agent + researcher) | 🔒 BLOCKED | Design team needs live canvas + ngrok access |

### Demo 2: Cloudflare Artifacts (#1173, PR #641)

| Asset | Status | Notes |
|---|---|---|
| `marketing/demos/cloudflare-artifacts/README.md` | 📦 STAGED | 5 scenario working demo + 1-min screencast outline + TTS script |
| `marketing/demos/cloudflare-artifacts/storyboard.md` | 📦 STAGED | Full production storyboard (camera, VO pacing, green success pulse, 4 moments) |
| `marketing/demos/cloudflare-artifacts/narration.mp3` | 📦 STAGED | 30s TTS (en-US-AriaNeural) |
| Repo link | 📦 STAGED | `workspace-server/internal/handlers/artifacts.go` on `molecule-core` main |
| **GitHub issue comment** | 🔒 BLOCKED | `comment-1173.json` staged; GH_TOKEN must refresh |

---

## Audio / Video Assets

| File | Duration | Voice | Status | Needs |
|---|---|---|---|---|
| `marketing/audio/phase30-announce.mp3` | ~30s | en-US-AriaNeural | 📦 STAGED | CM to pair with social copy |
| `marketing/audio/phase30-video-vo.mp3` | ~67–75s | en-US-AriaNeural | 📦 STAGED | Video Editor to lock against timeline |
| `marketing/audio/phase30-video-vo-mandarin.mp3` | ~70s | zh-CN-XiaoxiaoNeural | 📦 STAGED | PMM to confirm authoritative script |
| `marketing/audio/chrome-devtools-mcp-summary.mp3` | ~77s | en-US-AriaNeural (+30%) | 📦 STAGED | Slightly over 65–75s target; trim 2s if needed |
| `marketing/audio/quickstart-audio.mp3` | ~67–75s | en-US-AriaNeural | 📦 STAGED | CM to pair with quickstart guide |
| `marketing/audio/phase30-video-vo-mandarin-script.txt` | 188 chars | — | 📦 STAGED | PMM to confirm path + authoritative script |

---

## Visual Assets

| File | Status | Notes |
|---|---|---|
| `marketing/assets/phase30-fleet-diagram.png` | 📦 STAGED | 126KB matplotlib; dark navy, purple REMOTE, blue platform; design notes in `phase30-fleet-diagram-notes.txt` |
| ASSET: Canvas screenshot (remote badge) | 🔒 BLOCKED | Design team needs live canvas + ngrok |
| ASSET: `phase30-canvas-remote-badge.png` | 🔒 BLOCKED | Same blocker as above |

---

## Launch Execution

| File | Status | Notes |
|---|---|---|
| `marketing/drip/post-push-checklist.md` | 📦 STAGED | 6-phase sequencing: push → PR → docs → social → email → verify |
| `marketing/drip/phase30-email-drip.md` | 📦 STAGED | 3-email CRM sequence (Day 1/3–4/7) with placeholders |
| `marketing/community/hacker-news-launch.md` | 📦 STAGED | HN guide, 3 title options, post body template, comment responses |
| `marketing/community/community-announcements.md` | 📦 STAGED | Discord + Slack + Reddit copy, channel-by-channel |

## Sales Enablement

| File | Status | Notes |
|---|---|---|
| `marketing/sales/phase30-sales-enablement.md` | 📦 STAGED | 4 competitive battlecards, 5 objection handlers, 3-min demo script |
| `marketing/sales/phase30-one-pager.md` | 📦 STAGED | 1-page PDF-ready asset with feature table, pricing, quick-start |

---

## Scripts & Helpers

| File | Status | Notes |
|---|---|---|
| `marketing/demos/post-issue-comments.sh` | 📦 STAGED | curl-based helper to post comments to #1172 + #1173 once GH_TOKEN refreshes |
| `comment-1172.json` | 📦 STAGED | Raw JSON body for #1172 comment |
| `comment-1173.json` | 📦 STAGED | Raw JSON body for #1173 comment |

---

## Pending Actions by Owner

### DevRel (this workspace)
- [ ] None currently — all deliverables committed

### Marketing Lead
- [ ] Review `docs/guides/remote-workspaces-faq.md` — voice + technical accuracy
- [ ] Review `marketing/copy/phase30-landing-copy.md` — brand voice
- [ ] Review `docs/blog/2026-04-20-remote-workspaces/index.md` — final read before publish
- [ ] Post `phase30-social-copy.md` — schedule X posts (all 4 versions) + LinkedIn post
- [ ] Post `chrome-devtools-mcp-social-copy.md` — schedule alongside blog post
- [ ] Schedule 3-email drip sequence after blog post is live
- [ ] Submit or assign Hacker News post (see `hacker-news-launch.md`)

### Community Manager
- [ ] Schedule social copy posts (see Marketing Lead row)
- [ ] Post community announcements per `community-announcements.md`

### Video Editor
- [ ] Begin Phase 30 video assembly per `phase30-video-production.md`

### Sales / Solutions Engineering
- [ ] Review `phase30-sales-enablement.md` — customize talk tracks to seller voice
- [ ] Review `phase30-one-pager.md` — replace link placeholders before distributing

### PMM
- [ ] Confirm authoritative path for `marketing/social/phase30-launch-plan.md` (currently confirmed missing from internal repo)
- [ ] Confirm `phase30-video-vo-mandarin-script.txt` is the right script (188-char DevRel-authored placeholder)
- [ ] Supply canvas screenshot (`phase30-canvas-remote-badge.png`) using live canvas + ngrok

### Design Team
- [ ] Capture canvas screenshot showing REMOTE badge on workspace card
- [ ] Refine `phase30-fleet-diagram.png` per `phase30-fleet-diagram-notes.txt` design checklist

### SEO Analyst
- [ ] Surface and publish `docs/marketing/seo/keywords.md`

### Support
- [ ] Review troubleshooting section of `docs/guides/remote-workspaces-faq.md`

### Security Lead
- [ ] Review `/cp/*` allowlist section in `docs/guides/same-origin-canvas-fetches.md`
- [ ] Review `docs/blog/2026-04-20-secure-by-design/index.md`

### CEO / Token Owner
- [ ] **CRITICAL:** Refresh `GH_TOKEN` — all pushes and issue comments are blocked until this is done

---

*Maintained by DevRel. Update status columns as content ships or blockers clear.*
