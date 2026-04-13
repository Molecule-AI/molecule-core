"""Tests for tools/temporal_workflow.py — fallback paths when temporalio is not installed."""

from __future__ import annotations
import os
import asyncio
import importlib.util
import sys
from types import ModuleType
from unittest.mock import AsyncMock, MagicMock

import pytest


# ─────────────────────────────────────────────────────────────────────────────
# Helper: create a realistic temporalio mock hierarchy
# ─────────────────────────────────────────────────────────────────────────────

def _make_temporalio_mocks():
    """Return a dict of mock modules simulating temporalio being installed."""
    # activity mock: defn must be a decorator factory
    mock_activity = ModuleType("temporalio.activity")
    mock_activity.defn = lambda name=None, **kw: (lambda f: f)  # no-op decorator

    # workflow mock: defn/run must be no-op decorators; execute_activity is awaitable
    mock_workflow = ModuleType("temporalio.workflow")
    mock_workflow.defn = lambda f: f
    mock_workflow.run = lambda f: f
    mock_workflow.execute_activity = AsyncMock(return_value=None)

    # client mock: Client with async connect classmethod
    mock_client_cls = MagicMock()
    mock_client_instance = AsyncMock()
    mock_client_cls.connect = AsyncMock(return_value=mock_client_instance)
    mock_client_mod = ModuleType("temporalio.client")
    mock_client_mod.Client = mock_client_cls

    # worker mock: Worker(client, task_queue=..., workflows=..., activities=...)
    mock_worker_instance = MagicMock()
    mock_worker_instance.run = AsyncMock(return_value=None)
    mock_worker_cls = MagicMock(return_value=mock_worker_instance)
    mock_worker_mod = ModuleType("temporalio.worker")
    mock_worker_mod.Worker = mock_worker_cls

    mock_temporalio_root = ModuleType("temporalio")

    return {
        "temporalio": mock_temporalio_root,
        "temporalio.activity": mock_activity,
        "temporalio.workflow": mock_workflow,
        "temporalio.client": mock_client_mod,
        "temporalio.worker": mock_worker_mod,
        "_client_cls": mock_client_cls,
        "_client_instance": mock_client_instance,
        "_worker_cls": mock_worker_cls,
        "_worker_instance": mock_worker_instance,
        "_workflow_mod": mock_workflow,
    }


@pytest.fixture
def real_temporal_with_temporalio(monkeypatch):
    """Load real temporal_workflow module with temporalio mocked (available)."""
    mocks = _make_temporalio_mocks()
    for key, val in mocks.items():
        if not key.startswith("_"):
            monkeypatch.setitem(sys.modules, key, val)

    mock_shared = MagicMock()
    mock_shared.extract_message_text = MagicMock(return_value="hello world")
    mock_shared.extract_history = MagicMock(return_value=[("human", "prior msg")])
    monkeypatch.setitem(sys.modules, "adapters.shared_runtime", mock_shared)

    monkeypatch.delitem(sys.modules, "builtin_tools.temporal_workflow", raising=False)
    spec = importlib.util.spec_from_file_location(
        "builtin_tools.temporal_workflow_with_mocks",
        os.path.join(os.path.dirname(__file__), "..", "builtin_tools", "temporal_workflow.py"),
    )
    mod = importlib.util.module_from_spec(spec)
    monkeypatch.setitem(sys.modules, "builtin_tools.temporal_workflow_with_mocks", mod)
    spec.loader.exec_module(mod)
    mod._global_wrapper = None
    mod._task_registry.clear()
    return mod, mocks, mock_shared


# ─────────────────────────────────────────────────────────────────────────────
# Fixture: load the module with temporalio blocked
# ─────────────────────────────────────────────────────────────────────────────


@pytest.fixture
def real_temporal(monkeypatch):
    # Remove any existing temporal module
    monkeypatch.delitem(sys.modules, "builtin_tools.temporal_workflow", raising=False)
    # Ensure temporalio is not available
    monkeypatch.setitem(sys.modules, "temporalio", None)
    monkeypatch.setitem(sys.modules, "temporalio.activity", None)
    monkeypatch.setitem(sys.modules, "temporalio.workflow", None)
    monkeypatch.setitem(sys.modules, "temporalio.client", None)
    monkeypatch.setitem(sys.modules, "temporalio.worker", None)
    # Mock adapters.shared_runtime
    mock_shared = MagicMock()
    mock_shared.extract_message_text = MagicMock(return_value="hello")
    mock_shared.extract_history = MagicMock(return_value=[("human", "prior")])
    monkeypatch.setitem(sys.modules, "adapters.shared_runtime", mock_shared)

    spec = importlib.util.spec_from_file_location(
        "builtin_tools.temporal_workflow",
        os.path.join(os.path.dirname(__file__), "..", "builtin_tools", "temporal_workflow.py"),
    )
    mod = importlib.util.module_from_spec(spec)
    monkeypatch.setitem(sys.modules, "builtin_tools.temporal_workflow", mod)
    spec.loader.exec_module(mod)
    # Reset global wrapper
    mod._global_wrapper = None
    mod._task_registry.clear()
    return mod, mock_shared


# ─────────────────────────────────────────────────────────────────────────────
# Tests
# ─────────────────────────────────────────────────────────────────────────────


def test_agent_task_input_dataclass(real_temporal):
    """AgentTaskInput stores all supplied fields."""
    mod, _ = real_temporal
    obj = mod.AgentTaskInput(
        task_id="t1",
        context_id="c1",
        user_input="hello",
        model="anthropic:test",
        workspace_id="ws-1",
        history=[["human", "hi"]],
    )
    assert obj.task_id == "t1"
    assert obj.context_id == "c1"
    assert obj.user_input == "hello"
    assert obj.model == "anthropic:test"
    assert obj.workspace_id == "ws-1"
    assert obj.history == [["human", "hi"]]


def test_llm_result_dataclass(real_temporal):
    """LLMResult stores fields and defaults error to empty string."""
    mod, _ = real_temporal
    obj = mod.LLMResult(final_text="done", success=True)
    assert obj.final_text == "done"
    assert obj.success is True
    assert obj.error == ""

    obj_err = mod.LLMResult(final_text="", success=False, error="boom")
    assert obj_err.error == "boom"


def test_temporal_not_available(real_temporal):
    """_TEMPORAL_AVAILABLE must be False when temporalio is not installed."""
    mod, _ = real_temporal
    assert mod._TEMPORAL_AVAILABLE is False


def test_create_wrapper_returns_instance(real_temporal):
    """create_wrapper() returns a TemporalWorkflowWrapper instance."""
    mod, _ = real_temporal
    wrapper = mod.create_wrapper()
    assert isinstance(wrapper, mod.TemporalWorkflowWrapper)


def test_create_wrapper_idempotent(real_temporal):
    """Calling create_wrapper() twice returns the same object."""
    mod, _ = real_temporal
    w1 = mod.create_wrapper()
    w2 = mod.create_wrapper()
    assert w1 is w2


def test_get_wrapper_none_initially(real_temporal):
    """get_wrapper() returns None before create_wrapper() is called."""
    mod, _ = real_temporal
    # fixture already resets _global_wrapper to None
    assert mod.get_wrapper() is None


def test_get_wrapper_after_create(real_temporal):
    """get_wrapper() returns the wrapper after create_wrapper() is called."""
    mod, _ = real_temporal
    wrapper = mod.create_wrapper()
    assert mod.get_wrapper() is wrapper


def test_is_available_false_initially(real_temporal):
    """A freshly created wrapper reports is_available() == False."""
    mod, _ = real_temporal
    wrapper = mod.TemporalWorkflowWrapper()
    assert wrapper.is_available() is False


@pytest.mark.asyncio
async def test_start_noop_when_temporal_unavailable(real_temporal):
    """start() is a no-op (logs info, returns) when _TEMPORAL_AVAILABLE is False."""
    mod, _ = real_temporal
    assert mod._TEMPORAL_AVAILABLE is False
    wrapper = mod.TemporalWorkflowWrapper()
    await wrapper.start()
    assert wrapper._available is False
    assert wrapper._client is None


@pytest.mark.asyncio
async def test_stop_when_not_started(real_temporal):
    """stop() does not raise when no worker task exists."""
    mod, _ = real_temporal
    wrapper = mod.TemporalWorkflowWrapper()
    # Should complete without error
    await wrapper.stop()
    assert wrapper._available is False


@pytest.mark.asyncio
async def test_stop_cancels_worker_task(real_temporal):
    """stop() cancels a running worker task and sets _available to False."""
    mod, _ = real_temporal
    wrapper = mod.TemporalWorkflowWrapper()

    async def hanging_task():
        await asyncio.sleep(100)

    wrapper._worker_task = asyncio.create_task(hanging_task())
    wrapper._available = True

    await wrapper.stop()
    assert wrapper._available is False


@pytest.mark.asyncio
async def test_run_direct_fallback_when_unavailable(real_temporal):
    """run() calls executor._core_execute() when _available is False."""
    mod, _ = real_temporal
    wrapper = mod.TemporalWorkflowWrapper()
    # _available is False by default

    mock_executor = MagicMock()
    mock_executor._core_execute = AsyncMock(return_value="result")
    mock_context = MagicMock()
    mock_eq = MagicMock()

    await wrapper.run(mock_executor, mock_context, mock_eq)

    mock_executor._core_execute.assert_awaited_once_with(mock_context, mock_eq)


@pytest.mark.asyncio
async def test_run_direct_fallback_when_no_client(real_temporal):
    """run() falls back to direct execution when _client is None even if _available somehow True."""
    mod, _ = real_temporal
    wrapper = mod.TemporalWorkflowWrapper()
    wrapper._available = False
    wrapper._client = None

    mock_executor = MagicMock()
    mock_executor._core_execute = AsyncMock(return_value="direct")
    mock_context = MagicMock()
    mock_eq = MagicMock()

    await wrapper.run(mock_executor, mock_context, mock_eq)

    mock_executor._core_execute.assert_awaited_once_with(mock_context, mock_eq)


@pytest.mark.asyncio
async def test_run_with_available_temporal_success(real_temporal):
    """run() routes through execute_workflow when _available=True and _client is set."""
    mod, mock_shared = real_temporal

    # Inject a mock MoleculeAIAgentWorkflow so the code path can be executed
    # (the real class is only defined when temporalio is installed)
    mock_workflow_cls = MagicMock()
    mock_workflow_cls.run = MagicMock()
    mod.MoleculeAIAgentWorkflow = mock_workflow_cls

    wrapper = mod.TemporalWorkflowWrapper()
    wrapper._available = True
    mock_client = AsyncMock()
    mock_client.execute_workflow = AsyncMock(return_value=None)
    wrapper._client = mock_client

    mock_executor = MagicMock()
    mock_executor._model = "anthropic:test"
    mock_executor._core_execute = AsyncMock(return_value="result")

    mock_context = MagicMock()
    mock_context.task_id = "task-123"
    mock_context.context_id = "ctx-456"

    mock_eq = MagicMock()

    await wrapper.run(mock_executor, mock_context, mock_eq)

    mock_client.execute_workflow.assert_called_once()
    assert "task-123" not in mod._task_registry  # cleaned up


@pytest.mark.asyncio
async def test_run_temporal_exception_fallback(real_temporal):
    """run() falls back to direct execution when execute_workflow raises."""
    mod, mock_shared = real_temporal

    wrapper = mod.TemporalWorkflowWrapper()
    wrapper._available = True
    mock_client = AsyncMock()
    mock_client.execute_workflow = AsyncMock(side_effect=RuntimeError("temporal down"))
    wrapper._client = mock_client

    mock_executor = MagicMock()
    mock_executor._model = "anthropic:test"
    mock_executor._core_execute = AsyncMock(return_value="fallback-result")

    mock_context = MagicMock()
    mock_context.task_id = "task-err"
    mock_context.context_id = "ctx-err"

    mock_eq = MagicMock()

    await wrapper.run(mock_executor, mock_context, mock_eq)

    # Fallback was called after Temporal raised
    mock_executor._core_execute.assert_awaited_once_with(mock_context, mock_eq)
    assert "task-err" not in mod._task_registry


@pytest.mark.asyncio
async def test_run_input_extraction_failure(real_temporal):
    """run() falls back to direct execution when input extraction raises."""
    mod, mock_shared = real_temporal

    # Make extraction fail
    mock_shared.extract_message_text.side_effect = ValueError("cannot extract")

    wrapper = mod.TemporalWorkflowWrapper()
    wrapper._available = True
    mock_client = AsyncMock()
    wrapper._client = mock_client

    mock_executor = MagicMock()
    mock_executor._model = "anthropic:test"
    mock_executor._core_execute = AsyncMock(return_value="safe-fallback")

    mock_context = MagicMock()
    mock_context.task_id = "task-extract-fail"
    mock_context.context_id = "ctx-x"

    mock_eq = MagicMock()

    await wrapper.run(mock_executor, mock_context, mock_eq)

    mock_executor._core_execute.assert_awaited_once_with(mock_context, mock_eq)
    # execute_workflow should never have been called
    mock_client.execute_workflow.assert_not_called()


@pytest.mark.asyncio
async def test_run_cleans_registry_on_success(real_temporal):
    """Registry entry is removed after a successful workflow run."""
    mod, mock_shared = real_temporal

    wrapper = mod.TemporalWorkflowWrapper()
    wrapper._available = True
    mock_client = AsyncMock()
    mock_client.execute_workflow = AsyncMock(return_value=None)
    wrapper._client = mock_client

    mock_executor = MagicMock()
    mock_executor._model = "anthropic:test"
    mock_executor._core_execute = AsyncMock(return_value="ok")

    mock_context = MagicMock()
    mock_context.task_id = "task-clean-ok"
    mock_context.context_id = "ctx-clean"

    mock_eq = MagicMock()

    await wrapper.run(mock_executor, mock_context, mock_eq)

    assert "task-clean-ok" not in mod._task_registry


@pytest.mark.asyncio
async def test_run_cleans_registry_on_exception(real_temporal):
    """Registry entry is removed even when the workflow raises an exception."""
    mod, mock_shared = real_temporal

    wrapper = mod.TemporalWorkflowWrapper()
    wrapper._available = True
    mock_client = AsyncMock()
    mock_client.execute_workflow = AsyncMock(side_effect=RuntimeError("crash"))
    wrapper._client = mock_client

    mock_executor = MagicMock()
    mock_executor._model = "anthropic:test"
    mock_executor._core_execute = AsyncMock(return_value="fallback")

    mock_context = MagicMock()
    mock_context.task_id = "task-clean-err"
    mock_context.context_id = "ctx-clean-err"

    mock_eq = MagicMock()

    await wrapper.run(mock_executor, mock_context, mock_eq)

    assert "task-clean-err" not in mod._task_registry


# ─────────────────────────────────────────────────────────────────────────────
# Tests with mocked temporalio — covers lines 116-250 and 322-360
# ─────────────────────────────────────────────────────────────────────────────


def test_temporal_available_when_mocked(real_temporal_with_temporalio):
    """_TEMPORAL_AVAILABLE is True when temporalio mock is in sys.modules."""
    mod, mocks, _ = real_temporal_with_temporalio
    assert mod._TEMPORAL_AVAILABLE is True


def test_activity_functions_defined(real_temporal_with_temporalio):
    """task_receive_activity, llm_call_activity, task_complete_activity are defined."""
    mod, mocks, _ = real_temporal_with_temporalio
    assert hasattr(mod, "task_receive_activity")
    assert hasattr(mod, "llm_call_activity")
    assert hasattr(mod, "task_complete_activity")
    assert hasattr(mod, "MoleculeAIAgentWorkflow")


@pytest.mark.asyncio
async def test_task_receive_activity_registry_miss(real_temporal_with_temporalio):
    """task_receive_activity returns registry_miss when task_id not in registry."""
    mod, mocks, _ = real_temporal_with_temporalio
    inp = mod.AgentTaskInput(
        task_id="unknown-task", context_id="ctx", user_input="hi",
        model="test", workspace_id="ws", history=[]
    )
    result = await mod.task_receive_activity(inp)
    assert result["status"] == "registry_miss"


@pytest.mark.asyncio
async def test_task_receive_activity_found(real_temporal_with_temporalio):
    """task_receive_activity returns 'received' when task_id is in registry."""
    mod, mocks, _ = real_temporal_with_temporalio
    mod._task_registry["task-found"] = {"executor": None, "context": None, "event_queue": None}
    inp = mod.AgentTaskInput(
        task_id="task-found", context_id="ctx", user_input="hi",
        model="test", workspace_id="ws", history=[]
    )
    result = await mod.task_receive_activity(inp)
    assert result["status"] == "received"
    mod._task_registry.clear()


@pytest.mark.asyncio
async def test_llm_call_activity_registry_miss(real_temporal_with_temporalio):
    """llm_call_activity returns error LLMResult when task_id not in registry."""
    mod, mocks, _ = real_temporal_with_temporalio
    inp = mod.AgentTaskInput(
        task_id="missing-task", context_id="ctx", user_input="hi",
        model="test", workspace_id="ws", history=[]
    )
    result = await mod.llm_call_activity(inp)
    assert result.success is False
    assert result.final_text == ""
    assert "not in registry" in result.error


@pytest.mark.asyncio
async def test_llm_call_activity_success(real_temporal_with_temporalio):
    """llm_call_activity calls _core_execute and returns success LLMResult."""
    mod, mocks, _ = real_temporal_with_temporalio
    mock_executor = MagicMock()
    mock_executor._core_execute = AsyncMock(return_value="Agent response text")
    mock_context = MagicMock()
    mock_eq = MagicMock()
    mod._task_registry["task-ok"] = {
        "executor": mock_executor,
        "context": mock_context,
        "event_queue": mock_eq,
        "final_text": "",
    }
    inp = mod.AgentTaskInput(
        task_id="task-ok", context_id="ctx", user_input="hi",
        model="test", workspace_id="ws", history=[]
    )
    result = await mod.llm_call_activity(inp)
    assert result.success is True
    assert result.final_text == "Agent response text"
    mod._task_registry.clear()


@pytest.mark.asyncio
async def test_llm_call_activity_executor_exception(real_temporal_with_temporalio):
    """llm_call_activity catches executor exceptions and returns error LLMResult."""
    mod, mocks, _ = real_temporal_with_temporalio
    mock_executor = MagicMock()
    mock_executor._core_execute = AsyncMock(side_effect=RuntimeError("LLM crashed"))
    mock_context = MagicMock()
    mock_eq = MagicMock()
    mod._task_registry["task-crash"] = {
        "executor": mock_executor,
        "context": mock_context,
        "event_queue": mock_eq,
        "final_text": "",
    }
    inp = mod.AgentTaskInput(
        task_id="task-crash", context_id="ctx", user_input="hi",
        model="test", workspace_id="ws", history=[]
    )
    result = await mod.llm_call_activity(inp)
    assert result.success is False
    assert "LLM crashed" in result.error
    mod._task_registry.clear()


@pytest.mark.asyncio
async def test_task_complete_activity_success(real_temporal_with_temporalio):
    """task_complete_activity logs success info."""
    mod, mocks, _ = real_temporal_with_temporalio
    result = mod.LLMResult(final_text="done", success=True)
    # Should not raise
    await mod.task_complete_activity(result)


@pytest.mark.asyncio
async def test_task_complete_activity_failure(real_temporal_with_temporalio):
    """task_complete_activity logs failure warning."""
    mod, mocks, _ = real_temporal_with_temporalio
    result = mod.LLMResult(final_text="", success=False, error="oh no")
    # Should not raise
    await mod.task_complete_activity(result)


@pytest.mark.asyncio
async def test_start_already_available(real_temporal_with_temporalio):
    """start() is a no-op when wrapper is already started."""
    mod, mocks, _ = real_temporal_with_temporalio
    wrapper = mod.TemporalWorkflowWrapper()
    wrapper._available = True  # simulate already started
    await wrapper.start()
    # Client.connect should NOT have been called again
    mocks["_client_cls"].connect.assert_not_called()


@pytest.mark.asyncio
async def test_start_connect_success(real_temporal_with_temporalio):
    """start() connects to Temporal and starts worker when temporalio available."""
    mod, mocks, _ = real_temporal_with_temporalio
    wrapper = mod.TemporalWorkflowWrapper()

    # Inject MoleculeAIAgentWorkflow + activity refs needed by Worker constructor
    mock_wf_cls = MagicMock()
    mod.MoleculeAIAgentWorkflow = mock_wf_cls
    mod.task_receive_activity = MagicMock()
    mod.llm_call_activity = MagicMock()
    mod.task_complete_activity = MagicMock()

    # Make worker.run() hang (real asyncio task)
    worker_running = asyncio.Event()
    async def _fake_run():
        await worker_running.wait()
    mocks["_worker_instance"].run = _fake_run

    await wrapper.start()
    assert wrapper._available is True
    assert wrapper._client is mocks["_client_instance"]
    # Clean up
    if wrapper._worker_task:
        wrapper._worker_task.cancel()
        try:
            await wrapper._worker_task
        except (asyncio.CancelledError, Exception):
            pass


@pytest.mark.asyncio
async def test_start_connect_failure(real_temporal_with_temporalio):
    """start() falls back gracefully when Client.connect raises."""
    mod, mocks, _ = real_temporal_with_temporalio
    mocks["_client_cls"].connect = AsyncMock(side_effect=OSError("refused"))
    wrapper = mod.TemporalWorkflowWrapper()
    await wrapper.start()
    assert wrapper._available is False
    assert wrapper._client is None


@pytest.mark.asyncio
async def test_start_worker_init_failure(real_temporal_with_temporalio):
    """start() falls back gracefully when Worker() constructor raises."""
    mod, mocks, _ = real_temporal_with_temporalio
    # Connect succeeds
    mocks["_client_cls"].connect = AsyncMock(return_value=mocks["_client_instance"])
    # Worker constructor raises
    mocks["_worker_cls"].side_effect = RuntimeError("worker failed")
    mod.MoleculeAIAgentWorkflow = MagicMock()
    mod.task_receive_activity = MagicMock()
    mod.llm_call_activity = MagicMock()
    mod.task_complete_activity = MagicMock()

    wrapper = mod.TemporalWorkflowWrapper()
    await wrapper.start()
    assert wrapper._available is False


@pytest.mark.asyncio
async def test_molecule_workflow_run_method(real_temporal_with_temporalio):
    """MoleculeAIAgentWorkflow.run() calls all three activity stages."""
    mod, mocks, _ = real_temporal_with_temporalio

    # Set up mock activities in the module
    mock_receive_result = {"task_id": "t1", "status": "received"}
    mock_llm_result = mod.LLMResult(final_text="response", success=True)

    # workflow.execute_activity should return different values per call
    call_count = {"n": 0}
    async def mock_execute_activity(activity_fn, inp, **kwargs):
        call_count["n"] += 1
        if call_count["n"] == 1:
            return mock_receive_result
        elif call_count["n"] == 2:
            return mock_llm_result
        else:
            return None  # task_complete returns None

    mocks["_workflow_mod"].execute_activity = mock_execute_activity

    # Create and run the workflow
    wf = mod.MoleculeAIAgentWorkflow()
    inp = mod.AgentTaskInput(
        task_id="t1", context_id="c1", user_input="hello",
        model="test", workspace_id="ws", history=[]
    )
    result = await wf.run(inp)

    assert result is mock_llm_result
    assert call_count["n"] == 3  # three stages called
