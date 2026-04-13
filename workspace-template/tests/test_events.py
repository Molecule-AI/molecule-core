"""Tests for events.py — PlatformEventSubscriber WebSocket handling."""

import asyncio
import json
import logging
import sys
from types import ModuleType
from unittest.mock import AsyncMock, MagicMock, patch, call

import pytest

from events import PlatformEventSubscriber, REBUILD_EVENTS


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _make_ws_mock(messages):
    """Return an async-context-manager mock that yields messages one-by-one.

    `messages` is a list of raw strings (or exceptions to raise).
    """
    ws = MagicMock()

    async def _aiter():
        for item in messages:
            if isinstance(item, BaseException):
                raise item
            yield item

    ws.__aiter__ = lambda self: _aiter()
    ws.__aenter__ = AsyncMock(return_value=ws)
    ws.__aexit__ = AsyncMock(return_value=False)
    return ws


# ---------------------------------------------------------------------------
# __init__ — URL conversion
# ---------------------------------------------------------------------------

def test_init_http_to_ws():
    """http:// platform URLs are converted to ws://."""
    sub = PlatformEventSubscriber("http://platform:8080", "ws-1")
    assert sub.ws_url == "ws://platform:8080/ws"


def test_init_https_to_wss():
    """https:// platform URLs are converted to wss://."""
    sub = PlatformEventSubscriber("https://platform:8080", "ws-1")
    assert sub.ws_url == "wss://platform:8080/ws"


def test_init_stores_attrs():
    """Constructor stores workspace_id, on_peer_change, initial state."""
    cb = MagicMock()
    sub = PlatformEventSubscriber("http://p:8080", "ws-42", on_peer_change=cb)
    assert sub.workspace_id == "ws-42"
    assert sub.on_peer_change is cb
    assert sub._running is False
    assert sub._reconnect_delay == 1.0


def test_init_on_peer_change_defaults_none():
    """on_peer_change defaults to None when not supplied."""
    sub = PlatformEventSubscriber("http://p:8080", "ws-1")
    assert sub.on_peer_change is None


# ---------------------------------------------------------------------------
# stop()
# ---------------------------------------------------------------------------

def test_stop_sets_running_false():
    """stop() sets _running to False."""
    sub = PlatformEventSubscriber("http://p:8080", "ws-1")
    sub._running = True
    sub.stop()
    assert sub._running is False


# ---------------------------------------------------------------------------
# _connect() — websockets ImportError path
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_connect_no_websockets_package(monkeypatch):
    """_connect() disables running and returns when websockets is not installed."""
    sub = PlatformEventSubscriber("http://p:8080", "ws-1")
    sub._running = True

    # Hide websockets from sys.modules
    original = sys.modules.pop("websockets", None)
    # Also prevent import by making it raise ImportError via builtins
    import builtins
    real_import = builtins.__import__

    def _no_websockets(name, *args, **kwargs):
        if name == "websockets":
            raise ImportError("No module named 'websockets'")
        return real_import(name, *args, **kwargs)

    monkeypatch.setattr(builtins, "__import__", _no_websockets)
    try:
        await sub._connect()
    finally:
        if original is not None:
            sys.modules["websockets"] = original
        monkeypatch.setattr(builtins, "__import__", real_import)

    assert sub._running is False


# ---------------------------------------------------------------------------
# _connect() — message processing
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_connect_rebuild_event_calls_on_peer_change():
    """REBUILD_EVENTS trigger the on_peer_change callback."""
    peer_events = []

    async def on_peer_change(event):
        peer_events.append(event)

    sub = PlatformEventSubscriber("http://p:8080", "ws-1", on_peer_change=on_peer_change)
    sub._running = True

    event_msg = json.dumps({"event": "WORKSPACE_ONLINE", "workspace_id": "ws-2"})
    ws_mock = _make_ws_mock([event_msg])

    websockets_mod = MagicMock()
    websockets_mod.connect = MagicMock(return_value=ws_mock)

    with patch.dict(sys.modules, {"websockets": websockets_mod}):
        await sub._connect()

    assert len(peer_events) == 1
    assert peer_events[0]["event"] == "WORKSPACE_ONLINE"


@pytest.mark.asyncio
async def test_connect_all_rebuild_event_types():
    """Every event type in REBUILD_EVENTS triggers on_peer_change."""
    for event_type in REBUILD_EVENTS:
        received = []

        async def on_peer_change(event, _et=event_type):
            received.append(event)

        sub = PlatformEventSubscriber("http://p:8080", "ws-1", on_peer_change=on_peer_change)
        sub._running = True

        msg = json.dumps({"event": event_type, "workspace_id": "ws-x"})
        ws_mock = _make_ws_mock([msg])

        websockets_mod = MagicMock()
        websockets_mod.connect = MagicMock(return_value=ws_mock)

        with patch.dict(sys.modules, {"websockets": websockets_mod}):
            await sub._connect()

        assert len(received) == 1, f"Expected callback for {event_type}"


@pytest.mark.asyncio
async def test_connect_ignored_event_no_callback():
    """Events not in REBUILD_EVENTS do not invoke on_peer_change."""
    called = []

    async def on_peer_change(event):
        called.append(event)

    sub = PlatformEventSubscriber("http://p:8080", "ws-1", on_peer_change=on_peer_change)
    sub._running = True

    msg = json.dumps({"event": "HEARTBEAT", "workspace_id": "ws-2"})
    ws_mock = _make_ws_mock([msg])

    websockets_mod = MagicMock()
    websockets_mod.connect = MagicMock(return_value=ws_mock)

    with patch.dict(sys.modules, {"websockets": websockets_mod}):
        await sub._connect()

    assert called == []


@pytest.mark.asyncio
async def test_connect_no_on_peer_change_rebuild_event():
    """REBUILD_EVENTS are handled without error when on_peer_change is None."""
    sub = PlatformEventSubscriber("http://p:8080", "ws-1", on_peer_change=None)
    sub._running = True

    msg = json.dumps({"event": "WORKSPACE_ONLINE", "workspace_id": "ws-3"})
    ws_mock = _make_ws_mock([msg])

    websockets_mod = MagicMock()
    websockets_mod.connect = MagicMock(return_value=ws_mock)

    with patch.dict(sys.modules, {"websockets": websockets_mod}):
        await sub._connect()  # Should not raise


@pytest.mark.asyncio
async def test_connect_json_decode_error_continues():
    """Malformed JSON messages are silently skipped (no crash, no callback)."""
    called = []

    async def on_peer_change(event):
        called.append(event)

    sub = PlatformEventSubscriber("http://p:8080", "ws-1", on_peer_change=on_peer_change)
    sub._running = True

    # Mix bad JSON with a valid message
    good_msg = json.dumps({"event": "WORKSPACE_ONLINE", "workspace_id": "ws-4"})
    ws_mock = _make_ws_mock(["not-valid-json{{{", good_msg])

    websockets_mod = MagicMock()
    websockets_mod.connect = MagicMock(return_value=ws_mock)

    with patch.dict(sys.modules, {"websockets": websockets_mod}):
        await sub._connect()

    # The good message after the bad one should still fire the callback
    assert len(called) == 1


@pytest.mark.asyncio
async def test_connect_processing_exception_logged(caplog):
    """Exceptions during event processing are logged as warnings and skipped."""
    async def bad_callback(event):
        raise RuntimeError("callback blew up")

    sub = PlatformEventSubscriber("http://p:8080", "ws-1", on_peer_change=bad_callback)
    sub._running = True

    msg = json.dumps({"event": "WORKSPACE_ONLINE", "workspace_id": "ws-5"})
    ws_mock = _make_ws_mock([msg])

    websockets_mod = MagicMock()
    websockets_mod.connect = MagicMock(return_value=ws_mock)

    with patch.dict(sys.modules, {"websockets": websockets_mod}):
        with caplog.at_level(logging.WARNING, logger="events"):
            await sub._connect()

    assert "Error processing event" in caplog.text


@pytest.mark.asyncio
async def test_connect_resets_reconnect_delay():
    """A successful connection resets _reconnect_delay to 1.0."""
    sub = PlatformEventSubscriber("http://p:8080", "ws-1")
    sub._running = True
    sub._reconnect_delay = 16.0  # Simulate previous backoff

    ws_mock = _make_ws_mock([])  # No messages; connects and exits cleanly

    websockets_mod = MagicMock()
    websockets_mod.connect = MagicMock(return_value=ws_mock)

    with patch.dict(sys.modules, {"websockets": websockets_mod}):
        await sub._connect()

    assert sub._reconnect_delay == 1.0


@pytest.mark.asyncio
async def test_connect_uses_workspace_id_header():
    """_connect() passes X-Workspace-ID header to websockets.connect."""
    sub = PlatformEventSubscriber("http://p:8080", "ws-hdr", on_peer_change=None)
    sub._running = True

    ws_mock = _make_ws_mock([])

    websockets_mod = MagicMock()
    websockets_mod.connect = MagicMock(return_value=ws_mock)

    with patch.dict(sys.modules, {"websockets": websockets_mod}):
        await sub._connect()

    call_kwargs = websockets_mod.connect.call_args[1]
    assert call_kwargs.get("additional_headers") == {"X-Workspace-ID": "ws-hdr"}


# ---------------------------------------------------------------------------
# start() — reconnect with backoff
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_start_sets_running_true():
    """start() sets _running=True before entering the loop."""
    sub = PlatformEventSubscriber("http://p:8080", "ws-1")

    connect_calls = [0]

    async def fake_connect():
        connect_calls[0] += 1
        sub._running = False  # Stop after first connect

    sub._connect = fake_connect
    await sub.start()

    assert connect_calls[0] == 1


@pytest.mark.asyncio
async def test_start_reconnects_on_exception():
    """start() reconnects after a connection exception with backoff sleep."""
    sub = PlatformEventSubscriber("http://p:8080", "ws-1")

    connect_calls = [0]
    sleep_calls = []

    async def fake_connect():
        connect_calls[0] += 1
        if connect_calls[0] == 1:
            raise ConnectionError("refused")
        sub._running = False

    async def fake_sleep(secs):
        sleep_calls.append(secs)

    sub._connect = fake_connect

    with patch("events.asyncio.sleep", side_effect=fake_sleep):
        await sub.start()

    assert connect_calls[0] == 2
    assert sleep_calls == [1.0]  # initial _reconnect_delay


@pytest.mark.asyncio
async def test_start_backoff_doubles_each_reconnect():
    """Reconnect delay doubles on each consecutive failure, capped at 30s."""
    sub = PlatformEventSubscriber("http://p:8080", "ws-1")

    connect_calls = [0]
    sleep_calls = []

    async def fake_connect():
        connect_calls[0] += 1
        if connect_calls[0] < 4:
            raise ConnectionError("fail")
        sub._running = False

    async def fake_sleep(secs):
        sleep_calls.append(secs)

    sub._connect = fake_connect

    with patch("events.asyncio.sleep", side_effect=fake_sleep):
        await sub.start()

    # Delays: 1.0, 2.0, 4.0
    assert sleep_calls == [1.0, 2.0, 4.0]


@pytest.mark.asyncio
async def test_start_backoff_capped_at_30():
    """Reconnect delay is capped at 30 seconds."""
    sub = PlatformEventSubscriber("http://p:8080", "ws-1")
    sub._reconnect_delay = 20.0  # Already near the cap

    connect_calls = [0]
    sleep_calls = []

    async def fake_connect():
        connect_calls[0] += 1
        if connect_calls[0] < 3:
            raise ConnectionError("fail")
        sub._running = False

    async def fake_sleep(secs):
        sleep_calls.append(secs)

    sub._connect = fake_connect

    with patch("events.asyncio.sleep", side_effect=fake_sleep):
        await sub.start()

    # 20.0 then min(40.0, 30.0)=30.0
    assert sleep_calls == [20.0, 30.0]


@pytest.mark.asyncio
async def test_start_stops_when_running_false_after_exception():
    """If stop() is called while reconnecting, the loop exits cleanly."""
    sub = PlatformEventSubscriber("http://p:8080", "ws-1")

    connect_calls = [0]

    async def fake_connect():
        connect_calls[0] += 1
        # Mark stopped before raising so the 'if not self._running: break' fires
        sub._running = False
        raise ConnectionError("closed")

    async def fake_sleep(secs):
        pass  # Should not be reached

    sub._connect = fake_connect

    with patch("events.asyncio.sleep", side_effect=fake_sleep):
        await sub.start()

    # Connected once, then saw _running=False and broke out
    assert connect_calls[0] == 1


@pytest.mark.asyncio
async def test_start_logs_reconnect_warning(caplog):
    """start() logs a warning message when a reconnect is needed."""
    sub = PlatformEventSubscriber("http://p:8080", "ws-1")

    connect_calls = [0]

    async def fake_connect():
        connect_calls[0] += 1
        if connect_calls[0] == 1:
            raise ConnectionError("timed out")
        sub._running = False

    async def fake_sleep(secs):
        pass

    sub._connect = fake_connect

    with patch("events.asyncio.sleep", side_effect=fake_sleep):
        with caplog.at_level(logging.WARNING, logger="events"):
            await sub.start()

    assert "WebSocket disconnected" in caplog.text
    assert "Reconnecting" in caplog.text
