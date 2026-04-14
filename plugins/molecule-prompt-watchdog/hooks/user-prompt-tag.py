#!/usr/bin/env python3
"""UserPromptSubmit — inject context warnings for destructive-keyword prompts."""
import os
import sys
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from _lib import read_input, add_context, warn_to_stderr  # noqa

PATTERNS = [
    (
        ["force push", "force-push", "git push -f", "--force"],
        "Mention of force-push detected. Confirm scope (which branch? to main? careful-mode REFUSES force to main).",
    ),
    (
        ["delete all", "drop all", "wipe all", "remove all", "clear all"],
        "'all'-scoped destructive operation detected. Re-confirm exact target set (which workspaces / which rows / which files) before tooling.",
    ),
    (
        ["drop table", "truncate", "delete from", "drop database"],
        "Direct SQL DDL/DML detected. Use a migration via goose or a parameterized query through platform handlers — not raw psql against prod.",
    ),
    (
        ["merge directly", "push to main", "commit to main", "directly to main"],
        "Mention of working on main detected. Standing rule: never push to main. Use a branch + PR.",
    ),
]

CLOSE_BULK = ["close all", "close every"]
CLOSE_OBJ = ["pr", "issue", "workspace"]


def main() -> None:
    data = read_input()
    prompt = data.get("prompt", "").lower()
    if not prompt:
        return

    warnings = []
    for needles, msg in PATTERNS:
        if any(n in prompt for n in needles):
            warnings.append(f"• {msg}")

    if any(b in prompt for b in CLOSE_BULK) and any(o in prompt for o in CLOSE_OBJ):
        warnings.append("• Bulk close requested. List the targets first; do NOT loop a close command.")

    if warnings:
        add_context(
            "## ⚠ Prompt-watchdog warnings\n\n"
            + "\n".join(warnings)
            + "\n\ncareful-mode applies — re-confirm scope before any destructive tool call."
        )


if __name__ == "__main__":
    try:
        main()
    except Exception as e:
        warn_to_stderr(f"[prompt-tag hook error] {e}")
        sys.exit(0)
