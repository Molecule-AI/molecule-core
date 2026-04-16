"""Hermes escalation ladder — promote to stronger models on transient failure.

Every workspace in the Hermes adapter path has a single pinned model today
(``provider_cfg.default_model`` overridden by ``runtime_config.model`` in
``config.yaml``). That's fine when the pinned model is the best fit, but
it leaves four recurring failure classes unhandled:

1. **Rate limits** (Claude Max saturation, Anthropic 429, OpenAI 429). We're
   currently saturating 3× Claude Max subscriptions — the first 429 is now
   the norm, not the exception.
2. **Transient 5xx** from any provider (overloaded 529, 500, 502, 503).
3. **Context-length exceeded** on the smaller-window model (Haiku has 200k,
   cheaper Gemini flash tiers have less, OpenAI nano/mini have 128k).
4. **Refusal / empty response** from a cheaper tier that the next tier up
   would handle — less common but real in practice.

An escalation ladder is a workspace-configured list of ``LadderRung`` entries
(provider + model). On a qualifying failure, the executor advances to the
next rung and retries the same user_message + history. If the ladder is
exhausted, the last error is raised.

## Config shape

``config.yaml``::

    hermes:
      escalation_ladder:
        - provider: gemini
          model: gemini-2.5-flash      # fast/cheap probe
        - provider: anthropic
          model: claude-haiku-4-5-20251001
        - provider: anthropic
          model: claude-sonnet-4-5-20250929
        - provider: anthropic
          model: claude-opus-4-1-20250805   # frontier rescue

When ``escalation_ladder`` is absent, the executor behaves exactly as before:
one call, one model, errors bubble.

## What this module does NOT do (yet)

- **No uncertainty-driven escalation.** Only transient-failure escalation.
  Promoting on "the answer felt thin" requires a judge pass — follow-up.
- **No streaming partial-result aggregation.** The first rung that succeeds
  returns; we don't splice responses across rungs.
- **No per-workspace budget tracking.** Each escalation is one more paid
  call. Follow-up work (#305 budget cap) handles that.
"""

from __future__ import annotations

import logging
from dataclasses import dataclass
from typing import Optional

logger = logging.getLogger(__name__)


@dataclass(frozen=True)
class LadderRung:
    """One rung on the escalation ladder.

    ``provider`` is a canonical short name from ``providers.PROVIDERS``.
    ``model`` overrides the provider's default for this rung.
    """

    provider: str
    model: str


def parse_ladder(raw: Optional[list]) -> list[LadderRung]:
    """Parse the ``escalation_ladder`` list from ``config.yaml`` into rungs.

    Accepts either dict-shaped entries (``{"provider": ..., "model": ...}``)
    or pre-built LadderRung instances (for programmatic callers). Skips
    malformed entries with a warning rather than raising — a missing rung
    is worse than a noisy one during boot.

    Empty / None / missing input returns an empty list (caller interprets
    as "no ladder configured, single-shot dispatch").
    """
    if not raw:
        return []
    rungs: list[LadderRung] = []
    for i, entry in enumerate(raw):
        if isinstance(entry, LadderRung):
            rungs.append(entry)
            continue
        if not isinstance(entry, dict):
            logger.warning(
                "Hermes ladder: rung %d is not a dict (%r), skipping", i, type(entry).__name__,
            )
            continue
        provider = entry.get("provider")
        model = entry.get("model")
        if not provider or not model:
            logger.warning(
                "Hermes ladder: rung %d missing provider or model (%r), skipping", i, entry,
            )
            continue
        rungs.append(LadderRung(provider=str(provider), model=str(model)))
    return rungs


# Error-type names that indicate a transient failure worth escalating.
# We match on the class name (not the module) so this works regardless of
# whether the workspace imported the new or old anthropic / openai SDK.
# See ``should_escalate`` for the matching logic.
_ESCALATABLE_ERROR_CLASSES = frozenset({
    # openai SDK
    "RateLimitError",       # 429
    "APITimeoutError",      # connect/read timeout
    "APIConnectionError",   # TCP / DNS
    "InternalServerError",  # 500
    # anthropic SDK
    "OverloadedError",      # 529
    "APIStatusError",       # generic 5xx wrapper
    # common across both: network-level errors
    "ConnectionError",
    "Timeout",
    "ReadTimeout",
})

# Error-message substrings that indicate context-length exceeded. These map
# to distinct HTTP 400 responses from each provider rather than a typed
# exception, so we match on substring.
_CONTEXT_LENGTH_MARKERS = (
    "maximum context length",      # openai
    "context_length_exceeded",     # openai error.code
    "prompt is too long",          # anthropic
    "prompt_too_long",             # anthropic error.code
    "context window",              # gemini
)

# Error-message substrings that indicate a transient gateway issue. These
# sometimes come through as generic exceptions without typed classes.
_TRANSIENT_GATEWAY_MARKERS = (
    "502 bad gateway",
    "503 service unavailable",
    "504 gateway timeout",
    "overloaded",
    "please try again",
    "temporarily unavailable",
)

# Error-message substrings that definitively DO NOT qualify for escalation.
# Auth and malformed-payload errors don't get better by retrying on a
# different model — they indicate config / code bugs.
_NON_ESCALATABLE_MARKERS = (
    "invalid api key",
    "authentication_error",
    "401",
    "403",
    "forbidden",
    "permission_denied",
    "unauthorized",
)


def should_escalate(exc: BaseException) -> bool:
    """Decide whether ``exc`` justifies moving to the next ladder rung.

    Returns True when the failure is one of:
    - Rate limit (429 / RateLimitError / OverloadedError)
    - Transient gateway (5xx, overload, timeout, connection reset)
    - Context-length exceeded on the current model

    Returns False for auth, permission, malformed-payload, and other
    config-bug classes — escalating those just wastes the next-tier quota.
    """
    if exc is None:
        return False

    cls_name = exc.__class__.__name__
    msg = str(exc).lower()

    # Hard reject: never escalate auth/permission errors regardless of
    # what the class name says. A wrapped RateLimitError that actually
    # contains "401 Unauthorized" is a config bug, not a rate limit.
    for marker in _NON_ESCALATABLE_MARKERS:
        if marker in msg:
            return False

    if cls_name in _ESCALATABLE_ERROR_CLASSES:
        return True

    for marker in _CONTEXT_LENGTH_MARKERS:
        if marker in msg:
            return True

    for marker in _TRANSIENT_GATEWAY_MARKERS:
        if marker in msg:
            return True

    # Status-code prefixes are a common tell for HTTP-wrapped provider errors.
    if "429" in msg or "529" in msg:
        return True
    if any(code in msg for code in ("500 ", "502 ", "503 ", "504 ")):
        return True

    return False
