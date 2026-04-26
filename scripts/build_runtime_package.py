#!/usr/bin/env python3
"""Build the molecule-ai-workspace-runtime PyPI package from monorepo workspace/.

Monorepo workspace/ is the single source-of-truth for runtime code. The PyPI
package is a publish-time mirror produced by this script, NOT a parallel
editable copy. Anyone editing the runtime should edit workspace/, never the
sibling molecule-ai-workspace-runtime repo.

What this does
--------------
1. Copies workspace/ source into build/molecule_runtime/ (note the rename:
   bare modules become a real Python package).
2. Rewrites top-level imports so e.g. `from a2a_client import X` becomes
   `from molecule_runtime.a2a_client import X`. The rewrite is regex-based
   on a closed allowlist of modules — third-party imports like `from a2a.X`
   (the a2a-sdk package) are left alone because the regex is anchored on
   exact module names.
3. Writes a pyproject.toml with the requested version + the README + the
   py.typed marker.
4. Leaves the build dir ready for `python -m build` to produce a wheel/sdist.

Usage
-----
  scripts/build_runtime_package.py --version 0.1.6 --out /tmp/runtime-build
  cd /tmp/runtime-build && python -m build
  python -m twine upload dist/*

The publish workflow (.github/workflows/publish-runtime.yml) drives this
on every `runtime-v*` tag push.
"""

from __future__ import annotations

import argparse
import re
import shutil
import sys
from pathlib import Path

# Top-level Python modules in workspace/ that become molecule_runtime.X.
# Anything imported as `from <name> import` or `import <name>` (where <name>
# matches one of these) gets rewritten to use the package prefix.
#
# Closed list (not "every .py we copy") because a typo in workspace/ would
# otherwise leak into a wrong rewrite. Update this when adding a new
# top-level module to workspace/.
TOP_LEVEL_MODULES = {
    "a2a_cli",
    "a2a_client",
    "a2a_executor",
    "a2a_mcp_server",
    "a2a_tools",
    "adapter_base",
    "agent",
    "agents_md",
    "claude_sdk_executor",
    "cli_executor",
    "config",
    "consolidation",
    "coordinator",
    "events",
    "executor_helpers",
    "heartbeat",
    "hermes_executor",
    "initial_prompt",
    "main",
    "molecule_ai_status",
    "platform_auth",
    "plugins",
    "preflight",
    "prompt",
    "shared_runtime",
}

# Subdirectory packages — these are already real packages (they have or will
# have __init__.py) so the rewrite is `from <pkg>` → `from molecule_runtime.<pkg>`.
SUBPACKAGES = {
    "adapters",
    "builtin_tools",
    "plugins_registry",
    "policies",
    "skill_loader",
}

# Files in workspace/ NOT included in the published package. These are
# build artifacts, dev scripts, or monorepo-only scaffolding.
EXCLUDE_FILES = {
    "Dockerfile",
    "build-all.sh",
    "rebuild-runtime-images.sh",
    "entrypoint.sh",
    "pytest.ini",
    "requirements.txt",
    # Note: adapter_base.py, agents_md.py, hermes_executor.py, shared_runtime.py
    # are kept (referenced by adapters/__init__.py and other modules); they get
    # their imports rewritten via TOP_LEVEL_MODULES. Excluding them broke the
    # smoke-test install with `ModuleNotFoundError: adapter_base`.
}

EXCLUDE_DIRS = {
    "__pycache__",
    "tests",
    "lib",
    "molecule_audit",
    "scripts",
}


def build_import_rewriter() -> re.Pattern:
    """Compile a single regex matching all import statements that need
    rewriting. The match groups capture the keyword + module name so the
    replacement preserves whitespace and trailing punctuation.

    Modules included: TOP_LEVEL_MODULES ∪ SUBPACKAGES.

    The negative-lookahead on `\\.` in the suffix prevents matching
    `from a2a.server.X import Y` against bare `a2a` (which isn't in our
    set, but the principle matters for any future short module name that
    happens to be a prefix of a real package name).
    """
    names = sorted(TOP_LEVEL_MODULES | SUBPACKAGES)
    alt = "|".join(re.escape(n) for n in names)
    # Matches:
    #   from <name>(\.|\s|import)
    #   import <name>(\s|$|,)
    # And captures the keyword + name so we can re-emit with prefix.
    pattern = (
        r"(?m)^(?P<indent>\s*)"          # leading whitespace (preserved)
        r"(?P<kw>from|import)\s+"        # 'from' or 'import'
        r"(?P<mod>" + alt + r")"          # the module name
        r"(?P<rest>[\s.,]|$)"            # what follows: '.subpath', ' import …', ',', whitespace, EOL
    )
    return re.compile(pattern)


def rewrite_imports(text: str, regex: re.Pattern) -> str:
    """Replace bare imports with package-prefixed ones.

    `import X`           → `import molecule_runtime.X as X`  (preserve binding)
    `from X import Y`    → `from molecule_runtime.X import Y`
    `from X.sub import Y` → `from molecule_runtime.X.sub import Y`
    """
    def repl(m: re.Match) -> str:
        indent, kw, mod, rest = m.group("indent"), m.group("kw"), m.group("mod"), m.group("rest")
        if kw == "from":
            # `from X` or `from X.sub` — always safe to prefix.
            return f"{indent}from molecule_runtime.{mod}{rest}"
        # `import X` — preserve the binding name `X` (callers do `X.foo`)
        # by aliasing. `import X.sub` is uncommon for our modules and would
        # need a different binding form, but isn't used in workspace/ today.
        if rest.startswith("."):
            # `import X.sub` — rewrite as `import molecule_runtime.X.sub` and
            # leave the trailing dot pattern intact for the rest of the line.
            return f"{indent}import molecule_runtime.{mod}{rest}"
        # Plain `import X` — alias preserves the local name.
        return f"{indent}import molecule_runtime.{mod} as {mod}{rest}"
    return regex.sub(repl, text)


def copy_tree_filtered(src: Path, dst: Path) -> list[Path]:
    """Copy src/ → dst/ skipping EXCLUDE_FILES + EXCLUDE_DIRS. Returns the
    list of .py files copied so the caller can run the import rewrite over
    them in one pass."""
    py_files: list[Path] = []
    if dst.exists():
        shutil.rmtree(dst)
    dst.mkdir(parents=True)
    for entry in src.iterdir():
        if entry.is_dir():
            if entry.name in EXCLUDE_DIRS:
                continue
            sub_py = copy_tree_filtered(entry, dst / entry.name)
            py_files.extend(sub_py)
        else:
            if entry.name in EXCLUDE_FILES:
                continue
            shutil.copy2(entry, dst / entry.name)
            if entry.suffix == ".py":
                py_files.append(dst / entry.name)
    return py_files


PYPROJECT_TEMPLATE = """\
[build-system]
requires = ["setuptools>=68.0", "wheel"]
build-backend = "setuptools.build_meta"

[project]
name = "molecule-ai-workspace-runtime"
version = "{version}"
description = "Molecule AI workspace runtime — shared infrastructure for all agent adapters"
requires-python = ">=3.11"
license = {{text = "BSL-1.1"}}
readme = "README.md"
dependencies = [
    "a2a-sdk[http-server]>=1.0.0,<2.0",
    "httpx>=0.27.0",
    "uvicorn>=0.30.0",
    "starlette>=0.38.0",
    "websockets>=12.0",
    "pyyaml>=6.0",
    "langchain-core>=0.3.0",
    "opentelemetry-api>=1.24.0",
    "opentelemetry-sdk>=1.24.0",
    "opentelemetry-exporter-otlp-proto-http>=1.24.0",
    "temporalio>=1.7.0",
]

[project.scripts]
molecule-runtime = "molecule_runtime.main:main_sync"

[tool.setuptools.packages.find]
where = ["."]
include = ["molecule_runtime*"]

[tool.setuptools.package-data]
"molecule_runtime" = ["py.typed"]
"""


README_TEMPLATE = """\
# molecule-ai-workspace-runtime

Shared workspace runtime for [Molecule AI](https://github.com/Molecule-AI/molecule-core)
agent adapters. Installed by every workspace template image
(`workspace-template-claude-code`, `-langgraph`, `-hermes`, etc.) to provide
A2A delegation, heartbeat, memory, plugin loading, and skill management.

This package is **published from the molecule-core monorepo `workspace/`
directory** by the `publish-runtime` GitHub Actions workflow on every
`runtime-v*` tag push. **Do not edit this package directly** — edit
`workspace/` in the monorepo.

See [`docs/workspace-runtime-package.md`](https://github.com/Molecule-AI/molecule-core/blob/main/docs/workspace-runtime-package.md)
for the publish flow and architecture.
"""


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--version", required=True, help="Package version, e.g. 0.1.6")
    parser.add_argument("--out", required=True, type=Path, help="Build output directory (will be wiped)")
    parser.add_argument("--source", type=Path, default=Path(__file__).resolve().parent.parent / "workspace",
                        help="Path to monorepo workspace/ directory (default: ../workspace from this script)")
    args = parser.parse_args()

    src = args.source.resolve()
    out = args.out.resolve()
    if not src.is_dir():
        print(f"error: source not a directory: {src}", file=sys.stderr)
        return 2

    pkg_dir = out / "molecule_runtime"
    print(f"[build] source: {src}")
    print(f"[build] output: {out}")
    print(f"[build] package: {pkg_dir}")

    if out.exists():
        shutil.rmtree(out)
    out.mkdir(parents=True)

    py_files = copy_tree_filtered(src, pkg_dir)
    print(f"[build] copied {len(py_files)} .py files")

    # Ensure top-level package marker exists. workspace/ doesn't have one
    # (it's not a package in monorepo), but the published artifact must.
    init = pkg_dir / "__init__.py"
    if not init.exists():
        init.write_text('"""Molecule AI workspace runtime."""\n')

    # Touch py.typed so type-checkers in adapter consumers see the package
    # as typed. Empty file is the convention.
    (pkg_dir / "py.typed").touch()

    # Rewrite imports in every .py file we copied + the new __init__.py.
    regex = build_import_rewriter()
    rewrites = 0
    for f in [*py_files, init]:
        original = f.read_text()
        rewritten = rewrite_imports(original, regex)
        if rewritten != original:
            f.write_text(rewritten)
            rewrites += 1
    print(f"[build] rewrote imports in {rewrites} files")

    # Emit pyproject.toml + README at build root.
    (out / "pyproject.toml").write_text(PYPROJECT_TEMPLATE.format(version=args.version))
    (out / "README.md").write_text(README_TEMPLATE)

    print(f"[build] done. To publish:")
    print(f"  cd {out}")
    print(f"  python -m build")
    print(f"  python -m twine upload dist/*")
    return 0


if __name__ == "__main__":
    sys.exit(main())
