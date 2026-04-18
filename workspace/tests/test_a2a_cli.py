"""Tests for a2a_cli.py — CLI tool for inter-workspace communication.

Uses importlib.util.spec_from_file_location to load the real module, bypassing
conftest mocks. Tests call async functions directly rather than going through
main() to avoid sys.exit() complications.
"""

import importlib.util
import json as json_mod
import sys
from pathlib import Path

import pytest

ROOT = Path(__file__).resolve().parents[1]


def _load_cli(monkeypatch, *, platform_url="http://platform.test", workspace_id="ws-test"):
    """Load the real a2a_cli.py in isolation."""
    monkeypatch.setenv("PLATFORM_URL", platform_url)
    monkeypatch.setenv("WORKSPACE_ID", workspace_id)

    spec = importlib.util.spec_from_file_location(
        "_test_a2a_cli",
        ROOT / "a2a_cli.py",
    )
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    # Patch module-level constants to match env
    mod.PLATFORM_URL = platform_url
    mod.WORKSPACE_ID = workspace_id
    return mod


class _FakeResponse:
    def __init__(self, status_code, payload):
        self.status_code = status_code
        self._payload = payload
        self.text = str(payload)

    def json(self):
        return self._payload


class _FakeBadJsonResponse:
    def __init__(self, status_code):
        self.status_code = status_code
        self.text = "not json"

    def json(self):
        raise ValueError("invalid json")


# ---------------------------------------------------------------------------
# discover()
# ---------------------------------------------------------------------------

class TestDiscover:

    async def test_discover_200(self, monkeypatch):
        mod = _load_cli(monkeypatch)

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass
            async def get(self, url, headers=None):
                assert "ws-target" in url
                assert headers.get("X-Workspace-ID") == "ws-test"
                return _FakeResponse(200, {"id": "ws-target", "url": "http://target.test/a2a"})

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)

        result = await mod.discover("ws-target")
        assert result == {"id": "ws-target", "url": "http://target.test/a2a"}

    async def test_discover_non_200_returns_none(self, monkeypatch):
        mod = _load_cli(monkeypatch)

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass
            async def get(self, url, headers=None):
                return _FakeResponse(403, {"error": "forbidden"})

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)

        result = await mod.discover("ws-target")
        assert result is None


# ---------------------------------------------------------------------------
# delegate() — sync mode
# ---------------------------------------------------------------------------

class TestDelegate:

    async def test_delegate_sync_success(self, monkeypatch, capsys):
        mod = _load_cli(monkeypatch)

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass

            async def get(self, url, headers=None):
                return _FakeResponse(200, {"url": "http://target.test/a2a"})

            async def post(self, url, json=None):
                return _FakeResponse(200, {
                    "result": {
                        "parts": [{"kind": "text", "text": "Task result!"}]
                    }
                })

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)

        await mod.delegate("ws-target", "do something")
        captured = capsys.readouterr()
        assert "Task result!" in captured.out

    async def test_delegate_sync_no_peer(self, monkeypatch, capsys):
        """When discover returns None, prints error and sys.exit(1) is called."""
        mod = _load_cli(monkeypatch)

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass
            async def get(self, url, headers=None):
                return _FakeResponse(404, {})

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)

        with pytest.raises(SystemExit) as exc_info:
            await mod.delegate("ws-target", "do something")
        assert exc_info.value.code == 1
        captured = capsys.readouterr()
        assert "cannot reach workspace" in captured.err

    async def test_delegate_sync_no_url(self, monkeypatch, capsys):
        """When peer has no URL, prints error and sys.exit(1)."""
        mod = _load_cli(monkeypatch)

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass
            async def get(self, url, headers=None):
                return _FakeResponse(200, {"url": ""})

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)

        with pytest.raises(SystemExit) as exc_info:
            await mod.delegate("ws-target", "do something")
        assert exc_info.value.code == 1
        captured = capsys.readouterr()
        assert "no URL" in captured.err

    async def test_delegate_sync_invalid_json_response(self, monkeypatch, capsys):
        """When A2A response is not valid JSON, prints error and sys.exit(1)."""
        mod = _load_cli(monkeypatch)

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass
            async def get(self, url, headers=None):
                return _FakeResponse(200, {"url": "http://target.test/a2a"})
            async def post(self, url, json=None):
                return _FakeBadJsonResponse(200)

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)

        with pytest.raises(SystemExit) as exc_info:
            await mod.delegate("ws-target", "do something")
        assert exc_info.value.code == 1
        captured = capsys.readouterr()
        assert "invalid JSON" in captured.err

    async def test_delegate_sync_error_response_exits(self, monkeypatch, capsys):
        """When A2A responds with error (non-rate-limit), prints error and sys.exit(1)."""
        mod = _load_cli(monkeypatch)

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass
            async def get(self, url, headers=None):
                return _FakeResponse(200, {"url": "http://target.test/a2a"})
            async def post(self, url, json=None):
                return _FakeResponse(200, {"error": {"message": "Permission denied"}})

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)

        with pytest.raises(SystemExit) as exc_info:
            await mod.delegate("ws-target", "do something")
        assert exc_info.value.code == 1
        captured = capsys.readouterr()
        assert "Permission denied" in captured.err

    async def test_delegate_sync_empty_response_final_attempt(self, monkeypatch, capsys):
        """Empty result on all retries prints fallback message."""
        mod = _load_cli(monkeypatch)

        # Mock asyncio.sleep to be instant
        monkeypatch.setattr(mod.asyncio, "sleep", lambda s: _instant_sleep())

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass
            async def get(self, url, headers=None):
                return _FakeResponse(200, {"url": "http://target.test/a2a"})
            async def post(self, url, json=None):
                return _FakeResponse(200, {"result": {"parts": [{"text": ""}]}})

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)

        await mod.delegate("ws-target", "do something")
        captured = capsys.readouterr()
        assert "no response after retries" in captured.out

    async def test_delegate_sync_rate_limit_then_success(self, monkeypatch, capsys):
        """Rate-limited response retries and eventually succeeds."""
        mod = _load_cli(monkeypatch)

        monkeypatch.setattr(mod.asyncio, "sleep", lambda s: _instant_sleep())

        call_count = {"n": 0}

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass
            async def get(self, url, headers=None):
                return _FakeResponse(200, {"url": "http://target.test/a2a"})
            async def post(self, url, json=None):
                call_count["n"] += 1
                if call_count["n"] < 2:
                    return _FakeResponse(200, {"error": {"message": "rate limit exceeded"}})
                return _FakeResponse(200, {"result": {"parts": [{"text": "Done"}]}})

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)

        await mod.delegate("ws-target", "do something")
        captured = capsys.readouterr()
        assert "Done" in captured.out

    async def test_delegate_sync_timeout_retries_then_fails(self, monkeypatch, capsys):
        """TimeoutException on all retries exits with error."""
        mod = _load_cli(monkeypatch)

        monkeypatch.setattr(mod.asyncio, "sleep", lambda s: _instant_sleep())

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass
            async def get(self, url, headers=None):
                return _FakeResponse(200, {"url": "http://target.test/a2a"})
            async def post(self, url, json=None):
                raise mod.httpx.TimeoutException("timed out")

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)

        with pytest.raises(SystemExit) as exc_info:
            await mod.delegate("ws-target", "do something")
        assert exc_info.value.code == 1
        captured = capsys.readouterr()
        assert "timed out" in captured.err

    async def test_delegate_sync_timeout_retry_then_success(self, monkeypatch, capsys):
        """TimeoutException on first attempt retries and eventually succeeds."""
        mod = _load_cli(monkeypatch)

        monkeypatch.setattr(mod.asyncio, "sleep", lambda s: _instant_sleep())

        call_count = {"n": 0}

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass
            async def get(self, url, headers=None):
                return _FakeResponse(200, {"url": "http://target.test/a2a"})
            async def post(self, url, json=None):
                call_count["n"] += 1
                if call_count["n"] == 1:
                    raise mod.httpx.TimeoutException("timed out")
                return _FakeResponse(200, {"result": {"parts": [{"text": "Success after retry"}]}})

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)

        await mod.delegate("ws-target", "do something")
        captured = capsys.readouterr()
        assert "Success after retry" in captured.out


# ---------------------------------------------------------------------------
# delegate() — async mode
# ---------------------------------------------------------------------------

class TestDelegateAsync:

    async def test_delegate_async_success(self, monkeypatch, capsys):
        mod = _load_cli(monkeypatch)

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass
            async def get(self, url, headers=None):
                return _FakeResponse(200, {"url": "http://target.test/a2a"})
            async def post(self, url, json=None):
                return _FakeResponse(200, {"jsonrpc": "2.0"})

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)

        await mod.delegate("ws-target", "do something", async_mode=True)
        captured = capsys.readouterr()
        parsed = json_mod.loads(captured.out)
        assert parsed["status"] == "submitted"
        assert parsed["target"] == "ws-target"

    async def test_delegate_async_timeout(self, monkeypatch, capsys):
        """TimeoutException in async mode prints uncertain status to stderr."""
        mod = _load_cli(monkeypatch)

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass
            async def get(self, url, headers=None):
                return _FakeResponse(200, {"url": "http://target.test/a2a"})
            async def post(self, url, json=None):
                raise mod.httpx.TimeoutException("timed out")

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)

        await mod.delegate("ws-target", "do something", async_mode=True)
        captured = capsys.readouterr()
        parsed = json_mod.loads(captured.err)
        assert parsed["status"] == "uncertain"


# ---------------------------------------------------------------------------
# peers()
# ---------------------------------------------------------------------------

class TestPeers:

    async def test_peers_success(self, monkeypatch, capsys):
        mod = _load_cli(monkeypatch)

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass
            async def get(self, url):
                return _FakeResponse(200, [
                    {"id": "ws-1", "name": "Alpha Worker", "role": "worker", "status": "online"},
                    {"id": "ws-2", "name": "Beta Analyst", "role": "analyst", "status": "idle"},
                ])

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)

        await mod.peers()
        captured = capsys.readouterr()
        assert "ws-1" in captured.out
        assert "Alpha Worker" in captured.out
        assert "ws-2" in captured.out

    async def test_peers_failure_exits(self, monkeypatch, capsys):
        mod = _load_cli(monkeypatch)

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass
            async def get(self, url):
                return _FakeResponse(500, {})

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)

        with pytest.raises(SystemExit) as exc_info:
            await mod.peers()
        assert exc_info.value.code == 1
        captured = capsys.readouterr()
        assert "could not fetch peers" in captured.err


# ---------------------------------------------------------------------------
# info()
# ---------------------------------------------------------------------------

class TestInfo:

    async def test_info_success(self, monkeypatch, capsys):
        mod = _load_cli(monkeypatch)

        workspace_data = {
            "id": "ws-test",
            "name": "Test Workspace",
            "role": "worker",
            "tier": "standard",
            "status": "active",
            "parent_id": "ws-parent",
        }

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass
            async def get(self, url):
                assert "ws-test" in url
                return _FakeResponse(200, workspace_data)

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)

        await mod.info()
        captured = capsys.readouterr()
        assert "ws-test" in captured.out
        assert "Test Workspace" in captured.out
        assert "worker" in captured.out
        assert "standard" in captured.out
        assert "active" in captured.out
        assert "ws-parent" in captured.out

    async def test_info_non_200_no_output(self, monkeypatch, capsys):
        """When platform returns non-200, info() prints nothing (no crash)."""
        mod = _load_cli(monkeypatch)

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass
            async def get(self, url):
                return _FakeResponse(404, {"error": "not found"})

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)

        # No exception — just no output
        await mod.info()
        captured = capsys.readouterr()
        assert captured.out == ""


# ---------------------------------------------------------------------------
# check_status()
# ---------------------------------------------------------------------------

class TestCheckStatus:

    async def test_check_status_completed(self, monkeypatch, capsys):
        mod = _load_cli(monkeypatch)

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass
            async def get(self, url, headers=None):
                return _FakeResponse(200, {"url": "http://target.test/a2a"})
            async def post(self, url, json=None):
                return _FakeResponse(200, {
                    "result": {
                        "status": {"state": "completed"},
                        "artifacts": [
                            {"parts": [{"text": "Artifact result"}]}
                        ],
                    }
                })

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)

        await mod.check_status("ws-target", "task-123")
        captured = capsys.readouterr()
        assert "completed" in captured.out
        assert "Artifact result" in captured.out

    async def test_check_status_no_peer(self, monkeypatch, capsys):
        mod = _load_cli(monkeypatch)

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass
            async def get(self, url, headers=None):
                return _FakeResponse(404, {})

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)

        with pytest.raises(SystemExit) as exc_info:
            await mod.check_status("ws-target", "task-123")
        assert exc_info.value.code == 1
        captured = capsys.readouterr()
        assert "cannot reach workspace" in captured.err

    async def test_check_status_error_response(self, monkeypatch, capsys):
        mod = _load_cli(monkeypatch)

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass
            async def get(self, url, headers=None):
                return _FakeResponse(200, {"url": "http://target.test/a2a"})
            async def post(self, url, json=None):
                return _FakeResponse(200, {"error": {"message": "task not found"}})

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)

        await mod.check_status("ws-target", "task-999")
        captured = capsys.readouterr()
        assert "task not found" in captured.out

    async def test_check_status_running(self, monkeypatch, capsys):
        """Status in non-completed state — no artifacts printed."""
        mod = _load_cli(monkeypatch)

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass
            async def get(self, url, headers=None):
                return _FakeResponse(200, {"url": "http://target.test/a2a"})
            async def post(self, url, json=None):
                return _FakeResponse(200, {
                    "result": {
                        "status": {"state": "running"},
                        "artifacts": [],
                    }
                })

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)

        await mod.check_status("ws-target", "task-456")
        captured = capsys.readouterr()
        assert "running" in captured.out


# ---------------------------------------------------------------------------
# main() — via command dispatch
# ---------------------------------------------------------------------------

class TestMain:

    def test_main_no_args_exits(self, monkeypatch, capsys):
        mod = _load_cli(monkeypatch)
        monkeypatch.setattr(sys, "argv", ["a2a"])

        with pytest.raises(SystemExit) as exc_info:
            mod.main()
        assert exc_info.value.code == 1
        captured = capsys.readouterr()
        assert "Usage" in captured.out

    def test_main_unknown_command_exits(self, monkeypatch, capsys):
        mod = _load_cli(monkeypatch)
        monkeypatch.setattr(sys, "argv", ["a2a", "unknown-cmd"])

        with pytest.raises(SystemExit) as exc_info:
            mod.main()
        assert exc_info.value.code == 1
        captured = capsys.readouterr()
        assert "Unknown command" in captured.err

    def test_main_delegate_missing_args_exits(self, monkeypatch, capsys):
        mod = _load_cli(monkeypatch)
        monkeypatch.setattr(sys, "argv", ["a2a", "delegate"])

        with pytest.raises(SystemExit) as exc_info:
            mod.main()
        assert exc_info.value.code == 1
        captured = capsys.readouterr()
        assert "Usage" in captured.err

    def test_main_status_missing_args_exits(self, monkeypatch, capsys):
        mod = _load_cli(monkeypatch)
        monkeypatch.setattr(sys, "argv", ["a2a", "status", "only-one-arg"])

        with pytest.raises(SystemExit) as exc_info:
            mod.main()
        assert exc_info.value.code == 1
        captured = capsys.readouterr()
        assert "Usage" in captured.err

    def test_main_delegate_calls_asyncio_run(self, monkeypatch):
        mod = _load_cli(monkeypatch)
        monkeypatch.setattr(sys, "argv", ["a2a", "delegate", "ws-target", "do something"])

        called_with = {}

        def fake_asyncio_run(coro):
            called_with["coro"] = coro
            # Close the coroutine to avoid ResourceWarning
            coro.close()

        monkeypatch.setattr(mod.asyncio, "run", fake_asyncio_run)

        mod.main()
        assert "coro" in called_with

    def test_main_delegate_async_flag(self, monkeypatch):
        mod = _load_cli(monkeypatch)
        monkeypatch.setattr(sys, "argv", ["a2a", "delegate", "--async", "ws-target", "do something"])

        called_with = {}

        def fake_asyncio_run(coro):
            called_with["coro"] = coro
            coro.close()

        monkeypatch.setattr(mod.asyncio, "run", fake_asyncio_run)

        mod.main()
        assert "coro" in called_with

    def test_main_status_calls_asyncio_run(self, monkeypatch):
        mod = _load_cli(monkeypatch)
        monkeypatch.setattr(sys, "argv", ["a2a", "status", "ws-target", "task-abc"])

        called_with = {}

        def fake_asyncio_run(coro):
            called_with["coro"] = coro
            coro.close()

        monkeypatch.setattr(mod.asyncio, "run", fake_asyncio_run)

        mod.main()
        assert "coro" in called_with

    def test_main_peers_calls_asyncio_run(self, monkeypatch):
        mod = _load_cli(monkeypatch)
        monkeypatch.setattr(sys, "argv", ["a2a", "peers"])

        called_with = {}

        def fake_asyncio_run(coro):
            called_with["coro"] = coro
            coro.close()

        monkeypatch.setattr(mod.asyncio, "run", fake_asyncio_run)

        mod.main()
        assert "coro" in called_with

    def test_main_info_calls_asyncio_run(self, monkeypatch):
        mod = _load_cli(monkeypatch)
        monkeypatch.setattr(sys, "argv", ["a2a", "info"])

        called_with = {}

        def fake_asyncio_run(coro):
            called_with["coro"] = coro
            coro.close()

        monkeypatch.setattr(mod.asyncio, "run", fake_asyncio_run)

        mod.main()
        assert "coro" in called_with


# ---------------------------------------------------------------------------
# Helper coroutine for instant sleep mock
# ---------------------------------------------------------------------------

async def _instant_sleep():
    """No-op coroutine to replace asyncio.sleep in tests."""
    pass
