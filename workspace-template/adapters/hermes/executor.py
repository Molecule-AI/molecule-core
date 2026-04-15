"""Hermes adapter executor — Phase 2 multi-provider with native SDK dispatch.

Hermes supports 15 providers via the shared ``providers.py`` registry. Each
provider's ``auth_scheme`` field controls which client + request shape the
executor uses:

- ``auth_scheme="openai"`` (13 providers) — OpenAI-compat ``/v1/chat/completions``
  via the ``openai`` Python SDK. Covers: Nous Portal, OpenRouter, OpenAI, xAI,
  Qwen, GLM, Kimi, MiniMax, DeepSeek, Groq, Together, Fireworks, Mistral.

- ``auth_scheme="anthropic"`` (1 provider — anthropic) — native Messages API via
  the ``anthropic`` Python SDK. Phase 2a: better tool calling, vision support,
  extended thinking semantics. If the ``anthropic`` package isn't installed in
  the workspace image, ``_do_anthropic_native`` raises a clear error with
  install instructions rather than silently falling back to the OpenAI-compat
  shim (which would lose fidelity invisibly).

- ``auth_scheme="gemini"`` (1 provider — gemini) — native ``generateContent`` API
  via the official ``google-genai`` Python SDK. Phase 2b: first-class vision
  content blocks, tool/function calling, system instructions, and thinking
  config — all of which the OpenAI-compat shim at ``/v1beta/openai`` either
  strips or mis-translates. Same fail-loud semantics as the anthropic path.

Key resolution order (unchanged from Phase 1)
----------------------------------------------
1. ``hermes_api_key`` parameter (explicit call-site override — routes to Nous Portal)
2. ``provider`` parameter (explicit provider name — looks up its env var(s))
3. Auto-detect: walk ``providers.RESOLUTION_ORDER`` and pick the first provider
   whose env var is set.

Raises ``ValueError`` if nothing resolves. The error message lists every env var
that was checked so the operator knows their options without reading source.
"""

from __future__ import annotations

import logging
import os
from typing import Optional

from .providers import PROVIDERS, ProviderConfig, resolve_provider

logger = logging.getLogger(__name__)


def create_executor(
    hermes_api_key: Optional[str] = None,
    provider: Optional[str] = None,
    model: Optional[str] = None,
):
    """Create and return a LangGraph-compatible executor for the Hermes adapter.

    Parameters
    ----------
    hermes_api_key:
        Explicit API key. When provided, the call routes to Nous Portal (the
        PR 2 back-compat path) regardless of ``provider``.
    provider:
        Canonical provider short name from ``providers.PROVIDERS`` (e.g.
        ``"openai"``, ``"anthropic"``, ``"qwen"``, ``"xai"``). When set, the
        registry entry's env vars are used to find the API key and its
        base URL + default model override the auto-detect path. When unset,
        auto-detect walks ``providers.RESOLUTION_ORDER`` until it finds a
        provider whose env var is set.
    model:
        Override the provider's default model. Passed straight through to
        ``chat.completions.create``.

    Returns
    -------
    HermesA2AExecutor
        A ready-to-use executor wired with the resolved api_key + base_url
        + model.

    Raises
    ------
    ValueError
        If ``provider`` is an unknown name, if ``provider`` is known but its
        env vars are all empty, or if auto-detect finds nothing.
    """
    # Path 1: PR 2 back-compat — explicit hermes_api_key routes to Nous Portal.
    if hermes_api_key:
        cfg = PROVIDERS["nous_portal"]
        logger.debug("Hermes: using explicit hermes_api_key param (Nous Portal)")
        return HermesA2AExecutor(
            provider_cfg=cfg,
            api_key=hermes_api_key,
            model=model or cfg.default_model,
        )

    # Path 2/3: registry resolution (either explicit provider name or auto-detect).
    cfg, api_key = resolve_provider(provider)
    logger.info(
        "Hermes: provider=%s auth_scheme=%s base_url=%s model=%s",
        cfg.name,
        cfg.auth_scheme,
        cfg.base_url,
        model or cfg.default_model,
    )
    return HermesA2AExecutor(
        provider_cfg=cfg,
        api_key=api_key,
        model=model or cfg.default_model,
    )


class HermesA2AExecutor:
    """LangGraph-compatible AgentExecutor for Hermes-style multi-provider LLMs.

    Dispatches each inference call based on ``provider_cfg.auth_scheme``:

    - ``"openai"`` → OpenAI-compat ``/v1/chat/completions`` via the ``openai`` SDK
    - ``"anthropic"`` → native Messages API via the ``anthropic`` SDK

    The ``execute()`` and ``cancel()`` async methods satisfy the
    ``a2a.server.agent_execution.AgentExecutor`` interface so this
    executor can be dropped into the A2A server's DefaultRequestHandler.
    """

    def __init__(
        self,
        provider_cfg: ProviderConfig,
        api_key: str,
        model: str,
        heartbeat=None,
    ):
        self.provider_cfg = provider_cfg
        self.api_key = api_key
        self.base_url = provider_cfg.base_url
        self.model = model
        self._heartbeat = heartbeat

    # ------------------------------------------------------------------
    # History → provider-specific message list converters
    # ------------------------------------------------------------------
    #
    # The A2A shared runtime gives us history as ``list[tuple[str, str]]``
    # with roles ``"human"`` / ``"ai"``. Each provider wants a different
    # shape:
    #
    #   OpenAI-compat: [{"role":"user"|"assistant", "content": str}, ...]
    #   Anthropic:     [{"role":"user"|"assistant", "content": str}, ...]  (same)
    #   Gemini:        [{"role":"user"|"model", "parts": [{"text": str}]}, ...]
    #
    # Before Phase 2c these were flattened into a single user turn via
    # ``shared_runtime.build_task_text``, which worked for basic text
    # handoff but lost the model's native multi-turn awareness (system
    # prompts, tool-use history, role attribution for instruction
    # following). Phase 2c keeps the turns as turns.

    @staticmethod
    def _history_to_openai_messages(
        user_message: str,
        history: "list[tuple[str, str]]",
    ) -> "list[dict]":
        """Convert A2A history + current turn to OpenAI Chat Completions shape."""
        messages: list[dict] = []
        for role, text in history or []:
            messages.append({
                "role": "user" if role == "human" else "assistant",
                "content": text,
            })
        messages.append({"role": "user", "content": user_message})
        return messages

    @staticmethod
    def _history_to_anthropic_messages(
        user_message: str,
        history: "list[tuple[str, str]]",
    ) -> "list[dict]":
        """Convert A2A history + current turn to Anthropic Messages API shape.

        Identical wire format to OpenAI (``role`` + ``content``) for text-only
        turns, so we just delegate. The difference matters for tool_use /
        content blocks, which are Phase 2d territory.
        """
        return HermesA2AExecutor._history_to_openai_messages(user_message, history)

    @staticmethod
    def _history_to_gemini_contents(
        user_message: str,
        history: "list[tuple[str, str]]",
    ) -> "list[dict]":
        """Convert A2A history + current turn to Gemini generateContent shape.

        Gemini uses ``role: "user" | "model"`` (NOT "assistant") and wraps
        text in a ``parts: [{"text": ...}]`` list.
        """
        contents: list[dict] = []
        for role, text in history or []:
            contents.append({
                "role": "user" if role == "human" else "model",
                "parts": [{"text": text}],
            })
        contents.append({"role": "user", "parts": [{"text": user_message}]})
        return contents

    # ------------------------------------------------------------------
    # Per-provider inference paths
    # ------------------------------------------------------------------

    async def _do_openai_compat(
        self,
        user_message: str,
        history: "list[tuple[str, str]] | None" = None,
    ) -> str:
        """OpenAI-compat inference — used by every provider with auth_scheme='openai'.

        13 of the 15 registered providers route here. Uses ``openai.AsyncOpenAI``
        pointed at the provider's base_url; every provider's API is wire-
        compatible with the OpenAI Chat Completions shape.

        Phase 2c: accepts multi-turn history. The old single-``task_text`` call
        shape (pre-2c) is preserved — pass the flattened text as ``user_message``
        with no history and the call degrades gracefully to the original
        behavior. See ``_history_to_openai_messages`` for the conversion.
        """
        import openai

        client = openai.AsyncOpenAI(
            api_key=self.api_key,
            base_url=self.base_url,
        )
        messages = self._history_to_openai_messages(user_message, history or [])
        response = await client.chat.completions.create(
            model=self.model,
            messages=messages,
        )
        return response.choices[0].message.content or ""

    async def _do_anthropic_native(
        self,
        user_message: str,
        history: "list[tuple[str, str]] | None" = None,
    ) -> str:
        """Native Anthropic Messages API inference.

        Uses the official ``anthropic`` Python SDK for correct tool-calling,
        vision, and extended-thinking semantics that don't translate cleanly
        through the OpenAI-compat shim.

        If the ``anthropic`` package is not installed in the workspace image,
        we raise a clear error rather than silently falling back to the
        OpenAI-compat path — silent fallback would mask the fidelity loss
        (tool_use blocks become plain text, vision gets stripped, etc.).

        Phase 2a: single-turn text in, text out. Phase 2c: multi-turn history.
        Tools + vision remain Phase 2d.
        """
        try:
            import anthropic
        except ImportError as exc:  # pragma: no cover — exercised by test_missing_sdk
            raise RuntimeError(
                "Hermes anthropic native path requires the `anthropic` package. "
                "Install in the workspace image with `pip install anthropic>=0.39.0` "
                "or set HERMES provider=openrouter to route Claude models through "
                "OpenRouter's OpenAI-compat shim instead."
            ) from exc

        client = anthropic.AsyncAnthropic(api_key=self.api_key)
        messages = self._history_to_anthropic_messages(user_message, history or [])
        response = await client.messages.create(
            model=self.model,
            max_tokens=4096,
            messages=messages,
        )
        # response.content is a list of ContentBlock; for text-only the first
        # block is a TextBlock with a .text attribute.
        if response.content and hasattr(response.content[0], "text"):
            return response.content[0].text
        return ""

    async def _do_gemini_native(
        self,
        user_message: str,
        history: "list[tuple[str, str]] | None" = None,
    ) -> str:
        """Native Google Gemini ``generateContent`` inference.

        Uses the official ``google-genai`` Python SDK for correct vision
        content blocks, tool/function calling, system instructions, and
        thinking config. These all get stripped or mis-translated through
        the OpenAI-compat ``/v1beta/openai`` shim.

        If the ``google-genai`` package is not installed in the workspace
        image, raise a clear error with install instructions rather than
        silently falling back to the OpenAI-compat shim (same fail-loud
        semantics as the anthropic path).

        Phase 2b: single-turn text in, text out. Phase 2c: multi-turn history
        via Gemini's ``contents=[{role,parts}]`` shape (note: role is
        ``"user"`` / ``"model"``, NOT ``"assistant"``).
        """
        try:
            from google import genai  # type: ignore[import-not-found]
        except ImportError as exc:  # pragma: no cover — exercised by test_missing_sdk
            raise RuntimeError(
                "Hermes gemini native path requires the `google-genai` package. "
                "Install in the workspace image with `pip install google-genai>=1.0.0` "
                "or set HERMES provider=openrouter to route Gemini models through "
                "OpenRouter's OpenAI-compat shim instead."
            ) from exc

        client = genai.Client(api_key=self.api_key)
        contents = self._history_to_gemini_contents(user_message, history or [])
        response = await client.aio.models.generate_content(
            model=self.model,
            contents=contents,
        )
        # response.text is the flattened text across all parts of the first
        # candidate. For text-only that's the whole reply.
        return response.text or ""

    async def _do_inference(
        self,
        user_message: str,
        history: "list[tuple[str, str]] | None" = None,
    ) -> str:
        """Dispatch to the right inference path based on provider auth_scheme.

        Phase 2c: takes ``user_message`` + optional ``history`` list-of-tuples,
        passes through to the chosen path. Each path has its own history →
        provider-message conversion via the static helpers above.
        """
        scheme = self.provider_cfg.auth_scheme
        if scheme == "anthropic":
            return await self._do_anthropic_native(user_message, history)
        if scheme == "gemini":
            return await self._do_gemini_native(user_message, history)
        if scheme == "openai":
            return await self._do_openai_compat(user_message, history)
        # Unknown scheme — treat as openai-compat for forward-compat with any
        # future provider the registry adds without yet having a native path.
        logger.warning(
            "Hermes: unknown auth_scheme=%r for provider=%s — falling back to openai-compat",
            scheme, self.provider_cfg.name,
        )
        return await self._do_openai_compat(user_message, history)

    # ------------------------------------------------------------------
    # AgentExecutor interface
    # ------------------------------------------------------------------

    async def execute(self, context, event_queue):  # pragma: no cover
        """Execute a Hermes inference request and push the reply to event_queue.

        Phase 2c: passes the conversation history to the dispatch layer as a
        structured list of (role, text) turns instead of flattening via
        ``build_task_text``. Each provider path converts the list into its
        native multi-turn message shape (OpenAI messages, Anthropic messages,
        or Gemini contents). This gives the model its native multi-turn
        awareness for instruction following.
        """
        from a2a.utils import new_agent_text_message
        from adapters.shared_runtime import (
            brief_task,
            extract_history,
            extract_message_text,
            set_current_task,
        )

        user_message = extract_message_text(context)
        if not user_message:
            await event_queue.enqueue_event(new_agent_text_message("No message provided"))
            return

        await set_current_task(self._heartbeat, brief_task(user_message))

        try:
            history = extract_history(context)
            reply = await self._do_inference(user_message, history)
        except Exception as exc:
            logger.exception("Hermes executor error: %s", exc)
            reply = f"Hermes error: {exc}"
        finally:
            await set_current_task(self._heartbeat, "")

        await event_queue.enqueue_event(new_agent_text_message(reply))

    async def cancel(self, context, event_queue):  # pragma: no cover
        """No-op cancel — Hermes requests are not cancellable mid-flight."""
        pass
