"""Per-process runtime-wedge state.

Adapter executors that hit a non-recoverable wedge (e.g. claude-agent-sdk's
`Control request timeout: initialize` corrupting the client process's
internal state) call mark_wedged(reason). The heartbeat task reads
is_wedged() / wedge_reason() and forwards them in the heartbeat payload's
runtime_state field — the platform then flips workspace status to
`degraded` so the canvas surfaces a Restart hint instead of leaving the
user staring at a green dot while every chat hangs.

Module scope (not instance scope) is deliberate: the wedge is a property
of the Python process, not any particular executor. With one executor
per workspace process today this is the simplest lock-free
read+write fit. A future per-org multi-executor design could move this
to a shared registry.

This module lives in molecule-runtime (NOT in any adapter / template
repo) because:

  1. workspace/heartbeat.py reads it on every heartbeat — cross-cutting
     concern, runtime owns it.
  2. Multiple adapter executors can mark themselves wedged with their
     own reason; the runtime aggregates one flag for the platform.
  3. Decoupling from claude_sdk_executor is the prerequisite for the
     universal-runtime refactor (molecule-core task #87) — without
     this extraction, claude_sdk_executor.py couldn't move to its
     template repo because heartbeat would lose access to the wedge
     state.

Public API: mark_wedged(reason), clear_wedge(), is_wedged(),
wedge_reason(). The reset_for_test() helper is for unit tests only.
"""
from __future__ import annotations

import logging

logger = logging.getLogger(__name__)


# Single-flag state. None = healthy; non-empty string = wedged with that
# human-readable reason. Surfaced verbatim as the canvas's degraded-card
# banner text via heartbeat.sample_error.
_wedged_reason: str | None = None


def is_wedged() -> bool:
    """True if some adapter executor in this process has marked itself
    wedged. Sticky until the same executor calls clear_wedge() on
    observed recovery (or the process restarts)."""
    return _wedged_reason is not None


def wedge_reason() -> str:
    """Human-readable description of the wedge cause, or empty string
    when not wedged. Surfaced to the canvas via heartbeat sample_error."""
    return _wedged_reason or ""


def mark_wedged(reason: str) -> None:
    """Flag the runtime as wedged. Only the FIRST call wins so a
    subsequent identical-class wedge can't overwrite a more specific
    initial reason — the operator-visible banner stays stable.

    Adapters call this from their executor's exception path when the
    SDK has hit a non-recoverable error class. Safe to call multiple
    times; the no-op when already wedged is intentional.
    """
    global _wedged_reason
    if _wedged_reason is None:
        _wedged_reason = reason
        logger.error(
            "runtime wedge detected: %s — workspace will report degraded until cleared",
            reason,
        )


def clear_wedge() -> None:
    """Auto-recovery: adapter calls this after an observed successful
    operation. The original wedge could be transient (single network
    blip during the SDK's first-message handshake), and a sticky-only
    flag would lock the workspace into degraded forever even after the
    SDK started working again. Clearing on observed success means the
    next heartbeat after a working query reports runtime_state empty
    and the platform flips status back to online.

    No-op when not wedged (the common case)."""
    global _wedged_reason
    if _wedged_reason is not None:
        logger.info("runtime wedge cleared after successful operation — workspace will recover to online on next heartbeat")
        _wedged_reason = None


def reset_for_test() -> None:
    """Test-only escape hatch. Production code clears the wedge via
    clear_wedge() on observed success; this helper is for unit tests
    that need to reset between cases without going through the full
    SDK round-trip."""
    global _wedged_reason
    _wedged_reason = None
