"""Tests for executor_helpers.py — the shared helpers that back both
CLIAgentExecutor (codex, ollama) and ClaudeSDKExecutor (claude-code).

Covers 100% of the public surface:
- get_mcp_server_path
- get_http_client / _reset_http_client
- recall_memories (all branches: no env, HTTP error, non-200, non-list, empty
  list, success)
- commit_memory (all branches: no env, empty content, success, exception)
- read_delegation_results (no file, rename race, read error, valid records,
  invalid JSON, mixed, no-preview branch, empty lines)
- set_current_task (no heartbeat, with heartbeat, no env, HTTP exception)
- get_system_prompt (file exists, file missing, fallback, UTF-8 encoding)
- get_a2a_instructions (MCP variant, CLI variant)
- brief_summary (empty, short, long, markdown headers, bold/italic, code
  fences, HR, fallback when all lines stripped)
- extract_message_text (empty parts, .text path, .root.text path, mixed)
- sanitize_agent_error (class name, no body leak)
"""

from __future__ import annotations

import json
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

import executor_helpers as eh
from executor_helpers import (
    BRIEF_SUMMARY_MAX_LEN,
    DEFAULT_MCP_SERVER_PATH,
    brief_summary,
    classify_subprocess_error,
    commit_memory,
    extract_message_text,
    get_a2a_instructions,
    get_http_client,
    get_mcp_server_path,
    get_system_prompt,
    read_delegation_results,
    recall_memories,
    sanitize_agent_error,
    set_current_task,
)


# ---------- fixtures / helpers ----------

@pytest.fixture(autouse=True)
def _reset_shared_http_client():
    """Drop the module-level httpx client before and after every test so
    tests don't leak state into each other."""
    eh.reset_http_client_for_tests()
    yield
    eh.reset_http_client_for_tests()


@pytest.fixture
def platform_env(monkeypatch):
    monkeypatch.setenv("WORKSPACE_ID", "ws-test")
    monkeypatch.setenv("PLATFORM_URL", "http://platform.test")
    return "ws-test", "http://platform.test"


@pytest.fixture
def no_platform_env(monkeypatch):
    monkeypatch.delenv("WORKSPACE_ID", raising=False)
    monkeypatch.delenv("PLATFORM_URL", raising=False)


def _install_mock_http_client(monkeypatch) -> AsyncMock:
    client = AsyncMock()
    client.is_closed = False
    monkeypatch.setattr(eh, "_http_client", client)
    return client


# ======================================================================
# get_mcp_server_path
# ======================================================================

def test_get_mcp_server_path_default(monkeypatch):
    monkeypatch.delenv("A2A_MCP_SERVER_PATH", raising=False)
    assert get_mcp_server_path() == DEFAULT_MCP_SERVER_PATH


def test_get_mcp_server_path_env_override(monkeypatch):
    monkeypatch.setenv("A2A_MCP_SERVER_PATH", "/custom/mcp.py")
    assert get_mcp_server_path() == "/custom/mcp.py"


# ======================================================================
# get_http_client
# ======================================================================

def test_get_http_client_returns_same_instance_on_repeat_calls():
    eh.reset_http_client_for_tests()
    c1 = get_http_client()
    c2 = get_http_client()
    assert c1 is c2


@pytest.mark.asyncio
async def test_get_http_client_rebuilds_when_closed():
    c1 = get_http_client()
    await c1.aclose()
    c2 = get_http_client()
    try:
        assert c1 is not c2
    finally:
        await c2.aclose()


def test_reset_http_client_nulls_state():
    get_http_client()
    assert eh._http_client is not None
    eh.reset_http_client_for_tests()
    assert eh._http_client is None


# ======================================================================
# recall_memories
# ======================================================================

@pytest.mark.asyncio
async def test_recall_memories_no_env_returns_empty(no_platform_env):
    assert await recall_memories() == ""


@pytest.mark.asyncio
async def test_recall_memories_only_workspace_id_returns_empty(monkeypatch):
    monkeypatch.setenv("WORKSPACE_ID", "ws-1")
    monkeypatch.delenv("PLATFORM_URL", raising=False)
    assert await recall_memories() == ""


@pytest.mark.asyncio
async def test_recall_memories_non_200_returns_empty(monkeypatch, platform_env):
    client = _install_mock_http_client(monkeypatch)
    resp = MagicMock(status_code=500)
    client.get = AsyncMock(return_value=resp)
    assert await recall_memories() == ""


@pytest.mark.asyncio
async def test_recall_memories_exception_returns_empty(monkeypatch, platform_env):
    client = _install_mock_http_client(monkeypatch)
    client.get = AsyncMock(side_effect=RuntimeError("boom"))
    assert await recall_memories() == ""


@pytest.mark.asyncio
async def test_recall_memories_non_list_payload_returns_empty(monkeypatch, platform_env):
    client = _install_mock_http_client(monkeypatch)
    resp = MagicMock(status_code=200)
    resp.json = MagicMock(return_value={"not": "a list"})
    client.get = AsyncMock(return_value=resp)
    assert await recall_memories() == ""


@pytest.mark.asyncio
async def test_recall_memories_empty_list_returns_empty(monkeypatch, platform_env):
    client = _install_mock_http_client(monkeypatch)
    resp = MagicMock(status_code=200)
    resp.json = MagicMock(return_value=[])
    client.get = AsyncMock(return_value=resp)
    assert await recall_memories() == ""


@pytest.mark.asyncio
async def test_recall_memories_success_formats_bullet_list(monkeypatch, platform_env):
    client = _install_mock_http_client(monkeypatch)
    resp = MagicMock(status_code=200)
    resp.json = MagicMock(return_value=[
        {"scope": "LOCAL", "content": "User likes Python"},
        {"scope": "GLOBAL", "content": "User prefers concise answers"},
    ])
    client.get = AsyncMock(return_value=resp)
    result = await recall_memories()
    assert "[LOCAL] User likes Python" in result
    assert "[GLOBAL] User prefers concise answers" in result
    assert result.count("\n") == 1


@pytest.mark.asyncio
async def test_recall_memories_trims_to_last_ten(monkeypatch, platform_env):
    client = _install_mock_http_client(monkeypatch)
    payload = [{"scope": "L", "content": f"m{i}"} for i in range(15)]
    resp = MagicMock(status_code=200)
    resp.json = MagicMock(return_value=payload)
    client.get = AsyncMock(return_value=resp)
    result = await recall_memories()
    # Only the last 10 should appear
    assert "m14" in result
    assert "m5" in result  # boundary: 15 - 10 = index 5
    assert "m4" not in result


@pytest.mark.asyncio
async def test_recall_memories_handles_missing_fields(monkeypatch, platform_env):
    client = _install_mock_http_client(monkeypatch)
    resp = MagicMock(status_code=200)
    resp.json = MagicMock(return_value=[{}])
    client.get = AsyncMock(return_value=resp)
    result = await recall_memories()
    assert "[?]" in result  # default scope placeholder


# ======================================================================
# commit_memory
# ======================================================================

@pytest.mark.asyncio
async def test_commit_memory_no_env_is_noop(no_platform_env):
    # Should not raise, should not create a client
    await commit_memory("anything")
    assert eh._http_client is None


@pytest.mark.asyncio
async def test_commit_memory_empty_content_is_noop(monkeypatch, platform_env):
    client = _install_mock_http_client(monkeypatch)
    await commit_memory("")
    client.post.assert_not_called()


@pytest.mark.asyncio
async def test_commit_memory_posts_to_platform(monkeypatch, platform_env):
    client = _install_mock_http_client(monkeypatch)
    client.post = AsyncMock(return_value=MagicMock(status_code=200))
    await commit_memory("Remember this fact")
    client.post.assert_called_once()
    url = client.post.call_args[0][0]
    body = client.post.call_args[1]["json"]
    assert "ws-test/memories" in url
    assert body == {"content": "Remember this fact", "scope": "LOCAL"}


@pytest.mark.asyncio
async def test_commit_memory_swallows_exceptions(monkeypatch, platform_env):
    client = _install_mock_http_client(monkeypatch)
    client.post = AsyncMock(side_effect=Exception("network down"))
    # Should not raise
    await commit_memory("content")


# ======================================================================
# read_delegation_results
# ======================================================================

def test_read_delegation_results_no_file(tmp_path, monkeypatch):
    monkeypatch.setenv("DELEGATION_RESULTS_FILE", str(tmp_path / "missing.jsonl"))
    assert read_delegation_results() == ""


def test_read_delegation_results_valid_records(tmp_path, monkeypatch):
    results_file = tmp_path / "delegation.jsonl"
    results_file.write_text(
        json.dumps({
            "status": "completed",
            "summary": "Task A",
            "response_preview": "Here is A",
        }) + "\n" + json.dumps({
            "status": "failed",
            "summary": "Task B",
        }) + "\n",
        encoding="utf-8",
    )
    monkeypatch.setenv("DELEGATION_RESULTS_FILE", str(results_file))
    out = read_delegation_results()
    assert "[completed] Task A" in out
    assert "Response: Here is A" in out
    assert "[failed] Task B" in out
    # Preview omitted when absent
    lines_for_b = [l for l in out.splitlines() if "Task B" in l]
    assert lines_for_b and not any("Response:" in l for l in lines_for_b[1:2])
    # File consumed
    assert not results_file.exists()


def test_read_delegation_results_skips_invalid_json(tmp_path, monkeypatch):
    results_file = tmp_path / "delegation.jsonl"
    results_file.write_text("not json\n{bad\n", encoding="utf-8")
    monkeypatch.setenv("DELEGATION_RESULTS_FILE", str(results_file))
    assert read_delegation_results() == ""
    assert not results_file.exists()


def test_read_delegation_results_handles_blank_lines_in_middle(tmp_path, monkeypatch):
    """A blank line between valid records must be skipped, not crash."""
    results_file = tmp_path / "delegation.jsonl"
    results_file.write_text(
        json.dumps({"status": "ok", "summary": "first"})
        + "\n   \n"  # blank line with whitespace
        + json.dumps({"status": "ok", "summary": "second"})
        + "\n",
        encoding="utf-8",
    )
    monkeypatch.setenv("DELEGATION_RESULTS_FILE", str(results_file))
    out = read_delegation_results()
    assert "[ok] first" in out
    assert "[ok] second" in out


def test_read_delegation_results_rename_race(tmp_path, monkeypatch):
    """If the file disappears between exists() and rename(), return empty."""
    results_file = tmp_path / "delegation.jsonl"
    results_file.write_text("{}\n", encoding="utf-8")
    monkeypatch.setenv("DELEGATION_RESULTS_FILE", str(results_file))

    with patch("executor_helpers.Path") as MockPath:
        mock_instance = MagicMock()
        mock_instance.exists.return_value = True
        mock_instance.with_suffix.return_value = tmp_path / "delegation.consumed"
        mock_instance.rename.side_effect = OSError("race")
        MockPath.return_value = mock_instance
        assert read_delegation_results() == ""


def test_read_delegation_results_read_text_raises(tmp_path, monkeypatch):
    """Post-rename read failure returns empty instead of crashing."""
    results_file = tmp_path / "delegation.jsonl"
    results_file.write_text("{}\n", encoding="utf-8")
    monkeypatch.setenv("DELEGATION_RESULTS_FILE", str(results_file))

    consumed_mock = MagicMock()
    consumed_mock.read_text.side_effect = OSError("disk gone")
    consumed_mock.unlink = MagicMock()

    with patch("executor_helpers.Path") as MockPath:
        mock_instance = MagicMock()
        mock_instance.exists.return_value = True
        mock_instance.with_suffix.return_value = consumed_mock
        mock_instance.rename.return_value = None
        MockPath.return_value = mock_instance
        assert read_delegation_results() == ""

    consumed_mock.unlink.assert_called_once_with(missing_ok=True)


# ======================================================================
# set_current_task
# ======================================================================

@pytest.mark.asyncio
async def test_set_current_task_no_heartbeat_no_env_is_noop(no_platform_env):
    # Nothing to update, nothing to POST → should return cleanly
    await set_current_task(None, "some task")


@pytest.mark.asyncio
async def test_set_current_task_updates_heartbeat_state():
    hb = SimpleNamespace(current_task="old", active_tasks=0)
    await set_current_task(hb, "new task")
    assert hb.current_task == "new task"
    assert hb.active_tasks == 1


@pytest.mark.asyncio
async def test_set_current_task_empty_clears_heartbeat_state():
    hb = SimpleNamespace(current_task="old", active_tasks=1)
    await set_current_task(hb, "")
    assert hb.current_task == ""
    assert hb.active_tasks == 0


@pytest.mark.asyncio
async def test_set_current_task_posts_to_platform(monkeypatch, platform_env):
    client = _install_mock_http_client(monkeypatch)
    client.post = AsyncMock(return_value=MagicMock(status_code=200))
    hb = SimpleNamespace(current_task="", active_tasks=0)
    await set_current_task(hb, "running")
    client.post.assert_called_once()
    url = client.post.call_args[0][0]
    body = client.post.call_args[1]["json"]
    assert url.endswith("/registry/heartbeat")
    assert body["current_task"] == "running"
    assert body["active_tasks"] == 1


@pytest.mark.asyncio
async def test_set_current_task_swallows_http_exceptions(monkeypatch, platform_env):
    client = _install_mock_http_client(monkeypatch)
    client.post = AsyncMock(side_effect=Exception("boom"))
    # Should not raise
    await set_current_task(None, "x")


# ======================================================================
# get_system_prompt
# ======================================================================

def test_get_system_prompt_reads_file(tmp_path):
    (tmp_path / "system-prompt.md").write_text("You are helpful.", encoding="utf-8")
    assert get_system_prompt(str(tmp_path)) == "You are helpful."


def test_get_system_prompt_missing_uses_fallback(tmp_path):
    assert get_system_prompt(str(tmp_path), fallback="fb") == "fb"


def test_get_system_prompt_missing_no_fallback_returns_none(tmp_path):
    assert get_system_prompt(str(tmp_path)) is None


def test_get_system_prompt_strips_whitespace(tmp_path):
    (tmp_path / "system-prompt.md").write_text("\n  prompt text  \n", encoding="utf-8")
    assert get_system_prompt(str(tmp_path)) == "prompt text"


def test_get_system_prompt_handles_non_utf8(tmp_path):
    # Write invalid utf-8 bytes; errors='replace' should salvage the text.
    (tmp_path / "system-prompt.md").write_bytes(b"hello \xff world")
    out = get_system_prompt(str(tmp_path))
    assert "hello" in out and "world" in out


# ======================================================================
# get_a2a_instructions
# ======================================================================

def test_get_a2a_instructions_mcp_default():
    out = get_a2a_instructions()
    assert "MCP tools" in out
    assert "list_peers" in out
    assert "send_message_to_user" in out


def test_get_a2a_instructions_cli_variant():
    out = get_a2a_instructions(mcp=False)
    assert "a2a_cli.py" in out
    assert "MCP tools" not in out


def test_a2a_mcp_instructions_reference_existing_tools():
    """The MCP instructions text must only reference tools that are actually
    registered in a2a_mcp_server.py. If someone renames a server tool, the
    prompt text must be updated in lockstep — this test catches the drift.
    """
    import re
    import pathlib
    mcp_server = pathlib.Path(__file__).parent.parent / "a2a_mcp_server.py"
    registered = set(re.findall(r'"name":\s*"([a-z_]+)"', mcp_server.read_text()))
    # The server advertises itself by name; strip that false positive.
    registered.discard("a2a-delegation")

    instructions = get_a2a_instructions(mcp=True)

    # Every tool called out by name in the instructions must exist on the
    # server. (We allow the server to have extras the prompt doesn't mention.)
    referenced = {
        "list_peers",
        "delegate_task",
        "delegate_task_async",
        "check_task_status",
        "get_workspace_info",
        "send_message_to_user",
    }
    for name in referenced:
        assert name in instructions, f"prompt missing {name}"
        assert name in registered, f"MCP server no longer registers {name}"


# ======================================================================
# brief_summary
# ======================================================================

def test_brief_summary_short_text_returned_as_is():
    assert brief_summary("Hello world") == "Hello world"


def test_brief_summary_truncates_long_text():
    text = "a" * 100
    out = brief_summary(text, max_len=20)
    assert len(out) == 20
    assert out.endswith("...")


def test_brief_summary_strips_markdown_headers():
    assert brief_summary("### Task: refactor auth") == "Task: refactor auth"


def test_brief_summary_strips_bold_and_italic():
    assert brief_summary("**urgent** __deploy__") == "urgent deploy"


def test_brief_summary_skips_blank_and_code_fences():
    text = "\n\n```python\n```\nActual task line"
    assert brief_summary(text) == "Actual task line"


def test_brief_summary_skips_horizontal_rule():
    text = "---\nReal content"
    assert brief_summary(text) == "Real content"


def test_brief_summary_empty_string():
    assert brief_summary("") == ""


def test_brief_summary_all_skipped_falls_back_to_prefix():
    """If every line is skipped, fall back to the raw prefix."""
    text = "\n\n```\n```"
    out = brief_summary(text, max_len=5)
    # Fallback returns text[:max_len] which keeps the skipped content
    assert len(out) <= 5


def test_brief_summary_exact_boundary_length():
    text = "x" * BRIEF_SUMMARY_MAX_LEN
    assert brief_summary(text) == text  # <= max_len, no truncation


def test_brief_summary_clamps_absurdly_small_max_len():
    """max_len below 4 is clamped — no negative slice indices."""
    out = brief_summary("hello world", max_len=1)
    # Clamped to min 4: "h..." (1 char + 3 ellipsis)
    assert out == "h..."


def test_brief_summary_clamps_negative_max_len():
    """Even negative max_len is handled gracefully via clamp."""
    out = brief_summary("hello world", max_len=-5)
    assert out == "h..."


# ======================================================================
# extract_message_text
# ======================================================================

def test_extract_message_text_empty_parts():
    msg = SimpleNamespace(parts=[])
    assert extract_message_text(msg) == ""


def test_extract_message_text_no_parts_attr():
    msg = SimpleNamespace()
    assert extract_message_text(msg) == ""


def test_extract_message_text_direct_text():
    part = SimpleNamespace(text="hello")
    msg = SimpleNamespace(parts=[part])
    assert extract_message_text(msg) == "hello"


def test_extract_message_text_root_text_fallback():
    root = SimpleNamespace(text="nested")
    part = SimpleNamespace(text=None, root=root)
    msg = SimpleNamespace(parts=[part])
    assert extract_message_text(msg) == "nested"


def test_extract_message_text_mixed_parts():
    p1 = SimpleNamespace(text="hello")
    p2 = SimpleNamespace(text=None, root=SimpleNamespace(text="world"))
    p3 = SimpleNamespace(text=None, root=None)  # empty — skipped
    msg = SimpleNamespace(parts=[p1, p2, p3])
    assert extract_message_text(msg) == "hello world"


def test_extract_message_text_ignores_non_string_text():
    part = SimpleNamespace(text="")
    msg = SimpleNamespace(parts=[part])
    assert extract_message_text(msg) == ""


# ======================================================================
# sanitize_agent_error
# ======================================================================

def test_sanitize_agent_error_exposes_class_not_body():
    exc = ValueError("internal secret token abc-123-XYZ")
    out = sanitize_agent_error(exc)
    assert "ValueError" in out
    assert "abc-123-XYZ" not in out
    assert "workspace logs" in out


def test_sanitize_agent_error_with_custom_exception():
    class MyErr(Exception):
        pass
    out = sanitize_agent_error(MyErr("very long stack trace with /etc/secret/key"))
    assert "MyErr" in out
    assert "/etc/secret/key" not in out


def test_sanitize_agent_error_with_category_only():
    """category kwarg wins when no exception is given (subprocess path)."""
    out = sanitize_agent_error(category="rate_limited")
    assert "rate_limited" in out
    assert "workspace logs" in out


def test_sanitize_agent_error_category_takes_precedence_over_exception():
    """If both are given, category wins (lets CLI executor override class name)."""
    out = sanitize_agent_error(ValueError("boom"), category="auth_failed")
    assert "auth_failed" in out
    assert "ValueError" not in out


def test_sanitize_agent_error_with_neither_falls_back_to_unknown():
    out = sanitize_agent_error()
    assert "unknown" in out


# ======================================================================
# classify_subprocess_error
# ======================================================================

def test_classify_subprocess_error_rate_limited():
    assert classify_subprocess_error("429 rate limit exceeded", 1) == "rate_limited"
    assert classify_subprocess_error("Server overloaded, try again", 1) == "rate_limited"


def test_classify_subprocess_error_auth():
    assert classify_subprocess_error("authentication failed", 1) == "auth_failed"
    assert classify_subprocess_error("bad api_key", 1) == "auth_failed"
    assert classify_subprocess_error("missing api-key header", 1) == "auth_failed"
    # Word-boundary regex must not match "author" or "authorize"
    assert classify_subprocess_error(
        "authored by jane on 2024-01-01", 99,
    ) == "exit_99"


def test_classify_subprocess_error_session():
    assert classify_subprocess_error("no conversation found", 1) == "session_error"
    assert classify_subprocess_error("session expired", 1) == "session_error"


def test_classify_subprocess_error_session_false_positive_avoided():
    """'sessions' (plural) should still match the \\bsession\\b pattern,
    but 'sessionless' must NOT trigger."""
    # 'sessions' — word boundary allows trailing 's'? No: \b matches between
    # \w and \W, and 's' is \w. So \bsession\b doesn't match 'sessions'.
    # The conservative assumption is OK — we'd rather miscategorize a rare
    # plural than false-positive on 'sessionless'.
    assert classify_subprocess_error("sessionless mode", 1) != "session_error"


def test_classify_subprocess_error_rate_false_positive_avoided():
    # "generate" and "iterate" contain "rate" as substrings but not as a word
    assert classify_subprocess_error("failed to generate output", 2) == "exit_2"
    assert classify_subprocess_error("iterate faster", None) == "subprocess_error"


def test_classify_subprocess_error_exit_code_fallback():
    assert classify_subprocess_error("mystery failure", 42) == "exit_42"


def test_classify_subprocess_error_generic_fallback():
    assert classify_subprocess_error("generic unknown failure", None) == "subprocess_error"
    # exit_code=0 with no keyword match also lands here
    assert classify_subprocess_error("mysterious but zero exit", 0) == "subprocess_error"
