"""Tests for agent.py — LangGraph agent factory.

Uses importlib.util.spec_from_file_location to load the real module, bypassing
any conftest mocks that might interfere.
"""

import importlib.util
import sys
from pathlib import Path
from types import ModuleType
from unittest.mock import MagicMock, patch

import pytest

ROOT = Path(__file__).resolve().parents[1]


def _load_agent(monkeypatch, extra_sys_modules=None):
    """Load the real agent.py in isolation."""
    spec = importlib.util.spec_from_file_location(
        "_test_agent",
        ROOT / "agent.py",
    )
    mod = importlib.util.module_from_spec(spec)
    # Patch langgraph before exec
    fake_langgraph = ModuleType("langgraph")
    fake_prebuilt = ModuleType("langgraph.prebuilt")
    fake_create = MagicMock(return_value=MagicMock(name="agent_instance"))
    fake_prebuilt.create_react_agent = fake_create
    fake_langgraph.prebuilt = fake_prebuilt

    monkeypatch.setitem(sys.modules, "langgraph", fake_langgraph)
    monkeypatch.setitem(sys.modules, "langgraph.prebuilt", fake_prebuilt)

    if extra_sys_modules:
        for k, v in extra_sys_modules.items():
            monkeypatch.setitem(sys.modules, k, v)

    spec.loader.exec_module(mod)
    # Attach the create_react_agent mock to module for inspection
    mod._fake_create_react_agent = fake_create
    return mod


# ---------------------------------------------------------------------------
# create_agent — provider tests
# ---------------------------------------------------------------------------

class TestCreateAgent:

    def test_anthropic_provider(self, monkeypatch):
        """anthropic: prefix uses ChatAnthropic."""
        fake_llm_cls = MagicMock(return_value=MagicMock(name="llm"))
        fake_lc_anthropic = ModuleType("langchain_anthropic")
        fake_lc_anthropic.ChatAnthropic = fake_llm_cls

        mod = _load_agent(monkeypatch, {"langchain_anthropic": fake_lc_anthropic})

        monkeypatch.delenv("ANTHROPIC_BASE_URL", raising=False)
        monkeypatch.delenv("LANGFUSE_HOST", raising=False)
        monkeypatch.delenv("LANGFUSE_PUBLIC_KEY", raising=False)
        monkeypatch.delenv("LANGFUSE_SECRET_KEY", raising=False)

        agent = mod.create_agent("anthropic:claude-test", [], "sys prompt")

        fake_llm_cls.assert_called_once_with(model="claude-test")
        mod._fake_create_react_agent.assert_called_once()
        assert agent is not None

    def test_anthropic_with_base_url(self, monkeypatch):
        """anthropic: with ANTHROPIC_BASE_URL passes anthropic_api_url."""
        fake_llm_cls = MagicMock(return_value=MagicMock(name="llm"))
        fake_lc_anthropic = ModuleType("langchain_anthropic")
        fake_lc_anthropic.ChatAnthropic = fake_llm_cls

        mod = _load_agent(monkeypatch, {"langchain_anthropic": fake_lc_anthropic})

        monkeypatch.setenv("ANTHROPIC_BASE_URL", "http://proxy.test")
        monkeypatch.delenv("LANGFUSE_HOST", raising=False)
        monkeypatch.delenv("LANGFUSE_PUBLIC_KEY", raising=False)
        monkeypatch.delenv("LANGFUSE_SECRET_KEY", raising=False)

        mod.create_agent("anthropic:claude-test", [], "sys prompt")

        fake_llm_cls.assert_called_once_with(model="claude-test", anthropic_api_url="http://proxy.test")

    def test_openai_provider(self, monkeypatch):
        """openai: prefix uses ChatOpenAI."""
        fake_llm_cls = MagicMock(return_value=MagicMock(name="llm"))
        fake_lc_openai = ModuleType("langchain_openai")
        fake_lc_openai.ChatOpenAI = fake_llm_cls

        mod = _load_agent(monkeypatch, {"langchain_openai": fake_lc_openai})

        monkeypatch.delenv("OPENAI_BASE_URL", raising=False)
        monkeypatch.delenv("LANGFUSE_HOST", raising=False)
        monkeypatch.delenv("LANGFUSE_PUBLIC_KEY", raising=False)
        monkeypatch.delenv("LANGFUSE_SECRET_KEY", raising=False)

        mod.create_agent("openai:gpt-4o", [], "sys prompt")
        fake_llm_cls.assert_called_once_with(model="gpt-4o")

    def test_openai_with_base_url(self, monkeypatch):
        """openai: with OPENAI_BASE_URL passes openai_api_base."""
        fake_llm_cls = MagicMock(return_value=MagicMock(name="llm"))
        fake_lc_openai = ModuleType("langchain_openai")
        fake_lc_openai.ChatOpenAI = fake_llm_cls

        mod = _load_agent(monkeypatch, {"langchain_openai": fake_lc_openai})

        monkeypatch.setenv("OPENAI_BASE_URL", "http://openai-proxy.test")
        monkeypatch.delenv("LANGFUSE_HOST", raising=False)
        monkeypatch.delenv("LANGFUSE_PUBLIC_KEY", raising=False)
        monkeypatch.delenv("LANGFUSE_SECRET_KEY", raising=False)

        mod.create_agent("openai:gpt-4o", [], "sys")
        fake_llm_cls.assert_called_once_with(model="gpt-4o", openai_api_base="http://openai-proxy.test")

    def test_openrouter_provider(self, monkeypatch):
        """openrouter: prefix uses ChatOpenAI with openrouter base URL."""
        fake_llm_cls = MagicMock(return_value=MagicMock(name="llm"))
        fake_lc_openai = ModuleType("langchain_openai")
        fake_lc_openai.ChatOpenAI = fake_llm_cls

        mod = _load_agent(monkeypatch, {"langchain_openai": fake_lc_openai})

        monkeypatch.setenv("OPENROUTER_API_KEY", "sk-router-test")
        monkeypatch.setenv("MAX_TOKENS", "1024")
        monkeypatch.delenv("LANGFUSE_HOST", raising=False)
        monkeypatch.delenv("LANGFUSE_PUBLIC_KEY", raising=False)
        monkeypatch.delenv("LANGFUSE_SECRET_KEY", raising=False)

        mod.create_agent("openrouter:mistral-7b", [], "sys")
        fake_llm_cls.assert_called_once_with(
            model="mistral-7b",
            openai_api_key="sk-router-test",
            openai_api_base="https://openrouter.ai/api/v1",
            max_tokens=1024,
        )

    def test_openrouter_fallback_api_key(self, monkeypatch):
        """openrouter falls back to OPENAI_API_KEY when OPENROUTER_API_KEY absent."""
        fake_llm_cls = MagicMock(return_value=MagicMock(name="llm"))
        fake_lc_openai = ModuleType("langchain_openai")
        fake_lc_openai.ChatOpenAI = fake_llm_cls

        mod = _load_agent(monkeypatch, {"langchain_openai": fake_lc_openai})

        monkeypatch.delenv("OPENROUTER_API_KEY", raising=False)
        monkeypatch.setenv("OPENAI_API_KEY", "sk-openai-fallback")
        monkeypatch.delenv("MAX_TOKENS", raising=False)
        monkeypatch.delenv("LANGFUSE_HOST", raising=False)
        monkeypatch.delenv("LANGFUSE_PUBLIC_KEY", raising=False)
        monkeypatch.delenv("LANGFUSE_SECRET_KEY", raising=False)

        mod.create_agent("openrouter:mistral-7b", [], "sys")
        call_kwargs = fake_llm_cls.call_args
        assert call_kwargs.kwargs["openai_api_key"] == "sk-openai-fallback"

    def test_groq_provider(self, monkeypatch):
        """groq: prefix uses ChatOpenAI with groq base URL."""
        fake_llm_cls = MagicMock(return_value=MagicMock(name="llm"))
        fake_lc_openai = ModuleType("langchain_openai")
        fake_lc_openai.ChatOpenAI = fake_llm_cls

        mod = _load_agent(monkeypatch, {"langchain_openai": fake_lc_openai})

        monkeypatch.setenv("GROQ_API_KEY", "gsk-test")
        monkeypatch.delenv("LANGFUSE_HOST", raising=False)
        monkeypatch.delenv("LANGFUSE_PUBLIC_KEY", raising=False)
        monkeypatch.delenv("LANGFUSE_SECRET_KEY", raising=False)

        mod.create_agent("groq:llama3-70b", [], "sys")
        fake_llm_cls.assert_called_once_with(
            model="llama3-70b",
            openai_api_key="gsk-test",
            openai_api_base="https://api.groq.com/openai/v1",
        )

    def test_no_provider_prefix_defaults_to_anthropic(self, monkeypatch):
        """model string without colon defaults to anthropic provider."""
        fake_llm_cls = MagicMock(return_value=MagicMock(name="llm"))
        fake_lc_anthropic = ModuleType("langchain_anthropic")
        fake_lc_anthropic.ChatAnthropic = fake_llm_cls

        mod = _load_agent(monkeypatch, {"langchain_anthropic": fake_lc_anthropic})

        monkeypatch.delenv("ANTHROPIC_BASE_URL", raising=False)
        monkeypatch.delenv("LANGFUSE_HOST", raising=False)
        monkeypatch.delenv("LANGFUSE_PUBLIC_KEY", raising=False)
        monkeypatch.delenv("LANGFUSE_SECRET_KEY", raising=False)

        mod.create_agent("claude-3-opus", [], "sys")
        fake_llm_cls.assert_called_once_with(model="claude-3-opus")

    def test_unsupported_provider_raises_value_error(self, monkeypatch):
        """Unknown provider raises ValueError."""
        mod = _load_agent(monkeypatch)
        monkeypatch.delenv("LANGFUSE_HOST", raising=False)
        monkeypatch.delenv("LANGFUSE_PUBLIC_KEY", raising=False)
        monkeypatch.delenv("LANGFUSE_SECRET_KEY", raising=False)

        with pytest.raises(ValueError, match="Unsupported model provider"):
            mod.create_agent("bogus:some-model", [], "sys")

    def test_google_genai_provider(self, monkeypatch):
        """google_genai: prefix uses ChatGoogleGenerativeAI."""
        fake_llm_cls = MagicMock(return_value=MagicMock(name="llm"))
        fake_lc_google = ModuleType("langchain_google_genai")
        fake_lc_google.ChatGoogleGenerativeAI = fake_llm_cls

        mod = _load_agent(monkeypatch, {"langchain_google_genai": fake_lc_google})

        monkeypatch.delenv("LANGFUSE_HOST", raising=False)
        monkeypatch.delenv("LANGFUSE_PUBLIC_KEY", raising=False)
        monkeypatch.delenv("LANGFUSE_SECRET_KEY", raising=False)

        mod.create_agent("google_genai:gemini-pro", [], "sys")
        # google_genai falls into the else: llm = LLMClass(model=model_name) branch
        fake_llm_cls.assert_called_once_with(model="gemini-pro")

    def test_ollama_provider(self, monkeypatch):
        """ollama: prefix uses ChatOllama."""
        fake_llm_cls = MagicMock(return_value=MagicMock(name="llm"))
        fake_lc_ollama = ModuleType("langchain_ollama")
        fake_lc_ollama.ChatOllama = fake_llm_cls

        mod = _load_agent(monkeypatch, {"langchain_ollama": fake_lc_ollama})

        monkeypatch.delenv("LANGFUSE_HOST", raising=False)
        monkeypatch.delenv("LANGFUSE_PUBLIC_KEY", raising=False)
        monkeypatch.delenv("LANGFUSE_SECRET_KEY", raising=False)

        mod.create_agent("ollama:llama3", [], "sys")
        fake_llm_cls.assert_called_once_with(model="llama3")

    def test_import_error_raises_import_error(self, monkeypatch):
        """ImportError from provider package is re-raised as ImportError."""
        # Remove langchain_anthropic from sys.modules so the import fails
        monkeypatch.delitem(sys.modules, "langchain_anthropic", raising=False)

        mod = _load_agent(monkeypatch)
        monkeypatch.delenv("LANGFUSE_HOST", raising=False)
        monkeypatch.delenv("LANGFUSE_PUBLIC_KEY", raising=False)
        monkeypatch.delenv("LANGFUSE_SECRET_KEY", raising=False)

        # Patch builtins.__import__ to raise for langchain_anthropic
        original_import = __builtins__.__import__ if hasattr(__builtins__, "__import__") else __import__

        def fake_import(name, *args, **kwargs):
            if name == "langchain_anthropic":
                raise ImportError("no module named langchain_anthropic")
            return original_import(name, *args, **kwargs)

        import builtins
        monkeypatch.setattr(builtins, "__import__", fake_import)

        with pytest.raises(ImportError, match="langchain-anthropic"):
            mod.create_agent("anthropic:claude-test", [], "sys")


# ---------------------------------------------------------------------------
# _setup_langfuse
# ---------------------------------------------------------------------------

class TestSetupLangfuse:

    def test_no_env_vars_returns_empty_list(self, monkeypatch):
        mod = _load_agent(monkeypatch)
        monkeypatch.delenv("LANGFUSE_HOST", raising=False)
        monkeypatch.delenv("LANGFUSE_PUBLIC_KEY", raising=False)
        monkeypatch.delenv("LANGFUSE_SECRET_KEY", raising=False)

        result = mod._setup_langfuse()
        assert result == []

    def test_partial_env_vars_returns_empty_list(self, monkeypatch):
        """Only some langfuse vars set — should return []."""
        mod = _load_agent(monkeypatch)
        monkeypatch.setenv("LANGFUSE_HOST", "http://langfuse.test")
        monkeypatch.delenv("LANGFUSE_PUBLIC_KEY", raising=False)
        monkeypatch.delenv("LANGFUSE_SECRET_KEY", raising=False)

        result = mod._setup_langfuse()
        assert result == []

    def test_all_vars_langfuse_installed(self, monkeypatch):
        """All langfuse vars present and package available returns [handler]."""
        mod = _load_agent(monkeypatch)
        monkeypatch.setenv("LANGFUSE_HOST", "http://langfuse.test")
        monkeypatch.setenv("LANGFUSE_PUBLIC_KEY", "pk-test")
        monkeypatch.setenv("LANGFUSE_SECRET_KEY", "sk-test")

        fake_handler = MagicMock(name="langfuse_handler")
        fake_callback_mod = ModuleType("langfuse.callback")
        fake_callback_mod.CallbackHandler = MagicMock(return_value=fake_handler)
        fake_langfuse = ModuleType("langfuse")
        fake_langfuse.callback = fake_callback_mod

        monkeypatch.setitem(sys.modules, "langfuse", fake_langfuse)
        monkeypatch.setitem(sys.modules, "langfuse.callback", fake_callback_mod)

        result = mod._setup_langfuse()
        assert len(result) == 1
        assert result[0] is fake_handler

    def test_langfuse_import_error_returns_empty_list(self, monkeypatch):
        """ImportError from langfuse package returns []."""
        mod = _load_agent(monkeypatch)
        monkeypatch.setenv("LANGFUSE_HOST", "http://langfuse.test")
        monkeypatch.setenv("LANGFUSE_PUBLIC_KEY", "pk-test")
        monkeypatch.setenv("LANGFUSE_SECRET_KEY", "sk-test")

        # Make sure langfuse is NOT in sys.modules
        monkeypatch.delitem(sys.modules, "langfuse", raising=False)
        monkeypatch.delitem(sys.modules, "langfuse.callback", raising=False)

        import builtins
        original_import = builtins.__import__

        def fake_import(name, *args, **kwargs):
            if name == "langfuse.callback":
                raise ImportError("no module named langfuse")
            return original_import(name, *args, **kwargs)

        monkeypatch.setattr(builtins, "__import__", fake_import)

        result = mod._setup_langfuse()
        assert result == []

    def test_langfuse_exception_returns_empty_list(self, monkeypatch):
        """Exception during CallbackHandler construction returns []."""
        mod = _load_agent(monkeypatch)
        monkeypatch.setenv("LANGFUSE_HOST", "http://langfuse.test")
        monkeypatch.setenv("LANGFUSE_PUBLIC_KEY", "pk-test")
        monkeypatch.setenv("LANGFUSE_SECRET_KEY", "sk-test")

        fake_callback_mod = ModuleType("langfuse.callback")
        fake_callback_mod.CallbackHandler = MagicMock(side_effect=RuntimeError("connect failed"))
        fake_langfuse = ModuleType("langfuse")
        fake_langfuse.callback = fake_callback_mod

        monkeypatch.setitem(sys.modules, "langfuse", fake_langfuse)
        monkeypatch.setitem(sys.modules, "langfuse.callback", fake_callback_mod)

        result = mod._setup_langfuse()
        assert result == []

    def test_langfuse_callbacks_attached_to_llm(self, monkeypatch):
        """When langfuse is configured, callbacks are attached to the LLM."""
        fake_llm = MagicMock(name="llm")
        fake_llm_cls = MagicMock(return_value=fake_llm)
        fake_lc_anthropic = ModuleType("langchain_anthropic")
        fake_lc_anthropic.ChatAnthropic = fake_llm_cls

        mod = _load_agent(monkeypatch, {"langchain_anthropic": fake_lc_anthropic})

        monkeypatch.setenv("LANGFUSE_HOST", "http://langfuse.test")
        monkeypatch.setenv("LANGFUSE_PUBLIC_KEY", "pk-test")
        monkeypatch.setenv("LANGFUSE_SECRET_KEY", "sk-test")
        monkeypatch.delenv("ANTHROPIC_BASE_URL", raising=False)

        fake_handler = MagicMock(name="handler")
        fake_callback_mod = ModuleType("langfuse.callback")
        fake_callback_mod.CallbackHandler = MagicMock(return_value=fake_handler)
        fake_langfuse = ModuleType("langfuse")
        fake_langfuse.callback = fake_callback_mod

        monkeypatch.setitem(sys.modules, "langfuse", fake_langfuse)
        monkeypatch.setitem(sys.modules, "langfuse.callback", fake_callback_mod)

        mod.create_agent("anthropic:claude-test", [], "sys")
        assert fake_llm.callbacks == [fake_handler]
