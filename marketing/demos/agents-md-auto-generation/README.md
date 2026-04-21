# AGENTS.md Auto-Generation — Working Demo

> **PR:** #763 — AGENTS.md auto-generation for Molecule AI workspaces  
> **What it ships:** `workspace/agents_md.py` — generates `AGENTS.md` at boot  
> **Acceptance criteria:** working demo + repo link + 1-min screencast

---

## What This Demo Shows

An AI agent (the "coordinator") reads another agent's `AGENTS.md` file to discover its identity, A2A endpoint, and toolset — without reading the full system prompt. This is the AAIF / Linux Foundation AGENTS.md standard in action.

**The flow:**
1. A PM workspace starts up — `agents_md.py` auto-generates `AGENTS.md`
2. A researcher workspace starts up — same process
3. The researcher reads the PM's `AGENTS.md` to understand what tools it has and how to reach it
4. The researcher dispatches a task to the PM via A2A using the discovered endpoint

---

## Prerequisites

- Molecule AI platform running (`go run ./cmd/server` from `workspace-server/`)
- Canvas open at `http://localhost:3000`
- Two workspaces: one running as PM role, one as researcher
- For the script demo: `python3` and `requests`

---

## Working Demo Script

### 1. Check the AGENTS.md file on a running workspace

On the PM workspace container:

```bash
# Inside the PM workspace container
cat /workspace/AGENTS.md
```

Expected output:
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
- commit_memory
- recall_memory
```

The file was generated automatically at boot by `agents_md.py`. It reflects the workspace `config.yaml` in real time — any change to the role, description, or plugins is reflected on the next regeneration.

---

### 2. See the generation in the workspace logs

The workspace startup log includes:

```
INFO agents_md: Generated AGENTS.md at /workspace/AGENTS.md for workspace 'pm-agent'
```

This confirms `generate_agents_md()` ran as part of `main.py` startup.

---

### 3. See the regeneration on config change

If you edit `config.yaml` and call `generate_agents_md()` again:

```bash
# On the PM workspace
python3 -c "
from agents_md import generate_agents_md
generate_agents_md('/configs', '/workspace/AGENTS.md')
print('Regenerated')
"
cat /workspace/AGENTS.md
```

The file reflects the updated role or description immediately.

---

### 4. See a peer agent read the AGENTS.md (demo scenario)

This is the coordination moment — the scenario from issue #1172.

```python
# Researcher workspace: read PM's AGENTS.md via the platform files API

import requests, base64

PLATFORM_URL = "http://localhost:8080"
WORKSPACE_TOKEN = "researcher-workspace-token"

# Get the PM workspace ID (known from canvas or registry)
# For this demo: PM workspace ID = ws-pm-123

# Read PM's AGENTS.md via the platform's file API
resp = requests.get(
    f"{PLATFORM_URL}/workspaces/ws-pm-123/files/AGENTS.md",
    headers={"Authorization": f"Bearer {WORKSPACE_TOKEN}"},
)
print(resp.json()["content"])
```

Parses the PM's `AGENTS.md`:
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

Now the researcher knows:
- PM's role is "Project Manager" → it dispatches, not executes
- PM's A2A endpoint → where to send coordination requests
- PM has `delegate_to_workspace` tool → it can cascade tasks to reports

The researcher then uses this to coordinate: sends a status report to the PM, knowing the PM will route it up or dispatch a follow-up task.

---

## Screencast Outline (1 min)

**0:00–0:10** Canvas shows two workspaces online — pm-agent and researcher. Researcher node shows current task: "idle".

**0:10–0:25** Terminal on researcher workspace: `curl` or Python script reads PM's `AGENTS.md` via the platform files API. Output shows the PM's role, A2A endpoint, and tools.

**0:25–0:40** Researcher sends an A2A task to the PM: "Status: data pipeline complete, ready for review." PM receives it in its canvas chat.

**0:40–0:55** PM's `AGENTS.md` is shown briefly in the researcher terminal — the researcher used it to understand PM's capabilities before sending the task.

**0:55–1:00** Canvas shows both workspaces active. Narration: *"AGENTS.md means every agent knows what its peers can do — without reading system prompts."*

---

## Code Reference

| File | What it does |
|---|---|
| `workspace/agents_md.py` | `generate_agents_md()` — reads `config.yaml`, writes `AGENTS.md` |
| `workspace/main.py` | Calls `generate_agents_md()` at startup |
| `config.py` | `load_config()` — reads `config.yaml` |

**Source:** `workspace/agents_md.py` (PR #763)

```python
from agents_md import generate_agents_md

# Called automatically at startup; can be called again on config change
generate_agents_md(config_dir="/configs", output_path="/workspace/AGENTS.md")
```

---

## TTS Narration Script (30s)

> When a PM agent starts up in Molecule AI, it generates an AGENTS.md file automatically — not manually written, not kept in sync by hand. It reflects the workspace config in real time. Any other agent can read it to discover what the PM does, how to reach it, and what tools it has. No system prompts, no guessing. Just the facts. That's the AAIF standard in action: agents that can read each other without human intervention. AGENTS.md auto-generation, from Molecule AI workspace.