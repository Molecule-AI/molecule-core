"""Smoke tests for all 6 agent runtime adapters.

Verifies that each adapter:
  1. Exposes the correct static identity methods
  2. Exports a valid config schema
  3. Has setup() and create_executor() coroutines
  4. setup() raises RuntimeError when its framework dep is missing
  5. create_executor() returns an AgentExecutor-compatible object

Heavy framework deps (crewai, autogen-agentchat, etc.) are mocked so these
tests run without installing the full dependency tree.
"""

import asyncio
import sys
from types import ModuleType
from unittest.mock import AsyncMock, MagicMock

import pytest


# ---------------------------------------------------------------------------
# Helper: build a minimal AdapterConfig
# ---------------------------------------------------------------------------

def _make_config(**overrides):
    from adapters.base import AdapterConfig
    defaults = dict(
        model="openai:gpt-4o-mini",
        system_prompt="You are a test agent.",
        tools=[],
        runtime_config={},
        config_path="/tmp/test-configs",
        workspace_id="ws-test",
        prompt_files=[],
        a2a_port=8000,
        heartbeat=None,
    )
    defaults.update(overrides)
    return AdapterConfig(**defaults)


# ---------------------------------------------------------------------------
# Helper: patch _common_setup on a BaseAdapter subclass to avoid full stack
# ---------------------------------------------------------------------------

def _stub_common_setup(adapter_instance, monkeypatch):
    """Replace _common_setup with a no-op returning a minimal SetupResult."""
    from adapters.base import SetupResult
    result = SetupResult(
        system_prompt="stub prompt",
        loaded_skills=[],
        langchain_tools=[],
        is_coordinator=False,
        children=[],
    )
    monkeypatch.setattr(type(adapter_instance), "_common_setup", AsyncMock(return_value=result))


# ============================================================================
# 1. LangGraph Adapter
# ============================================================================

class TestLangGraphAdapter:

    def test_static_identity(self):
        from adapters.langgraph.adapter import LangGraphAdapter
        assert LangGraphAdapter.name() == "langgraph"
        assert LangGraphAdapter.display_name() == "LangGraph"
        assert isinstance(LangGraphAdapter.description(), str)
        assert len(LangGraphAdapter.description()) > 0

    def test_config_schema(self):
        from adapters.langgraph.adapter import LangGraphAdapter
        schema = LangGraphAdapter.get_config_schema()
        assert isinstance(schema, dict)
        assert "model" in schema

    def test_has_setup_and_create_executor(self):
        from adapters.langgraph.adapter import LangGraphAdapter
        import inspect
        adapter = LangGraphAdapter()
        assert inspect.iscoroutinefunction(adapter.setup)
        assert inspect.iscoroutinefunction(adapter.create_executor)

    @pytest.mark.asyncio
    async def test_setup_stores_tools_and_prompt(self, monkeypatch):
        from adapters.langgraph.adapter import LangGraphAdapter
        adapter = LangGraphAdapter()
        _stub_common_setup(adapter, monkeypatch)

        await adapter.setup(_make_config())

        assert adapter.system_prompt == "stub prompt"
        assert adapter.all_tools == []
        assert adapter.loaded_skills == []

    @pytest.mark.asyncio
    async def test_create_executor_returns_executor(self, monkeypatch):
        from adapters.langgraph.adapter import LangGraphAdapter

        # Mock create_agent and LangGraphA2AExecutor
        fake_agent = MagicMock()
        fake_executor = MagicMock()
        monkeypatch.setitem(sys.modules, "agent", MagicMock(create_agent=MagicMock(return_value=fake_agent)))
        monkeypatch.setitem(sys.modules, "a2a_executor", MagicMock(LangGraphA2AExecutor=MagicMock(return_value=fake_executor)))

        adapter = LangGraphAdapter()
        adapter.all_tools = []
        adapter.system_prompt = "test"
        adapter.loaded_skills = []

        result = await adapter.create_executor(_make_config())

        assert result is fake_executor


# ============================================================================
# 2. CrewAI Adapter
# ============================================================================

class TestCrewAIAdapter:

    def test_static_identity(self):
        from adapters.crewai.adapter import CrewAIAdapter
        assert CrewAIAdapter.name() == "crewai"
        assert CrewAIAdapter.display_name() == "CrewAI"
        assert isinstance(CrewAIAdapter.description(), str)

    def test_config_schema(self):
        from adapters.crewai.adapter import CrewAIAdapter
        schema = CrewAIAdapter.get_config_schema()
        assert isinstance(schema, dict)
        assert "model" in schema

    def test_has_setup_and_create_executor(self):
        from adapters.crewai.adapter import CrewAIAdapter
        import inspect
        adapter = CrewAIAdapter()
        assert inspect.iscoroutinefunction(adapter.setup)
        assert inspect.iscoroutinefunction(adapter.create_executor)

    @pytest.mark.asyncio
    async def test_setup_raises_when_crewai_missing(self, monkeypatch):
        from adapters.crewai.adapter import CrewAIAdapter
        adapter = CrewAIAdapter()
        # Hide crewai from imports
        monkeypatch.setitem(sys.modules, "crewai", None)

        with pytest.raises(RuntimeError, match="crewai not installed"):
            await adapter.setup(_make_config())

    @pytest.mark.asyncio
    async def test_setup_succeeds_with_crewai_present(self, monkeypatch):
        from adapters.crewai.adapter import CrewAIAdapter
        adapter = CrewAIAdapter()
        _stub_common_setup(adapter, monkeypatch)

        fake_crewai = ModuleType("crewai")
        fake_crewai.__version__ = "0.99.0"
        monkeypatch.setitem(sys.modules, "crewai", fake_crewai)

        await adapter.setup(_make_config())
        assert adapter.system_prompt == "stub prompt"

    @pytest.mark.asyncio
    async def test_create_executor_returns_crewai_executor(self, monkeypatch):
        from adapters.crewai.adapter import CrewAIAdapter, CrewAIA2AExecutor
        adapter = CrewAIAdapter()
        adapter.system_prompt = "backstory"
        adapter.crewai_tools = []

        result = await adapter.create_executor(_make_config())

        assert isinstance(result, CrewAIA2AExecutor)
        assert result.model == "openai:gpt-4o-mini"
        assert result.system_prompt == "backstory"

    @pytest.mark.asyncio
    async def test_crewai_executor_handles_empty_message(self, monkeypatch):
        from adapters.crewai.adapter import CrewAIA2AExecutor
        import adapters.shared_runtime as shared_rt

        executor = CrewAIA2AExecutor(
            model="openai:gpt-4o-mini",
            system_prompt="test",
            crewai_tools=[],
        )

        events = []
        event_queue = MagicMock()
        event_queue.enqueue_event = AsyncMock(side_effect=events.append)

        context = MagicMock()
        # Patch on the shared_runtime module (where it's imported from inside execute())
        monkeypatch.setattr(shared_rt, "extract_message_text", lambda ctx: "")
        monkeypatch.setattr(shared_rt, "set_current_task", AsyncMock())

        await executor.execute(context, event_queue)

        event_queue.enqueue_event.assert_awaited_once()
        assert events[0] == "No message provided"


# ============================================================================
# 3. Claude Code Adapter
# ============================================================================

class TestClaudeCodeAdapter:

    def test_static_identity(self):
        from adapters.claude_code.adapter import ClaudeCodeAdapter
        assert ClaudeCodeAdapter.name() == "claude-code"
        assert ClaudeCodeAdapter.display_name() == "Claude Code"
        assert isinstance(ClaudeCodeAdapter.description(), str)

    def test_config_schema(self):
        from adapters.claude_code.adapter import ClaudeCodeAdapter
        schema = ClaudeCodeAdapter.get_config_schema()
        assert isinstance(schema, dict)
        assert "model" in schema
        assert "timeout" in schema

    def test_has_setup_and_create_executor(self):
        from adapters.claude_code.adapter import ClaudeCodeAdapter
        import inspect
        adapter = ClaudeCodeAdapter()
        assert inspect.iscoroutinefunction(adapter.setup)
        assert inspect.iscoroutinefunction(adapter.create_executor)

    @pytest.mark.asyncio
    async def test_setup_warns_when_cli_missing(self, monkeypatch, caplog):
        """setup() should warn but NOT raise when the CLI is absent."""
        from adapters.claude_code.adapter import ClaudeCodeAdapter
        import shutil
        monkeypatch.setattr(shutil, "which", lambda cmd: None)

        adapter = ClaudeCodeAdapter()
        config = _make_config(runtime_config={"command": "claude"})
        # Should not raise
        await adapter.setup(config)

    @pytest.mark.asyncio
    async def test_create_executor_returns_sdk_executor(self, monkeypatch):
        from adapters.claude_code.adapter import ClaudeCodeAdapter

        fake_executor = MagicMock()
        fake_sdk_module = MagicMock()
        fake_sdk_module.ClaudeSDKExecutor = MagicMock(return_value=fake_executor)
        monkeypatch.setitem(sys.modules, "claude_sdk_executor", fake_sdk_module)

        adapter = ClaudeCodeAdapter()
        result = await adapter.create_executor(
            _make_config(runtime_config={"model": "opus"})
        )

        assert result is fake_executor
        # Verify model was forwarded from runtime_config
        kwargs = fake_sdk_module.ClaudeSDKExecutor.call_args.kwargs
        assert kwargs["model"] == "opus"


# ============================================================================
# 4. AutoGen Adapter
# ============================================================================

class TestAutoGenAdapter:

    def test_static_identity(self):
        from adapters.autogen.adapter import AutoGenAdapter
        assert AutoGenAdapter.name() == "autogen"
        assert AutoGenAdapter.display_name() == "AutoGen"
        assert isinstance(AutoGenAdapter.description(), str)

    def test_config_schema(self):
        from adapters.autogen.adapter import AutoGenAdapter
        schema = AutoGenAdapter.get_config_schema()
        assert isinstance(schema, dict)
        assert "model" in schema

    def test_has_setup_and_create_executor(self):
        from adapters.autogen.adapter import AutoGenAdapter
        import inspect
        adapter = AutoGenAdapter()
        assert inspect.iscoroutinefunction(adapter.setup)
        assert inspect.iscoroutinefunction(adapter.create_executor)

    @pytest.mark.asyncio
    async def test_setup_raises_when_autogen_missing(self, monkeypatch):
        from adapters.autogen.adapter import AutoGenAdapter
        adapter = AutoGenAdapter()
        monkeypatch.setitem(sys.modules, "autogen_agentchat", None)
        monkeypatch.setitem(sys.modules, "autogen_agentchat.agents", None)

        with pytest.raises((RuntimeError, ImportError)):
            await adapter.setup(_make_config())

    @pytest.mark.asyncio
    async def test_setup_succeeds_with_autogen_present(self, monkeypatch):
        from adapters.autogen.adapter import AutoGenAdapter
        adapter = AutoGenAdapter()
        _stub_common_setup(adapter, monkeypatch)

        fake_autogen_mod = ModuleType("autogen_agentchat")
        fake_agents_mod = ModuleType("autogen_agentchat.agents")
        fake_agents_mod.AssistantAgent = MagicMock()
        fake_autogen_mod.agents = fake_agents_mod
        monkeypatch.setitem(sys.modules, "autogen_agentchat", fake_autogen_mod)
        monkeypatch.setitem(sys.modules, "autogen_agentchat.agents", fake_agents_mod)

        await adapter.setup(_make_config())
        assert adapter.system_prompt == "stub prompt"

    @pytest.mark.asyncio
    async def test_create_executor_returns_autogen_executor(self, monkeypatch):
        from adapters.autogen.adapter import AutoGenAdapter, AutoGenA2AExecutor
        adapter = AutoGenAdapter()
        adapter.system_prompt = "autogen system"
        adapter.autogen_tools = []

        result = await adapter.create_executor(_make_config())

        assert isinstance(result, AutoGenA2AExecutor)
        assert result.system_prompt == "autogen system"


# ============================================================================
# 5. DeepAgents Adapter
# ============================================================================

class TestDeepAgentsAdapter:

    def test_static_identity(self):
        from adapters.deepagents.adapter import DeepAgentsAdapter
        assert DeepAgentsAdapter.name() == "deepagents"
        assert DeepAgentsAdapter.display_name() == "DeepAgents"
        assert isinstance(DeepAgentsAdapter.description(), str)

    def test_config_schema(self):
        from adapters.deepagents.adapter import DeepAgentsAdapter
        schema = DeepAgentsAdapter.get_config_schema()
        assert isinstance(schema, dict)
        assert "model" in schema

    def test_has_setup_and_create_executor(self):
        from adapters.deepagents.adapter import DeepAgentsAdapter
        import inspect
        adapter = DeepAgentsAdapter()
        assert inspect.iscoroutinefunction(adapter.setup)
        assert inspect.iscoroutinefunction(adapter.create_executor)

    @pytest.mark.asyncio
    async def test_setup_raises_when_deepagents_missing(self, monkeypatch):
        from adapters.deepagents.adapter import DeepAgentsAdapter
        adapter = DeepAgentsAdapter()
        monkeypatch.setitem(sys.modules, "deepagents", None)

        with pytest.raises((RuntimeError, ImportError)):
            await adapter.setup(_make_config())

    @pytest.mark.asyncio
    async def test_setup_succeeds_with_deepagents_present(self, monkeypatch):
        from adapters.deepagents.adapter import DeepAgentsAdapter
        adapter = DeepAgentsAdapter()
        _stub_common_setup(adapter, monkeypatch)

        fake_agent = MagicMock()

        # Mock deepagents package with all imports used by setup()
        fake_deepagents = ModuleType("deepagents")
        fake_deepagents.create_deep_agent = MagicMock(return_value=fake_agent)
        fake_deepagents.FilesystemPermission = MagicMock()
        monkeypatch.setitem(sys.modules, "deepagents", fake_deepagents)

        fake_backends = ModuleType("deepagents.backends")
        fake_backends.FilesystemBackend = MagicMock()
        monkeypatch.setitem(sys.modules, "deepagents.backends", fake_backends)

        fake_checkpoint = ModuleType("langgraph.checkpoint.memory")
        fake_checkpoint.MemorySaver = MagicMock()
        monkeypatch.setitem(sys.modules, "langgraph.checkpoint.memory", fake_checkpoint)
        monkeypatch.setitem(sys.modules, "langgraph.checkpoint", ModuleType("langgraph.checkpoint"))
        monkeypatch.setitem(sys.modules, "langgraph", ModuleType("langgraph"))

        fake_cache_mod = ModuleType("langchain_core.caches")
        fake_cache_mod.InMemoryCache = MagicMock()
        monkeypatch.setitem(sys.modules, "langchain_core.caches", fake_cache_mod)
        monkeypatch.setitem(sys.modules, "langchain_core", ModuleType("langchain_core"))

        # Mock the LLM creation
        monkeypatch.setattr(adapter, "_create_llm", lambda model: MagicMock())

        await adapter.setup(_make_config())
        assert adapter.agent is fake_agent
        # virtual_mode must be False so read_file/ls/write_file/edit_file
        # hit the real bind-mounted /workspace instead of an in-memory
        # snapshot that silently drifts from what `bash` sees.
        fs_call = fake_backends.FilesystemBackend.call_args
        assert fs_call is not None, "FilesystemBackend was never constructed"
        assert fs_call.kwargs.get("virtual_mode") is False, (
            "FilesystemBackend must be built with virtual_mode=False — "
            "virtual_mode=True caused agents to report real files as missing "
            "and silently dropped writes across restarts. See commit bc563d1."
        )

    @pytest.mark.asyncio
    async def test_create_executor_returns_langgraph_executor(self, monkeypatch):
        from adapters.deepagents.adapter import DeepAgentsAdapter
        fake_executor = MagicMock()
        fake_a2a_executor_mod = MagicMock()
        fake_a2a_executor_mod.LangGraphA2AExecutor = MagicMock(return_value=fake_executor)
        monkeypatch.setitem(sys.modules, "a2a_executor", fake_a2a_executor_mod)

        adapter = DeepAgentsAdapter()
        adapter.agent = MagicMock()

        result = await adapter.create_executor(_make_config())
        assert result is fake_executor

    def test_create_llm_openai(self, monkeypatch):
        from adapters.deepagents.adapter import DeepAgentsAdapter
        adapter = DeepAgentsAdapter()

        fake_openai_mod = ModuleType("langchain_openai")
        fake_llm = MagicMock()
        fake_openai_mod.ChatOpenAI = MagicMock(return_value=fake_llm)
        monkeypatch.setitem(sys.modules, "langchain_openai", fake_openai_mod)

        result = adapter._create_llm("openai:gpt-4o")
        assert result is fake_llm

    def test_create_llm_anthropic(self, monkeypatch):
        from adapters.deepagents.adapter import DeepAgentsAdapter
        adapter = DeepAgentsAdapter()

        fake_anthropic_mod = ModuleType("langchain_anthropic")
        fake_llm = MagicMock()
        fake_anthropic_mod.ChatAnthropic = MagicMock(return_value=fake_llm)
        monkeypatch.setitem(sys.modules, "langchain_anthropic", fake_anthropic_mod)

        result = adapter._create_llm("anthropic:claude-sonnet-4-6")
        assert result is fake_llm

    def test_create_llm_cerebras(self, monkeypatch):
        from adapters.deepagents.adapter import DeepAgentsAdapter
        adapter = DeepAgentsAdapter()

        fake_openai_mod = ModuleType("langchain_openai")
        fake_llm = MagicMock()
        fake_openai_mod.ChatOpenAI = MagicMock(return_value=fake_llm)
        monkeypatch.setitem(sys.modules, "langchain_openai", fake_openai_mod)
        monkeypatch.setenv("CEREBRAS_API_KEY", "test-key")

        result = adapter._create_llm("cerebras:llama3.1-8b")
        assert result is fake_llm
        fake_openai_mod.ChatOpenAI.assert_called_once_with(
            model="llama3.1-8b",
            openai_api_key="test-key",
            openai_api_base="https://api.cerebras.ai/v1",
        )

    def test_create_llm_google_genai(self, monkeypatch):
        from adapters.deepagents.adapter import DeepAgentsAdapter
        adapter = DeepAgentsAdapter()

        fake_genai_mod = ModuleType("langchain_google_genai")
        fake_llm = MagicMock()
        fake_genai_mod.ChatGoogleGenerativeAI = MagicMock(return_value=fake_llm)
        monkeypatch.setitem(sys.modules, "langchain_google_genai", fake_genai_mod)

        result = adapter._create_llm("google_genai:gemini-2.5-flash")
        assert result is fake_llm
        fake_genai_mod.ChatGoogleGenerativeAI.assert_called_once_with(model="gemini-2.5-flash")

    def test_create_llm_ollama(self, monkeypatch):
        from adapters.deepagents.adapter import DeepAgentsAdapter
        adapter = DeepAgentsAdapter()

        fake_ollama_mod = ModuleType("langchain_ollama")
        fake_llm = MagicMock()
        fake_ollama_mod.ChatOllama = MagicMock(return_value=fake_llm)
        monkeypatch.setitem(sys.modules, "langchain_ollama", fake_ollama_mod)

        result = adapter._create_llm("ollama:llama3")
        assert result is fake_llm
        fake_ollama_mod.ChatOllama.assert_called_once_with(model="llama3")

    def test_create_llm_unknown_provider_raises(self):
        from adapters.deepagents.adapter import DeepAgentsAdapter
        adapter = DeepAgentsAdapter()

        with pytest.raises(ValueError, match="Unsupported model provider"):
            adapter._create_llm("badprovider:some-model")

    def test_create_llm_default_provider_is_anthropic(self, monkeypatch):
        from adapters.deepagents.adapter import DeepAgentsAdapter
        adapter = DeepAgentsAdapter()

        fake_anthropic_mod = ModuleType("langchain_anthropic")
        fake_llm = MagicMock()
        fake_anthropic_mod.ChatAnthropic = MagicMock(return_value=fake_llm)
        monkeypatch.setitem(sys.modules, "langchain_anthropic", fake_anthropic_mod)

        result = adapter._create_llm("claude-sonnet-4-6")
        assert result is fake_llm

    @pytest.mark.asyncio
    async def test_create_executor_raises_without_setup(self, monkeypatch):
        from adapters.deepagents.adapter import DeepAgentsAdapter
        fake_a2a_executor_mod = MagicMock()
        monkeypatch.setitem(sys.modules, "a2a_executor", fake_a2a_executor_mod)

        adapter = DeepAgentsAdapter()
        with pytest.raises(RuntimeError, match="setup\\(\\) must be called"):
            await adapter.create_executor(_make_config())


# ============================================================================
# 6. OpenClaw Adapter
# ============================================================================

class TestOpenClawAdapter:

    def test_static_identity(self):
        from adapters.openclaw.adapter import OpenClawAdapter
        assert OpenClawAdapter.name() == "openclaw"
        assert OpenClawAdapter.display_name() == "OpenClaw"
        assert isinstance(OpenClawAdapter.description(), str)

    def test_config_schema(self):
        from adapters.openclaw.adapter import OpenClawAdapter
        schema = OpenClawAdapter.get_config_schema()
        assert isinstance(schema, dict)
        assert "model" in schema
        assert "gateway_port" in schema

    def test_has_setup_and_create_executor(self):
        from adapters.openclaw.adapter import OpenClawAdapter
        import inspect
        adapter = OpenClawAdapter()
        assert inspect.iscoroutinefunction(adapter.setup)
        assert inspect.iscoroutinefunction(adapter.create_executor)

    @pytest.mark.asyncio
    async def test_setup_raises_when_openclaw_cli_install_fails(self, monkeypatch):
        """setup() raises RuntimeError if npm install for openclaw CLI fails."""
        import shutil
        import subprocess
        from adapters.openclaw.adapter import OpenClawAdapter

        monkeypatch.setattr(shutil, "which", lambda cmd: None)  # CLI not found

        fake_result = MagicMock()
        fake_result.returncode = 1
        fake_result.stderr = "npm ERR! not found"
        monkeypatch.setattr(subprocess, "run", MagicMock(return_value=fake_result))

        adapter = OpenClawAdapter()
        with pytest.raises(RuntimeError, match="Failed to install OpenClaw"):
            await adapter.setup(_make_config())

    @pytest.mark.asyncio
    async def test_create_executor_returns_openclaw_executor(self, monkeypatch):
        from adapters.openclaw.adapter import OpenClawAdapter, OpenClawA2AExecutor
        adapter = OpenClawAdapter()

        result = await adapter.create_executor(_make_config())

        assert isinstance(result, OpenClawA2AExecutor)

    @pytest.mark.asyncio
    async def test_openclaw_executor_handles_empty_message(self, monkeypatch):
        from adapters.openclaw.adapter import OpenClawA2AExecutor

        executor = OpenClawA2AExecutor()
        events = []
        event_queue = MagicMock()
        event_queue.enqueue_event = AsyncMock(side_effect=events.append)
        context = MagicMock()

        monkeypatch.setattr("adapters.openclaw.adapter.extract_message_text", lambda ctx: "")
        monkeypatch.setattr("adapters.openclaw.adapter.set_current_task", AsyncMock())

        await executor.execute(context, event_queue)

        event_queue.enqueue_event.assert_awaited_once()
        assert events[0] == "No message provided"

    @pytest.mark.asyncio
    async def test_openclaw_executor_timeout(self, monkeypatch):
        """Executor returns a timeout error message when OpenClaw CLI times out."""
        import asyncio as _asyncio
        from adapters.openclaw.adapter import OpenClawA2AExecutor

        executor = OpenClawA2AExecutor()
        events = []
        event_queue = MagicMock()
        event_queue.enqueue_event = AsyncMock(side_effect=events.append)
        context = MagicMock()
        context.task_id = "t-1"

        monkeypatch.setattr("adapters.openclaw.adapter.extract_message_text", lambda ctx: "hello")
        monkeypatch.setattr("adapters.openclaw.adapter.set_current_task", AsyncMock())
        monkeypatch.setattr("adapters.openclaw.adapter.brief_task", lambda t: t)

        # Make asyncio.create_subprocess_exec raise TimeoutError via wait_for
        async def fake_create_subprocess_exec(*args, **kwargs):
            proc = MagicMock()
            async def communicate():
                raise _asyncio.TimeoutError()
            proc.communicate = communicate
            return proc

        monkeypatch.setattr(_asyncio, "create_subprocess_exec", fake_create_subprocess_exec)

        await executor.execute(context, event_queue)

        event_queue.enqueue_event.assert_awaited_once()
        reply = events[0]
        assert "timed out" in reply.lower() or "timeout" in reply.lower() or "120s" in reply


# ============================================================================
# Cross-adapter: Adapter registry
# ============================================================================

class TestAdapterRegistry:
    """Verify the adapter __init__.py discovers all 6 adapters."""

    @pytest.fixture(autouse=True)
    def clear_adapter_cache(self):
        """Clear the adapter cache before each registry test."""
        import adapters as _adapters_pkg
        _adapters_pkg._ADAPTER_CACHE.clear()
        yield
        _adapters_pkg._ADAPTER_CACHE.clear()

    def test_all_adapters_registered(self):
        from adapters import discover_adapters
        adapters = discover_adapters()
        names = set(adapters.keys())
        expected = {"langgraph", "crewai", "claude-code", "autogen", "deepagents", "openclaw", "hermes"}
        assert expected == names, f"Missing: {expected - names}, Extra: {names - expected}"

    def test_no_duplicate_names(self):
        from adapters import discover_adapters
        names = list(discover_adapters().keys())
        assert len(names) == len(set(names)), "Duplicate adapter names detected"

    def test_all_adapters_have_display_name(self):
        from adapters import discover_adapters
        for name, cls in discover_adapters().items():
            assert cls.display_name(), f"{name} has empty display_name"

    def test_all_adapters_have_description(self):
        from adapters import discover_adapters
        for name, cls in discover_adapters().items():
            assert len(cls.description()) > 10, f"{name} description too short"

    def test_discover_adapters_cache_hit(self):
        """Second call to discover_adapters() returns the cached dict without re-scanning."""
        from adapters import discover_adapters
        first = discover_adapters()
        # Call again — should return the exact same object (cache hit)
        second = discover_adapters()
        assert first is second

    def test_discover_adapters_skips_failing_import(self, monkeypatch, tmp_path):
        """discover_adapters() logs debug and continues when an adapter import fails."""
        import importlib
        import adapters as _adapters_pkg

        # Make importlib.import_module raise for any "adapters.X" import
        original_import = importlib.import_module

        def failing_import(name, *args, **kwargs):
            if name.startswith("adapters.") and name != "adapters.base":
                raise ImportError(f"Simulated missing dep for {name}")
            return original_import(name, *args, **kwargs)

        monkeypatch.setattr(importlib, "import_module", failing_import)

        result = _adapters_pkg.discover_adapters()
        # Cache was cleared by the fixture, so we get a fresh (empty) result
        assert isinstance(result, dict)

    def test_get_adapter_unknown_runtime_raises_key_error(self):
        """get_adapter() raises KeyError for an unrecognised runtime name."""
        from adapters import get_adapter
        import pytest
        with pytest.raises(KeyError, match="Unknown runtime"):
            get_adapter("not-a-real-runtime")

    def test_list_adapters_returns_metadata_dicts(self):
        """list_adapters() returns a list with name/display_name/description/config_schema."""
        from adapters import list_adapters
        result = list_adapters()
        assert isinstance(result, list)
        assert len(result) > 0
        for item in result:
            assert "name" in item
            assert "display_name" in item
            assert "description" in item
            assert "config_schema" in item


# ============================================================================
# AutoGen execute() — full execution path coverage
# ============================================================================

class TestAutoGenExecute:

    @pytest.mark.asyncio
    async def test_execute_success_with_str_reply(self, monkeypatch):
        """execute() extracts the last str-content message from result.messages."""
        from unittest.mock import patch, AsyncMock as _AsyncMock

        mock_autogen = MagicMock()
        mock_ext = MagicMock()
        monkeypatch.setitem(sys.modules, "autogen_agentchat", mock_autogen)
        monkeypatch.setitem(sys.modules, "autogen_agentchat.agents", mock_autogen.agents)
        monkeypatch.setitem(sys.modules, "autogen_ext", mock_ext)
        monkeypatch.setitem(sys.modules, "autogen_ext.models", mock_ext.models)
        monkeypatch.setitem(sys.modules, "autogen_ext.models.openai", mock_ext.models.openai)

        from adapters.autogen.adapter import AutoGenA2AExecutor

        mock_msg = MagicMock()
        mock_msg.content = "The answer is 42"
        mock_result = MagicMock()
        mock_result.messages = [mock_msg]

        mock_agent = AsyncMock()
        mock_agent.run = AsyncMock(return_value=mock_result)
        mock_autogen.agents.AssistantAgent.return_value = mock_agent

        executor = AutoGenA2AExecutor(
            model="openai:gpt-4o-mini",
            system_prompt="You are helpful",
            autogen_tools=[],
            heartbeat=None,
        )

        context = MagicMock()
        event_queue = AsyncMock()

        with patch("adapters.autogen.adapter.extract_message_text", return_value="What is 6*7?"), \
             patch("adapters.autogen.adapter.set_current_task", new_callable=_AsyncMock), \
             patch("adapters.autogen.adapter.extract_history", return_value=[]), \
             patch("adapters.autogen.adapter.build_task_text", return_value="What is 6*7?"):
            await executor.execute(context, event_queue)

        event_queue.enqueue_event.assert_called_once()
        call_arg = str(event_queue.enqueue_event.call_args[0][0])
        assert "The answer is 42" in call_arg

    @pytest.mark.asyncio
    async def test_execute_fallback_to_str_result_when_no_str_message(self, monkeypatch):
        """When no message has str content, reply falls back to str(result)."""
        from unittest.mock import patch, AsyncMock as _AsyncMock

        mock_autogen = MagicMock()
        mock_ext = MagicMock()
        monkeypatch.setitem(sys.modules, "autogen_agentchat", mock_autogen)
        monkeypatch.setitem(sys.modules, "autogen_agentchat.agents", mock_autogen.agents)
        monkeypatch.setitem(sys.modules, "autogen_ext", mock_ext)
        monkeypatch.setitem(sys.modules, "autogen_ext.models", mock_ext.models)
        monkeypatch.setitem(sys.modules, "autogen_ext.models.openai", mock_ext.models.openai)

        from adapters.autogen.adapter import AutoGenA2AExecutor

        # Message with non-str content — no valid reply extracted, falls back to str(result)
        mock_msg = MagicMock()
        mock_msg.content = 12345  # not a str
        mock_result = MagicMock()
        mock_result.messages = [mock_msg]
        mock_result.__str__ = lambda self: "fallback-str-result"

        mock_agent = AsyncMock()
        mock_agent.run = AsyncMock(return_value=mock_result)
        mock_autogen.agents.AssistantAgent.return_value = mock_agent

        executor = AutoGenA2AExecutor(
            model="gpt-4o",  # no colon → model_name = model_str
            system_prompt=None,
            autogen_tools=[],
            heartbeat=None,
        )

        context = MagicMock()
        event_queue = AsyncMock()

        with patch("adapters.autogen.adapter.extract_message_text", return_value="hello"), \
             patch("adapters.autogen.adapter.set_current_task", new_callable=_AsyncMock), \
             patch("adapters.autogen.adapter.extract_history", return_value=[]), \
             patch("adapters.autogen.adapter.build_task_text", return_value="hello"):
            await executor.execute(context, event_queue)

        event_queue.enqueue_event.assert_called_once()
        call_arg = str(event_queue.enqueue_event.call_args[0][0])
        assert "fallback-str-result" in call_arg

    @pytest.mark.asyncio
    async def test_execute_exception_path(self, monkeypatch):
        """When the agent raises, reply is 'AutoGen error: ...'."""
        from unittest.mock import patch, AsyncMock as _AsyncMock

        mock_autogen = MagicMock()
        mock_ext = MagicMock()
        monkeypatch.setitem(sys.modules, "autogen_agentchat", mock_autogen)
        monkeypatch.setitem(sys.modules, "autogen_agentchat.agents", mock_autogen.agents)
        monkeypatch.setitem(sys.modules, "autogen_ext", mock_ext)
        monkeypatch.setitem(sys.modules, "autogen_ext.models", mock_ext.models)
        monkeypatch.setitem(sys.modules, "autogen_ext.models.openai", mock_ext.models.openai)

        from adapters.autogen.adapter import AutoGenA2AExecutor

        mock_agent = AsyncMock()
        mock_agent.run = AsyncMock(side_effect=RuntimeError("model exploded"))
        mock_autogen.agents.AssistantAgent.return_value = mock_agent

        executor = AutoGenA2AExecutor(
            model="openai:gpt-4o-mini",
            system_prompt="test",
            autogen_tools=[],
            heartbeat=None,
        )

        context = MagicMock()
        event_queue = AsyncMock()

        with patch("adapters.autogen.adapter.extract_message_text", return_value="hello"), \
             patch("adapters.autogen.adapter.set_current_task", new_callable=_AsyncMock), \
             patch("adapters.autogen.adapter.extract_history", return_value=[]), \
             patch("adapters.autogen.adapter.build_task_text", return_value="hello"):
            await executor.execute(context, event_queue)

        call_arg = str(event_queue.enqueue_event.call_args[0][0])
        assert "AutoGen error" in call_arg
        assert "model exploded" in call_arg

    @pytest.mark.asyncio
    async def test_execute_empty_message_returns_early(self, monkeypatch):
        """When extract_message_text returns empty, enqueues 'No message provided'."""
        from unittest.mock import patch
        from adapters.autogen.adapter import AutoGenA2AExecutor

        executor = AutoGenA2AExecutor(
            model="openai:gpt-4o-mini",
            system_prompt="test",
            autogen_tools=[],
            heartbeat=None,
        )

        context = MagicMock()
        event_queue = AsyncMock()

        with patch("adapters.autogen.adapter.extract_message_text", return_value=""):
            await executor.execute(context, event_queue)

        call_arg = str(event_queue.enqueue_event.call_args[0][0])
        assert "No message provided" in call_arg

    @pytest.mark.asyncio
    async def test_execute_finally_clears_task(self, monkeypatch):
        """set_current_task("") is called in finally block even after exception."""
        from unittest.mock import patch, AsyncMock as _AsyncMock

        mock_autogen = MagicMock()
        mock_ext = MagicMock()
        monkeypatch.setitem(sys.modules, "autogen_agentchat", mock_autogen)
        monkeypatch.setitem(sys.modules, "autogen_agentchat.agents", mock_autogen.agents)
        monkeypatch.setitem(sys.modules, "autogen_ext", mock_ext)
        monkeypatch.setitem(sys.modules, "autogen_ext.models", mock_ext.models)
        monkeypatch.setitem(sys.modules, "autogen_ext.models.openai", mock_ext.models.openai)

        from adapters.autogen.adapter import AutoGenA2AExecutor

        mock_agent = AsyncMock()
        mock_agent.run = AsyncMock(side_effect=Exception("boom"))
        mock_autogen.agents.AssistantAgent.return_value = mock_agent

        executor = AutoGenA2AExecutor(
            model="openai:gpt-4o-mini",
            system_prompt="test",
            autogen_tools=[],
            heartbeat=MagicMock(),
        )

        context = MagicMock()
        event_queue = AsyncMock()
        set_task_calls = []

        async def fake_set_current_task(hb, task):
            set_task_calls.append(task)

        with patch("adapters.autogen.adapter.extract_message_text", return_value="hello"), \
             patch("adapters.autogen.adapter.set_current_task", side_effect=fake_set_current_task), \
             patch("adapters.autogen.adapter.extract_history", return_value=[]), \
             patch("adapters.autogen.adapter.build_task_text", return_value="hello"):
            await executor.execute(context, event_queue)

        # Last call must clear the task
        assert set_task_calls[-1] == ""


# ============================================================================
# CrewAI execute() — full execution path coverage
# ============================================================================

class TestCrewAIExecute:

    @pytest.mark.asyncio
    async def test_execute_success(self, monkeypatch):
        """execute() calls crew.kickoff and enqueues the result string."""
        from unittest.mock import patch, AsyncMock as _AsyncMock

        mock_crewai = MagicMock()
        monkeypatch.setitem(sys.modules, "crewai", mock_crewai)

        from adapters.crewai.adapter import CrewAIA2AExecutor

        mock_crew_instance = MagicMock()
        mock_crew_instance.kickoff.return_value = "Crew result here"
        mock_crewai.Crew.return_value = mock_crew_instance
        mock_crewai.Agent.return_value = MagicMock()
        mock_crewai.Task.return_value = MagicMock()

        executor = CrewAIA2AExecutor(
            model="openai:gpt-4o-mini",
            system_prompt="Be helpful",
            crewai_tools=[],
            heartbeat=None,
        )

        context = MagicMock()
        event_queue = AsyncMock()

        async def fake_to_thread(fn, *args, **kwargs):
            return fn(*args, **kwargs)

        import adapters.shared_runtime as _srt
        with patch("adapters.crewai.adapter.asyncio.to_thread", side_effect=fake_to_thread), \
             patch.object(_srt, "extract_message_text", return_value="Hello crew"), \
             patch.object(_srt, "set_current_task", new_callable=_AsyncMock), \
             patch.object(_srt, "extract_history", return_value=[]), \
             patch.object(_srt, "build_task_text", return_value="Hello crew"):
            await executor.execute(context, event_queue)

        event_queue.enqueue_event.assert_called_once()
        call_arg = str(event_queue.enqueue_event.call_args[0][0])
        assert "Crew result here" in call_arg

    @pytest.mark.asyncio
    async def test_execute_model_conversion(self, monkeypatch):
        """openai: prefix is converted to openai/ for CrewAI."""
        from unittest.mock import patch, AsyncMock as _AsyncMock

        mock_crewai = MagicMock()
        monkeypatch.setitem(sys.modules, "crewai", mock_crewai)

        from adapters.crewai.adapter import CrewAIA2AExecutor

        captured_model = []

        def capture_agent(**kwargs):
            captured_model.append(kwargs.get("llm"))
            return MagicMock()

        mock_crewai.Agent.side_effect = capture_agent
        mock_crewai.Task.return_value = MagicMock()
        mock_crew_instance = MagicMock()
        mock_crew_instance.kickoff.return_value = "ok"
        mock_crewai.Crew.return_value = mock_crew_instance

        executor = CrewAIA2AExecutor(
            model="openai:gpt-4.1",
            system_prompt="test",
            crewai_tools=[],
            heartbeat=None,
        )

        context = MagicMock()
        event_queue = AsyncMock()

        async def fake_to_thread(fn, *a, **kw):
            return fn(*a, **kw)

        import adapters.shared_runtime as _srt
        with patch("adapters.crewai.adapter.asyncio.to_thread", side_effect=fake_to_thread), \
             patch.object(_srt, "extract_message_text", return_value="hello"), \
             patch.object(_srt, "set_current_task", new_callable=_AsyncMock), \
             patch.object(_srt, "extract_history", return_value=[]), \
             patch.object(_srt, "build_task_text", return_value="hello"):
            await executor.execute(context, event_queue)

        assert captured_model[0] == "openai/gpt-4.1"

    @pytest.mark.asyncio
    async def test_execute_exception_path(self, monkeypatch):
        """When crew.kickoff raises, reply is 'CrewAI error: ...'."""
        from unittest.mock import patch, AsyncMock as _AsyncMock

        mock_crewai = MagicMock()
        monkeypatch.setitem(sys.modules, "crewai", mock_crewai)

        from adapters.crewai.adapter import CrewAIA2AExecutor

        mock_crewai.Agent.return_value = MagicMock()
        mock_crewai.Task.return_value = MagicMock()
        mock_crew_instance = MagicMock()
        mock_crew_instance.kickoff.side_effect = RuntimeError("crew failure")
        mock_crewai.Crew.return_value = mock_crew_instance

        executor = CrewAIA2AExecutor(
            model="openai:gpt-4o-mini",
            system_prompt="test",
            crewai_tools=[],
            heartbeat=None,
        )

        context = MagicMock()
        event_queue = AsyncMock()

        async def fake_to_thread(fn, *a, **kw):
            return fn(*a, **kw)

        import adapters.shared_runtime as _srt
        with patch("adapters.crewai.adapter.asyncio.to_thread", side_effect=fake_to_thread), \
             patch.object(_srt, "extract_message_text", return_value="hello"), \
             patch.object(_srt, "set_current_task", new_callable=_AsyncMock), \
             patch.object(_srt, "extract_history", return_value=[]), \
             patch.object(_srt, "build_task_text", return_value="hello"):
            await executor.execute(context, event_queue)

        call_arg = str(event_queue.enqueue_event.call_args[0][0])
        assert "CrewAI error" in call_arg
        assert "crew failure" in call_arg

    @pytest.mark.asyncio
    async def test_execute_finally_clears_task(self, monkeypatch):
        """set_current_task("") is called in the finally block."""
        from unittest.mock import patch, AsyncMock as _AsyncMock

        mock_crewai = MagicMock()
        monkeypatch.setitem(sys.modules, "crewai", mock_crewai)

        from adapters.crewai.adapter import CrewAIA2AExecutor

        mock_crewai.Agent.return_value = MagicMock()
        mock_crewai.Task.return_value = MagicMock()
        mock_crew_instance = MagicMock()
        mock_crew_instance.kickoff.side_effect = Exception("boom")
        mock_crewai.Crew.return_value = mock_crew_instance

        executor = CrewAIA2AExecutor(
            model="openai:gpt-4o-mini",
            system_prompt="test",
            crewai_tools=[],
            heartbeat=MagicMock(),
        )

        context = MagicMock()
        event_queue = AsyncMock()
        set_task_calls = []

        async def fake_set_current_task(hb, task):
            set_task_calls.append(task)

        async def fake_to_thread(fn, *a, **kw):
            return fn(*a, **kw)

        import adapters.shared_runtime as _srt
        with patch("adapters.crewai.adapter.asyncio.to_thread", side_effect=fake_to_thread), \
             patch.object(_srt, "extract_message_text", return_value="hi"), \
             patch.object(_srt, "set_current_task", side_effect=fake_set_current_task), \
             patch.object(_srt, "extract_history", return_value=[]), \
             patch.object(_srt, "build_task_text", return_value="hi"):
            await executor.execute(context, event_queue)

        assert set_task_calls[-1] == ""


# ============================================================================
# DeepAgents _create_llm() — uncovered provider branches
# ============================================================================

class TestDeepAgentsCreateLlmBranches:

    def test_create_llm_no_colon_defaults_to_anthropic(self, monkeypatch):
        """Model string without ':' defaults to anthropic provider."""
        from types import ModuleType
        fake_anthropic = ModuleType("langchain_anthropic")
        fake_llm = MagicMock()
        fake_anthropic.ChatAnthropic = MagicMock(return_value=fake_llm)
        monkeypatch.setitem(sys.modules, "langchain_anthropic", fake_anthropic)

        from adapters.deepagents.adapter import DeepAgentsAdapter
        adapter = DeepAgentsAdapter()
        result = adapter._create_llm("claude-sonnet-4-6")

        fake_anthropic.ChatAnthropic.assert_called_once_with(model="claude-sonnet-4-6")
        assert result is fake_llm

    def test_create_llm_openai_with_base_url(self, monkeypatch):
        """When OPENAI_BASE_URL is set, openai_api_base is passed to ChatOpenAI."""
        from types import ModuleType
        fake_openai = ModuleType("langchain_openai")
        fake_llm = MagicMock()
        fake_openai.ChatOpenAI = MagicMock(return_value=fake_llm)
        monkeypatch.setitem(sys.modules, "langchain_openai", fake_openai)
        monkeypatch.setenv("OPENAI_BASE_URL", "http://localhost:11434/v1")

        from adapters.deepagents.adapter import DeepAgentsAdapter
        adapter = DeepAgentsAdapter()
        adapter._create_llm("openai:my-model")

        call_kwargs = fake_openai.ChatOpenAI.call_args[1]
        assert call_kwargs.get("openai_api_base") == "http://localhost:11434/v1"
        assert call_kwargs["model"] == "my-model"

    def test_create_llm_openrouter(self, monkeypatch):
        """openrouter provider uses ChatOpenAI with openrouter base URL."""
        from types import ModuleType
        fake_openai = ModuleType("langchain_openai")
        fake_llm = MagicMock()
        fake_openai.ChatOpenAI = MagicMock(return_value=fake_llm)
        monkeypatch.setitem(sys.modules, "langchain_openai", fake_openai)
        monkeypatch.setenv("OPENROUTER_API_KEY", "sk-or-test")

        from adapters.deepagents.adapter import DeepAgentsAdapter
        adapter = DeepAgentsAdapter()
        result = adapter._create_llm("openrouter:google/gemini-2.5-flash")

        call_kwargs = fake_openai.ChatOpenAI.call_args[1]
        assert "openrouter.ai" in call_kwargs.get("openai_api_base", "")
        assert call_kwargs["model"] == "google/gemini-2.5-flash"
        assert result is fake_llm

    def test_create_llm_groq(self, monkeypatch):
        """groq provider uses ChatOpenAI with groq base URL."""
        from types import ModuleType
        fake_openai = ModuleType("langchain_openai")
        fake_llm = MagicMock()
        fake_openai.ChatOpenAI = MagicMock(return_value=fake_llm)
        monkeypatch.setitem(sys.modules, "langchain_openai", fake_openai)
        monkeypatch.setenv("GROQ_API_KEY", "gsk_test")

        from adapters.deepagents.adapter import DeepAgentsAdapter
        adapter = DeepAgentsAdapter()
        result = adapter._create_llm("groq:llama3-8b-8192")

        call_kwargs = fake_openai.ChatOpenAI.call_args[1]
        assert "groq.com" in call_kwargs.get("openai_api_base", "")
        assert call_kwargs["model"] == "llama3-8b-8192"
        assert result is fake_llm

    def test_create_llm_anthropic_with_base_url(self, monkeypatch):
        """When ANTHROPIC_BASE_URL is set, anthropic_api_url is passed to ChatAnthropic."""
        from types import ModuleType
        fake_anthropic = ModuleType("langchain_anthropic")
        fake_llm = MagicMock()
        fake_anthropic.ChatAnthropic = MagicMock(return_value=fake_llm)
        monkeypatch.setitem(sys.modules, "langchain_anthropic", fake_anthropic)
        monkeypatch.setenv("ANTHROPIC_BASE_URL", "http://proxy:8080")

        from adapters.deepagents.adapter import DeepAgentsAdapter
        adapter = DeepAgentsAdapter()
        result = adapter._create_llm("anthropic:claude-sonnet-4-6")

        call_kwargs = fake_anthropic.ChatAnthropic.call_args[1]
        assert call_kwargs.get("anthropic_api_url") == "http://proxy:8080"
        assert result is fake_llm

    def test_create_llm_unknown_provider_raises(self):
        """Unknown provider raises ValueError instead of silently falling back."""
        from adapters.deepagents.adapter import DeepAgentsAdapter
        adapter = DeepAgentsAdapter()

        with pytest.raises(ValueError, match="Unsupported model provider"):
            adapter._create_llm("someunknown:my-model")

    def test_create_llm_multiple_colons_preserves_model(self, monkeypatch):
        """Model like 'google_genai:models/gemini-2.5-flash' splits on first colon only."""
        from types import ModuleType
        fake_genai = ModuleType("langchain_google_genai")
        fake_llm = MagicMock()
        fake_genai.ChatGoogleGenerativeAI = MagicMock(return_value=fake_llm)
        monkeypatch.setitem(sys.modules, "langchain_google_genai", fake_genai)

        from adapters.deepagents.adapter import DeepAgentsAdapter
        adapter = DeepAgentsAdapter()
        result = adapter._create_llm("google_genai:models/gemini-2.5-flash:latest")

        fake_genai.ChatGoogleGenerativeAI.assert_called_once_with(model="models/gemini-2.5-flash:latest")
        assert result is fake_llm

    def test_create_llm_openrouter_fallback_to_openai_key(self, monkeypatch):
        """When OPENROUTER_API_KEY is unset, falls back to OPENAI_API_KEY."""
        from types import ModuleType
        fake_openai = ModuleType("langchain_openai")
        fake_llm = MagicMock()
        fake_openai.ChatOpenAI = MagicMock(return_value=fake_llm)
        monkeypatch.setitem(sys.modules, "langchain_openai", fake_openai)
        monkeypatch.delenv("OPENROUTER_API_KEY", raising=False)
        monkeypatch.setenv("OPENAI_API_KEY", "sk-fallback-key")

        from adapters.deepagents.adapter import DeepAgentsAdapter
        adapter = DeepAgentsAdapter()
        adapter._create_llm("openrouter:meta-llama/llama-3-8b")

        call_kwargs = fake_openai.ChatOpenAI.call_args[1]
        assert call_kwargs["openai_api_key"] == "sk-fallback-key"

    def test_create_llm_openrouter_both_keys_unset(self, monkeypatch):
        """When both OPENROUTER_API_KEY and OPENAI_API_KEY are unset, empty string is used."""
        from types import ModuleType
        fake_openai = ModuleType("langchain_openai")
        fake_llm = MagicMock()
        fake_openai.ChatOpenAI = MagicMock(return_value=fake_llm)
        monkeypatch.setitem(sys.modules, "langchain_openai", fake_openai)
        monkeypatch.delenv("OPENROUTER_API_KEY", raising=False)
        monkeypatch.delenv("OPENAI_API_KEY", raising=False)

        from adapters.deepagents.adapter import DeepAgentsAdapter
        adapter = DeepAgentsAdapter()
        adapter._create_llm("openrouter:meta-llama/llama-3-8b")

        call_kwargs = fake_openai.ChatOpenAI.call_args[1]
        assert call_kwargs["openai_api_key"] == ""

    def test_create_llm_openai_without_base_url(self, monkeypatch):
        """When OPENAI_BASE_URL is not set, openai_api_base should NOT be passed."""
        from types import ModuleType
        fake_openai = ModuleType("langchain_openai")
        fake_llm = MagicMock()
        fake_openai.ChatOpenAI = MagicMock(return_value=fake_llm)
        monkeypatch.setitem(sys.modules, "langchain_openai", fake_openai)
        monkeypatch.delenv("OPENAI_BASE_URL", raising=False)

        from adapters.deepagents.adapter import DeepAgentsAdapter
        adapter = DeepAgentsAdapter()
        adapter._create_llm("openai:gpt-4o")

        call_kwargs = fake_openai.ChatOpenAI.call_args[1]
        assert "openai_api_base" not in call_kwargs

    def test_create_llm_anthropic_without_base_url(self, monkeypatch):
        """When ANTHROPIC_BASE_URL is not set, anthropic_api_url should NOT be passed."""
        from types import ModuleType
        fake_anthropic = ModuleType("langchain_anthropic")
        fake_llm = MagicMock()
        fake_anthropic.ChatAnthropic = MagicMock(return_value=fake_llm)
        monkeypatch.setitem(sys.modules, "langchain_anthropic", fake_anthropic)
        monkeypatch.delenv("ANTHROPIC_BASE_URL", raising=False)

        from adapters.deepagents.adapter import DeepAgentsAdapter
        adapter = DeepAgentsAdapter()
        adapter._create_llm("anthropic:claude-sonnet-4-6")

        call_kwargs = fake_anthropic.ChatAnthropic.call_args[1]
        assert "anthropic_api_url" not in call_kwargs

    def test_create_llm_groq_empty_api_key(self, monkeypatch):
        """When GROQ_API_KEY is not set, empty string is passed."""
        from types import ModuleType
        fake_openai = ModuleType("langchain_openai")
        fake_llm = MagicMock()
        fake_openai.ChatOpenAI = MagicMock(return_value=fake_llm)
        monkeypatch.setitem(sys.modules, "langchain_openai", fake_openai)
        monkeypatch.delenv("GROQ_API_KEY", raising=False)

        from adapters.deepagents.adapter import DeepAgentsAdapter
        adapter = DeepAgentsAdapter()
        adapter._create_llm("groq:llama3-8b")

        call_kwargs = fake_openai.ChatOpenAI.call_args[1]
        assert call_kwargs["openai_api_key"] == ""

    def test_create_llm_cerebras_empty_api_key(self, monkeypatch):
        """When CEREBRAS_API_KEY is not set, empty string is passed."""
        from types import ModuleType
        fake_openai = ModuleType("langchain_openai")
        fake_llm = MagicMock()
        fake_openai.ChatOpenAI = MagicMock(return_value=fake_llm)
        monkeypatch.setitem(sys.modules, "langchain_openai", fake_openai)
        monkeypatch.delenv("CEREBRAS_API_KEY", raising=False)

        from adapters.deepagents.adapter import DeepAgentsAdapter
        adapter = DeepAgentsAdapter()
        adapter._create_llm("cerebras:llama3.1-8b")

        call_kwargs = fake_openai.ChatOpenAI.call_args[1]
        assert call_kwargs["openai_api_key"] == ""

    def test_create_llm_openrouter_max_tokens(self, monkeypatch):
        """OpenRouter reads MAX_TOKENS env var."""
        from types import ModuleType
        fake_openai = ModuleType("langchain_openai")
        fake_llm = MagicMock()
        fake_openai.ChatOpenAI = MagicMock(return_value=fake_llm)
        monkeypatch.setitem(sys.modules, "langchain_openai", fake_openai)
        monkeypatch.setenv("OPENROUTER_API_KEY", "test")
        monkeypatch.setenv("MAX_TOKENS", "4096")

        from adapters.deepagents.adapter import DeepAgentsAdapter
        adapter = DeepAgentsAdapter()
        adapter._create_llm("openrouter:meta-llama/llama-3-8b")

        call_kwargs = fake_openai.ChatOpenAI.call_args[1]
        assert call_kwargs["max_tokens"] == 4096


# ============================================================================
# ClaudeCode create_executor() — system-prompt.md file reading
# ============================================================================

class TestClaudeCodeSystemPromptFile:

    @pytest.mark.asyncio
    async def test_create_executor_reads_prompt_file_when_system_prompt_none(self, tmp_path, monkeypatch):
        """When system_prompt is None and system-prompt.md exists, it is read."""
        from unittest.mock import patch

        prompt_file = tmp_path / "system-prompt.md"
        prompt_file.write_text("Be helpful and concise.")

        from adapters.claude_code.adapter import ClaudeCodeAdapter

        class FakeCLIAgentExecutor:
            def __init__(self, **kwargs):
                self.system_prompt = kwargs.get("system_prompt")

        fake_cli_mod = MagicMock()
        fake_cli_mod.CLIAgentExecutor = FakeCLIAgentExecutor
        monkeypatch.setitem(sys.modules, "cli_executor", fake_cli_mod)

        from config import RuntimeConfig
        monkeypatch.setitem(sys.modules, "config", MagicMock(RuntimeConfig=RuntimeConfig))

        adapter = ClaudeCodeAdapter()
        result = await adapter.create_executor(
            _make_config(config_path=str(tmp_path), system_prompt=None)
        )

        assert result.system_prompt == "Be helpful and concise."

    @pytest.mark.asyncio
    async def test_create_executor_skips_prompt_file_when_system_prompt_set(self, tmp_path, monkeypatch):
        """When system_prompt is already set, the prompt file is not used."""
        prompt_file = tmp_path / "system-prompt.md"
        prompt_file.write_text("Should not be used.")

        from adapters.claude_code.adapter import ClaudeCodeAdapter

        class FakeCLIAgentExecutor:
            def __init__(self, **kwargs):
                self.system_prompt = kwargs.get("system_prompt")

        fake_cli_mod = MagicMock()
        fake_cli_mod.CLIAgentExecutor = FakeCLIAgentExecutor
        monkeypatch.setitem(sys.modules, "cli_executor", fake_cli_mod)

        from config import RuntimeConfig
        monkeypatch.setitem(sys.modules, "config", MagicMock(RuntimeConfig=RuntimeConfig))

        adapter = ClaudeCodeAdapter()
        result = await adapter.create_executor(
            _make_config(config_path=str(tmp_path), system_prompt="Use the provided prompt.")
        )

        assert result.system_prompt == "Use the provided prompt."

    @pytest.mark.asyncio
    async def test_create_executor_no_prompt_file_no_system_prompt(self, tmp_path, monkeypatch):
        """When system_prompt is None and no file exists, system_prompt stays None."""
        from adapters.claude_code.adapter import ClaudeCodeAdapter

        class FakeCLIAgentExecutor:
            def __init__(self, **kwargs):
                self.system_prompt = kwargs.get("system_prompt")

        fake_cli_mod = MagicMock()
        fake_cli_mod.CLIAgentExecutor = FakeCLIAgentExecutor
        monkeypatch.setitem(sys.modules, "cli_executor", fake_cli_mod)

        from config import RuntimeConfig
        monkeypatch.setitem(sys.modules, "config", MagicMock(RuntimeConfig=RuntimeConfig))

        adapter = ClaudeCodeAdapter()
        # tmp_path has no system-prompt.md
        result = await adapter.create_executor(
            _make_config(config_path=str(tmp_path), system_prompt=None)
        )

        assert result.system_prompt is None


# ============================================================================
# BaseAdapter _common_setup() — plugin names, plugin skills, coordinator prompt
# ============================================================================

class TestCommonSetupMissingPaths:

    def _make_test_adapter_and_config(self, tmp_path):
        from adapters.base import BaseAdapter, AdapterConfig

        class TestAdapter(BaseAdapter):
            @staticmethod
            def name(): return "test"
            @staticmethod
            def display_name(): return "Test"
            @staticmethod
            def description(): return "Test adapter"
            async def setup(self, config): pass
            async def create_executor(self, config): pass

        adapter = TestAdapter()
        config = AdapterConfig(
            model="openai:test",
            config_path=str(tmp_path),
            workspace_id="ws-test",
        )
        return adapter, config

    @pytest.mark.asyncio
    async def test_common_setup_logs_plugin_names(self, tmp_path):
        """When plugins.plugin_names is non-empty, the logger.info path is reached."""
        from unittest.mock import patch, AsyncMock as _AsyncMock

        adapter, config = self._make_test_adapter_and_config(tmp_path)

        mock_plugins = MagicMock()
        mock_plugins.plugin_names = ["plugin-alpha"]
        mock_plugins.skill_dirs = []
        mock_plugins.prompt_fragments = []
        mock_plugins.rules = []

        with patch("plugins.load_plugins", return_value=mock_plugins), \
             patch("skill_loader.loader.load_skills", return_value=[]), \
             patch("coordinator.get_children", return_value=[]), \
             patch("coordinator.get_parent_context", return_value=[]), \
             patch("coordinator.build_children_description", return_value=""), \
             patch("prompt.get_peer_capabilities", return_value=[]), \
             patch("prompt.build_system_prompt", return_value="system prompt with plugin"):
            result = await adapter._common_setup(config)

        assert result.system_prompt == "system prompt with plugin"
        assert result.is_coordinator is False

    @pytest.mark.asyncio
    async def test_common_setup_loads_plugin_skill_dirs(self, tmp_path):
        """Plugin skill_dirs are iterated and new (non-duplicate) skills are appended."""
        import os
        from unittest.mock import patch, AsyncMock as _AsyncMock

        plugin_skills_root = tmp_path / "plugin_skills"
        plugin_skills_root.mkdir()
        (plugin_skills_root / "my_plugin_skill").mkdir()

        adapter, config = self._make_test_adapter_and_config(tmp_path)

        mock_plugins = MagicMock()
        mock_plugins.plugin_names = []
        mock_plugins.skill_dirs = [str(plugin_skills_root)]
        mock_plugins.prompt_fragments = []
        mock_plugins.rules = []

        fake_plugin_skill = MagicMock()
        fake_plugin_skill.metadata.id = "my_plugin_skill"
        fake_plugin_skill.tools = []

        def fake_load_skills(path, names):
            if str(path) == str(plugin_skills_root):
                return [fake_plugin_skill]
            return []

        with patch("plugins.load_plugins", return_value=mock_plugins), \
             patch("skill_loader.loader.load_skills", side_effect=fake_load_skills), \
             patch("coordinator.get_children", return_value=[]), \
             patch("coordinator.get_parent_context", return_value=[]), \
             patch("coordinator.build_children_description", return_value=""), \
             patch("prompt.get_peer_capabilities", return_value=[]), \
             patch("prompt.build_system_prompt", return_value="system"):
            result = await adapter._common_setup(config)

        skill_ids = [s.metadata.id for s in result.loaded_skills]
        assert "my_plugin_skill" in skill_ids

    @pytest.mark.asyncio
    async def test_common_setup_deduplicates_plugin_skills(self, tmp_path):
        """A plugin skill with the same id as a workspace skill is not appended twice."""
        from unittest.mock import patch, AsyncMock as _AsyncMock

        plugin_skills_root = tmp_path / "plugin_skills"
        plugin_skills_root.mkdir()
        (plugin_skills_root / "shared_skill").mkdir()

        adapter, config = self._make_test_adapter_and_config(tmp_path)

        mock_plugins = MagicMock()
        mock_plugins.plugin_names = []
        mock_plugins.skill_dirs = [str(plugin_skills_root)]
        mock_plugins.prompt_fragments = []
        mock_plugins.rules = []

        fake_workspace_skill = MagicMock()
        fake_workspace_skill.metadata.id = "shared_skill"
        fake_workspace_skill.tools = []

        fake_plugin_skill = MagicMock()
        fake_plugin_skill.metadata.id = "shared_skill"
        fake_plugin_skill.tools = []

        def fake_load_skills(path, names):
            if str(path) == str(plugin_skills_root):
                return [fake_plugin_skill]
            return [fake_workspace_skill]

        with patch("plugins.load_plugins", return_value=mock_plugins), \
             patch("skill_loader.loader.load_skills", side_effect=fake_load_skills), \
             patch("coordinator.get_children", return_value=[]), \
             patch("coordinator.get_parent_context", return_value=[]), \
             patch("coordinator.build_children_description", return_value=""), \
             patch("prompt.get_peer_capabilities", return_value=[]), \
             patch("prompt.build_system_prompt", return_value="system"):
            result = await adapter._common_setup(config)

        ids = [s.metadata.id for s in result.loaded_skills]
        assert ids.count("shared_skill") == 1

    @pytest.mark.asyncio
    async def test_common_setup_coordinator_prompt_appended(self, tmp_path):
        """When is_coordinator=True, build_children_description output is added to extra_prompts."""
        from unittest.mock import patch, AsyncMock as _AsyncMock

        adapter, config = self._make_test_adapter_and_config(tmp_path)

        mock_plugins = MagicMock()
        mock_plugins.plugin_names = []
        mock_plugins.skill_dirs = []
        mock_plugins.prompt_fragments = []
        mock_plugins.rules = []

        children = [{"id": "child-1", "name": "Worker", "role": "Worker", "status": "online"}]
        captured_extra_prompts = []

        def fake_build_system_prompt(*args, **kwargs):
            captured_extra_prompts.extend(kwargs.get("plugin_prompts", []))
            return "coordinator system prompt"

        fake_route_tool = MagicMock()
        fake_route_tool.name = "route_task_to_team"

        with patch("plugins.load_plugins", return_value=mock_plugins), \
             patch("skill_loader.loader.load_skills", return_value=[]), \
             patch("coordinator.get_children", return_value=children), \
             patch("coordinator.get_parent_context", return_value=[]), \
             patch("coordinator.build_children_description", return_value="## Team\n- Worker"), \
             patch("coordinator.route_task_to_team", fake_route_tool), \
             patch("prompt.get_peer_capabilities", return_value=[]), \
             patch("prompt.build_system_prompt", side_effect=fake_build_system_prompt):
            result = await adapter._common_setup(config)

        assert result.is_coordinator is True
        assert "## Team\n- Worker" in captured_extra_prompts


# ============================================================================
# BaseAdapter.get_config_schema() default implementation (line 73)
# ============================================================================

def test_base_adapter_default_get_config_schema():
    """The default get_config_schema() returns an empty dict."""
    from adapters.base import BaseAdapter, AdapterConfig

    # Create a minimal concrete subclass that does NOT override get_config_schema
    class MinimalAdapter(BaseAdapter):
        @staticmethod
        def name(): return "minimal"
        @staticmethod
        def display_name(): return "Minimal"
        @staticmethod
        def description(): return "Minimal test adapter"
        async def setup(self, config): pass
        async def create_executor(self, config): pass

    schema = MinimalAdapter.get_config_schema()
    assert schema == {}


# ============================================================================
# CrewAI _langchain_to_crewai wrapper body (lines 28-29)
# ============================================================================

def test_langchain_to_crewai_wrapper_invokes_tool(monkeypatch):
    """The sync wrapper returned by _langchain_to_crewai calls lc_tool.ainvoke."""
    from types import ModuleType
    from unittest.mock import AsyncMock, MagicMock

    # Ensure crewai is mocked so _langchain_to_crewai can import crewai.tools
    if "crewai" not in sys.modules or sys.modules.get("crewai") is None:
        crewai_mod = ModuleType("crewai")
        crewai_tools_mod = ModuleType("crewai.tools")
        # @tool decorator: returns the function unchanged
        crewai_tools_mod.tool = lambda name: (lambda f: f)
        crewai_mod.tools = crewai_tools_mod
        crewai_mod.__version__ = "0.0.0-mock"
        monkeypatch.setitem(sys.modules, "crewai", crewai_mod)
        monkeypatch.setitem(sys.modules, "crewai.tools", crewai_tools_mod)

    mock_tool = MagicMock()
    mock_tool.name = "calc_tool"
    mock_tool.description = "A calculator tool."
    mock_tool.ainvoke = AsyncMock(return_value="42")

    from adapters.crewai.adapter import _langchain_to_crewai

    wrapped = _langchain_to_crewai(mock_tool)
    # The crewai @tool mock returns the raw wrapper function unchanged,
    # so 'wrapped' IS the inner wrapper() — call it synchronously.
    result = wrapped(x=6, y=7)

    mock_tool.ainvoke.assert_called_once_with({"x": 6, "y": 7})
    assert result == "42"


# ============================================================================
# Openclaw execute() output parsing (lines 214-227, 231-232)
# ============================================================================

class TestOpenClawExecuteOutputParsing:
    """Cover the subprocess output parsing branches in OpenClawA2AExecutor.execute()."""

    def _make_proc(self, returncode, stdout_bytes, stderr_bytes=b""):
        proc = MagicMock()
        proc.returncode = returncode
        proc.kill = MagicMock()
        async def communicate():
            return stdout_bytes, stderr_bytes
        proc.communicate = communicate
        return proc

    @pytest.mark.asyncio
    async def test_execute_json_output_with_payloads(self, monkeypatch):
        """Lines 216-221: returncode=0, valid JSON with payloads list."""
        import asyncio as _asyncio
        import json as _json
        from adapters.openclaw.adapter import OpenClawA2AExecutor

        executor = OpenClawA2AExecutor()
        events = []
        event_queue = MagicMock()
        event_queue.enqueue_event = AsyncMock(side_effect=events.append)
        context = MagicMock()
        context.task_id = "t-1"

        monkeypatch.setattr("adapters.openclaw.adapter.extract_message_text", lambda ctx: "hello")
        monkeypatch.setattr("adapters.openclaw.adapter.set_current_task", AsyncMock())
        monkeypatch.setattr("adapters.openclaw.adapter.brief_task", lambda t: t)

        payload_json = _json.dumps({"result": {"payloads": [{"text": "great answer"}]}}).encode()
        proc = self._make_proc(0, payload_json)

        async def fake_create_subprocess_exec(*args, **kwargs):
            return proc

        monkeypatch.setattr(_asyncio, "create_subprocess_exec", fake_create_subprocess_exec)

        await executor.execute(context, event_queue)

        event_queue.enqueue_event.assert_awaited_once()
        # reply should be the text from payloads
        from a2a.utils import new_agent_text_message
        assert events[0] == new_agent_text_message("great answer")

    @pytest.mark.asyncio
    async def test_execute_json_output_no_payloads(self, monkeypatch):
        """Lines 222-223: returncode=0, valid JSON but empty payloads."""
        import asyncio as _asyncio
        import json as _json
        from adapters.openclaw.adapter import OpenClawA2AExecutor

        executor = OpenClawA2AExecutor()
        events = []
        event_queue = MagicMock()
        event_queue.enqueue_event = AsyncMock(side_effect=events.append)
        context = MagicMock()
        context.task_id = "t-1"

        monkeypatch.setattr("adapters.openclaw.adapter.extract_message_text", lambda ctx: "hi")
        monkeypatch.setattr("adapters.openclaw.adapter.set_current_task", AsyncMock())
        monkeypatch.setattr("adapters.openclaw.adapter.brief_task", lambda t: t)

        data = {"result": {"payloads": []}, "status": "ok"}
        proc = self._make_proc(0, _json.dumps(data).encode())

        async def fake_create_subprocess_exec(*args, **kwargs):
            return proc

        monkeypatch.setattr(_asyncio, "create_subprocess_exec", fake_create_subprocess_exec)

        await executor.execute(context, event_queue)

        event_queue.enqueue_event.assert_awaited_once()
        # reply is str(data)
        assert str(data) in str(events[0])

    @pytest.mark.asyncio
    async def test_execute_non_json_output(self, monkeypatch):
        """Line 224-225: returncode=0, output is not valid JSON."""
        import asyncio as _asyncio
        from adapters.openclaw.adapter import OpenClawA2AExecutor

        executor = OpenClawA2AExecutor()
        events = []
        event_queue = MagicMock()
        event_queue.enqueue_event = AsyncMock(side_effect=events.append)
        context = MagicMock()
        context.task_id = "t-1"

        monkeypatch.setattr("adapters.openclaw.adapter.extract_message_text", lambda ctx: "hi")
        monkeypatch.setattr("adapters.openclaw.adapter.set_current_task", AsyncMock())
        monkeypatch.setattr("adapters.openclaw.adapter.brief_task", lambda t: t)

        proc = self._make_proc(0, b"plain text output, not json")

        async def fake_create_subprocess_exec(*args, **kwargs):
            return proc

        monkeypatch.setattr(_asyncio, "create_subprocess_exec", fake_create_subprocess_exec)

        await executor.execute(context, event_queue)

        event_queue.enqueue_event.assert_awaited_once()
        assert "plain text output, not json" in str(events[0])

    @pytest.mark.asyncio
    async def test_execute_nonzero_returncode_with_stderr(self, monkeypatch):
        """Line 227: returncode!=0, includes stderr in reply."""
        import asyncio as _asyncio
        from adapters.openclaw.adapter import OpenClawA2AExecutor

        executor = OpenClawA2AExecutor()
        events = []
        event_queue = MagicMock()
        event_queue.enqueue_event = AsyncMock(side_effect=events.append)
        context = MagicMock()
        context.task_id = "t-1"

        monkeypatch.setattr("adapters.openclaw.adapter.extract_message_text", lambda ctx: "hi")
        monkeypatch.setattr("adapters.openclaw.adapter.set_current_task", AsyncMock())
        monkeypatch.setattr("adapters.openclaw.adapter.brief_task", lambda t: t)

        proc = self._make_proc(1, b"", b"some error message")

        async def fake_create_subprocess_exec(*args, **kwargs):
            return proc

        monkeypatch.setattr(_asyncio, "create_subprocess_exec", fake_create_subprocess_exec)

        await executor.execute(context, event_queue)

        event_queue.enqueue_event.assert_awaited_once()
        assert "OpenClaw error" in str(events[0])

    @pytest.mark.asyncio
    async def test_execute_generic_exception(self, monkeypatch):
        """Lines 231-232: generic Exception (not TimeoutError) is caught."""
        import asyncio as _asyncio
        from adapters.openclaw.adapter import OpenClawA2AExecutor

        executor = OpenClawA2AExecutor()
        events = []
        event_queue = MagicMock()
        event_queue.enqueue_event = AsyncMock(side_effect=events.append)
        context = MagicMock()
        context.task_id = "t-1"

        monkeypatch.setattr("adapters.openclaw.adapter.extract_message_text", lambda ctx: "hi")
        monkeypatch.setattr("adapters.openclaw.adapter.set_current_task", AsyncMock())
        monkeypatch.setattr("adapters.openclaw.adapter.brief_task", lambda t: t)

        async def fake_create_subprocess_exec(*args, **kwargs):
            raise RuntimeError("unexpected failure")

        monkeypatch.setattr(_asyncio, "create_subprocess_exec", fake_create_subprocess_exec)

        await executor.execute(context, event_queue)

        event_queue.enqueue_event.assert_awaited_once()
        assert "OpenClaw error" in str(events[0])
        assert "unexpected failure" in str(events[0])


# ============================================================================
# adapters/__init__.py: get_adapter() success path (line 41)
# ============================================================================

def test_get_adapter_valid_runtime_returns_class():
    """get_adapter() returns the adapter class when runtime is valid (line 41)."""
    from adapters import get_adapter
    from adapters.base import BaseAdapter

    # "langgraph" should always be available
    cls = get_adapter("langgraph")
    assert issubclass(cls, BaseAdapter)
    assert cls.name() == "langgraph"
