"""Tests for the shared _common_setup() pipeline and tool conversion helpers."""

import importlib.util
import sys
from types import ModuleType
from unittest.mock import AsyncMock, MagicMock, patch

import pytest


# --- Mock missing optional deps ---

def _ensure_crewai_mock():
    if "crewai" not in sys.modules:
        crewai_mod = ModuleType("crewai")
        crewai_tools_mod = ModuleType("crewai.tools")
        # Make @tool a passthrough decorator that preserves the function
        crewai_tools_mod.tool = lambda name: (lambda f: f)
        crewai_mod.tools = crewai_tools_mod
        crewai_mod.__version__ = "0.0.0-mock"
        sys.modules["crewai"] = crewai_mod
        sys.modules["crewai.tools"] = crewai_tools_mod


def _ensure_autogen_mock():
    if "autogen_agentchat" not in sys.modules:
        mod = ModuleType("autogen_agentchat")
        agents_mod = ModuleType("autogen_agentchat.agents")
        agents_mod.AssistantAgent = MagicMock
        mod.agents = agents_mod
        sys.modules["autogen_agentchat"] = mod
        sys.modules["autogen_agentchat.agents"] = agents_mod


_ensure_crewai_mock()
_ensure_autogen_mock()


# --- Mock helpers ---

def _mock_load_plugins(**kwargs):
    plugins = MagicMock()
    plugins.plugin_names = []
    plugins.skill_dirs = []
    plugins.prompt_fragments = []
    plugins.rules = []
    return plugins


def _mock_load_skills(config_path, tools):
    return []


async def _mock_get_children():
    return []


async def _mock_get_children_with_kids():
    return [{"id": "child-1", "name": "Child", "role": "Worker", "status": "online"}]


async def _mock_get_parent_context():
    return []


async def _mock_get_peer_capabilities(platform_url, workspace_id):
    return [{"id": "peer-1", "name": "Peer", "status": "online", "agent_card": {"skills": []}}]


def _mock_build_system_prompt(*args, **kwargs):
    return "You are a test agent."


def _mock_build_children_description(children):
    return "## Team\n- Child: Worker"


# All patches needed for _common_setup
_SETUP_PATCHES = {
    "plugins.load_plugins": _mock_load_plugins,
    "skill_loader.loader.load_skills": _mock_load_skills,
    "coordinator.get_children": _mock_get_children,
    "coordinator.get_parent_context": _mock_get_parent_context,
    "coordinator.build_children_description": _mock_build_children_description,
    "prompt.get_peer_capabilities": _mock_get_peer_capabilities,
    "prompt.build_system_prompt": _mock_build_system_prompt,
}


def _make_test_adapter():
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

    return TestAdapter(), AdapterConfig(model="openai:test", config_path="/tmp", workspace_id="ws-test")


# --- Common Setup Tests ---

@pytest.mark.asyncio
async def test_common_setup_returns_core_tools():
    """_common_setup returns 5 core tools."""
    adapter, config = _make_test_adapter()

    patches = {k: v for k, v in _SETUP_PATCHES.items()}
    with patch.dict("os.environ", {"PLATFORM_URL": "http://test:8080"}):
        ctx = [patch(k, v) for k, v in patches.items()]
        for c in ctx:
            c.start()
        try:
            result = await adapter._common_setup(config)
        finally:
            for c in ctx:
                c.stop()

    assert len(result.langchain_tools) == 6  # 6 core tools
    tool_names = [t.name for t in result.langchain_tools]
    assert "delegate_to_workspace" in tool_names
    assert "check_delegation_status" in tool_names
    assert "request_approval" in tool_names
    assert "commit_memory" in tool_names
    assert "search_memory" in tool_names
    assert "run_code" in tool_names
    assert result.system_prompt == "You are a test agent."
    assert result.is_coordinator is False


@pytest.mark.asyncio
async def test_common_setup_coordinator_adds_routing_tool():
    """When workspace has children, coordinator tool is added."""
    adapter, config = _make_test_adapter()

    patches = {k: v for k, v in _SETUP_PATCHES.items()}
    patches["coordinator.get_children"] = _mock_get_children_with_kids

    with patch.dict("os.environ", {"PLATFORM_URL": "http://test:8080"}):
        ctx = [patch(k, v) for k, v in patches.items()]
        for c in ctx:
            c.start()
        try:
            result = await adapter._common_setup(config)
        finally:
            for c in ctx:
                c.stop()

    assert result.is_coordinator is True
    assert len(result.langchain_tools) == 7  # 6 core + route_task_to_team
    # Last tool should be route_task_to_team (function name or .name attribute)
    last_tool = result.langchain_tools[-1]
    tool_id = getattr(last_tool, "name", None) or getattr(last_tool, "__name__", "")
    assert "route_task_to_team" in tool_id


# --- Tool Conversion Tests ---

def test_langchain_to_crewai_preserves_name():
    """CrewAI wrapper preserves tool name and description."""
    from adapters.crewai.adapter import _langchain_to_crewai

    mock_tool = MagicMock()
    mock_tool.name = "test_tool"
    mock_tool.description = "A test tool for testing."
    mock_tool.ainvoke = AsyncMock(return_value={"result": "ok"})

    wrapped = _langchain_to_crewai(mock_tool)
    # With our mock @tool decorator, the wrapper is the raw function
    assert wrapped.__doc__ == "A test tool for testing."


@pytest.mark.skipif(
    not importlib.util.find_spec("autogen_core"),
    reason="autogen_core not installed",
)
def test_langchain_to_autogen_preserves_name():
    """AutoGen wrapper preserves tool name and description via FunctionTool."""
    from adapters.autogen.adapter import _langchain_to_autogen

    mock_tool = MagicMock()
    mock_tool.name = "test_tool"
    mock_tool.description = "A test tool for testing."
    mock_tool.ainvoke = AsyncMock(return_value={"result": "ok"})

    wrapped = _langchain_to_autogen(mock_tool)
    assert wrapped.name == "test_tool"
    assert wrapped.description == "A test tool for testing."


@pytest.mark.skipif(
    not importlib.util.find_spec("autogen_core"),
    reason="autogen_core not installed",
)
@pytest.mark.asyncio
async def test_langchain_to_autogen_calls_ainvoke():
    """AutoGen FunctionTool wrapper calls the original tool's ainvoke."""
    from adapters.autogen.adapter import _langchain_to_autogen

    mock_tool = MagicMock()
    mock_tool.name = "delegate"
    mock_tool.description = "Delegate a task."
    mock_tool.ainvoke = AsyncMock(return_value={"success": True})

    wrapped = _langchain_to_autogen(mock_tool)
    # FunctionTool.run_json expects a JSON dict with the function params
    result = await wrapped.run_json({"input": '{"workspace_id": "ws-1", "task": "do stuff"}'}, cancellation_token=None)
    mock_tool.ainvoke.assert_called_once_with({"workspace_id": "ws-1", "task": "do stuff"})
    assert "True" in str(result)
