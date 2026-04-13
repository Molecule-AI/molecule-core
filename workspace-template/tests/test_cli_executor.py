"""Tests for cli_executor.py — CLI-based agent executor."""

import asyncio
import json
import os
from pathlib import Path
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from config import RuntimeConfig
from cli_executor import CLIAgentExecutor
from executor_helpers import brief_summary as _brief_summary


def _make_executor(
    runtime="codex",
    runtime_config=None,
    system_prompt="You are a helpful agent.",
    heartbeat=None,
    config_path="/configs",
):
    """Build a CLIAgentExecutor with mocked externals."""
    if runtime_config is None:
        runtime_config = RuntimeConfig()
    with patch("shutil.which", return_value="/usr/bin/claude"):
        executor = CLIAgentExecutor(
            runtime=runtime,
            runtime_config=runtime_config,
            system_prompt=system_prompt,
            heartbeat=heartbeat,
            config_path=config_path,
        )
    return executor


def _make_context(text_parts, context_id="ctx-test"):
    """Helper to build a mock RequestContext."""
    parts = []
    for t in text_parts:
        p = MagicMock()
        p.text = t
        parts.append(p)
    context = MagicMock()
    context.message.parts = parts
    context.context_id = context_id
    return context


def _make_event_queue():
    """Helper to build a mock EventQueue with async enqueue_event."""
    return AsyncMock()


# ---------- _build_command tests ----------


def test_build_command_codex_defaults():
    """Verify codex preset produces the expected flags."""
    executor = _make_executor()
    cmd = executor._build_command("Hello world")

    assert cmd[0] == "codex"
    assert "--print" in cmd
    assert "--dangerously-skip-permissions" in cmd
    # No --output-format json anymore — that was a dead claude-code branch.
    assert "--output-format" not in cmd
    # Prompt flag and message at the end
    assert "-p" in cmd
    idx = cmd.index("-p")
    assert cmd[idx + 1] == "Hello world"


def test_cli_executor_rejects_claude_code_runtime():
    """Claude-code is served by ClaudeSDKExecutor — CLI path must refuse it."""
    from cli_executor import CLIAgentExecutor
    with pytest.raises(ValueError, match="ClaudeSDKExecutor"):
        CLIAgentExecutor(
            runtime="claude-code",
            runtime_config=RuntimeConfig(),
        )


# classify_subprocess_error / sanitize_agent_error tests moved to
# test_executor_helpers.py — the function lives in executor_helpers now.


def test_build_command_model_flag():
    """Verify --model flag is included when model is set."""
    rc = RuntimeConfig(model="opus")
    executor = _make_executor(runtime_config=rc)

    cmd = executor._build_command("Test")

    assert "--model" in cmd
    idx = cmd.index("--model")
    assert cmd[idx + 1] == "opus"


def test_build_command_no_model_flag_when_empty():
    """Verify --model flag is NOT included when model is empty."""
    rc = RuntimeConfig(model="")
    executor = _make_executor(runtime_config=rc)

    cmd = executor._build_command("Test")

    assert "--model" not in cmd


def test_system_prompt_included_every_call():
    """System prompt is injected on every call now that the CLI executor
    no longer tracks session state."""
    executor = _make_executor(system_prompt="Be helpful")
    cmd_first = executor._build_command("First message")
    cmd_second = executor._build_command("Second message")
    assert "--system-prompt" in cmd_first
    assert "--system-prompt" in cmd_second


# ---------- execute tests ----------


@pytest.mark.asyncio
async def test_set_current_task_on_execute():
    """Heartbeat is updated with the task summary, then cleared."""
    heartbeat = MagicMock()
    heartbeat.current_task = ""
    heartbeat.active_tasks = 0

    executor = _make_executor(heartbeat=heartbeat)
    executor._run_cli = AsyncMock()

    task_values = []

    async def tracking_set(hb, task):
        task_values.append(task)
        if hb:
            hb.current_task = task
            hb.active_tasks = 1 if task else 0

    part = MagicMock()
    part.text = "Build the feature"
    context = MagicMock()
    context.message.parts = [part]
    eq = _make_event_queue()

    with patch("cli_executor.set_current_task", new=tracking_set), \
         patch("cli_executor.read_delegation_results", return_value=""), \
         patch("cli_executor.recall_memories", new=AsyncMock(return_value="")), \
         patch("cli_executor.commit_memory", new=AsyncMock()):
        await executor.execute(context, eq)

    assert len(task_values) == 2
    assert task_values[0] != ""   # brief summary set at start
    assert task_values[1] == ""   # cleared at end


@pytest.mark.asyncio
async def test_empty_message_rejected():
    """Verify empty message returns error event."""
    executor = _make_executor()

    # Part with no text
    part = MagicMock(spec=[])  # no .text attribute

    context = MagicMock()
    context.message.parts = [part]
    eq = _make_event_queue()

    await executor.execute(context, eq)

    eq.enqueue_event.assert_called_once()
    event_arg = eq.enqueue_event.call_args[0][0]
    assert "Error" in str(event_arg) or "no text" in str(event_arg)


# ---------- _brief_summary tests ----------


def test_brief_summary_simple():
    """Simple single-line text is returned as-is."""
    assert _brief_summary("Hello world") == "Hello world"


def test_brief_summary_truncation():
    """Long text is truncated with ellipsis."""
    long_text = "A" * 100
    result = _brief_summary(long_text, max_len=80)
    assert len(result) == 80
    assert result.endswith("...")


def test_brief_summary_strips_markdown_headers():
    """Markdown headers (# ## ###) are stripped."""
    assert _brief_summary("## Build the feature") == "Build the feature"
    assert _brief_summary("### Deploy to prod") == "Deploy to prod"


def test_brief_summary_skips_empty_and_code_fences():
    """Empty lines and code fence markers are skipped; first non-skippable line returned."""
    text = "\n\n```python\nActual summary"
    assert _brief_summary(text) == "Actual summary"


def test_brief_summary_strips_bold_italic():
    """Markdown bold/italic markers are removed."""
    assert _brief_summary("**Important** task") == "Important task"
    assert _brief_summary("__Urgent__ fix") == "Urgent fix"


def test_brief_summary_skips_hr():
    """Horizontal rules (---) are skipped."""
    text = "---\nReal content"
    assert _brief_summary(text) == "Real content"


def test_brief_summary_fallback():
    """When all lines are empty/fences/hr, falls back to truncated raw text."""
    text = "\n\n---\n```\n"
    result = _brief_summary(text)
    # All lines are skippable, so falls back to text[:max_len]
    assert result == text[:80]


# ---------- _run_cli tests ----------


@pytest.mark.asyncio
async def test_run_cli_success():
    """Successful CLI execution enqueues the raw stdout as the response."""
    executor = _make_executor()

    mock_proc = AsyncMock()
    mock_proc.communicate = AsyncMock(return_value=(b"Done!", b""))
    mock_proc.returncode = 0

    eq = _make_event_queue()

    with patch("asyncio.create_subprocess_exec", return_value=mock_proc):
        await executor._run_cli("Do something", eq)

    eq.enqueue_event.assert_called_once()
    event_arg = eq.enqueue_event.call_args[0][0]
    assert "Done!" in str(event_arg)


@pytest.mark.asyncio
async def test_run_cli_timeout():
    """CLI timeout enqueues a timeout error message."""
    rc = RuntimeConfig(timeout=10)
    executor = _make_executor(runtime_config=rc)

    mock_proc = AsyncMock()
    mock_proc.communicate = AsyncMock(side_effect=asyncio.TimeoutError())
    mock_proc.kill = AsyncMock()
    mock_proc.wait = AsyncMock()

    eq = _make_event_queue()

    with patch("asyncio.create_subprocess_exec", return_value=mock_proc):
        with patch("asyncio.wait_for", side_effect=asyncio.TimeoutError()):
            await executor._run_cli("Slow task", eq)

    eq.enqueue_event.assert_called_once()
    event_arg = eq.enqueue_event.call_args[0][0]
    assert "timeout" in str(event_arg)


@pytest.mark.asyncio
async def test_run_cli_extra_args():
    """Extra args from RuntimeConfig are included in the command."""
    rc = RuntimeConfig(args=["--verbose", "--no-cache"])
    executor = _make_executor(runtime_config=rc)

    cmd = executor._build_command("Test task")
    assert "--verbose" in cmd
    assert "--no-cache" in cmd


# ---------- constructor / preset tests ----------


def test_unknown_runtime_raises():
    """Unknown runtime raises ValueError."""
    with patch("shutil.which", return_value="/usr/bin/whatever"):
        with pytest.raises(ValueError, match="Unknown runtime"):
            CLIAgentExecutor(
                runtime="totally-unknown",
                runtime_config=RuntimeConfig(),
            )


def test_custom_runtime_preset():
    """custom runtime builds preset from RuntimeConfig.command."""
    rc = RuntimeConfig(command="myagent")
    with patch("shutil.which", return_value="/usr/bin/myagent"):
        executor = CLIAgentExecutor(runtime="custom", runtime_config=rc)
    assert executor.preset["command"] == "myagent"
    assert executor.preset["prompt_flag"] == "-p"


# ---------- _resolve_auth_token from file (lines 173-179) ----------


def test_resolve_auth_token_from_file(tmp_path):
    """Auth token is read from file when the token file exists."""
    token_file = tmp_path / ".my-token"
    token_file.write_text("file-secret-token\n")

    rc = RuntimeConfig(auth_token_file=".my-token")
    with patch("shutil.which", return_value="/usr/bin/claude"):
        with patch.dict(os.environ, {}, clear=False):
            # Ensure no env var override
            os.environ.pop("CLAUDE_CODE_OAUTH_TOKEN", None)
            executor = CLIAgentExecutor(
                runtime="codex",
                runtime_config=rc,
                config_path=str(tmp_path),
            )
    assert executor._auth_token == "file-secret-token"


def test_resolve_auth_token_from_preset_default_file(tmp_path):
    """Auth token from the preset's default_auth_file when config file_name is empty."""
    # Use a test runtime with a default_auth_file preset entry so we don't
    # depend on the (now-removed) claude-code preset.
    from cli_executor import RUNTIME_PRESETS
    RUNTIME_PRESETS["filetoken-test"] = {
        "command": "fakecli",
        "base_args": [],
        "prompt_flag": "-p",
        "model_flag": None,
        "system_prompt_flag": None,
        "auth_pattern": None,
        "default_auth_env": "",
        "default_auth_file": ".auth-token",
    }
    try:
        token_file = tmp_path / ".auth-token"
        token_file.write_text("preset-default-token")
        rc = RuntimeConfig()  # no explicit auth_token_file
        with patch("shutil.which", return_value="/usr/bin/fakecli"):
            executor = CLIAgentExecutor(
                runtime="filetoken-test",
                runtime_config=rc,
                config_path=str(tmp_path),
            )
        assert executor._auth_token == "preset-default-token"
    finally:
        RUNTIME_PRESETS.pop("filetoken-test", None)


def test_resolve_auth_token_returns_none_when_no_file_and_no_env(tmp_path):
    """Returns None when neither env var nor file is present."""
    rc = RuntimeConfig()
    with patch("shutil.which", return_value="/usr/bin/claude"):
        with patch.dict(os.environ, {}, clear=False):
            os.environ.pop("CLAUDE_CODE_OAUTH_TOKEN", None)
            executor = CLIAgentExecutor(
                runtime="codex",
                runtime_config=rc,
                config_path=str(tmp_path),  # no .auth-token file here
            )
    assert executor._auth_token is None


# ---------- _create_auth_helper (lines 183-189) and line 139 ----------


def test_create_auth_helper_creates_executable_script(tmp_path):
    """_create_auth_helper creates a shell script that outputs the token."""
    rc = RuntimeConfig()
    with patch("shutil.which", return_value="/usr/bin/claude"):
        executor = CLIAgentExecutor(
            runtime="codex",
            runtime_config=rc,
            config_path=str(tmp_path),
        )

    helper_path = executor._create_auth_helper("my-secret-token")
    assert helper_path is not None
    script_content = Path(helper_path).read_text()
    assert "#!/bin/sh" in script_content
    assert "my-secret-token" in script_content
    # Cleanup
    import os as _os
    _os.unlink(helper_path)


def test_auth_helper_created_when_apiKeyHelper_pattern(tmp_path):
    """_auth_helper_path is set when auth_pattern=apiKeyHelper and token present."""
    from cli_executor import RUNTIME_PRESETS
    # We'll use a custom runtime and monkeypatch the preset
    rc = RuntimeConfig(command="fakecli", auth_token_env="FAKE_API_KEY")

    with patch("shutil.which", return_value="/usr/bin/fakecli"):
        with patch.dict(os.environ, {"FAKE_API_KEY": "test-api-key"}):
            executor = CLIAgentExecutor(
                runtime="custom",
                runtime_config=rc,
                config_path=str(tmp_path),
            )
            # Patch the preset to apiKeyHelper pattern and call manually
            executor.preset["auth_pattern"] = "apiKeyHelper"
            executor._auth_token = "test-api-key"
            helper_path = executor._create_auth_helper("test-api-key")
            executor._auth_helper_path = helper_path

    assert executor._auth_helper_path is not None
    content = Path(executor._auth_helper_path).read_text()
    assert "test-api-key" in content


# A2A instructions tests moved to test_executor_helpers.py — the CLI executor
# now calls get_a2a_instructions() from the shared module directly in
# _build_command(), with no wrapper method of its own.


# Helper-method tests for _set_current_task, _recall_memories, _commit_memory
# moved to tests/test_executor_helpers.py — they exercise shared code in
# executor_helpers.py that both CLIAgentExecutor and ClaudeSDKExecutor call.


async def test_execute_injects_delegation_results_into_prompt(tmp_path):
    """When delegation results are present, execute() prepends them to the prompt."""
    executor = _make_executor(config_path=str(tmp_path))
    ctx = _make_context(["Follow up question"])
    eq = _make_event_queue()

    captured = {}

    async def fake_run_cli(user_input, _event_queue):
        captured["user_input"] = user_input

    with patch("cli_executor.read_delegation_results",
               return_value="- [completed] Prior task done"), \
         patch("cli_executor.recall_memories", new=AsyncMock(return_value="")), \
         patch("cli_executor.set_current_task", new=AsyncMock()), \
         patch("cli_executor.commit_memory", new=AsyncMock()), \
         patch.object(executor, "_run_cli", side_effect=fake_run_cli):
        await executor.execute(ctx, eq)

    assert "Delegation results received while you were idle" in captured["user_input"]
    assert "Prior task done" in captured["user_input"]
    assert "Follow up question" in captured["user_input"]


async def test_run_cli_timeout_kill_already_exited():
    """ProcessLookupError from kill() (proc already exited) is silently skipped."""
    executor = _make_executor(
        runtime_config=RuntimeConfig(timeout=1),
    )
    eq = _make_event_queue()

    proc = AsyncMock()
    proc.communicate = AsyncMock(side_effect=asyncio.TimeoutError())
    proc.kill = MagicMock(side_effect=ProcessLookupError())
    proc.wait = AsyncMock()

    with patch("asyncio.create_subprocess_exec", return_value=proc):
        await executor._run_cli("task", eq)

    eq.enqueue_event.assert_called_once()
    assert "timeout" in str(eq.enqueue_event.call_args[0][0])


async def test_run_cli_env_pattern_propagates_auth_token(tmp_path):
    """When auth_pattern=env, the auth token is injected into subprocess env."""
    rc = RuntimeConfig(auth_token_env="MY_TOKEN")
    with patch("shutil.which", return_value="/usr/bin/claude"), \
         patch.dict(os.environ, {"MY_TOKEN": "secret-token-xyz"}, clear=False):
        executor = CLIAgentExecutor(
            runtime="codex",
            runtime_config=rc,
            config_path=str(tmp_path),
        )

    captured_env = {}

    async def fake_create_subprocess_exec(*args, **kwargs):
        captured_env.update(kwargs.get("env") or {})
        proc = AsyncMock()
        proc.returncode = 0
        proc.communicate = AsyncMock(return_value=(b'{"result": "ok"}', b""))
        return proc

    eq = _make_event_queue()
    with patch("asyncio.create_subprocess_exec", side_effect=fake_create_subprocess_exec):
        await executor._run_cli("hi", eq)

    assert captured_env.get("MY_TOKEN") == "secret-token-xyz"


async def test_run_cli_session_error_exhausts_all_retries():
    """Session errors retried until exhaustion then surface as error."""
    executor = _make_executor()
    eq = _make_event_queue()

    proc = AsyncMock()
    proc.returncode = 1
    proc.communicate = AsyncMock(return_value=(b"", b"no conversation found with that id"))

    with patch("asyncio.create_subprocess_exec", return_value=proc), \
         patch("asyncio.sleep", new=AsyncMock()):
        await executor._run_cli("task", eq)

    eq.enqueue_event.assert_called_once()


async def test_run_cli_timeout_kill_raises_generic_exception():
    """Kill raising non-ProcessLookupError is logged and swallowed."""
    executor = _make_executor(
        runtime_config=RuntimeConfig(timeout=1),
    )
    eq = _make_event_queue()

    proc = AsyncMock()
    proc.communicate = AsyncMock(side_effect=asyncio.TimeoutError())
    proc.kill = MagicMock(side_effect=RuntimeError("kill refused"))
    proc.wait = AsyncMock()

    with patch("asyncio.create_subprocess_exec", return_value=proc):
        await executor._run_cli("task", eq)

    eq.enqueue_event.assert_called_once()
    assert "timeout" in str(eq.enqueue_event.call_args[0][0])


async def test_run_cli_timeout_proc_wait_raises_generic_exception():
    """proc.wait() raising non-TimeoutError exception is logged and swallowed."""
    executor = _make_executor(
        runtime_config=RuntimeConfig(timeout=1),
    )
    eq = _make_event_queue()

    proc = AsyncMock()
    proc.communicate = AsyncMock(side_effect=asyncio.TimeoutError())
    proc.kill = MagicMock()
    proc.wait = AsyncMock(side_effect=RuntimeError("wait broken"))

    with patch("asyncio.create_subprocess_exec", return_value=proc):
        await executor._run_cli("task", eq)

    eq.enqueue_event.assert_called_once()
    assert "timeout" in str(eq.enqueue_event.call_args[0][0])


# ---------- _build_command ollama model positional arg (line 316) ----------


def test_build_command_ollama_model_positional(tmp_path):
    """Ollama runtime: model is appended positionally (no model_flag)."""
    rc = RuntimeConfig(model="llama3")
    with patch("shutil.which", return_value="/usr/bin/ollama"):
        executor = CLIAgentExecutor(
            runtime="ollama",
            runtime_config=rc,
            config_path=str(tmp_path),
        )
    cmd = executor._build_command("Hello")
    assert "llama3" in cmd
    # Model should appear after "run" and before prompt
    run_idx = cmd.index("run")
    model_idx = cmd.index("llama3")
    assert model_idx > run_idx


# ---------- _build_command apiKeyHelper auth settings (lines 331-332) ----------


def test_build_command_includes_apiKeyHelper_settings(tmp_path):
    """Command includes --settings with apiKeyHelper when auth_helper_path is set."""
    rc = RuntimeConfig(command="fakecli", auth_token_env="FAKE_KEY")
    with patch("shutil.which", return_value="/usr/bin/fakecli"):
        with patch.dict(os.environ, {"FAKE_KEY": "my-api-key"}):
            executor = CLIAgentExecutor(
                runtime="custom",
                runtime_config=rc,
                config_path=str(tmp_path),
            )
    # Manually simulate apiKeyHelper auth pattern
    executor.preset["auth_pattern"] = "apiKeyHelper"
    executor._auth_helper_path = "/tmp/fake-helper.sh"

    cmd = executor._build_command("test message")
    assert "--settings" in cmd
    settings_idx = cmd.index("--settings")
    settings_val = json.loads(cmd[settings_idx + 1])
    assert settings_val["apiKeyHelper"] == "/tmp/fake-helper.sh"


# ---------- _build_command positional prompt for ollama (line 351) ----------


def test_build_command_ollama_positional_prompt(tmp_path):
    """Ollama runtime: prompt is appended positionally (no prompt_flag)."""
    rc = RuntimeConfig()
    with patch("shutil.which", return_value="/usr/bin/ollama"):
        executor = CLIAgentExecutor(
            runtime="ollama",
            runtime_config=rc,
            config_path=str(tmp_path),
        )
    cmd = executor._build_command("my ollama prompt")
    # prompt_flag is None, so prompt goes at end positionally
    assert cmd[-1] == "my ollama prompt"
    assert "-p" not in cmd


# Session-id-from-JSON test removed: the claude-code runtime used to emit
# --output-format json and the CLI executor parsed it. Now claude-code goes
# through ClaudeSDKExecutor, so the CLI executor no longer JSON-parses stdout.


# ---------- _run_cli: rate limit retry (lines 442-443, 468-474) ----------


async def test_run_cli_rate_limit_retry_then_success():
    """Rate limit on stderr triggers retry; second attempt succeeds."""
    executor = _make_executor()

    call_count = 0

    async def mock_communicate():
        nonlocal call_count
        call_count += 1
        if call_count == 1:
            return (b"", b"rate limit exceeded (429)")
        return (b'{"result": "Success after retry"}', b"")

    mock_proc = AsyncMock()
    mock_proc.returncode = 1
    mock_proc.communicate = mock_communicate

    # Second call proc has returncode 0
    mock_proc2 = AsyncMock()
    mock_proc2.communicate = AsyncMock(
        return_value=(b'{"result": "Success after retry"}', b"")
    )
    mock_proc2.returncode = 0

    procs = iter([mock_proc, mock_proc2])

    eq = _make_event_queue()

    with patch("asyncio.create_subprocess_exec", side_effect=lambda *a, **kw: next(procs)):
        with patch("asyncio.sleep", new_callable=AsyncMock):
            await executor._run_cli("Do task", eq)

    assert eq.enqueue_event.call_count >= 1


async def test_run_cli_rate_limit_in_stderr_retries():
    """Rate limit keyword in stderr causes retry with backoff."""
    executor = _make_executor()

    attempts = []

    proc_fail = AsyncMock()
    proc_fail.returncode = 1
    proc_fail.communicate = AsyncMock(return_value=(b"", b"overloaded"))

    proc_ok = AsyncMock()
    proc_ok.returncode = 0
    proc_ok.communicate = AsyncMock(return_value=(b'{"result": "ok"}', b""))

    call_iter = iter([proc_fail, proc_ok])

    eq = _make_event_queue()

    with patch("asyncio.create_subprocess_exec", side_effect=lambda *a, **kw: next(call_iter)):
        with patch("asyncio.sleep", new_callable=AsyncMock):
            await executor._run_cli("task", eq)

    eq.enqueue_event.assert_called_once()


# ---------- _run_cli: auth error clears session and retries (line 364+) ----------


async def test_run_cli_auth_error_retries():
    """Auth error in stderr triggers retry."""
    executor = _make_executor()

    proc_auth_err = AsyncMock()
    proc_auth_err.returncode = 1
    proc_auth_err.communicate = AsyncMock(
        return_value=(b"", b"authentication error: invalid X-Api-Key")
    )

    proc_ok = AsyncMock()
    proc_ok.returncode = 0
    proc_ok.communicate = AsyncMock(return_value=(b"retried ok", b""))

    call_iter = iter([proc_auth_err, proc_ok])

    eq = _make_event_queue()

    with patch("asyncio.create_subprocess_exec", side_effect=lambda *a, **kw: next(call_iter)):
        with patch("asyncio.sleep", new_callable=AsyncMock):
            await executor._run_cli("task", eq)

    assert eq.enqueue_event.call_count >= 1


# ---------- _run_cli: empty result all retries exhausted (lines 455-464) ----------


async def test_run_cli_empty_result_all_retries_returns_no_response():
    """When all retries return empty stdout, enqueue 'no response' message."""
    executor = _make_executor()

    proc = AsyncMock()
    proc.returncode = 0
    proc.communicate = AsyncMock(return_value=(b"", b""))

    eq = _make_event_queue()

    with patch("asyncio.create_subprocess_exec", return_value=proc):
        with patch("asyncio.sleep", new_callable=AsyncMock):
            await executor._run_cli("task", eq)

    eq.enqueue_event.assert_called_once()
    event_text = str(eq.enqueue_event.call_args[0][0])
    assert "no response" in event_text


async def test_run_cli_empty_result_on_intermediate_attempt_retries():
    """Empty stdout on first attempt triggers retry before giving up."""
    executor = _make_executor()

    call_count = 0

    async def varying_communicate():
        nonlocal call_count
        call_count += 1
        if call_count < 3:
            return (b"", b"")
        return (b"finally got one", b"")

    proc = AsyncMock()
    proc.returncode = 0
    proc.communicate = varying_communicate

    eq = _make_event_queue()

    with patch("asyncio.create_subprocess_exec", return_value=proc):
        with patch("asyncio.sleep", new_callable=AsyncMock):
            await executor._run_cli("task", eq)

    eq.enqueue_event.assert_called_once()
    assert "finally got one" in str(eq.enqueue_event.call_args[0][0])
    assert "finally got one" in str(eq.enqueue_event.call_args[0][0])


# ---------- _run_cli: timeout with proc.kill raising (lines 497-498) ----------


async def test_run_cli_timeout_proc_kill_raises():
    """Timeout handler swallows exception from proc.kill()."""
    rc = RuntimeConfig(timeout=5)
    executor = _make_executor(runtime_config=rc)

    mock_proc = AsyncMock()
    mock_proc.kill = MagicMock(side_effect=OSError("no such process"))
    mock_proc.wait = AsyncMock()

    eq = _make_event_queue()

    with patch("asyncio.create_subprocess_exec", return_value=mock_proc):
        with patch("asyncio.wait_for", side_effect=asyncio.TimeoutError()):
            await executor._run_cli("slow task", eq)

    eq.enqueue_event.assert_called_once()
    assert "timeout" in str(eq.enqueue_event.call_args[0][0])


async def test_run_cli_timeout_calls_proc_wait_to_reap_zombie():
    """On timeout, proc.kill() is followed by proc.wait() to reap the zombie process."""
    rc = RuntimeConfig(timeout=5)
    executor = _make_executor(runtime_config=rc)

    mock_proc = AsyncMock()
    mock_proc.kill = MagicMock()
    mock_proc.wait = AsyncMock()

    eq = _make_event_queue()

    # First wait_for call (for proc.communicate) raises TimeoutError
    # Second wait_for call (for proc.wait inside the timeout handler) succeeds
    call_count = {"n": 0}
    original_wait_for = asyncio.wait_for

    async def patched_wait_for(coro, timeout):
        call_count["n"] += 1
        if call_count["n"] == 1:
            # Cancel the coro to avoid resource warning, then raise
            try:
                coro.close()
            except Exception:
                pass
            raise asyncio.TimeoutError()
        # Subsequent calls (for proc.wait reap) succeed immediately
        try:
            await coro
        except Exception:
            pass

    with patch("asyncio.create_subprocess_exec", return_value=mock_proc):
        with patch("asyncio.wait_for", side_effect=patched_wait_for):
            await executor._run_cli("slow task", eq)

    # Verify proc.kill was called
    mock_proc.kill.assert_called_once()
    # Verify proc.wait was called (to reap the zombie)
    mock_proc.wait.assert_called()
    # And we got the timeout message
    eq.enqueue_event.assert_called_once()
    assert "timeout" in str(eq.enqueue_event.call_args[0][0])


async def test_run_cli_timeout_proc_wait_also_times_out():
    """If proc.wait() also times out (truly stuck), we still send the timeout message."""
    rc = RuntimeConfig(timeout=5)
    executor = _make_executor(runtime_config=rc)

    mock_proc = AsyncMock()
    mock_proc.kill = MagicMock()
    mock_proc.wait = AsyncMock()

    eq = _make_event_queue()

    with patch("asyncio.create_subprocess_exec", return_value=mock_proc):
        with patch("asyncio.wait_for", side_effect=asyncio.TimeoutError()):
            await executor._run_cli("slow task", eq)

    # Even though both wait_for calls timed out, we still emit the timeout event
    eq.enqueue_event.assert_called_once()
    assert "timeout" in str(eq.enqueue_event.call_args[0][0])
    # And we still tried to kill
    mock_proc.kill.assert_called_once()


# ---------- _run_cli: non-zero exit with no stderr (line 466) ----------


async def test_run_cli_nonzero_exit_no_stderr_uses_exit_code():
    """Non-zero exit with no stderr falls back to 'Exit code N' message."""
    executor = _make_executor(runtime="ollama")

    proc = AsyncMock()
    proc.returncode = 2
    proc.communicate = AsyncMock(return_value=(b"", b""))

    eq = _make_event_queue()

    with patch("asyncio.create_subprocess_exec", return_value=proc):
        await executor._run_cli("task", eq)

    event_text = str(eq.enqueue_event.call_args[0][0])
    assert "Exit code" in event_text or "Agent error" in event_text


# ---------- _run_cli: generic exception (lines 503-508) ----------


async def test_run_cli_generic_exception_enqueues_error():
    """Unexpected exception from subprocess is caught and enqueued as error."""
    executor = _make_executor()
    eq = _make_event_queue()

    with patch(
        "asyncio.create_subprocess_exec",
        side_effect=RuntimeError("fork failed"),
    ):
        await executor._run_cli("task", eq)

    eq.enqueue_event.assert_called_once()
    assert "Agent error" in str(eq.enqueue_event.call_args[0][0])


# ---------- _run_cli: non-JSON output for claude-code runtime ----------


async def test_run_cli_non_json_output_used_raw():
    """claude-code runtime: non-JSON stdout is passed through as-is."""
    executor = _make_executor()

    proc = AsyncMock()
    proc.returncode = 0
    proc.communicate = AsyncMock(return_value=(b"plain text output", b""))

    eq = _make_event_queue()

    with patch("asyncio.create_subprocess_exec", return_value=proc):
        await executor._run_cli("task", eq)

    event_text = str(eq.enqueue_event.call_args[0][0])
    assert "plain text output" in event_text


# ---------- execute: memory injection path ----------


async def test_execute_injects_memories_into_prompt(tmp_path):
    """Memories are prepended to the prompt when returned."""
    executor = _make_executor(config_path=str(tmp_path))

    captured_inputs = []

    async def capture_run_cli(user_input, event_queue):
        captured_inputs.append(user_input)
        await event_queue.enqueue_event(MagicMock())

    executor._run_cli = capture_run_cli

    with patch("cli_executor.recall_memories",
               new=AsyncMock(return_value="- [LOCAL] remember this")), \
         patch("cli_executor.commit_memory", new=AsyncMock()), \
         patch("cli_executor.set_current_task", new=AsyncMock()), \
         patch("cli_executor.read_delegation_results", return_value=""):
        context = _make_context(["Do the task"])
        eq = _make_event_queue()
        await executor.execute(context, eq)

    assert len(captured_inputs) == 1
    assert "Prior context from memory" in captured_inputs[0]
    assert "remember this" in captured_inputs[0]
    assert "Do the task" in captured_inputs[0]


# ---------- execute: no memories, no injection ----------


async def test_execute_no_memories_no_injection(tmp_path):
    """No memories = prompt passed through unchanged."""
    executor = _make_executor(config_path=str(tmp_path))

    captured_inputs = []

    async def capture_run_cli(user_input, event_queue):
        captured_inputs.append(user_input)

    executor._run_cli = capture_run_cli

    with patch("cli_executor.recall_memories", new=AsyncMock(return_value="")), \
         patch("cli_executor.commit_memory", new=AsyncMock()), \
         patch("cli_executor.set_current_task", new=AsyncMock()), \
         patch("cli_executor.read_delegation_results", return_value=""):
        context = _make_context(["Clean task without memories"])
        eq = _make_event_queue()
        await executor.execute(context, eq)

    assert captured_inputs[0] == "Clean task without memories"


# ---------- line 139: _create_auth_helper called in __init__ ----------


def test_init_creates_auth_helper_when_apiKeyHelper_pattern(tmp_path):
    """Executor calls _create_auth_helper in __init__ when preset has apiKeyHelper pattern."""
    from cli_executor import RUNTIME_PRESETS

    api_key_preset = {
        "command": "fakecli",
        "base_args": [],
        "prompt_flag": "-p",
        "model_flag": None,
        "system_prompt_flag": None,
        "auth_pattern": "apiKeyHelper",
        "default_auth_env": "FAKE_API_KEY",
        "default_auth_file": "",
    }

    original = dict(RUNTIME_PRESETS)
    RUNTIME_PRESETS["api-key-test"] = api_key_preset
    try:
        rc = RuntimeConfig()
        with patch("shutil.which", return_value="/usr/bin/fakecli"):
            with patch.dict(os.environ, {"FAKE_API_KEY": "secret-key-value"}):
                executor = CLIAgentExecutor(
                    runtime="api-key-test",
                    runtime_config=rc,
                    config_path=str(tmp_path),
                )
        assert executor._auth_helper_path is not None
        content = Path(executor._auth_helper_path).read_text()
        assert "secret-key-value" in content
    finally:
        RUNTIME_PRESETS.clear()
        RUNTIME_PRESETS.update(original)


# ---------- line 161: warning when command not found in PATH ----------


def test_init_warns_when_command_not_found(tmp_path, caplog):
    """CLIAgentExecutor logs a warning when CLI command not found in PATH."""
    import logging
    rc = RuntimeConfig()
    with patch("shutil.which", return_value=None):
        with caplog.at_level(logging.WARNING, logger="cli_executor"):
            CLIAgentExecutor(
                runtime="codex",
                runtime_config=rc,
                config_path=str(tmp_path),
            )
    assert any("not found" in msg for msg in caplog.messages)


# ---------- lines 233-234: heartbeat updated in _set_current_task ----------


# Heartbeat-update tests for set_current_task moved to test_executor_helpers.py —
# the CLI executor now calls set_current_task() directly from the shared module
# with no wrapper of its own.


# ---------- line 296: _get_system_prompt reads from file ----------


# get_system_prompt tests moved to test_executor_helpers.py — the CLI executor
# now calls the shared helper directly with no wrapper.


# ---------- line 364: auth error retries exhaust → enqueues error ----------


async def test_run_cli_auth_error_exhausts_all_retries():
    """Auth error on every attempt eventually enqueues error (all retries spent)."""
    executor = _make_executor()

    proc = AsyncMock()
    proc.returncode = 1
    proc.communicate = AsyncMock(
        return_value=(b"", b"authentication error: invalid api_key")
    )

    eq = _make_event_queue()

    with patch("asyncio.create_subprocess_exec", return_value=proc):
        with patch("asyncio.sleep", new_callable=AsyncMock):
            await executor._run_cli("task", eq)

    assert eq.enqueue_event.call_count == 1
    event_text = str(eq.enqueue_event.call_args[0][0])
    assert "Agent error" in event_text


# ---------- line 364: execute() handles part.root.text ----------


async def test_execute_uses_root_text_when_no_direct_text(tmp_path):
    """execute() extracts text from part.root.text when part.text is absent."""
    executor = _make_executor(config_path=str(tmp_path))

    captured_inputs = []

    async def capture_run_cli(user_input, event_queue):
        captured_inputs.append(user_input)

    executor._run_cli = capture_run_cli

    # Build a part that has no .text but has .root.text
    part = MagicMock(spec=["root"])
    part.root = MagicMock()
    part.root.text = "text from root attribute"

    context = MagicMock()
    context.message.parts = [part]
    eq = _make_event_queue()
    with patch("cli_executor.recall_memories", new=AsyncMock(return_value="")), \
         patch("cli_executor.commit_memory", new=AsyncMock()), \
         patch("cli_executor.set_current_task", new=AsyncMock()), \
         patch("cli_executor.read_delegation_results", return_value=""):
        await executor.execute(context, eq)

    assert len(captured_inputs) == 1
    assert "text from root attribute" in captured_inputs[0]


# Delegation results tests moved to tests/test_executor_helpers.py — the
# function now lives in executor_helpers.read_delegation_results() and is
# shared by both executors.
