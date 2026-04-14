"""Smoke tests for adapters.hermes.create_executor().

Verifies key resolution order and ValueError on missing keys.
No real network calls are made — the executor object is just instantiated.
"""
import os
import pytest
from unittest.mock import patch

from adapters.hermes import create_executor


def test_create_executor_with_param():
    """create_executor() works when key passed directly as param."""
    executor = create_executor(hermes_api_key="test-key-direct")
    assert executor is not None


def test_create_executor_with_hermes_env():
    """create_executor() works when HERMES_API_KEY env var is set."""
    with patch.dict(os.environ, {"HERMES_API_KEY": "test-hermes-key"}, clear=False):
        os.environ.pop("OPENROUTER_API_KEY", None)
        executor = create_executor()
        assert executor is not None


def test_create_executor_falls_back_to_openrouter():
    """create_executor() falls back to OPENROUTER_API_KEY when HERMES_API_KEY absent."""
    env = {"OPENROUTER_API_KEY": "test-openrouter-key"}
    with patch.dict(os.environ, env, clear=False):
        os.environ.pop("HERMES_API_KEY", None)
        executor = create_executor()
        assert executor is not None


def test_create_executor_raises_without_keys():
    """create_executor() raises ValueError when no keys available."""
    with patch.dict(os.environ, {}, clear=False):
        os.environ.pop("HERMES_API_KEY", None)
        os.environ.pop("OPENROUTER_API_KEY", None)
        with pytest.raises(ValueError):
            create_executor()


# ---------------------------------------------------------------------------
# Additional assertions — verify key routing is correct
# ---------------------------------------------------------------------------

def test_param_key_uses_nous_base_url():
    """When called with explicit key, base_url points at Nous Portal."""
    executor = create_executor(hermes_api_key="nous-key")
    assert "nousresearch.com" in executor.base_url


def test_hermes_env_uses_nous_base_url():
    """HERMES_API_KEY maps to Nous Portal base URL."""
    with patch.dict(os.environ, {"HERMES_API_KEY": "nous-key"}, clear=False):
        os.environ.pop("OPENROUTER_API_KEY", None)
        executor = create_executor()
    assert "nousresearch.com" in executor.base_url


def test_openrouter_fallback_uses_openrouter_base_url():
    """OPENROUTER_API_KEY fallback maps to OpenRouter base URL."""
    with patch.dict(os.environ, {"OPENROUTER_API_KEY": "or-key"}, clear=False):
        os.environ.pop("HERMES_API_KEY", None)
        executor = create_executor()
    assert "openrouter.ai" in executor.base_url


def test_param_takes_priority_over_hermes_env():
    """Explicit param overrides HERMES_API_KEY env var."""
    with patch.dict(os.environ, {"HERMES_API_KEY": "env-key"}, clear=False):
        executor = create_executor(hermes_api_key="param-key")
    assert executor.api_key == "param-key"


def test_hermes_env_takes_priority_over_openrouter():
    """HERMES_API_KEY overrides OPENROUTER_API_KEY fallback."""
    env = {"HERMES_API_KEY": "hermes-key", "OPENROUTER_API_KEY": "or-key"}
    with patch.dict(os.environ, env, clear=False):
        executor = create_executor()
    assert executor.api_key == "hermes-key"
    assert "nousresearch.com" in executor.base_url
