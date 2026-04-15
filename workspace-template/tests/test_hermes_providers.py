"""Tests for workspace-template/adapters/hermes/providers.py.

These tests exercise resolve_provider() in isolation — they do not import
anything from adapters/__init__.py so they don't need the a2a runtime deps.
"""

from __future__ import annotations

import importlib
import os
import sys
from pathlib import Path

import pytest

# Make the hermes package importable without pulling in adapters/__init__.py
# (which imports the a2a SDK). We load providers.py directly from its file path.
_HERMES_DIR = Path(__file__).parent.parent / "adapters" / "hermes"
sys.path.insert(0, str(_HERMES_DIR))
import providers  # type: ignore  # noqa: E402


_ALL_PROVIDER_ENV_VARS = (
    "HERMES_API_KEY",
    "NOUS_API_KEY",
    "OPENROUTER_API_KEY",
    "OPENAI_API_KEY",
    "ANTHROPIC_API_KEY",
    "XAI_API_KEY",
    "GROK_API_KEY",
    "GEMINI_API_KEY",
    "GOOGLE_API_KEY",
    "QWEN_API_KEY",
    "DASHSCOPE_API_KEY",
    "GLM_API_KEY",
    "ZHIPU_API_KEY",
    "KIMI_API_KEY",
    "MOONSHOT_API_KEY",
    "MINIMAX_API_KEY",
    "DEEPSEEK_API_KEY",
    "GROQ_API_KEY",
    "TOGETHER_API_KEY",
    "FIREWORKS_API_KEY",
    "MISTRAL_API_KEY",
)


@pytest.fixture(autouse=True)
def _clean_env():
    """Clear every provider env var before each test and restore to the
    exact pre-test state on teardown.

    Implementation note: earlier version used pytest's monkeypatch fixture,
    which tracks deltas from the state at fixture entry. That was buggy
    because several tests in this file mutate os.environ directly
    (os.environ["HERMES_API_KEY"] = ...), bypassing monkeypatch's
    tracking. The direct mutations leaked into the NEXT test file
    (test_hermes_smoke.py::test_create_executor_raises_without_keys),
    causing a file-order-dependent failure. Pure snapshot/restore
    avoids all the delta-tracking edge cases.
    """
    saved = {k: os.environ.get(k) for k in _ALL_PROVIDER_ENV_VARS}
    for k in _ALL_PROVIDER_ENV_VARS:
        os.environ.pop(k, None)
    try:
        yield
    finally:
        for k, v in saved.items():
            if v is None:
                os.environ.pop(k, None)
            else:
                os.environ[k] = v


def test_registry_is_populated():
    """Phase 1 ships at least 15 providers and every entry is self-consistent."""
    assert len(providers.PROVIDERS) >= 15
    assert len(providers.RESOLUTION_ORDER) == len(providers.PROVIDERS)
    for name, cfg in providers.PROVIDERS.items():
        assert cfg.name == name, f"{name}: config.name should match dict key"
        assert cfg.env_vars, f"{name}: must declare at least one env var"
        assert cfg.base_url.startswith("http"), f"{name}: base_url must be http(s)"
        assert cfg.default_model, f"{name}: must declare a default model"
        assert name in providers.RESOLUTION_ORDER, f"{name}: missing from resolution order"


def test_resolution_order_has_no_duplicates():
    assert len(providers.RESOLUTION_ORDER) == len(set(providers.RESOLUTION_ORDER))


def test_backcompat_hermes_api_key_first():
    """PR 2 back-compat — HERMES_API_KEY auto-detect still routes to Nous Portal."""
    os.environ["HERMES_API_KEY"] = "hermes-test-key"
    cfg, key = providers.resolve_provider()
    assert cfg.name == "nous_portal"
    assert key == "hermes-test-key"


def test_backcompat_openrouter_api_key_second():
    """PR 2 back-compat — OPENROUTER_API_KEY still routes to OpenRouter when HERMES_API_KEY is absent."""
    os.environ["OPENROUTER_API_KEY"] = "or-test-key"
    cfg, key = providers.resolve_provider()
    assert cfg.name == "openrouter"


def test_auto_detect_openai():
    os.environ["OPENAI_API_KEY"] = "sk-test"
    cfg, key = providers.resolve_provider()
    assert cfg.name == "openai"
    assert cfg.base_url == "https://api.openai.com/v1"


def test_auto_detect_anthropic():
    os.environ["ANTHROPIC_API_KEY"] = "ant-test"
    cfg, key = providers.resolve_provider()
    assert cfg.name == "anthropic"


@pytest.mark.parametrize(
    "env_var,expected",
    [
        ("XAI_API_KEY", "xai"),
        ("GROK_API_KEY", "xai"),
        ("QWEN_API_KEY", "qwen"),
        ("DASHSCOPE_API_KEY", "qwen"),
        ("GLM_API_KEY", "glm"),
        ("ZHIPU_API_KEY", "glm"),
        ("KIMI_API_KEY", "kimi"),
        ("MOONSHOT_API_KEY", "kimi"),
        ("GROQ_API_KEY", "groq"),
        ("DEEPSEEK_API_KEY", "deepseek"),
        ("MISTRAL_API_KEY", "mistral"),
        ("TOGETHER_API_KEY", "together"),
        ("FIREWORKS_API_KEY", "fireworks"),
        ("MINIMAX_API_KEY", "minimax"),
        ("GEMINI_API_KEY", "gemini"),
        ("GOOGLE_API_KEY", "gemini"),
    ],
)
def test_every_provider_env_var_resolves(env_var, expected):
    """Every env var listed in PROVIDERS resolves to the right provider
    — this guards against typos in the registry dict."""
    os.environ[env_var] = "test-key"
    cfg, _ = providers.resolve_provider()
    assert cfg.name == expected, (
        f"{env_var} should route to {expected}, got {cfg.name}"
    )


def test_explicit_provider_wins_over_auto_detect():
    """When `provider=` is given, auto-detect is bypassed."""
    os.environ["HERMES_API_KEY"] = "hermes-key"  # would auto-detect
    os.environ["OPENAI_API_KEY"] = "openai-key"
    cfg, key = providers.resolve_provider("openai")
    assert cfg.name == "openai"
    assert key == "openai-key"


def test_unknown_provider_raises():
    with pytest.raises(ValueError, match="Unknown Hermes provider"):
        providers.resolve_provider("this_provider_does_not_exist")


def test_explicit_provider_with_missing_env_raises():
    """If the operator asks for a specific provider but its env var is empty,
    we raise — we do NOT fall back to auto-detect because that would be
    surprising ("why is my openai config talking to anthropic?")."""
    os.environ["HERMES_API_KEY"] = "some-value"  # auto-detect would succeed
    with pytest.raises(ValueError, match="no env var set"):
        providers.resolve_provider("anthropic")


def test_auto_detect_with_no_env_lists_all_options():
    """The error message should list every env var the caller could set,
    so operators don't have to read the source."""
    # No env vars set (autouse fixture clears them all)
    with pytest.raises(ValueError) as exc_info:
        providers.resolve_provider()
    msg = str(exc_info.value)
    # Spot-check: the message names at least a few providers
    for env_var in ("OPENAI_API_KEY", "ANTHROPIC_API_KEY", "QWEN_API_KEY"):
        assert env_var in msg, f"error message should mention {env_var}"
