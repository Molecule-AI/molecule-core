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
    """Phase 2a: anthropic's auth_scheme is 'anthropic'."""
    cfg = providers.PROVIDERS["anthropic"]
    assert cfg.auth_scheme == "anthropic"


def test_gemini_entry_has_gemini_scheme():
    """Phase 2b: gemini's auth_scheme is 'gemini'."""
    cfg = providers.PROVIDERS["gemini"]
    assert cfg.auth_scheme == "gemini"
    # Base URL no longer has the /v1beta/openai suffix — native SDK uses bare host.
    assert "/openai" not in cfg.base_url
    assert cfg.base_url.startswith("https://generativelanguage.googleapis.com")


def test_all_other_providers_still_openai_scheme():
    """Phase 2 changes only anthropic + gemini. Every other provider keeps auth_scheme='openai'."""
    native_providers = {"anthropic", "gemini"}
    for name, cfg in providers.PROVIDERS.items():
        if name in native_providers:
            continue
        assert cfg.auth_scheme == "openai", (
            f"{name} unexpectedly has auth_scheme={cfg.auth_scheme!r}"
        )


@pytest.mark.asyncio
async def test_dispatch_openai_scheme_calls_openai_compat():
    """auth_scheme='openai' → _do_openai_compat runs, native paths do not."""
    executor = _make_executor("openai")
    executor._do_openai_compat = AsyncMock(return_value="openai-result")
    executor._do_anthropic_native = AsyncMock(return_value="should-not-run")
    executor._do_gemini_native = AsyncMock(return_value="should-not-run")

    result = await executor._do_inference("hello")

    # Phase 2c: _do_inference passes (user_message, history) to the path;
    # when no history supplied, second arg is None.
    executor._do_openai_compat.assert_awaited_once_with("hello", None)
    executor._do_anthropic_native.assert_not_awaited()
    executor._do_gemini_native.assert_not_awaited()
    assert result == "openai-result"


@pytest.mark.asyncio
async def test_dispatch_anthropic_scheme_calls_anthropic_native():
    """auth_scheme='anthropic' → _do_anthropic_native runs, others do not."""
    executor = _make_executor("anthropic")
    executor._do_openai_compat = AsyncMock(return_value="should-not-run")
    executor._do_anthropic_native = AsyncMock(return_value="anthropic-result")
    executor._do_gemini_native = AsyncMock(return_value="should-not-run")

    result = await executor._do_inference("hello")

    executor._do_anthropic_native.assert_awaited_once_with("hello", None)
    executor._do_openai_compat.assert_not_awaited()
    executor._do_gemini_native.assert_not_awaited()
    assert result == "anthropic-result"


@pytest.mark.asyncio
async def test_dispatch_gemini_scheme_calls_gemini_native():
    """auth_scheme='gemini' → _do_gemini_native runs, others do not. Phase 2b."""
    executor = _make_executor("gemini")
    executor._do_openai_compat = AsyncMock(return_value="should-not-run")
    executor._do_anthropic_native = AsyncMock(return_value="should-not-run")
    executor._do_gemini_native = AsyncMock(return_value="gemini-result")

    result = await executor._do_inference("hello")

    executor._do_gemini_native.assert_awaited_once_with("hello", None)
    executor._do_openai_compat.assert_not_awaited()
    executor._do_anthropic_native.assert_not_awaited()
    assert result == "gemini-result"


# ---------------------------------------------------------------------------
# Phase 2c — history-to-message conversion tests
# ---------------------------------------------------------------------------


def test_history_to_openai_messages_empty_history():
    """No history → single user message (back-compat with pre-2c single-turn shape)."""
    import importlib.util
    src = (_HERMES_DIR / "executor.py").read_text().replace(
        "from .providers import", "from providers import"
    )
    ns: dict = {}
    exec(compile(src, str(_HERMES_DIR / "executor.py"), "exec"), ns)
    HermesA2AExecutor = ns["HermesA2AExecutor"]

    msgs = HermesA2AExecutor._history_to_openai_messages("current turn", [])
    assert msgs == [{"role": "user", "content": "current turn"}]


def test_history_to_openai_messages_multi_turn():
    """A2A history roles map: human→user, ai→assistant. Current turn appended as user."""
    import importlib.util
    src = (_HERMES_DIR / "executor.py").read_text().replace(
        "from .providers import", "from providers import"
    )
    ns: dict = {}
    exec(compile(src, str(_HERMES_DIR / "executor.py"), "exec"), ns)
    HermesA2AExecutor = ns["HermesA2AExecutor"]

    history = [("human", "first question"), ("ai", "first answer"), ("human", "follow-up")]
    msgs = HermesA2AExecutor._history_to_openai_messages("current turn", history)
    assert msgs == [
        {"role": "user", "content": "first question"},
        {"role": "assistant", "content": "first answer"},
        {"role": "user", "content": "follow-up"},
        {"role": "user", "content": "current turn"},
    ]


def test_history_to_anthropic_messages_same_as_openai():
    """Anthropic Messages API uses the same wire shape as OpenAI for text-only turns."""
    import importlib.util
    src = (_HERMES_DIR / "executor.py").read_text().replace(
        "from .providers import", "from providers import"
    )
    ns: dict = {}
    exec(compile(src, str(_HERMES_DIR / "executor.py"), "exec"), ns)
    HermesA2AExecutor = ns["HermesA2AExecutor"]

    history = [("human", "hello"), ("ai", "hi")]
    openai_msgs = HermesA2AExecutor._history_to_openai_messages("how are you?", history)
    anth_msgs = HermesA2AExecutor._history_to_anthropic_messages("how are you?", history)
    assert openai_msgs == anth_msgs


def test_history_to_gemini_contents_uses_model_role_and_parts_wrapper():
    """Gemini uses role='user'|'model' (NOT 'assistant') and wraps text in parts=[{text}]."""
    import importlib.util
    src = (_HERMES_DIR / "executor.py").read_text().replace(
        "from .providers import", "from providers import"
    )
    ns: dict = {}
    exec(compile(src, str(_HERMES_DIR / "executor.py"), "exec"), ns)
    HermesA2AExecutor = ns["HermesA2AExecutor"]

    history = [("human", "hi"), ("ai", "hello back")]
    contents = HermesA2AExecutor._history_to_gemini_contents("follow-up?", history)
    assert contents == [
        {"role": "user", "parts": [{"text": "hi"}]},
        {"role": "model", "parts": [{"text": "hello back"}]},
        {"role": "user", "parts": [{"text": "follow-up?"}]},
    ]


@pytest.mark.asyncio
async def test_dispatch_passes_history_through():
    """When _do_inference is called with history, it flows through to the provider path."""
    executor = _make_executor("anthropic")
    executor._do_anthropic_native = AsyncMock(return_value="reply-with-history")
    executor._do_openai_compat = AsyncMock()
    executor._do_gemini_native = AsyncMock()

    history = [("human", "prior q"), ("ai", "prior a")]
    result = await executor._do_inference("current", history)

    executor._do_anthropic_native.assert_awaited_once_with("current", history)
    assert result == "reply-with-history"


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
    executor._do_gemini_native = AsyncMock()

    result = await executor._do_inference("hello")

    executor._do_openai_compat.assert_awaited_once()
    executor._do_anthropic_native.assert_not_awaited()
    executor._do_gemini_native.assert_not_awaited()
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


@pytest.mark.asyncio
async def test_gemini_native_raises_clear_error_when_sdk_missing(monkeypatch):
    """If the google-genai package is not installed, _do_gemini_native raises
    a clear RuntimeError with install instructions — same fail-loud semantics
    as the anthropic native path."""
    executor = _make_executor("gemini")

    # Simulate ImportError on `from google import genai`. Clobbering
    # sys.modules["google"] forces the submodule import to fail.
    monkeypatch.setitem(sys.modules, "google", None)

    with pytest.raises(RuntimeError, match="google-genai"):
        await executor._do_gemini_native("hello")


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

    # Path 3: Phase 2b — gemini explicit resolution
    os.environ["GEMINI_API_KEY"] = "gem-test"
    try:
        exec3 = create_executor(provider="gemini")
        assert exec3.provider_cfg.name == "gemini"
        assert exec3.provider_cfg.auth_scheme == "gemini"
        assert exec3.model == "gemini-2.5-flash"
    finally:
        os.environ.pop("GEMINI_API_KEY", None)
