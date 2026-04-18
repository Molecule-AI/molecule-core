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


# ─────────────────────────────────────────────────────────────────────────────
# Issue #790 — Case 6: Non-fatal checkpoint failure
#
# _save_checkpoint() is called from task_receive_activity and llm_call_activity
# after their main work completes. If the HTTP POST to the platform returns an
# error status (e.g. 500 Internal Server Error) or raises a network exception,
# the activity must NOT propagate the error — the workflow continues normally.
# ─────────────────────────────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_save_checkpoint_failure_is_nonfatal_on_http_error(
    real_temporal_with_temporalio, monkeypatch
):
    """_save_checkpoint raises httpx.HTTPStatusError (500) → activity succeeds.

    Injects a checkpoint endpoint failure into task_receive_activity by patching
    _save_checkpoint to raise an HTTPStatusError.  The activity must return
    normally with status='received' regardless.
    """
    mod, _mocks, _mock_shared = real_temporal_with_temporalio

    # Track whether the mock was called.
    save_calls: list[dict] = []

    async def _fail_checkpoint(workspace_id, workflow_id, step_name, step_index, payload=None):
        save_calls.append({
            "workspace_id": workspace_id,
            "workflow_id": workflow_id,
            "step_name": step_name,
            "step_index": step_index,
            "payload": payload,
        })
        # Simulate HTTP 500 from the platform checkpoint endpoint.
        import httpx as _httpx
        request = _httpx.Request("POST", "http://localhost:8080/workspaces/ws-1/checkpoints")
        response = _httpx.Response(500, request=request, text="Internal Server Error")
        raise _httpx.HTTPStatusError("500", request=request, response=response)

    monkeypatch.setattr(mod, "_save_checkpoint", _fail_checkpoint)

    # Register a minimal task entry so the activity doesn't take the registry-miss path.
    task_id = "t-nonfatal-ckpt"
    mod._task_registry[task_id] = {
        "executor": None,
        "context": None,
        "event_queue": None,
        "final_text": "",
    }

    inp = mod.AgentTaskInput(
        task_id=task_id,
        context_id="ctx-1",
        user_input="hello",
        model="test-model",
        workspace_id="ws-1",
        history=[],
    )

    # Act: call task_receive_activity directly.  It should succeed despite
    # _save_checkpoint raising HTTPStatusError.
    result = await mod.task_receive_activity(inp)

    # Assert: activity returned successfully — checkpoint failure was swallowed.
    assert result == {"task_id": task_id, "status": "received"}, (
        f"task_receive_activity must succeed even when checkpoint POST fails; "
        f"got {result!r}"
    )
    # The checkpoint attempt was made (once, for task_receive).
    assert len(save_calls) == 1
    assert save_calls[0]["step_name"] == "task_receive"
    assert save_calls[0]["step_index"] == 0

    # Cleanup registry.
    mod._task_registry.pop(task_id, None)


@pytest.mark.asyncio
async def test_save_checkpoint_failure_is_nonfatal_on_network_error(
    real_temporal_with_temporalio, monkeypatch
):
    """_save_checkpoint raises a generic network error → llm_call_activity succeeds.

    Tests the llm_call_activity path: even if _save_checkpoint raises a
    ConnectError (network unreachable), the activity returns its LLMResult.
    """
    mod, _mocks, _mock_shared = real_temporal_with_temporalio

    save_calls: list[str] = []

    async def _network_fail_checkpoint(
        workspace_id, workflow_id, step_name, step_index, payload=None
    ):
        save_calls.append(step_name)
        import httpx as _httpx
        raise _httpx.ConnectError("Connection refused")

    monkeypatch.setattr(mod, "_save_checkpoint", _network_fail_checkpoint)

    # Build a mock executor whose _core_execute returns a known string.
    mock_executor = MagicMock()
    mock_executor._core_execute = AsyncMock(return_value="workflow output")
    mock_context = MagicMock()
    mock_event_queue = MagicMock()

    task_id = "t-network-fail"
    mod._task_registry[task_id] = {
        "executor": mock_executor,
        "context": mock_context,
        "event_queue": mock_event_queue,
        "final_text": "",
    }

    inp = mod.AgentTaskInput(
        task_id=task_id,
        context_id="ctx-2",
        user_input="test",
        model="test-model",
        workspace_id="ws-2",
        history=[],
    )

    # Act: llm_call_activity must complete successfully.
    result = await mod.llm_call_activity(inp)

    # Assert: successful LLMResult returned despite checkpoint ConnectError.
    assert isinstance(result, mod.LLMResult), f"Expected LLMResult, got {type(result)}"
    assert result.success is True, f"llm_call must succeed when checkpoint fails; got {result!r}"
    assert result.final_text == "workflow output"
    # _core_execute was called (actual work happened).
    mock_executor._core_execute.assert_awaited_once_with(mock_context, mock_event_queue)
    # Checkpoint was attempted (once, for llm_call at step_index=1).
    assert "llm_call" in save_calls

    mod._task_registry.pop(task_id, None)


@pytest.mark.asyncio
async def test_save_checkpoint_success_path(
    real_temporal_with_temporalio, monkeypatch
):
    """When _save_checkpoint succeeds, activity returns correctly and checkpoint is recorded.

    Verifies the happy path: checkpoint is called with the right arguments and
    the activity return value is unaffected by a successful checkpoint save.
    """
    mod, _mocks, _mock_shared = real_temporal_with_temporalio

    save_calls: list[dict] = []

    async def _noop_checkpoint(workspace_id, workflow_id, step_name, step_index, payload=None):
        save_calls.append({
            "workspace_id": workspace_id,
            "workflow_id": workflow_id,
            "step_name": step_name,
            "step_index": step_index,
            "payload": payload,
        })

    monkeypatch.setattr(mod, "_save_checkpoint", _noop_checkpoint)

    task_id = "t-success-ckpt"
    mod._task_registry[task_id] = {
        "executor": None,
        "context": None,
        "event_queue": None,
        "final_text": "",
    }

    inp = mod.AgentTaskInput(
        task_id=task_id,
        context_id="ctx-3",
        user_input="hi",
        model="test-model",
        workspace_id="ws-3",
        history=[],
    )

    result = await mod.task_receive_activity(inp)

    assert result == {"task_id": task_id, "status": "received"}
    assert len(save_calls) == 1
    assert save_calls[0]["workspace_id"] == "ws-3"
    assert save_calls[0]["workflow_id"] == task_id
    assert save_calls[0]["step_name"] == "task_receive"
    assert save_calls[0]["step_index"] == 0

    mod._task_registry.pop(task_id, None)


@pytest.mark.asyncio
async def test_save_checkpoint_standalone_http_error_is_swallowed(
    real_temporal_with_temporalio, monkeypatch
):
    """_save_checkpoint() itself swallows HTTP errors — direct call test.

    Calls the real _save_checkpoint function (patching httpx.AsyncClient)
    and asserts it returns None without raising even when the platform
    returns a 500 status.
    """
    import httpx as _httpx

    mod, _mocks, _mock_shared = real_temporal_with_temporalio

    # Patch platform_auth to avoid disk reads in the test environment.
    mock_platform_auth = MagicMock()
    mock_platform_auth.auth_headers = MagicMock(return_value={"Authorization": "Bearer test-tok"})
    monkeypatch.setitem(
        __import__("sys").modules, "platform_auth", mock_platform_auth
    )

    # Simulate the AsyncClient.post returning a 500.
    mock_response = MagicMock()
    mock_response.raise_for_status.side_effect = _httpx.HTTPStatusError(
        "500",
        request=_httpx.Request("POST", "http://localhost:8080/workspaces/ws-x/checkpoints"),
        response=_httpx.Response(500),
    )

    mock_client = AsyncMock()
    mock_client.__aenter__ = AsyncMock(return_value=mock_client)
    mock_client.__aexit__ = AsyncMock(return_value=False)
    mock_client.post = AsyncMock(return_value=mock_response)

    with monkeypatch.context() as m:
        m.setattr(_httpx, "AsyncClient", MagicMock(return_value=mock_client))

        # Must NOT raise — non-fatal contract.
        result = await mod._save_checkpoint(
            workspace_id="ws-x",
            workflow_id="wf-x",
            step_name="task_receive",
            step_index=0,
            payload={"task_id": "t-x"},
        )

    assert result is None, "_save_checkpoint must return None (no exception) on HTTP 500"


# ─────────────────────────────────────────────────────────────────────────────
# _fetch_latest_checkpoint — unit tests (issue #837)
# ─────────────────────────────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_fetch_latest_checkpoint_returns_none_on_404(
    real_temporal_with_temporalio, monkeypatch
):
    """_fetch_latest_checkpoint returns None when the platform responds 404.

    404 is the expected response for a freshly provisioned workspace that has
    never completed a checkpoint.  The caller must not crash.
    """
    import httpx as _httpx

    mod, _mocks, _mock_shared = real_temporal_with_temporalio

    mock_platform_auth = MagicMock()
    mock_platform_auth.auth_headers = MagicMock(return_value={"Authorization": "Bearer tok"})
    monkeypatch.setitem(__import__("sys").modules, "platform_auth", mock_platform_auth)

    mock_response = MagicMock()
    mock_response.status_code = 404

    mock_client = AsyncMock()
    mock_client.__aenter__ = AsyncMock(return_value=mock_client)
    mock_client.__aexit__ = AsyncMock(return_value=False)
    mock_client.get = AsyncMock(return_value=mock_response)

    with monkeypatch.context() as m:
        m.setattr(_httpx, "AsyncClient", MagicMock(return_value=mock_client))
        result = await mod._fetch_latest_checkpoint("ws-404")

    assert result is None, "404 from platform must return None (non-fatal)"


@pytest.mark.asyncio
async def test_fetch_latest_checkpoint_returns_dict_on_200(
    real_temporal_with_temporalio, monkeypatch
):
    """_fetch_latest_checkpoint returns the parsed JSON dict on a 200 OK."""
    import httpx as _httpx

    mod, _mocks, _mock_shared = real_temporal_with_temporalio

    mock_platform_auth = MagicMock()
    mock_platform_auth.auth_headers = MagicMock(return_value={"Authorization": "Bearer tok"})
    monkeypatch.setitem(__import__("sys").modules, "platform_auth", mock_platform_auth)

    checkpoint_payload = {
        "id": "ckpt-1",
        "workspace_id": "ws-200",
        "workflow_id": "wf-abc",
        "step_name": "llm_call",
        "step_index": 1,
        "completed_at": "2026-04-18T10:00:00Z",
        "payload": None,
    }

    mock_response = MagicMock()
    mock_response.status_code = 200
    mock_response.raise_for_status = MagicMock()  # no-op
    mock_response.json = MagicMock(return_value=checkpoint_payload)

    mock_client = AsyncMock()
    mock_client.__aenter__ = AsyncMock(return_value=mock_client)
    mock_client.__aexit__ = AsyncMock(return_value=False)
    mock_client.get = AsyncMock(return_value=mock_response)

    with monkeypatch.context() as m:
        m.setattr(_httpx, "AsyncClient", MagicMock(return_value=mock_client))
        result = await mod._fetch_latest_checkpoint("ws-200")

    assert result == checkpoint_payload, "200 OK should return the parsed checkpoint dict"
    assert result["step_name"] == "llm_call"
    assert result["workflow_id"] == "wf-abc"


@pytest.mark.asyncio
async def test_fetch_latest_checkpoint_swallows_exceptions(
    real_temporal_with_temporalio, monkeypatch
):
    """_fetch_latest_checkpoint returns None and does NOT raise on network error.

    Non-fatal contract: a transient network failure or misconfiguration must
    never propagate to the caller — the workflow should start fresh instead.
    """
    import httpx as _httpx

    mod, _mocks, _mock_shared = real_temporal_with_temporalio

    mock_platform_auth = MagicMock()
    mock_platform_auth.auth_headers = MagicMock(return_value={"Authorization": "Bearer tok"})
    monkeypatch.setitem(__import__("sys").modules, "platform_auth", mock_platform_auth)

    mock_client = AsyncMock()
    mock_client.__aenter__ = AsyncMock(return_value=mock_client)
    mock_client.__aexit__ = AsyncMock(return_value=False)
    mock_client.get = AsyncMock(
        side_effect=_httpx.ConnectError("connection refused")
    )

    with monkeypatch.context() as m:
        m.setattr(_httpx, "AsyncClient", MagicMock(return_value=mock_client))
        result = await mod._fetch_latest_checkpoint("ws-err")

    assert result is None, "network error must be swallowed — non-fatal contract"


@pytest.mark.asyncio
async def test_execute_injects_checkpoint_into_history(
    real_temporal_with_temporalio, monkeypatch
):
    """execute() prepends a [system, ...] checkpoint note to AgentTaskInput.history.

    When _fetch_latest_checkpoint returns a checkpoint dict, the wrapper must
    prepend a synthetic system context entry to the serialised history before
    submitting the Temporal workflow.  The injected entry starts with '[SYSTEM:'
    and contains the workflow_id and step_name from the checkpoint.
    """
    mod, mocks, mock_shared = real_temporal_with_temporalio

    # Patch _fetch_latest_checkpoint to return a preset checkpoint
    fake_ckpt = {
        "id": "ckpt-inject",
        "workspace_id": "ws-inject",
        "workflow_id": "wf-prev",
        "step_name": "task_receive",
        "step_index": 0,
        "completed_at": "2026-04-18T09:00:00Z",
    }
    monkeypatch.setattr(mod, "_fetch_latest_checkpoint", AsyncMock(return_value=fake_ckpt))
    monkeypatch.setenv("WORKSPACE_ID", "ws-inject")

    # Wire a TemporalWorkflowWrapper in available mode with the mock client
    client_instance = mocks["_client_instance"]
    client_instance.execute_workflow = AsyncMock(return_value=None)

    wrapper = mod.TemporalWorkflowWrapper.__new__(mod.TemporalWorkflowWrapper)
    wrapper._available = True
    wrapper._client = client_instance

    # Minimal mock executor and context
    executor = MagicMock()
    executor._model = "claude-3-5-sonnet-20241022"
    executor._core_execute = AsyncMock()

    context = MagicMock()
    context.task_id = "t-inject"
    context.context_id = "ctx-inject"

    event_queue = MagicMock()

    # shared_runtime mocks already set via fixture:
    #   extract_message_text → "hello world"
    #   extract_history → [("human", "prior msg")]

    await wrapper.run(executor, context, event_queue)

    assert client_instance.execute_workflow.called, "execute_workflow must be called"

    # The second positional arg to execute_workflow is the AgentTaskInput
    call_args = client_instance.execute_workflow.call_args
    inp = call_args[0][1]  # positional args[1]

    assert isinstance(inp, mod.AgentTaskInput)
    assert len(inp.history) >= 2, "history must have at least the injected note + original entry"

    system_entry = inp.history[0]
    assert system_entry[0] == "system", "first history entry must be a system message"
    assert "[SYSTEM:" in system_entry[1], "injected note must start with [SYSTEM:"
    assert "wf-prev" in system_entry[1], "injected note must include the prior workflow_id"
    assert "task_receive" in system_entry[1], "injected note must include the last step_name"

    # Original history entries must still follow the injected system note
    assert inp.history[1] == ["human", "prior msg"], "original history must be preserved after injection"
