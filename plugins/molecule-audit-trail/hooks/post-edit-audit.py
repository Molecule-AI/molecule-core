#!/usr/bin/env python3
"""PostToolUse:Edit/Write — append one-line audit record to .claude/audit.jsonl."""
import datetime as dt
import json
import os
import sys
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from _lib import read_input, warn_to_stderr  # noqa

REPO = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
AUDIT = os.path.join(REPO, ".claude", "audit.jsonl")


def main() -> None:
    data = read_input()
    target = data.get("tool_input", {}).get("file_path") or data.get("tool_input", {}).get("notebook_path") or ""
    if target.startswith(REPO + "/"):
        target = target[len(REPO) + 1:]

    record = {
        "ts": dt.datetime.now(dt.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        "tool": data.get("tool_name", "unknown"),
        "file": target,
        "ok": data.get("tool_response", {}).get("success", True),
    }
    try:
        with open(AUDIT, "a") as f:
            f.write(json.dumps(record) + "\n")
    except Exception:
        pass  # never block tool execution on audit-write failure


if __name__ == "__main__":
    try:
        main()
    except Exception as e:
        warn_to_stderr(f"[audit hook error] {e}")
        sys.exit(0)
