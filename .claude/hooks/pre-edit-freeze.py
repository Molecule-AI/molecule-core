#!/usr/bin/env python3
"""PreToolUse:Edit/Write — enforce /freeze scope from .claude/freeze."""
import os
import sys
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from _lib import read_input, deny_pretooluse, warn_to_stderr  # noqa

REPO = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
FREEZE = os.path.join(REPO, ".claude", "freeze")


def main() -> None:
    if not os.path.isfile(FREEZE):
        return
    with open(FREEZE) as f:
        allowed = f.readline().strip()
    if not allowed:
        return

    data = read_input()
    target = data.get("tool_input", {}).get("file_path") or data.get("tool_input", {}).get("notebook_path") or ""
    if not target:
        return

    # Always allow .claude/ writes (so unfreeze still works)
    if "/.claude/" in target or target.endswith("/.claude") or "/.claude" in target:
        return

    if allowed in target:
        return

    deny_pretooluse(
        f"freeze: edit to {target} refused — scope locked to '{allowed}'. "
        f"Remove .claude/freeze to unlock."
    )


if __name__ == "__main__":
    try:
        main()
    except Exception as e:
        warn_to_stderr(f"[freeze hook error] {e}")
        sys.exit(0)
