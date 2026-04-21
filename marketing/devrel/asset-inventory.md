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
| `docs/marketing/devrel/demos/org-api-keys/README.md` | 📦 STAGED | DevRel | Marketing Lead brand voice review |
| `docs/marketing/devrel/demos/org-api-keys/storyboard.md` | 📦 STAGED | DevRel | Video Editor production package ready |
| `docs/marketing/devrel/demos/org-api-keys/narration.mp3` | 📦 STAGED | DevRel | TTS VO complete (en-US-AriaNeural) |
| `docs/guides/remote-workspaces-faq.md` | 📦 STAGED | DevRel | Marketing Lead (voice), Doc Specialist (technical), Support (troubleshooting) |
| `docs/marketing/seo/keywords.md` | 📦 STAGED | DevRel + SEO | DevRel drafted initial keywords; SEO Analyst to finalize publish targets |

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

### Merged PR Verification Docs (this cycle)

| File | Status | Notes |
|---|---|---|
| `marketing/devrel/demos/snapshot-secret-scrubber-walkthrough.md` | 📦 STAGED | Marketing walkthrough: 3-scenario Python demo, 60s screencast outline, full pattern table, Q&A |
| `marketing/devrel/demos/snapshot-scrub-demo-verification.md` | 📦 STAGED | Source-accurate: all 4 functions + 12 patterns verified against `snapshot_scrub.py` |
| `marketing/devrel/demos/memory-inspector-panel-demo-verification.md` | 📦 STAGED | Endpoint + field verified against `memories.go` + `MemoryInspectorPanel.tsx` |
| `marketing/devrel/demos/failed-workspace-ec2-console-demo-verification.md` | 📦 STAGED | UI demo verification: source feature check, 9-step screencast checklist |
| `marketing/devrel/demos/assets/snapshot-scrubber-before-after.png` | 📦 STAGED | See Visual Assets above |
| `marketing/devrel/demos/assets/memory-inspector-panel-ui.png` | 📦 STAGED | See Visual Assets above |
| `marketing/devrel/demos/assets/ec2-console-canvas.png` | 📦 STAGED | See Visual Assets above |

### Demo 2: Cloudflare Artifacts (#1173, PR #641)

| Asset | Status | Notes |
|---|---|---|
| `marketing/demos/cloudflare-artifacts/README.md` | 📦 STAGED | 5 scenario working demo + 1-min screencast outline + TTS script |
| `marketing/demos/cloudflare-artifacts/storyboard.md` | 📦 STAGED | Full production storyboard (camera, VO pacing, green success pulse, 4 moments) |
| `marketing/demos/cloudflare-artifacts/narration.mp3` | 📦 STAGED | 30s TTS (en-US-AriaNeural) |
| Repo link | 📦 STAGED | `workspace-server/internal/handlers/artifacts.go` on `molecule-core` main |
| `marketing/devrel/demos/cloudflare-artifacts-walkthrough.md` | 📦 STAGED | Marketing walkthrough: 5-step demo, screencast outline, security notes, common Q&A |
| `marketing/devrel/demos/cloudflare-artifacts-demo-verification.md` | 📦 STAGED | Source-accurate verification; blog/demo/storyboard/TTS all verified |
| `marketing/devrel/demos/assets/cf-artifacts-workflow.png` | 📦 STAGED | Workflow diagram (see Visual Assets above) |
| `screencasts/phase30-screencast-02-cloudflare-artifacts.mp4` | 📦 STAGED | 60s .mp4 (dark zinc, H.264+AAC, 1920x1088) |
| `screencasts/phase30-screencast-06-cloudflare-artifacts.mp4` | 📦 STAGED | 60s .mp4 v2 (from PMM storyboard PR #1306) |
| **GitHub issue comment** | 🔒 BLOCKED | `comment-1173.json` staged; GH_TOKEN must refresh |

### Demo 3: Org-Scoped API Keys (PR #1105)

| Asset | Status | Notes |
|---|---|---|
| `docs/marketing/devrel/demos/org-api-keys/README.md` | 📦 STAGED | Working demo: mint, use, revoke, confirm 401 — full curl walkthrough |
| `docs/marketing/devrel/demos/org-api-keys/storyboard.md` | 📦 STAGED | 60s production storyboard |
| `docs/marketing/devrel/demos/org-api-keys/narration.mp3` | 📦 STAGED | 30s TTS (en-US-AriaNeural) |
| Repo link | 📦 STAGED | `workspace-server/internal/handlers/org_tokens.go` on `molecule-core` main |

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
| `marketing/devrel/demos/assets/snapshot-scrubber-before-after.png` | 📦 STAGED | Before/after terminal output; red = raw, green = scrubbed; shows scrub_snapshot() transformation |
| `marketing/devrel/demos/assets/memory-inspector-panel-ui.png` | 📦 STAGED | Canvas Memory Inspector UI mockup; scope tabs, namespace dropdown, semantic search, 5 entry rows |
| `marketing/devrel/demos/assets/cf-artifacts-workflow.png` | 📦 STAGED | CF Artifacts git workflow; 4-node flow + scrubber gate + detail boxes |
| `marketing/devrel/demos/assets/ec2-console-canvas.png` | 📦 STAGED | Failed workspace EC2 console in Canvas; FAILED badge, error panel, no-AWS-Console explanation |
| `marketing/devrel/demos/make_demo_assets.py` | 📦 STAGED | Python script generating all 4 PNG diagrams above (matplotlib) |
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

## Screencasts (Phase 30 launch)

> Master index: `/workspace/marketing/devrel/demos/SCREENCASTS.md`
> All reference images: `/workspace/marketing/devrel/demos/screencast-refs/`
> **Storyboard:** `marketing/devrel/storyboard-memory-inspector-panel.md` (PMM-specified path; keyed to `MemoryInspectorPanel.tsx`) | **TTS:** `memory-inspector-narration.mp3` (523 KB, en-US-AriaNeural, +20%)

| Screencast | Duration | Priority | Status | Storyboard | Reference Image |
|---|---|---|---|---|---|
| AGENTS.md Auto-Generation (#42/#763) | 60s | HIGH | ✅ DONE | `storyboard.md` | N/A |
| Cloudflare Artifacts (#58/#641) | 60s | HIGH | ✅ DONE | `storyboard.md` | N/A |
| EC2 Console Output (#68/#1178) | 60s | HIGH | ✅ DONE | `screencast-storyboard-ec2-console.md` | `screencast-refs/ec2-console-canvas.png` |
| MemoryInspectorPanel (#65/#1127) | 60s | HIGH | ✅ DONE (PMM version) | `screencast-storyboard-memory-inspector-panel.md` | `screencast-refs/memory-inspector-panel-ui.png` |
| Snapshot Secret Scrubber (#63/#977) | 60s | HIGH | ✅ DONE (PMM version) | `screencast-storyboard-snapshot-scrubber.md` | `screencast-refs/snapshot-scrubber-before-after.png` |

### Screencast Deliverables (local files)

```
/workspace/marketing/devrel/demos/
├── SCREENCASTS.md                                             ← master index
├── generate_screencasts.py                                    ← production script (matplotlib + edge_tts)
├── screencast-storyboard-ec2-console.md
├── screencast-ec2-console-production-package.md
├── screencast-storyboard-cloudflare-artifacts.md
├── screencast-cf-artifacts-production-package.md
├── content-blog-cloudflare-artifacts-security-note.md        ← add to blog
├── screencast-storyboard-memory-inspector-panel.md
├── screencast-memory-inspector-production-package.md
├── screencast-storyboard-snapshot-scrubber.md
├── screencast-snapshot-scrubber-production-package.md
├── screencast-refs/                                           ← reference images
│   ├── ec2-console-canvas.png
│   ├── cf-artifacts-workflow.png
│   ├── memory-inspector-panel-ui.png
│   └── snapshot-scrubber-before-after.png
└── screencasts/                                               ← TTS narration + animated GIF previews
    ├── ec2-console.mp3  (489 KB)                             ← en-US-AriaNeural TTS
    ├── ec2-console.gif  (55 KB, 3 frames)
    ├── memory-inspector.mp3  (444 KB)
    ├── memory-inspector.gif  (55 KB, 4 frames)
    ├── snapshot-scrubber.mp3  (393 KB)
    ├── snapshot-scrubber.gif  (61 KB, 4 frames)
    ├── cf-artifacts.mp3  (396 KB)
    └── cf-artifacts.gif  (67 KB, 5 frames)
```

## Scripts & Helpers

| File | Status | Notes |
|---|---|---|
| `marketing/demos/post-issue-comments.sh` | 📦 STAGED | curl-based helper to post comments to #1172 + #1173 once GH_TOKEN refreshes |
| `comment-1172.json` | 📦 STAGED | Raw JSON body for #1172 comment |
| `comment-1173.json` | 📦 STAGED | Raw JSON body for #1173 comment |
| `marketing/devrel/demos/make_demo_assets.py` | 📦 STAGED | Python script generating all 4 PNG diagrams |

---

## Pending Actions by Owner

### DevRel (this workspace)
- [x] 8 screencast .mp4 files produced (60s each, H.264+AAC, 1920x1088, dark zinc theme)
  - #1 EC2 Console, #2 CF Artifacts, #3 MemoryInspector, #4 SnapshotScrubber (batch 1)
  - #5 AGENTS.md, #6 CF Artifacts v2 (batch 2, PMM storyboards)
  - #7 MemoryInspectorPanel PMM version, #8 SnapshotScrubber PMM version (batch 3, PMM callouts)
- [x] All storyboards, production packages, demo verification docs complete
- [x] 4 reference images generated (matplotlib, dark brand theme)
- [x] Master screencast index (`SCREENCASTS.md`) and asset inventory updated

### Video Editor
- [ ] **Optional:** Produce broadcast-quality versions from live screen capture — DevRel .mp4s are storyboard-accurate previews; higher-fidelity footage requires OBS/ScreenFlow/Camtasia with actual Canvas + terminal access

### Marketing Lead
- [ ] Review `docs/guides/remote-workspaces-faq.md` — voice + technical accuracy
- [ ] Review `marketing/copy/phase30-landing-copy.md` — brand voice
- [ ] Review `docs/blog/2026-04-20-remote-workspaces/index.md` — final read before publish
- [ ] Post `phase30-social-copy.md` — schedule X posts (all 4 versions) + LinkedIn post
- [ ] Post `chrome-devtools-mcp-social-copy.md` — schedule alongside blog post
- [ ] Schedule 3-email drip sequence after blog post is live
- [ ] Submit or assign Hacker News post (see `hacker-news-launch.md`) — **blocked: blog post not yet live (GH_TOKEN down)**

### Content Marketer
- [ ] Add security note to `content/blog/2026-04-21-cloudflare-artifacts/index.mdx` — staged at `content-blog-cloudflare-artifacts-security-note.md`. 3 sentences about `snapshot_scrub.py` running before git serialization.

### Community Manager
- [ ] Schedule social copy posts (see Marketing Lead row)
- [ ] Post community announcements per `community-announcements.md`

### Sales / Solutions Engineering
- [ ] Review `phase30-sales-enablement.md` — customize talk tracks to seller voice
- [ ] Review `phase30-one-pager.md` — replace link placeholders before distributing; verify quick-start CLI commands

### PMM
- [ ] Confirm authoritative path for `marketing/social/phase30-launch-plan.md` (currently confirmed missing from internal repo)
- [ ] Confirm `phase30-video-vo-mandarin-script.txt` is the right script (188-char DevRel-authored placeholder)
- [ ] Supply canvas screenshot (`phase30-canvas-remote-badge.png`) using live canvas + ngrok

### Design Team
- [ ] Capture canvas screenshot showing REMOTE badge on workspace card
- [ ] Refine `phase30-fleet-diagram.png` per `phase30-fleet-diagram-notes.txt` design checklist

### SEO Analyst
- [x] Surface and publish `docs/marketing/seo/keywords.md` ← DevRel drafted; SEO Analyst to finalize publish targets and keyword volume estimates

### Support
- [ ] Review troubleshooting section of `docs/guides/remote-workspaces-faq.md`

### Security Lead
- [ ] Review `/cp/*` allowlist section in `docs/guides/same-origin-canvas-fetches.md`
- [ ] Review `docs/blog/2026-04-20-secure-by-design/index.md`

### CEO / Token Owner
- [ ] **CRITICAL:** Refresh `GH_TOKEN` — all pushes and issue comments are blocked until this is done

---

*Maintained by DevRel. Update status columns as content ships or blockers clear.*
