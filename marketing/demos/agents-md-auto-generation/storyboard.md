# Screencast Storyboard — AGENTS.md Auto-Generation

> **PR:** #763 | **Feature:** `workspace/agents_md.py` | **Duration:** 60 seconds
> **Format:** Terminal-led with Canvas overlay cuts

---

## Pre-roll (0:00–0:03)

**Canvas — full screen**
Two workspace cards in Canvas: `pm-agent [ONLINE]` and `researcher [IDLE]`.

Narration (VO, 0:00–0:03):
> "Two agents. The PM coordinates. The researcher does the work. They need to talk to each other — without humans in the loop."

**Camera:** Static Canvas view. No cursor movement. Clean frame.

---

## Moment 1 — PM boots, AGENTS.md generated (0:03–0:12)

**Cut to:** Terminal window, terminal prompt: `agent@pm-workspace:~$`

```bash
# Simulate the workspace startup — truncated log
INFO main: Starting workspace pm-agent
INFO agents_md: Generating AGENTS.md for workspace 'pm-agent'
INFO agents_md: Generated AGENTS.md at /workspace/AGENTS.md
INFO a2a: A2A server listening on :8000
INFO main: Workspace 'pm-agent' online
```

**Camera:** Type-in animation. Cursor blinks. Text appears line by line (simulate with playback speed 2x).

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

**Cut to:** Terminal output (scroll):

```
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

**Camera:** Scroll to show the full file. Hold 2s.

Narration (0:14–0:22):
> "The researcher reads the PM's AGENTS.md — through the platform API. Instantly knows the PM's role, its A2A endpoint, and the tools it has."

**Highlight:** `A2A Endpoint` and `MCP Tools` lines — brief underline pulse.

**Callout text appears bottom-left:**
`No system prompts. No documentation lookup. Just the facts.`

---

## Moment 3 — Researcher dispatches A2A task (0:25–0:42)

**Terminal continues:**

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

```
{"task_id": "task-abc-456", "status": "queued", "pm_receipt": "2026-04-21T00:00:22Z"}
```

**Camera:** Type-in animation. Brief hold on result JSON.

Narration (0:27–0:35):
> "Now the researcher has everything it needs. It sends an A2A task to the PM — using the endpoint it discovered from AGENTS.md. No hardcoded addresses."

---

## Moment 4 — PM receives task (0:42–0:52)

**Cut to:** Canvas — pm-agent card.

New message bubble appears in pm-agent's canvas chat:
`researcher: Status report — data-pipeline complete. 1 artifact ready.`

Below the message, status indicator changes: `pm-agent [ACTIVE]`

Researcher card shows: `researcher [DISPATCHED]`

Narration (0:42–0:48):
> "The PM receives it in Canvas. Status updated. The coordination happened without human input — AAIF in action."

---

## Close (0:52–1:00)

**Canvas — full frame.** Both cards visible. `pm-agent [ACTIVE]` + `researcher [DISPATCHED]`.

Narration (0:52–0:58):
> "AGENTS.md means every agent knows what its peers can do — without reading system prompts. Auto-generated. Always current. That's the AAIF standard, from Molecule AI."

**End card:**

```
AGENTS.md Auto-Generation
workspace/agents_md.py — molecule-core#763
```

**Fade to black.**

---

## Production Notes

- **Terminal theme:** Dark, monospace, minimal chrome. Use `ITerm2` profile "Molecule Dark" or equivalent.
- **Font:** SF Mono 14pt or JetBrains Mono 13pt.
- **Canvas cutaways:** Use the dev canvas at `localhost:3000` with two workspaces in active states. Pre-record these moments.
- **Camera:** Screenflow or Camtasia for macOS. Record at 1440×900, export at 1080p.
- **VO recording:** Record after final edit is locked. Use `en-US-AriaNeural` as reference voice for timing.
- **Narration pacing:** Read the script against the timeline before locking the VO session.
- **Music:** No music — keep it clean and technical. Consider a subtle 2s click sound at 0:03 (boot log) to anchor the start.
- **Highlights:** Use a yellow/amber ring `#E8A000` with 1s fade-in/out for callouts.
- **End card:** Centered, white text on dark background. 1080p canvas.
