"""Hermes adapter executor — implements create_executor() for PR 2.

Hermes models (Nous Research) are accessed via an OpenAI-compatible API,
either through the Nous Portal directly or via OpenRouter as a fallback.

Key resolution order
--------------------
1. ``hermes_api_key`` parameter (explicit call-site override)
2. ``HERMES_API_KEY`` environment variable (Nous Portal key)
3. ``OPENROUTER_API_KEY`` environment variable (OpenRouter fallback)

Raises ``ValueError`` if none of the three sources yields a non-empty key.
"""

from __future__ import annotations

import logging
import os

logger = logging.getLogger(__name__)

# Default base URLs
_NOUS_BASE_URL = "https://inference-prod.nousresearch.com/v1"
_OPENROUTER_BASE_URL = "https://openrouter.ai/api/v1"

# Default model when routing through OpenRouter
_DEFAULT_MODEL = "nousresearch/hermes-3-llama-3.1-405b"


def create_executor(hermes_api_key: str | None = None):
    """Create and return a LangGraph-compatible executor for the Hermes adapter.

    Key resolution order:
    1. hermes_api_key parameter (if provided)
    2. HERMES_API_KEY environment variable
    3. OPENROUTER_API_KEY environment variable (fallback)
    Raises ValueError if none of the above are found.

    Parameters
    ----------
    hermes_api_key:
        Explicit API key. When provided, the Nous Portal base URL is used.
        When absent and OPENROUTER_API_KEY is the fallback, OpenRouter's
        base URL is used instead.

    Returns
    -------
    HermesA2AExecutor
        A ready-to-use executor instance wired with the resolved key
        and matching base URL.
    """
    api_key: str | None = None
    base_url: str = _NOUS_BASE_URL

    if hermes_api_key:
        api_key = hermes_api_key
        base_url = _NOUS_BASE_URL
        logger.debug("Hermes: using explicit hermes_api_key param")
    else:
        env_hermes = os.environ.get("HERMES_API_KEY", "").strip()
        if env_hermes:
            api_key = env_hermes
            base_url = _NOUS_BASE_URL
            logger.debug("Hermes: using HERMES_API_KEY env var")
        else:
            env_openrouter = os.environ.get("OPENROUTER_API_KEY", "").strip()
            if env_openrouter:
                api_key = env_openrouter
                base_url = _OPENROUTER_BASE_URL
                logger.debug("Hermes: using OPENROUTER_API_KEY env var (fallback)")

    if not api_key:
        raise ValueError(
            "No API key found: provide hermes_api_key param, "
            "or set HERMES_API_KEY or OPENROUTER_API_KEY env var"
        )

    return HermesA2AExecutor(api_key=api_key, base_url=base_url)


class HermesA2AExecutor:
    """LangGraph-compatible AgentExecutor for Hermes models.

    Uses the OpenAI-compatible ``openai`` client pointed at either the
    Nous Portal or OpenRouter, matching the pattern of sibling adapters
    (AutoGen, LangGraph) which all use OpenAI-compatible clients.

    The ``execute()`` and ``cancel()`` async methods satisfy the
    ``a2a.server.agent_execution.AgentExecutor`` interface so this
    executor can be dropped into the A2A server's DefaultRequestHandler.
    """

    def __init__(
        self,
        api_key: str,
        base_url: str = _NOUS_BASE_URL,
        model: str = _DEFAULT_MODEL,
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
