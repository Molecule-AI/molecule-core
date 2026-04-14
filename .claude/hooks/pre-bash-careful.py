#!/usr/bin/env python3
"""PreToolUse:Bash — enforce careful-mode patterns on shell commands."""
import sys
import os
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from _lib import read_input, deny_pretooluse, warn_to_stderr  # noqa


def main() -> None:
    data = read_input()
    cmd = data.get("tool_input", {}).get("command", "")
    if not cmd:
        return

    # REFUSE list — hard stops
    refuse_patterns = [
        ("git push --force", "main", "git push --force to main is REFUSED. Use --force-with-lease on a feature branch only."),
        ("git push -f", "main", "git push -f to main is REFUSED."),
        ("git push --force", "master", "git push --force to master is REFUSED."),
        ("git push -f", "master", "git push -f to master is REFUSED."),
    ]
    for needle1, needle2, msg in refuse_patterns:
        if needle1 in cmd and needle2 in cmd:
            deny_pretooluse(f"careful-mode: {msg}")

    if "git reset --hard" in cmd and ("origin/main" in cmd or " main" in cmd or "/main" in cmd):
        deny_pretooluse("careful-mode: git reset --hard against main is REFUSED. Stash, branch, then reset.")

    # SQL DDL/DML against prod-like names
    sql_destructive = ["DROP TABLE", "DROP DATABASE", "TRUNCATE TABLE"]
    for tok in sql_destructive:
        if tok in cmd:
            # Allow against test/sandbox patterns
            allow_substrings = ["_test", "sandbox", "/tmp/", "_dev", "test_"]
            if not any(a in cmd for a in allow_substrings):
                deny_pretooluse(f"careful-mode: '{tok}' against production-like schema is REFUSED. Use a migration with explicit review.")

    # rm -rf at scary paths
    if "rm -rf" in cmd:
        scary = [" /", " ~", " $HOME", "/.git ", "/.git/"]
        scratch_ok = ["/tmp/", "node_modules", "dist", ".next", "__pycache__", ".pytest_cache", "coverage"]
        if any(s in cmd for s in scary) and not any(s in cmd for s in scratch_ok):
            # Check for migrations dir specifically
            if "migrations" in cmd:
                deny_pretooluse("careful-mode: rm -rf inside a migrations dir is REFUSED.")
            deny_pretooluse(f"careful-mode: rm -rf at filesystem root, HOME, or .git is REFUSED. Command: {cmd[:200]}")
        if "/.git" in cmd:
            deny_pretooluse("careful-mode: rm -rf .git is REFUSED. Re-clone if you need a fresh repo.")

    # WARN list — log but allow
    if "git push --force-with-lease" in cmd:
        warn_to_stderr("[careful-mode WARN] force-with-lease: safer than --force but still rewrites remote history.")
    if "gh pr close" in cmd or "gh issue close" in cmd:
        warn_to_stderr("[careful-mode WARN] closing a PR/issue is irreversible from this bot's standpoint. Confirm intent.")


if __name__ == "__main__":
    try:
        main()
    except Exception as e:  # never break tool execution due to hook bug
        warn_to_stderr(f"[careful-mode hook error] {e}")
        sys.exit(0)
