"""OpenAI-compat A2A executor for Hermes models with native reasoning support.

Dispatches to OpenRouter / Nous Portal (or any OpenAI-compatible endpoint)
and enables Hermes 4 native reasoning when the model supports it.

Reasoning (Hermes 4 only)
--------------------------
Hermes 4 is a hybrid-reasoning model trained on ``<think>`` tags.  When
``reasoning_supported`` is True for the active model, this executor appends:

    extra_body={"reasoning": {"enabled": True}}

to the ``chat.completions.create()`` call.  The ``openai`` SDK forwards
``extra_body`` verbatim to the upstream provider, so both OpenRouter and
Nous Portal receive it without needing provider-specific code paths.

On response, ``choices[0].message.reasoning`` and
``choices[0].message.reasoning_details`` are extracted and written to an
OTEL activity span so operators can inspect the thinking trace in Langfuse
/ Jaeger.  The reasoning content is deliberately **not** included in the
A2A reply — doing so would contaminate the agent's next-turn context with
the model's internal scratchpad.

Native tools (#497)
-------------------
Tool definitions are passed via the OpenAI-native ``tools`` parameter instead
of injecting them as text into the system prompt.  Each entry must follow the
standard OpenAI function-calling schema::

    {
        "type": "function",
        "function": {
            "name": "...",
            "description": "...",
            "parameters": {        # JSON Schema object
                "type": "object",
                "properties": {...},
                "required": [...]
            }
        }
    }

**Empty list rule:** when ``tools`` is ``None`` or ``[]``, the ``tools``
parameter is **omitted** from the API call entirely.  Sending ``tools=[]``
to some OpenAI-compat providers causes a 400 / unexpected behaviour; omitting
the key is always safe and signals "no tool use."

**Tool-call response handling:** when the model returns
``choice.message.tool_calls`` with no text content (``finish_reason`` is
``"tool_calls"``), the executor serialises the tool-call list as a JSON string
and enqueues that as the A2A reply.  This keeps the executor thin (single API
call per turn, no ReAct loop) while surfacing function-call intent to the
caller in a structured, parseable format.

Hermes 3 / unknown models
--------------------------
No ``extra_body`` is sent.  The response is processed identically to any
other OpenAI-compat model call.  The Hermes 3 path is exercised by the
existing adapter test suite and must remain unchanged.

response_format / structured output (#498)
------------------------------------------
Pass ``response_format={"type": "json_schema", "json_schema": {...}}`` (or
``{"type": "json_object"}`` / ``{"type": "text"}``) to request structured
output from the upstream provider.  The value is forwarded verbatim as the
``response_format=`` kwarg on ``chat.completions.create()``.

Validation is performed **before** the API call via
``_validate_response_format()``.  If the dict is invalid (unknown type,
missing ``json_schema`` key for ``type="json_schema"``, etc.) the executor
enqueues an error message and returns early without calling the API.

When ``response_format`` is ``None`` (the default) the kwarg is omitted
entirely from the API call so older / strict providers do not receive an
unexpected field.
"""

from __future__ import annotations

import logging
import os
from typing import TYPE_CHECKING, Any

from a2a.server.agent_execution import AgentExecutor, RequestContext
from a2a.server.events import EventQueue
from a2a.utils import new_agent_text_message

if TYPE_CHECKING:
    from heartbeat import HeartbeatLoop

logger = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# Per-model reasoning capability detection
# ---------------------------------------------------------------------------

# Substrings that identify a Hermes 4 model slug from either provider:
#   OpenRouter:  "nousresearch/hermes-4-*", "nousresearch/nous-hermes-4-*"
#   Nous Portal: "hermes-4", "nous-hermes-4"
#
# Hermes 3 slugs ("hermes-3-llama-3.1-70b", etc.) do NOT contain any of
# these patterns, so they correctly resolve to reasoning_supported=False.
_HERMES4_PATTERNS: tuple[str, ...] = (
    "hermes-4",
    "hermes4",
)


def _reasoning_supported(model: str) -> bool:
    """Return True if *model* identifies a Hermes 4 variant.

    Case-insensitive substring match against ``_HERMES4_PATTERNS``.

    >>> _reasoning_supported("nousresearch/hermes-4-0")
    True
    >>> _reasoning_supported("nousresearch/nous-hermes-4")
    True
    >>> _reasoning_supported("nousresearch/hermes-3-llama-3.1-70b")
    False
    >>> _reasoning_supported("gpt-4o")
    False
    """
    model_lower = model.lower()
    return any(pat in model_lower for pat in _HERMES4_PATTERNS)


# ---------------------------------------------------------------------------
# response_format validation (#498)
# ---------------------------------------------------------------------------

_VALID_RESPONSE_FORMAT_TYPES: frozenset[str] = frozenset(
    {"json_schema", "json_object", "text"}
)


def _validate_response_format(rf: dict) -> "str | None":
    """Validate a ``response_format`` dict before forwarding to the API.

    Returns ``None`` if *rf* is valid, or an error message string describing
    the first validation failure found.

    Valid ``type`` values are ``"json_schema"``, ``"json_object"``, and
    ``"text"``.  For ``type="json_schema"``, the dict must also contain a
    ``"json_schema"`` key whose value is a dict with at least a ``"name"``
    key (str).  If ``json_schema.schema`` is present it must be a dict.

    Examples::

        >>> _validate_response_format({"type": "json_object"}) is None
        True
        >>> _validate_response_format({"type": "bad"}) is not None
        True
    """
    rf_type = rf.get("type")
    if rf_type not in _VALID_RESPONSE_FORMAT_TYPES:
        return (
            f"type must be one of {sorted(_VALID_RESPONSE_FORMAT_TYPES)!r}, "
            f"got {rf_type!r}"
        )

    if rf_type == "json_schema":
        js = rf.get("json_schema")
        if not isinstance(js, dict):
            return "json_schema must be a dict when type='json_schema'"
        if not isinstance(js.get("name"), str):
            return "json_schema.name must be a string"
        schema = js.get("schema")
        if schema is not None and not isinstance(schema, dict):
            return "json_schema.schema must be a dict if present"

    return None


# ---------------------------------------------------------------------------
# ProviderConfig — per-provider / per-model capability flags
# ---------------------------------------------------------------------------


class ProviderConfig:
    """Immutable capability record derived from a model identifier string.

    Attributes:
        model:               Full model identifier (e.g. "nousresearch/hermes-4-0").
        reasoning_supported: True for Hermes 4 entries on OpenRouter / Nous
                             Portal; False for Hermes 3 and all other models.

    Example::

        cfg = ProviderConfig("nousresearch/hermes-4-0")
        assert cfg.reasoning_supported is True

        cfg3 = ProviderConfig("nousresearch/hermes-3-llama-3.1-70b")
        assert cfg3.reasoning_supported is False
    """

    __slots__ = ("model", "reasoning_supported")

    def __init__(self, model: str) -> None:
        self.model: str = model
        self.reasoning_supported: bool = _reasoning_supported(model)

    def __repr__(self) -> str:  # pragma: no cover
        return (
            f"ProviderConfig(model={self.model!r}, "
            f"reasoning_supported={self.reasoning_supported})"
        )


# ---------------------------------------------------------------------------
# HermesA2AExecutor
# ---------------------------------------------------------------------------


class HermesA2AExecutor(AgentExecutor):
    """A2A executor for Hermes models via OpenAI-compatible API.

    Compared to the LangGraph executor, this is intentionally thin:

    - Single API call per turn (no streaming or ReAct tool loop).
    - System prompt injected as the first ``messages[]`` entry.
    - Hermes 4 reasoning enabled via ``extra_body`` when supported.
    - Reasoning trace logged to OTEL span — never echoed in the reply.
    - Tool definitions passed via native ``tools`` parameter when supplied.

    Parameters
    ----------
    model:
        Full model identifier string (e.g. ``"nousresearch/hermes-4-0"``).
        Used to select the upstream model AND detect reasoning support.
    system_prompt:
        Optional system prompt prepended to every conversation.
    base_url:
        OpenAI-compat endpoint base URL.  Defaults to
        ``OPENAI_BASE_URL`` env var, then ``https://openrouter.ai/api/v1``.
    api_key:
        Provider API key.  Defaults to ``OPENAI_API_KEY`` env var.
    heartbeat:
        Optional ``HeartbeatLoop`` instance used to surface the current
        task description in the platform UI.
    response_format:
        Optional OpenAI-native ``response_format`` dict forwarded verbatim
        to ``chat.completions.create()``.  Supported types:
        ``{"type": "json_schema", "json_schema": {"name": ..., "schema": {...}}}``
        ``{"type": "json_object"}``
        ``{"type": "text"}``
        When ``None`` (default) the parameter is omitted from the API call.
        Invalid dicts cause ``execute()`` to enqueue an error and return
        early without calling the API.
    tools:
        Optional list of OpenAI-format tool definitions to pass via the
        native ``tools`` parameter.  Each entry must have ``"type"`` and
        ``"function"`` keys matching the OpenAI function-calling schema.
        ``None`` or ``[]`` → the ``tools`` key is **omitted** from the API
        call entirely (never sent as ``tools=[]``).
    _client:
        Inject a pre-built ``AsyncOpenAI`` (or compatible mock) — for
        testing only.  When provided, ``base_url`` and ``api_key`` are
        ignored.
    """

    def __init__(
        self,
        model: str,
        system_prompt: str | None = None,
        base_url: str | None = None,
        api_key: str | None = None,
        heartbeat: "HeartbeatLoop | None" = None,
        response_format: "dict | None" = None,
        tools: list[dict] | None = None,
        _client: Any = None,
    ) -> None:
        self.model = model
        self.system_prompt = system_prompt
        self._heartbeat = heartbeat
        self._response_format = response_format
        self._provider = ProviderConfig(model)
        # Empty list and None are treated identically: no tools → omit the
        # parameter from the API call rather than sending tools=[].
        self._tools: list[dict] = list(tools) if tools else []

        if _client is not None:
            # Test injection path — skip real AsyncOpenAI construction so
            # unit tests don't need a live OpenAI API key.
            self._client = _client
        else:
            # Lazy import keeps ``openai`` out of the global module-load path
            # so callers that never use HermesA2AExecutor don't pay the import
            # cost, and tests can stub ``sys.modules["openai"]`` before import.
            from openai import AsyncOpenAI

            self._client = AsyncOpenAI(
                base_url=(
                    base_url
                    or os.environ.get("OPENAI_BASE_URL", "https://openrouter.ai/api/v1")
                ),
                api_key=(
                    api_key
                    or os.environ.get("OPENAI_API_KEY", "")
                ),
            )

    # ------------------------------------------------------------------
    # Internal helpers
    # ------------------------------------------------------------------

    def _build_messages(self, user_input: str) -> list[dict]:
        """Assemble the ``messages`` list: optional system prompt then user turn."""
        msgs: list[dict] = []
        if self.system_prompt:
            msgs.append({"role": "system", "content": self.system_prompt})
        msgs.append({"role": "user", "content": user_input})
        return msgs

    def _log_reasoning(
        self,
        context: RequestContext,
        reasoning: str | None,
        reasoning_details: object | None,
    ) -> None:
        """Write the Hermes 4 reasoning trace to an OTEL span.

        The trace is surfaced to Langfuse / Jaeger for operator inspection.
        It is intentionally **not** returned to the caller — including it in
        the A2A reply would contaminate the agent's next-turn context.

        Any exception is swallowed so a telemetry failure never blocks the
        response being returned.
        """
        try:
            from builtin_tools.telemetry import (
                A2A_TASK_ID,
                WORKSPACE_ID_ATTR,
                get_tracer,
            )

            workspace_id = os.environ.get("WORKSPACE_ID", "unknown")
            tracer = get_tracer()
            with tracer.start_as_current_span("hermes.reasoning") as span:
                span.set_attribute(WORKSPACE_ID_ATTR, workspace_id)
                span.set_attribute(A2A_TASK_ID, context.context_id or "")
                span.set_attribute("hermes.model", self.model)
                span.set_attribute("hermes.reasoning_length", len(reasoning or ""))
                if reasoning:
                    # Cap the preview attribute at 512 chars — full trace is
                    # stored in the span exporter's data store.
                    span.set_attribute("hermes.reasoning_preview", reasoning[:512])
                if reasoning_details is not None:
                    span.set_attribute("hermes.has_reasoning_details", True)
        except Exception:
            logger.debug(
                "hermes_executor: reasoning OTEL log failed (non-fatal)", exc_info=True
            )

    # ------------------------------------------------------------------
    # AgentExecutor interface
    # ------------------------------------------------------------------

    async def execute(self, context: RequestContext, event_queue: EventQueue) -> None:
        """Run a single Hermes turn and enqueue the reply as an A2A Message.

        Sequence:
        1. Extract user text from A2A message parts.
        2. Build ``messages[]`` (optional system + user).
        3. Call OpenAI-compat API; include ``extra_body`` for Hermes 4 and
           ``tools`` when tool definitions are configured.
        4. Extract and log reasoning trace — does NOT appear in the reply.
        5a. If the model returned text content, enqueue it as the reply.
        5b. If the model returned tool calls with no text (``finish_reason``
            ``"tool_calls"``), serialise the calls as JSON and enqueue that.
        """
        import json

        from shared_runtime import extract_message_text

        user_input = extract_message_text(context)
        if not user_input:
            parts = getattr(getattr(context, "message", None), "parts", None)
            logger.warning("HermesA2AExecutor: no text in message parts: %s", parts)
            await event_queue.enqueue_event(
                new_agent_text_message("Error: message contained no text content.")
            )
            return

        messages = self._build_messages(user_input)

        # Validate response_format before hitting the API — invalid dicts
        # enqueue an error and return early without making an API call.
        if self._response_format is not None:
            detail = _validate_response_format(self._response_format)
            if detail is not None:
                await event_queue.enqueue_event(
                    new_agent_text_message(f"Error: invalid response_format — {detail}")
                )
                return

        # Only Hermes 4 entries get extra_body — sending it to Hermes 3
        # or other models is a no-op at best; a 400 at worst.
        extra_body: dict | None = None
        if self._provider.reasoning_supported:
            extra_body = {"reasoning": {"enabled": True}}

        # Build create() kwargs; omit response_format and tools entirely when
        # not set so strict / older providers do not receive unexpected fields.
        create_kwargs: dict = {
            "model": self.model,
            "messages": messages,
            "extra_body": extra_body,
        }
        if self._response_format is not None:
            create_kwargs["response_format"] = self._response_format
        if self._tools:
            create_kwargs["tools"] = self._tools

        try:
            response = await self._client.chat.completions.create(**create_kwargs)

            choice = response.choices[0]
            content: str = choice.message.content or ""

            # ``reasoning`` and ``reasoning_details`` are Hermes 4 / provider
            # extensions not defined in the openai SDK's ChatCompletionMessage
            # schema.  They arrive as dynamic attributes when the upstream API
            # returns them; getattr guards against their absence.
            reasoning: str | None = getattr(choice.message, "reasoning", None)
            reasoning_details: object | None = getattr(
                choice.message, "reasoning_details", None
            )

            if reasoning or reasoning_details:
                logger.info(
                    "hermes_executor: reasoning trace [model=%s len=%d]: %.200s...",
                    self.model,
                    len(reasoning or ""),
                    reasoning or "",
                )
                # Log to OTEL — intentionally omitted from the A2A reply.
                self._log_reasoning(context, reasoning, reasoning_details)

            # Handle tool-call response: when the model returns tool calls
            # with no text content, serialise the calls as JSON so the caller
            # receives structured, parseable output.  This keeps the executor
            # thin (single API call per turn) while not silently discarding
            # function-call intent.
            if not content:
                tool_calls = getattr(choice.message, "tool_calls", None)
                if tool_calls:
                    serialised = json.dumps([
                        {
                            "id": getattr(tc, "id", ""),
                            "type": getattr(tc, "type", "function"),
                            "function": {
                                "name": getattr(
                                    getattr(tc, "function", None), "name", ""
                                ),
                                "arguments": getattr(
                                    getattr(tc, "function", None), "arguments", "{}"
                                ),
                            },
                        }
                        for tc in tool_calls
                    ])
                    logger.info(
                        "hermes_executor: tool_calls response [model=%s n=%d]",
                        self.model,
                        len(tool_calls),
                    )
                    await event_queue.enqueue_event(new_agent_text_message(serialised))
                    return

            final_text = content.strip() or "(no response generated)"
            await event_queue.enqueue_event(new_agent_text_message(final_text))

        except Exception as exc:
            logger.error(
                "hermes_executor: API error [model=%s]: %s",
                self.model,
                type(exc).__name__,
                exc_info=True,
            )
            # Expose only the exception class name — not the message body,
            # which may contain API keys, rate-limit metadata, or provider
            # error details that shouldn't reach the end user.
            # Mirrors the sanitize_agent_error() convention in cli_executor.py.
            await event_queue.enqueue_event(
                new_agent_text_message(f"Agent error: {type(exc).__name__}")
            )

    async def cancel(self, context: RequestContext, event_queue: EventQueue) -> None:
        """Cancel a running task — emits canceled state per A2A protocol."""
        from a2a.types import TaskState, TaskStatus, TaskStatusUpdateEvent

        await event_queue.enqueue_event(
            TaskStatusUpdateEvent(
                status=TaskStatus(state=TaskState.canceled),
                final=True,
            )
        )
