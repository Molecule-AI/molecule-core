"""Pre-stop serialization for pause/resume — GH#1391.

Captures the agent's in-memory state just before the container exits so it
survives intentional pause and unplanned restart. All content is scrubbed
with lib.snapshot_scrub before being written to disk so that a snapshot blob
obtained by an attacker cannot recover API keys, tokens, or arbitrary sandbox
output (GH#823).

State captured
--------------
- ``workspace_id``           — identity for cross-container restore
- ``current_task``           — active task label from heartbeat (what the canvas sees)
- ``active_tasks``           — task count
- ``session_id``             — SDK session handle (Claude Code); key for full session
- ``transcript_lines``        — recent session log lines from the adapter
- ``uptime_seconds``         — how long this container has been running
- ``timestamp``              — when the snapshot was taken (ISO-8601)

Scrubbing
---------
Every text field passes through scrub_snapshot before being written.
Sandbox-sourced content (tool=run_code, source=sandbox, [sandbox_output]) is
dropped wholesale. Secrets matching the pattern library are replaced with
[REDACTED:TYPE] markers.

Storage
-------
Snapshots are written to /configs/.agent_snapshot.json by default. The
config volume survives container restarts so the file is durable. The path
is also overridable via ``AGENT_SNAPSHOT_PATH`` for testing or custom layouts.
"""

from __future__ import annotations

import json
import logging
import os
from datetime import datetime, timezone
from typing import TYPE_CHECKING, Any

from .snapshot_scrub import scrub_snapshot

if TYPE_CHECKING:
    from heartbeat import HeartbeatLoop

logger = logging.getLogger(__name__)

# Default snapshot path — on the config volume, survives container restarts.
DEFAULT_SNAPSHOT_PATH = os.environ.get(
    "AGENT_SNAPSHOT_PATH",
    "/configs/.agent_snapshot.json",
)

# How many transcript lines to capture in the snapshot (recent window).
MAX_TRANSCRIPT_LINES = 200


def build_snapshot(
    heartbeat: "HeartbeatLoop | None",
    adapter_state: dict[str, Any],
) -> dict[str, Any]:
    """Build a raw snapshot dict from live workspace state.

    Args:
        heartbeat:      HeartbeatLoop instance; provides current_task, session_id, etc.
        adapter_state:  Arbitrary state dict from the adapter's pre_stop_state() hook.
                        Keys are free-form; all string values in nested dicts/lists are
                        scrubbed before writing.

    Returns a raw (not yet scrubbed) snapshot dict.
    """
    import time

    raw: dict[str, Any] = {
        "workspace_id": os.environ.get("WORKSPACE_ID", "unknown"),
        "timestamp": datetime.now(timezone.utc).isoformat(),
        # Defaults — heartbeat block below overwrites these when available:
        "current_task": "",
        "active_tasks": 0,
    }

    if heartbeat is not None:
        raw["current_task"] = heartbeat.current_task or ""
        raw["active_tasks"] = heartbeat.active_tasks
        if hasattr(heartbeat, "start_time"):
            raw["uptime_seconds"] = int(time.time() - heartbeat.start_time)
        # session_id lives in the adapter but we also accept it via heartbeat
        # for convenience (avoids requiring every adapter to pass it separately).
        if not adapter_state.get("session_id"):
            raw["session_id"] = getattr(heartbeat, "_session_id", None) or ""

    # Adapter-supplied state (conversation history, reasoning traces, etc.)
    raw["adapter"] = adapter_state

    return raw


def _scrub_value(value: Any) -> Any:
    """Recursively scrub all secret patterns from a value.

    - Strings:  scrub_content() replaces patterns with [REDACTED:TYPE].
    - Dicts:    return a new dict with all values scrubbed recursively.
    - Lists:    drop entries that are sandbox content; scrub remaining items.
    - Other:    pass through unchanged.
    """
    from .snapshot_scrub import is_sandbox_content, scrub_content

    if isinstance(value, str):
        return scrub_content(value)
    if isinstance(value, dict):
        return {k: _scrub_value(v) for k, v in value.items()}
    if isinstance(value, list):
        result = []
        for item in value:
            if isinstance(item, str) and is_sandbox_content(item):
                continue  # Drop sandbox entries wholesale
            result.append(_scrub_value(item))
        return result
    return value


def write_snapshot(
    snapshot: dict[str, Any],
    path: str | None = None,
) -> bool:
    """Scrub and write a snapshot to disk.

    Args:
        snapshot:  Raw snapshot dict from build_snapshot().
        path:     Target file path (default: DEFAULT_SNAPSHOT_PATH).

    Returns:
        True if the snapshot was written successfully; False on any error.
        Errors are logged but never raise — pre-stop serialization must be
        best-effort to avoid blocking shutdown.
    """
    target = path or DEFAULT_SNAPSHOT_PATH

    try:
        # Deep-scrub every string value in the snapshot to remove API keys,
        # tokens, and arbitrary sandbox output before writing to disk.
        scrubbed = _scrub_value(snapshot)

        # Ensure parent directory exists.
        parent = os.path.dirname(target)
        if parent:
            os.makedirs(parent, exist_ok=True)

        with open(target, "w") as f:
            json.dump(scrubbed, f, indent=2, default=str)

        logger.info(
            "Pre-stop snapshot written: %s (workspace=%s, task=%r, lines=%d)",
            target,
            scrubbed.get("workspace_id", "?"),
            scrubbed.get("current_task", ""),
            len(scrubbed.get("adapter", {}).get("transcript_lines", [])),
        )
        return True

    except Exception as exc:
        logger.warning("Pre-stop snapshot write failed (%s): %s", target, exc)
        return False


def read_snapshot(
    path: str | None = None,
) -> dict[str, Any] | None:
    """Read and return a previously-written snapshot, or None if absent/invalid."""
    target = path or DEFAULT_SNAPSHOT_PATH

    if not os.path.exists(target):
        return None

    try:
        with open(target) as f:
            return json.load(f)
    except Exception as exc:
        logger.debug("Snapshot read failed (%s): %s", target, exc)
        return None


def delete_snapshot(path: str | None = None) -> None:
    """Remove a snapshot file. Idempotent — no error if absent."""
    target = path or DEFAULT_SNAPSHOT_PATH
    try:
        os.remove(target)
        logger.debug("Snapshot deleted: %s", target)
    except FileNotFoundError:
        pass
    except Exception as exc:
        logger.warning("Snapshot delete failed (%s): %s", target, exc)
