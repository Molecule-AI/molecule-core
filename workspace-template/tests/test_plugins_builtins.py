"""Edge-case tests for :class:`AgentskillsAdaptor`.

Covers:
  - Uninstall removes copied skill dirs and strips CLAUDE.md markers
  - Re-install is idempotent (skill already present → skip, marker → skip)
  - Plugin with only prompt fragments (no rules/, no skills/)
  - Empty rules directory doesn't write an empty block
  - README.md / CHANGELOG.md are skipped at the root (not treated as fragments)
  - Uninstall is safe on a plugin that was never installed
"""

from __future__ import annotations

import logging
import sys
from pathlib import Path

import pytest

_WS_TEMPLATE = Path(__file__).resolve().parents[1]
if str(_WS_TEMPLATE) not in sys.path:
    sys.path.insert(0, str(_WS_TEMPLATE))

from plugins_registry import InstallContext  # noqa: E402
from plugins_registry.builtins import AgentskillsAdaptor  # noqa: E402


def _make_ctx(configs_dir: Path, plugin_root: Path) -> InstallContext:
    def _append(filename: str, content: str) -> None:
        target = configs_dir / filename
        existing = target.read_text() if target.exists() else ""
        first_line = content.splitlines()[0] if content else ""
        if first_line and first_line in existing:
            return
        with open(target, "a") as f:
            if existing and not existing.endswith("\n"):
                f.write("\n")
            f.write(content + "\n")

    return InstallContext(
        configs_dir=configs_dir,
        workspace_id="ws-test",
        runtime="claude_code",
        plugin_root=plugin_root,
        append_to_memory=_append,
        logger=logging.getLogger("test"),
    )


@pytest.fixture
def full_plugin(tmp_path: Path) -> Path:
    """Plugin with rules + skills + a fragment + a skip-list file."""
    p = tmp_path / "my-plugin"
    (p / "rules").mkdir(parents=True)
    (p / "rules" / "r1.md").write_text("- rule one\n")
    (p / "skills" / "my-skill").mkdir(parents=True)
    (p / "skills" / "my-skill" / "SKILL.md").write_text("# skill\n")
    (p / "fragment.md").write_text("extra prompt\n")
    (p / "README.md").write_text("should be ignored\n")  # skip list
    (p / "CHANGELOG.md").write_text("should be ignored\n")
    return p


async def test_uninstall_removes_skills_and_strips_markers(tmp_path: Path, full_plugin: Path):
    configs = tmp_path / "configs"
    configs.mkdir()
    adaptor = AgentskillsAdaptor("my-plugin", "claude_code")
    ctx = _make_ctx(configs, full_plugin)

    await adaptor.install(ctx)
    assert (configs / "skills" / "my-skill" / "SKILL.md").exists()
    claude_md = configs / "CLAUDE.md"
    assert "# Plugin: my-plugin / rule: r1.md" in claude_md.read_text()
    assert "# Plugin: my-plugin / fragment: fragment.md" in claude_md.read_text()

    await adaptor.uninstall(ctx)
    # Skill dir gone, markers removed (at least their header lines).
    assert not (configs / "skills" / "my-skill").exists()
    remaining = claude_md.read_text()
    assert "# Plugin: my-plugin /" not in remaining


async def test_install_is_idempotent_on_skills_and_memory(tmp_path: Path, full_plugin: Path):
    configs = tmp_path / "configs"
    configs.mkdir()
    adaptor = AgentskillsAdaptor("my-plugin", "claude_code")
    ctx = _make_ctx(configs, full_plugin)

    await adaptor.install(ctx)
    await adaptor.install(ctx)
    # Skill dir still exists and wasn't duplicated.
    assert (configs / "skills" / "my-skill" / "SKILL.md").exists()
    # Marker present but only once — count unique header lines.
    text = (configs / "CLAUDE.md").read_text()
    assert text.count("# Plugin: my-plugin / rule: r1.md") == 1
    assert text.count("# Plugin: my-plugin / fragment: fragment.md") == 1


async def test_readme_and_changelog_not_treated_as_fragments(tmp_path: Path, full_plugin: Path):
    configs = tmp_path / "configs"
    configs.mkdir()
    await AgentskillsAdaptor("my-plugin", "claude_code").install(_make_ctx(configs, full_plugin))
    text = (configs / "CLAUDE.md").read_text()
    assert "should be ignored" not in text
    assert "# Plugin: my-plugin / fragment: README.md" not in text


async def test_plugin_with_no_content_is_noop(tmp_path: Path):
    """Empty plugin dir → install succeeds, no CLAUDE.md created, no skills/."""
    configs = tmp_path / "configs"
    configs.mkdir()
    plugin_root = tmp_path / "bare"
    plugin_root.mkdir()

    result = await AgentskillsAdaptor("bare", "claude_code").install(_make_ctx(configs, plugin_root))
    assert result.plugin_name == "bare"
    assert not (configs / "CLAUDE.md").exists()
    assert not (configs / "skills").exists()


async def test_plugin_with_empty_rules_dir(tmp_path: Path):
    """Plugin has a rules/ dir but no .md files → no memory write."""
    configs = tmp_path / "configs"
    configs.mkdir()
    plugin_root = tmp_path / "demo"
    (plugin_root / "rules").mkdir(parents=True)
    # no .md files

    await AgentskillsAdaptor("demo", "claude_code").install(_make_ctx(configs, plugin_root))
    assert not (configs / "CLAUDE.md").exists()


async def test_uninstall_safe_when_never_installed(tmp_path: Path, full_plugin: Path):
    configs = tmp_path / "configs"
    configs.mkdir()
    # Never install — uninstall must not raise.
    await AgentskillsAdaptor("my-plugin", "claude_code").uninstall(_make_ctx(configs, full_plugin))


async def test_install_preserves_unrelated_claude_md_content(tmp_path: Path, full_plugin: Path):
    """User-authored CLAUDE.md content must not be touched by install/uninstall."""
    configs = tmp_path / "configs"
    configs.mkdir()
    (configs / "CLAUDE.md").write_text("# User Note\n\nHand-written content.\n")

    adaptor = AgentskillsAdaptor("my-plugin", "claude_code")
    ctx = _make_ctx(configs, full_plugin)
    await adaptor.install(ctx)
    await adaptor.uninstall(ctx)

    remaining = (configs / "CLAUDE.md").read_text()
    assert "Hand-written content" in remaining
    assert "# User Note" in remaining


async def test_install_ignores_non_dir_entries_in_skills(tmp_path: Path):
    """A stray file (not a directory) inside skills/ is skipped, not copied."""
    configs = tmp_path / "configs"
    configs.mkdir()
    plugin_root = tmp_path / "demo"
    (plugin_root / "skills").mkdir(parents=True)
    (plugin_root / "skills" / "loose-file.txt").write_text("not a skill")
    (plugin_root / "skills" / "real-skill").mkdir()
    (plugin_root / "skills" / "real-skill" / "SKILL.md").write_text("# ok")

    await AgentskillsAdaptor("demo", "claude_code").install(_make_ctx(configs, plugin_root))
    assert (configs / "skills" / "real-skill" / "SKILL.md").exists()
    # The loose file must not have been copied to /configs/skills/ as a file.
    assert not (configs / "skills" / "loose-file.txt").exists()


async def test_raw_drop_copies_skills_for_unsupported_runtime(tmp_path: Path):
    """When a plugin falls through to raw-drop, skills still land under
    /configs/plugins/<name>/skills/ (not /configs/skills/) so the user can
    at least inspect them."""
    from plugins_registry import resolve, AdaptorSource

    configs = tmp_path / "configs"
    configs.mkdir()
    plugin_root = tmp_path / "novel-plugin"
    (plugin_root / "skills" / "magic").mkdir(parents=True)
    (plugin_root / "skills" / "magic" / "SKILL.md").write_text("# magic")

    adaptor, source = resolve("novel-plugin", "unknown_runtime", plugin_root)
    assert source == AdaptorSource.RAW_DROP
    result = await adaptor.install(_make_ctx(configs, plugin_root))
    assert result.warnings  # warning was surfaced
    assert (configs / "plugins" / "novel-plugin" / "skills" / "magic" / "SKILL.md").exists()


async def test_install_skips_skill_when_already_present(tmp_path: Path, full_plugin: Path):
    """If /configs/skills/<name>/ already exists (e.g. user placed it there
    manually or from another plugin), install must not overwrite or raise."""
    configs = tmp_path / "configs"
    (configs / "skills" / "my-skill").mkdir(parents=True)
    (configs / "skills" / "my-skill" / "SKILL.md").write_text("# USER'S OWN")

    await AgentskillsAdaptor("my-plugin", "claude_code").install(_make_ctx(configs, full_plugin))
    # Pre-existing content preserved.
    assert (configs / "skills" / "my-skill" / "SKILL.md").read_text() == "# USER'S OWN"


# ---------------------------------------------------------------------------
# memory_filename plumbing — AgentskillsAdaptor must honour a non-default
# memory file (for runtimes that read AGENTS.md, .windsurfrules, etc.).
# ---------------------------------------------------------------------------


async def test_agentskills_adaptor_honours_non_default_memory_filename(tmp_path: Path, full_plugin: Path):
    """Overriding ctx.memory_filename routes rule/fragment writes there."""
    configs = tmp_path / "configs"
    configs.mkdir()

    written = {}
    def _append(filename: str, content: str) -> None:
        written[filename] = content

    ctx = InstallContext(
        configs_dir=configs,
        workspace_id="ws",
        runtime="custom_runtime",
        plugin_root=full_plugin,
        memory_filename="AGENTS.md",   # non-default
        append_to_memory=_append,
        logger=logging.getLogger("test"),
    )

    await AgentskillsAdaptor("my-plugin", "custom_runtime").install(ctx)

    # Memory writes went to AGENTS.md, not CLAUDE.md.
    assert "AGENTS.md" in written
    assert "CLAUDE.md" not in written
    assert "# Plugin: my-plugin /" in written["AGENTS.md"]


async def test_agentskills_adaptor_uninstall_honours_non_default_memory_filename(tmp_path: Path, full_plugin: Path):
    """Uninstall strips markers from the same non-default memory file."""
    configs = tmp_path / "configs"
    configs.mkdir()
    (configs / "AGENTS.md").write_text(
        "# User content\n\n# Plugin: my-plugin / rule: r1.md\n\n- rule\n"
    )

    ctx = InstallContext(
        configs_dir=configs,
        workspace_id="ws",
        runtime="custom_runtime",
        plugin_root=full_plugin,
        memory_filename="AGENTS.md",
        logger=logging.getLogger("test"),
    )

    await AgentskillsAdaptor("my-plugin", "custom_runtime").uninstall(ctx)

    remaining = (configs / "AGENTS.md").read_text()
    assert "# User content" in remaining
    assert "# Plugin: my-plugin /" not in remaining
    # CLAUDE.md must not have been created as a side effect.
    assert not (configs / "CLAUDE.md").exists()


def test_install_context_default_memory_filename_is_claude_md():
    """Regression check: the default plumbing picks CLAUDE.md so existing
    runtimes (Claude Code, DeepAgents) keep working without change."""
    from plugins_registry.protocol import DEFAULT_MEMORY_FILENAME
    assert DEFAULT_MEMORY_FILENAME == "CLAUDE.md"

    ctx = InstallContext(
        configs_dir=Path("/tmp"),
        workspace_id="w",
        runtime="claude_code",
        plugin_root=Path("/tmp"),
    )
    assert ctx.memory_filename == "CLAUDE.md"


async def test_base_adapter_memory_filename_override_flows_through_install(tmp_path: Path):
    """End-to-end: a BaseAdapter subclass overriding memory_filename() has
    its value populated into ctx.memory_filename by install_plugins_via_registry.
    Plumbs W2 all the way from BaseAdapter hook down to AgentskillsAdaptor.install."""
    from types import SimpleNamespace
    from adapters.base import BaseAdapter, AdapterConfig

    class _CustomRuntime(BaseAdapter):
        @staticmethod
        def name() -> str: return "custom_runtime"
        @staticmethod
        def display_name() -> str: return "Custom"
        @staticmethod
        def description() -> str: return "test runtime"
        def memory_filename(self) -> str: return "AGENTS.md"
        async def setup(self, config): return None
        async def create_executor(self, config): return None

    # Plant a plugin with our registered claude_code adapter (runtime name
    # coercion: custom_runtime has no adapter → raw-drop, but AgentskillsAdaptor
    # is used when we ship adapters/custom_runtime.py).
    plugin_root = tmp_path / "plugins" / "my-plugin"
    (plugin_root / "rules").mkdir(parents=True)
    (plugin_root / "rules" / "r.md").write_text("- rule")
    (plugin_root / "adapters").mkdir()
    (plugin_root / "adapters" / "custom_runtime.py").write_text(
        "from plugins_registry.builtins import AgentskillsAdaptor as Adaptor\n"
    )

    configs = tmp_path / "configs"
    configs.mkdir()
    cfg = AdapterConfig(
        model="x", config_path=str(configs), workspace_id="ws",
    )
    plugins = SimpleNamespace(
        plugins=[SimpleNamespace(name="my-plugin", path=str(plugin_root))],
    )

    await _CustomRuntime().install_plugins_via_registry(cfg, plugins)

    # The hook value (AGENTS.md) propagated into the memory file path.
    assert (configs / "AGENTS.md").exists()
    assert "# Plugin: my-plugin /" in (configs / "AGENTS.md").read_text()
    assert not (configs / "CLAUDE.md").exists()


# ---------- setup.sh hook ----------------------------------------------------

async def test_setup_sh_runs_with_configs_dir_env(tmp_path: Path):
    """setup.sh in plugin root must execute with CONFIGS_DIR exported and
    cwd at plugin_root. Marker file proves the hook ran."""
    plugin = tmp_path / "p"
    (plugin / "skills" / "s1").mkdir(parents=True)
    (plugin / "skills" / "s1" / "SKILL.md").write_text("---\nname: s1\ndescription: d\n---\n")
    setup = plugin / "setup.sh"
    setup.write_text(
        '#!/bin/bash\nset -e\n'
        'echo "ran from $PWD" > "$CONFIGS_DIR/setup-trace.txt"\n'
    )
    setup.chmod(0o755)
    configs = tmp_path / "configs"
    configs.mkdir()

    result = await AgentskillsAdaptor("p", "claude_code").install(_make_ctx(configs, plugin))

    trace = configs / "setup-trace.txt"
    assert trace.is_file(), "setup.sh did not run"
    assert str(plugin) in trace.read_text(), "setup.sh did not run with cwd=plugin_root"
    assert result.warnings == [], "successful setup must not warn"


async def test_setup_sh_nonzero_exit_records_warning_does_not_raise(tmp_path: Path):
    """A failing setup.sh must NOT abort install — skills/rules still land,
    the failure is surfaced as a warning on InstallResult."""
    plugin = tmp_path / "p"
    plugin.mkdir()
    setup = plugin / "setup.sh"
    setup.write_text('#!/bin/bash\necho "boom" >&2\nexit 7\n')
    setup.chmod(0o755)
    configs = tmp_path / "configs"
    configs.mkdir()

    result = await AgentskillsAdaptor("p", "claude_code").install(_make_ctx(configs, plugin))

    assert result.warnings, "non-zero exit must produce a warning"
    assert "exited 7" in result.warnings[0]
    assert "boom" in result.warnings[0]


async def test_setup_sh_timeout_records_warning(tmp_path: Path, monkeypatch):
    """A hanging setup.sh must be killed after the bounded timeout and
    surfaced as a warning — not allowed to wedge install indefinitely."""
    import subprocess as _sp
    plugin = tmp_path / "p"
    plugin.mkdir()
    (plugin / "setup.sh").write_text("#!/bin/bash\nsleep 999\n")
    (plugin / "setup.sh").chmod(0o755)
    configs = tmp_path / "configs"
    configs.mkdir()

    def _raise_timeout(*a, **kw):
        raise _sp.TimeoutExpired(cmd=a[0], timeout=120)
    monkeypatch.setattr("plugins_registry.builtins.subprocess.run", _raise_timeout)

    result = await AgentskillsAdaptor("p", "claude_code").install(_make_ctx(configs, plugin))

    assert any("timed out" in w for w in result.warnings)


async def test_setup_sh_absent_no_warning(tmp_path: Path):
    """No setup.sh → no hook executed, no warnings."""
    plugin = tmp_path / "p"
    plugin.mkdir()
    configs = tmp_path / "configs"
    configs.mkdir()

    result = await AgentskillsAdaptor("p", "claude_code").install(_make_ctx(configs, plugin))

    assert result.warnings == []
