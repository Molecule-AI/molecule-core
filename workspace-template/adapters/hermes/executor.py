"""Hermes adapter executor — Phase 2 multi-provider with native SDK dispatch.

Hermes supports 15 providers via the shared ``providers.py`` registry. Each
provider's ``auth_scheme`` field controls which client + request shape the
executor uses:

- ``auth_scheme="openai"`` (14 providers) — OpenAI-compat ``/v1/chat/completions``
  via the ``openai`` Python SDK. Covers: Nous Portal, OpenRouter, OpenAI, xAI,
  Gemini, Qwen, GLM, Kimi, MiniMax, DeepSeek, Groq, Together, Fireworks, Mistral.

- ``auth_scheme="anthropic"`` (1 provider — anthropic) — native Messages API via
  the ``anthropic`` Python SDK. Phase 2 addition: better tool calling, vision
  support, extended thinking semantics. If the ``anthropic`` package isn't
  installed in the workspace image, ``_do_anthropic_native`` raises a clear
  error with install instructions rather than silently falling back to the
  OpenAI-compat shim (which would lose fidelity invisibly).

Key resolution order (unchanged from Phase 1)
----------------------------------------------
1. ``hermes_api_key`` parameter (explicit call-site override — routes to Nous Portal)
2. ``provider`` parameter (explicit provider name — looks up its env var(s))
3. Auto-detect: walk ``providers.RESOLUTION_ORDER`` and pick the first provider
   whose env var is set.

Raises ``ValueError`` if nothing resolves. The error message lists every env var
that was checked so the operator knows their options without reading source.

Gemini native path (``auth_scheme="gemini"``) is intentionally NOT in this PR
— Phase 2b will land it after measuring Phase 2a's Anthropic rollout.
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
    # Per-provider inference paths
    # ------------------------------------------------------------------

    async def _do_openai_compat(self, task_text: str) -> str:
        """OpenAI-compat inference — used by every provider with auth_scheme='openai'.

        14 of the 15 registered providers route here. Uses ``openai.AsyncOpenAI``
        pointed at the provider's base_url; every provider's API is wire-
        compatible with the OpenAI Chat Completions shape.
        """
        import openai

        client = openai.AsyncOpenAI(
            api_key=self.api_key,
            base_url=self.base_url,
        )
        response = await client.chat.completions.create(
            model=self.model,
            messages=[{"role": "user", "content": task_text}],
        )
        return response.choices[0].message.content or ""

    async def _do_anthropic_native(self, task_text: str) -> str:
        """Native Anthropic Messages API inference.

        Uses the official ``anthropic`` Python SDK for correct tool-calling,
        vision, and extended-thinking semantics that don't translate cleanly
        through the OpenAI-compat shim.

        If the ``anthropic`` package is not installed in the workspace image,
        we raise a clear error rather than silently falling back to the
        OpenAI-compat path — silent fallback would mask the fidelity loss
        (tool_use blocks become plain text, vision gets stripped, etc.).

        Phase 2a minimum viable: single-turn text in, text out, no tools, no
        vision. Phase 2b will add tool-calling, vision, and streaming via
        the same path (still within this method).
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
        response = await client.messages.create(
            model=self.model,
            max_tokens=4096,
            messages=[{"role": "user", "content": task_text}],
        )
        # response.content is a list of ContentBlock; for single-turn text-only
        # the first block is a TextBlock with a .text attribute.
        if response.content and hasattr(response.content[0], "text"):
            return response.content[0].text
        return ""

    async def _do_inference(self, task_text: str) -> str:
        """Dispatch to the right inference path based on provider auth_scheme."""
        scheme = self.provider_cfg.auth_scheme
        if scheme == "anthropic":
            return await self._do_anthropic_native(task_text)
        if scheme == "openai":
            return await self._do_openai_compat(task_text)
        # Unknown scheme — treat as openai-compat for forward-compat with any
        # future provider the registry adds without yet having a native path.
        logger.warning(
            "Hermes: unknown auth_scheme=%r for provider=%s — falling back to openai-compat",
            scheme, self.provider_cfg.name,
        )
        return await self._do_openai_compat(task_text)

    # ------------------------------------------------------------------
    # AgentExecutor interface
    # ------------------------------------------------------------------

    async def execute(self, context, event_queue):  # pragma: no cover
        """Execute a Hermes inference request and push the reply to event_queue."""
        from a2a.utils import new_agent_text_message
        from adapters.shared_runtime import (
            brief_task,
            build_task_text,
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
            task_text = build_task_text(user_message, extract_history(context))
            reply = await self._do_inference(task_text)
        except Exception as exc:
            logger.exception("Hermes executor error: %s", exc)
            reply = f"Hermes error: {exc}"
        finally:
            await set_current_task(self._heartbeat, "")

        await event_queue.enqueue_event(new_agent_text_message(reply))

    async def cancel(self, context, event_queue):  # pragma: no cover
        """No-op cancel — Hermes requests are not cancellable mid-flight."""
        pass
