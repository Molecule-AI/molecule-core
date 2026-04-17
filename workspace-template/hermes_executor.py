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

Hermes 3 / unknown models
--------------------------
No ``extra_body`` is sent.  The response is processed identically to any
other OpenAI-compat model call.  The Hermes 3 path is exercised by the
existing adapter test suite and must remain unchanged.
"""

from __future__ import annotations

import logging
import os
from typing import TYPE_CHECKING, Any, Sequence

from a2a.server.agent_execution import AgentExecutor, RequestContext
from a2a.server.events import EventQueue
from a2a.utils import new_agent_text_message

if TYPE_CHECKING:
    from heartbeat import HeartbeatLoop

logger = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# Stacked system-message merge
# ---------------------------------------------------------------------------


def _merge_system_messages(messages: list[dict]) -> list[dict]:
    """Collapse consecutive leading system messages into a single system message.

    vLLM (and the Nous Hermes portal) accept exactly **one** system message.
    When a messages array is built from multiple sources — e.g. a base system
    prompt, a workspace-level config block, and a per-session user override —
    the consecutive ``{"role": "system"}`` entries at the front cause vLLM to
    reject or silently drop all but the first.

    This function is a stateless pre-flight transform applied in
    ``_build_messages`` before the array is forwarded to the API.

    Rules:
    - Only the **uninterrupted leading run** of ``role == "system"`` entries is
      merged.  A system message that appears after a ``user`` or ``assistant``
      turn is left in place.
    - Content strings are joined with ``"\\n\\n"`` (double newline).
    - A single leading system message is returned as-is (no copy).
    - An empty list is returned as-is.

    Example::

        >>> _merge_system_messages([
        ...     {"role": "system", "content": "Base prompt."},
        ...     {"role": "system", "content": "Workspace config."},
        ...     {"role": "user",   "content": "Hello!"},
        ... ])
        [{"role": "system", "content": "Base prompt.\\n\\nWorkspace config."}, {"role": "user", "content": "Hello!"}]
    """
    # Find the end of the leading system-message run.
    end = 0
    while end < len(messages) and messages[end].get("role") == "system":
        end += 1

    # Zero or one system message — nothing to merge, return unchanged.
    if end <= 1:
        return messages

    merged_content = "\n\n".join(
        m.get("content", "") for m in messages[:end]
    )
    return [{"role": "system", "content": merged_content}, *messages[end:]]


# ---------------------------------------------------------------------------
# Per-model reasoning capability detection
# ---------------------------------------------------------------------------

# Nous-recommended sampling defaults for Hermes models.
# Applied ONLY when the caller does not supply an explicit value for that
# parameter.  This matches the recommended settings from the Nous Research
# documentation for Hermes 3 and Hermes 4.
#
# Reference: https://nousresearch.com/hermes3/ — "Recommended Settings"
_HERMES_SAMPLING_DEFAULTS: dict = {
    "temperature": 0.7,
    "top_p": 0.9,
    "top_k": 50,
    "repetition_penalty": 1.1,
}

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
    sampling_params:
        Optional mapping of sampling overrides (``temperature``, ``top_p``,
        ``top_k``, ``repetition_penalty``).  Values supplied here take
        precedence over ``_HERMES_SAMPLING_DEFAULTS``; values absent from
        both fall back to the provider's own defaults.  Pass
        ``{"temperature": 1.0}`` to raise the temperature while keeping the
        other Nous defaults active.  Pass ``{}`` to disable all defaults.
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
        sampling_params: dict | None = None,
        _client: Any = None,
    ) -> None:
        self.model = model
        self.system_prompt = system_prompt
        self._heartbeat = heartbeat
        self._provider = ProviderConfig(model)
        # Merge caller-supplied overrides on top of Nous defaults.
        # None      → all four defaults applied.
        # {}        → empty dict explicitly opts out of all defaults.
        # {k: v, …} → start from defaults, apply the supplied overrides.
        if sampling_params is None:
            self._sampling: dict = dict(_HERMES_SAMPLING_DEFAULTS)
        elif not sampling_params:
            self._sampling = {}
        else:
            merged = dict(_HERMES_SAMPLING_DEFAULTS)
            merged.update(sampling_params)
            self._sampling = merged

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
        """Assemble the ``messages`` list: optional system prompt then user turn.

        After constructing the list, ``_merge_system_messages`` is applied so
        that any consecutive leading system entries (e.g. from stacked prompts
        injected by subclasses or future callers) are collapsed into one before
        the array is forwarded to vLLM.
        """
        msgs: list[dict] = []
        if self.system_prompt:
            msgs.append({"role": "system", "content": self.system_prompt})
        msgs.append({"role": "user", "content": user_input})
        return _merge_system_messages(msgs)

    def _build_sampling_kwargs(self) -> dict:
        """Return sampling keyword arguments to pass to ``completions.create()``.

        Returns a copy of ``self._sampling`` so the instance dict is never
        mutated by the caller.  Returns an empty dict when ``self._sampling``
        is empty (operator explicitly opted out of defaults).
        """
        return dict(self._sampling)

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
        3. Call OpenAI-compat API; include ``extra_body`` for Hermes 4.
        4. Extract and log reasoning trace — does NOT appear in the reply.
        5. Enqueue a final ``Message`` with the content text.
        """
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

        # Only Hermes 4 entries get extra_body — sending it to Hermes 3
        # or other models is a no-op at best; a 400 at worst.
        extra_body: dict | None = None
        if self._provider.reasoning_supported:
            extra_body = {"reasoning": {"enabled": True}}

        # Apply Nous-recommended sampling defaults (temperature=0.7, top_p=0.9,
        # top_k=50, repetition_penalty=1.1) unless the caller has supplied
        # explicit overrides via the ``sampling_params`` constructor argument.
        sampling_kwargs = self._build_sampling_kwargs()

        try:
            response = await self._client.chat.completions.create(
                model=self.model,
                messages=messages,
                extra_body=extra_body,
                **sampling_kwargs,
            )

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
