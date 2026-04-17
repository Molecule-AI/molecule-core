#!/usr/bin/env python3
"""Deduplicate hook entries in .claude/settings.json across all workspace containers.

Root cause: molecule_runtime's _deep_merge_hooks() uses unconditional list.extend()
when merging plugin settings-fragment.json files. On every plugin install/reinstall
each hook handler is appended again, producing 3-4x duplicates that cause every
hook to fire 3-4x per event.

This script fixes the live settings.json in every running workspace container via
the shared /proc/<PID>/root filesystem (no docker CLI required), then validates the
output is clean JSON. Safe to re-run — idempotent (already-clean files are skipped).

Upstream fix needed: molecule_runtime.plugins_registry.builtins._deep_merge_hooks()
should deduplicate by (matcher, frozenset(commands)) before writing. Tracked in
molecule-core issue (filed separately).

Usage:
    python3 scripts/dedup_settings_hooks.py [--dry-run]
"""

from __future__ import annotations

import glob
import json
import sys

DRY_RUN = "--dry-run" in sys.argv


def dedup_settings(data: dict) -> tuple[dict, dict[str, tuple[int, int]]]:
    """Return (deduped_data, stats) where stats[event] = (before_count, after_count)."""
    if "hooks" not in data:
        return data, {}
    new_hooks: dict = {}
    stats: dict[str, tuple[int, int]] = {}
    for event, handlers in data["hooks"].items():
        seen: set = set()
        deduped: list = []
        for handler in handlers:
            matcher = handler.get("matcher", "")
            commands = frozenset(h.get("command", "") for h in handler.get("hooks", []))
            key = (matcher, commands)
            if key not in seen:
                seen.add(key)
                deduped.append(handler)
        stats[event] = (len(handlers), len(deduped))
        new_hooks[event] = deduped
    return {**data, "hooks": new_hooks}, stats


def main() -> None:
    pattern = "/proc/*/root/configs/.claude/settings.json"
    paths = sorted(glob.glob(pattern))

    fixed: list[tuple[str, dict]] = []
    already_clean: list[str] = []
    errors: list[tuple[str, str]] = []

    for path in paths:
        try:
            with open(path) as f:
                data = json.load(f)
            deduped, stats = dedup_settings(data)
            changed = any(before != after for before, after in stats.values())
            if changed:
                if not DRY_RUN:
                    with open(path, "w") as f:
                        json.dump(deduped, f, indent=2)
                        f.write("\n")
                fixed.append((path, stats))
            else:
                already_clean.append(path)
        except PermissionError as e:
            errors.append((path, f"PermissionError: {e}"))
        except json.JSONDecodeError as e:
            errors.append((path, f"JSONDecodeError: {e}"))
        except Exception as e:
            errors.append((path, str(e)))

    mode = "[DRY RUN] " if DRY_RUN else ""
    print(f"{mode}Fixed: {len(fixed)}")
    for path, stats in fixed:
        pid = path.split("/")[2]
        summary = ", ".join(f"{ev}: {b}→{a}" for ev, (b, a) in stats.items() if b != a)
        print(f"  PID {pid}: {summary}")
    print(f"{mode}Already clean: {len(already_clean)}")
    if errors:
        print(f"Errors: {len(errors)}")
        for path, err in errors:
            print(f"  {path}: {err}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
