"""Helpers for the workspace's one-shot initial_prompt.

Kept as a standalone module (no heavy imports like uvicorn) so the marker
logic is unit-testable without standing up the full workspace runtime.

Background: the workspace runtime supports an `initial_prompt` that runs once
on first boot (clone the repo, set git hooks, read CLAUDE.md, commit_memory).
A marker file `.initial_prompt_done` prevents the prompt from re-running on
subsequent boots.

Prior behaviour wrote the marker AFTER the prompt completed successfully. If
the prompt crashed mid-execution (e.g. ProcessError from a stale Claude
session), the marker was never written; every subsequent container boot
replayed the same failing prompt, cascading into "every message crashes until
an operator intervenes." See GitHub issue #71.

Fix (2026-04-12): write the marker BEFORE firing the prompt. If the prompt
fails, operators re-send it manually via chat — cheap and available — instead
of trapping the workspace in a crash loop.
"""
from __future__ import annotations

import os


def resolve_initial_prompt_marker(config_path: str) -> str:
    """Return the path where the `.initial_prompt_done` marker should live.

    Prefers ``<config_path>/.initial_prompt_done`` when the directory is
    writable; falls back to ``/workspace/.initial_prompt_done`` for containers
    where ``/configs`` is read-only.
    """
    if os.access(config_path, os.W_OK):
        return os.path.join(config_path, ".initial_prompt_done")
    return "/workspace/.initial_prompt_done"


def mark_initial_prompt_attempted(marker_path: str) -> bool:
    """Write the marker best-effort. Return True on success, False on I/O error.

    Called BEFORE the initial-prompt self-message is sent. If the attempt
    later fails, the marker is still present — so the next container boot
    does NOT replay the same failing prompt. Operators retry manually via
    the chat interface instead of relying on auto-replay.
    """
    try:
        with open(marker_path, "w") as f:
            f.write("attempted")
        return True
    except OSError:
        return False
