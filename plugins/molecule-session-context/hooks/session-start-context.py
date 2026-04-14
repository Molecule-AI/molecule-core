#!/usr/bin/env python3
"""SessionStart hook — auto-load recent cron-learnings, freeze status,
and a one-line repo snapshot into Claude's context.
"""
import os
import subprocess
import sys
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from _lib import add_context, warn_to_stderr  # noqa

REPO = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
LEARNINGS = os.path.expanduser(
    "~/.claude/projects/-Users-hongming-Documents-GitHub-molecule-monorepo/memory/cron-learnings.jsonl"
)
FREEZE = os.path.join(REPO, ".claude", "freeze")


def tail(path: str, n: int) -> str:
    if not os.path.isfile(path):
        return ""
    try:
        with open(path) as f:
            lines = f.readlines()
        return "".join(lines[-n:]).rstrip()
    except Exception:
        return ""


def gh_count(args: list) -> str:
    try:
        out = subprocess.run(
            ["gh"] + args + ["--json", "number"],
            capture_output=True, text=True, timeout=4,
        )
        if out.returncode != 0:
            return "?"
        import json
        return str(len(json.loads(out.stdout or "[]")))
    except Exception:
        return "?"


def main() -> None:
    parts = []

    learnings = tail(LEARNINGS, 20)
    if learnings:
        parts.append(f"## Recent cron learnings (last 20)\n{learnings}")

    if os.path.isfile(FREEZE):
        try:
            with open(FREEZE) as f:
                frozen = f.readline().strip()
            parts.append(f"## ⚠ FREEZE ACTIVE\nEdits restricted to: {frozen}\nRemove .claude/freeze to unlock.")
        except Exception:
            pass

    pr = gh_count(["pr", "list", "--repo", "Molecule-AI/molecule-monorepo", "--state", "open"])
    iss = gh_count(["issue", "list", "--repo", "Molecule-AI/molecule-monorepo", "--state", "open"])
    parts.append(f"## Repo state\nOpen PRs: {pr} · Open issues: {iss}")

    if parts:
        add_context("\n\n".join(parts))


if __name__ == "__main__":
    try:
        main()
    except Exception as e:
        warn_to_stderr(f"[session-start hook error] {e}")
        sys.exit(0)
