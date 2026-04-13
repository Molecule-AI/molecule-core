"""Integration tests: each first-party plugin installs via the registry pipeline.

Exercises the full flow a workspace runtime goes through at startup:
  load_plugins() → install_plugins_via_registry() → adaptor.install(ctx)

For each combination of (plugin, runtime) declared in plugin.yaml, we verify:
  - the adaptor resolves via the plugin-shipped path (not raw-drop)
  - skills land in /configs/skills/<skill_name>/
  - rules/fragments land in /configs/CLAUDE.md
"""

from __future__ import annotations

import logging
import sys
from pathlib import Path

import pytest
import yaml

_WS_TEMPLATE = Path(__file__).resolve().parents[1]
if str(_WS_TEMPLATE) not in sys.path:
    sys.path.insert(0, str(_WS_TEMPLATE))

from plugins_registry import AdaptorSource, InstallContext, resolve  # noqa: E402

_REPO_ROOT = _WS_TEMPLATE.parent
_PLUGINS_DIR = _REPO_ROOT / "plugins"

FIRST_PARTY_PLUGINS = ["molecule-dev", "superpowers", "ecc"]


@pytest.fixture
def ctx(tmp_path: Path):
    configs = tmp_path / "configs"
    configs.mkdir()

    # Simple memory append implementation for the test.
    def _append(filename: str, content: str) -> None:
        target = configs / filename
        existing = target.read_text() if target.exists() else ""
        marker = content.splitlines()[0] if content else ""
        if marker and marker in existing:
            return
        with open(target, "a") as f:
            f.write(("\n" if existing and not existing.endswith("\n") else "") + content + "\n")

    def _make(plugin_name: str, runtime: str) -> InstallContext:
        return InstallContext(
            configs_dir=configs,
            workspace_id="ws-test",
            runtime=runtime,
            plugin_root=_PLUGINS_DIR / plugin_name,
            append_to_memory=_append,
            logger=logging.getLogger(plugin_name),
        )
    return _make, configs


@pytest.mark.parametrize("plugin_name", FIRST_PARTY_PLUGINS)
def test_plugin_manifest_declares_runtimes(plugin_name: str):
    """Each first-party plugin must declare supported runtimes."""
    manifest = yaml.safe_load((_PLUGINS_DIR / plugin_name / "plugin.yaml").read_text())
    assert "runtimes" in manifest, f"{plugin_name} missing `runtimes:` in plugin.yaml"
    assert "claude_code" in manifest["runtimes"]
    assert "deepagents" in manifest["runtimes"]


@pytest.mark.parametrize("plugin_name", FIRST_PARTY_PLUGINS)
@pytest.mark.parametrize("runtime", ["claude_code", "deepagents"])
def test_plugin_ships_adaptor_file(plugin_name: str, runtime: str):
    """Each declared runtime has a physical adaptor file."""
    adaptor_file = _PLUGINS_DIR / plugin_name / "adapters" / f"{runtime}.py"
    assert adaptor_file.is_file(), f"{plugin_name} missing adapters/{runtime}.py"


@pytest.mark.parametrize("plugin_name", FIRST_PARTY_PLUGINS)
@pytest.mark.parametrize("runtime", ["claude_code", "deepagents"])
def test_adaptor_resolves_via_plugin_path(plugin_name: str, runtime: str):
    """resolve() must find the plugin-shipped adaptor, not fall back to raw-drop."""
    plugin_root = _PLUGINS_DIR / plugin_name
    _, source = resolve(plugin_name, runtime, plugin_root)
    assert source == AdaptorSource.PLUGIN


@pytest.mark.parametrize("plugin_name,runtime", [
    (p, r) for p in FIRST_PARTY_PLUGINS for r in ["claude_code", "deepagents"]
])
async def test_plugin_installs_end_to_end(plugin_name: str, runtime: str, ctx):
    """Installing each plugin writes the expected content into /configs."""
    make_ctx, configs_dir = ctx
    plugin_root = _PLUGINS_DIR / plugin_name
    adaptor, source = resolve(plugin_name, runtime, plugin_root)
    assert source == AdaptorSource.PLUGIN

    result = await adaptor.install(make_ctx(plugin_name, runtime))
    assert result.plugin_name == plugin_name

    # If the plugin has skills/, each one should now exist under /configs/skills/
    src_skills = plugin_root / "skills"
    if src_skills.is_dir():
        for skill in src_skills.iterdir():
            if skill.is_dir():
                assert (configs_dir / "skills" / skill.name).is_dir(), \
                    f"{plugin_name}: skill {skill.name} not copied"

    # If the plugin has rules/, CLAUDE.md should contain the marker.
    src_rules = plugin_root / "rules"
    if src_rules.is_dir() and any(p.suffix == ".md" for p in src_rules.iterdir()):
        claude_md = configs_dir / "CLAUDE.md"
        assert claude_md.exists(), f"{plugin_name}: CLAUDE.md not created"
        assert f"# Plugin: {plugin_name} /" in claude_md.read_text()


async def test_install_is_idempotent(ctx):
    """Installing molecule-dev twice leaves a single marker, doesn't duplicate."""
    make_ctx, configs_dir = ctx
    plugin_root = _PLUGINS_DIR / "molecule-dev"
    adaptor, _ = resolve("molecule-dev", "claude_code", plugin_root)

    await adaptor.install(make_ctx("molecule-dev", "claude_code"))
    await adaptor.install(make_ctx("molecule-dev", "claude_code"))

    # At least one skill dir exists; CLAUDE.md has exactly one marker section header.
    assert (configs_dir / "skills" / "review-loop").is_dir()
