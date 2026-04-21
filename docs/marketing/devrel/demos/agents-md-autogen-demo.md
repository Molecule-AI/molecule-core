# AGENTS.md Auto-Generation — Interactive Demo Script
**Issue:** #1172 | **Source:** PR #763 | **Acceptance:** Working demo + 1-min screencast

---

## What This Demo Shows

1. A workspace with a `role` and `description` in `config.yaml`
2. `generate_agents_md()` called at startup
3. The resulting `AGENTS.md` that peer agents can read
4. A second agent discovering the first via A2A

**Time:** ~60 seconds | **Language:** Python | **Key File:** `workspace-template/agents_md.py`

---

## Demo Script

### Step 1: Show the Source

```python
from agents_md import generate_agents_md

# Generate AGENTS.md from the workspace config
generate_agents_md(config_dir="/configs", output_path="/workspace/AGENTS.md")

# Read what was generated
print(Path("/workspace/AGENTS.md").read_text())
```

### Step 2: Show the Generated Output

Running the above on a workspace with:

```yaml
# config.yaml
name: Code Reviewer
role: Senior Code Reviewer
description: Reviews pull requests, flags security issues, suggests test coverage improvements.
a2a:
  port: 8000
tools:
  - read_file
  - write_file
  - search_code
plugins:
  - github
  - slack
```

Produces:

```markdown
# Code Reviewer

**Role:** Senior Code Reviewer

## Description
Reviews pull requests, flags security issues, suggests test coverage improvements.

## A2A Endpoint
http://localhost:8000/a2a

## MCP Tools
- read_file
- write_file
- search_code
- github
- slack
```

### Step 3: Show a Peer Agent Discovering It

```python
# A PM agent discovers the Code Reviewer via A2A
from a2a.client import A2AClient

client = A2AClient("http://codereviewer:8000/a2a")
card = client.discover()  # Reads their AGENTS.md

print(f"Discovered agent: {card.name} ({card.role})")
print(f"Available tools: {card.tools}")
```

Output:
```
Discovered agent: Code Reviewer (Senior Code Reviewer)
Available tools: ['read_file', 'write_file', 'search_code', 'github', 'slack']
```

**Narrative:** "No configuration files to maintain. No registry to update. Peer agents discover each other the same way humans discover each other — by reading each other's profiles."

---

## Screencast Outline (~60s)

| Time | Action |
|------|--------|
| 0–15s | Open `config.yaml` — show `role` field |
| 15–30s | Show `generate_agents_md()` call in `main.py` — "called at startup" |
| 30–45s | Run it — show the generated `AGENTS.md` |
| 45–60s | Show a second agent discovering the first via A2A — "peer agents find each other automatically" |

**Key visual:** The `AGENTS.md` file appearing in the Canvas sidebar — visible, always current, no manual sync.

---

## The AGENTS.md Standard

This implements the [AAIF / Linux Foundation AGENTS.md standard](https://github.com/AI-Agents/AGENTS.md). Key properties:

- **Self-describing** — agents publish their own identity, role, and tools
- **Startup-generated** — always current, no drift from config
- **A2A-native** — discovery happens over the A2A protocol, no external registry

---

## Files

- Demo script: `docs/marketing/devrel/demos/agents-md-autogen-demo.md`
- Source file: `workspace-template/agents_md.py` (PR #763)
