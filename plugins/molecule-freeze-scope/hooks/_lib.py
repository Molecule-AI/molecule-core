"""Common helpers for Claude Code hooks. Imported by the .py hook scripts.

Hooks receive JSON on stdin per the Claude Code hook spec, and may emit
JSON on stdout or exit with code 2 to block. This module wraps both.
"""
import json
import sys


def read_input() -> dict:
    """Parse stdin JSON. Empty input → empty dict."""
    raw = sys.stdin.read().strip()
    if not raw:
        return {}
    try:
        return json.loads(raw)
    except json.JSONDecodeError:
        return {}


def emit(payload: dict) -> None:
    """Print JSON payload to stdout for the harness to interpret."""
    print(json.dumps(payload))


def deny_pretooluse(reason: str) -> None:
    """Emit a PreToolUse denial with reason and exit 0."""
    emit({
        "hookSpecificOutput": {
            "hookEventName": "PreToolUse",
            "permissionDecision": "deny",
            "permissionDecisionReason": reason,
        }
    })
    sys.exit(0)


def add_context(text: str) -> None:
    """Emit additionalContext for SessionStart / UserPromptSubmit hooks."""
    if text and text.strip():
        emit({"additionalContext": text})


def warn_to_stderr(msg: str) -> None:
    """Non-blocking warning visible to the next agent turn via stderr."""
    print(msg, file=sys.stderr)
