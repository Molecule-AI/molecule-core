"""Tests for Hermes escalation-ladder classification and config parsing.

The truth table in ``should_escalate`` is the single chokepoint that
decides whether an inference failure wastes the next ladder rung's
quota or triggers a useful retry. These tests pin that table against
real exception shapes from anthropic / openai / google-genai SDKs and
the wrapped-error strings we've observed in platform logs.
"""
from __future__ import annotations

import sys
from pathlib import Path

import pytest

# Make the workspace-template/ modules importable without installing.
sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from adapters.hermes.escalation import (  # noqa: E402
    LadderRung,
    parse_ladder,
    should_escalate,
)


# --------------------------------------------------------------------------
# parse_ladder
# --------------------------------------------------------------------------

def test_parse_ladder_empty_returns_empty():
    assert parse_ladder(None) == []
    assert parse_ladder([]) == []


def test_parse_ladder_accepts_dicts():
    raw = [
        {"provider": "gemini", "model": "gemini-2.5-flash"},
        {"provider": "anthropic", "model": "claude-opus-4-1-20250805"},
    ]
    rungs = parse_ladder(raw)
    assert len(rungs) == 2
    assert rungs[0] == LadderRung("gemini", "gemini-2.5-flash")
    assert rungs[1] == LadderRung("anthropic", "claude-opus-4-1-20250805")


def test_parse_ladder_passes_through_rung_instances():
    # Programmatic callers can pass already-constructed rungs.
    existing = LadderRung("openai", "gpt-4o-mini")
    rungs = parse_ladder([existing])
    assert rungs == [existing]


def test_parse_ladder_skips_malformed_entries():
    # Missing model / missing provider / wrong type — all skipped with
    # a warning, not raised. A missing rung is less bad than a boot fail.
    raw = [
        {"provider": "gemini"},           # no model
        {"model": "gpt-4o"},              # no provider
        "not a dict",                     # wrong type
        {"provider": "anthropic", "model": "claude-opus-4-1-20250805"},  # good
    ]
    rungs = parse_ladder(raw)
    assert len(rungs) == 1
    assert rungs[0].provider == "anthropic"


# --------------------------------------------------------------------------
# should_escalate — truth table
# --------------------------------------------------------------------------

class _FakeRateLimitError(Exception):
    """Stand-in with the same class name the openai SDK uses (rate limits)."""
    pass
_FakeRateLimitError.__name__ = "RateLimitError"


class _FakeOverloadedError(Exception):
    """Stand-in for anthropic.OverloadedError (HTTP 529)."""
    pass
_FakeOverloadedError.__name__ = "OverloadedError"


class _FakeAPITimeoutError(Exception):
    pass
_FakeAPITimeoutError.__name__ = "APITimeoutError"


class _FakeAPIConnectionError(Exception):
    pass
_FakeAPIConnectionError.__name__ = "APIConnectionError"


class _FakeInternalServerError(Exception):
    pass
_FakeInternalServerError.__name__ = "InternalServerError"


@pytest.mark.parametrize("exc,expected", [
    # --- Escalatable: typed rate-limit / overload / timeout classes ---
    (_FakeRateLimitError("rate_limit_exceeded on gpt-4o"), True),
    (_FakeOverloadedError("overloaded_error"), True),
    (_FakeAPITimeoutError("Request timed out."), True),
    (_FakeAPIConnectionError("Connection error."), True),
    (_FakeInternalServerError("Internal server error 500."), True),

    # --- Escalatable: context-length exceeded on current model ---
    (ValueError("This model's maximum context length is 200000 tokens. However, your messages resulted in ..."), True),
    (RuntimeError("error: context_length_exceeded"), True),
    (RuntimeError("prompt is too long: 210000 tokens"), True),
    (RuntimeError("error.type: prompt_too_long"), True),
    (RuntimeError("exceeds model context window of 1048576"), True),

    # --- Escalatable: gateway markers (HTTP-wrapped) ---
    (RuntimeError("Upstream 502 Bad Gateway"), True),
    (RuntimeError("503 Service Unavailable"), True),
    (RuntimeError("Service is temporarily unavailable, please try again."), True),
    (RuntimeError("Anthropic API is overloaded."), True),

    # --- Escalatable: status-code substrings ---
    (RuntimeError("HTTP 429 Too Many Requests"), True),
    (RuntimeError("HTTP 529 Overloaded"), True),

    # --- NOT escalatable: auth / permission (config bugs, wasting quota) ---
    (RuntimeError("401 Unauthorized — invalid api key"), False),
    (RuntimeError("403 Forbidden: permission_denied"), False),
    (RuntimeError("authentication_error: invalid_api_key"), False),

    # --- NOT escalatable: auth-wrapped rate-limit (priority = hard-reject auth) ---
    # If we see '401' + rate-limit markers simultaneously, prefer not escalating
    # because the underlying 401 won't get better on a different model.
    (_FakeRateLimitError("RateLimitError wrapping 401 Unauthorized"), False),

    # --- NOT escalatable: unrelated errors ---
    (ValueError("bad config"), False),
    (KeyError("missing key"), False),
    (None, False),
])
def test_should_escalate_truth_table(exc, expected):
    assert should_escalate(exc) is expected


def test_should_escalate_case_insensitive():
    # We lowercase the message before substring matching so "OVERLOADED"
    # from one provider and "overloaded" from another both match.
    assert should_escalate(RuntimeError("SERVICE OVERLOADED")) is True
    assert should_escalate(RuntimeError("503 SERVICE UNAVAILABLE")) is True
