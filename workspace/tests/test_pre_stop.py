"""Tests for lib.pre_stop — GH#1391 pre-stop serialization."""

import json
import os
import tempfile

import pytest


class _MockHeartbeat:
    """Minimal heartbeat for testing — matches heartbeat.HeartbeatLoop shape."""

    def __init__(self):
        self.current_task = "Implementing feature X"
        self.active_tasks = 1
        self.start_time = 1000.0
        self._session_id = None


class _MockAdapter:
    """Minimal adapter that returns known pre_stop_state for testing."""

    def pre_stop_state(self):
        return {
            "session_id": "sess_abc123xyz",
            "transcript_lines": [
                "User: hello",
                "Agent: Hi! How can I help?",
            ],
        }


def test_build_snapshot_basic():
    """build_snapshot returns workspace_id, timestamp, and heartbeat fields."""
    from lib.pre_stop import build_snapshot

    hb = _MockHeartbeat()
    adapter_state = {"session_id": "sess_abc", "transcript_lines": ["line1"]}
    snap = build_snapshot(hb, adapter_state)

    assert snap["workspace_id"] == os.environ.get("WORKSPACE_ID", "unknown")
    assert "timestamp" in snap
    assert snap["current_task"] == "Implementing feature X"
    assert snap["active_tasks"] == 1
    assert snap["adapter"] == adapter_state


def test_build_snapshot_none_heartbeat():
    """build_snapshot handles None heartbeat gracefully."""
    from lib.pre_stop import build_snapshot

    snap = build_snapshot(None, {"session_id": "sess_xyz"})
    assert snap["current_task"] == ""
    assert snap["active_tasks"] == 0
    # session_id is NOT promoted to top-level when heartbeat is absent;
    # it stays nested inside adapter.
    assert "session_id" not in snap
    assert snap["adapter"]["session_id"] == "sess_xyz"


def test_build_snapshot_scrubbed_secrets():
    """Snapshot content with API keys is scrubbed by write_snapshot."""
    from lib.pre_stop import build_snapshot, write_snapshot

    hb = _MockHeartbeat()
    adapter_state = {
        "session_id": "sess_secret",
        "transcript_lines": [
            "Authorization: Bearer abc123.def456.ghi789",
            "token_used: Bearer xyz.token.placeholder",
        ],
    }
    snap = build_snapshot(hb, adapter_state)

    with tempfile.NamedTemporaryFile(suffix=".json", delete=False) as f:
        path = f.name

    try:
        ok = write_snapshot(snap, path=path)
        assert ok, "write_snapshot should return True on success"

        with open(path) as f:
            loaded = json.load(f)

        lines = loaded["adapter"]["transcript_lines"]
        assert not any("Bearer abc" in l for l in lines), "Bearer token should be scrubbed"
        assert any("REDACTED" in l for l in lines), "Scrub markers should be present"
    finally:
        os.unlink(path)


def test_build_snapshot_scrub_drops_sandbox_content():
    """Sandbox-sourced transcript lines are dropped entirely."""
    from lib.pre_stop import build_snapshot, write_snapshot

    hb = _MockHeartbeat()
    adapter_state = {
        "session_lines": [
            "source=sandbox echo hello",
            "Normal message",
        ],
    }
    snap = build_snapshot(hb, adapter_state)

    with tempfile.NamedTemporaryFile(suffix=".json", delete=False) as f:
        path = f.name

    try:
        write_snapshot(snap, path=path)
        with open(path) as f:
            loaded = json.load(f)
        # scrub_snapshot drops sandbox entries from lists
        lines = loaded["adapter"].get("session_lines", [])
        assert not any("sandbox" in l for l in lines), "Sandbox lines should be dropped"
    finally:
        os.unlink(path)


def test_read_snapshot_missing_returns_none():
    """read_snapshot returns None when the file doesn't exist."""
    from lib.pre_stop import read_snapshot

    result = read_snapshot(path="/nonexistent/path/12345.json")
    assert result is None


def test_read_snapshot_returns_data():
    """read_snapshot returns the parsed JSON when the file exists."""
    from lib.pre_stop import read_snapshot

    data = {"workspace_id": "test-ws", "current_task": "test"}
    with tempfile.NamedTemporaryFile(suffix=".json", delete=False, mode="w") as f:
        json.dump(data, f)
        path = f.name

    try:
        result = read_snapshot(path=path)
        assert result == data
        assert result["workspace_id"] == "test-ws"
    finally:
        os.unlink(path)


def test_delete_snapshot_removes_file():
    """delete_snapshot removes the file and is idempotent on missing file."""
    from lib.pre_stop import delete_snapshot

    with tempfile.NamedTemporaryFile(suffix=".json", delete=False) as f:
        path = f.name

    delete_snapshot(path=path)
    assert not os.path.exists(path), "File should be removed"

    # Idempotent: no error if already absent
    delete_snapshot(path=path)


def test_write_snapshot_returns_false_on_error(monkeypatch):
    """write_snapshot returns False on I/O errors and logs a warning."""
    from lib.pre_stop import build_snapshot, write_snapshot

    hb = _MockHeartbeat()

    # Make the parent dir unreadable to trigger an error.
    # We can't easily make /nonexistent readonly, so we mock open().
    import unittest.mock as mock

    snap = build_snapshot(hb, {})

    with mock.patch("builtins.open", side_effect=OSError("disk full")):
        ok = write_snapshot(snap, path="/tmp/fake.json")
    assert ok is False, "write_snapshot should return False on error"


def test_restore_state_stores_on_adapter():
    """restore_state stores snapshot fields as adapter attributes."""
    from adapter_base import BaseAdapter

    class DummyAdapter(BaseAdapter):
        def name(self): return "dummy"
        def display_name(self): return "Dummy"
        def description(self): return "dummy"
        async def setup(self, cfg): pass
        async def create_executor(self, cfg): pass

    adapter = DummyAdapter()
    snap = {
        "session_id": "sess_restored_123",
        "transcript_lines": ["line1", "line2"],
        "current_task": "Old task",
    }
    adapter.restore_state(snap)

    assert adapter._snapshot_session_id == "sess_restored_123"
    assert adapter._snapshot_transcript == ["line1", "line2"]


def test_pre_stop_state_default_returns_empty():
    """Default pre_stop_state (BaseAdapter) returns an empty dict."""
    from adapter_base import BaseAdapter

    class DummyAdapter(BaseAdapter):
        def name(self): return "dummy"
        def display_name(self): return "Dummy"
        def description(self): return "dummy"
        async def setup(self, cfg): pass
        async def create_executor(self, cfg): pass

    adapter = DummyAdapter()
    state = adapter.pre_stop_state()
    assert state == {}


def test_pre_stop_state_with_executor_session_id():
    """pre_stop_state captures _executor._session_id when available."""
    from adapter_base import BaseAdapter

    class DummyExecutor:
        pass

    class DummyAdapter(BaseAdapter):
        def name(self): return "dummy"
        def display_name(self): return "Dummy"
        def description(self): return "dummy"
        async def setup(self, cfg): pass
        async def create_executor(self, cfg):
            # Simulate storing the executor so pre_stop_state can find it
            self._executor = DummyExecutor()
            self._executor._session_id = "sess_from_executor_456"
            return self._executor

    adapter = DummyAdapter()
    # Simulate executor was already created
    adapter._executor = DummyExecutor()
    adapter._executor._session_id = "sess_from_executor_456"

    state = adapter.pre_stop_state()
    assert state["session_id"] == "sess_from_executor_456"


def test_pre_stop_state_transcript_included():
    """pre_stop_state includes transcript_lines when transcript is supported."""
    from adapter_base import BaseAdapter

    class DummyExecutor:
        pass

    class DummyAdapter(BaseAdapter):
        def name(self): return "dummy"
        def display_name(self): return "Dummy"
        def description(self): return "dummy"
        async def setup(self, cfg): pass
        async def create_executor(self, cfg):
            self._executor = DummyExecutor()
            return self._executor

        def transcript_lines(self, since=0, limit=100):
            return {
                "supported": True,
                "lines": ["User: test", "Agent: response"],
                "cursor": 2,
                "more": False,
            }

    adapter = DummyAdapter()
    adapter._executor = DummyExecutor()
    state = adapter.pre_stop_state()

    assert "transcript_lines" in state
    assert state["transcript_lines"] == ["User: test", "Agent: response"]
