"""Built-in sub-type adapters for the SDK.

One class per agent shape. Currently ships :class:`AgentskillsAdaptor`
(the `agentskills.io <https://agentskills.io>`_-format default); more
will be added as new shapes emerge in the ecosystem
(``MCPServerAdaptor``, ``DeepAgentsSubagentAdaptor``, ``RAGPipelineAdaptor``,
etc.).

SDK authors pick a sub-type by import:

.. code-block:: python

    # adapters/claude_code.py
    from molecule_plugin import AgentskillsAdaptor as Adaptor

Plugins whose shape doesn't match any built-in ship a custom adapter
class in Python — unlimited expressiveness, no framework constraint.
"""

from __future__ import annotations

import json
import os
import shutil
import subprocess
from pathlib import Path

from .protocol import SKILLS_SUBDIR, InstallContext, InstallResult

# Files at the plugin root that are never treated as prompt fragments.
SKIP_ROOT_MD = frozenset({"readme.md", "changelog.md", "license.md", "contributing.md"})


class AgentskillsAdaptor:
    """Sub-type adaptor for `agentskills.io <https://agentskills.io>`_-format skills.

    The default adapter for the "skills + rules" shape — installs
    ``skills/<name>/SKILL.md`` into ``/configs/skills/`` (where native
    agentskills runtimes like Claude Code activate them automatically)
    and appends Molecule AI-level ``rules/*.md`` + root prompt fragments to
    the runtime memory file.

    Matches the behaviour of the workspace runtime's
    ``plugins_registry.builtins.AgentskillsAdaptor``. Kept as a separate
    copy here so SDK users can unit-test their plugins without installing
    the full workspace runtime.
    """

    def __init__(self, plugin_name: str, runtime: str) -> None:
        self.plugin_name = plugin_name
        self.runtime = runtime

    async def install(self, ctx: InstallContext) -> InstallResult:
        result = InstallResult(plugin_name=self.plugin_name, runtime=self.runtime, source="plugin")

        rules_dir = ctx.plugin_root / "rules"
        blocks: list[str] = []
        if rules_dir.is_dir():
            for p in sorted(rules_dir.iterdir()):
                if p.is_file() and p.suffix == ".md":
                    content = p.read_text().strip()
                    if content:
                        blocks.append(f"# Plugin: {self.plugin_name} / rule: {p.name}\n\n{content}")

        if ctx.plugin_root.is_dir():
            for p in sorted(ctx.plugin_root.iterdir()):
                if p.is_file() and p.suffix == ".md" and p.name.lower() not in SKIP_ROOT_MD:
                    content = p.read_text().strip()
                    if content:
                        blocks.append(f"# Plugin: {self.plugin_name} / fragment: {p.name}\n\n{content}")

        if blocks:
            ctx.append_to_memory(ctx.memory_filename, "\n\n".join(blocks))

        src_skills = ctx.plugin_root / "skills"
        if src_skills.is_dir():
            dst_root = ctx.configs_dir / SKILLS_SUBDIR
            dst_root.mkdir(parents=True, exist_ok=True)
            for entry in sorted(src_skills.iterdir()):
                if not entry.is_dir():
                    continue
                dst = dst_root / entry.name
                if dst.exists():
                    continue
                shutil.copytree(entry, dst)
                for p in dst.rglob("*"):
                    if p.is_file():
                        result.files_written.append(str(p.relative_to(ctx.configs_dir)))

        # 4. Setup script — run setup.sh if present (npm/pip dependencies).
        # Mirrors workspace-template/plugins_registry/builtins.py — must stay
        # in sync (drift guard: tests/test_plugins_builtins_drift.py).
        setup_script = ctx.plugin_root / "setup.sh"
        if setup_script.is_file():
            ctx.logger.info("%s: running setup.sh", self.plugin_name)
            try:
                proc = subprocess.run(
                    ["bash", str(setup_script)],
                    capture_output=True, text=True, timeout=120,
                    cwd=str(ctx.plugin_root),
                    env={**os.environ, "CONFIGS_DIR": str(ctx.configs_dir)},
                )
                if proc.returncode == 0:
                    ctx.logger.info("%s: setup.sh completed successfully", self.plugin_name)
                else:
                    result.warnings.append(f"setup.sh exited {proc.returncode}: {proc.stderr[:200]}")
                    ctx.logger.warning("%s: setup.sh failed: %s", self.plugin_name, proc.stderr[:200])
            except subprocess.TimeoutExpired:
                result.warnings.append("setup.sh timed out (120s)")
                ctx.logger.warning("%s: setup.sh timed out", self.plugin_name)

        # Claude Code layer — hooks/, commands/, settings-fragment.json.
        # Mirrors workspace-template/plugins_registry/builtins.py — drift
        # guarded by tests/test_plugins_builtins_drift.py.
        _install_claude_layer(ctx, result, self.plugin_name)

        return result

    async def uninstall(self, ctx: InstallContext) -> None:
        src_skills = ctx.plugin_root / "skills"
        if src_skills.is_dir():
            for entry in src_skills.iterdir():
                dst = ctx.configs_dir / SKILLS_SUBDIR / entry.name
                if dst.exists() and dst.is_dir():
                    shutil.rmtree(dst)

        memory_path = ctx.configs_dir / ctx.memory_filename
        if memory_path.exists():
            prefix = f"# Plugin: {self.plugin_name} / "
            kept = [ln for ln in memory_path.read_text().splitlines(keepends=True) if not ln.startswith(prefix)]
            memory_path.write_text("".join(kept))




# ----------------------------------------------------------------------
# Claude Code layer — mirrors workspace-template/plugins_registry/builtins.py.
# Drift-guarded by workspace-template/tests/test_plugins_builtins_drift.py.
# ----------------------------------------------------------------------

def _install_claude_layer(ctx: InstallContext, result: InstallResult, plugin_name: str) -> None:
    claude_dir = ctx.configs_dir / ".claude"
    claude_dir.mkdir(parents=True, exist_ok=True)
    _copy_dir_files(ctx.plugin_root / "hooks", claude_dir / "hooks", result, executable_suffix=".sh")
    _copy_dir_files(ctx.plugin_root / "commands", claude_dir / "commands", result, only_suffix=".md")
    _merge_settings_fragment(ctx, claude_dir, result, plugin_name)


def _copy_dir_files(src: Path, dst: Path, result: InstallResult,
                    executable_suffix: str | None = None,
                    only_suffix: str | None = None) -> None:
    if not src.is_dir():
        return
    dst.mkdir(parents=True, exist_ok=True)
    for f in src.iterdir():
        if not f.is_file():
            continue
        if only_suffix and f.suffix != only_suffix:
            if not (executable_suffix and f.suffix == ".py"):
                continue
        target = dst / f.name
        shutil.copy2(f, target)
        if executable_suffix and f.suffix == executable_suffix:
            target.chmod(0o755)
        result.files_written.append(str(target.relative_to(target.parents[2])))


def _merge_settings_fragment(ctx: InstallContext, claude_dir: Path,
                              result: InstallResult, plugin_name: str) -> None:
    fragment_path = ctx.plugin_root / "settings-fragment.json"
    if not fragment_path.is_file():
        return
    try:
        fragment = json.loads(fragment_path.read_text())
    except Exception as e:
        result.warnings.append(f"settings-fragment.json invalid: {e}")
        return
    settings_path = claude_dir / "settings.json"
    if settings_path.is_file():
        try:
            existing = json.loads(settings_path.read_text())
        except Exception:
            existing = {}
    else:
        existing = {}
    rewritten = _rewrite_hook_paths(fragment, claude_dir)
    merged = _deep_merge_hooks(existing, rewritten)
    settings_path.write_text(json.dumps(merged, indent=2) + "\n")
    result.files_written.append(str(settings_path.relative_to(ctx.configs_dir)))
    ctx.logger.info("%s: merged hook config into %s", plugin_name, settings_path)


def _rewrite_hook_paths(fragment: dict, claude_dir: Path) -> dict:
    out = json.loads(json.dumps(fragment))
    for handlers in out.get("hooks", {}).values():
        for handler in handlers:
            for h in handler.get("hooks", []):
                h["command"] = h.get("command", "").replace("${CLAUDE_DIR}", str(claude_dir))
    return out


def _deep_merge_hooks(existing: dict, fragment: dict) -> dict:
    out = dict(existing)
    out.setdefault("hooks", {})
    for event, handlers in fragment.get("hooks", {}).items():
        out["hooks"].setdefault(event, [])
        out["hooks"][event].extend(handlers)
    for key, val in fragment.items():
        if key == "hooks":
            continue
        out.setdefault(key, val)
    return out
