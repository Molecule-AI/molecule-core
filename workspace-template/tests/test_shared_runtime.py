"""Tests for shared runtime helpers used by A2A-backed executors."""

from types import SimpleNamespace
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from adapters.shared_runtime import (
    append_peer_guidance,
    build_peer_section,
    build_task_text,
    brief_task,
    extract_history,
    extract_message_text,
    format_conversation_history,
    summarize_peer_cards,
    set_current_task,
)


def _make_context(parts=None, metadata=None):
    context = MagicMock()
    context.message.parts = parts or []
    context.metadata = metadata or {}
    return context


def test_extract_message_text_prefers_text_then_root_text():
    part1 = MagicMock()
    part1.text = "Hello"
    part2 = MagicMock(spec=[])
    part2.root = SimpleNamespace(text="World")
    assert extract_message_text(_make_context([part1, part2])) == "Hello World"


def test_extract_message_text_supports_dict_parts():
    parts = [{"text": "Hello"}, {"root": {"text": "World"}}]
    assert extract_message_text(parts) == "Hello World"


def test_extract_history_and_formatting():
    ctx = _make_context(
        metadata={
            "history": [
                {"role": "user", "parts": [{"text": "First"}]},
                {"role": "agent", "parts": [{"text": "Second"}]},
            ]
        }
    )

    history = extract_history(ctx)

    assert history == [("human", "First"), ("ai", "Second")]
    assert format_conversation_history(history) == "User: First\nAgent: Second"
    assert (
        build_task_text("Current request", history)
        == "Conversation so far:\nUser: First\nAgent: Second\n\nCurrent request: Current request"
    )


def test_append_peer_guidance_is_optional():
    assert append_peer_guidance(None, "", default_text="Base", tool_name="delegate") == "Base"
    assert (
        append_peer_guidance("Base", "Peer A", default_text="Base", tool_name="delegate")
        == "Base\n\n## Peers\nPeer A\nUse delegate to communicate with them."
    )


def test_summarize_peer_cards_and_render_section():
    peers = [
        {
            "id": "peer-1",
            "status": "online",
            "agent_card": {
                "name": "Alpha",
                "skills": [{"name": "research"}, {"id": "write"}],
            },
        },
        {"id": "peer-2", "status": "offline", "agent_card": None},
    ]

    assert summarize_peer_cards(peers) == [
        {
            "id": "peer-1",
            "name": "Alpha",
            "status": "online",
            "skills": ["research", "write"],
        }
    ]

    section = build_peer_section(peers)
    assert "## Your Peers" in section
    assert "**Alpha** (id: `peer-1`, status: online)" in section
    assert "Skills: research, write" in section
    assert "delegate_to_workspace" in section


def test_brief_task_truncates_at_sixty_chars():
    assert brief_task("x" * 59) == "x" * 59
    assert brief_task("x" * 60) == "x" * 60
    assert brief_task("x" * 61) == ("x" * 60) + "..."


@pytest.mark.asyncio
async def test_set_current_task_updates_heartbeat():
    heartbeat = SimpleNamespace(current_task="", active_tasks=0)

    await set_current_task(heartbeat, "Working")
    assert heartbeat.current_task == "Working"
    assert heartbeat.active_tasks == 1

    await set_current_task(heartbeat, "")
    assert heartbeat.current_task == ""
    assert heartbeat.active_tasks == 0


@pytest.mark.asyncio
async def test_set_current_task_is_noop_for_none():
    await set_current_task(None, "Working")


# ---------------------------------------------------------------------------
# build_task_text() with no history
# ---------------------------------------------------------------------------

def test_build_task_text_no_history_returns_user_message():
    """When history is empty, build_task_text() returns the user_message directly."""
    result = build_task_text("What is the weather?", [])
    assert result == "What is the weather?"


# ---------------------------------------------------------------------------
# summarize_peer_cards() edge cases
# ---------------------------------------------------------------------------

def test_summarize_peer_cards_invalid_json_string_skipped():
    """A peer whose agent_card is an invalid JSON string is skipped entirely."""
    peers = [
        {"id": "peer-bad", "status": "online", "agent_card": "{not valid json}"},
        {
            "id": "peer-good",
            "status": "online",
            "agent_card": {"name": "Good Peer", "skills": []},
        },
    ]
    result = summarize_peer_cards(peers)
    assert len(result) == 1
    assert result[0]["id"] == "peer-good"


def test_summarize_peer_cards_json_string_not_dict_skipped():
    """A peer whose agent_card is a JSON-encoded list (not a dict) is skipped."""
    import json
    peers = [
        {"id": "peer-list", "status": "online", "agent_card": json.dumps(["skill1"])},
        {
            "id": "peer-dict",
            "status": "online",
            "agent_card": {"name": "Dict Peer", "skills": []},
        },
    ]
    result = summarize_peer_cards(peers)
    assert len(result) == 1
    assert result[0]["id"] == "peer-dict"


# ---------------------------------------------------------------------------
# set_current_task() httpx exception is swallowed
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_set_current_task_httpx_exception_is_silenced(monkeypatch):
    """set_current_task() silently ignores exceptions from the httpx heartbeat push."""
    monkeypatch.setenv("WORKSPACE_ID", "ws-test")
    monkeypatch.setenv("PLATFORM_URL", "http://platform:8080")

    mock_client = AsyncMock()
    mock_client.__aenter__ = AsyncMock(return_value=mock_client)
    mock_client.__aexit__ = AsyncMock(return_value=False)
    mock_client.post = AsyncMock(side_effect=Exception("Connection refused"))

    # httpx is imported lazily inside the function, so patch at the httpx module level
    with patch("httpx.AsyncClient", return_value=mock_client):
        # Should not raise — exception is swallowed with pass
        heartbeat = SimpleNamespace(current_task="", active_tasks=0)
        await set_current_task(heartbeat, "Doing work")

    assert heartbeat.current_task == "Doing work"
    assert heartbeat.active_tasks == 1
