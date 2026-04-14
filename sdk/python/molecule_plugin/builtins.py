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

import os
import shutil
import subprocess
from pathlib import Path

from .protocol import SKILLS_SUBDIR, InstallContext, InstallResult

# ---------------------------------------------------------------------------
# Security: credential scrubbing for plugin setup.sh subprocesses
# ---------------------------------------------------------------------------
# setup.sh runs arbitrary bash inside the workspace container. It must not
# inherit credentials from the parent process environment — an attacker who
# controls setup.sh content (via a malicious plugin) would otherwise be able
# to exfiltrate ANTHROPIC_API_KEY, WORKSPACE_AUTH_TOKEN, GITHUB_TOKEN, etc.
#
# Strategy: denylist by suffix/name rather than allowlist, so legitimate
# npm/pip env vars (NPM_CONFIG_*, PIP_*) keep working without maintenance.
# Mirrors workspace-template/plugins_registry/builtins.py — must stay in sync
# (drift guard: tests/test_plugins_builtins_drift.py).

_SENSITIVE_SUFFIXES = (
    "_KEY", "_TOKEN", "_SECRET", "_PASSWORD",
    "_CREDENTIAL", "_API_KEY",
)
_SENSITIVE_NAMES = frozenset({
    "DATABASE_URL", "REDIS_URL", "AWARENESS_URL",
})


def _scrub_env_for_setup(extra: dict | None = None) -> dict:
    """Return a copy of os.environ with credentials stripped.

    setup.sh should never see API keys, auth tokens, or DB credentials that
    are present in the parent-process environment.
    """
    safe: dict = {}
    for k, v in os.environ.items():
        ku = k.upper()
        if ku in _SENSITIVE_NAMES:
            continue
        if any(ku.endswith(suffix) for suffix in _SENSITIVE_SUFFIXES):
            continue
        safe[k] = v
    if extra:
        safe.update(extra)
    return safe

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
                    env=_scrub_env_for_setup({"CONFIGS_DIR": str(ctx.configs_dir)}),
                )
                if proc.returncode == 0:
                    ctx.logger.info("%s: setup.sh completed successfully", self.plugin_name)
                else:
                    result.warnings.append(f"setup.sh exited {proc.returncode}: {proc.stderr[:200]}")
                    ctx.logger.warning("%s: setup.sh failed: %s", self.plugin_name, proc.stderr[:200])
            except subprocess.TimeoutExpired:
                result.warnings.append("setup.sh timed out (120s)")
                ctx.logger.warning("%s: setup.sh timed out", self.plugin_name)

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


