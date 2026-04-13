"""Tests for consolidation.py — ConsolidationLoop memory summarization."""

import asyncio
import logging
from unittest.mock import AsyncMock, MagicMock, patch

import pytest
import httpx

import consolidation as consolidation_mod
from consolidation import ConsolidationLoop, CONSOLIDATION_INTERVAL, CONSOLIDATION_THRESHOLD


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _make_http_client_mock(get_status=200, get_json=None, post_status=200):
    """Build an AsyncMock httpx.AsyncClient with configurable responses."""
    client = AsyncMock()

    get_resp = MagicMock()
    get_resp.status_code = get_status
    get_resp.json = MagicMock(return_value=get_json or [])

    post_resp = MagicMock()
    post_resp.status_code = post_status

    client.get = AsyncMock(return_value=get_resp)
    client.post = AsyncMock(return_value=post_resp)
    client.delete = AsyncMock(return_value=MagicMock(status_code=204))

    client.__aenter__ = AsyncMock(return_value=client)
    client.__aexit__ = AsyncMock(return_value=False)
    return client


def _memories(n):
    """Return a list of n fake memory dicts."""
    return [{"id": f"mem-{i}", "content": f"fact {i}"} for i in range(n)]


# ---------------------------------------------------------------------------
# __init__
# ---------------------------------------------------------------------------

def test_init_default_agent():
    """Constructor stores agent=None and _running=False by default."""
    loop = ConsolidationLoop()
    assert loop.agent is None
    assert loop._running is False


def test_init_with_agent():
    """Constructor stores provided agent reference."""
    agent = MagicMock()
    loop = ConsolidationLoop(agent=agent)
    assert loop.agent is agent


# ---------------------------------------------------------------------------
# stop()
# ---------------------------------------------------------------------------

def test_stop_sets_running_false():
    """stop() sets _running to False."""
    loop = ConsolidationLoop()
    loop._running = True
    loop.stop()
    assert loop._running is False


# ---------------------------------------------------------------------------
# start()
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_start_sets_running_true():
    """start() sets _running=True before entering the loop."""
    loop = ConsolidationLoop()

    consolidate_calls = [0]

    async def fake_sleep(secs):
        consolidate_calls[0] += 1
        loop._running = False  # Exit after first iteration

    with patch("consolidation.asyncio.sleep", side_effect=fake_sleep):
        # _consolidate will be called but we don't care about its result
        with patch.object(loop, "_consolidate", new_callable=AsyncMock):
            await loop.start()

    assert consolidate_calls[0] == 1


@pytest.mark.asyncio
async def test_start_exits_when_running_false_after_sleep():
    """Loop exits immediately when _running is set to False after the sleep."""
    loop = ConsolidationLoop()

    async def fake_sleep(secs):
        loop._running = False  # Mark stopped; the 'if not self._running: break' fires

    with patch("consolidation.asyncio.sleep", side_effect=fake_sleep):
        with patch.object(loop, "_consolidate", new_callable=AsyncMock) as mock_consolidate:
            await loop.start()

    # _consolidate should NOT be called because the break happens before it
    mock_consolidate.assert_not_called()


@pytest.mark.asyncio
async def test_start_logs_startup_info(caplog):
    """start() emits an INFO log naming interval and threshold."""
    loop = ConsolidationLoop()

    async def fake_sleep(secs):
        loop._running = False

    with patch("consolidation.asyncio.sleep", side_effect=fake_sleep):
        with patch.object(loop, "_consolidate", new_callable=AsyncMock):
            with caplog.at_level(logging.INFO, logger="consolidation"):
                await loop.start()

    assert "consolidation loop started" in caplog.text.lower()


@pytest.mark.asyncio
async def test_start_catches_consolidate_exception(caplog):
    """start() catches exceptions from _consolidate and logs a warning."""
    loop = ConsolidationLoop()
    call_count = [0]

    async def fake_sleep(secs):
        call_count[0] += 1
        if call_count[0] >= 2:
            loop._running = False

    async def bad_consolidate():
        raise RuntimeError("consolidation exploded")

    with patch("consolidation.asyncio.sleep", side_effect=fake_sleep):
        with patch.object(loop, "_consolidate", side_effect=bad_consolidate):
            with caplog.at_level(logging.WARNING, logger="consolidation"):
                await loop.start()

    assert "Consolidation error" in caplog.text


@pytest.mark.asyncio
async def test_start_multiple_iterations():
    """start() runs _consolidate on each wake-up until stopped."""
    loop = ConsolidationLoop()
    call_count = [0]
    consolidate_calls = [0]

    async def fake_sleep(secs):
        call_count[0] += 1
        if call_count[0] >= 3:
            loop._running = False

    async def fake_consolidate():
        consolidate_calls[0] += 1

    with patch("consolidation.asyncio.sleep", side_effect=fake_sleep):
        with patch.object(loop, "_consolidate", side_effect=fake_consolidate):
            await loop.start()

    assert consolidate_calls[0] == 2  # 3 sleeps, 3rd sets _running=False → 2 consolidations


# ---------------------------------------------------------------------------
# _consolidate() — HTTP error path
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_consolidate_returns_on_non_200(monkeypatch):
    """_consolidate exits early when the GET memories response is not 200."""
    loop = ConsolidationLoop()
    mock_client = _make_http_client_mock(get_status=500, get_json=[])

    with patch("consolidation.httpx.AsyncClient", return_value=mock_client):
        await loop._consolidate()  # Should not raise

    mock_client.post.assert_not_called()


# ---------------------------------------------------------------------------
# _consolidate() — below threshold
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_consolidate_below_threshold_does_nothing(monkeypatch):
    """_consolidate does not summarize when memory count is below threshold."""
    loop = ConsolidationLoop()
    # CONSOLIDATION_THRESHOLD is at least 1; use 0 memories to stay below
    mock_client = _make_http_client_mock(get_status=200, get_json=[])

    with patch("consolidation.httpx.AsyncClient", return_value=mock_client):
        await loop._consolidate()

    mock_client.post.assert_not_called()


@pytest.mark.asyncio
async def test_consolidate_exactly_at_threshold_triggers(monkeypatch):
    """_consolidate runs when len(memories) == CONSOLIDATION_THRESHOLD."""
    loop = ConsolidationLoop(agent=None)
    mems = _memories(CONSOLIDATION_THRESHOLD)
    mock_client = _make_http_client_mock(get_status=200, get_json=mems)

    with patch("consolidation.httpx.AsyncClient", return_value=mock_client):
        await loop._consolidate()

    # Fallback path (no agent) should have called POST
    mock_client.post.assert_called_once()


# ---------------------------------------------------------------------------
# _consolidate() — no agent (concatenation fallback)
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_consolidate_no_agent_posts_concatenated_memory():
    """Without an agent, _consolidate POSTs a concatenated TEAM memory."""
    loop = ConsolidationLoop(agent=None)
    mems = _memories(CONSOLIDATION_THRESHOLD)
    mock_client = _make_http_client_mock(get_status=200, get_json=mems)

    with patch("consolidation.httpx.AsyncClient", return_value=mock_client):
        await loop._consolidate()

    mock_client.post.assert_called_once()
    call_kwargs = mock_client.post.call_args[1]
    body = call_kwargs["json"]
    assert body["scope"] == "TEAM"
    assert body["content"].startswith("[Consolidated]")
    assert "fact 0" in body["content"]


@pytest.mark.asyncio
async def test_consolidate_no_agent_concatenates_up_to_20():
    """Without an agent, _consolidate only uses the first 20 memories."""
    loop = ConsolidationLoop(agent=None)
    mems = _memories(25)  # More than 20
    mock_client = _make_http_client_mock(get_status=200, get_json=mems)

    with patch("consolidation.httpx.AsyncClient", return_value=mock_client):
        await loop._consolidate()

    body = mock_client.post.call_args[1]["json"]
    # "fact 20" and "fact 21"... should NOT appear if only first 20 are used
    assert "fact 20" not in body["content"]
    assert "fact 19" in body["content"]


# ---------------------------------------------------------------------------
# _consolidate() — with agent, success path
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_consolidate_with_agent_success_stores_summary_and_deletes():
    """With an agent that returns a summary, _consolidate POSTs and DELETEs."""
    agent = AsyncMock()
    summary_msg = MagicMock()
    summary_msg.content = "Key fact about the project."
    summary_msg.type = "ai"

    agent.ainvoke = AsyncMock(return_value={"messages": [summary_msg]})

    loop = ConsolidationLoop(agent=agent)
    mems = _memories(CONSOLIDATION_THRESHOLD)
    mock_client = _make_http_client_mock(get_status=200, get_json=mems)

    with patch("consolidation.httpx.AsyncClient", return_value=mock_client):
        await loop._consolidate()

    # POST the consolidated memory
    mock_client.post.assert_called_once()
    body = mock_client.post.call_args[1]["json"]
    assert "[Consolidated]" in body["content"]
    assert "Key fact about the project." in body["content"]
    assert body["scope"] == "TEAM"

    # DELETE each original memory
    assert mock_client.delete.call_count == len(mems)


@pytest.mark.asyncio
async def test_consolidate_with_agent_picks_last_non_human_message():
    """_consolidate uses the last non-human message as the summary."""
    agent = AsyncMock()

    human_msg = MagicMock()
    human_msg.content = "Summarize this."
    human_msg.type = "human"

    ai_msg_1 = MagicMock()
    ai_msg_1.content = "First AI response."
    ai_msg_1.type = "ai"

    ai_msg_2 = MagicMock()
    ai_msg_2.content = "Second AI response."
    ai_msg_2.type = "ai"

    # reversed(messages) → ai_msg_2 is found first
    agent.ainvoke = AsyncMock(return_value={"messages": [human_msg, ai_msg_1, ai_msg_2]})

    loop = ConsolidationLoop(agent=agent)
    mems = _memories(CONSOLIDATION_THRESHOLD)
    mock_client = _make_http_client_mock(get_status=200, get_json=mems)

    with patch("consolidation.httpx.AsyncClient", return_value=mock_client):
        await loop._consolidate()

    body = mock_client.post.call_args[1]["json"]
    assert "Second AI response." in body["content"]


@pytest.mark.asyncio
async def test_consolidate_with_agent_empty_messages_falls_back():
    """Agent returning no usable messages triggers the concatenation fallback."""
    agent = AsyncMock()
    agent.ainvoke = AsyncMock(return_value={"messages": []})

    loop = ConsolidationLoop(agent=agent)
    mems = _memories(CONSOLIDATION_THRESHOLD)
    mock_client = _make_http_client_mock(get_status=200, get_json=mems)

    with patch("consolidation.httpx.AsyncClient", return_value=mock_client):
        await loop._consolidate()

    # Fallback should still POST exactly once
    mock_client.post.assert_called_once()
    body = mock_client.post.call_args[1]["json"]
    assert "[Consolidated]" in body["content"]
    # No DELETE when fallback
    mock_client.delete.assert_not_called()


@pytest.mark.asyncio
async def test_consolidate_with_agent_human_only_messages_falls_back():
    """All-human messages means no summary extracted → fallback is used."""
    agent = AsyncMock()

    human_msg = MagicMock()
    human_msg.content = "Human text."
    human_msg.type = "human"

    agent.ainvoke = AsyncMock(return_value={"messages": [human_msg]})

    loop = ConsolidationLoop(agent=agent)
    mems = _memories(CONSOLIDATION_THRESHOLD)
    mock_client = _make_http_client_mock(get_status=200, get_json=mems)

    with patch("consolidation.httpx.AsyncClient", return_value=mock_client):
        await loop._consolidate()

    mock_client.post.assert_called_once()
    # No deletes in fallback mode
    mock_client.delete.assert_not_called()


@pytest.mark.asyncio
async def test_consolidate_with_agent_empty_content_skipped():
    """Messages with empty/whitespace content are skipped when finding summary."""
    agent = AsyncMock()

    blank_msg = MagicMock()
    blank_msg.content = "   "
    blank_msg.type = "ai"

    good_msg = MagicMock()
    good_msg.content = "Real summary here."
    good_msg.type = "ai"

    # reversed order: blank_msg first, then good_msg
    agent.ainvoke = AsyncMock(return_value={"messages": [good_msg, blank_msg]})

    loop = ConsolidationLoop(agent=agent)
    mems = _memories(CONSOLIDATION_THRESHOLD)
    mock_client = _make_http_client_mock(get_status=200, get_json=mems)

    with patch("consolidation.httpx.AsyncClient", return_value=mock_client):
        await loop._consolidate()

    body = mock_client.post.call_args[1]["json"]
    # blank_msg skipped → good_msg used
    assert "Real summary here." in body["content"]


# ---------------------------------------------------------------------------
# _consolidate() — agent failure (fallback path)
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_consolidate_agent_exception_falls_back(caplog):
    """When agent.ainvoke raises, the concatenation fallback is used."""
    agent = AsyncMock()
    agent.ainvoke = AsyncMock(side_effect=RuntimeError("rate limit"))

    loop = ConsolidationLoop(agent=agent)
    mems = _memories(CONSOLIDATION_THRESHOLD)
    mock_client = _make_http_client_mock(get_status=200, get_json=mems)

    with patch("consolidation.httpx.AsyncClient", return_value=mock_client):
        with caplog.at_level(logging.ERROR, logger="consolidation"):
            await loop._consolidate()

    # Should log the error message
    assert "CONSOLIDATION" in caplog.text
    assert "Falling back to simple concatenation" in caplog.text

    # Should still produce a fallback POST
    mock_client.post.assert_called_once()
    body = mock_client.post.call_args[1]["json"]
    assert "[Consolidated]" in body["content"]
    assert body["scope"] == "TEAM"


@pytest.mark.asyncio
async def test_consolidate_agent_exception_no_deletes(caplog):
    """When agent fails, original memories are NOT deleted (fallback path)."""
    agent = AsyncMock()
    agent.ainvoke = AsyncMock(side_effect=Exception("model error"))

    loop = ConsolidationLoop(agent=agent)
    mems = _memories(CONSOLIDATION_THRESHOLD)
    mock_client = _make_http_client_mock(get_status=200, get_json=mems)

    with patch("consolidation.httpx.AsyncClient", return_value=mock_client):
        with caplog.at_level(logging.ERROR, logger="consolidation"):
            await loop._consolidate()

    mock_client.delete.assert_not_called()


# ---------------------------------------------------------------------------
# Module-level environment variable defaults
# ---------------------------------------------------------------------------

def test_module_constants_defaults(monkeypatch):
    """Module-level constants have correct defaults when env vars are unset."""
    # These are set at import time, so we check their values directly
    assert CONSOLIDATION_INTERVAL == float(
        __import__("os").environ.get("CONSOLIDATION_INTERVAL", "300")
    )
    assert CONSOLIDATION_THRESHOLD == int(
        __import__("os").environ.get("CONSOLIDATION_THRESHOLD", "10")
    )


@pytest.mark.asyncio
async def test_start_while_exits_when_running_false_at_loop_condition():
    """Cover the while-loop exit branch: _running becomes False between iterations
    so the while condition evaluates to False and the loop exits cleanly."""
    loop = ConsolidationLoop()
    sleep_calls = [0]

    async def fake_sleep(secs):
        sleep_calls[0] += 1
        # First sleep: leave _running True so we enter the body (break path)
        # Second sleep: this should not be called; the while exits instead
        if sleep_calls[0] == 1:
            # Don't change _running here; let _consolidate run
            pass

    consolidate_calls = [0]

    async def fake_consolidate():
        consolidate_calls[0] += 1
        # After consolidating, set _running=False so the while condition
        # fails on the NEXT evaluation (covering the 38->exit branch)
        loop._running = False

    with patch("consolidation.asyncio.sleep", side_effect=fake_sleep):
        with patch.object(loop, "_consolidate", side_effect=fake_consolidate):
            await loop.start()

    assert sleep_calls[0] == 1
    assert consolidate_calls[0] == 1


@pytest.mark.asyncio
async def test_consolidation_loop_logs_correct_interval(caplog):
    """Log message in start() references the CONSOLIDATION_INTERVAL value."""
    loop = ConsolidationLoop()

    async def fake_sleep(secs):
        loop._running = False

    with patch("consolidation.asyncio.sleep", side_effect=fake_sleep):
        with patch.object(loop, "_consolidate", new_callable=AsyncMock):
            with caplog.at_level(logging.INFO, logger="consolidation"):
                await loop.start()

    assert str(int(CONSOLIDATION_INTERVAL)) in caplog.text or str(CONSOLIDATION_INTERVAL) in caplog.text
