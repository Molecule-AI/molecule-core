"""Tests for tools/awareness_client.py — workspace-scoped awareness backend wrapper.

Uses importlib.util.spec_from_file_location to load the real module, bypassing
the conftest mock at tools.awareness_client.
"""

import importlib.util
import sys
from pathlib import Path
from types import ModuleType
from unittest.mock import MagicMock

import pytest

ROOT = Path(__file__).resolve().parents[1]
TOOLS_DIR = ROOT / "builtin_tools"


def _load_awareness_client(monkeypatch):
    """Load the real tools/awareness_client.py in isolation."""
    # Ensure policies.namespaces is importable
    if "policies" not in sys.modules:
        policies_mod = ModuleType("policies")
        policies_mod.__path__ = [str(ROOT / "policies")]
        monkeypatch.setitem(sys.modules, "policies", policies_mod)

    if "policies.namespaces" not in sys.modules:
        spec = importlib.util.spec_from_file_location(
            "policies.namespaces",
            ROOT / "policies" / "namespaces.py",
        )
        ns_mod = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(ns_mod)
        monkeypatch.setitem(sys.modules, "policies.namespaces", ns_mod)

    spec = importlib.util.spec_from_file_location(
        "_test_awareness_client",
        TOOLS_DIR / "awareness_client.py",
    )
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


class _FakeResponse:
    def __init__(self, status_code, payload, text=None):
        self.status_code = status_code
        self._payload = payload
        self.text = text if text is not None else str(payload)

    def json(self):
        return self._payload


class _FakeBadJsonResponse:
    """Response whose .json() raises ValueError (simulates non-JSON body)."""
    def __init__(self, status_code, text="bad json"):
        self.status_code = status_code
        self.text = text

    def json(self):
        raise ValueError("invalid json")


# ---------------------------------------------------------------------------
# get_awareness_config
# ---------------------------------------------------------------------------

class TestGetAwarenessConfig:

    def test_no_url_returns_none(self, monkeypatch):
        mod = _load_awareness_client(monkeypatch)
        monkeypatch.delenv("AWARENESS_URL", raising=False)
        monkeypatch.setenv("WORKSPACE_ID", "ws-test")

        result = mod.get_awareness_config()
        assert result is None

    def test_with_url_and_workspace_id_returns_dict(self, monkeypatch):
        mod = _load_awareness_client(monkeypatch)
        monkeypatch.setenv("AWARENESS_URL", "http://awareness.test")
        monkeypatch.setenv("WORKSPACE_ID", "ws-abc")
        monkeypatch.delenv("AWARENESS_NAMESPACE", raising=False)

        result = mod.get_awareness_config()
        assert result is not None
        assert result["base_url"] == "http://awareness.test"
        assert result["namespace"] == "workspace:ws-abc"

    def test_with_url_and_configured_namespace(self, monkeypatch):
        mod = _load_awareness_client(monkeypatch)
        monkeypatch.setenv("AWARENESS_URL", "http://awareness.test/")
        monkeypatch.setenv("WORKSPACE_ID", "ws-abc")
        monkeypatch.setenv("AWARENESS_NAMESPACE", "custom-ns")

        result = mod.get_awareness_config()
        assert result is not None
        assert result["base_url"] == "http://awareness.test"  # trailing slash stripped
        assert result["namespace"] == "custom-ns"

    def test_no_workspace_id_and_no_namespace_returns_none(self, monkeypatch):
        mod = _load_awareness_client(monkeypatch)
        monkeypatch.setenv("AWARENESS_URL", "http://awareness.test")
        monkeypatch.delenv("WORKSPACE_ID", raising=False)
        monkeypatch.delenv("AWARENESS_NAMESPACE", raising=False)

        # Both workspace_id and configured_namespace are empty
        # The code: if not workspace_id and not configured_namespace: return None
        result = mod.get_awareness_config()
        assert result is None


# ---------------------------------------------------------------------------
# build_awareness_client
# ---------------------------------------------------------------------------

class TestBuildAwarenessClient:

    def test_returns_none_when_no_config(self, monkeypatch):
        mod = _load_awareness_client(monkeypatch)
        monkeypatch.delenv("AWARENESS_URL", raising=False)
        monkeypatch.setenv("WORKSPACE_ID", "ws-test")

        result = mod.build_awareness_client()
        assert result is None

    def test_returns_client_when_configured(self, monkeypatch):
        mod = _load_awareness_client(monkeypatch)
        monkeypatch.setenv("AWARENESS_URL", "http://awareness.test")
        monkeypatch.setenv("WORKSPACE_ID", "ws-xyz")
        monkeypatch.delenv("AWARENESS_NAMESPACE", raising=False)

        result = mod.build_awareness_client()
        assert result is not None
        assert isinstance(result, mod.AwarenessClient)
        assert result.base_url == "http://awareness.test"
        assert result.namespace == "workspace:ws-xyz"


# ---------------------------------------------------------------------------
# AwarenessClient.commit
# ---------------------------------------------------------------------------

class TestAwarenessClientCommit:

    async def test_commit_success_201(self, monkeypatch):
        mod = _load_awareness_client(monkeypatch)

        class FakeClient:
            def __init__(self, timeout): self.timeout = timeout
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass
            async def post(self, url, json):
                assert url == "http://awareness.test/api/v1/namespaces/ws-ns/memories"
                assert json == {"content": "hello", "scope": "TEAM"}
                return _FakeResponse(201, {"id": "mem-001"})

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)

        client = mod.AwarenessClient("http://awareness.test", "ws-ns")
        result = await client.commit("hello", "TEAM")
        assert result == {"success": True, "id": "mem-001", "scope": "TEAM"}

    async def test_commit_success_200(self, monkeypatch):
        mod = _load_awareness_client(monkeypatch)

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass
            async def post(self, url, json):
                return _FakeResponse(200, {"id": "mem-002"})

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)

        client = mod.AwarenessClient("http://awareness.test", "ws-ns")
        result = await client.commit("content", "LOCAL")
        assert result["success"] is True
        assert result["id"] == "mem-002"

    async def test_commit_failure(self, monkeypatch):
        mod = _load_awareness_client(monkeypatch)

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass
            async def post(self, url, json):
                return _FakeResponse(500, {"error": "server error"})

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)

        client = mod.AwarenessClient("http://awareness.test", "ws-ns")
        result = await client.commit("content", "TEAM")
        assert result["success"] is False
        assert "server error" in str(result["error"])

    async def test_commit_failure_invalid_json(self, monkeypatch):
        mod = _load_awareness_client(monkeypatch)

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass
            async def post(self, url, json):
                return _FakeBadJsonResponse(400, "bad request body")

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)

        client = mod.AwarenessClient("http://awareness.test", "ws-ns")
        result = await client.commit("content", "TEAM")
        assert result["success"] is False
        assert "bad request body" in str(result["error"])


# ---------------------------------------------------------------------------
# AwarenessClient.search
# ---------------------------------------------------------------------------

class TestAwarenessClientSearch:

    async def test_search_success_list_response(self, monkeypatch):
        mod = _load_awareness_client(monkeypatch)

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass
            async def get(self, url, params):
                assert params == {"q": "test query", "scope": "TEAM"}
                return _FakeResponse(200, [{"content": "mem1"}, {"content": "mem2"}])

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)

        client = mod.AwarenessClient("http://awareness.test", "ws-ns")
        result = await client.search(query="test query", scope="TEAM")
        assert result["success"] is True
        assert result["count"] == 2
        assert len(result["memories"]) == 2

    async def test_search_success_dict_response(self, monkeypatch):
        """Search with dict-wrapped memories response."""
        mod = _load_awareness_client(monkeypatch)

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass
            async def get(self, url, params):
                return _FakeResponse(200, {"memories": [{"content": "item"}]})

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)

        client = mod.AwarenessClient("http://awareness.test", "ws-ns")
        result = await client.search(query="q")
        assert result["success"] is True
        assert result["count"] == 1

    async def test_search_no_query_no_scope(self, monkeypatch):
        """Search with no query/scope sends empty params."""
        mod = _load_awareness_client(monkeypatch)

        captured = {}

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass
            async def get(self, url, params):
                captured["params"] = params
                return _FakeResponse(200, [])

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)

        client = mod.AwarenessClient("http://awareness.test", "ws-ns")
        result = await client.search()
        assert result["success"] is True
        assert result["count"] == 0
        assert captured["params"] == {}

    async def test_search_failure(self, monkeypatch):
        mod = _load_awareness_client(monkeypatch)

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass
            async def get(self, url, params):
                return _FakeResponse(503, {"error": "service unavailable"})

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)

        client = mod.AwarenessClient("http://awareness.test", "ws-ns")
        result = await client.search(query="q")
        assert result["success"] is False
        assert "service unavailable" in str(result["error"])

    async def test_search_failure_invalid_json(self, monkeypatch):
        mod = _load_awareness_client(monkeypatch)

        class FakeClient:
            def __init__(self, timeout): pass
            async def __aenter__(self): return self
            async def __aexit__(self, *a): pass
            async def get(self, url, params):
                return _FakeBadJsonResponse(500, "internal server error")

        monkeypatch.setattr(mod.httpx, "AsyncClient", FakeClient)

        client = mod.AwarenessClient("http://awareness.test", "ws-ns")
        result = await client.search()
        assert result["success"] is False
        assert "internal server error" in str(result["error"])


# ---------------------------------------------------------------------------
# _memories_url helper
# ---------------------------------------------------------------------------

class TestMemoriesUrl:

    def test_memories_url_format(self, monkeypatch):
        mod = _load_awareness_client(monkeypatch)
        client = mod.AwarenessClient("http://awareness.test/", "my-namespace")
        # base_url strips trailing slash
        assert client._memories_url() == "http://awareness.test/api/v1/namespaces/my-namespace/memories"


# ---------------------------------------------------------------------------
# _resolve_async_client — fallback paths
# ---------------------------------------------------------------------------

class TestResolveAsyncClient:

    def test_resolve_from_httpx_directly(self, monkeypatch):
        """When httpx.AsyncClient exists, it is returned directly."""
        mod = _load_awareness_client(monkeypatch)

        fake_cls = MagicMock(name="AsyncClient")
        monkeypatch.setattr(mod.httpx, "AsyncClient", fake_cls)

        result = mod._resolve_async_client()
        assert result is fake_cls

    def test_resolve_from_tools_memory_fallback(self, monkeypatch):
        """When httpx.AsyncClient is None, falls back to tools.memory.httpx.AsyncClient."""
        mod = _load_awareness_client(monkeypatch)

        # Simulate httpx.AsyncClient being None (as when httpx unavailable)
        monkeypatch.setattr(mod.httpx, "AsyncClient", None)

        # Inject a fake tools.memory module with its own httpx mock
        fake_async_client = MagicMock(name="MemoryAsyncClient")
        fake_memory_httpx = MagicMock()
        fake_memory_httpx.AsyncClient = fake_async_client
        fake_memory_mod = MagicMock()
        fake_memory_mod.httpx = fake_memory_httpx

        monkeypatch.setitem(sys.modules, "builtin_tools.memory", fake_memory_mod)

        result = mod._resolve_async_client()
        assert result is fake_async_client

    def test_resolve_raises_when_unavailable(self, monkeypatch):
        """When both httpx and tools.memory are unavailable, raises RuntimeError."""
        mod = _load_awareness_client(monkeypatch)

        monkeypatch.setattr(mod.httpx, "AsyncClient", None)
        # Make sure tools.memory is not in sys.modules
        monkeypatch.delitem(sys.modules, "builtin_tools.memory", raising=False)

        with pytest.raises(RuntimeError, match="httpx.AsyncClient is unavailable"):
            mod._resolve_async_client()

    def test_resolve_from_tools_memory_with_none_async_client(self, monkeypatch):
        """When tools.memory.httpx.AsyncClient is None too, raises RuntimeError."""
        mod = _load_awareness_client(monkeypatch)

        monkeypatch.setattr(mod.httpx, "AsyncClient", None)

        fake_memory_httpx = MagicMock()
        fake_memory_httpx.AsyncClient = None
        fake_memory_mod = MagicMock()
        fake_memory_mod.httpx = fake_memory_httpx

        monkeypatch.setitem(sys.modules, "builtin_tools.memory", fake_memory_mod)

        with pytest.raises(RuntimeError, match="httpx.AsyncClient is unavailable"):
            mod._resolve_async_client()
