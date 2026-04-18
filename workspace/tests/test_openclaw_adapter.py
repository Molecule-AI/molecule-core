"""Unit tests for OpenClaw adapter env-var key selection and provider URL routing.

The key-selection and URL-routing logic lives inline in OpenClawAdapter.setup()
(adapter.py lines 84-92).  Since setup() carries heavy subprocess dependencies,
these tests isolate the selection logic by reproducing the exact Python expressions
from the adapter source — if the adapter's logic changes, these tests must be kept
in sync.

Organisation:
  TestEnvKeyChain          — priority order of the 3 currently supported keys
  TestProviderUrlMapping   — model-prefix → provider URL dict correctness
  TestNegativeAndFallback  — no keys set / unsupported keys
  xfail stubs              — AISTUDIO + QIANFAN documented as not-yet-implemented
"""
from __future__ import annotations

import os
from unittest.mock import patch

import pytest


# ---------------------------------------------------------------------------
# Helpers — mirror the exact expressions from adapter.py lines 84-92.
# Must be kept in sync with the adapter source.
# ---------------------------------------------------------------------------

def _select_key(env: dict) -> str:
    """Mirror line 84: nested os.environ.get priority chain."""
    return env.get("OPENAI_API_KEY",
                   env.get("GROQ_API_KEY",
                           env.get("OPENROUTER_API_KEY", "")))


_PROVIDER_URLS: dict[str, str] = {
    "openai":     "https://api.openai.com/v1",
    "groq":       "https://api.groq.com/openai/v1",
    "openrouter": "https://openrouter.ai/api/v1",
}


def _select_url(model: str, runtime_config: dict | None = None) -> str:
    """Mirror lines 86-92: model-prefix → provider URL with optional override."""
    prefix = model.split(":")[0] if ":" in model else "openai"
    return (runtime_config or {}).get(
        "provider_url",
        _PROVIDER_URLS.get(prefix, "https://api.openai.com/v1"),
    )


# ---------------------------------------------------------------------------
# 1. Env-var key priority chain (3 keys currently in adapter.py)
# ---------------------------------------------------------------------------

class TestEnvKeyChain:

    def test_openai_key_selected(self):
        with patch.dict(os.environ, {"OPENAI_API_KEY": "sk-openai-test"}, clear=True):
            assert _select_key(os.environ) == "sk-openai-test"

    def test_groq_key_selected_when_openai_absent(self):
        with patch.dict(os.environ, {"GROQ_API_KEY": "sk-groq-test"}, clear=True):
            assert _select_key(os.environ) == "sk-groq-test"

    def test_openrouter_key_selected_when_openai_and_groq_absent(self):
        with patch.dict(os.environ, {"OPENROUTER_API_KEY": "sk-or-test"}, clear=True):
            assert _select_key(os.environ) == "sk-or-test"

    def test_openai_beats_groq_when_both_set(self):
        with patch.dict(os.environ, {"OPENAI_API_KEY": "openai", "GROQ_API_KEY": "groq"}, clear=True):
            assert _select_key(os.environ) == "openai"

    def test_groq_beats_openrouter_when_openai_absent(self):
        with patch.dict(os.environ, {"GROQ_API_KEY": "groq", "OPENROUTER_API_KEY": "or"}, clear=True):
            assert _select_key(os.environ) == "groq"


# ---------------------------------------------------------------------------
# 2. Model-prefix → provider URL routing
# ---------------------------------------------------------------------------

class TestProviderUrlMapping:

    def test_openai_prefix_routes_to_openai(self):
        assert _select_url("openai:gpt-4o") == "https://api.openai.com/v1"

    def test_groq_prefix_routes_to_groq(self):
        assert _select_url("groq:llama3-70b") == "https://api.groq.com/openai/v1"

    def test_openrouter_prefix_routes_to_openrouter(self):
        assert _select_url("openrouter:meta-llama/llama-3.3-70b") == "https://openrouter.ai/api/v1"

    def test_runtime_config_override_wins_over_prefix(self):
        url = _select_url("openai:gpt-4o", {"provider_url": "https://custom.example.com/v1"})
        assert url == "https://custom.example.com/v1"

    def test_unknown_prefix_falls_back_to_openai(self):
        assert _select_url("some-unknown-model") == "https://api.openai.com/v1"


# ---------------------------------------------------------------------------
# 3. Negative / fallback cases
# ---------------------------------------------------------------------------

class TestNegativeAndFallback:

    def test_no_keys_returns_empty_string(self):
        with patch.dict(os.environ, {}, clear=True):
            assert _select_key(os.environ) == ""

    def test_unsupported_aistudio_key_returns_empty(self):
        """Documents that AISTUDIO_API_KEY is NOT yet in the adapter's key chain."""
        with patch.dict(os.environ, {"AISTUDIO_API_KEY": "sk-ai"}, clear=True):
            assert _select_key(os.environ) == ""

    def test_unsupported_qianfan_key_returns_empty(self):
        """Documents that QIANFAN_API_KEY is NOT yet in the adapter's key chain."""
        with patch.dict(os.environ, {"QIANFAN_API_KEY": "sk-qf"}, clear=True):
            assert _select_key(os.environ) == ""


# ---------------------------------------------------------------------------
# 4. AISTUDIO + QIANFAN — xfail stubs (not yet implemented in adapter.py)
#    These fail now; they should be promoted to passing tests once the adapter
#    adds AISTUDIO_API_KEY and QIANFAN_API_KEY to its key chain and provider_urls.
# ---------------------------------------------------------------------------

@pytest.mark.xfail(
    strict=True,
    reason=(
        "AISTUDIO_API_KEY not yet in openclaw adapter env-var chain — "
        "add to adapter.py line 84 and provider_urls dict with "
        "URL https://generativelanguage.googleapis.com/v1beta/openai"
    ),
)
def test_aistudio_key_routes_to_aistudio_url():
    with patch.dict(os.environ, {"AISTUDIO_API_KEY": "sk-ai-test"}, clear=True):
        assert _select_key(os.environ) == "sk-ai-test"
    assert _select_url("gemini-2.5-flash") == "https://generativelanguage.googleapis.com/v1beta/openai"


@pytest.mark.xfail(
    strict=True,
    reason=(
        "QIANFAN_API_KEY not yet in openclaw adapter env-var chain — "
        "add to adapter.py line 84 and provider_urls dict with "
        "URL https://qianfan.baidubce.com/v2"
    ),
)
def test_qianfan_key_routes_to_qianfan_url():
    with patch.dict(os.environ, {"QIANFAN_API_KEY": "sk-qf-test"}, clear=True):
        assert _select_key(os.environ) == "sk-qf-test"
    assert _select_url("ernie-4.5") == "https://qianfan.baidubce.com/v2"
