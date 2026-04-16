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

from .escalation import LadderRung, parse_ladder, should_escalate
from .providers import PROVIDERS, ProviderConfig, resolve_provider

logger = logging.getLogger(__name__)


def create_executor(
    hermes_api_key: Optional[str] = None,
    provider: Optional[str] = None,
    model: Optional[str] = None,
    config_path: Optional[str] = None,
    escalation_ladder: Optional[list] = None,
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
    config_path:
        Path to the workspace's ``/configs`` directory. Phase 2d-i reads
        ``system-prompt.md`` from here on every ``execute()`` call and
        passes the content as a system instruction to the native SDK.
        Optional — omit to skip system-prompt injection (tests do this).

    Returns
    -------
    HermesA2AExecutor
        A ready-to-use executor wired with the resolved api_key + base_url
        + model + config_path.

    Raises
    ------
    ValueError
        If ``provider`` is an unknown name, if ``provider`` is known but its
        env vars are all empty, or if auto-detect finds nothing.
    """
    ladder = parse_ladder(escalation_ladder)
    if ladder:
        logger.info(
            "Hermes: escalation ladder configured — %d rungs (%s)",
            len(ladder),
            " → ".join(f"{r.provider}:{r.model}" for r in ladder),
        )

    # Path 1: PR 2 back-compat — explicit hermes_api_key routes to Nous Portal.
    if hermes_api_key:
        cfg = PROVIDERS["nous_portal"]
        logger.debug("Hermes: using explicit hermes_api_key param (Nous Portal)")
        return HermesA2AExecutor(
            provider_cfg=cfg,
            api_key=hermes_api_key,
            model=model or cfg.default_model,
            config_path=config_path,
            escalation_ladder=ladder,
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
        config_path=config_path,
        escalation_ladder=ladder,
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
        config_path: Optional[str] = None,
        escalation_ladder: Optional[list] = None,
    ):
        self.provider_cfg = provider_cfg
        self.api_key = api_key
        self.base_url = provider_cfg.base_url
        self.model = model
        self._heartbeat = heartbeat
        # Phase 2d-i: config_path lets execute() read /configs/system-prompt.md
        # on each turn and pass it to the native SDK's `system=` /
        # `system_instruction=` / prepended message. Optional because older
        # callers + tests construct executors directly.
        self._config_path = config_path
        # Phase 3: escalation ladder. When non-empty, _do_inference retries
        # transient-failure classes (rate limit, 5xx, overload, context-length)
        # on each rung in turn before raising. Empty / None = single-shot,
        # original behaviour. See adapters.hermes.escalation.
        self._ladder: list[LadderRung] = parse_ladder(escalation_ladder) or []

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
        system_prompt: Optional[str] = None,
    ) -> str:
        """OpenAI-compat inference — used by every provider with auth_scheme='openai'.

        13 of the 15 registered providers route here. Uses ``openai.AsyncOpenAI``
        pointed at the provider's base_url; every provider's API is wire-
        compatible with the OpenAI Chat Completions shape.

        Phase 2c: accepts multi-turn history.
        Phase 2d-i: accepts optional system_prompt, prepended as a
        ``{"role":"system"}`` message per the OpenAI Chat Completions convention.
        """
        import openai

        client = openai.AsyncOpenAI(
            api_key=self.api_key,
            base_url=self.base_url,
        )
        messages = self._history_to_openai_messages(user_message, history or [])
        if system_prompt:
            messages = [{"role": "system", "content": system_prompt}, *messages]
        response = await client.chat.completions.create(
            model=self.model,
            messages=messages,
        )
        return response.choices[0].message.content or ""

    async def _do_anthropic_native(
        self,
        user_message: str,
        history: "list[tuple[str, str]] | None" = None,
        system_prompt: Optional[str] = None,
    ) -> str:
        """Native Anthropic Messages API inference.

        Uses the official ``anthropic`` Python SDK for correct tool-calling,
        vision, and extended-thinking semantics that don't translate cleanly
        through the OpenAI-compat shim.

        Phase 2a: single-turn text.
        Phase 2c: multi-turn history.
        Phase 2d-i: optional system_prompt passed via Anthropic's native
        top-level ``system=`` parameter — NOT as a message in the messages
        list (Anthropic's Messages API requires system prompts to be at the
        top level, not inline like OpenAI).
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
        create_kwargs: dict = {
            "model": self.model,
            "max_tokens": 4096,
            "messages": messages,
        }
        if system_prompt:
            create_kwargs["system"] = system_prompt
        response = await client.messages.create(**create_kwargs)
        # response.content is a list of ContentBlock; for text-only the first
        # block is a TextBlock with a .text attribute.
        if response.content and hasattr(response.content[0], "text"):
            return response.content[0].text
        return ""

    async def _do_gemini_native(
        self,
        user_message: str,
        history: "list[tuple[str, str]] | None" = None,
        system_prompt: Optional[str] = None,
    ) -> str:
        """Native Google Gemini ``generateContent`` inference.

        Uses the official ``google-genai`` Python SDK for correct vision
        content blocks, tool/function calling, system instructions, and
        thinking config. These all get stripped or mis-translated through
        the OpenAI-compat ``/v1beta/openai`` shim.

        Phase 2b: single-turn text.
        Phase 2c: multi-turn history via Gemini's ``contents=[{role,parts}]``
        shape (note: role is ``"user"`` / ``"model"``, NOT ``"assistant"``).
        Phase 2d-i: system_prompt passed via native
        ``config.system_instruction`` — Gemini's top-level system field.
        """
        try:
            from google import genai  # type: ignore[import-not-found]
            from google.genai import types as genai_types  # type: ignore[import-not-found]
        except ImportError as exc:  # pragma: no cover — exercised by test_missing_sdk
            raise RuntimeError(
                "Hermes gemini native path requires the `google-genai` package. "
                "Install in the workspace image with `pip install google-genai>=1.0.0` "
                "or set HERMES provider=openrouter to route Gemini models through "
                "OpenRouter's OpenAI-compat shim instead."
            ) from exc

        client = genai.Client(api_key=self.api_key)
        contents = self._history_to_gemini_contents(user_message, history or [])
        generate_kwargs: dict = {
            "model": self.model,
            "contents": contents,
        }
        if system_prompt:
            generate_kwargs["config"] = genai_types.GenerateContentConfig(
                system_instruction=system_prompt,
            )
        response = await client.aio.models.generate_content(**generate_kwargs)
        # response.text is the flattened text across all parts of the first
        # candidate. For text-only that's the whole reply.
        return response.text or ""

    async def _do_inference(
        self,
        user_message: str,
        history: "list[tuple[str, str]] | None" = None,
        system_prompt: Optional[str] = None,
    ) -> str:
        """Dispatch to the right inference path based on provider auth_scheme.

        Phase 2c: multi-turn history.
        Phase 2d-i: optional system_prompt is passed through to the native
        system field of whichever path wins dispatch.
        Phase 3: when an escalation ladder is configured, transient failures
        (rate limit, 5xx, overload, context-length) promote to the next rung
        before raising. No ladder = single-shot, original behaviour.
        """
        # Fast path: no ladder configured — single call on the pinned model.
        if not self._ladder:
            return await self._dispatch(
                self.provider_cfg, self.model, user_message, history, system_prompt,
            )

        # Slow path: walk the ladder. Start with the pinned (provider, model)
        # so the first attempt matches non-ladder behaviour exactly — the
        # ladder only kicks in when the first attempt fails escalatably.
        attempts: list[tuple[ProviderConfig, str]] = [(self.provider_cfg, self.model)]
        for rung in self._ladder:
            rung_cfg = PROVIDERS.get(rung.provider)
            if rung_cfg is None:
                logger.warning(
                    "Hermes ladder: provider %r not in registry, skipping rung",
                    rung.provider,
                )
                continue
            attempts.append((rung_cfg, rung.model))

        last_exc: Optional[BaseException] = None
        for i, (cfg, model) in enumerate(attempts):
            try:
                reply = await self._dispatch(
                    cfg, model, user_message, history, system_prompt,
                )
                if i > 0:
                    logger.info(
                        "Hermes ladder: succeeded on rung %d (%s:%s) after %d failed attempt(s)",
                        i, cfg.name, model, i,
                    )
                return reply
            except Exception as exc:
                last_exc = exc
                if i == len(attempts) - 1:
                    logger.error(
                        "Hermes ladder: exhausted all %d rungs — raising. Last error on %s:%s: %s",
                        len(attempts), cfg.name, model, exc,
                    )
                    raise
                if not should_escalate(exc):
                    logger.info(
                        "Hermes ladder: non-escalatable error on %s:%s — raising without advancing: %s",
                        cfg.name, model, exc,
                    )
                    raise
                logger.warning(
                    "Hermes ladder: escalatable failure on rung %d (%s:%s), advancing. Error: %s",
                    i, cfg.name, model, exc,
                )

        # Unreachable — the last iteration either returns or raises, but
        # satisfying the type checker without a blank return.
        if last_exc is not None:
            raise last_exc
        return ""  # pragma: no cover

    async def _dispatch(
        self,
        cfg: ProviderConfig,
        model: str,
        user_message: str,
        history: "list[tuple[str, str]] | None",
        system_prompt: Optional[str],
    ) -> str:
        """Single-attempt dispatch on (cfg, model).

        Temporarily rebinds ``self.provider_cfg`` + ``self.base_url`` + ``self.model``
        so the existing per-provider paths pick up the rung's config. Restores
        the original values in a finally block so a raised error leaves the
        executor pinned to its constructor-given state (next call on the same
        executor instance starts fresh at the top of the ladder).

        For the ladder's non-first rungs, ``self.api_key`` must be the rung's
        provider key — we resolve it here via ``resolve_provider`` so the
        first-rung API key (for the pinned provider) isn't mis-used against a
        different provider's base URL. That lookup can raise ``ValueError``
        when the rung's env var isn't set; ``should_escalate(ValueError)``
        returns False so the ladder correctly STOPS rather than escalating
        further into nothing.
        """
        # Fast path: rung matches the executor's pinned config — reuse the
        # existing api_key, skip the provider re-resolve.
        if cfg is self.provider_cfg and model == self.model:
            scheme = cfg.auth_scheme
            if scheme == "anthropic":
                return await self._do_anthropic_native(user_message, history, system_prompt)
            if scheme == "gemini":
                return await self._do_gemini_native(user_message, history, system_prompt)
            if scheme == "openai":
                return await self._do_openai_compat(user_message, history, system_prompt)
            logger.warning(
                "Hermes: unknown auth_scheme=%r for provider=%s — falling back to openai-compat",
                scheme, cfg.name,
            )
            return await self._do_openai_compat(user_message, history, system_prompt)

        # Different rung — temporarily rebind provider_cfg + model + api_key.
        # resolve_provider reads the rung's env vars fresh.
        _, rung_key = resolve_provider(cfg.name)
        orig_cfg, orig_model, orig_key, orig_base = (
            self.provider_cfg, self.model, self.api_key, self.base_url,
        )
        try:
            self.provider_cfg = cfg
            self.model = model
            self.api_key = rung_key
            self.base_url = cfg.base_url
            scheme = cfg.auth_scheme
            if scheme == "anthropic":
                return await self._do_anthropic_native(user_message, history, system_prompt)
            if scheme == "gemini":
                return await self._do_gemini_native(user_message, history, system_prompt)
            if scheme == "openai":
                return await self._do_openai_compat(user_message, history, system_prompt)
            logger.warning(
                "Hermes: unknown auth_scheme=%r for provider=%s — falling back to openai-compat",
                scheme, cfg.name,
            )
            return await self._do_openai_compat(user_message, history, system_prompt)
        finally:
            self.provider_cfg = orig_cfg
            self.model = orig_model
            self.api_key = orig_key
            self.base_url = orig_base

    # ------------------------------------------------------------------
    # AgentExecutor interface
    # ------------------------------------------------------------------

    async def execute(self, context, event_queue):  # pragma: no cover
        """Execute a Hermes inference request and push the reply to event_queue.

        Phase 2c: multi-turn history.
        Phase 2d-i: reads ``/configs/system-prompt.md`` via
        ``executor_helpers.get_system_prompt`` each turn (supports hot-reload)
        and passes the text to the dispatch layer. Each provider path uses
        its native system field — Anthropic's top-level ``system=``, Gemini's
        ``system_instruction=`` via ``GenerateContentConfig``, or OpenAI's
        ``{"role":"system"}`` message at the head of the messages list.
        """
        from a2a.utils import new_agent_text_message
        from adapters.shared_runtime import (
            brief_task,
            extract_history,
            extract_message_text,
            set_current_task,
        )
        from executor_helpers import get_system_prompt

        user_message = extract_message_text(context)
        if not user_message:
            await event_queue.enqueue_event(new_agent_text_message("No message provided"))
            return

        await set_current_task(self._heartbeat, brief_task(user_message))

        try:
            history = extract_history(context)
            system_prompt = (
                get_system_prompt(self._config_path) if self._config_path else None
            )
            reply = await self._do_inference(user_message, history, system_prompt)
        except Exception as exc:
            logger.exception("Hermes executor error: %s", exc)
            reply = f"Hermes error: {exc}"
        finally:
            await set_current_task(self._heartbeat, "")

        await event_queue.enqueue_event(new_agent_text_message(reply))

    async def cancel(self, context, event_queue):  # pragma: no cover
        """No-op cancel — Hermes requests are not cancellable mid-flight."""
        pass
