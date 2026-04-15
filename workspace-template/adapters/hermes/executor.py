"""Hermes adapter executor — Phase 1 multi-provider.

Hermes models are accessed via an OpenAI-compatible API. Phase 1 supports 15
providers via the shared ``providers.py`` registry: Nous Portal, OpenRouter,
OpenAI, Anthropic, xAI, Gemini, Qwen, GLM, Kimi, MiniMax, DeepSeek, Groq,
Together, Fireworks, Mistral. Every provider is reached through an OpenAI-compat
``/v1/chat/completions`` endpoint, so one code path handles all of them.

Key resolution order (unchanged from PR 2, extended)
-----------------------------------------------------
1. ``hermes_api_key`` parameter (explicit call-site override — routes to Nous Portal)
2. ``provider`` parameter (explicit provider name — looks up its env var(s))
3. Auto-detect: walk ``providers.RESOLUTION_ORDER`` and pick the first provider
   whose env var is set (``HERMES_API_KEY`` / ``OPENROUTER_API_KEY`` still come
   first so PR 2 back-compat holds).

Raises ``ValueError`` if nothing resolves. The error message lists every env var
that was checked so the operator knows their options without reading source.
"""

from __future__ import annotations

import logging
import os
from typing import Optional

from .providers import PROVIDERS, resolve_provider

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
            api_key=hermes_api_key,
            base_url=cfg.base_url,
            model=model or cfg.default_model,
        )

    # Path 2/3: registry resolution (either explicit provider name or auto-detect).
    cfg, api_key = resolve_provider(provider)
    logger.info(
        "Hermes: provider=%s base_url=%s model=%s",
        cfg.name,
        cfg.base_url,
        model or cfg.default_model,
    )
    return HermesA2AExecutor(
        api_key=api_key,
        base_url=cfg.base_url,
        model=model or cfg.default_model,
    )


class HermesA2AExecutor:
    """LangGraph-compatible AgentExecutor for Hermes-style multi-provider LLMs.

    Uses the OpenAI-compatible ``openai`` client pointed at whichever provider
    was resolved by ``create_executor`` (Nous Portal, OpenRouter, OpenAI,
    Anthropic, xAI, Gemini, Qwen, GLM, Kimi, MiniMax, DeepSeek, Groq, Together,
    Fireworks, Mistral). Matches the pattern of sibling adapters (AutoGen,
    LangGraph) which also use OpenAI-compat clients.

    The ``execute()`` and ``cancel()`` async methods satisfy the
    ``a2a.server.agent_execution.AgentExecutor`` interface so this
    executor can be dropped into the A2A server's DefaultRequestHandler.
    """

    def __init__(
        self,
        api_key: str,
        base_url: str,
        model: str,
        heartbeat=None,
    ):
        self.api_key = api_key
        self.base_url = base_url
        self.model = model
        self._heartbeat = heartbeat

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
            import openai

            client = openai.AsyncOpenAI(
                api_key=self.api_key,
                base_url=self.base_url,
            )

            task_text = build_task_text(user_message, extract_history(context))

            response = await client.chat.completions.create(
                model=self.model,
                messages=[{"role": "user", "content": task_text}],
            )
            reply = response.choices[0].message.content or ""

        except Exception as exc:
            logger.exception("Hermes executor error: %s", exc)
            reply = f"Hermes error: {exc}"
        finally:
            await set_current_task(self._heartbeat, "")

        await event_queue.enqueue_event(new_agent_text_message(reply))

    async def cancel(self, context, event_queue):  # pragma: no cover
        """No-op cancel — Hermes requests are not cancellable mid-flight."""
        pass
