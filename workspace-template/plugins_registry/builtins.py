"""Built-in plugin adaptors — one per agent shape.

The adapter layer is our extensibility surface. Each agent "shape" (form
of installable capability) gets its own named sub-type adapter. A plugin
picks which sub-type to use by importing it as ``Adaptor`` in its
per-runtime file:

.. code-block:: python

    # plugins/<name>/adapters/claude_code.py
    from plugins_registry.builtins import AgentskillsAdaptor as Adaptor

Shape taxonomy (one class per shape; add more as the ecosystem evolves):

* :class:`AgentskillsAdaptor` — skills in the `agentskills.io
  <https://agentskills.io>`_ format (``SKILL.md`` + ``scripts/`` +
  ``references/`` + ``assets/``), plus Molecule AI's optional ``rules/`` and
  root-level prompt fragments at the plugin level. Works on every runtime
  we support (the spec's filesystem layout makes activation trivial on
  Claude Code, our adapter code does the equivalent on DeepAgents /
  LangGraph / etc.). **This is the default and covers the common case.**

Planned as the ecosystem matures (none are implemented yet — rule of
three: promote a class here only after 3+ plugins ship the same custom
shape via their own ``adapters/<runtime>.py``):

* ``MCPServerAdaptor`` — install a plugin as an MCP server *(TODO)*
* ``DeepAgentsSubagentAdaptor`` — register a DeepAgents sub-agent
  (runtime-locked to deepagents) *(TODO)*
* ``LangGraphSubgraphAdaptor`` — install a LangGraph sub-graph *(TODO)*
* ``RAGPipelineAdaptor`` — wire a retriever + index *(TODO)*
* ``SwarmAdaptor`` — bind an OpenAI-swarm / AutoGen-swarm *(TODO)*
* ``WebhookAdaptor`` — register an event handler *(TODO)*

Plugins whose shape doesn't match any built-in ship their own adapter
class in ``plugins/<name>/adapters/<runtime>.py`` — full Python, no
constraint. When 3+ plugins ship the same custom pattern, we promote
the class into this module.
"""

from __future__ import annotations

import os
import shutil
import subprocess
from pathlib import Path

from .protocol import SKILLS_SUBDIR, InstallContext, InstallResult

# Files at the plugin root that are never treated as prompt fragments,
# even if they're markdown. Module-level so tests and other adapters can
# import the set rather than re-declaring it.
SKIP_ROOT_MD = frozenset({"readme.md", "changelog.md", "license.md", "contributing.md"})


def _read_md_files(directory: Path) -> list[tuple[str, str]]:
    """Return [(filename, content)] for all *.md files in directory, sorted."""
    if not directory.is_dir():
        return []
    out: list[tuple[str, str]] = []
    for p in sorted(directory.iterdir()):
        if p.is_file() and p.suffix == ".md":
            out.append((p.name, p.read_text().strip()))
    return out


class AgentskillsAdaptor:
    """Sub-type adaptor for `agentskills.io <https://agentskills.io>`_-format skills.

    This is the default adapter for the "skills + rules" shape — the most
    common pattern. A plugin using this adapter ships:

    * ``skills/<name>/SKILL.md`` (+ optional ``scripts/``, ``references/``,
      ``assets/``) — each skill is a spec-compliant agentskills unit,
      portable to Claude Code, Cursor, Codex, and ~35 other skill-compatible
      tools without modification.
    * ``rules/*.md`` (optional, Molecule AI extension) — always-on prose that
      gets appended to the runtime's memory file (CLAUDE.md).
    * Root-level ``*.md`` (optional) — prompt fragments, also appended to
      memory.

    On ``install()``:
      1. Rules → append to ``/configs/<memory_filename>``, wrapped in a
         ``# Plugin: <name>`` marker for idempotent re-install.
      2. Prompt fragments (``*.md`` at plugin root, excl. README/CHANGELOG/etc.)
         → same treatment.
      3. Skills (``skills/<skill_name>/``) → copied to
         ``/configs/skills/<skill_name>/``. Runtimes with native agentskills
         activation (Claude Code) pick them up automatically; other runtimes'
         loaders scan the same path.

    Uninstall reverses the file copies and strips the rule/fragment block by
    marker (best-effort — if the user edited CLAUDE.md manually, only the
    marker line itself is removed).

    For shapes other than agentskills (MCP server, DeepAgents sub-agent,
    LangGraph sub-graph, RAG pipeline, swarm, webhook handler, etc.), see
    the module docstring for the planned sibling adapters, or ship a custom
    adapter class in the plugin's ``adapters/<runtime>.py``.
    """

    def __init__(self, plugin_name: str, runtime: str) -> None:
        self.plugin_name = plugin_name
        self.runtime = runtime

    # ------------------------------------------------------------------
    # install
    # ------------------------------------------------------------------

    async def install(self, ctx: InstallContext) -> InstallResult:
        result = InstallResult(
            plugin_name=self.plugin_name,
            runtime=self.runtime,
            source="plugin",  # overridden by registry caller if source==registry
        )

        # 1. Rules — append to memory file.
        rules = _read_md_files(ctx.plugin_root / "rules")
        # 2. Prompt fragments — any *.md at plugin root except skip list.
        root_fragments: list[tuple[str, str]] = []
        if ctx.plugin_root.is_dir():
            for p in sorted(ctx.plugin_root.iterdir()):
                if p.is_file() and p.suffix == ".md" and p.name.lower() not in SKIP_ROOT_MD:
                    content = p.read_text().strip()
                    if content:
                        root_fragments.append((p.name, content))

        memory_blocks: list[str] = []
        for filename, content in rules:
            memory_blocks.append(f"# Plugin: {self.plugin_name} / rule: {filename}\n\n{content}")
        for filename, content in root_fragments:
            memory_blocks.append(f"# Plugin: {self.plugin_name} / fragment: {filename}\n\n{content}")

        if memory_blocks:
            joined = "\n\n".join(memory_blocks)
            ctx.append_to_memory(ctx.memory_filename, joined)
            ctx.logger.info(
                "%s: injected %d rule+fragment block(s) into %s",
                self.plugin_name, len(memory_blocks), ctx.memory_filename,
            )

        # 3. Skills — copy each skill dir to /configs/skills/.
        src_skills_dir = ctx.plugin_root / "skills"
        if src_skills_dir.is_dir():
            dst_skills_root = ctx.configs_dir / SKILLS_SUBDIR
            dst_skills_root.mkdir(parents=True, exist_ok=True)
            copied = 0
            for entry in sorted(src_skills_dir.iterdir()):
                if not entry.is_dir():
                    continue
                dst = dst_skills_root / entry.name
                if dst.exists():
                    ctx.logger.debug("%s: skill %s already present, skipping", self.plugin_name, entry.name)
                    continue
                shutil.copytree(entry, dst)
                copied += 1
                for p in dst.rglob("*"):
                    if p.is_file():
                        result.files_written.append(str(p.relative_to(ctx.configs_dir)))
            if copied:
                ctx.logger.info("%s: copied %d skill dir(s) to %s", self.plugin_name, copied, dst_skills_root)

        # 4. Setup script — run setup.sh if present (for npm/pip dependencies).
        # Mirrors sdk/python/molecule_plugin/builtins.py — must stay in sync
        # (drift guard: tests/test_plugins_builtins_drift.py).
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

        return result

    # ------------------------------------------------------------------
    # uninstall
    # ------------------------------------------------------------------

    async def uninstall(self, ctx: InstallContext) -> None:
        # Remove copied skill dirs.
        src_skills_dir = ctx.plugin_root / "skills"
        if src_skills_dir.is_dir():
            for entry in src_skills_dir.iterdir():
                dst = ctx.configs_dir / SKILLS_SUBDIR / entry.name
                if dst.exists() and dst.is_dir():
                    shutil.rmtree(dst)
                    ctx.logger.info("%s: removed %s", self.plugin_name, dst)

        # Best-effort strip of our markers from CLAUDE.md. Users can always
        # edit manually; we only guarantee the injected block's first line
        # is removed so re-install re-adds cleanly.
        memory_path = ctx.configs_dir / ctx.memory_filename
        if not memory_path.exists():
            return
        text = memory_path.read_text()
        prefix = f"# Plugin: {self.plugin_name} / "
        lines = text.splitlines(keepends=True)
        kept = [line for line in lines if not line.startswith(prefix)]
        if len(kept) != len(lines):
            memory_path.write_text("".join(kept))
            ctx.logger.info("%s: stripped markers from %s", self.plugin_name, ctx.memory_filename)


