"""Tests for hermes_executor.py — Hermes OpenAI-compat A2A executor.

Coverage targets
----------------
- _reasoning_supported()        — model name pattern detection
- ProviderConfig                — capability flags derived from model name
- HermesA2AExecutor.__init__   — field assignment, client injection, tools (#497)
- HermesA2AExecutor._build_messages — system prompt + user turn assembly
- HermesA2AExecutor._log_reasoning  — OTEL span emission + swallowed errors
- HermesA2AExecutor.execute    — happy path, empty input, API error,
                                  Hermes 4 extra_body, Hermes 3 no extra_body,
                                  reasoning not in reply, reasoning_details,
                                  tools serialized in request body (#497),
                                  empty tools → no tools field (#497),
                                  tool_call response → JSON text (#497)
- HermesA2AExecutor.cancel     — TaskStatusUpdateEvent emitted

The ``openai`` module is stubbed in sys.modules so no real API call is made.
The A2A SDK types are already stubbed by conftest.py.
"""

import sys
from types import ModuleType
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

# ---------------------------------------------------------------------------
# Stub openai before hermes_executor is imported so AsyncOpenAI resolves to a
# controllable mock.  conftest.py already stubs a2a and builtin_tools.
# ---------------------------------------------------------------------------

if "openai" not in sys.modules:
    _openai_mod = ModuleType("openai")

    class _StubAsyncOpenAI:
        """Minimal stand-in for openai.AsyncOpenAI — tests override this."""

        def __init__(self, base_url=None, api_key=None):
            self.base_url = base_url
            self.api_key = api_key
            self.chat = MagicMock()

    _openai_mod.AsyncOpenAI = _StubAsyncOpenAI
    sys.modules["openai"] = _openai_mod

# ---------------------------------------------------------------------------
# Stub shared_runtime.extract_message_text (mirrors the real implementation).
# ---------------------------------------------------------------------------

if "shared_runtime" not in sys.modules:
    _sr_mod = ModuleType("shared_runtime")

    def _extract_message_text(context_or_parts) -> str:
        parts = getattr(getattr(context_or_parts, "message", None), "parts", None)
        if parts is None:
            parts = context_or_parts
        texts = []
        for p in parts or []:
            t = getattr(p, "text", None) or getattr(
                getattr(p, "root", None), "text", None
            ) or ""
            if t:
                texts.append(t)
        return " ".join(texts).strip()

    _sr_mod.extract_message_text = _extract_message_text
    sys.modules["shared_runtime"] = _sr_mod

# Now import the module under test
from hermes_executor import (  # noqa: E402
    HermesA2AExecutor,
    ProviderConfig,
    _HERMES4_PATTERNS,
    _reasoning_supported,
)


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _make_context(text: str, context_id: str = "ctx-test") -> MagicMock:
    """Return a mock RequestContext with the given text in message.parts."""
    part = MagicMock()
    part.text = text
    ctx = MagicMock()
    ctx.message.parts = [part]
    ctx.context_id = context_id
    return ctx


def _make_empty_context() -> MagicMock:
    """Return a context whose message parts contain no text."""
    part = MagicMock(spec=[])  # no .text attribute
    part.root = MagicMock(spec=[])  # no .root.text either
    ctx = MagicMock()
    ctx.message.parts = [part]
    ctx.context_id = "ctx-empty"
    return ctx


class _FakeMessage:
    """Minimal stand-in for openai ChatCompletionMessage.

    Only sets *reasoning* / *reasoning_details* as real attributes when
    explicitly provided — matching what an upstream OpenAI-compat provider
    returns (the SDK does NOT define these fields on ChatCompletionMessage;
    they arrive as dynamic extras).  Using a plain class rather than
    MagicMock avoids MagicMock's auto-attribute creation, which would cause
    ``getattr(msg, "reasoning", None)`` to return a truthy MagicMock even
    when the field was never set.
    """

    def __init__(
        self,
        content: str,
        reasoning: str | None = None,
        reasoning_details=None,
        *,
        _set_reasoning: bool = False,
        _set_reasoning_details: bool = False,
    ) -> None:
        self.content = content
        if _set_reasoning or reasoning is not None:
            self.reasoning = reasoning
        if _set_reasoning_details or reasoning_details is not None:
            self.reasoning_details = reasoning_details


def _make_api_response(content: str, reasoning: str | None = None, reasoning_details=None):
    """Build a mock OpenAI ChatCompletion response."""
    msg = _FakeMessage(content=content, reasoning=reasoning, reasoning_details=reasoning_details)
    choice = MagicMock()
    choice.message = msg
    response = MagicMock()
    response.choices = [choice]
    return response


def _make_executor(
    model: str = "nousresearch/hermes-4-0",
    system_prompt: str | None = "You are Hermes.",
) -> tuple[HermesA2AExecutor, AsyncMock]:
    """Return (executor, mock_client) with a pre-wired async mock client."""
    mock_client = MagicMock()
    mock_client.chat.completions.create = AsyncMock()
    executor = HermesA2AExecutor(
        model=model,
        system_prompt=system_prompt,
        _client=mock_client,
    )
    return executor, mock_client


# ---------------------------------------------------------------------------
# _reasoning_supported
# ---------------------------------------------------------------------------


def test_reasoning_supported_hermes4_slug():
    """Exact "hermes-4" substring → True."""
    assert _reasoning_supported("nousresearch/hermes-4-0") is True


def test_reasoning_supported_hermes4_nous_portal():
    """Nous Portal style slug containing "hermes-4" → True."""
    assert _reasoning_supported("nous-hermes-4") is True


def test_reasoning_supported_hermes4_uppercase():
    """Case-insensitive match — uppercase "HERMES-4" → True."""
    assert _reasoning_supported("NOUSRESEARCH/HERMES-4") is True


def test_reasoning_supported_hermes4_compact():
    """Compact "hermes4" pattern → True."""
    assert _reasoning_supported("hermes4-fine-tuned") is True


def test_reasoning_not_supported_hermes3():
    """Hermes 3 slug → False (pattern "hermes-3" not in _HERMES4_PATTERNS)."""
    assert _reasoning_supported("nousresearch/hermes-3-llama-3.1-70b") is False


def test_reasoning_not_supported_gpt4():
    """Unrelated model → False."""
    assert _reasoning_supported("gpt-4o") is False


def test_reasoning_not_supported_empty():
    """Empty string → False."""
    assert _reasoning_supported("") is False


# ---------------------------------------------------------------------------
# ProviderConfig
# ---------------------------------------------------------------------------


def test_provider_config_hermes4():
    """Hermes 4 model → reasoning_supported=True."""
    cfg = ProviderConfig("nousresearch/hermes-4-0")
    assert cfg.model == "nousresearch/hermes-4-0"
    assert cfg.reasoning_supported is True


def test_provider_config_hermes3():
    """Hermes 3 model → reasoning_supported=False."""
    cfg = ProviderConfig("nousresearch/hermes-3-llama-3.1-70b")
    assert cfg.reasoning_supported is False


def test_provider_config_unknown():
    """Unknown model → reasoning_supported=False."""
    cfg = ProviderConfig("mistralai/mixtral-8x7b")
    assert cfg.reasoning_supported is False


# ---------------------------------------------------------------------------
# HermesA2AExecutor construction
# ---------------------------------------------------------------------------


def test_constructor_fields_stored():
    """All constructor fields are persisted as attributes."""
    mock_client = MagicMock()
    executor = HermesA2AExecutor(
        model="nousresearch/hermes-4-0",
        system_prompt="sys",
        _client=mock_client,
    )
    assert executor.model == "nousresearch/hermes-4-0"
    assert executor.system_prompt == "sys"
    assert executor._client is mock_client
    assert isinstance(executor._provider, ProviderConfig)
    assert executor._provider.reasoning_supported is True


def test_constructor_hermes3_reasoning_not_enabled():
    """Hermes 3 model → _provider.reasoning_supported is False."""
    executor = HermesA2AExecutor(
        model="nousresearch/hermes-3-llama-3.1-70b",
        _client=MagicMock(),
    )
    assert executor._provider.reasoning_supported is False


def test_constructor_uses_injected_client():
    """When _client is supplied, AsyncOpenAI is never called."""
    stub = MagicMock()
    executor = HermesA2AExecutor(model="hermes-4", _client=stub)
    assert executor._client is stub


# ---------------------------------------------------------------------------
# _build_messages
# ---------------------------------------------------------------------------


def test_build_messages_with_system_prompt():
    """System prompt is prepended as role=system."""
    executor = HermesA2AExecutor(
        model="hermes-4", system_prompt="Be helpful.", _client=MagicMock()
    )
    msgs = executor._build_messages("Hello!")
    assert msgs[0] == {"role": "system", "content": "Be helpful."}
    assert msgs[1] == {"role": "user", "content": "Hello!"}


def test_build_messages_no_system_prompt():
    """Without system_prompt only the user turn is present."""
    executor = HermesA2AExecutor(
        model="hermes-4", system_prompt=None, _client=MagicMock()
    )
    msgs = executor._build_messages("Hello!")
    assert len(msgs) == 1
    assert msgs[0] == {"role": "user", "content": "Hello!"}


# ---------------------------------------------------------------------------
# execute — happy path
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_execute_returns_content():
    """Successful API call → content is enqueued as A2A message."""
    executor, mock_client = _make_executor()
    mock_client.chat.completions.create.return_value = _make_api_response("42")

    ctx = _make_context("What is 6×7?")
    eq = AsyncMock()

    await executor.execute(ctx, eq)

    eq.enqueue_event.assert_called_once_with("42")


@pytest.mark.asyncio
async def test_execute_empty_content_returns_fallback():
    """Empty content string → fallback message '(no response generated)'."""
    executor, mock_client = _make_executor()
    mock_client.chat.completions.create.return_value = _make_api_response("")

    ctx = _make_context("ping")
    eq = AsyncMock()

    await executor.execute(ctx, eq)

    eq.enqueue_event.assert_called_once_with("(no response generated)")


@pytest.mark.asyncio
async def test_execute_strips_whitespace_content():
    """Content with only whitespace is treated as empty → fallback."""
    executor, mock_client = _make_executor()
    mock_client.chat.completions.create.return_value = _make_api_response("   \n  ")

    ctx = _make_context("ping")
    eq = AsyncMock()

    await executor.execute(ctx, eq)

    eq.enqueue_event.assert_called_once_with("(no response generated)")


# ---------------------------------------------------------------------------
# execute — empty input
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_execute_empty_input_returns_error():
    """Message with no extractable text → error message, no API call."""
    executor, mock_client = _make_executor()

    ctx = _make_empty_context()
    eq = AsyncMock()

    await executor.execute(ctx, eq)

    eq.enqueue_event.assert_called_once_with(
        "Error: message contained no text content."
    )
    mock_client.chat.completions.create.assert_not_called()


# ---------------------------------------------------------------------------
# execute — Hermes 4 extra_body
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_execute_hermes4_sends_reasoning_extra_body():
    """Hermes 4 model → extra_body with reasoning enabled is sent."""
    executor, mock_client = _make_executor(model="nousresearch/hermes-4-0")
    mock_client.chat.completions.create.return_value = _make_api_response("ok")

    await executor.execute(_make_context("hello"), AsyncMock())

    call_kwargs = mock_client.chat.completions.create.call_args[1]
    assert call_kwargs["extra_body"] == {"reasoning": {"enabled": True}}


@pytest.mark.asyncio
async def test_execute_hermes3_no_extra_body():
    """Hermes 3 model → extra_body=None, no reasoning injection."""
    executor, mock_client = _make_executor(model="nousresearch/hermes-3-llama-3.1-70b")
    mock_client.chat.completions.create.return_value = _make_api_response("ok")

    await executor.execute(_make_context("hello"), AsyncMock())

    call_kwargs = mock_client.chat.completions.create.call_args[1]
    assert call_kwargs["extra_body"] is None


@pytest.mark.asyncio
async def test_execute_model_passed_to_api():
    """The model name is forwarded verbatim to the API call."""
    model = "nousresearch/hermes-4-0"
    executor, mock_client = _make_executor(model=model)
    mock_client.chat.completions.create.return_value = _make_api_response("ok")

    await executor.execute(_make_context("hi"), AsyncMock())

    call_kwargs = mock_client.chat.completions.create.call_args[1]
    assert call_kwargs["model"] == model


# ---------------------------------------------------------------------------
# execute — reasoning trace handling
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_execute_reasoning_not_in_reply():
    """Reasoning trace is present in response but NOT included in the A2A reply."""
    executor, mock_client = _make_executor(model="nousresearch/hermes-4-0")
    response = _make_api_response(
        content="The answer is 42.",
        reasoning="<think>First I compute 6×7...</think>",
    )
    mock_client.chat.completions.create.return_value = response

    eq = AsyncMock()
    await executor.execute(_make_context("6×7?"), eq)

    # Reply must contain ONLY the content, not the reasoning
    enqueued = eq.enqueue_event.call_args[0][0]
    assert enqueued == "The answer is 42."
    assert "<think>" not in enqueued
    assert "6×7" not in enqueued  # reasoning text excluded


@pytest.mark.asyncio
async def test_execute_reasoning_logged_via_otel(monkeypatch):
    """Reasoning trace → _log_reasoning is called."""
    executor, mock_client = _make_executor(model="nousresearch/hermes-4-0")
    response = _make_api_response(
        content="Answer.",
        reasoning="<think>reasoning here</think>",
    )
    mock_client.chat.completions.create.return_value = response

    log_calls: list = []

    original_log = executor._log_reasoning

    def capturing_log(context, reasoning, reasoning_details):
        log_calls.append((reasoning, reasoning_details))
        return original_log(context, reasoning, reasoning_details)

    monkeypatch.setattr(executor, "_log_reasoning", capturing_log)

    await executor.execute(_make_context("test"), AsyncMock())

    assert len(log_calls) == 1
    assert log_calls[0][0] == "<think>reasoning here</think>"


@pytest.mark.asyncio
async def test_execute_reasoning_details_logged(monkeypatch):
    """reasoning_details field is passed through to _log_reasoning."""
    executor, mock_client = _make_executor(model="hermes-4")
    details = {"steps": ["step1", "step2"]}
    response = _make_api_response(
        content="ok",
        reasoning="some reasoning",
        reasoning_details=details,
    )
    mock_client.chat.completions.create.return_value = response

    log_calls: list = []

    def capturing_log(context, reasoning, reasoning_details):
        log_calls.append((reasoning, reasoning_details))

    monkeypatch.setattr(executor, "_log_reasoning", capturing_log)

    await executor.execute(_make_context("test"), AsyncMock())

    assert log_calls[0][1] is details


@pytest.mark.asyncio
async def test_execute_no_reasoning_field_no_log(monkeypatch):
    """Response with no reasoning attribute → _log_reasoning not called."""
    executor, mock_client = _make_executor(model="nousresearch/hermes-4-0")
    # _make_api_response with no reasoning arg → no .reasoning attribute set
    response = _make_api_response(content="ok")
    mock_client.chat.completions.create.return_value = response

    log_calls: list = []
    monkeypatch.setattr(executor, "_log_reasoning", lambda *a: log_calls.append(a))

    await executor.execute(_make_context("test"), AsyncMock())

    assert log_calls == []


# ---------------------------------------------------------------------------
# execute — API error handling
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_execute_api_error_returns_sanitized_message():
    """API exception → class name only in the A2A reply (no message body)."""
    executor, mock_client = _make_executor()

    class FakeAPIError(Exception):
        pass

    mock_client.chat.completions.create.side_effect = FakeAPIError(
        "api_key=sk-secret123 rate limit exceeded"
    )

    eq = AsyncMock()
    await executor.execute(_make_context("hello"), eq)

    enqueued = eq.enqueue_event.call_args[0][0]
    assert enqueued == "Agent error: FakeAPIError"
    # Secret must NOT leak
    assert "sk-secret" not in enqueued
    assert "rate limit" not in enqueued


@pytest.mark.asyncio
async def test_execute_api_error_is_logged(caplog):
    """API exception is logged at ERROR level."""
    import logging

    executor, mock_client = _make_executor()
    mock_client.chat.completions.create.side_effect = ValueError("bad request")

    with caplog.at_level(logging.ERROR, logger="hermes_executor"):
        await executor.execute(_make_context("hello"), AsyncMock())

    assert any("API error" in r.message for r in caplog.records)


# ---------------------------------------------------------------------------
# _log_reasoning — direct unit tests
# ---------------------------------------------------------------------------


def test_log_reasoning_otel_span_attributes():
    """_log_reasoning sets the expected OTEL span attributes."""
    executor, _ = _make_executor(model="nousresearch/hermes-4-0")

    mock_span = MagicMock()
    mock_tracer = MagicMock()
    mock_tracer.start_as_current_span.return_value.__enter__ = MagicMock(
        return_value=mock_span
    )
    mock_tracer.start_as_current_span.return_value.__exit__ = MagicMock(
        return_value=False
    )

    ctx = MagicMock()
    ctx.context_id = "ctx-abc"

    with patch("hermes_executor.os.environ.get", return_value="ws-123"), \
         patch("hermes_executor.logger"):
        # Patch builtin_tools.telemetry inside the method
        import builtin_tools.telemetry as _tel
        original_get_tracer = _tel.get_tracer
        _tel.get_tracer = MagicMock(return_value=mock_tracer)
        try:
            executor._log_reasoning(ctx, "deep thinking here", None)
        finally:
            _tel.get_tracer = original_get_tracer

    mock_span.set_attribute.assert_any_call("hermes.model", "nousresearch/hermes-4-0")
    mock_span.set_attribute.assert_any_call("hermes.reasoning_length", len("deep thinking here"))
    mock_span.set_attribute.assert_any_call("hermes.reasoning_preview", "deep thinking here")


def test_log_reasoning_swallows_telemetry_error(caplog):
    """_log_reasoning never raises even when OTEL throws."""
    import logging

    executor, _ = _make_executor()
    ctx = MagicMock()
    ctx.context_id = "ctx-xyz"

    with patch("builtin_tools.telemetry.get_tracer", side_effect=RuntimeError("boom")):
        # Must not raise
        executor._log_reasoning(ctx, "reasoning text", None)


def test_log_reasoning_has_reasoning_details_attribute():
    """reasoning_details → has_reasoning_details span attribute set to True."""
    executor, _ = _make_executor(model="hermes-4")

    mock_span = MagicMock()
    mock_tracer = MagicMock()
    mock_tracer.start_as_current_span.return_value.__enter__ = MagicMock(
        return_value=mock_span
    )
    mock_tracer.start_as_current_span.return_value.__exit__ = MagicMock(
        return_value=False
    )

    ctx = MagicMock()
    ctx.context_id = "ctx-rd"

    import builtin_tools.telemetry as _tel
    original = _tel.get_tracer
    _tel.get_tracer = MagicMock(return_value=mock_tracer)
    try:
        executor._log_reasoning(ctx, None, {"steps": []})
    finally:
        _tel.get_tracer = original

    mock_span.set_attribute.assert_any_call("hermes.has_reasoning_details", True)


def test_log_reasoning_no_preview_when_reasoning_is_none():
    """When reasoning is None, hermes.reasoning_preview attribute is not set."""
    executor, _ = _make_executor(model="hermes-4")

    mock_span = MagicMock()
    mock_tracer = MagicMock()
    mock_tracer.start_as_current_span.return_value.__enter__ = MagicMock(
        return_value=mock_span
    )
    mock_tracer.start_as_current_span.return_value.__exit__ = MagicMock(
        return_value=False
    )

    ctx = MagicMock()
    ctx.context_id = "ctx-none"

    import builtin_tools.telemetry as _tel
    original = _tel.get_tracer
    _tel.get_tracer = MagicMock(return_value=mock_tracer)
    try:
        executor._log_reasoning(ctx, None, None)
    finally:
        _tel.get_tracer = original

    # hermes.reasoning_preview should NOT have been set
    preview_calls = [
        c for c in mock_span.set_attribute.call_args_list
        if c[0][0] == "hermes.reasoning_preview"
    ]
    assert preview_calls == []


# ---------------------------------------------------------------------------
# cancel
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_cancel_emits_canceled_event():
    """cancel() enqueues a TaskStatusUpdateEvent with state=canceled."""
    executor, _ = _make_executor()

    # Stub a2a.types if not already present with minimal TaskStatusUpdateEvent
    import a2a.types as a2a_types

    class _TaskState:
        canceled = "canceled"

    class _TaskStatus:
        def __init__(self, state):
            self.state = state

    class _TaskStatusUpdateEvent:
        def __init__(self, status, final):
            self.status = status
            self.final = final

    a2a_types.TaskState = _TaskState
    a2a_types.TaskStatus = _TaskStatus
    a2a_types.TaskStatusUpdateEvent = _TaskStatusUpdateEvent

    eq = AsyncMock()
    ctx = MagicMock()
    await executor.cancel(ctx, eq)

    eq.enqueue_event.assert_called_once()
    event = eq.enqueue_event.call_args[0][0]
    assert isinstance(event, _TaskStatusUpdateEvent)
    assert event.status.state == "canceled"
    assert event.final is True


# ---------------------------------------------------------------------------
# Integration: system prompt is sent with messages
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_system_prompt_included_in_api_call():
    """System prompt appears as first message in the API call."""
    executor, mock_client = _make_executor(
        model="hermes-4", system_prompt="You are a math tutor."
    )
    mock_client.chat.completions.create.return_value = _make_api_response("6")

    await executor.execute(_make_context("3+3?"), AsyncMock())

    msgs = mock_client.chat.completions.create.call_args[1]["messages"]
    assert msgs[0] == {"role": "system", "content": "You are a math tutor."}
    assert msgs[1]["role"] == "user"
    assert "3+3?" in msgs[1]["content"]


@pytest.mark.asyncio
async def test_no_system_prompt_only_user_message():
    """Without system_prompt, only the user turn is in messages."""
    executor, mock_client = _make_executor(model="hermes-4", system_prompt=None)
    mock_client.chat.completions.create.return_value = _make_api_response("ok")

    await executor.execute(_make_context("hello"), AsyncMock())

    msgs = mock_client.chat.completions.create.call_args[1]["messages"]
    assert len(msgs) == 1
    assert msgs[0]["role"] == "user"


# ---------------------------------------------------------------------------
# Native tools parameter — issue #497
# ---------------------------------------------------------------------------

# Minimal OpenAI-format tool definition used across the tools tests.
_SAMPLE_TOOL: dict = {
    "type": "function",
    "function": {
        "name": "get_weather",
        "description": "Get current weather for a location.",
        "parameters": {
            "type": "object",
            "properties": {
                "location": {"type": "string", "description": "City name"},
            },
            "required": ["location"],
        },
    },
}

_SAMPLE_TOOL_2: dict = {
    "type": "function",
    "function": {
        "name": "search_web",
        "description": "Search the web.",
        "parameters": {
            "type": "object",
            "properties": {"query": {"type": "string"}},
            "required": ["query"],
        },
    },
}


class _FakeFunction:
    """Stand-in for openai ChatCompletionMessageToolCall.function."""

    def __init__(self, name: str, arguments: str) -> None:
        self.name = name
        self.arguments = arguments


class _FakeToolCall:
    """Stand-in for openai ChatCompletionMessageToolCall."""

    def __init__(self, tc_id: str, name: str, arguments: str = "{}") -> None:
        self.id = tc_id
        self.type = "function"
        self.function = _FakeFunction(name=name, arguments=arguments)


def _make_tool_call_response(tool_calls: list, content: str = ""):
    """Build a mock API response that includes tool_calls on the message."""

    class _MsgWithToolCalls:
        def __init__(self):
            self.content = content
            self.tool_calls = tool_calls

    choice = MagicMock()
    choice.message = _MsgWithToolCalls()
    response = MagicMock()
    response.choices = [choice]
    return response


def test_constructor_tools_stored_correctly():
    """tools list is stored as _tools attribute."""
    executor = HermesA2AExecutor(
        model="hermes-4",
        tools=[_SAMPLE_TOOL, _SAMPLE_TOOL_2],
        _client=MagicMock(),
    )
    assert executor._tools == [_SAMPLE_TOOL, _SAMPLE_TOOL_2]


def test_constructor_none_tools_stored_as_empty_list():
    """tools=None → _tools is [] (empty list, not None)."""
    executor = HermesA2AExecutor(model="hermes-4", tools=None, _client=MagicMock())
    assert executor._tools == []


def test_constructor_empty_list_stored_as_empty_list():
    """tools=[] → _tools is []."""
    executor = HermesA2AExecutor(model="hermes-4", tools=[], _client=MagicMock())
    assert executor._tools == []


def test_constructor_tools_is_independent_copy():
    """_tools is a copy — mutating the input list doesn't affect the executor."""
    original = [_SAMPLE_TOOL]
    executor = HermesA2AExecutor(
        model="hermes-4", tools=original, _client=MagicMock()
    )
    original.append(_SAMPLE_TOOL_2)
    assert executor._tools == [_SAMPLE_TOOL]


@pytest.mark.asyncio
async def test_execute_tools_serialized_in_request_body():
    """Non-empty tools list is forwarded to chat.completions.create as tools=."""
    mock_client = MagicMock()
    mock_client.chat.completions.create = AsyncMock(
        return_value=_make_api_response("Paris is sunny.")
    )
    executor = HermesA2AExecutor(
        model="hermes-4",
        tools=[_SAMPLE_TOOL],
        _client=mock_client,
    )

    await executor.execute(_make_context("weather?"), AsyncMock())

    call_kwargs = mock_client.chat.completions.create.call_args[1]
    assert "tools" in call_kwargs
    assert call_kwargs["tools"] == [_SAMPLE_TOOL]


@pytest.mark.asyncio
async def test_execute_multiple_tools_all_forwarded():
    """All tool definitions are forwarded — not truncated."""
    mock_client = MagicMock()
    mock_client.chat.completions.create = AsyncMock(
        return_value=_make_api_response("ok")
    )
    executor = HermesA2AExecutor(
        model="hermes-4",
        tools=[_SAMPLE_TOOL, _SAMPLE_TOOL_2],
        _client=mock_client,
    )

    await executor.execute(_make_context("search?"), AsyncMock())

    call_kwargs = mock_client.chat.completions.create.call_args[1]
    assert call_kwargs["tools"] == [_SAMPLE_TOOL, _SAMPLE_TOOL_2]


@pytest.mark.asyncio
async def test_execute_empty_tools_no_tools_field_in_request():
    """Empty tools list → 'tools' key absent from API call (not tools=[])."""
    mock_client = MagicMock()
    mock_client.chat.completions.create = AsyncMock(
        return_value=_make_api_response("ok")
    )
    executor = HermesA2AExecutor(model="hermes-4", tools=[], _client=mock_client)

    await executor.execute(_make_context("hello"), AsyncMock())

    call_kwargs = mock_client.chat.completions.create.call_args[1]
    assert "tools" not in call_kwargs


@pytest.mark.asyncio
async def test_execute_none_tools_no_tools_field_in_request():
    """tools=None → 'tools' key absent from API call."""
    mock_client = MagicMock()
    mock_client.chat.completions.create = AsyncMock(
        return_value=_make_api_response("ok")
    )
    executor = HermesA2AExecutor(model="hermes-4", tools=None, _client=mock_client)

    await executor.execute(_make_context("hello"), AsyncMock())

    call_kwargs = mock_client.chat.completions.create.call_args[1]
    assert "tools" not in call_kwargs


@pytest.mark.asyncio
async def test_execute_default_no_tools_field_in_request():
    """Constructor with no tools kwarg → 'tools' key absent from API call."""
    executor, mock_client = _make_executor(model="hermes-4")
    mock_client.chat.completions.create.return_value = _make_api_response("ok")

    await executor.execute(_make_context("hello"), AsyncMock())

    call_kwargs = mock_client.chat.completions.create.call_args[1]
    assert "tools" not in call_kwargs


@pytest.mark.asyncio
async def test_execute_tool_call_response_returns_json():
    """Model returns tool_calls with no content → reply is JSON-serialised calls."""
    import json

    mock_client = MagicMock()
    tc = _FakeToolCall("call_abc123", "get_weather", '{"location":"Paris"}')
    mock_client.chat.completions.create = AsyncMock(
        return_value=_make_tool_call_response(tool_calls=[tc], content="")
    )
    executor = HermesA2AExecutor(
        model="hermes-4",
        tools=[_SAMPLE_TOOL],
        _client=mock_client,
    )

    eq = AsyncMock()
    await executor.execute(_make_context("weather in Paris?"), eq)

    eq.enqueue_event.assert_called_once()
    reply = eq.enqueue_event.call_args[0][0]
    # Must be valid JSON
    parsed = json.loads(reply)
    assert isinstance(parsed, list)
    assert len(parsed) == 1
    assert parsed[0]["function"]["name"] == "get_weather"
    assert parsed[0]["function"]["arguments"] == '{"location":"Paris"}'
    assert parsed[0]["id"] == "call_abc123"
    assert parsed[0]["type"] == "function"


@pytest.mark.asyncio
async def test_execute_multiple_tool_calls_all_in_json():
    """Multiple tool calls are all serialised into the JSON reply."""
    import json

    mock_client = MagicMock()
    tc1 = _FakeToolCall("call_1", "get_weather", '{"location":"Paris"}')
    tc2 = _FakeToolCall("call_2", "search_web", '{"query":"news"}')
    mock_client.chat.completions.create = AsyncMock(
        return_value=_make_tool_call_response(tool_calls=[tc1, tc2], content="")
    )
    executor = HermesA2AExecutor(
        model="hermes-4",
        tools=[_SAMPLE_TOOL, _SAMPLE_TOOL_2],
        _client=mock_client,
    )

    eq = AsyncMock()
    await executor.execute(_make_context("do both"), eq)

    reply = eq.enqueue_event.call_args[0][0]
    parsed = json.loads(reply)
    assert len(parsed) == 2
    assert parsed[0]["function"]["name"] == "get_weather"
    assert parsed[1]["function"]["name"] == "search_web"


@pytest.mark.asyncio
async def test_execute_text_content_wins_over_tool_calls():
    """When model returns both text content AND tool_calls, text is used."""
    mock_client = MagicMock()
    tc = _FakeToolCall("call_xyz", "get_weather", '{"location":"Berlin"}')
    mock_client.chat.completions.create = AsyncMock(
        return_value=_make_tool_call_response(
            tool_calls=[tc], content="The weather is fine."
        )
    )
    executor = HermesA2AExecutor(
        model="hermes-4",
        tools=[_SAMPLE_TOOL],
        _client=mock_client,
    )

    eq = AsyncMock()
    await executor.execute(_make_context("weather?"), eq)

    reply = eq.enqueue_event.call_args[0][0]
    assert reply == "The weather is fine."
