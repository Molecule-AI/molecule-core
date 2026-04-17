#!/usr/bin/env python3
"""Verify settings.json hook deduplication across all workspace containers.

Exits 0 if all containers have clean (no-duplicate) hook lists.
Exits 1 if any container still has duplicate hook entries.

Usage:
    python3 scripts/verify_settings_hooks.py
"""

from __future__ import annotations

import glob
import json
import sys


def has_duplicates(data: dict) -> tuple[bool, dict[str, tuple[int, int]]]:
    stats: dict[str, tuple[int, int]] = {}
    duplicate_found = False
    for event, handlers in data.get("hooks", {}).items():
        seen: set = set()
        for handler in handlers:
            matcher = handler.get("matcher", "")
            commands = frozenset(h.get("command", "") for h in handler.get("hooks", []))
            key = (matcher, commands)
            if key in seen:
                duplicate_found = True
            seen.add(key)
        stats[event] = (len(handlers), len(seen))
    return duplicate_found, stats


def main() -> None:
    pattern = "/proc/*/root/configs/.claude/settings.json"
    paths = sorted(glob.glob(pattern))

    dirty: list[tuple[str, dict]] = []
    clean = 0
    errors: list[tuple[str, str]] = []

    for path in paths:
        try:
            with open(path) as f:
                data = json.load(f)
            dup, stats = has_duplicates(data)
            if dup:
                dirty.append((path, stats))
            else:
                clean += 1
        except Exception as e:
            errors.append((path, str(e)))

    print(f"Clean: {clean}  Dirty: {len(dirty)}  Errors: {len(errors)}")
    for path, stats in dirty:
        pid = path.split("/")[2]
        summary = ", ".join(f"{ev}: {total} total/{unique} unique" for ev, (total, unique) in stats.items())
        print(f"  DIRTY PID {pid}: {summary}")
    for path, err in errors:
        print(f"  ERROR {path}: {err}", file=sys.stderr)

    if dirty or errors:
        sys.exit(1)


if __name__ == "__main__":
    main()
