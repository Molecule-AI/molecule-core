"""Tests for tools/delegation.py (async delegation model).

The delegation tool now returns immediately with a task_id and runs the
A2A request in the background. Tests verify:
1. Immediate return with task_id
2. Background task completion
3. check_delegation_status retrieval
4. Error handling (RBAC, discovery, network)
"""

import asyncio
import importlib.util
import os
import sys
from unittest.mock import AsyncMock, MagicMock, patch

import httpx
import pytest


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _make_mock_client(
    discover_status=200,
    discover_payload=None,
    discover_exc=None,
    a2a_status=200,
    a2a_payload=None,
):
    """Return (mock_client, mock_client_class) for patching httpx.AsyncClient."""
    if discover_payload is None:
        discover_payload = {"url": "http://peer:8000"}
    if a2a_payload is None:
        a2a_payload = {
            "result": {
                "parts": [{"kind": "text", "text": "done"}],
                "artifacts": [],
            }
        }

    mock_resp_discover = MagicMock()
    mock_resp_discover.status_code = discover_status
    mock_resp_discover.json.return_value = discover_payload

    mock_resp_a2a = MagicMock()
    mock_resp_a2a.status_code = a2a_status
    mock_resp_a2a.json.return_value = a2a_payload

    mock_client = AsyncMock()
    if discover_exc:
        mock_client.get = AsyncMock(side_effect=discover_exc)
    else:
        mock_client.get = AsyncMock(return_value=mock_resp_discover)
    mock_client.post = AsyncMock(return_value=mock_resp_a2a)

    mock_cls = MagicMock()
    mock_cls.return_value.__aenter__ = AsyncMock(return_value=mock_client)
    mock_cls.return_value.__aexit__ = AsyncMock(return_value=False)

    return mock_client, mock_cls


@pytest.fixture
def delegation_mocks(monkeypatch):
    """Load the real delegation module with mocked dependencies."""
    mock_audit = MagicMock()
    mock_audit.check_permission = MagicMock(return_value=True)
    mock_audit.get_workspace_roles = MagicMock(return_value=(["operator"], {}))
    mock_audit.log_event = MagicMock()

    mock_span = MagicMock()
    mock_span.set_attribute = MagicMock()
    mock_span.record_exception = MagicMock()
    mock_span.__enter__ = MagicMock(return_value=mock_span)
    mock_span.__exit__ = MagicMock(return_value=False)

    mock_tracer = MagicMock()
    mock_tracer.start_as_current_span = MagicMock(return_value=mock_span)

    mock_telemetry = MagicMock()
    mock_telemetry.get_tracer = MagicMock(return_value=mock_tracer)
    mock_telemetry.inject_trace_headers = MagicMock(side_effect=lambda h: h)
    mock_telemetry.get_current_traceparent = MagicMock(return_value="")
    for attr in ["A2A_SOURCE_WORKSPACE", "A2A_TARGET_WORKSPACE", "A2A_TASK_ID", "WORKSPACE_ID_ATTR"]:
        setattr(mock_telemetry, attr, attr)

    monkeypatch.setitem(sys.modules, "builtin_tools.audit", mock_audit)
    monkeypatch.setitem(sys.modules, "builtin_tools.telemetry", mock_telemetry)
    monkeypatch.setenv("WORKSPACE_ID", "ws-self")
    monkeypatch.setenv("PLATFORM_URL", "http://test:8080")

    spec = importlib.util.spec_from_file_location(
        "builtin_tools.delegation",
        os.path.join(os.path.dirname(__file__), "..", "builtin_tools", "delegation.py"),
    )
    mod = importlib.util.module_from_spec(spec)
    monkeypatch.setitem(sys.modules, "builtin_tools.delegation", mod)
    spec.loader.exec_module(mod)

    mod.DELEGATION_RETRY_ATTEMPTS = 2
    mod.DELEGATION_RETRY_DELAY = 0.0
    # Clear state between tests
    mod._delegations.clear()
    mod._background_tasks.clear()

    return mod, mock_audit, mock_telemetry, mock_span


async def _invoke(mod, workspace_id="target", task="do stuff"):
    """Call delegate_to_workspace and return the immediate result."""
    fn = mod.delegate_to_workspace
    if hasattr(fn, "ainvoke"):
        return await fn.ainvoke({"workspace_id": workspace_id, "task": task})
    return await fn(workspace_id=workspace_id, task=task)


async def _invoke_and_wait(mod, workspace_id="target", task="do stuff"):
    """Call delegate_to_workspace, wait for background task, return status."""
    result = await _invoke(mod, workspace_id, task)
    # Wait for all background tasks to complete
    if mod._background_tasks:
        await asyncio.gather(*mod._background_tasks, return_exceptions=True)
    # Get final status
    if "task_id" in result:
        fn = mod.check_delegation_status
        if hasattr(fn, "ainvoke"):
            return await fn.ainvoke({"task_id": result["task_id"]})
        return await fn(task_id=result["task_id"])
    return result


# ---------------------------------------------------------------------------
# Tests
# ---------------------------------------------------------------------------

class TestRBAC:

    @pytest.mark.asyncio
    async def test_rbac_deny(self, delegation_mocks):
        mod, mock_audit, *_ = delegation_mocks
        mock_audit.check_permission.return_value = False

        result = await _invoke(mod)

        assert result["success"] is False
        assert "RBAC" in result["error"]


class TestAsyncDelegation:

    @pytest.mark.asyncio
    async def test_returns_immediately_with_task_id(self, delegation_mocks):
        mod, *_ = delegation_mocks
        _, mock_cls = _make_mock_client()

        with patch("httpx.AsyncClient", mock_cls):
            result = await _invoke(mod)

        assert result["success"] is True
        assert "task_id" in result
        assert result["status"] == "delegated"

    @pytest.mark.asyncio
    async def test_background_task_completes(self, delegation_mocks):
        mod, *_ = delegation_mocks
        _, mock_cls = _make_mock_client()

        with patch("httpx.AsyncClient", mock_cls):
            status = await _invoke_and_wait(mod)

        assert status["status"] == "completed"
        assert "done" in status["result"]

    @pytest.mark.asyncio
    async def test_check_delegation_list_all(self, delegation_mocks):
        mod, *_ = delegation_mocks
        _, mock_cls = _make_mock_client()

        with patch("httpx.AsyncClient", mock_cls):
            await _invoke(mod, workspace_id="ws-a", task="task A")
            await _invoke(mod, workspace_id="ws-b", task="task B")

        fn = mod.check_delegation_status
        if hasattr(fn, "ainvoke"):
            result = await fn.ainvoke({"task_id": ""})
        else:
            result = await fn(task_id="")

        assert result["count"] == 2

    @pytest.mark.asyncio
    async def test_check_delegation_not_found(self, delegation_mocks):
        mod, *_ = delegation_mocks

        fn = mod.check_delegation_status
        if hasattr(fn, "ainvoke"):
            result = await fn.ainvoke({"task_id": "nonexistent"})
        else:
            result = await fn(task_id="nonexistent")

        assert "error" in result


class TestDiscovery:

    @pytest.mark.asyncio
    async def test_discovery_403(self, delegation_mocks):
        mod, *_ = delegation_mocks
        _, mock_cls = _make_mock_client(discover_status=403)

        with patch("httpx.AsyncClient", mock_cls):
            status = await _invoke_and_wait(mod)

        assert status["status"] == "failed"
        assert "Discovery failed" in status.get("error", "")

    @pytest.mark.asyncio
    async def test_discovery_404(self, delegation_mocks):
        mod, *_ = delegation_mocks
        _, mock_cls = _make_mock_client(discover_status=404)

        with patch("httpx.AsyncClient", mock_cls):
            status = await _invoke_and_wait(mod)

        assert status["status"] == "failed"

    @pytest.mark.asyncio
    async def test_discovery_no_url(self, delegation_mocks):
        mod, *_ = delegation_mocks
        _, mock_cls = _make_mock_client(discover_payload={"url": ""})

        with patch("httpx.AsyncClient", mock_cls):
            status = await _invoke_and_wait(mod)

        assert status["status"] == "failed"
        assert "No URL" in status.get("error", "")

    @pytest.mark.asyncio
    async def test_discovery_exception(self, delegation_mocks):
        mod, *_ = delegation_mocks
        _, mock_cls = _make_mock_client(discover_exc=Exception("dns fail"))

        with patch("httpx.AsyncClient", mock_cls):
            status = await _invoke_and_wait(mod)

        assert status["status"] == "failed"
        assert "dns fail" in status.get("error", "")


class TestA2ASuccess:

    @pytest.mark.asyncio
    async def test_success_with_parts(self, delegation_mocks):
        mod, *_ = delegation_mocks
        _, mock_cls = _make_mock_client(
            a2a_payload={"result": {"parts": [{"kind": "text", "text": "hello world"}]}}
        )

        with patch("httpx.AsyncClient", mock_cls):
            status = await _invoke_and_wait(mod)

        assert status["status"] == "completed"
        assert "hello world" in status["result"]

    @pytest.mark.asyncio
    async def test_success_with_artifacts(self, delegation_mocks):
        mod, *_ = delegation_mocks
        _, mock_cls = _make_mock_client(
            a2a_payload={
                "result": {
                    "artifacts": [{"parts": [{"kind": "text", "text": "artifact text"}]}],
                    "parts": [],
                }
            }
        )

        with patch("httpx.AsyncClient", mock_cls):
            status = await _invoke_and_wait(mod)

        assert status["status"] == "completed"
        assert "artifact text" in status["result"]


class TestA2AErrors:

    @pytest.mark.asyncio
    async def test_rpc_error(self, delegation_mocks):
        mod, *_ = delegation_mocks
        _, mock_cls = _make_mock_client(
            a2a_payload={"error": {"message": "internal error"}}
        )

        with patch("httpx.AsyncClient", mock_cls):
            status = await _invoke_and_wait(mod)

        assert status["status"] == "failed"

    @pytest.mark.asyncio
    async def test_network_error(self, delegation_mocks):
        mod, *_ = delegation_mocks
        mock_client, mock_cls = _make_mock_client()
        mock_client.post = AsyncMock(side_effect=httpx.ConnectError("refused"))

        with patch("httpx.AsyncClient", mock_cls):
            status = await _invoke_and_wait(mod)

        assert status["status"] == "failed"
        assert "refused" in status.get("error", "")


# ---------- #64: platform-mirroring helpers ----------

import asyncio as _asyncio_64
from unittest.mock import AsyncMock as _AsyncMock_64, patch as _patch_64


def test_record_delegation_on_platform_fires_http_post(delegation_mocks):
    """Agent registers the delegation on the platform so GET /delegations sees it."""
    mod, _, _, _ = delegation_mocks

    calls = []

    class FakeClient:
        def __init__(self, *a, **kw): pass
        async def __aenter__(self): return self
        async def __aexit__(self, *a): return False
        async def post(self, url, json=None):
            calls.append({"url": url, "json": json})
            class R:
                status_code = 202
            return R()

    with _patch_64.object(mod.httpx, "AsyncClient", FakeClient):
        with _patch_64.object(mod, "WORKSPACE_ID", "src-ws"), \
             _patch_64.object(mod, "PLATFORM_URL", "http://platform"):
            _asyncio_64.run(
                mod._record_delegation_on_platform("task-1", "target-ws", "hello")
            )

    assert len(calls) == 1
    assert calls[0]["url"] == "http://platform/workspaces/src-ws/delegations/record"
    body = calls[0]["json"]
    assert body == {"target_id": "target-ws", "task": "hello", "delegation_id": "task-1"}


def test_record_delegation_on_platform_best_effort_on_error(delegation_mocks):
    """Platform unreachable must NOT block the A2A delegation path."""
    mod, _, _, _ = delegation_mocks

    class FailingClient:
        def __init__(self, *a, **kw): pass
        async def __aenter__(self): return self
        async def __aexit__(self, *a): return False
        async def post(self, *a, **kw):
            raise RuntimeError("platform unreachable")

    with _patch_64.object(mod.httpx, "AsyncClient", FailingClient):
        with _patch_64.object(mod, "WORKSPACE_ID", "src-ws"), \
             _patch_64.object(mod, "PLATFORM_URL", "http://platform"):
            # Must not raise
            _asyncio_64.run(
                mod._record_delegation_on_platform("task-1", "target-ws", "hello")
            )


def test_update_delegation_on_platform_completed(delegation_mocks):
    mod, _, _, _ = delegation_mocks
    calls = []

    class FakeClient:
        def __init__(self, *a, **kw): pass
        async def __aenter__(self): return self
        async def __aexit__(self, *a): return False
        async def post(self, url, json=None):
            calls.append({"url": url, "json": json})
            class R:
                status_code = 200
            return R()

    with _patch_64.object(mod.httpx, "AsyncClient", FakeClient):
        with _patch_64.object(mod, "WORKSPACE_ID", "src-ws"), \
             _patch_64.object(mod, "PLATFORM_URL", "http://platform"):
            _asyncio_64.run(
                mod._update_delegation_on_platform(
                    "task-1", "completed", "", "the result text"
                )
            )

    assert calls[0]["url"] == "http://platform/workspaces/src-ws/delegations/task-1/update"
    assert calls[0]["json"]["status"] == "completed"
    assert calls[0]["json"]["response_preview"] == "the result text"


def test_update_delegation_on_platform_truncates_large_preview(delegation_mocks):
    """500-char cap protects log volume + mirrors the platform's 300-char truncate."""
    mod, _, _, _ = delegation_mocks
    calls = []

    class FakeClient:
        def __init__(self, *a, **kw): pass
        async def __aenter__(self): return self
        async def __aexit__(self, *a): return False
        async def post(self, url, json=None):
            calls.append({"url": url, "json": json})
            class R:
                status_code = 200
            return R()

    huge = "X" * 10000
    with _patch_64.object(mod.httpx, "AsyncClient", FakeClient):
        with _patch_64.object(mod, "WORKSPACE_ID", "src-ws"), \
             _patch_64.object(mod, "PLATFORM_URL", "http://platform"):
            _asyncio_64.run(
                mod._update_delegation_on_platform("task-1", "completed", "", huge)
            )
    assert len(calls[0]["json"]["response_preview"]) == 500
