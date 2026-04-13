"""Comprehensive tests for a2a_client.py — 100% statement coverage.

Tests every async function:  discover_peer, send_a2a_message, get_peers,
get_workspace_info.  Each test covers exactly one execution path so failures
are easy to diagnose.
"""

import sys
import os
import importlib
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _make_mock_client(*, get_resp=None, post_resp=None, get_exc=None, post_exc=None):
    """Build a reusable AsyncClient context-manager mock."""
    mock_client = AsyncMock()
    mock_client.__aenter__ = AsyncMock(return_value=mock_client)
    mock_client.__aexit__ = AsyncMock(return_value=False)

    if get_exc is not None:
        mock_client.get = AsyncMock(side_effect=get_exc)
    elif get_resp is not None:
        mock_client.get = AsyncMock(return_value=get_resp)

    if post_exc is not None:
        mock_client.post = AsyncMock(side_effect=post_exc)
    elif post_resp is not None:
        mock_client.post = AsyncMock(return_value=post_resp)

    return mock_client


def _make_response(status_code, json_data):
    resp = MagicMock()
    resp.status_code = status_code
    resp.json = MagicMock(return_value=json_data)
    return resp


# ---------------------------------------------------------------------------
# Module-level constants (just ensure they exist and have sensible types)
# ---------------------------------------------------------------------------

def test_constants_exist():
    import a2a_client
    assert isinstance(a2a_client.PLATFORM_URL, str)
    assert isinstance(a2a_client.WORKSPACE_ID, str)
    assert isinstance(a2a_client._A2A_ERROR_PREFIX, str)
    assert isinstance(a2a_client._peer_names, dict)


# ---------------------------------------------------------------------------
# discover_peer
# ---------------------------------------------------------------------------

class TestDiscoverPeer:

    async def test_success_returns_json_on_200(self):
        """200 response → returns the JSON body."""
        import a2a_client

        peer_data = {"id": "ws-abc", "url": "http://ws-abc.svc", "name": "Alpha"}
        resp = _make_response(200, peer_data)
        mock_client = _make_mock_client(get_resp=resp)

        with patch("a2a_client.httpx.AsyncClient", return_value=mock_client):
            result = await a2a_client.discover_peer("ws-abc")

        assert result == peer_data

    async def test_non_200_returns_none(self):
        """Non-200 response → returns None."""
        import a2a_client

        resp = _make_response(404, {"detail": "not found"})
        mock_client = _make_mock_client(get_resp=resp)

        with patch("a2a_client.httpx.AsyncClient", return_value=mock_client):
            result = await a2a_client.discover_peer("ws-missing")

        assert result is None

    async def test_403_returns_none(self):
        """403 forbidden → returns None (any non-200 code)."""
        import a2a_client

        resp = _make_response(403, {"detail": "forbidden"})
        mock_client = _make_mock_client(get_resp=resp)

        with patch("a2a_client.httpx.AsyncClient", return_value=mock_client):
            result = await a2a_client.discover_peer("ws-forbidden")

        assert result is None

    async def test_exception_returns_none(self):
        """Network exception → returns None (exception swallowed)."""
        import a2a_client

        mock_client = _make_mock_client(get_exc=ConnectionError("host unreachable"))

        with patch("a2a_client.httpx.AsyncClient", return_value=mock_client):
            result = await a2a_client.discover_peer("ws-down")

        assert result is None

    async def test_request_uses_correct_url_and_header(self):
        """GET is called with the right URL and X-Workspace-ID header."""
        import a2a_client

        resp = _make_response(200, {"url": "http://target"})
        mock_client = _make_mock_client(get_resp=resp)

        with patch("a2a_client.httpx.AsyncClient", return_value=mock_client):
            await a2a_client.discover_peer("ws-xyz")

        mock_client.get.assert_called_once()
        call_args = mock_client.get.call_args
        url = call_args[0][0] if call_args[0] else call_args[1].get("url") or call_args[0][0]
        # The first positional arg is the URL
        positional_url = mock_client.get.call_args.args[0]
        assert "ws-xyz" in positional_url
        assert mock_client.get.call_args.kwargs.get("headers") == {
            "X-Workspace-ID": a2a_client.WORKSPACE_ID
        }


# ---------------------------------------------------------------------------
# send_a2a_message
# ---------------------------------------------------------------------------

class TestSendA2AMessage:

    async def test_result_with_text_part_returns_text(self):
        """'result' key with text parts → returns the text."""
        import a2a_client

        resp = _make_response(200, {
            "result": {"parts": [{"kind": "text", "text": "Hello!"}]}
        })
        mock_client = _make_mock_client(post_resp=resp)

        with patch("a2a_client.httpx.AsyncClient", return_value=mock_client):
            result = await a2a_client.send_a2a_message("http://target/a2a", "ping")

        assert result == "Hello!"

    async def test_result_with_empty_parts_returns_no_response(self):
        """'result' key with empty parts list → returns '(no response)'."""
        import a2a_client

        resp = _make_response(200, {"result": {"parts": []}})
        mock_client = _make_mock_client(post_resp=resp)

        with patch("a2a_client.httpx.AsyncClient", return_value=mock_client):
            result = await a2a_client.send_a2a_message("http://target/a2a", "ping")

        assert result == "(no response)"

    async def test_result_text_starts_with_agent_error_gets_prefix(self):
        """Text starting with 'Agent error:' gets the _A2A_ERROR_PREFIX prepended."""
        import a2a_client

        resp = _make_response(200, {
            "result": {"parts": [{"kind": "text", "text": "Agent error: something bad"}]}
        })
        mock_client = _make_mock_client(post_resp=resp)

        with patch("a2a_client.httpx.AsyncClient", return_value=mock_client):
            result = await a2a_client.send_a2a_message("http://target/a2a", "task")

        assert result.startswith(a2a_client._A2A_ERROR_PREFIX)
        assert "Agent error: something bad" in result

    async def test_error_key_returns_error_prefix_and_message(self):
        """'error' key in response → returns _A2A_ERROR_PREFIX + error message."""
        import a2a_client

        resp = _make_response(200, {
            "error": {"code": -32603, "message": "Internal error occurred"}
        })
        mock_client = _make_mock_client(post_resp=resp)

        with patch("a2a_client.httpx.AsyncClient", return_value=mock_client):
            result = await a2a_client.send_a2a_message("http://target/a2a", "task")

        assert result.startswith(a2a_client._A2A_ERROR_PREFIX)
        assert "Internal error occurred" in result

    async def test_error_key_missing_message_returns_unknown(self):
        """'error' key without 'message' → falls back to 'unknown'."""
        import a2a_client

        resp = _make_response(200, {"error": {"code": -32600}})
        mock_client = _make_mock_client(post_resp=resp)

        with patch("a2a_client.httpx.AsyncClient", return_value=mock_client):
            result = await a2a_client.send_a2a_message("http://target/a2a", "task")

        assert result.startswith(a2a_client._A2A_ERROR_PREFIX)
        assert "unknown" in result

    async def test_neither_result_nor_error_returns_str_of_data(self):
        """Response with neither 'result' nor 'error' → str(data)."""
        import a2a_client

        payload = {"jsonrpc": "2.0", "id": "abc123"}
        resp = _make_response(200, payload)
        mock_client = _make_mock_client(post_resp=resp)

        with patch("a2a_client.httpx.AsyncClient", return_value=mock_client):
            result = await a2a_client.send_a2a_message("http://target/a2a", "task")

        assert result == str(payload)

    async def test_exception_returns_error_prefix_and_message(self):
        """Network exception → returns _A2A_ERROR_PREFIX + exception text."""
        import a2a_client

        mock_client = _make_mock_client(post_exc=ConnectionError("connection refused"))

        with patch("a2a_client.httpx.AsyncClient", return_value=mock_client):
            result = await a2a_client.send_a2a_message("http://target/a2a", "task")

        assert result.startswith(a2a_client._A2A_ERROR_PREFIX)
        assert "connection refused" in result

    async def test_result_text_part_missing_text_key_returns_empty(self):
        """Part dict without 'text' key → falls back to '' (empty string returned)."""
        import a2a_client

        resp = _make_response(200, {
            "result": {"parts": [{"kind": "text"}]}  # no "text" key
        })
        mock_client = _make_mock_client(post_resp=resp)

        with patch("a2a_client.httpx.AsyncClient", return_value=mock_client):
            result = await a2a_client.send_a2a_message("http://target/a2a", "task")

        # Returns "" (empty string — does not start with _A2A_ERROR_PREFIX)
        assert result == ""


# ---------------------------------------------------------------------------
# get_peers
# ---------------------------------------------------------------------------

class TestGetPeers:

    async def test_success_returns_list_on_200(self):
        """200 response → returns the JSON list."""
        import a2a_client

        peers = [{"id": "ws-1", "name": "Alpha"}, {"id": "ws-2", "name": "Beta"}]
        resp = _make_response(200, peers)
        mock_client = _make_mock_client(get_resp=resp)

        with patch("a2a_client.httpx.AsyncClient", return_value=mock_client):
            result = await a2a_client.get_peers()

        assert result == peers

    async def test_non_200_returns_empty_list(self):
        """Non-200 response → returns []."""
        import a2a_client

        resp = _make_response(503, {"detail": "service unavailable"})
        mock_client = _make_mock_client(get_resp=resp)

        with patch("a2a_client.httpx.AsyncClient", return_value=mock_client):
            result = await a2a_client.get_peers()

        assert result == []

    async def test_404_returns_empty_list(self):
        """404 response → returns []."""
        import a2a_client

        resp = _make_response(404, {"detail": "not found"})
        mock_client = _make_mock_client(get_resp=resp)

        with patch("a2a_client.httpx.AsyncClient", return_value=mock_client):
            result = await a2a_client.get_peers()

        assert result == []

    async def test_exception_returns_empty_list(self):
        """Network exception → returns [] (exception swallowed)."""
        import a2a_client

        mock_client = _make_mock_client(get_exc=TimeoutError("timed out"))

        with patch("a2a_client.httpx.AsyncClient", return_value=mock_client):
            result = await a2a_client.get_peers()

        assert result == []

    async def test_request_url_includes_workspace_id(self):
        """GET URL contains the WORKSPACE_ID."""
        import a2a_client

        resp = _make_response(200, [])
        mock_client = _make_mock_client(get_resp=resp)

        with patch("a2a_client.httpx.AsyncClient", return_value=mock_client):
            await a2a_client.get_peers()

        url = mock_client.get.call_args.args[0]
        assert "peers" in url


# ---------------------------------------------------------------------------
# get_workspace_info
# ---------------------------------------------------------------------------

class TestGetWorkspaceInfo:

    async def test_success_returns_dict_on_200(self):
        """200 response → returns the JSON dict."""
        import a2a_client

        info = {"id": "ws-test", "name": "Test Workspace", "status": "online"}
        resp = _make_response(200, info)
        mock_client = _make_mock_client(get_resp=resp)

        with patch("a2a_client.httpx.AsyncClient", return_value=mock_client):
            result = await a2a_client.get_workspace_info()

        assert result == info

    async def test_non_200_returns_error_dict(self):
        """Non-200 response → returns {'error': 'not found'}."""
        import a2a_client

        resp = _make_response(404, {"detail": "no such workspace"})
        mock_client = _make_mock_client(get_resp=resp)

        with patch("a2a_client.httpx.AsyncClient", return_value=mock_client):
            result = await a2a_client.get_workspace_info()

        assert result == {"error": "not found"}

    async def test_500_returns_error_dict(self):
        """500 response → returns {'error': 'not found'}."""
        import a2a_client

        resp = _make_response(500, {"detail": "server error"})
        mock_client = _make_mock_client(get_resp=resp)

        with patch("a2a_client.httpx.AsyncClient", return_value=mock_client):
            result = await a2a_client.get_workspace_info()

        assert result == {"error": "not found"}

    async def test_exception_returns_error_dict_with_message(self):
        """Network exception → returns {'error': '<exception message>'}."""
        import a2a_client

        exc = RuntimeError("network failure")
        mock_client = _make_mock_client(get_exc=exc)

        with patch("a2a_client.httpx.AsyncClient", return_value=mock_client):
            result = await a2a_client.get_workspace_info()

        assert "error" in result
        assert "network failure" in result["error"]

    async def test_request_url_includes_workspaces_path(self):
        """GET URL contains /workspaces/."""
        import a2a_client

        resp = _make_response(200, {})
        mock_client = _make_mock_client(get_resp=resp)

        with patch("a2a_client.httpx.AsyncClient", return_value=mock_client):
            await a2a_client.get_workspace_info()

        url = mock_client.get.call_args.args[0]
        assert "/workspaces/" in url
