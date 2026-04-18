"""Tests for coordinator.py — get_parent_context() and get_children() functions."""

import asyncio
from unittest.mock import AsyncMock, patch, MagicMock

import pytest

from coordinator import get_parent_context, get_children, build_children_description


@pytest.mark.asyncio
async def test_get_parent_context_no_env(monkeypatch):
    """Returns empty list when PARENT_ID is not set."""
    monkeypatch.delenv("PARENT_ID", raising=False)
    result = await get_parent_context()
    assert result == []


@pytest.mark.asyncio
async def test_get_parent_context_success(monkeypatch):
    """Fetches shared context files from parent workspace via httpx."""
    monkeypatch.setenv("PARENT_ID", "parent-123")
    monkeypatch.setenv("WORKSPACE_ID", "child-456")
    monkeypatch.setenv("PLATFORM_URL", "http://localhost:8080")

    # Reload module-level constants after env change
    import coordinator
    monkeypatch.setattr(coordinator, "PLATFORM_URL", "http://localhost:8080")
    monkeypatch.setattr(coordinator, "WORKSPACE_ID", "child-456")

    mock_response = MagicMock()
    mock_response.status_code = 200
    mock_response.json.return_value = [
        {"path": "guidelines.md", "content": "Be concise."},
        {"path": "arch.md", "content": "Use microservices."},
    ]

    mock_client = AsyncMock()
    mock_client.get.return_value = mock_response
    mock_client.__aenter__ = AsyncMock(return_value=mock_client)
    mock_client.__aexit__ = AsyncMock(return_value=False)

    with patch("coordinator.httpx.AsyncClient", return_value=mock_client):
        result = await get_parent_context()

    assert len(result) == 2
    assert result[0]["path"] == "guidelines.md"
    assert result[0]["content"] == "Be concise."
    assert result[1]["path"] == "arch.md"

    # Verify the correct URL was called
    mock_client.get.assert_called_once_with(
        "http://localhost:8080/workspaces/parent-123/shared-context",
        headers={"X-Workspace-ID": "child-456"},
    )


@pytest.mark.asyncio
async def test_get_parent_context_failure(monkeypatch):
    """Returns empty list when httpx raises an exception."""
    monkeypatch.setenv("PARENT_ID", "parent-123")
    monkeypatch.setenv("WORKSPACE_ID", "child-456")

    import coordinator
    monkeypatch.setattr(coordinator, "PLATFORM_URL", "http://localhost:8080")
    monkeypatch.setattr(coordinator, "WORKSPACE_ID", "child-456")

    mock_client = AsyncMock()
    mock_client.get.side_effect = Exception("Connection refused")
    mock_client.__aenter__ = AsyncMock(return_value=mock_client)
    mock_client.__aexit__ = AsyncMock(return_value=False)

    with patch("coordinator.httpx.AsyncClient", return_value=mock_client):
        result = await get_parent_context()

    assert result == []


# ---------------------------------------------------------------------------
# get_children() tests
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_get_children_success(monkeypatch):
    """get_children() returns only peers whose parent_id matches WORKSPACE_ID."""
    import coordinator
    monkeypatch.setattr(coordinator, "PLATFORM_URL", "http://localhost:8080")
    monkeypatch.setattr(coordinator, "WORKSPACE_ID", "parent-ws")

    mock_resp = MagicMock()
    mock_resp.status_code = 200
    mock_resp.json.return_value = [
        {"id": "child-1", "parent_id": "parent-ws"},
        {"id": "peer-2", "parent_id": "other-ws"},
        {"id": "child-3", "parent_id": "parent-ws"},
    ]

    mock_client = AsyncMock()
    mock_client.__aenter__ = AsyncMock(return_value=mock_client)
    mock_client.__aexit__ = AsyncMock(return_value=False)
    mock_client.get = AsyncMock(return_value=mock_resp)

    with patch("coordinator.httpx.AsyncClient", return_value=mock_client):
        result = await get_children()

    assert len(result) == 2
    assert result[0]["id"] == "child-1"
    assert result[1]["id"] == "child-3"


@pytest.mark.asyncio
async def test_get_children_non_200(monkeypatch):
    """get_children() returns [] when the response status is not 200."""
    import coordinator
    monkeypatch.setattr(coordinator, "PLATFORM_URL", "http://localhost:8080")
    monkeypatch.setattr(coordinator, "WORKSPACE_ID", "parent-ws")

    mock_resp = MagicMock()
    mock_resp.status_code = 503

    mock_client = AsyncMock()
    mock_client.__aenter__ = AsyncMock(return_value=mock_client)
    mock_client.__aexit__ = AsyncMock(return_value=False)
    mock_client.get = AsyncMock(return_value=mock_resp)

    with patch("coordinator.httpx.AsyncClient", return_value=mock_client):
        result = await get_children()

    assert result == []


@pytest.mark.asyncio
async def test_get_children_exception(monkeypatch):
    """get_children() returns [] when httpx raises an exception."""
    import coordinator
    monkeypatch.setattr(coordinator, "PLATFORM_URL", "http://localhost:8080")
    monkeypatch.setattr(coordinator, "WORKSPACE_ID", "parent-ws")

    mock_client = AsyncMock()
    mock_client.__aenter__ = AsyncMock(return_value=mock_client)
    mock_client.__aexit__ = AsyncMock(return_value=False)
    mock_client.get = AsyncMock(side_effect=Exception("Network error"))

    with patch("coordinator.httpx.AsyncClient", return_value=mock_client):
        result = await get_children()

    assert result == []


def test_build_children_description_empty_returns_empty_string():
    """build_children_description() with empty list returns '' (covers line 72)."""
    result = build_children_description([])
    assert result == ""


def test_build_children_description_with_children():
    """build_children_description() formats children correctly."""
    children = [
        {"id": "child-1", "name": "Worker A", "description": "Does work A"},
        {"id": "child-2", "name": "Worker B"},
    ]
    result = build_children_description(children)
    assert result != ""
    assert "Coordination Rules" in result
