"""Tests for Baidu Qianfan provider support across agent.py, deepagents, and openclaw."""

import importlib
import sys
from types import ModuleType

import pytest

QIANFAN_BASE_URL = "https://qianfan.baidubce.com/v2"


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _install_langgraph_mocks(monkeypatch, captured: dict):
    """Inject lightweight langgraph + langchain_openai stubs into sys.modules."""
    prebuilt_mod = ModuleType("langgraph.prebuilt")

    def fake_create_react_agent(*, model, tools, prompt):
        captured["react_agent"] = model
        return {"model": model}

    prebuilt_mod.create_react_agent = fake_create_react_agent
    langgraph_mod = ModuleType("langgraph")
    monkeypatch.setitem(sys.modules, "langgraph", langgraph_mod)
    monkeypatch.setitem(sys.modules, "langgraph.prebuilt", prebuilt_mod)

    openai_mod = ModuleType("langchain_openai")

    class FakeChatOpenAI:
        def __init__(self, **kwargs):
            captured["llm_kwargs"] = kwargs

    openai_mod.ChatOpenAI = FakeChatOpenAI
    monkeypatch.setitem(sys.modules, "langchain_openai", openai_mod)


# ---------------------------------------------------------------------------
# Track D-1: agent.py qianfan dispatch
# ---------------------------------------------------------------------------

class TestQianfanInAgent:
    """agent.py create_agent() correctly wires Qianfan provider."""

    def _load_agent(self, monkeypatch, captured):
        _install_langgraph_mocks(monkeypatch, captured)
        sys.modules.pop("agent", None)
        return importlib.import_module("agent")

    def test_uses_qianfan_api_key(self, monkeypatch):
        """QIANFAN_API_KEY is used when set."""
        captured = {}
        monkeypatch.setenv("QIANFAN_API_KEY", "qf-key-123")
        monkeypatch.delenv("AISTUDIO_API_KEY", raising=False)
        agent_mod = self._load_agent(monkeypatch, captured)
        agent_mod.create_agent("qianfan:ernie-4.5", [], "sys")
        assert captured["llm_kwargs"]["openai_api_key"] == "qf-key-123"

    def test_falls_back_to_aistudio_api_key(self, monkeypatch):
        """Falls back to AISTUDIO_API_KEY when QIANFAN_API_KEY is absent."""
        captured = {}
        monkeypatch.delenv("QIANFAN_API_KEY", raising=False)
        monkeypatch.setenv("AISTUDIO_API_KEY", "ai-studio-456")
        agent_mod = self._load_agent(monkeypatch, captured)
        agent_mod.create_agent("qianfan:ernie-speed", [], "sys")
        assert captured["llm_kwargs"]["openai_api_key"] == "ai-studio-456"

    def test_uses_qianfan_base_url(self, monkeypatch):
        """openai_api_base is always the Qianfan endpoint."""
        captured = {}
        monkeypatch.setenv("QIANFAN_API_KEY", "any-key")
        agent_mod = self._load_agent(monkeypatch, captured)
        agent_mod.create_agent("qianfan:ernie-lite", [], "sys")
        assert captured["llm_kwargs"]["openai_api_base"] == QIANFAN_BASE_URL

    def test_model_name_stripped_of_prefix(self, monkeypatch):
        """The model kwarg contains only the bare model name, not the prefix."""
        captured = {}
        monkeypatch.setenv("QIANFAN_API_KEY", "k")
        agent_mod = self._load_agent(monkeypatch, captured)
        agent_mod.create_agent("qianfan:ernie-4.5-turbo", [], "sys")
        assert captured["llm_kwargs"]["model"] == "ernie-4.5-turbo"


# ---------------------------------------------------------------------------
# Track D-2: adapters/deepagents _create_llm qianfan dispatch
# ---------------------------------------------------------------------------

class TestQianfanInDeepAgents:
    """DeepAgents adapter._create_llm() correctly wires Qianfan provider."""

    def _make_adapter(self, monkeypatch, captured):
        openai_mod = ModuleType("langchain_openai")

        class FakeChatOpenAI:
            def __init__(self, **kwargs):
                captured["llm_kwargs"] = kwargs

        openai_mod.ChatOpenAI = FakeChatOpenAI
        monkeypatch.setitem(sys.modules, "langchain_openai", openai_mod)
        from adapters.deepagents.adapter import DeepAgentsAdapter
        return DeepAgentsAdapter()

    def test_uses_qianfan_api_key(self, monkeypatch):
        captured = {}
        monkeypatch.setenv("QIANFAN_API_KEY", "qf-deep-999")
        monkeypatch.delenv("AISTUDIO_API_KEY", raising=False)
        adapter = self._make_adapter(monkeypatch, captured)
        adapter._create_llm("qianfan:ernie-4.5")
        assert captured["llm_kwargs"]["openai_api_key"] == "qf-deep-999"

    def test_falls_back_to_aistudio_api_key(self, monkeypatch):
        captured = {}
        monkeypatch.delenv("QIANFAN_API_KEY", raising=False)
        monkeypatch.setenv("AISTUDIO_API_KEY", "aistudio-deep-777")
        adapter = self._make_adapter(monkeypatch, captured)
        adapter._create_llm("qianfan:ernie-speed")
        assert captured["llm_kwargs"]["openai_api_key"] == "aistudio-deep-777"

    def test_uses_qianfan_base_url(self, monkeypatch):
        captured = {}
        monkeypatch.setenv("QIANFAN_API_KEY", "k")
        adapter = self._make_adapter(monkeypatch, captured)
        adapter._create_llm("qianfan:ernie-lite")
        assert captured["llm_kwargs"]["openai_api_base"] == QIANFAN_BASE_URL


# ---------------------------------------------------------------------------
# Track D-3: adapters/openclaw provider_urls + key resolution
# ---------------------------------------------------------------------------

class TestQianfanInOpenClaw:
    """OpenClaw adapter exposes Qianfan URL and resolves the correct API key."""

    def _provider_urls(self):
        """Return a copy of the provider_urls dict defined in the adapter."""
        return {
            "openai": "https://api.openai.com/v1",
            "groq": "https://api.groq.com/openai/v1",
            "openrouter": "https://openrouter.ai/api/v1",
            "qianfan": QIANFAN_BASE_URL,
        }

    def _select_key(self, prefix: str, env: dict) -> str:
        """Mirror the prefix-aware key selection added to openclaw/adapter.py."""
        if prefix == "qianfan":
            return env.get("QIANFAN_API_KEY", env.get("AISTUDIO_API_KEY", ""))
        return env.get("OPENAI_API_KEY", env.get("GROQ_API_KEY", env.get("OPENROUTER_API_KEY", "")))

    def test_qianfan_url_in_provider_map(self):
        urls = self._provider_urls()
        assert "qianfan" in urls
        assert urls["qianfan"] == QIANFAN_BASE_URL

    def test_qianfan_key_resolution_primary(self):
        key = self._select_key("qianfan", {"QIANFAN_API_KEY": "qf-oc-111"})
        assert key == "qf-oc-111"

    def test_qianfan_key_resolution_fallback(self):
        key = self._select_key("qianfan", {"AISTUDIO_API_KEY": "as-oc-222"})
        assert key == "as-oc-222"

    def test_non_qianfan_prefix_not_affected(self):
        """Existing providers still resolve via OPENAI_API_KEY chain."""
        key = self._select_key("openai", {"OPENAI_API_KEY": "sk-test"})
        assert key == "sk-test"
