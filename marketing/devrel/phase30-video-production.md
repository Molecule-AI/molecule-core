# Phase 30 Launch Video — Video Editor Production Package

> **For:** Video Editor | **Cycle:** Marketing work cycle
> **Status:** Ready for production
> **Branch:** `content/blog/memory-backup-restore` (10 commits; push blocked on GH_TOKEN)

This doc tells the video editor how to assemble the Phase 30 launch video from existing DevRel assets. All source files are in the repo. No new recording needed.

---

## Assembled Video: "Agents That Run Where You Need Them"

**Target length:** 60–90 seconds
**Purpose:** Hero launch video for docs site, social, and email campaign
**Tone:** Clean, confident, technical-but-accessible. Not salesy. Show, then tell.

---

## Video Structure (3 Acts)

### Act 1 — The Fleet (0:00–0:20)

**Visual:** `phase30-fleet-diagram.png` — the matplotlib diagram we already generated.
Dark navy background, purple REMOTE workspace boxes, blue platform, green canvas.
**Animation suggestion:** Fade in platform first (0:00–0:03), then platform connections draw in (0:03–0:08), then REMOTE boxes slide in from right edge (0:08–0:15), then canvas at bottom fades in (0:15–0:20). Total build: ~20s.

**VO:** `phase30-video-vo.mp3` plays over the full sequence (67–75s). Use the script at `marketing/audio/phase30-video-vo-script.txt` as the narration lock.

**Narration start (approx 0:00–0:20 passage):**
> "Most AI agent platforms assume all agents run inside the platform. Molecule AI didn't."

---

### Act 2 — The Detail (0:20–0:50)

**Visual:** A split or sequence showing:
1. Terminal window — `python3 run.py` + agent registration output (show the `INFO workspace: registered` log line)
2. Canvas — workspace card with REMOTE badge in purple
3. Same card, active — A2A message incoming

**How to capture these:**
- Use the dev canvas at `localhost:3000` with a remote workspace in active state
- Record the registration log output from a terminal running the Python SDK
- Cut between the three frames at 0:20 / 0:35 / 0:45 marks

**VO continues:** Middle section of `phase30-video-vo.mp3`. The narration covers the mixed-fleet story (see script).

---

### Act 3 — The Close (0:50–0:75)

**Visual:** Return to the fleet diagram — fully built, all connections lit.
**Animation:** A gentle pulse along one A2A connection line (simulate a task dispatch).

**VO:** Final passage of `phase30-video-vo.mp3`:
> "Phase 30. Remote Workspaces. Your agents. Your infrastructure. One canvas."

**End card:** Molecule AI logo + "Phase 30 — Now GA" + link: `moleculesai.app/docs/guides/remote-workspaces`
**Duration:** 2s hold, 1s fade to black.

---

## Asset Checklist

| Asset | Location | Status | Notes |
|---|---|---|---|
| Fleet diagram (PNG) | `marketing/assets/phase30-fleet-diagram.png` | ✅ Ready | 126KB, dark navy. Use for Act 1 + Act 3 return. |
| VO track (EN) | `marketing/audio/phase30-video-vo.mp3` | ✅ Ready | 67–75s, en-US-AriaNeural. Lock against timeline. |
| VO track (ZH) | `marketing/audio/phase30-video-vo-mandarin.mp3` | ✅ Ready | ~70s, zh-CN-XiaoxiaoNeural. For Mandarin cut. |
| VO script (EN) | `marketing/audio/phase30-video-vo-script.txt` | ✅ Ready | Reference for timing and lock-points. |
| VO script (ZH) | `marketing/audio/phase30-video-vo-mandarin-script.txt` | ✅ Ready | 188-char Mandarin. |
| Phase 30 blog post | `docs/blog/2026-04-20-remote-workspaces/index.md` | 📦 STAGED | Link in end card. |
| Quickstart guide | `docs/guides/remote-workspaces.md` | 📦 STAGED | Secondary link in end card. |
| Announcement audio | `marketing/audio/phase30-announce.mp3` | 📦 STAGED | 30s. Use for social cut-down (0:00–0:30 of X clip). |

---

## Specs for Editor

- **Format:** 1080p H.264, 30fps (social) / 24fps (docs site)
- **Aspect ratios needed:** 16:9 (docs site + YouTube), 9:16 (X/TikTok Reel), 1:1 (LinkedIn)
- **Music:** No music in primary cut. Consider a light ambient bed (60–75bpm, non-melodic) for the 9:16 social cut only — keep VO clean in primary cut.
- **Color grade:** Match fleet diagram's dark navy + purple palette. Avoid blowing out the canvas screenshots — keep them readable against dark background.
- **Captions:** Burn in captions for the VO (for muted playback on social). Use `en-US-AriaNeural` timing from `phase30-video-vo-script.txt` for sync.
- **Muting:** Primary cut (docs site) can run without captions if VO is present. Social cut (X) must have captions burned in — most users watch muted.

---

## Alt Cuts

### Short Announcement (30s) — X/TikTok Reel
**Source assets:** `phase30-announce.mp3` (30s VO) + fleet diagram + REMOTE badge screenshot
**Structure:** Fleet diagram build (0:00–0:15) → REMOTE badge screenshot (0:15–0:20) → End card (0:20–0:30)
**Use for:** X timeline, TikTok, Instagram Reels

### Mandarin Cut (60–75s)
**Source assets:** `phase30-video-vo-mandarin.mp3` + same visuals as primary cut
**VO script:** `phase30-video-vo-mandarin-script.txt` (188 chars)
**Use for:** WeChat, Chinese-language social channels, LinkedIn (zh-CN audience)

---

## Review Checklist (before publishing)

- [ ] VO is locked and plays cleanly over fleet diagram build
- [ ] REMOTE badge is visible in the canvas cutaways
- [ ] End card links are correct (live URLs, not localhost)
- [ ] Captions are synced for muted playback
- [ ] Alt cuts export cleanly at correct aspect ratios
- [ ] Blog post `docs/blog/2026-04-20-remote-workspaces/index.md` is published before the video goes live (avoid broken link in end card)

---

*Source files: repo at `content/blog/memory-backup-restore`. All assets committed. Push pending on GH_TOKEN refresh — video editor can begin assembly now using staged files.*
