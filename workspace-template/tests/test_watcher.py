"""Tests for watcher.py — ConfigWatcher polling, hashing, and change detection."""

import asyncio
import hashlib
import os
import tempfile
from pathlib import Path
from unittest.mock import AsyncMock, MagicMock, patch, call

import pytest

from watcher import ConfigWatcher, POLL_INTERVAL, DEBOUNCE_SECONDS


# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------

@pytest.fixture
def tmp_config(tmp_path):
    """Return a temporary config directory path (string)."""
    return str(tmp_path)


@pytest.fixture
def watcher(tmp_config):
    """Return a ConfigWatcher pointed at a temporary config directory."""
    return ConfigWatcher(
        config_path=tmp_config,
        platform_url="http://platform:8080",
        workspace_id="ws-test",
    )


# ---------------------------------------------------------------------------
# __init__
# ---------------------------------------------------------------------------

def test_init_stores_attrs(tmp_config):
    """Constructor stores all provided arguments as attributes."""
    cb = MagicMock()
    w = ConfigWatcher(
        config_path=tmp_config,
        platform_url="http://platform:8080",
        workspace_id="ws-42",
        on_reload=cb,
    )
    assert w.config_path == tmp_config
    assert w.platform_url == "http://platform:8080"
    assert w.workspace_id == "ws-42"
    assert w.on_reload is cb
    assert w._file_hashes == {}
    assert w._running is False


def test_init_defaults_on_reload(tmp_config):
    """on_reload defaults to None when not supplied."""
    w = ConfigWatcher(tmp_config, "http://p:8080", "ws-1")
    assert w.on_reload is None


# ---------------------------------------------------------------------------
# _hash_file
# ---------------------------------------------------------------------------

def test_hash_file_real_file(tmp_path, watcher):
    """_hash_file returns md5 hex digest of the file's bytes."""
    f = tmp_path / "sample.txt"
    f.write_bytes(b"hello world")
    expected = hashlib.md5(b"hello world").hexdigest()
    assert watcher._hash_file(str(f)) == expected


def test_hash_file_missing_returns_empty(watcher):
    """_hash_file returns '' for a non-existent file (OSError path)."""
    result = watcher._hash_file("/nonexistent/path/file.txt")
    assert result == ""


def test_hash_file_ioerror(tmp_path, watcher, monkeypatch):
    """_hash_file returns '' when Path.read_bytes raises IOError."""
    f = tmp_path / "bad.txt"
    f.write_bytes(b"data")
    monkeypatch.setattr(Path, "read_bytes", lambda self: (_ for _ in ()).throw(IOError("read error")))
    assert watcher._hash_file(str(f)) == ""


# ---------------------------------------------------------------------------
# _scan_hashes
# ---------------------------------------------------------------------------

def test_scan_hashes_empty_directory(watcher):
    """_scan_hashes returns an empty dict for an empty config dir."""
    result = watcher._scan_hashes()
    assert result == {}


def test_scan_hashes_skips_dotfiles(tmp_path, watcher):
    """_scan_hashes ignores files whose names start with '.'."""
    (tmp_path / ".hidden").write_bytes(b"secret")
    (tmp_path / "visible.yaml").write_bytes(b"data: 1")
    result = watcher._scan_hashes()
    keys = list(result.keys())
    assert all(not k.startswith(".") for k in keys)
    assert any("visible.yaml" in k for k in keys)


def test_scan_hashes_returns_relative_paths(tmp_path, watcher):
    """_scan_hashes keys are relative to config_path, not absolute."""
    (tmp_path / "config.yaml").write_bytes(b"key: value")
    result = watcher._scan_hashes()
    assert "config.yaml" in result


def test_scan_hashes_subdirectory(tmp_path, watcher):
    """_scan_hashes recurses into subdirectories."""
    sub = tmp_path / "subdir"
    sub.mkdir()
    (sub / "nested.json").write_bytes(b"{}")
    result = watcher._scan_hashes()
    # relative path should be like "subdir/nested.json" or "subdir\\nested.json"
    assert any("nested.json" in k for k in result.keys())


def test_scan_hashes_multiple_files(tmp_path, watcher):
    """_scan_hashes captures all non-hidden files in the directory."""
    for name in ("a.yaml", "b.yaml", "c.json"):
        (tmp_path / name).write_bytes(name.encode())
    result = watcher._scan_hashes()
    assert len(result) == 3


# ---------------------------------------------------------------------------
# _detect_changes
# ---------------------------------------------------------------------------

def test_detect_changes_no_changes(tmp_path, watcher):
    """_detect_changes returns an empty list when nothing changed."""
    (tmp_path / "file.yaml").write_bytes(b"content")
    # Seed the hashes
    watcher._file_hashes = watcher._scan_hashes()
    changed = watcher._detect_changes()
    assert changed == []


def test_detect_changes_new_file(tmp_path, watcher):
    """_detect_changes reports a file that appeared since last scan."""
    watcher._file_hashes = {}
    (tmp_path / "new.yaml").write_bytes(b"new")
    changed = watcher._detect_changes()
    assert any("new.yaml" in p for p in changed)


def test_detect_changes_modified_file(tmp_path, watcher):
    """_detect_changes reports a file whose content has changed."""
    f = tmp_path / "mod.yaml"
    f.write_bytes(b"original")
    watcher._file_hashes = watcher._scan_hashes()
    # Modify the file
    f.write_bytes(b"modified")
    changed = watcher._detect_changes()
    assert any("mod.yaml" in p for p in changed)


def test_detect_changes_deleted_file(tmp_path, watcher):
    """_detect_changes reports a file that was deleted since last scan."""
    f = tmp_path / "gone.yaml"
    f.write_bytes(b"was here")
    watcher._file_hashes = watcher._scan_hashes()
    # Delete the file
    f.unlink()
    changed = watcher._detect_changes()
    assert any("gone.yaml" in p for p in changed)


def test_detect_changes_updates_cached_hashes(tmp_path, watcher):
    """After _detect_changes, _file_hashes reflects the current state."""
    f = tmp_path / "track.yaml"
    f.write_bytes(b"v1")
    watcher._file_hashes = {}
    watcher._detect_changes()
    assert any("track.yaml" in k for k in watcher._file_hashes)


# ---------------------------------------------------------------------------
# _notify_platform
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_notify_platform_success(watcher):
    """_notify_platform POSTs the agent card to the correct URL."""
    mock_response = MagicMock()
    mock_client = AsyncMock()
    mock_client.post = AsyncMock(return_value=mock_response)
    mock_client.__aenter__ = AsyncMock(return_value=mock_client)
    mock_client.__aexit__ = AsyncMock(return_value=False)

    with patch("watcher.httpx.AsyncClient", return_value=mock_client):
        await watcher._notify_platform({"name": "MyAgent"})

    mock_client.post.assert_called_once()
    call_args = mock_client.post.call_args
    assert call_args[0][0] == "http://platform:8080/registry/update-card"
    payload = call_args[1]["json"]
    assert payload["workspace_id"] == "ws-test"
    assert payload["agent_card"] == {"name": "MyAgent"}


@pytest.mark.asyncio
async def test_notify_platform_failure_logs_warning(watcher, caplog):
    """_notify_platform logs a warning when the POST raises an exception."""
    mock_client = AsyncMock()
    mock_client.post = AsyncMock(side_effect=Exception("connection refused"))
    mock_client.__aenter__ = AsyncMock(return_value=mock_client)
    mock_client.__aexit__ = AsyncMock(return_value=False)

    import logging
    with patch("watcher.httpx.AsyncClient", return_value=mock_client):
        with caplog.at_level(logging.WARNING, logger="watcher"):
            await watcher._notify_platform({})

    assert "Failed to update Agent Card" in caplog.text


# ---------------------------------------------------------------------------
# start() / stop()
# ---------------------------------------------------------------------------

@pytest.mark.asyncio
async def test_stop_sets_running_false(watcher):
    """stop() sets _running to False."""
    watcher._running = True
    watcher.stop()
    assert watcher._running is False


@pytest.mark.asyncio
async def test_start_sets_running_true_and_seeds_hashes(tmp_path, watcher):
    """start() sets _running=True and seeds _file_hashes before looping."""
    (tmp_path / "seed.yaml").write_bytes(b"data")

    sleep_calls = []

    async def fake_sleep(secs):
        sleep_calls.append(secs)
        # Stop after the first POLL_INTERVAL sleep
        watcher._running = False

    with patch("watcher.asyncio.sleep", side_effect=fake_sleep):
        await watcher.start()

    assert any("seed.yaml" in k for k in watcher._file_hashes)
    # First sleep should be POLL_INTERVAL
    assert sleep_calls[0] == POLL_INTERVAL


@pytest.mark.asyncio
async def test_start_no_changes_continues_loop(tmp_path, watcher):
    """When no files change, the loop iterates without calling debounce sleep."""
    (tmp_path / "stable.yaml").write_bytes(b"stable")

    iteration = [0]

    async def fake_sleep(secs):
        iteration[0] += 1
        if iteration[0] >= 2:
            watcher._running = False

    with patch("watcher.asyncio.sleep", side_effect=fake_sleep):
        await watcher.start()

    # Should have slept twice (both POLL_INTERVAL), no DEBOUNCE sleep
    assert iteration[0] == 2


@pytest.mark.asyncio
async def test_start_detects_change_and_debounces(tmp_path, watcher):
    """When changes are detected, start() sleeps for DEBOUNCE_SECONDS too."""
    (tmp_path / "change.yaml").write_bytes(b"v1")

    sleep_calls = []
    call_count = [0]

    async def fake_sleep(secs):
        sleep_calls.append(secs)
        call_count[0] += 1
        if call_count[0] == 1:
            # After POLL_INTERVAL sleep, modify the file to trigger a change
            (tmp_path / "change.yaml").write_bytes(b"v2")
        elif call_count[0] >= 2:
            # After DEBOUNCE sleep, stop
            watcher._running = False

    with patch("watcher.asyncio.sleep", side_effect=fake_sleep):
        await watcher.start()

    assert POLL_INTERVAL in sleep_calls
    assert DEBOUNCE_SECONDS in sleep_calls


@pytest.mark.asyncio
async def test_start_calls_on_reload_callback(tmp_path):
    """start() invokes on_reload callback when changes are detected."""
    reload_called = []

    async def on_reload():
        reload_called.append(True)

    w = ConfigWatcher(
        config_path=str(tmp_path),
        platform_url="http://p:8080",
        workspace_id="ws-1",
        on_reload=on_reload,
    )

    (tmp_path / "watched.yaml").write_bytes(b"initial")

    call_count = [0]

    async def fake_sleep(secs):
        call_count[0] += 1
        if call_count[0] == 1:
            # Trigger a change on first POLL_INTERVAL sleep
            (tmp_path / "watched.yaml").write_bytes(b"changed")
        elif call_count[0] >= 2:
            w._running = False

    with patch("watcher.asyncio.sleep", side_effect=fake_sleep):
        await w.start()

    assert reload_called, "on_reload should have been called"


@pytest.mark.asyncio
async def test_start_on_reload_exception_logged(tmp_path, caplog):
    """start() logs an error when on_reload callback raises an exception."""
    import logging

    async def bad_reload():
        raise RuntimeError("reload exploded")

    w = ConfigWatcher(
        config_path=str(tmp_path),
        platform_url="http://p:8080",
        workspace_id="ws-1",
        on_reload=bad_reload,
    )

    (tmp_path / "trigger.yaml").write_bytes(b"before")

    call_count = [0]

    async def fake_sleep(secs):
        call_count[0] += 1
        if call_count[0] == 1:
            (tmp_path / "trigger.yaml").write_bytes(b"after")
        elif call_count[0] >= 2:
            w._running = False

    with patch("watcher.asyncio.sleep", side_effect=fake_sleep):
        with caplog.at_level(logging.ERROR, logger="watcher"):
            await w.start()

    assert "Reload callback failed" in caplog.text


@pytest.mark.asyncio
async def test_start_without_on_reload_no_error(tmp_path):
    """start() handles changes gracefully even when on_reload is None."""
    w = ConfigWatcher(
        config_path=str(tmp_path),
        platform_url="http://p:8080",
        workspace_id="ws-1",
        on_reload=None,
    )

    (tmp_path / "file.yaml").write_bytes(b"v1")

    call_count = [0]

    async def fake_sleep(secs):
        call_count[0] += 1
        if call_count[0] == 1:
            (tmp_path / "file.yaml").write_bytes(b"v2")
        elif call_count[0] >= 2:
            w._running = False

    with patch("watcher.asyncio.sleep", side_effect=fake_sleep):
        await w.start()  # Should not raise


@pytest.mark.asyncio
async def test_start_logs_on_startup(tmp_path, caplog):
    """start() logs an info message naming the config_path."""
    import logging

    w = ConfigWatcher(str(tmp_path), "http://p:8080", "ws-1")

    async def fake_sleep(secs):
        w._running = False

    with patch("watcher.asyncio.sleep", side_effect=fake_sleep):
        with caplog.at_level(logging.INFO, logger="watcher"):
            await w.start()

    assert "Config watcher started" in caplog.text
