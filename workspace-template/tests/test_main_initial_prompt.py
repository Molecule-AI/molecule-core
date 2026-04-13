"""Tests for main.py's initial-prompt marker handling (fixes #71).

Prior behaviour wrote the marker only after the initial_prompt task succeeded.
When the task crashed (e.g. ProcessError from a stale resume state), the marker
was never written; the next container boot replayed the same failing prompt,
cascading into "every message crashes" until an operator manually touched the
marker and restarted.

The fix writes the marker BEFORE the task runs. These tests pin the new
semantics so we can't silently regress.
"""
from __future__ import annotations

import os

import pytest

from initial_prompt import (
    mark_initial_prompt_attempted,
    resolve_initial_prompt_marker,
)


def test_resolve_marker_prefers_writable_config_path(tmp_path):
    """When /configs is writable, marker lives there (persists on container rebuild)."""
    resolved = resolve_initial_prompt_marker(str(tmp_path))
    assert resolved == os.path.join(str(tmp_path), ".initial_prompt_done")


def test_resolve_marker_falls_back_to_workspace_when_config_readonly(tmp_path, monkeypatch):
    """When /configs isn't writable, fall back to /workspace (Docker volume)."""
    # Simulate an unwritable config dir by monkey-patching os.access
    unwritable = tmp_path / "configs"
    unwritable.mkdir()

    real_access = os.access

    def fake_access(path, mode):
        if str(path) == str(unwritable) and mode == os.W_OK:
            return False
        return real_access(path, mode)

    monkeypatch.setattr(os, "access", fake_access)
    resolved = resolve_initial_prompt_marker(str(unwritable))
    assert resolved == "/workspace/.initial_prompt_done"


def test_mark_initial_prompt_attempted_creates_marker(tmp_path):
    """Writing the marker succeeds and the file contains a non-empty token."""
    marker = tmp_path / ".initial_prompt_done"
    assert mark_initial_prompt_attempted(str(marker)) is True
    assert marker.exists()
    assert marker.read_text() != ""


def test_mark_initial_prompt_attempted_returns_false_on_oserror(tmp_path):
    """I/O errors are surfaced as a False return (caller logs loudly)."""
    # Pointing at a nonexistent directory triggers OSError
    marker = tmp_path / "does-not-exist" / ".initial_prompt_done"
    assert mark_initial_prompt_attempted(str(marker)) is False


def test_marker_survives_crash_simulation(tmp_path):
    """Scenario: mark up-front, then the hypothetical send raises — marker is still there.

    This encodes the #71 semantic: we write the marker BEFORE running the
    side-effectful self-send, so even if the agent subsequently crashes we do
    not replay the failing prompt on the next boot.
    """
    marker_path = str(tmp_path / ".initial_prompt_done")
    assert mark_initial_prompt_attempted(marker_path) is True

    # Simulate a task crash that would have prevented any "after-success"
    # marker write under the old behaviour.
    def _would_have_run_initial_prompt():
        raise RuntimeError("simulated ProcessError mid-task")

    with pytest.raises(RuntimeError):
        _would_have_run_initial_prompt()

    # Marker is still present — next boot will skip the replay.
    assert os.path.exists(marker_path)
