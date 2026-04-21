# Phase 30 Demos — DevRel Package

Demo specs for two Phase 30-adjacent features requiring working demonstrations.

---

## Demo 1: #1172 — AGENTS.md Auto-Generation

**Issue:** `Molecule-AI/internal#1172`  
**PR:** `molecule-core#763`  
**Feature:** `workspace/agents_md.py` — auto-generates `AGENTS.md` at boot using the AAIF standard  
**Acceptance:** working demo + repo link + 1-min screencast

### Files
| File | Description |
|---|---|
| `marketing/demos/agents-md-auto-generation/README.md` | Full working demo, API calls, screencast outline, TTS narration |
| `marketing/demos/agents-md-auto-generation/narration.mp3` | 30s narration audio |

### Screencast (1 min)
1. Canvas: pm-agent + researcher online
2. Terminal: read PM's AGENTS.md via platform files API
3. AGENTS.md output shown: role, A2A endpoint, tools
4. Researcher sends A2A task to PM using discovered endpoint
5. Canvas shows both active — close on "agents that can read each other"

### Repo link
`workspace/agents_md.py` on `molecule-core` main  
Direct: `workspace/agents_md.py`

---

## Demo 2: #1173 — Cloudflare Artifacts Integration

**Issue:** `Molecule-AI/internal#1173`  
**PR:** `molecule-core#641`  
**Feature:** `POST/GET /workspaces/:id/artifacts`, fork, token endpoints — "Git for agents"  
**Acceptance:** workspace snapshot to/from CF Artifacts + 1-min screencast

### Files
| File | Description |
|---|---|
| `marketing/demos/cloudflare-artifacts/README.md` | Full working demo, API calls, screencast outline, TTS narration |
| `marketing/demos/cloudflare-artifacts/narration.mp3` | 30s narration audio |

### Screencast (1 min)
1. Canvas: workspace online
2. Terminal: `POST /workspaces/:id/artifacts` — repo created, remote URL returned
3. Mint git credential, `git clone` with authenticated URL
4. Write snapshot, `git push` — push succeeds
5. Fork call: `POST /workspaces/:id/artifacts/fork` — new repo created
6. Close on "versioned agent state, built into the platform"

### Repo link
`workspace-server/internal/handlers/artifacts.go` on `molecule-core` main  
Direct: `workspace-server/internal/handlers/artifacts.go`

---

## Audio Assets

| File | Duration | Voice | Description |
|---|---|---|---|
| `agents-md-auto-generation/narration.mp3` | ~30s | en-US-AriaNeural | AGENTS.md auto-generation narration |
| `cloudflare-artifacts/narration.mp3` | ~30s | en-US-AriaNeural | Cloudflare Artifacts narration |