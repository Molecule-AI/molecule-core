# Screencast Storyboard — AGENTS.md Auto-Generation
**PR:** #763 | **Feature:** `workspace/agents_md.py` | **Duration:** 60 seconds
**Format:** Terminal-led with Canvas overlay cuts

---

## Pre-roll (0:00–0:03)

**Canvas — full screen**
Two workspace cards in Canvas: `pm-agent [ONLINE]` and `researcher [IDLE]`.

Narration (0:00–0:03):
> "Two agents. The PM coordinates. The researcher does the work. They need to talk to each other — without humans in the loop."

**Camera:** Static Canvas view. No cursor movement. Clean frame.

---

## Moment 1 — PM boots, AGENTS.md generated (0:03–0:12)

**Cut to:** Terminal window, terminal prompt: `agent@pm-workspace:~$`

```bash
INFO main: Starting workspace pm-agent
INFO agents_md: Generating AGENTS.md for workspace 'pm-agent'
INFO agents_md: Generated AGENTS.md at /workspace/AGENTS.md
INFO a2a: A2A server listening on :8000
INFO main: Workspace 'pm-agent' online
```

**Camera:** Type-in animation. Cursor blinks. Text appears line by line (playback speed 2x).

Narration (0:06–0:12):
> "When the PM workspace starts up, AGENTS.md is generated automatically — from the config file, not a human."

**Highlight:** `INFO agents_md: Generated AGENTS.md at /workspace/AGENTS.md` — brief yellow highlight ring (1s).

---

## Moment 2 — Researcher reads PM's AGENTS.md (0:12–0:25)

**Cut to:** Second terminal tab. Prompt: `agent@researcher:~$`

```python
import requests
resp = requests.get(
    "https://acme.moleculesai.app/workspaces/ws-pm-123/files/AGENTS.md",
    headers={"Authorization": "Bearer researcher-token-xxx"},
)
print(resp.json()["content"])
```

**Terminal output:**
```markdown
# pm-agent
**Role:** Project Manager
## Description
PM agent — coordinates tasks, dispatches to reports, manages timeline.
## A2A Endpoint
http://pm-workspace:8000/a2a
## MCP Tools
- delegate_to_workspace
- check_delegation_status
```

**Camera:** Scroll to full file. Hold 2s.

Narration (0:14–0:22):
> "The researcher reads the PM's AGENTS.md — through the platform API. Instantly knows the PM's role, its A2A endpoint, and the tools it has."

**Callout text (bottom-left):**
`No system prompts. No documentation lookup. Just the facts.`

---

## Moment 3 — Researcher dispatches A2A task (0:25–0:42)

```python
from a2a import A2ATask
task = A2ATask(
    to="http://pm-workspace:8000/a2a",
    type="status_report",
    payload={
        "milestone": "data-pipeline",
        "status": "complete",
        "artifacts": ["dataset-v3.parquet"],
    }
)
result = task.send()
print(result)
```

**Terminal output:**
```json
{"task_id": "task-abc-456", "status": "queued", "pm_receipt": "2026-04-21T00:00:22Z"}
```

Narration (0:27–0:35):
> "Now the researcher has everything it needs. It sends an A2A task to the PM — using the endpoint it discovered from AGENTS.md. No hardcoded addresses."

---

## Moment 4 — PM receives task (0:42–0:52)

**Cut to:** Canvas — pm-agent card.

New message bubble: `researcher: Status report — data-pipeline complete. 1 artifact ready.`
Status: `pm-agent [ACTIVE]`, `researcher [DISPATCHED]`

Narration (0:42–0:48):
> "The PM receives it in Canvas. Status updated. The coordination happened without human input — AAIF in action."

---

## Close (0:52–1:00)

**Canvas full frame.** Both cards visible.

Narration (0:52–0:58):
> "AGENTS.md means every agent knows what its peers can do — without reading system prompts. Auto-generated. Always current. That's the AAIF standard, from Molecule AI."

**End card:**
```
AGENTS.md Auto-Generation
workspace/agents_md.py — molecule-core#763
```
**Fade to black.**

---

## Production Spec

| Spec | Value |
|------|-------|
| Terminal theme | Dark, SF Mono 14pt / JetBrains Mono 13pt |
| Canvas cutaway | Dev canvas localhost:3000, pre-record before session |
| Camera | Screenflow / Camtasia, 1440×900 → 1080p export |
| VO voice | en-US-AriaNeural (reference) |
| Callout highlight | Amber ring `#E8A000`, 1s fade-in/out |
| Green success | Green ring `#22C55E` for success moments |
| Music | None — clean and technical |
| Sound FX | Subtle 2s click at 0:03 (boot log) |
| VO pacing | Read script against timeline before locking VO session |
