# Snapshot Secret Scrubber — Working Demo

> **PR:** #977 — `feat(workspace): snapshot secret scrubber (closes #823)`
> **Source module:** `workspace/lib/snapshot_scrub.py`
> **What it does:** Strips API keys, auth tokens, and arbitrary subprocess output from workspace memory snapshots before they are serialized for hibernation

---

## What This Demo Shows

Before a workspace serializes its memory for hibernation, every memory entry passes through the scrubber. API keys, bearer tokens, env-var assignments, and high-entropy base64 blobs are redacted. Sandbox-sourced entries (arbitrary subprocess output from `run_code`) are dropped entirely.

This prevents an attacker who obtains a snapshot blob from recovering credentials or secrets that were processed during the agent session.

---

## Runnable Snippet

```python
from workspace.lib.snapshot_scrub import (
    scrub_memory_entry,
    scrub_snapshot,
    scrub_content,
    is_sandbox_content,
)

# 1. Scrub a single memory entry
entry = {
    "id": "mem-001",
    "source": "agent",
    "content": "ANTHROPIC_API_KEY=sk-ant-xxxx configured for claude-3-5"
}
cleaned = scrub_memory_entry(entry)
print(cleaned["content"])
# Output: ANTHROPIC_API_KEY=API_KEY [redacted]

# 2. Scrub an entire snapshot before serialization
snapshot = {
    "workspace_id": "ws-abc",
    "memories": [
        {"id": "mem-002", "source": "agent", "content": "GitHub token: ghp_AbCdEfGhIjKlMnOpQrStUvW"},
        {"id": "mem-003", "source": "sandbox", "content": "source=sandbox: echo $SECRET"},
        {"id": "mem-004", "source": "agent", "content": "Bearer 0xdeadbeef used for /api endpoint"},
    ]
}
scrubbed = scrub_snapshot(snapshot)
print(scrubbed["memories"])
# Output: 2 entries (sandbox entry dropped entirely, other two scrubbed)

# 3. Just the scrub function directly
redacted = scrub_content("OPENAI_API_KEY=sk-proj-1234567890abcdef")
print(redacted)
# Output: OPENAI_API_KEY=SK_TOKEN [redacted]
```

---

## Context

**Why it matters:** Memory snapshots are the workspace's serialized state — saved to disk or transmitted for cross-workspace delegation. If the workspace processed an API key during its session, that key must not survive in the snapshot. `snapshot_scrub.py` is the gatekeeper.

**What gets scrubbed:** API key patterns (`sk-ant-`, `sk-proj-`, `ghp_`, `ghs_`, `AKIA…`, `cfut_`, `mol_pk_`, `ctx7_`), bearer token header values, env-var assignments, and high-entropy base64 blobs (33+ chars). Sandbox-sourced entries (`source=sandbox`, `tool=run_code`, `[sandbox_output]`) are dropped wholesale — they can contain arbitrary subprocess output that can't be safely pattern-matched.

**What doesn't change:** Workspace metadata, config fields, and non-secret memory entries pass through unchanged. The scrubber is a pure function — it's unit-tested independently and has no side effects.

---

## Code Reference

| File | What it does |
|---|---|
| `workspace/lib/snapshot_scrub.py` | Core module: `scrub_content()`, `scrub_memory_entry()`, `scrub_snapshot()`, `is_sandbox_content()` |
| `workspace/tests/test_snapshot_scrub.py` | 21 unit tests covering all pattern classes |
| `workspace/lib/__init__.py` | Module exports |
