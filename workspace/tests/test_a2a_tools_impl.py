"""Comprehensive tests for a2a_tools.py (root-level) — targeting 100% coverage.

Every async function is tested across its distinct execution paths:
    report_activity, tool_delegate_task, tool_delegate_task_async,
    tool_check_task_status, tool_send_message_to_user, tool_list_peers,
    tool_get_workspace_info, tool_commit_memory, tool_recall_memory.

Patching strategy
-----------------
* httpx.AsyncClient         — patched at ``a2a_tools.httpx.AsyncClient``
* a2a_client helper funcs   — patched at ``a2a_tools.<name>`` (they were
  imported with ``from a2a_client import ...``, so the name lives in the
  a2a_tools module namespace).
"""

import json
import sys
from unittest.mock import AsyncMock, MagicMock, patch

import httpx
import pytest


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _make_http_mock(*, post_resp=None, get_resp=None,
                    post_exc=None, get_exc=None):
    """Return a mock AsyncClient that behaves as an async context manager."""
    mc = AsyncMock()
    mc.__aenter__ = AsyncMock(return_value=mc)
    mc.__aexit__ = AsyncMock(return_value=False)

    if post_exc is not None:
        mc.post = AsyncMock(side_effect=post_exc)
    elif post_resp is not None:
        mc.post = AsyncMock(return_value=post_resp)
    else:
        mc.post = AsyncMock(return_value=_resp(200, {}))

    if get_exc is not None:
        mc.get = AsyncMock(side_effect=get_exc)
    elif get_resp is not None:
        mc.get = AsyncMock(return_value=get_resp)
    else:
        mc.get = AsyncMock(return_value=_resp(200, {}))

    return mc


def _resp(status_code, payload, text=None):
    """Create a lightweight mock HTTP response."""
    r = MagicMock()
    r.status_code = status_code
    r.json = MagicMock(return_value=payload)
    r.text = text or str(payload)
    return r


# ---------------------------------------------------------------------------
# report_activity
# ---------------------------------------------------------------------------

class TestReportActivity:

    async def test_posts_activity_without_summary(self):
        """Activity with no summary should NOT fire the heartbeat POST."""
        import a2a_tools

        mc = _make_http_mock()
        with patch("a2a_tools.httpx.AsyncClient", return_value=mc):
            await a2a_tools.report_activity("a2a_send", target_id="ws-1")

        # Only one POST (the activity one — heartbeat skipped because summary="")
        mc.post.assert_called_once()

    async def test_posts_activity_and_heartbeat_when_summary_set(self):
        """With a non-empty summary, both activity and heartbeat POST are fired."""
        import a2a_tools

        mc = _make_http_mock()
        with patch("a2a_tools.httpx.AsyncClient", return_value=mc):
            await a2a_tools.report_activity(
                "a2a_send", target_id="ws-1", summary="Delegating to Alpha"
            )

        assert mc.post.call_count == 2

    async def test_includes_task_text_in_payload_when_provided(self):
        """task_text non-empty → request_body added to POST payload."""
        import a2a_tools

        mc = _make_http_mock()
        with patch("a2a_tools.httpx.AsyncClient", return_value=mc):
            await a2a_tools.report_activity(
                "a2a_send", target_id="ws-1", task_text="do something"
            )

        call_kwargs = mc.post.call_args.kwargs
        payload = call_kwargs.get("json") or mc.post.call_args.args[1] if mc.post.call_args.args else None
        if payload is None:
            payload = mc.post.call_args[1].get("json")
        assert payload is not None
        assert "request_body" in payload

    async def test_includes_response_text_in_payload_when_provided(self):
        """response_text non-empty → response_body added to POST payload."""
        import a2a_tools

        mc = _make_http_mock()
        with patch("a2a_tools.httpx.AsyncClient", return_value=mc):
            await a2a_tools.report_activity(
                "a2a_receive", target_id="ws-1", response_text="done"
            )

        call_kwargs = mc.post.call_args.kwargs
        payload = call_kwargs.get("json")
        assert payload is not None
        assert "response_body" in payload

    async def test_exception_is_silently_swallowed(self):
        """Exceptions inside report_activity are silently swallowed (best-effort)."""
        import a2a_tools

        mc = _make_http_mock(post_exc=RuntimeError("platform down"))
        with patch("a2a_tools.httpx.AsyncClient", return_value=mc):
            # Must not raise
            await a2a_tools.report_activity("a2a_send", summary="test")


# ---------------------------------------------------------------------------
# tool_delegate_task
# ---------------------------------------------------------------------------

class TestToolDelegateTask:

    async def test_empty_workspace_id_returns_error(self):
        import a2a_tools
        result = await a2a_tools.tool_delegate_task("", "do task")
        assert "Error" in result
        assert "required" in result

    async def test_empty_task_returns_error(self):
        import a2a_tools
        result = await a2a_tools.tool_delegate_task("ws-1", "")
        assert "Error" in result
        assert "required" in result

    async def test_both_empty_returns_error(self):
        import a2a_tools
        result = await a2a_tools.tool_delegate_task("", "")
        assert "Error" in result

    async def test_peer_not_found_returns_error(self):
        import a2a_tools
        with patch("a2a_tools.discover_peer", return_value=None):
            result = await a2a_tools.tool_delegate_task("ws-missing", "task")
        assert "not found" in result or "Error" in result

    async def test_peer_has_no_url_returns_error(self):
        import a2a_tools
        with patch("a2a_tools.discover_peer", return_value={"id": "ws-1", "url": ""}):
            mc = _make_http_mock()
            with patch("a2a_tools.httpx.AsyncClient", return_value=mc):
                result = await a2a_tools.tool_delegate_task("ws-1", "task")
        assert "no URL" in result or "Error" in result

    async def test_success_returns_result_text(self):
        """Happy path: peer found with URL, A2A returns a result."""
        import a2a_tools

        peer = {"id": "ws-1", "url": "http://ws-1.svc/a2a", "name": "Worker"}
        with patch("a2a_tools.discover_peer", return_value=peer), \
             patch("a2a_tools.send_a2a_message", return_value="Task completed!"), \
             patch("a2a_tools.report_activity", new=AsyncMock()):
            result = await a2a_tools.tool_delegate_task("ws-1", "do something")

        assert result == "Task completed!"

    async def test_error_response_returns_delegation_failed_message(self):
        """When send_a2a_message returns _A2A_ERROR_PREFIX text, delegation fails."""
        import a2a_tools

        peer = {"id": "ws-1", "url": "http://ws-1.svc/a2a", "name": "Worker"}
        error_msg = f"{a2a_tools._A2A_ERROR_PREFIX}Agent error: something bad"
        with patch("a2a_tools.discover_peer", return_value=peer), \
             patch("a2a_tools.send_a2a_message", return_value=error_msg), \
             patch("a2a_tools.report_activity", new=AsyncMock()):
            result = await a2a_tools.tool_delegate_task("ws-1", "do something")

        assert "DELEGATION FAILED" in result
        assert "Worker" in result

    async def test_peer_name_cached_from_peer_names_dict(self):
        """When peer dict has no 'name' but _peer_names cache has one, uses cached name."""
        import a2a_tools

        # Pre-populate the cache
        a2a_tools._peer_names["ws-cached"] = "CachedName"
        peer = {"id": "ws-cached", "url": "http://ws-cached.svc/a2a"}  # no 'name'
        with patch("a2a_tools.discover_peer", return_value=peer), \
             patch("a2a_tools.send_a2a_message", return_value="done"), \
             patch("a2a_tools.report_activity", new=AsyncMock()):
            result = await a2a_tools.tool_delegate_task("ws-cached", "task")

        assert result == "done"

    async def test_peer_name_falls_back_to_id_prefix(self):
        """When peer has no name and cache is empty, name = first 8 chars of workspace_id."""
        import a2a_tools

        # Ensure not in cache
        a2a_tools._peer_names.pop("ws-nona000", None)
        peer = {"id": "ws-nona000", "url": "http://x.svc/a2a"}  # no 'name'
        with patch("a2a_tools.discover_peer", return_value=peer), \
             patch("a2a_tools.send_a2a_message", return_value="ok"), \
             patch("a2a_tools.report_activity", new=AsyncMock()):
            result = await a2a_tools.tool_delegate_task("ws-nona000", "task")

        assert result == "ok"
        # Cache should now have been set
        assert a2a_tools._peer_names.get("ws-nona000") is not None


# ---------------------------------------------------------------------------
# tool_delegate_task_async
# ---------------------------------------------------------------------------

class TestToolDelegateTaskAsync:

    async def test_empty_workspace_id_returns_error(self):
        import a2a_tools
        result = await a2a_tools.tool_delegate_task_async("", "task")
        assert "Error" in result
        assert "required" in result

    async def test_empty_task_returns_error(self):
        import a2a_tools
        result = await a2a_tools.tool_delegate_task_async("ws-1", "")
        assert "Error" in result
        assert "required" in result

    async def test_platform_delegation_success(self):
        """POST /delegate succeeds → returns JSON with status=delegated."""
        import a2a_tools

        mc = _make_http_mock(post_resp=_resp(202, {"delegation_id": "d-123", "status": "delegated"}))
        with patch("a2a_tools.httpx.AsyncClient", return_value=mc):
            result = await a2a_tools.tool_delegate_task_async("ws-1", "do task")

        data = json.loads(result)
        assert data["status"] == "delegated"
        assert data["workspace_id"] == "ws-1"
        assert data["delegation_id"] == "d-123"

    async def test_platform_delegation_failure(self):
        """POST /delegate fails → returns error string."""
        import a2a_tools

        mc = _make_http_mock(post_resp=_resp(500, {"error": "internal"}))
        with patch("a2a_tools.httpx.AsyncClient", return_value=mc):
            result = await a2a_tools.tool_delegate_task_async("ws-1", "do task")

        assert "Error" in result

    async def test_timeout_returns_error(self):
        """httpx exception → returns error string."""
        import a2a_tools

        mc = _make_http_mock(post_exc=httpx.ConnectError("connection refused"))
        with patch("a2a_tools.httpx.AsyncClient", return_value=mc):
            result = await a2a_tools.tool_delegate_task_async("ws-1", "do task")

        assert "Error" in result or "failed" in result.lower()


# ---------------------------------------------------------------------------
# tool_check_task_status
# ---------------------------------------------------------------------------

class TestToolCheckTaskStatus:

    async def test_returns_delegations_list(self):
        """GET /delegations succeeds → returns delegation summary."""
        import a2a_tools

        delegations = [
            {"delegation_id": "d-1", "target_id": "ws-t", "status": "completed", "summary": "done", "response_preview": "ok"},
            {"delegation_id": "d-2", "target_id": "ws-u", "status": "pending", "summary": "waiting"},
        ]
        mc = _make_http_mock(get_resp=_resp(200, delegations))
        with patch("a2a_tools.httpx.AsyncClient", return_value=mc):
            result = await a2a_tools.tool_check_task_status("ws-1", "")

        data = json.loads(result)
        assert data["count"] == 2
        assert data["delegations"][0]["status"] == "completed"

    async def test_filter_by_delegation_id(self):
        """Filter by specific delegation_id."""
        import a2a_tools

        delegations = [
            {"delegation_id": "d-1", "status": "completed", "response_preview": "result here"},
            {"delegation_id": "d-2", "status": "pending"},
        ]
        mc = _make_http_mock(get_resp=_resp(200, delegations))
        with patch("a2a_tools.httpx.AsyncClient", return_value=mc):
            result = await a2a_tools.tool_check_task_status("ws-1", "d-1")

        data = json.loads(result)
        assert data["delegation_id"] == "d-1"
        assert data["status"] == "completed"

    async def test_not_found_delegation_id(self):
        """Delegation ID not in results → returns not_found."""
        import a2a_tools

        mc = _make_http_mock(get_resp=_resp(200, []))
        with patch("a2a_tools.httpx.AsyncClient", return_value=mc):
            result = await a2a_tools.tool_check_task_status("ws-1", "d-missing")

        data = json.loads(result)
        assert data["status"] == "not_found"

    async def test_api_error_returns_error_string(self):
        """Platform API failure → returns error string."""
        import a2a_tools

        mc = _make_http_mock(get_resp=_resp(500, {"error": "db down"}))
        with patch("a2a_tools.httpx.AsyncClient", return_value=mc):
            result = await a2a_tools.tool_check_task_status("ws-1", "d-1")

        assert "Error" in result or "failed" in result.lower()


# ---------------------------------------------------------------------------
# tool_send_message_to_user
# ---------------------------------------------------------------------------

class TestToolSendMessageToUser:

    async def test_empty_message_returns_error(self):
        import a2a_tools
        result = await a2a_tools.tool_send_message_to_user("")
        assert "Error" in result
        assert "required" in result

    async def test_success_200_returns_sent_message(self):
        import a2a_tools
        mc = _make_http_mock(post_resp=_resp(200, {}))
        with patch("a2a_tools.httpx.AsyncClient", return_value=mc):
            result = await a2a_tools.tool_send_message_to_user("Hello user!")
        assert result == "Message sent to user"

    async def test_non_200_returns_status_code_in_error(self):
        import a2a_tools
        mc = _make_http_mock(post_resp=_resp(503, {}))
        with patch("a2a_tools.httpx.AsyncClient", return_value=mc):
            result = await a2a_tools.tool_send_message_to_user("Hello user!")
        assert "503" in result
        assert "Error" in result

    async def test_exception_returns_error_message(self):
        import a2a_tools
        mc = _make_http_mock(post_exc=RuntimeError("platform unreachable"))
        with patch("a2a_tools.httpx.AsyncClient", return_value=mc):
            result = await a2a_tools.tool_send_message_to_user("Hi!")
        assert "Error sending message" in result
        assert "platform unreachable" in result


# ---------------------------------------------------------------------------
# tool_list_peers
# ---------------------------------------------------------------------------

class TestToolListPeers:

    async def test_no_peers_returns_isolated_message(self):
        import a2a_tools
        with patch("a2a_tools.get_peers", return_value=[]):
            result = await a2a_tools.tool_list_peers()
        assert "No peers available" in result

    async def test_peers_returned_formatted_lines(self):
        """Peers list is formatted as '- name (ID: ..., status: ..., role: ...)'."""
        import a2a_tools

        peers = [
            {"id": "ws-1", "name": "Alpha", "status": "online", "role": "worker"},
            {"id": "ws-2", "name": "Beta", "status": "idle", "role": "analyst"},
        ]
        with patch("a2a_tools.get_peers", return_value=peers):
            result = await a2a_tools.tool_list_peers()

        assert "Alpha" in result
        assert "ws-1" in result
        assert "online" in result
        assert "worker" in result
        assert "Beta" in result
        assert "ws-2" in result

    async def test_peer_names_cached_after_list(self):
        """After tool_list_peers, _peer_names should contain the listed peer IDs."""
        import a2a_tools

        # Clear any prior cache entries for these IDs
        a2a_tools._peer_names.pop("ws-cache-test", None)
        peers = [{"id": "ws-cache-test", "name": "CacheMe", "status": "online", "role": "w"}]
        with patch("a2a_tools.get_peers", return_value=peers):
            await a2a_tools.tool_list_peers()

        assert a2a_tools._peer_names.get("ws-cache-test") == "CacheMe"

    async def test_peers_missing_optional_fields_still_format(self):
        """Peers with missing status/role use 'unknown'/'empty string' gracefully."""
        import a2a_tools

        peers = [{"id": "ws-3", "name": "Gamma"}]  # no status, no role
        with patch("a2a_tools.get_peers", return_value=peers):
            result = await a2a_tools.tool_list_peers()

        assert "Gamma" in result
        assert "ws-3" in result
        assert "unknown" in result  # default status


# ---------------------------------------------------------------------------
# tool_get_workspace_info
# ---------------------------------------------------------------------------

class TestToolGetWorkspaceInfo:

    async def test_returns_json_dumped_info(self):
        import a2a_tools

        info = {"id": "ws-test", "name": "My Workspace", "status": "online"}
        with patch("a2a_tools.get_workspace_info", return_value=info):
            result = await a2a_tools.tool_get_workspace_info()

        parsed = json.loads(result)
        assert parsed == info

    async def test_returns_error_dict_as_json(self):
        import a2a_tools

        with patch("a2a_tools.get_workspace_info", return_value={"error": "not found"}):
            result = await a2a_tools.tool_get_workspace_info()

        parsed = json.loads(result)
        assert parsed == {"error": "not found"}


# ---------------------------------------------------------------------------
# tool_commit_memory
# ---------------------------------------------------------------------------

class TestToolCommitMemory:

    async def test_empty_content_returns_error(self):
        import a2a_tools
        result = await a2a_tools.tool_commit_memory("")
        assert "Error" in result
        assert "required" in result

    async def test_scope_normalized_to_uppercase(self):
        """Scope 'local' → 'LOCAL', included in POST payload."""
        import a2a_tools

        mc = _make_http_mock(post_resp=_resp(201, {"id": "mem-1"}))
        with patch("a2a_tools.httpx.AsyncClient", return_value=mc):
            result = await a2a_tools.tool_commit_memory("Remember this", scope="local")

        data = json.loads(result)
        assert data["scope"] == "LOCAL"
        assert data["success"] is True

    async def test_invalid_scope_normalizes_to_local(self):
        """Unknown scope string defaults to 'LOCAL'."""
        import a2a_tools

        mc = _make_http_mock(post_resp=_resp(200, {"id": "mem-2"}))
        with patch("a2a_tools.httpx.AsyncClient", return_value=mc):
            result = await a2a_tools.tool_commit_memory("Remember this", scope="INVALID")

        data = json.loads(result)
        assert data["scope"] == "LOCAL"

    async def test_team_scope_accepted(self):
        import a2a_tools

        mc = _make_http_mock(post_resp=_resp(200, {"id": "mem-3"}))
        with patch("a2a_tools.httpx.AsyncClient", return_value=mc):
            result = await a2a_tools.tool_commit_memory("Team info", scope="TEAM")

        data = json.loads(result)
        assert data["scope"] == "TEAM"

    async def test_global_scope_accepted(self):
        import a2a_tools

        mc = _make_http_mock(post_resp=_resp(201, {"id": "mem-4"}))
        with patch("a2a_tools.httpx.AsyncClient", return_value=mc):
            result = await a2a_tools.tool_commit_memory("Global info", scope="GLOBAL")

        data = json.loads(result)
        assert data["scope"] == "GLOBAL"

    async def test_success_200_returns_success_json(self):
        import a2a_tools

        mc = _make_http_mock(post_resp=_resp(200, {"id": "mem-5"}))
        with patch("a2a_tools.httpx.AsyncClient", return_value=mc):
            result = await a2a_tools.tool_commit_memory("info")

        data = json.loads(result)
        assert data["success"] is True
        assert data["id"] == "mem-5"

    async def test_success_201_returns_success_json(self):
        import a2a_tools

        mc = _make_http_mock(post_resp=_resp(201, {"id": "mem-6"}))
        with patch("a2a_tools.httpx.AsyncClient", return_value=mc):
            result = await a2a_tools.tool_commit_memory("info")

        data = json.loads(result)
        assert data["success"] is True

    async def test_error_response_returns_error_string(self):
        """Non-200/201 → returns 'Error: <error field from JSON>'."""
        import a2a_tools

        mc = _make_http_mock(post_resp=_resp(400, {"error": "bad request payload"}))
        with patch("a2a_tools.httpx.AsyncClient", return_value=mc):
            result = await a2a_tools.tool_commit_memory("info")

        assert "Error" in result
        assert "bad request payload" in result

    async def test_exception_returns_error_message(self):
        import a2a_tools

        mc = _make_http_mock(post_exc=RuntimeError("storage failure"))
        with patch("a2a_tools.httpx.AsyncClient", return_value=mc):
            result = await a2a_tools.tool_commit_memory("info")

        assert "Error saving memory" in result
        assert "storage failure" in result


# ---------------------------------------------------------------------------
# tool_recall_memory
# ---------------------------------------------------------------------------

class TestToolRecallMemory:

    async def test_list_response_with_memories_returns_formatted_lines(self):
        import a2a_tools

        memories = [
            {"scope": "LOCAL", "content": "The capital of France is Paris"},
            {"scope": "TEAM", "content": "We use Python 3.11"},
        ]
        mc = _make_http_mock(get_resp=_resp(200, memories))
        with patch("a2a_tools.httpx.AsyncClient", return_value=mc):
            result = await a2a_tools.tool_recall_memory(query="capital")

        assert "[LOCAL]" in result
        assert "Paris" in result
        assert "[TEAM]" in result
        assert "Python 3.11" in result

    async def test_empty_list_response_returns_no_memories_found(self):
        import a2a_tools

        mc = _make_http_mock(get_resp=_resp(200, []))
        with patch("a2a_tools.httpx.AsyncClient", return_value=mc):
            result = await a2a_tools.tool_recall_memory(query="anything")

        assert result == "No memories found."

    async def test_non_list_response_returns_json_dumped(self):
        """When server returns a dict instead of a list, it's JSON-dumped."""
        import a2a_tools

        payload = {"error": "search unavailable"}
        mc = _make_http_mock(get_resp=_resp(200, payload))
        with patch("a2a_tools.httpx.AsyncClient", return_value=mc):
            result = await a2a_tools.tool_recall_memory()

        parsed = json.loads(result)
        assert parsed == payload

    async def test_exception_returns_error_message(self):
        import a2a_tools

        mc = _make_http_mock(get_exc=RuntimeError("search service down"))
        with patch("a2a_tools.httpx.AsyncClient", return_value=mc):
            result = await a2a_tools.tool_recall_memory(query="test")

        assert "Error recalling memory" in result
        assert "search service down" in result

    async def test_query_and_scope_passed_as_params(self):
        """query and scope are both forwarded as GET params."""
        import a2a_tools

        mc = _make_http_mock(get_resp=_resp(200, []))
        with patch("a2a_tools.httpx.AsyncClient", return_value=mc):
            await a2a_tools.tool_recall_memory(query="paris", scope="local")

        call_kwargs = mc.get.call_args.kwargs
        params = call_kwargs.get("params", {})
        assert params.get("q") == "paris"
        assert params.get("scope") == "LOCAL"  # uppercased

    async def test_no_query_or_scope_sends_empty_params(self):
        """With no query/scope, params dict is empty (no keys added)."""
        import a2a_tools

        mc = _make_http_mock(get_resp=_resp(200, []))
        with patch("a2a_tools.httpx.AsyncClient", return_value=mc):
            await a2a_tools.tool_recall_memory()

        call_kwargs = mc.get.call_args.kwargs
        params = call_kwargs.get("params", {})
        assert params == {}

    async def test_scope_only_uppercased_in_params(self):
        """scope without query → only 'scope' key in params, uppercased."""
        import a2a_tools

        mc = _make_http_mock(get_resp=_resp(200, []))
        with patch("a2a_tools.httpx.AsyncClient", return_value=mc):
            await a2a_tools.tool_recall_memory(scope="team")

        call_kwargs = mc.get.call_args.kwargs
        params = call_kwargs.get("params", {})
        assert "q" not in params
        assert params.get("scope") == "TEAM"
