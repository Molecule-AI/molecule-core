"""Tests for the per-runtime plugin adaptor resolver.

Covers:
  - Resolution order (registry > plugin-shipped > raw-drop)
  - Both adaptor-module conventions (Adaptor class + get_adaptor factory)
  - RawDropAdaptor copies plugin files and surfaces a warning
  - resolve() never raises — always returns a usable adaptor
"""

from __future__ import annotations

import logging
import sys
import textwrap
from pathlib import Path

import pytest

# Resolve workspace-template/ so `import plugins_registry` works in CI without
# requiring an installed package.
_WS_TEMPLATE = Path(__file__).resolve().parents[1]
if str(_WS_TEMPLATE) not in sys.path:
    sys.path.insert(0, str(_WS_TEMPLATE))

from plugins_registry import (  # noqa: E402
    AdaptorSource,
    InstallContext,
    PluginAdaptor,
    RawDropAdaptor,
    resolve,
)


# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------

@pytest.fixture
def configs_dir(tmp_path: Path) -> Path:
    d = tmp_path / "configs"
    d.mkdir()
    return d


@pytest.fixture
def plugin_root(tmp_path: Path) -> Path:
    p = tmp_path / "demo-plugin"
    (p / "rules").mkdir(parents=True)
    (p / "rules" / "rules.md").write_text("- be excellent\n")
    (p / "plugin.yaml").write_text("name: demo-plugin\nruntimes: [test_runtime]\n")
    return p


def _ctx(configs_dir: Path, plugin_root: Path, runtime: str = "test_runtime") -> InstallContext:
    return InstallContext(
        configs_dir=configs_dir,
        workspace_id="ws-test",
        runtime=runtime,
        plugin_root=plugin_root,
        logger=logging.getLogger("test"),
    )


# ---------------------------------------------------------------------------
# RawDropAdaptor
# ---------------------------------------------------------------------------

async def test_raw_drop_copies_plugin_and_warns(configs_dir: Path, plugin_root: Path):
    adaptor = RawDropAdaptor("demo-plugin", "test_runtime")
    result = await adaptor.install(_ctx(configs_dir, plugin_root))

    dst = configs_dir / "plugins" / "demo-plugin"
    assert dst.exists()
    assert (dst / "rules" / "rules.md").read_text() == "- be excellent\n"
    assert result.source == "raw_drop"
    assert any("no adaptor" in w for w in result.warnings)
    assert result.tools_registered == []


async def test_raw_drop_is_idempotent(configs_dir: Path, plugin_root: Path):
    adaptor = RawDropAdaptor("demo-plugin", "test_runtime")
    await adaptor.install(_ctx(configs_dir, plugin_root))
    # Second install must not raise (shutil.copytree would otherwise complain)
    result = await adaptor.install(_ctx(configs_dir, plugin_root))
    assert result.source == "raw_drop"


async def test_raw_drop_uninstall_removes_files(configs_dir: Path, plugin_root: Path):
    adaptor = RawDropAdaptor("demo-plugin", "test_runtime")
    ctx = _ctx(configs_dir, plugin_root)
    await adaptor.install(ctx)
    await adaptor.uninstall(ctx)
    assert not (configs_dir / "plugins" / "demo-plugin").exists()


# ---------------------------------------------------------------------------
# resolve() — order: registry > plugin-shipped > raw_drop
# ---------------------------------------------------------------------------

def test_resolve_falls_back_to_raw_drop_when_no_adaptor(plugin_root: Path):
    adaptor, source = resolve("nonexistent-plugin", "claude_code", plugin_root)
    assert source == AdaptorSource.RAW_DROP
    assert isinstance(adaptor, RawDropAdaptor)


def test_resolve_prefers_plugin_shipped_over_raw_drop(plugin_root: Path):
    """Plugin ships its own adaptor → must beat raw-drop."""
    (plugin_root / "adapters").mkdir()
    (plugin_root / "adapters" / "test_runtime.py").write_text(textwrap.dedent("""
        from plugins_registry.protocol import InstallResult

        class Adaptor:
            def __init__(self, plugin_name, runtime):
                self.plugin_name = plugin_name
                self.runtime = runtime
            async def install(self, ctx):
                return InstallResult(plugin_name=self.plugin_name, runtime=self.runtime, source="plugin")
            async def uninstall(self, ctx):
                pass
    """))

    adaptor, source = resolve("demo-plugin", "test_runtime", plugin_root)
    assert source == AdaptorSource.PLUGIN
    assert not isinstance(adaptor, RawDropAdaptor)


def test_resolve_supports_get_adaptor_factory(plugin_root: Path):
    """Adaptor module exposing get_adaptor() instead of Adaptor class."""
    (plugin_root / "adapters").mkdir()
    (plugin_root / "adapters" / "test_runtime.py").write_text(textwrap.dedent("""
        from plugins_registry.protocol import InstallResult

        class _Impl:
            def __init__(self, plugin_name, runtime):
                self.plugin_name = plugin_name
                self.runtime = runtime
            async def install(self, ctx):
                return InstallResult(plugin_name=self.plugin_name, runtime=self.runtime, source="plugin")
            async def uninstall(self, ctx):
                pass

        def get_adaptor(plugin_name, runtime):
            return _Impl(plugin_name, runtime)
    """))

    adaptor, source = resolve("demo-plugin", "test_runtime", plugin_root)
    assert source == AdaptorSource.PLUGIN


async def test_resolve_get_adaptor_factory_install(plugin_root: Path, tmp_path: Path):
    """Installing an adaptor returned by get_adaptor() works end-to-end."""
    (plugin_root / "adapters").mkdir()
    (plugin_root / "adapters" / "test_runtime.py").write_text(textwrap.dedent("""
        from plugins_registry.protocol import InstallResult
        class _Impl:
            def __init__(self, plugin_name, runtime):
                self.plugin_name = plugin_name
                self.runtime = runtime
            async def install(self, ctx):
                return InstallResult(plugin_name=self.plugin_name, runtime=self.runtime, source="plugin")
            async def uninstall(self, ctx): pass
        def get_adaptor(plugin_name, runtime):
            return _Impl(plugin_name, runtime)
    """))
    adaptor, _ = resolve("demo-plugin", "test_runtime", plugin_root)
    result = await adaptor.install(_ctx(tmp_path, plugin_root))
    assert result.source == "plugin"


async def test_resolve_registry_beats_plugin_shipped(plugin_root: Path, monkeypatch, tmp_path: Path):
    """Platform registry must override plugin-shipped adaptor (promote-to-default path)."""
    # Plant a plugin-shipped adaptor first.
    (plugin_root / "adapters").mkdir()
    (plugin_root / "adapters" / "test_runtime.py").write_text(textwrap.dedent("""
        from plugins_registry.protocol import InstallResult
        class Adaptor:
            def __init__(self, plugin_name, runtime):
                self.plugin_name = plugin_name
                self.runtime = runtime
            async def install(self, ctx):
                return InstallResult(plugin_name=self.plugin_name, runtime=self.runtime, source="plugin")
            async def uninstall(self, ctx): pass
    """))

    # Now plant a registry override by monkeypatching _REGISTRY_ROOT to a temp dir.
    fake_registry = tmp_path / "fake_registry"
    (fake_registry / "demo-plugin").mkdir(parents=True)
    (fake_registry / "demo-plugin" / "test_runtime.py").write_text(textwrap.dedent("""
        from plugins_registry.protocol import InstallResult
        class Adaptor:
            def __init__(self, plugin_name, runtime):
                self.plugin_name = plugin_name
                self.runtime = runtime
            async def install(self, ctx):
                return InstallResult(plugin_name=self.plugin_name, runtime=self.runtime, source="registry")
            async def uninstall(self, ctx): pass
    """))

    import plugins_registry as pr
    monkeypatch.setattr(pr, "_REGISTRY_ROOT", fake_registry)

    adaptor, source = pr.resolve("demo-plugin", "test_runtime", plugin_root)
    assert source == AdaptorSource.REGISTRY
    result = await adaptor.install(_ctx(tmp_path, plugin_root))
    assert result.source == "registry"


def test_resolve_handles_broken_adaptor_module(plugin_root: Path):
    """Broken adaptor file falls back gracefully — never crashes the install."""
    (plugin_root / "adapters").mkdir()
    (plugin_root / "adapters" / "test_runtime.py").write_text("syntax error this is not python")

    adaptor, source = resolve("demo-plugin", "test_runtime", plugin_root)
    # Falls through to raw-drop because the broken module fails to import.
    assert source == AdaptorSource.RAW_DROP


def test_protocol_runtime_check():
    """RawDropAdaptor must satisfy the Protocol at runtime."""
    assert isinstance(RawDropAdaptor("p", "r"), PluginAdaptor)


# ---------------------------------------------------------------------------
# Edge cases on adaptor loading
# ---------------------------------------------------------------------------

def test_resolve_module_with_neither_adaptor_nor_factory(plugin_root: Path):
    """Adaptor file that defines neither ``Adaptor`` nor ``get_adaptor()``
    falls back to raw-drop (can't instantiate anything)."""
    (plugin_root / "adapters").mkdir()
    (plugin_root / "adapters" / "test_runtime.py").write_text(
        "# no Adaptor, no get_adaptor — just a valid module\nX = 1\n"
    )
    _, source = resolve("demo-plugin", "test_runtime", plugin_root)
    assert source == AdaptorSource.RAW_DROP


def test_resolve_get_adaptor_factory_raises(plugin_root: Path):
    """get_adaptor() that raises → falls back to raw-drop gracefully."""
    (plugin_root / "adapters").mkdir()
    (plugin_root / "adapters" / "test_runtime.py").write_text(textwrap.dedent("""
        def get_adaptor(plugin_name, runtime):
            raise ValueError("kaboom")
    """))
    _, source = resolve("demo-plugin", "test_runtime", plugin_root)
    assert source == AdaptorSource.RAW_DROP


def test_resolve_adaptor_class_construction_raises(plugin_root: Path):
    """Adaptor class whose __init__ raises → falls back to raw-drop."""
    (plugin_root / "adapters").mkdir()
    (plugin_root / "adapters" / "test_runtime.py").write_text(textwrap.dedent("""
        class Adaptor:
            def __init__(self, *args, **kwargs):
                raise RuntimeError("nope")
    """))
    _, source = resolve("demo-plugin", "test_runtime", plugin_root)
    assert source == AdaptorSource.RAW_DROP


def test_resolve_adaptor_class_zero_arg_fallback(plugin_root: Path):
    """Adaptor class whose (name, runtime) ctor raises TypeError → try zero-arg."""
    (plugin_root / "adapters").mkdir()
    (plugin_root / "adapters" / "test_runtime.py").write_text(textwrap.dedent("""
        from plugins_registry.protocol import InstallResult
        class Adaptor:
            plugin_name = "demo-plugin"
            runtime = "test_runtime"
            def __init__(self):
                pass
            async def install(self, ctx):
                return InstallResult(plugin_name=self.plugin_name, runtime=self.runtime, source="plugin")
            async def uninstall(self, ctx):
                pass
    """))
    # TypeError forces the fallback path: `cls(plugin_name, runtime)` fails
    # because the class takes no args, so we retry with `cls()`.
    _, source = resolve("demo-plugin", "test_runtime", plugin_root)
    assert source == AdaptorSource.PLUGIN


def test_load_module_bailout_when_spec_is_none(monkeypatch, plugin_root: Path):
    """Defensive path: ``spec_from_file_location`` returns None. Forced via
    monkeypatch since real filesystems never trigger it for .py files."""
    import importlib.util as iu
    import plugins_registry as pr

    (plugin_root / "adapters").mkdir()
    (plugin_root / "adapters" / "test_runtime.py").write_text("class Adaptor: pass\n")

    real = iu.spec_from_file_location
    def fake_spec(name, path, *a, **kw):
        if path.name == "test_runtime.py":
            return None
        return real(name, path, *a, **kw)
    monkeypatch.setattr(pr.importlib.util, "spec_from_file_location", fake_spec)

    _, source = pr.resolve("demo-plugin", "test_runtime", plugin_root)
    assert source == AdaptorSource.RAW_DROP


def test_resolve_registry_bails_when_load_returns_none(monkeypatch, tmp_path: Path, plugin_root: Path):
    """Registry path exists but the module fails to load → falls through to
    plugin-shipped (or raw-drop if that's also missing). Exercises the
    ``if module is None: return None`` bail-out in ``_resolve_registry``."""
    import plugins_registry as pr

    fake_registry = tmp_path / "fake_registry"
    (fake_registry / "demo-plugin").mkdir(parents=True)
    (fake_registry / "demo-plugin" / "test_runtime.py").write_text("class Adaptor: pass\n")
    monkeypatch.setattr(pr, "_REGISTRY_ROOT", fake_registry)

    # Force _load_module_from_path to return None when asked for this module.
    monkeypatch.setattr(pr, "_load_module_from_path", lambda name, path: None)

    _, source = pr.resolve("demo-plugin", "test_runtime", plugin_root)
    # Both registry and plugin-shipped now yield None → raw-drop.
    assert source == AdaptorSource.RAW_DROP


def test_resolve_registry_missing_module_falls_through(monkeypatch, tmp_path: Path, plugin_root: Path):
    """Registry root exists but has neither plugin dir for this name →
    plugin-shipped or raw-drop takes over (not a crash)."""
    import plugins_registry as pr
    monkeypatch.setattr(pr, "_REGISTRY_ROOT", tmp_path / "empty-registry")
    _, source = pr.resolve("demo-plugin", "test_runtime", plugin_root)
    assert source == AdaptorSource.RAW_DROP
