# Screencast Production Handoff — Issue #1303
**Owner:** DevRel Engineer | **Status:** READY TO PRODUCE
**Updated:** 2026-04-21 by Marketing Lead

---

## Background

Issue #1303 ("Phase 30: record 4 launch screencasts") is ready for production. All storyboards are complete. This is the standing dispatch — DevRel Engineer owns recording and production.

**Hero video assembly** is a separate job owned by Content Marketer. See `phase30-video-production.md`.

---

## Your 4 Screencasts

### 1. EC2 Console Output Demo
**Source:** PR #68 | **Duration:** ~60s | **Format:** Canvas UI → terminal
- Storyboard: `marketing/demos/failed-workspace-ec2-console-demo.md`
- Reference: Canvas screenshot (dark zinc theme, failed workspace card + EC2 Console tab)
- TTS: 30s, use `phase30-announce.mp3` cadence as reference
- End card: `workspace-server/internal/handlers/container_files.go — molecule-core#1178`

### 2. Cloudflare Artifacts Demo
**Source:** PR #641 | **Duration:** ~60s | **Format:** Terminal-led, dark zinc theme
- Storyboard: `marketing/demos/cloudflare-artifacts/storyboard.md`
- TTS narration: `marketing/demos/cloudflare-artifacts/narration.mp3` ✅ (already recorded, use directly)
- Reference: Fleet diagram (`marketing/assets/phase30-fleet-diagram.png`)
- End card: `workspace-server/internal/handlers/artifacts.go — molecule-core#641`

### 3. MemoryInspectorPanel Demo
**Source:** PR #65 | **Duration:** ~60s | **Format:** Canvas UI + browser
- Storyboard: `marketing/demos/memory-inspector-panel/storyboard.md`
- Reference: Canvas screenshot — MemoryInspectorPanel with 10+ entries, similarity scores visible
- Source component: `canvas/src/components/MemoryInspectorPanel.tsx`
- End card: `canvas/src/components/MemoryInspectorPanel.tsx — molecule-core#1127`
- Note: Canvas needs 10+ memory entries pre-seeded (mock data fine)

### 4. Snapshot Secret Scrubber Demo
**Source:** PR #63 | **Duration:** ~60s | **Format:** Terminal + code walkthrough
- Storyboard: `marketing/demos/snapshot-scrub/storyboard.md`
- Reference: Terminal screenshot — `scrub_content()` function with test output
- Source: `workspace-server/internal/workspace/snapshot_scrub.py`
- End card: `workspace-server/internal/workspace/snapshot_scrub.py — molecule-core#977`
- Note: Pairs with Cloudflare Artifacts screencast — scrubber is why agents can safely version workspace state in CF Artifacts

---

## Production Spec (all 4)

| Spec | Value |
|------|-------|
| Format | 1080p H.264, 30fps |
| Aspect ratios | 16:9 (primary) + 9:16 (social cut) |
| Theme | Dark zinc #0f0f11, JetBrains Mono 14pt, blue-500 (#3b82f6) highlights, amber (#E8A000) callout rings |
| Captions | Burn in for muted playback |
| Music | None on primary cuts. Single-tone click at key transition moments per storyboard. |
| Duration | ~60s (+/- 5s) |

---

## Self-Review Gate (all 4 must pass)

- [ ] Recording is ~60s (+/- 5s)
- [ ] Dark zinc theme + blue accents
- [ ] All callout text readable (contrast + size)
- [ ] End card with source file + PR number present
- [ ] TTS narration synced (if VO used)
- [ ] No person names, no benchmark numbers, no competitor names in narration

---

## Output Location

`docs/marketing/devrel/demos/[screencast-name]/[name].mp4`

e.g. `docs/marketing/devrel/demos/cloudflare-artifacts/cloudflare-artifacts-demo.mp4`

---

## Brand Audio

- Cloudflare Artifacts: use `narration.mp3` directly (already recorded)
- EC2 Console + MemoryInspectorPanel + Snapshot Scrubber: generate TTS via edge-tts using the script text in each storyboard
- No music on primary cuts
- Single-tone click at transition moments per storyboard production notes

---

*Marketing Lead dispatch. DevRel Engineer to produce all 4 and report back with output file paths.*
