"""CLI: ``python -m molecule_plugin validate <kind> <path>...``.

Kinds:

* ``plugin``        — a plugin directory (plugin.yaml + skills/, adapters/…)
* ``workspace``     — a workspace-configs-template directory (config.yaml)
* ``org``           — an org-template directory (org.yaml)
* ``channel``       — a channel config YAML/JSON file (standalone or list)

Exit 0 on valid, 1 when errors found. Intended for CI and local author
workflows before publishing. ``validate <path>`` (kind omitted) is kept as
a back-compat shortcut for plugin validation.
"""

from __future__ import annotations

import argparse
import sys
from pathlib import Path

from .channel import validate_channel_file
from .manifest import validate_plugin
from .org import validate_org_template
from .workspace import validate_workspace_template


def _validate_plugin(paths: list[str], quiet: bool) -> int:
    total = 0
    for raw in paths:
        path = Path(raw)
        if not path.exists():
            print(f"✗ {path}: does not exist", file=sys.stderr)
            total += 1
            continue
        if not path.is_dir():
            print(f"✗ {path}: not a directory", file=sys.stderr)
            total += 1
            continue

        results = validate_plugin(path)
        if not results:
            if not quiet:
                print(f"✓ {path}: valid (plugin.yaml + all skills pass agentskills.io spec)")
            continue
        for source, errors in results.items():
            total += len(errors)
            for err in errors:
                print(f"✗ {path}/{source}: {err}", file=sys.stderr)
    return 0 if total == 0 else 1


def _validate_dir(
    kind: str,
    paths: list[str],
    validator,
    quiet: bool,
) -> int:
    total = 0
    for raw in paths:
        path = Path(raw)
        if not path.exists():
            print(f"✗ {path}: does not exist", file=sys.stderr)
            total += 1
            continue
        if not path.is_dir():
            print(f"✗ {path}: not a directory", file=sys.stderr)
            total += 1
            continue
        errors = validator(path)
        if not errors:
            if not quiet:
                print(f"✓ {path}: valid {kind}")
            continue
        total += len(errors)
        for err in errors:
            print(f"✗ {err.file}: {err.message}", file=sys.stderr)
    return 0 if total == 0 else 1


def _validate_channel(paths: list[str], quiet: bool) -> int:
    total = 0
    for raw in paths:
        path = Path(raw)
        if not path.exists():
            print(f"✗ {path}: does not exist", file=sys.stderr)
            total += 1
            continue
        errors = validate_channel_file(path)
        if not errors:
            if not quiet:
                print(f"✓ {path}: valid channel config")
            continue
        total += len(errors)
        for err in errors:
            print(f"✗ {err.file}: {err.message}", file=sys.stderr)
    return 0 if total == 0 else 1


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(prog="molecule_plugin")
    sub = parser.add_subparsers(dest="cmd", required=True)

    v = sub.add_parser("validate", help="Validate Molecule AI artifacts")
    v.add_argument("args", nargs="+", help="[kind] paths... — kind in {plugin,workspace,org,channel}; defaults to plugin")
    v.add_argument("--quiet", "-q", action="store_true")

    args = parser.parse_args(argv)
    kinds = {"plugin", "workspace", "org", "channel"}
    if args.args and args.args[0] in kinds:
        args.kind = args.args[0]
        args.paths = args.args[1:]
    else:
        args.kind = "plugin"
        args.paths = args.args
    if not args.paths:
        parser.error("at least one path is required")

    if args.kind == "plugin":
        return _validate_plugin(args.paths, args.quiet)
    if args.kind == "workspace":
        return _validate_dir("workspace template", args.paths, validate_workspace_template, args.quiet)
    if args.kind == "org":
        return _validate_dir("org template", args.paths, validate_org_template, args.quiet)
    if args.kind == "channel":
        return _validate_channel(args.paths, args.quiet)
    return 2  # pragma: no cover


if __name__ == "__main__":  # pragma: no cover
    raise SystemExit(main())
