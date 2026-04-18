"""Tests for a2a_mcp_server.py — handle_tool_call dispatch."""

from unittest.mock import AsyncMock, patch

import pytest


async def test_handle_tool_call_delegate_task():
    from a2a_mcp_server import handle_tool_call
    with patch("a2a_mcp_server.tool_delegate_task", new=AsyncMock(return_value="delegated")):
        result = await handle_tool_call("delegate_task", {"workspace_id": "ws1", "task": "do work"})
    assert result == "delegated"


async def test_handle_tool_call_delegate_task_async():
    from a2a_mcp_server import handle_tool_call
    with patch("a2a_mcp_server.tool_delegate_task_async", new=AsyncMock(return_value='{"task_id":"t1"}')):
        result = await handle_tool_call("delegate_task_async", {"workspace_id": "ws1", "task": "do work"})
    assert "t1" in result


async def test_handle_tool_call_check_task_status():
    from a2a_mcp_server import handle_tool_call
    with patch("a2a_mcp_server.tool_check_task_status", new=AsyncMock(return_value='{"status":"working"}')):
        result = await handle_tool_call("check_task_status", {"workspace_id": "ws1", "task_id": "t123"})
    assert "working" in result


async def test_handle_tool_call_send_message_to_user():
    from a2a_mcp_server import handle_tool_call
    with patch("a2a_mcp_server.tool_send_message_to_user", new=AsyncMock(return_value="Message sent to user")):
        result = await handle_tool_call("send_message_to_user", {"message": "Hello!"})
    assert result == "Message sent to user"


async def test_handle_tool_call_list_peers():
    from a2a_mcp_server import handle_tool_call
    with patch("a2a_mcp_server.tool_list_peers", new=AsyncMock(return_value="- peer1 (ID: ws1)")):
        result = await handle_tool_call("list_peers", {})
    assert "peer1" in result


async def test_handle_tool_call_get_workspace_info():
    from a2a_mcp_server import handle_tool_call
    with patch("a2a_mcp_server.tool_get_workspace_info", new=AsyncMock(return_value='{"id":"ws1"}')):
        result = await handle_tool_call("get_workspace_info", {})
    assert "ws1" in result


async def test_handle_tool_call_commit_memory():
    from a2a_mcp_server import handle_tool_call
    with patch("a2a_mcp_server.tool_commit_memory", new=AsyncMock(return_value='{"success":true}')):
        result = await handle_tool_call("commit_memory", {"content": "remember this", "scope": "LOCAL"})
    assert "true" in result


async def test_handle_tool_call_recall_memory():
    from a2a_mcp_server import handle_tool_call
    with patch("a2a_mcp_server.tool_recall_memory", new=AsyncMock(return_value="[LOCAL] remember this")):
        result = await handle_tool_call("recall_memory", {"query": "remember", "scope": "LOCAL"})
    assert "remember" in result


async def test_handle_tool_call_unknown_tool():
    from a2a_mcp_server import handle_tool_call
    result = await handle_tool_call("nonexistent_tool", {})
    assert "Unknown tool" in result


async def test_handle_tool_call_missing_args_defaults():
    """Test that missing args default to empty strings (defensive)."""
    from a2a_mcp_server import handle_tool_call
    with patch("a2a_mcp_server.tool_delegate_task", new=AsyncMock(return_value="ok")):
        # No workspace_id or task in arguments — defaults to ""
        result = await handle_tool_call("delegate_task", {})
    assert result == "ok"
