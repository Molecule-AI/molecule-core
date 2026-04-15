"""Tests for Phase 2 auth_scheme dispatch in adapters/hermes/executor.py.

These cover the NEW behavior only (HermesA2AExecutor._do_inference dispatch
based on ProviderConfig.auth_scheme). Phase 1 registry tests live in
test_hermes_providers.py — unchanged by Phase 2.
"""

from __future__ import annotations

import sys
from pathlib import Path
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

# Load providers.py directly (same pattern as test_hermes_providers.py)
_HERMES_DIR = Path(__file__).parent.parent / "adapters" / "hermes"
sys.path.insert(0, str(_HERMES_DIR))
import providers  # type: ignore  # noqa: E402


def _make_executor(provider_name: str):
    """Build a HermesA2AExecutor directly without going through create_executor.

    We import executor lazily inside the function because the module-level
    import chain (``from .providers import ...``) uses a relative import that
    only resolves when loaded as part of the ``adapters.hermes`` package.
    The test loads it via direct sys.path manipulation, which bypasses the
    package loader, so we import providers-as-sibling and then reconstruct
    the executor with the same shape.
    """
    # We can't import executor.py directly due to the relative-import head,
    # so instantiate the executor class by replaying its definition inline.
    # Simpler: test the dispatch logic via providers.PROVIDERS + the public
    # resolve helpers, plus a mock for the inference methods.
    cfg = providers.PROVIDERS[provider_name]
    # Reach into executor via sys.path trick
    import importlib.util
    spec = importlib.util.spec_from_file_location(
        "hermes_executor_under_test",
        _HERMES_DIR / "executor.py",
    )
    # The executor module has a relative import `from .providers import ...`
    # which fails under direct spec_from_file_location. Monkey-patch sys.modules
    # so the relative import resolves to our directly-loaded providers module.
    sys.modules["hermes_executor_under_test.providers"] = providers
    # Also alias the package-style import path so `from .providers import X`
    # inside executor.py finds it.
    pkg_name = "hermes_executor_under_test"
    sys.modules.setdefault(pkg_name, MagicMock())
    sys.modules[pkg_name].providers = providers  # type: ignore
    # Read + compile executor.py with the relative import rewritten
    src = (_HERMES_DIR / "executor.py").read_text()
    src = src.replace("from .providers import", "from providers import")
    ns: dict = {}
    exec(compile(src, str(_HERMES_DIR / "executor.py"), "exec"), ns)
    HermesA2AExecutor = ns["HermesA2AExecutor"]
    return HermesA2AExecutor(
        provider_cfg=cfg,
        api_key="test-key",
        model=cfg.default_model,
    )


def test_anthropic_entry_has_anthropic_scheme():
    """The registry flip: Phase 2 sets anthropic's auth_scheme to 'anthropic'."""
    cfg = providers.PROVIDERS["anthropic"]
    assert cfg.auth_scheme == "anthropic"


def test_all_other_providers_still_openai_scheme():
    """Phase 2 only changes anthropic. Every other provider keeps auth_scheme='openai'."""
    for name, cfg in providers.PROVIDERS.items():
        if name == "anthropic":
            continue
        assert cfg.auth_scheme == "openai", (
            f"{name} unexpectedly has auth_scheme={cfg.auth_scheme!r}"
        )


@pytest.mark.asyncio
async def test_dispatch_openai_scheme_calls_openai_compat():
    """auth_scheme='openai' → _do_openai_compat runs, _do_anthropic_native does not."""
    executor = _make_executor("openai")
    executor._do_openai_compat = AsyncMock(return_value="openai-result")
    executor._do_anthropic_native = AsyncMock(return_value="should-not-run")

    result = await executor._do_inference("hello")

    executor._do_openai_compat.assert_awaited_once_with("hello")
    executor._do_anthropic_native.assert_not_awaited()
    assert result == "openai-result"


@pytest.mark.asyncio
async def test_dispatch_anthropic_scheme_calls_anthropic_native():
    """auth_scheme='anthropic' → _do_anthropic_native runs, _do_openai_compat does not."""
    executor = _make_executor("anthropic")
    executor._do_openai_compat = AsyncMock(return_value="should-not-run")
    executor._do_anthropic_native = AsyncMock(return_value="anthropic-result")

    result = await executor._do_inference("hello")

    executor._do_anthropic_native.assert_awaited_once_with("hello")
    executor._do_openai_compat.assert_not_awaited()
    assert result == "anthropic-result"


@pytest.mark.asyncio
async def test_dispatch_unknown_scheme_falls_back_to_openai_compat():
    """Unknown auth_scheme → log a warning + fall back to openai-compat (forward-compat)."""
    executor = _make_executor("openai")
    # Mutate the cfg field to simulate an unknown scheme (testing the dispatch, not the registry)
    executor.provider_cfg = providers.ProviderConfig(
        name="futureprovider",
        env_vars=("FOO",),
        base_url="https://example.com/v1",
        default_model="foo",
        auth_scheme="some_future_scheme",
    )
    executor._do_openai_compat = AsyncMock(return_value="fallback-result")
    executor._do_anthropic_native = AsyncMock()

    result = await executor._do_inference("hello")

    executor._do_openai_compat.assert_awaited_once()
    executor._do_anthropic_native.assert_not_awaited()
    assert result == "fallback-result"


@pytest.mark.asyncio
async def test_anthropic_native_raises_clear_error_when_sdk_missing(monkeypatch):
    """If the anthropic package is not installed, _do_anthropic_native raises
    a clear RuntimeError with install instructions — it does NOT silently
    fall back to the OpenAI-compat shim (which would lose tool-calling +
    vision fidelity invisibly).
    """
    executor = _make_executor("anthropic")

    # Simulate ImportError on `import anthropic`. We do this by clobbering
    # the name in sys.modules so the import statement inside
    # _do_anthropic_native hits an ImportError.
    monkeypatch.setitem(sys.modules, "anthropic", None)

    with pytest.raises(RuntimeError, match="anthropic"):
        await executor._do_anthropic_native("hello")


def test_create_executor_passes_provider_cfg():
    """create_executor's back-compat paths should set .provider_cfg on the
    returned executor so dispatch has auth_scheme available at runtime."""
    # Direct-load executor module same way _make_executor does
    import importlib.util
    src = (_HERMES_DIR / "executor.py").read_text().replace(
        "from .providers import", "from providers import"
    )
    ns: dict = {}
    exec(compile(src, str(_HERMES_DIR / "executor.py"), "exec"), ns)
    create_executor = ns["create_executor"]

    # Path 1: hermes_api_key back-compat → nous_portal cfg
    exec1 = create_executor(hermes_api_key="test-key")
    assert exec1.provider_cfg.name == "nous_portal"
    assert exec1.provider_cfg.auth_scheme == "openai"

    # Path 2: explicit provider name → that cfg (anthropic has the new scheme)
    import os
    os.environ["ANTHROPIC_API_KEY"] = "ant-test"
    try:
        exec2 = create_executor(provider="anthropic")
        assert exec2.provider_cfg.name == "anthropic"
        assert exec2.provider_cfg.auth_scheme == "anthropic"
        assert exec2.model == "claude-sonnet-4-5"
    finally:
        os.environ.pop("ANTHROPIC_API_KEY", None)
