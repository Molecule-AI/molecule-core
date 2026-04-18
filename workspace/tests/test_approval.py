"""Tests for the approval tool — polling path, timeout, errors, and WebSocket path."""

import asyncio
import importlib
import sys
from types import ModuleType
from unittest.mock import AsyncMock, MagicMock, patch

import pytest


# ---------------------------------------------------------------------------
# Helpers to load the approval module in isolation with injectable mocks
# ---------------------------------------------------------------------------

def _load_approval(monkeypatch, *, platform_url="http://platform.test",
                    workspace_id="ws-test", poll_interval="0.01", timeout="1"):
    """Reload tools.approval with controlled env vars and httpx mock.

    Uses monkeypatch.setitem so sys.modules is restored after each test,
    preventing the real module from leaking into other test modules.
    """
    monkeypatch.setenv("PLATFORM_URL", platform_url)
    monkeypatch.setenv("WORKSPACE_ID", workspace_id)
    monkeypatch.setenv("APPROVAL_POLL_INTERVAL", poll_interval)
    monkeypatch.setenv("APPROVAL_TIMEOUT", timeout)

    # Ensure langchain_core.tools is mocked (decorator must be a no-op)
    if "langchain_core" not in sys.modules:
        lc = ModuleType("langchain_core")
        lc_tools = ModuleType("langchain_core.tools")
        lc_tools.tool = lambda f: f
        monkeypatch.setitem(sys.modules, "langchain_core", lc)
        monkeypatch.setitem(sys.modules, "langchain_core.tools", lc_tools)
    else:
        monkeypatch.setattr(sys.modules["langchain_core.tools"], "tool", lambda f: f, raising=False)

    import importlib.util as ilu
    import os
    spec = ilu.spec_from_file_location(
        "builtin_tools.approval",
        os.path.join(os.path.dirname(__file__), "..", "builtin_tools", "approval.py"),
    )
    mod = ilu.module_from_spec(spec)
    # Use setitem so monkeypatch restores the original mock after the test
    monkeypatch.setitem(sys.modules, "builtin_tools.approval", mod)
    spec.loader.exec_module(mod)
    return mod


class _FakeResponse:
    def __init__(self, status_code, payload):
        self.status_code = status_code
        self._payload = payload

    def json(self):
        return self._payload


# ---------------------------------------------------------------------------
# Polling path — happy paths
# ---------------------------------------------------------------------------

class TestPollingApproval:

    def test_approval_granted(self, monkeypatch):
        """request_approval returns approved=True when platform grants it."""
        mod = _load_approval(monkeypatch)

        call_count = {"n": 0}

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass

            async def post(self, url, json):
                assert url == "http://platform.test/workspaces/ws-test/approvals"
                assert json == {"action": "deploy", "reason": "need to ship"}
                return _FakeResponse(201, {"approval_id": "appr-1"})

            async def get(self, url):
                call_count["n"] += 1
                return _FakeResponse(200, [
                    {"id": "appr-1", "status": "approved", "decided_by": "alice@example.com"}
                ])

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)

        result = asyncio.run(mod.request_approval("deploy", "need to ship"))

        assert result["approved"] is True
        assert result["approval_id"] == "appr-1"
        assert result["decided_by"] == "alice@example.com"

    def test_approval_denied(self, monkeypatch):
        """request_approval returns approved=False when platform denies."""
        mod = _load_approval(monkeypatch)

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass

            async def post(self, url, json):
                return _FakeResponse(201, {"approval_id": "appr-2"})

            async def get(self, url):
                return _FakeResponse(200, [
                    {"id": "appr-2", "status": "denied", "decided_by": "bob@example.com"}
                ])

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)

        result = asyncio.run(mod.request_approval("delete everything", "spring cleaning"))

        assert result["approved"] is False
        assert result["approval_id"] == "appr-2"
        assert result["decided_by"] == "bob@example.com"
        assert result.get("message") == "Denied by human"

    def test_approval_pending_then_granted(self, monkeypatch):
        """Polls through pending state before receiving approved status."""
        mod = _load_approval(monkeypatch)

        responses = [
            [{"id": "appr-3", "status": "pending"}],
            [{"id": "appr-3", "status": "pending"}],
            [{"id": "appr-3", "status": "approved", "decided_by": "carol"}],
        ]
        idx = {"i": 0}

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass

            async def post(self, url, json):
                return _FakeResponse(201, {"approval_id": "appr-3"})

            async def get(self, url):
                payload = responses[min(idx["i"], len(responses) - 1)]
                idx["i"] += 1
                return _FakeResponse(200, payload)

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)

        result = asyncio.run(mod.request_approval("restart service", "memory leak"))

        assert result["approved"] is True
        assert result["approval_id"] == "appr-3"


# ---------------------------------------------------------------------------
# Failure / edge cases
# ---------------------------------------------------------------------------

class TestApprovalFailures:

    def test_post_failure_returns_error(self, monkeypatch):
        """Returns error dict when the approval creation POST fails."""
        mod = _load_approval(monkeypatch)

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass

            async def post(self, url, json):
                return _FakeResponse(500, {})

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)

        result = asyncio.run(mod.request_approval("explode", "YOLO"))

        assert result["approved"] is False
        assert "error" in result
        assert "500" in result["error"]

    def test_post_exception_returns_error(self, monkeypatch):
        """Returns error dict when POST raises a network exception."""
        mod = _load_approval(monkeypatch)

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass

            async def post(self, url, json):
                raise ConnectionError("platform unreachable")

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)

        result = asyncio.run(mod.request_approval("crash", "chaos"))

        assert result["approved"] is False
        assert "error" in result

    def test_timeout_returns_error(self, monkeypatch):
        """Returns error dict when approval times out before a decision."""
        # timeout=0.05s so the test is fast but exercises the timeout branch
        mod = _load_approval(monkeypatch, poll_interval="0.03", timeout="0.05")

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass

            async def post(self, url, json):
                return _FakeResponse(201, {"approval_id": "appr-timeout"})

            async def get(self, url):
                # Always return pending — never decide
                return _FakeResponse(200, [{"id": "appr-timeout", "status": "pending"}])

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)

        result = asyncio.run(mod.request_approval("hang forever", "testing timeout"))

        assert result["approved"] is False
        assert "error" in result or "approval_id" in result  # timed out
        # Key assertion: approval_id present and no "decided_by" (no human decided)
        assert result.get("approval_id") == "appr-timeout"
        assert "decided_by" not in result

    def test_poll_http_error_is_swallowed(self, monkeypatch):
        """Transient GET failures during polling are swallowed; tool keeps retrying."""
        mod = _load_approval(monkeypatch, poll_interval="0.01", timeout="0.5")

        call_count = {"n": 0}

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass

            async def post(self, url, json):
                return _FakeResponse(201, {"approval_id": "appr-flaky"})

            async def get(self, url):
                call_count["n"] += 1
                if call_count["n"] < 3:
                    raise ConnectionError("transient")
                return _FakeResponse(200, [
                    {"id": "appr-flaky", "status": "approved", "decided_by": "dave"}
                ])

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)

        result = asyncio.run(mod.request_approval("try again", "retry logic"))

        assert result["approved"] is True
        assert call_count["n"] >= 3

    def test_unrelated_approvals_ignored(self, monkeypatch):
        """Other approval records in the list don't affect the current request."""
        mod = _load_approval(monkeypatch)

        responses = iter([
            # First poll: only unrelated records
            [
                {"id": "appr-other", "status": "approved", "decided_by": "eve"},
            ],
            # Second poll: our approval is decided
            [
                {"id": "appr-other", "status": "approved", "decided_by": "eve"},
                {"id": "appr-target", "status": "approved", "decided_by": "frank"},
            ],
        ])

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass

            async def post(self, url, json):
                return _FakeResponse(201, {"approval_id": "appr-target"})

            async def get(self, url):
                try:
                    return _FakeResponse(200, next(responses))
                except StopIteration:
                    return _FakeResponse(200, [])

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)

        result = asyncio.run(mod.request_approval("targeted action", "specific reason"))

        assert result["approved"] is True
        assert result["approval_id"] == "appr-target"
        assert result["decided_by"] == "frank"


# ---------------------------------------------------------------------------
# WebSocket path (new implementation)
# ---------------------------------------------------------------------------

class TestWebSocketApproval:
    """Tests for the WebSocket-based notification path.

    When APPROVAL_USE_WEBSOCKET=true (or websockets is available), the tool
    should subscribe to the platform WebSocket and wait for an APPROVAL_DECIDED
    event instead of polling.
    """

    def test_websocket_path_granted(self, monkeypatch):
        """WebSocket path resolves immediately when APPROVAL_DECIDED event arrives."""
        mod = _load_approval(monkeypatch)

        # Skip if the module hasn't been upgraded to WebSocket support yet
        if not hasattr(mod, "request_approval_ws") and not getattr(mod, "APPROVAL_USE_WEBSOCKET", None):
            pytest.skip("WebSocket path not yet implemented in approval.py — see Track 2")

        # Mock websockets.connect — must be a sync callable returning an async ctx manager
        import json

        class FakeWSConn:
            """Async context manager that yields one APPROVAL_DECIDED message."""
            async def __aenter__(self_inner):
                return self_inner
            async def __aexit__(self_inner, *a):
                pass
            def __aiter__(self_inner):
                return self_inner
            async def __anext__(self_inner):
                return json.dumps({
                    "event": "APPROVAL_DECIDED",
                    "approval_id": "appr-ws-1",
                    "status": "approved",
                    "decided_by": "grace@example.com",
                })

        class FakeWSModule:
            @staticmethod
            def connect(url, additional_headers=None):
                return FakeWSConn()

        monkeypatch.setattr(mod, "websockets", FakeWSModule, raising=False)

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass
            async def post(self, url, json):
                return _FakeResponse(201, {"approval_id": "appr-ws-1"})

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)
        monkeypatch.setenv("APPROVAL_USE_WEBSOCKET", "true")

        result = asyncio.run(mod.request_approval("ws action", "ws reason"))

        assert result["approved"] is True
        assert result["approval_id"] == "appr-ws-1"
        assert result["decided_by"] == "grace@example.com"

    def test_websocket_path_denied(self, monkeypatch):
        """WebSocket path resolves with denied when APPROVAL_DECIDED event says denied."""
        mod = _load_approval(monkeypatch)

        if not hasattr(mod, "request_approval_ws") and not getattr(mod, "APPROVAL_USE_WEBSOCKET", None):
            pytest.skip("WebSocket path not yet implemented in approval.py — see Track 2")

        import json

        class FakeWSConnDeny:
            async def __aenter__(self_inner): return self_inner
            async def __aexit__(self_inner, *a): pass
            def __aiter__(self_inner): return self_inner
            async def __anext__(self_inner):
                return json.dumps({
                    "event": "APPROVAL_DECIDED",
                    "approval_id": "appr-ws-deny",
                    "status": "denied",
                    "decided_by": "heidi",
                })

        class FakeWSModule:
            @staticmethod
            def connect(url, additional_headers=None):
                return FakeWSConnDeny()

        monkeypatch.setattr(mod, "websockets", FakeWSModule, raising=False)

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass
            async def post(self, url, json):
                return _FakeResponse(201, {"approval_id": "appr-ws-deny"})

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)
        monkeypatch.setenv("APPROVAL_USE_WEBSOCKET", "true")

        result = asyncio.run(mod.request_approval("dangerous delete", "cleanup"))

        assert result["approved"] is False
        assert result["approval_id"] == "appr-ws-deny"

    def test_websocket_fallback_to_polling_on_import_error(self, monkeypatch):
        """Falls back to polling gracefully if websockets package is missing."""
        mod = _load_approval(monkeypatch)

        if not hasattr(mod, "request_approval_ws") and not getattr(mod, "APPROVAL_USE_WEBSOCKET", None):
            pytest.skip("WebSocket path not yet implemented in approval.py — see Track 2")

        # Simulate websockets not installed
        monkeypatch.setattr(mod, "websockets", None, raising=False)
        monkeypatch.setenv("APPROVAL_USE_WEBSOCKET", "true")

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass
            async def post(self, url, json):
                return _FakeResponse(201, {"approval_id": "appr-fallback"})
            async def get(self, url):
                return _FakeResponse(200, [
                    {"id": "appr-fallback", "status": "approved", "decided_by": "ivan"}
                ])

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)

        result = asyncio.run(mod.request_approval("fallback test", "ws unavailable"))

        assert result["approved"] is True


# ---------------------------------------------------------------------------
# Gap 6: Module-level _USE_WEBSOCKET_DEFAULT env-var branches (lines 65, 67, 72-73, 78-79)
# ---------------------------------------------------------------------------

class TestApprovalModuleLevelWebsocketBranches:

    def test_env_false_sets_use_websocket_false(self, monkeypatch):
        """Line 65: APPROVAL_USE_WEBSOCKET=false → _USE_WEBSOCKET_DEFAULT=False."""
        monkeypatch.setenv("APPROVAL_USE_WEBSOCKET", "false")
        mod = _load_approval(monkeypatch)
        assert mod._USE_WEBSOCKET_DEFAULT is False

    def test_env_true_sets_use_websocket_true(self, monkeypatch):
        """Line 67: APPROVAL_USE_WEBSOCKET=true → _USE_WEBSOCKET_DEFAULT=True."""
        monkeypatch.setenv("APPROVAL_USE_WEBSOCKET", "true")
        mod = _load_approval(monkeypatch)
        assert mod._USE_WEBSOCKET_DEFAULT is True

    def test_env_unset_websockets_installed_sets_true(self, monkeypatch):
        """Lines 72-73: no env var, websockets importable → _USE_WEBSOCKET_DEFAULT=True."""
        monkeypatch.delenv("APPROVAL_USE_WEBSOCKET", raising=False)
        # Inject a fake websockets module so import succeeds
        fake_ws = ModuleType("websockets")
        monkeypatch.setitem(sys.modules, "websockets", fake_ws)
        mod = _load_approval(monkeypatch)
        assert mod._USE_WEBSOCKET_DEFAULT is True

    def test_env_unset_websockets_not_installed_sets_false(self, monkeypatch):
        """Lines 78-79: no env var, websockets not importable → _USE_WEBSOCKET_DEFAULT=False."""
        monkeypatch.delenv("APPROVAL_USE_WEBSOCKET", raising=False)
        # Remove websockets so import fails
        monkeypatch.setitem(sys.modules, "websockets", None)
        mod = _load_approval(monkeypatch)
        assert mod._USE_WEBSOCKET_DEFAULT is False


# ---------------------------------------------------------------------------
# Gap 6: WebSocket _wait_websocket — invalid JSON, wrong event type, wrong ID
# ---------------------------------------------------------------------------

class TestWaitWebsocketEdgeCases:

    def test_websocket_invalid_json_message_skipped(self, monkeypatch):
        """Lines 126-127: invalid JSON message in WebSocket → continue (skipped)."""
        mod = _load_approval(monkeypatch)

        if not getattr(mod, "APPROVAL_USE_WEBSOCKET", None):
            pytest.skip("WebSocket path not yet implemented")

        import json as _json

        messages_iter = iter([
            "not valid json {{{",  # invalid JSON → continue
            _json.dumps({          # valid but wrong event type → continue
                "event": "SOME_OTHER_EVENT",
                "approval_id": "appr-ws-edge",
            }),
            _json.dumps({          # right event but wrong ID → continue
                "event": "APPROVAL_DECIDED",
                "approval_id": "appr-different-id",
                "status": "approved",
                "decided_by": "alice",
            }),
            _json.dumps({          # matching message
                "event": "APPROVAL_DECIDED",
                "approval_id": "appr-ws-edge",
                "status": "approved",
                "decided_by": "alice",
            }),
        ])

        class FakeWSConn:
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass
            def __aiter__(self): return self
            async def __anext__(self):
                try:
                    return next(messages_iter)
                except StopIteration:
                    raise StopAsyncIteration

        class FakeWSModule:
            @staticmethod
            def connect(url, additional_headers=None):
                return FakeWSConn()

        monkeypatch.setattr(mod, "websockets", FakeWSModule, raising=False)

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass
            async def post(self, url, json):
                return _FakeResponse(201, {"approval_id": "appr-ws-edge"})

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)
        monkeypatch.setenv("APPROVAL_USE_WEBSOCKET", "true")

        result = asyncio.run(mod.request_approval("edge case action", "testing edge cases"))

        assert result["approved"] is True
        assert result["approval_id"] == "appr-ws-edge"


# ---------------------------------------------------------------------------
# Gap 6: RBAC deny in request_approval (lines 215-224)
# ---------------------------------------------------------------------------

class TestRequestApprovalRBACDeny:

    def test_rbac_deny_returns_error(self, monkeypatch):
        """Lines 215-224: check_permission returns False → approved=False with RBAC error."""
        import importlib.util as ilu
        import os

        monkeypatch.setenv("PLATFORM_URL", "http://platform.test")
        monkeypatch.setenv("WORKSPACE_ID", "ws-test")
        monkeypatch.setenv("APPROVAL_POLL_INTERVAL", "0.01")
        monkeypatch.setenv("APPROVAL_TIMEOUT", "1")

        # Ensure langchain_core.tools is mocked
        if "langchain_core" not in sys.modules:
            lc = ModuleType("langchain_core")
            lc_tools = ModuleType("langchain_core.tools")
            lc_tools.tool = lambda f: f
            monkeypatch.setitem(sys.modules, "langchain_core", lc)
            monkeypatch.setitem(sys.modules, "langchain_core.tools", lc_tools)
        else:
            monkeypatch.setattr(sys.modules["langchain_core.tools"], "tool", lambda f: f, raising=False)

        # Build a mock tools.audit that denies the "approve" permission
        mock_audit_mod = ModuleType("builtin_tools.audit")
        mock_audit_mod.check_permission = MagicMock(return_value=False)
        mock_audit_mod.get_workspace_roles = MagicMock(return_value=(["read-only"], {}))
        mock_audit_mod.log_event = MagicMock(return_value="trace-rbac")
        monkeypatch.setitem(sys.modules, "builtin_tools.audit", mock_audit_mod)

        spec = ilu.spec_from_file_location(
            "builtin_tools.approval",
            os.path.join(os.path.dirname(__file__), "..", "builtin_tools", "approval.py"),
        )
        mod2 = ilu.module_from_spec(spec)
        monkeypatch.setitem(sys.modules, "builtin_tools.approval", mod2)
        spec.loader.exec_module(mod2)

        result = asyncio.run(mod2.request_approval("destroy everything", "chaos"))

        assert result["approved"] is False
        assert "error" in result
        assert "RBAC" in result["error"] or "approve" in result["error"]
        mock_audit_mod.log_event.assert_called_once()
