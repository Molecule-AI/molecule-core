"""Tests for explicit base URL support in model and CLI runtimes."""

import asyncio
import importlib
import sys
from types import ModuleType, SimpleNamespace

from cli_executor import CLIAgentExecutor
from config import RuntimeConfig


def _install_agent_mocks(monkeypatch, chat_module_name: str, class_name: str, captured: dict):
    """Install lightweight provider + langgraph mocks before importing agent.py."""

    prebuilt_mod = ModuleType("langgraph.prebuilt")

    def fake_create_react_agent(*, model, tools, prompt):
        captured["react_agent"] = {"model": model, "tools": tools, "prompt": prompt}
        return {"model": model, "tools": tools, "prompt": prompt}

    prebuilt_mod.create_react_agent = fake_create_react_agent

    langgraph_mod = ModuleType("langgraph")
    monkeypatch.setitem(sys.modules, "langgraph", langgraph_mod)
    monkeypatch.setitem(sys.modules, "langgraph.prebuilt", prebuilt_mod)

    provider_mod = ModuleType(chat_module_name)

    class FakeLLM:
        def __init__(self, **kwargs):
            captured["llm_kwargs"] = kwargs

    setattr(provider_mod, class_name, FakeLLM)
    monkeypatch.setitem(sys.modules, chat_module_name, provider_mod)


def test_create_agent_uses_anthropic_base_url(monkeypatch):
    """Anthropic models should pass ANTHROPIC_BASE_URL through explicitly."""
    captured = {}
    _install_agent_mocks(monkeypatch, "langchain_anthropic", "ChatAnthropic", captured)
    monkeypatch.setenv("ANTHROPIC_BASE_URL", "https://anthropic.example/v1")

    # Re-import after mocks so agent.py binds to our fake modules.
    sys.modules.pop("agent", None)
    agent_mod = importlib.import_module("agent")

    agent_mod.create_agent("anthropic:claude-sonnet-4-6", [], "system prompt")

    assert captured["llm_kwargs"]["model"] == "claude-sonnet-4-6"
    assert captured["llm_kwargs"]["anthropic_api_url"] == "https://anthropic.example/v1"


def test_codex_runtime_preserves_openai_base_url(monkeypatch):
    """Codex CLI runtime should pass OPENAI_BASE_URL into the subprocess env."""
    captured = {}

    class FakeProc:
        returncode = 0

        async def communicate(self):
            return (b"ok", b"")

    async def fake_create_subprocess_exec(*cmd, **kwargs):
        captured["cmd"] = cmd
        captured["env"] = kwargs["env"]
        return FakeProc()

    async def fake_set_current_task(_task: str):
        return None

    monkeypatch.setattr(asyncio, "create_subprocess_exec", fake_create_subprocess_exec)
    monkeypatch.setenv("OPENAI_BASE_URL", "https://codex.example/v1")
    monkeypatch.setenv("OPENAI_API_KEY", "sk-test")

    executor = CLIAgentExecutor(
        runtime="codex",
        runtime_config=RuntimeConfig(model="gpt-5.4"),
        system_prompt="system prompt",
        config_path="/tmp",
        heartbeat=None,
    )
    executor._set_current_task = fake_set_current_task

    class FakeQueue:
        def __init__(self):
            self.events = []

        async def enqueue_event(self, event):
            self.events.append(event)

    asyncio.run(executor._run_cli("hello", FakeQueue()))

    assert captured["env"]["OPENAI_API_KEY"] == "sk-test"
    assert captured["env"]["OPENAI_BASE_URL"] == "https://codex.example/v1"
