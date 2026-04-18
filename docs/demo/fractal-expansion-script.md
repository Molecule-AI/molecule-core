# Fractal Expansion Demo — Recording Script

This document specifies the exact steps, canvas state, and UI interactions
to record the hero GIF used in the README and marketing materials.

## Output Spec

| Property | Value |
|---|---|
| Format | `.gif` (or `.webm` converted to gif) |
| Resolution | `800 × 500 px` (2× for retina: record at 1600×1000, export at 800×500) |
| Frame rate | 20 fps |
| Max file size | 5 MB (use `gifsicle -O3` or `gifski` to compress) |
| Duration | ~12 seconds |
| Loop | Infinite |
| Alt text (for README) | `Molecule AI fractal expansion: a single Engineering Lead node expands into a Frontend Dev, Backend Dev, and QA sub-team, then an A2A task arrives and escalates to human approval` |

---

## Recording Tool Setup

**Recommended:** [Kap](https://getkap.co/) (macOS) or [LICEcap](https://www.cockos.com/licecap/) (Windows/macOS)

For highest quality, use [ScreenToGif](https://www.screentogif.com/) (Windows) or
record with QuickTime → convert with `ffmpeg` + `gifski`:

```bash
# Convert QuickTime .mov → high-quality GIF
ffmpeg -i recording.mov -vf "fps=20,scale=800:-1:flags=lanczos" frames/frame%04d.png
gifski --fps 20 --width 800 -o fractal-expansion.gif frames/*.png
```

---

## Pre-Recording Setup

### 1. Environment
- Run `docker compose up` and wait for all services to be healthy.
- Open Chrome at `http://localhost:3000`.
- Set browser zoom to 100%.
- Open DevTools → Network tab → enable "Slow 3G" throttling OFF (use full speed).
- Hide bookmarks bar for a clean capture area.

### 2. Canvas State (before recording)
Ensure the canvas has **exactly one workspace node** visible:

```
┌─────────────────────────────────────┐
│                                     │
│                                     │
│      ┌──────────────────────┐       │
│      │  🏗️  Engineering Lead │       │
│      │     status: online   │       │
│      └──────────────────────┘       │
│                                     │
│                                     │
└─────────────────────────────────────┘
```

Use `setup-org.sh` to provision the org, then delete all nodes except the
Engineering Lead. Or provision a fresh single workspace via:

```bash
curl -X POST http://localhost:8080/workspaces \
  -H "Content-Type: application/json" \
  -d '{"name":"Engineering Lead","role":"Engineering Lead","tier":1}'
```

### 3. Canvas Viewport
Pan and zoom so the Engineering Lead node is centered, slightly left of centre,
with breathing room above for the sub-nodes to appear during expansion.
Save viewport: `PUT /canvas/viewport`.

---

## Scene-by-Scene Script

### Scene 1 — Establishing Shot (0:00 – 1:00)

**Duration:** ~1 second (pause on static canvas)

**Canvas state:**
- Single "Engineering Lead" node, `status: online` (green dot)
- Node is centred on canvas

**Narrator voiceover / caption (optional):**
> "One node. One role."

---

### Scene 2 — Right-click → Expand to Team (1:00 – 2:00)

**Duration:** ~1 second

**Action:**
1. Move mouse smoothly to the Engineering Lead node (no jitter — use a mouse
   recording tool with smoothing enabled)
2. Right-click the node to open the context menu
3. The context menu appears with options including **"Expand to Team"**
4. Hover over "Expand to Team" — it highlights

**UI detail:**
The context menu is rendered by `WorkspaceContextMenu` in the canvas.
"Expand to Team" appears as the second item below "View Details".

---

### Scene 3 — Sub-nodes Materialise (2:00 – 5:00)

**Duration:** ~3 seconds

**Action:**
1. Click **"Expand to Team"**
2. The platform calls `POST /workspaces/:id/expand`
3. Three child nodes animate into view — they should slide in from the
   Engineering Lead node with the default React Flow `fade` transition:

```
          ┌──────────────────────┐
          │  🏗️  Engineering Lead │
          │     status: online   │
          └──────────┬───────────┘
                     │
          ┌──────────┼──────────┐
          │          │          │
   ┌──────┴──┐  ┌────┴────┐  ┌──┴──────┐
   │Frontend │  │Backend  │  │  QA     │
   │  Dev    │  │  Dev    │  │Engineer │
   └─────────┘  └─────────┘  └─────────┘
```

**Timing note:** If provisioning takes > 3 seconds in your recording, set the
workspace tier to 1 (no Docker pull needed) and pre-build the workspace image
(`docker build -t workspace-template:latest workspace/`).

---

### Scene 4 — Nodes Come Online (5:00 – 7:00)

**Duration:** ~2 seconds

**Action:**
- All three child nodes transition from `provisioning` (grey dot) → `online` (green dot)
- The Engineering Lead's tier badge updates to show "Team Lead" indicator
- WebSocket events trigger the canvas update in real time (no manual refresh)

**Visual target:**
All four nodes showing green online dots.

---

### Scene 5 — A2A Task Arrives (7:00 – 9:00)

**Duration:** ~2 seconds

**Action:**
1. Using the MCP server or `curl`, send an A2A message to the Engineering Lead:

```bash
curl -X POST http://localhost:8080/workspaces/<engineering-lead-id>/a2a \
  -H "Content-Type: application/json" \
  -d '{
    "method": "message/send",
    "params": {
      "message": {
        "role": "user",
        "parts": [{"kind": "text", "text": "Ship the login feature by Friday"}]
      }
    }
  }'
```

2. An amber **"current task"** banner appears on the Engineering Lead node:
   > *"Decomposing: Ship login feature"*

---

### Scene 6 — Approval Escalation (9:00 – 12:00)

**Duration:** ~3 seconds

**Action:**
1. The Engineering Lead detects a high-risk action (e.g., "deploy to production")
   and creates an approval request via `POST /workspaces/:id/approvals`
2. An **approval card** animates into view at the top of the canvas:
   ```
   ┌─────────────────────────────────────────────┐
   │ ⚠️  Approval Required                        │
   │ "Deploy login service to production?"        │
   │ Requested by: Engineering Lead               │
   │  [Approve]  [Deny]                           │
   └─────────────────────────────────────────────┘
   ```
3. End on this frame — the GIF loops back to Scene 1 (static canvas)

**How to trigger an approval for the demo:**
```bash
curl -X POST http://localhost:8080/workspaces/<engineering-lead-id>/approvals \
  -H "Content-Type: application/json" \
  -d '{
    "question": "Deploy login service to production?",
    "context": "The login feature is complete and passes all tests.",
    "risk_level": "high"
  }'
```

---

## Post-Processing

```bash
# Compress with gifsicle (reduces file size 40-60%)
gifsicle -O3 --lossy=80 fractal-expansion-raw.gif -o fractal-expansion.gif

# Verify file size
ls -lh fractal-expansion.gif    # target: < 5 MB

# Save to:
cp fractal-expansion.gif docs/demo/fractal-expansion.gif
```

Then update `README.md` — replace the placeholder comment with:
```markdown
![Molecule AI fractal expansion demo](./docs/demo/fractal-expansion.gif)
```

---

## Checklist Before Publishing

- [ ] GIF file size < 5 MB
- [ ] Resolution exactly 800 × 500 px
- [ ] All nodes visible and labelled
- [ ] No personally identifiable info in terminal/UI
- [ ] Alt text set in README img tag
- [ ] Loops cleanly (last frame matches first frame visually)
- [ ] Tested in GitHub README preview (GitHub caps animated GIFs at 10 MB)
