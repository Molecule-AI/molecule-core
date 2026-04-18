"""Unit tests for adapters/google-adk/adapter.py.

Coverage targets (100%)
-----------------------
- Module constants: _DEFAULT_AGENT_NAME, _DEFAULT_MAX_OUTPUT_TOKENS, etc.
- MissingContent sentinel class
- GoogleADKA2AExecutor.__init__    — field assignment + runner injection
- GoogleADKA2AExecutor._extract_text
- GoogleADKA2AExecutor._build_content
- GoogleADKA2AExecutor._ensure_session — first call (create), subsequent call (skip)
- GoogleADKA2AExecutor.execute     — happy path, empty input, API error,
                                     no final_response events, partial text
- GoogleADKA2AExecutor.cancel      — TaskStatusUpdateEvent emitted
- GoogleADKAdapter.name / display_name / description / get_config_schema
- GoogleADKAdapter.setup           — success, missing key, vertex override
- GoogleADKAdapter.create_executor — model stripping, defaults, rc overrides
- Adapter alias

All google-adk, google-genai, and shared_runtime calls are mocked.
No live API calls are made.
"""
from __future__ import annotations

import sys
from types import ModuleType
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

# ---------------------------------------------------------------------------
# Stub heavy external modules BEFORE the adapter is imported.
# conftest.py already stubs: a2a, builtin_tools, langchain_core.
# We need to additionally stub: google.adk, google.genai, shared_runtime.
# ---------------------------------------------------------------------------


def _make_a2a_stubs() -> None:
    """Register minimal a2a SDK stubs in sys.modules.

    Mirrors what workspace/tests/conftest.py does; needed because
    this test file lives outside the ``tests/`` directory and conftest.py
    is not automatically loaded for it.
    """
    if "a2a" in sys.modules:
        # Already mocked by conftest — just ensure new_agent_text_message is passthrough
        a2a_utils = sys.modules.get("a2a.utils")
        if a2a_utils and callable(getattr(a2a_utils, "new_agent_text_message", None)):
            a2a_utils.new_agent_text_message = lambda text, **kwargs: text
        return

    agent_execution_mod = ModuleType("a2a.server.agent_execution")

    class AgentExecutor:
        pass

    class RequestContext:
        pass

    agent_execution_mod.AgentExecutor = AgentExecutor
    agent_execution_mod.RequestContext = RequestContext

    events_mod = ModuleType("a2a.server.events")

    class EventQueue:
        pass

    events_mod.EventQueue = EventQueue

    tasks_mod = ModuleType("a2a.server.tasks")
    types_mod = ModuleType("a2a.types")

    class TextPart:
        def __init__(self, text=""):
            self.text = text

    class Part:
        def __init__(self, root=None):
            self.root = root

    types_mod.TextPart = TextPart
    types_mod.Part = Part

    utils_mod = ModuleType("a2a.utils")
    # Passthrough so tests can assert on the plain text string, matching the
    # hermes_executor test convention from conftest.py.
    utils_mod.new_agent_text_message = lambda text, **kwargs: text

    a2a_mod = ModuleType("a2a")
    a2a_server_mod = ModuleType("a2a.server")

    sys.modules["a2a"] = a2a_mod
    sys.modules["a2a.server"] = a2a_server_mod
    sys.modules["a2a.server.agent_execution"] = agent_execution_mod
    sys.modules["a2a.server.events"] = events_mod
    sys.modules["a2a.server.tasks"] = tasks_mod
    sys.modules["a2a.types"] = types_mod
    sys.modules["a2a.utils"] = utils_mod


def _make_google_adk_stubs() -> None:
    """Register minimal google.adk and google.genai stubs in sys.modules."""
    # google (top-level namespace package)
    google_mod = sys.modules.get("google") or ModuleType("google")
    google_mod.__path__ = []
    sys.modules.setdefault("google", google_mod)

    # google.genai
    google_genai_mod = ModuleType("google.genai")
    google_genai_mod.__path__ = []

    google_genai_types_mod = ModuleType("google.genai.types")

    class _Content:
        def __init__(self, role="user", parts=None):
            self.role = role
            self.parts = parts or []

    class _Part:
        def __init__(self, text=""):
            self.text = text

    google_genai_types_mod.Content = _Content
    google_genai_types_mod.Part = _Part

    sys.modules["google.genai"] = google_genai_mod
    sys.modules["google.genai.types"] = google_genai_types_mod

    # google.adk
    google_adk_mod = ModuleType("google.adk")
    google_adk_mod.__path__ = []

    # google.adk.agents
    google_adk_agents_mod = ModuleType("google.adk.agents")

    class _LlmAgent:
        def __init__(self, name="", model="", instruction="", tools=None):
            self.name = name
            self.model = model
            self.instruction = instruction
            self.tools = tools or []

    google_adk_agents_mod.LlmAgent = _LlmAgent

    # google.adk.runners
    google_adk_runners_mod = ModuleType("google.adk.runners")

    class _Runner:
        def __init__(self, agent=None, app_name="", session_service=None):
            self.agent = agent
            self.app_name = app_name
            self.session_service = session_service

        async def run_async(self, session_id, user_id, new_message):
            # Stub — tests override this via mock runner
            return
            yield  # make it an async generator

    google_adk_runners_mod.Runner = _Runner

    # google.adk.sessions
    google_adk_sessions_mod = ModuleType("google.adk.sessions")

    class _InMemorySessionService:
        def __init__(self):
            self._sessions: dict = {}

        async def get_session(self, app_name, user_id, session_id):
            return self._sessions.get((app_name, user_id, session_id))

        async def create_session(self, app_name, user_id, session_id):
            self._sessions[(app_name, user_id, session_id)] = {"id": session_id}
            return self._sessions[(app_name, user_id, session_id)]

    google_adk_sessions_mod.InMemorySessionService = _InMemorySessionService

    sys.modules["google.adk"] = google_adk_mod
    sys.modules["google.adk.agents"] = google_adk_agents_mod
    sys.modules["google.adk.runners"] = google_adk_runners_mod
    sys.modules["google.adk.sessions"] = google_adk_sessions_mod


def _make_shared_runtime_stub() -> None:
    """Register shared_runtime stub with extract_message_text."""
    if "shared_runtime" not in sys.modules:
        mod = ModuleType("shared_runtime")

        def _extract_message_text(ctx) -> str:
            parts = getattr(getattr(ctx, "message", None), "parts", None)
            if parts is None:
                parts = ctx
            texts = []
            for p in parts or []:
                t = getattr(p, "text", None) or getattr(
                    getattr(p, "root", None), "text", None
                ) or ""
                if t:
                    texts.append(t)
            return " ".join(texts).strip()

        mod.extract_message_text = _extract_message_text
        sys.modules["shared_runtime"] = mod


def _make_adapter_base_stub() -> None:
    """Register adapter_base stub in sys.modules."""
    if "adapter_base" not in sys.modules:
        mod = ModuleType("adapter_base")
        from dataclasses import dataclass, field
        from abc import ABC, abstractmethod

        @dataclass
        class AdapterConfig:
            model: str = "google:gemini-2.0-flash"
            system_prompt: str | None = None
            tools: list = field(default_factory=list)
            runtime_config: dict = field(default_factory=dict)
            config_path: str = "/configs"
            workspace_id: str = ""
            prompt_files: list = field(default_factory=list)
            a2a_port: int = 8000
            heartbeat: object = None

        class BaseAdapter(ABC):
            @staticmethod
            @abstractmethod
            def name() -> str: ...  # pragma: no cover

            @staticmethod
            @abstractmethod
            def display_name() -> str: ...  # pragma: no cover

            @staticmethod
            @abstractmethod
            def description() -> str: ...  # pragma: no cover

            @staticmethod
            def get_config_schema() -> dict:
                return {}

            def memory_filename(self) -> str:
                return "CLAUDE.md"

            def register_tool_hook(self, name, fn): return None  # noqa

            async def transcript_lines(self, since=0, limit=100): return {"supported": False}  # noqa

            def register_subagent_hook(self, name, spec): return None  # noqa

            def append_to_memory_hook(self, config, filename, content): pass  # noqa

            async def install_plugins_via_registry(self, config, plugins): return []  # noqa

            async def inject_plugins(self, config, plugins):
                await self.install_plugins_via_registry(config, plugins)

            async def _common_setup(self, config):
                from types import SimpleNamespace
                return SimpleNamespace(
                    system_prompt="mocked system prompt",
                    loaded_skills=[],
                    langchain_tools=[],
                    is_coordinator=False,
                    children=[],
                )

            @abstractmethod
            async def setup(self, config) -> None: ...  # pragma: no cover

            @abstractmethod
            async def create_executor(self, config): ...  # pragma: no cover

        mod.AdapterConfig = AdapterConfig
        mod.BaseAdapter = BaseAdapter
        mod.SetupResult = None
        sys.modules["adapter_base"] = mod


# Install all stubs before importing the module under test
# Order matters: a2a must be stubbed before adapter.py is imported so that
# `from a2a.utils import new_agent_text_message` resolves to the passthrough.
_make_a2a_stubs()
_make_google_adk_stubs()
_make_shared_runtime_stub()
_make_adapter_base_stub()

# Now safe to import the adapter
import sys as _sys
import os as _os
_adapter_dir = _os.path.dirname(_os.path.abspath(__file__))
if _adapter_dir not in _sys.path:
    _sys.path.insert(0, _adapter_dir)

from adapter import (  # noqa: E402
    Adapter,
    GoogleADKA2AExecutor,
    GoogleADKAdapter,
    MissingContent,
    _DEFAULT_AGENT_NAME,
    _DEFAULT_MAX_OUTPUT_TOKENS,
    _DEFAULT_TEMPERATURE,
    _NO_RESPONSE_MSG,
    _NO_TEXT_MSG,
)


# ---------------------------------------------------------------------------
# Fixtures and helpers
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
    part = MagicMock(spec=[])
    part.root = MagicMock(spec=[])
    ctx = MagicMock()
    ctx.message.parts = [part]
    ctx.context_id = "ctx-empty"
    return ctx


def _make_event(is_final: bool, text: str | None = None) -> MagicMock:
    """Build a mock ADK Event that optionally is a final response."""
    event = MagicMock()
    event.is_final_response = MagicMock(return_value=is_final)
    if text is not None:
        part = MagicMock()
        part.text = text
        event.response = MagicMock()
        event.response.content = MagicMock()
        event.response.content.parts = [part]
    else:
        event.response = None
    return event


async def _async_gen(*events):
    """Yield events one by one as an async generator."""
    for e in events:
        yield e


def _make_runner(events=None) -> MagicMock:
    """Return a mock Runner whose run_async yields the given events."""
    runner = MagicMock()
    runner.session_service = AsyncMock()
    runner.session_service.get_session = AsyncMock(return_value=None)
    runner.session_service.create_session = AsyncMock(return_value={"id": "s1"})
    evts = events or []
    runner.run_async = MagicMock(return_value=_async_gen(*evts))
    return runner


def _make_executor(
    model: str = "gemini-2.0-flash",
    system_prompt: str | None = "You are helpful.",
    runner: MagicMock | None = None,
) -> GoogleADKA2AExecutor:
    """Create a GoogleADKA2AExecutor with an injected mock runner."""
    return GoogleADKA2AExecutor(
        model=model,
        system_prompt=system_prompt,
        _runner=runner or _make_runner(),
    )


def _make_adapter_config(**kwargs) -> object:
    """Return an AdapterConfig with sensible defaults."""
    from adapter_base import AdapterConfig
    defaults = dict(
        model="google:gemini-2.0-flash",
        system_prompt="Test prompt.",
        runtime_config={},
        workspace_id="ws-test",
    )
    defaults.update(kwargs)
    return AdapterConfig(**defaults)


# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------


def test_default_agent_name():
    assert _DEFAULT_AGENT_NAME == "molecule-adk-agent"


def test_default_max_output_tokens():
    assert _DEFAULT_MAX_OUTPUT_TOKENS == 8192


def test_default_temperature():
    assert _DEFAULT_TEMPERATURE == 1.0


def test_no_text_msg_constant():
    assert "no text" in _NO_TEXT_MSG.lower()


def test_no_response_msg_constant():
    assert "no response" in _NO_RESPONSE_MSG.lower()


# ---------------------------------------------------------------------------
# MissingContent sentinel
# ---------------------------------------------------------------------------


def test_missing_content_has_empty_parts():
    mc = MissingContent()
    assert mc.parts == []


# ---------------------------------------------------------------------------
# GoogleADKA2AExecutor — construction
# ---------------------------------------------------------------------------


def test_constructor_stores_fields():
    runner = _make_runner()
    executor = GoogleADKA2AExecutor(
        model="gemini-1.5-pro",
        system_prompt="Hello",
        agent_name="my-agent",
        max_output_tokens=4096,
        temperature=0.5,
        _runner=runner,
    )
    assert executor.model == "gemini-1.5-pro"
    assert executor.system_prompt == "Hello"
    assert executor.agent_name == "my-agent"
    assert executor.max_output_tokens == 4096
    assert executor.temperature == 0.5
    assert executor._runner is runner
    assert executor._sessions_created == set()


def test_constructor_defaults():
    executor = GoogleADKA2AExecutor(model="gemini-2.0-flash", _runner=_make_runner())
    assert executor.system_prompt is None
    assert executor.agent_name == _DEFAULT_AGENT_NAME
    assert executor.max_output_tokens == _DEFAULT_MAX_OUTPUT_TOKENS
    assert executor.temperature == _DEFAULT_TEMPERATURE
    assert executor._heartbeat is None


def test_constructor_uses_injected_runner():
    stub = MagicMock()
    stub.session_service = MagicMock()
    executor = GoogleADKA2AExecutor(model="gemini-2.0-flash", _runner=stub)
    assert executor._runner is stub


# ---------------------------------------------------------------------------
# GoogleADKA2AExecutor — _extract_text
# ---------------------------------------------------------------------------


def test_extract_text_returns_message_text():
    executor = _make_executor()
    ctx = _make_context("Hello world")
    result = executor._extract_text(ctx)
    assert result == "Hello world"


def test_extract_text_empty_context():
    executor = _make_executor()
    ctx = _make_empty_context()
    result = executor._extract_text(ctx)
    assert result == ""


# ---------------------------------------------------------------------------
# GoogleADKA2AExecutor — _build_content
# ---------------------------------------------------------------------------


def test_build_content_creates_content_object():
    executor = _make_executor()
    content = executor._build_content("test message")
    assert content.role == "user"
    assert len(content.parts) == 1
    assert content.parts[0].text == "test message"


def test_build_content_empty_string():
    executor = _make_executor()
    content = executor._build_content("")
    assert content.parts[0].text == ""


# ---------------------------------------------------------------------------
# GoogleADKA2AExecutor — _ensure_session
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_ensure_session_creates_when_not_exists():
    runner = _make_runner()
    runner.session_service.get_session = AsyncMock(return_value=None)
    executor = GoogleADKA2AExecutor(
        model="gemini-2.0-flash", agent_name="test-agent", _runner=runner
    )
    await executor._ensure_session("session-1", "user-1")
    runner.session_service.create_session.assert_called_once_with(
        app_name="test-agent",
        user_id="user-1",
        session_id="session-1",
    )
    assert "session-1" in executor._sessions_created


@pytest.mark.asyncio
async def test_ensure_session_skips_if_already_tracked():
    runner = _make_runner()
    executor = GoogleADKA2AExecutor(
        model="gemini-2.0-flash", _runner=runner
    )
    executor._sessions_created.add("session-x")
    await executor._ensure_session("session-x", "user-1")
    # Neither get_session nor create_session should be called
    runner.session_service.get_session.assert_not_called()
    runner.session_service.create_session.assert_not_called()


@pytest.mark.asyncio
async def test_ensure_session_skips_create_when_existing():
    runner = _make_runner()
    runner.session_service.get_session = AsyncMock(return_value={"id": "s1"})
    executor = GoogleADKA2AExecutor(
        model="gemini-2.0-flash", agent_name="test-agent", _runner=runner
    )
    await executor._ensure_session("session-existing", "user-1")
    runner.session_service.create_session.assert_not_called()
    assert "session-existing" in executor._sessions_created


# ---------------------------------------------------------------------------
# GoogleADKA2AExecutor — execute: happy path
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_execute_returns_response_text():
    event = _make_event(is_final=True, text="The answer is 42.")
    runner = _make_runner(events=[event])
    executor = _make_executor(runner=runner)

    ctx = _make_context("What is 6×7?")
    eq = AsyncMock()
    await executor.execute(ctx, eq)

    eq.enqueue_event.assert_called_once_with("The answer is 42.")


@pytest.mark.asyncio
async def test_execute_concatenates_multiple_final_parts():
    part1 = MagicMock()
    part1.text = "Hello "
    part2 = MagicMock()
    part2.text = "world"
    event = MagicMock()
    event.is_final_response = MagicMock(return_value=True)
    event.response = MagicMock()
    event.response.content = MagicMock()
    event.response.content.parts = [part1, part2]

    runner = _make_runner(events=[event])
    executor = _make_executor(runner=runner)

    ctx = _make_context("Hi")
    eq = AsyncMock()
    await executor.execute(ctx, eq)

    eq.enqueue_event.assert_called_once_with("Hello world")


@pytest.mark.asyncio
async def test_execute_skips_non_final_events():
    non_final = _make_event(is_final=False, text="intermediate")
    final = _make_event(is_final=True, text="final answer")
    runner = _make_runner(events=[non_final, final])
    executor = _make_executor(runner=runner)

    ctx = _make_context("question")
    eq = AsyncMock()
    await executor.execute(ctx, eq)

    enqueued = eq.enqueue_event.call_args[0][0]
    assert enqueued == "final answer"


@pytest.mark.asyncio
async def test_execute_fallback_when_no_final_response_events():
    non_final = _make_event(is_final=False)
    runner = _make_runner(events=[non_final])
    executor = _make_executor(runner=runner)

    ctx = _make_context("hello")
    eq = AsyncMock()
    await executor.execute(ctx, eq)

    eq.enqueue_event.assert_called_once_with(_NO_RESPONSE_MSG)


@pytest.mark.asyncio
async def test_execute_fallback_when_response_is_none():
    event = MagicMock()
    event.is_final_response = MagicMock(return_value=True)
    event.response = None  # no response object

    runner = _make_runner(events=[event])
    executor = _make_executor(runner=runner)

    ctx = _make_context("ping")
    eq = AsyncMock()
    await executor.execute(ctx, eq)

    eq.enqueue_event.assert_called_once_with(_NO_RESPONSE_MSG)


@pytest.mark.asyncio
async def test_execute_fallback_when_parts_have_no_text():
    part = MagicMock()
    part.text = None  # no text on the part
    event = MagicMock()
    event.is_final_response = MagicMock(return_value=True)
    event.response = MagicMock()
    event.response.content = MagicMock()
    event.response.content.parts = [part]

    runner = _make_runner(events=[event])
    executor = _make_executor(runner=runner)

    ctx = _make_context("ping")
    eq = AsyncMock()
    await executor.execute(ctx, eq)

    eq.enqueue_event.assert_called_once_with(_NO_RESPONSE_MSG)


@pytest.mark.asyncio
async def test_execute_fallback_when_response_content_is_none():
    event = MagicMock()
    event.is_final_response = MagicMock(return_value=True)
    event.response = MagicMock()
    event.response.content = None  # content is None → MissingContent sentinel

    runner = _make_runner(events=[event])
    executor = _make_executor(runner=runner)

    ctx = _make_context("ping")
    eq = AsyncMock()
    await executor.execute(ctx, eq)

    eq.enqueue_event.assert_called_once_with(_NO_RESPONSE_MSG)


@pytest.mark.asyncio
async def test_execute_uses_context_id_as_session_id():
    event = _make_event(is_final=True, text="ok")
    runner = _make_runner(events=[event])
    executor = _make_executor(runner=runner)

    ctx = _make_context("hello", context_id="ctx-abc-123")
    eq = AsyncMock()
    await executor.execute(ctx, eq)

    runner.run_async.assert_called_once()
    call_kwargs = runner.run_async.call_args[1]
    assert call_kwargs["session_id"] == "ctx-abc-123"
    assert call_kwargs["user_id"] == "molecule-user"


@pytest.mark.asyncio
async def test_execute_falls_back_to_default_session_id_when_context_id_is_none():
    event = _make_event(is_final=True, text="ok")
    runner = _make_runner(events=[event])
    executor = _make_executor(runner=runner)

    ctx = _make_context("hello")
    ctx.context_id = None  # override
    eq = AsyncMock()
    await executor.execute(ctx, eq)

    call_kwargs = runner.run_async.call_args[1]
    assert call_kwargs["session_id"] == "default-session"


# ---------------------------------------------------------------------------
# GoogleADKA2AExecutor — execute: empty input
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_execute_empty_input_returns_error():
    runner = _make_runner()
    executor = _make_executor(runner=runner)

    ctx = _make_empty_context()
    eq = AsyncMock()
    await executor.execute(ctx, eq)

    eq.enqueue_event.assert_called_once_with(_NO_TEXT_MSG)
    runner.run_async.assert_not_called()


# ---------------------------------------------------------------------------
# GoogleADKA2AExecutor — execute: error handling
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_execute_api_error_returns_sanitized_message():
    runner = _make_runner()

    class _FakeAPIError(Exception):
        pass

    async def _raise(*args, **kwargs):
        raise _FakeAPIError("api_key=secret token_limit_exceeded")
        yield  # make it an async generator

    runner.run_async = MagicMock(return_value=_raise())
    executor = _make_executor(runner=runner)

    eq = AsyncMock()
    await executor.execute(_make_context("hello"), eq)

    enqueued = eq.enqueue_event.call_args[0][0]
    assert enqueued == "Agent error: _FakeAPIError"
    assert "secret" not in enqueued


@pytest.mark.asyncio
async def test_execute_api_error_is_logged(caplog):
    import logging

    runner = _make_runner()

    async def _raise(*args, **kwargs):
        raise ValueError("bad request")
        yield  # make it an async generator

    runner.run_async = MagicMock(return_value=_raise())
    executor = _make_executor(runner=runner)

    with caplog.at_level(logging.ERROR, logger="adapter"):
        await executor.execute(_make_context("hello"), AsyncMock())

    assert any("execution error" in r.message.lower() for r in caplog.records)


# ---------------------------------------------------------------------------
# GoogleADKA2AExecutor — cancel
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_cancel_emits_canceled_event():
    executor = _make_executor()

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
# GoogleADKAdapter — identity methods
# ---------------------------------------------------------------------------


def test_adapter_name():
    assert GoogleADKAdapter.name() == "google-adk"


def test_adapter_display_name():
    assert "Google ADK" in GoogleADKAdapter.display_name()


def test_adapter_description():
    desc = GoogleADKAdapter.description()
    assert "ADK" in desc or "Google" in desc


def test_adapter_get_config_schema():
    schema = GoogleADKAdapter.get_config_schema()
    assert schema["type"] == "object"
    assert "agent_name" in schema["properties"]
    assert "max_output_tokens" in schema["properties"]
    assert "temperature" in schema["properties"]


# ---------------------------------------------------------------------------
# GoogleADKAdapter — setup
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_setup_succeeds_with_api_key(monkeypatch):
    monkeypatch.setenv("GOOGLE_API_KEY", "fake-api-key")
    monkeypatch.delenv("GOOGLE_GENAI_USE_VERTEXAI", raising=False)

    adapter = GoogleADKAdapter()
    config = _make_adapter_config()

    await adapter.setup(config)

    assert adapter._setup_result is not None
    assert adapter._setup_result.system_prompt == "mocked system prompt"


@pytest.mark.asyncio
async def test_setup_succeeds_with_vertex_ai(monkeypatch):
    monkeypatch.delenv("GOOGLE_API_KEY", raising=False)
    monkeypatch.setenv("GOOGLE_GENAI_USE_VERTEXAI", "1")

    adapter = GoogleADKAdapter()
    config = _make_adapter_config()

    await adapter.setup(config)

    assert adapter._setup_result is not None


@pytest.mark.asyncio
async def test_setup_succeeds_with_vertex_ai_true_string(monkeypatch):
    monkeypatch.delenv("GOOGLE_API_KEY", raising=False)
    monkeypatch.setenv("GOOGLE_GENAI_USE_VERTEXAI", "True")

    adapter = GoogleADKAdapter()
    config = _make_adapter_config()

    await adapter.setup(config)
    assert adapter._setup_result is not None


@pytest.mark.asyncio
async def test_setup_raises_without_credentials(monkeypatch):
    monkeypatch.delenv("GOOGLE_API_KEY", raising=False)
    monkeypatch.delenv("GOOGLE_GENAI_USE_VERTEXAI", raising=False)

    adapter = GoogleADKAdapter()
    config = _make_adapter_config()

    with pytest.raises(RuntimeError, match="GOOGLE_API_KEY"):
        await adapter.setup(config)


# ---------------------------------------------------------------------------
# GoogleADKAdapter — create_executor
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_create_executor_strips_google_prefix(monkeypatch):
    monkeypatch.setenv("GOOGLE_API_KEY", "key")
    adapter = GoogleADKAdapter()
    config = _make_adapter_config(model="google:gemini-2.0-flash")
    await adapter.setup(config)

    executor = await adapter.create_executor(config)
    assert executor.model == "gemini-2.0-flash"


@pytest.mark.asyncio
async def test_create_executor_no_prefix_passthrough(monkeypatch):
    monkeypatch.setenv("GOOGLE_API_KEY", "key")
    adapter = GoogleADKAdapter()
    config = _make_adapter_config(model="gemini-1.5-pro")
    await adapter.setup(config)

    executor = await adapter.create_executor(config)
    assert executor.model == "gemini-1.5-pro"


@pytest.mark.asyncio
async def test_create_executor_uses_setup_system_prompt(monkeypatch):
    monkeypatch.setenv("GOOGLE_API_KEY", "key")
    adapter = GoogleADKAdapter()
    config = _make_adapter_config()
    await adapter.setup(config)

    executor = await adapter.create_executor(config)
    assert executor.system_prompt == "mocked system prompt"


@pytest.mark.asyncio
async def test_create_executor_runtime_config_overrides(monkeypatch):
    monkeypatch.setenv("GOOGLE_API_KEY", "key")
    adapter = GoogleADKAdapter()
    config = _make_adapter_config(
        runtime_config={
            "agent_name": "custom-agent",
            "max_output_tokens": 512,
            "temperature": 0.3,
        }
    )
    await adapter.setup(config)

    executor = await adapter.create_executor(config)
    assert executor.agent_name == "custom-agent"
    assert executor.max_output_tokens == 512
    assert executor.temperature == 0.3


@pytest.mark.asyncio
async def test_create_executor_defaults_without_runtime_config(monkeypatch):
    monkeypatch.setenv("GOOGLE_API_KEY", "key")
    adapter = GoogleADKAdapter()
    config = _make_adapter_config(runtime_config={})
    await adapter.setup(config)

    executor = await adapter.create_executor(config)
    assert executor.agent_name == _DEFAULT_AGENT_NAME
    assert executor.max_output_tokens == _DEFAULT_MAX_OUTPUT_TOKENS
    assert executor.temperature == _DEFAULT_TEMPERATURE


@pytest.mark.asyncio
async def test_create_executor_without_setup_uses_config_system_prompt(monkeypatch):
    """create_executor without prior setup falls back to config.system_prompt."""
    monkeypatch.setenv("GOOGLE_API_KEY", "key")
    adapter = GoogleADKAdapter()
    config = _make_adapter_config(system_prompt="fallback prompt")
    # Intentionally skip setup() — _setup_result remains None

    executor = await adapter.create_executor(config)
    assert executor.system_prompt == "fallback prompt"


@pytest.mark.asyncio
async def test_create_executor_without_setup_no_system_prompt(monkeypatch):
    """create_executor without setup and no system_prompt → empty string."""
    monkeypatch.setenv("GOOGLE_API_KEY", "key")
    adapter = GoogleADKAdapter()
    config = _make_adapter_config(system_prompt=None)
    # Skip setup()

    executor = await adapter.create_executor(config)
    assert executor.system_prompt == ""


@pytest.mark.asyncio
async def test_create_executor_heartbeat_passed(monkeypatch):
    monkeypatch.setenv("GOOGLE_API_KEY", "key")
    adapter = GoogleADKAdapter()
    heartbeat = MagicMock()
    config = _make_adapter_config(heartbeat=heartbeat)
    await adapter.setup(config)

    executor = await adapter.create_executor(config)
    assert executor._heartbeat is heartbeat


# ---------------------------------------------------------------------------
# Adapter alias
# ---------------------------------------------------------------------------


def test_adapter_alias_is_google_adk_adapter():
    assert Adapter is GoogleADKAdapter
