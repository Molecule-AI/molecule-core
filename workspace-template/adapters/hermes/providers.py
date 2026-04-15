"""Hermes adapter provider registry — Phase 1 of the multi-provider expansion.

Extends the original PR-2 Hermes executor (Nous Portal + OpenRouter only) to a
registry of 12 providers. Every provider in this registry is reached via its
OpenAI-compat endpoint, which means the existing ``openai.AsyncOpenAI`` client
and request shape in ``executor.py`` Just Works without any new dependencies.

Native SDK paths (Anthropic Messages API, Gemini generateContent API) are
Phase 2 — they give better tool-calling + vision fidelity but are not
required to unblock the basic "CEO wants Hermes on Qwen / GLM / xAI /
Gemini" asks that triggered this work.

## Design
- ``ProviderConfig`` captures everything needed to point the OpenAI client at
  a provider: env var(s), base URL, default model, auth scheme.
- ``PROVIDERS`` is a dict keyed by canonical short name (``"openai"``,
  ``"anthropic"``, ``"qwen"``, etc.).
- ``RESOLUTION_ORDER`` is the auto-detect sequence used when the caller
  doesn't specify a provider — it tries each provider's env vars in turn and
  picks the first one that's set.
- ``resolve_provider(explicit)`` returns ``(ProviderConfig, api_key)`` or
  raises ``ValueError`` with a helpful message listing every env var it
  checked.

## Back-compat
The original ``HERMES_API_KEY`` and ``OPENROUTER_API_KEY`` env vars still work
and still route to Nous Portal / OpenRouter respectively — they're just now
registered as two entries in ``PROVIDERS`` rather than hardcoded in
``create_executor``.

## Adding a new provider
1. Append a new ``ProviderConfig`` entry under ``PROVIDERS``
2. Add its short name to ``RESOLUTION_ORDER`` in the desired priority slot
3. Document the env var in the workspace ``.env.example`` (if present)
That's it. Nothing else needs to change — the executor reads the registry.
"""

from __future__ import annotations

import os
from dataclasses import dataclass
from typing import Optional


@dataclass(frozen=True)
class ProviderConfig:
    """Everything the Hermes executor needs to talk to a single LLM provider.

    Every provider in Phase 1 is reachable via an OpenAI-compatible
    ``/v1/chat/completions`` endpoint, so ``auth_scheme`` is always
    ``"openai"`` (Bearer token, OpenAI-style messages payload). Phase 2
    will add ``"anthropic"`` (native Messages API) and ``"gemini"`` (native
    generateContent API) for roles that need better tool-call fidelity.
    """

    name: str
    """Canonical short name — the key used in ``PROVIDERS`` and the ``provider`` kwarg."""

    env_vars: tuple[str, ...]
    """API key env vars, checked in order. First non-empty value wins.
    Supporting multiple env vars lets us accept common aliases
    (e.g. ``QWEN_API_KEY`` AND ``DASHSCOPE_API_KEY`` both work for Alibaba Qwen)."""

    base_url: str
    """OpenAI-compat base URL. Must include the ``/v1`` suffix where applicable."""

    default_model: str
    """Default model name to pass to ``chat.completions.create``.
    Per-call overrides are possible via the executor constructor."""

    auth_scheme: str = "openai"
    """``openai`` (Bearer token + OpenAI-style payload) for every Phase 1 provider.
    Phase 2 reserves ``anthropic`` and ``gemini`` for native-SDK paths."""

    docs: str = ""
    """Short note — which docs URL the config was derived from, or which quirks
    to know about. Not used programmatically; exists to make future audits of
    this file cheaper than re-Googling every entry."""


# --- Provider registry ------------------------------------------------------
#
# Ordering within this dict is not semantically meaningful — use
# ``RESOLUTION_ORDER`` below to control auto-detect priority. This dict is
# grouped by "who owns the provider" just for human readability.

PROVIDERS: dict[str, ProviderConfig] = {
    # --- Existing (PR 2 baseline) ---------------------------------------
    "nous_portal": ProviderConfig(
        name="nous_portal",
        env_vars=("HERMES_API_KEY", "NOUS_API_KEY"),
        base_url="https://inference-prod.nousresearch.com/v1",
        default_model="nousresearch/hermes-3-llama-3.1-405b",
        docs="Nous Research Portal — original Hermes adapter target from PR 2.",
    ),
    "openrouter": ProviderConfig(
        name="openrouter",
        env_vars=("OPENROUTER_API_KEY",),
        base_url="https://openrouter.ai/api/v1",
        default_model="anthropic/claude-sonnet-4.5",
        docs="OpenRouter — unified OpenAI-compat gateway to hundreds of models. "
             "Useful for A/B testing and as a fallback when a direct provider is down.",
    ),

    # --- Frontier commercial (US) ---------------------------------------
    "openai": ProviderConfig(
        name="openai",
        env_vars=("OPENAI_API_KEY",),
        base_url="https://api.openai.com/v1",
        default_model="gpt-4o",
        docs="OpenAI — canonical OpenAI-compat endpoint. Works out of the box.",
    ),
    "anthropic": ProviderConfig(
        name="anthropic",
        env_vars=("ANTHROPIC_API_KEY",),
        base_url="https://api.anthropic.com",
        default_model="claude-sonnet-4-5",
        auth_scheme="anthropic",
        docs="Anthropic — Phase 2 uses the native Messages API via the official "
             "`anthropic` Python SDK for correct tool calling, vision, and "
             "extended thinking semantics. If the SDK isn't installed in the "
             "workspace image, the executor raises a clear error pointing at "
             "`pip install anthropic>=0.39.0`.",
    ),
    "xai": ProviderConfig(
        name="xai",
        env_vars=("XAI_API_KEY", "GROK_API_KEY"),
        base_url="https://api.x.ai/v1",
        default_model="grok-4",
        docs="xAI — Grok family. OpenAI-compat via api.x.ai/v1.",
    ),
    "gemini": ProviderConfig(
        name="gemini",
        env_vars=("GEMINI_API_KEY", "GOOGLE_API_KEY"),
        base_url="https://generativelanguage.googleapis.com",
        default_model="gemini-2.5-flash",
        auth_scheme="gemini",
        docs="Google Gemini — Phase 2b uses the native generateContent API via "
             "the official `google-genai` Python SDK for correct vision content "
             "blocks, tool/function calling, and system instructions. Phase 1 "
             "used the /v1beta/openai compat shim. If the google-genai package "
             "isn't installed in the workspace image, the executor raises a "
             "clear error pointing at `pip install google-genai>=1.0.0`.",
    ),

    # --- Chinese providers ----------------------------------------------
    "qwen": ProviderConfig(
        name="qwen",
        env_vars=("QWEN_API_KEY", "DASHSCOPE_API_KEY"),
        base_url="https://dashscope-intl.aliyuncs.com/compatible-mode/v1",
        default_model="qwen3-235b-a22b",
        docs="Alibaba Qwen via DashScope international endpoint. OpenAI-compat mode. "
             "For domestic China use dashscope.aliyuncs.com (no -intl).",
    ),
    "glm": ProviderConfig(
        name="glm",
        env_vars=("GLM_API_KEY", "ZHIPU_API_KEY"),
        base_url="https://open.bigmodel.cn/api/paas/v4",
        default_model="glm-4-plus",
        docs="Zhipu AI GLM — open.bigmodel.cn, OpenAI-compat via /api/paas/v4.",
    ),
    "kimi": ProviderConfig(
        name="kimi",
        env_vars=("KIMI_API_KEY", "MOONSHOT_API_KEY"),
        base_url="https://api.moonshot.ai/v1",
        default_model="kimi-k2",
        docs="Moonshot AI Kimi K2 — OpenAI-compat at api.moonshot.ai/v1.",
    ),
    "minimax": ProviderConfig(
        name="minimax",
        env_vars=("MINIMAX_API_KEY",),
        base_url="https://api.minimax.io/v1",
        default_model="MiniMax-M2",
        docs="MiniMax — OpenAI-compat at api.minimax.io/v1. "
             "Note: older base URL api.minimaxi.chat is deprecated.",
    ),
    "deepseek": ProviderConfig(
        name="deepseek",
        env_vars=("DEEPSEEK_API_KEY",),
        base_url="https://api.deepseek.com/v1",
        default_model="deepseek-chat",
        docs="DeepSeek — very cheap, OpenAI-compat at api.deepseek.com/v1.",
    ),

    # --- OSS / alt providers --------------------------------------------
    "groq": ProviderConfig(
        name="groq",
        env_vars=("GROQ_API_KEY",),
        base_url="https://api.groq.com/openai/v1",
        default_model="llama-3.3-70b-versatile",
        docs="Groq LPU inference — very fast, OpenAI-compat at api.groq.com/openai/v1.",
    ),
    "together": ProviderConfig(
        name="together",
        env_vars=("TOGETHER_API_KEY",),
        base_url="https://api.together.xyz/v1",
        default_model="meta-llama/Meta-Llama-3.1-405B-Instruct-Turbo",
        docs="Together AI — OSS model hosting, OpenAI-compat at api.together.xyz/v1.",
    ),
    "fireworks": ProviderConfig(
        name="fireworks",
        env_vars=("FIREWORKS_API_KEY",),
        base_url="https://api.fireworks.ai/inference/v1",
        default_model="accounts/fireworks/models/llama-v3p3-70b-instruct",
        docs="Fireworks AI — OSS model hosting, OpenAI-compat at api.fireworks.ai/inference/v1.",
    ),
    "mistral": ProviderConfig(
        name="mistral",
        env_vars=("MISTRAL_API_KEY",),
        base_url="https://api.mistral.ai/v1",
        default_model="mistral-large-latest",
        docs="Mistral AI — OpenAI-compat at api.mistral.ai/v1.",
    ),
}


# --- Auto-detect resolution order -------------------------------------------
#
# When the caller doesn't specify a provider, resolve_provider() walks this
# list in order and picks the first provider whose env var is set. Order is
# chosen to preserve back-compat (the two original PR-2 providers come first)
# followed by the most likely-to-be-configured commercial APIs.

RESOLUTION_ORDER: tuple[str, ...] = (
    # Back-compat: PR 2 baseline
    "nous_portal",
    "openrouter",
    # Frontier commercial
    "anthropic",
    "openai",
    "gemini",
    "xai",
    # Chinese providers
    "qwen",
    "glm",
    "kimi",
    "minimax",
    "deepseek",
    # OSS / alt
    "groq",
    "mistral",
    "together",
    "fireworks",
)


def resolve_provider(explicit: Optional[str] = None) -> tuple[ProviderConfig, str]:
    """Resolve a provider name to a ``(ProviderConfig, api_key)`` pair.

    Resolution order:

    1. If ``explicit`` is given, look it up in ``PROVIDERS`` and try every
       env var on that provider's config. Raise with a clear message if the
       name is unknown or if all env vars are empty.

    2. Otherwise auto-detect: walk ``RESOLUTION_ORDER`` and return the first
       provider whose env var is set.

    Raises
    ------
    ValueError
        If ``explicit`` is an unknown provider name, if ``explicit`` is a
        known provider but its env vars are all empty, or if no env var is
        set for any provider in auto-detect mode.
    """
    if explicit:
        if explicit not in PROVIDERS:
            raise ValueError(
                f"Unknown Hermes provider: {explicit!r}. "
                f"Available: {sorted(PROVIDERS)}"
            )
        cfg = PROVIDERS[explicit]
        for env in cfg.env_vars:
            val = os.environ.get(env, "").strip()
            if val:
                return cfg, val
        raise ValueError(
            f"Hermes provider {explicit!r} specified but no env var set. "
            f"Tried: {cfg.env_vars}"
        )

    # Auto-detect — first provider with a non-empty env var wins.
    for name in RESOLUTION_ORDER:
        cfg = PROVIDERS[name]
        for env in cfg.env_vars:
            val = os.environ.get(env, "").strip()
            if val:
                return cfg, val

    # Nothing set — raise with the full list so the operator knows every
    # option they have without having to read the source.
    tried = []
    for name in RESOLUTION_ORDER:
        for env in PROVIDERS[name].env_vars:
            tried.append(env)
    raise ValueError(
        "No Hermes provider API key found. Set any one of: " + ", ".join(tried)
    )
