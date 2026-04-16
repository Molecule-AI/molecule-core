"""Integration-ish tests for the Hermes executor's escalation behaviour.

These tests exercise ``_do_inference`` against a mocked ``_dispatch``
to prove that:
- No-ladder path is a single call (original behaviour)
- Ladder path retries on escalatable errors
- Ladder path stops early on non-escalatable errors
- Ladder path raises the last error when every rung fails
- Successful rung logs the recovery and returns

No network calls, no provider SDKs. If this ever starts calling real
providers, that's a test-isolation regression worth flagging.
"""
from __future__ import annotations

import asyncio
import sys
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from adapters.hermes.escalation import LadderRung  # noqa: E402
from adapters.hermes.executor import HermesA2AExecutor  # noqa: E402
from adapters.hermes.providers import PROVIDERS  # noqa: E402


class _FakeRateLimitError(Exception):
    pass
_FakeRateLimitError.__name__ = "RateLimitError"


def _make_executor(monkeypatch, dispatch_behaviour, ladder=None):
    """Build an executor with a mocked ``_dispatch``.

    ``dispatch_behaviour`` is a callable that receives (cfg, model, user_msg,
    history, system_prompt) and returns a string OR raises. Use this to
    simulate success / failure per rung.
    """
    cfg = PROVIDERS["anthropic"]
    ex = HermesA2AExecutor(
        provider_cfg=cfg,
        api_key="test-key",
        model="claude-haiku-4-5-20251001",
        escalation_ladder=ladder,
    )

    calls: list[tuple[str, str]] = []

    async def fake_dispatch(cfg, model, user_msg, history, system_prompt):
        calls.append((cfg.name, model))
        result = dispatch_behaviour(cfg.name, model, user_msg, history, system_prompt)
        if isinstance(result, BaseException):
            raise result
        return result

    monkeypatch.setattr(ex, "_dispatch", fake_dispatch)
    return ex, calls


def _run(coro):
    return asyncio.get_event_loop().run_until_complete(coro) if not asyncio._get_running_loop() else asyncio.run(coro)


def test_no_ladder_single_call(monkeypatch):
    ex, calls = _make_executor(monkeypatch, lambda *_: "hello", ladder=None)
    reply = asyncio.run(ex._do_inference("test"))
    assert reply == "hello"
    assert calls == [("anthropic", "claude-haiku-4-5-20251001")]


def test_ladder_not_triggered_on_success(monkeypatch):
    # Ladder configured, but first attempt succeeds — ladder never engaged.
    ladder = [
        {"provider": "openai", "model": "gpt-4o-mini"},
        {"provider": "anthropic", "model": "claude-opus-4-1-20250805"},
    ]
    ex, calls = _make_executor(monkeypatch, lambda *_: "fast reply", ladder=ladder)
    reply = asyncio.run(ex._do_inference("test"))
    assert reply == "fast reply"
    assert len(calls) == 1
    assert calls[0] == ("anthropic", "claude-haiku-4-5-20251001")  # pinned (haiku) wins


def test_ladder_escalates_on_rate_limit(monkeypatch):
    # First rung rate-limits, second rung (opus) succeeds.
    attempt = {"n": 0}

    def behaviour(provider, model, *_):
        attempt["n"] += 1
        if attempt["n"] == 1:
            return _FakeRateLimitError("429 rate_limit_exceeded on anthropic")
        return f"escalated reply from {provider}:{model}"

    ladder = [
        {"provider": "anthropic", "model": "claude-opus-4-1-20250805"},
    ]
    ex, calls = _make_executor(monkeypatch, behaviour, ladder=ladder)
    reply = asyncio.run(ex._do_inference("test"))
    assert "escalated reply" in reply
    # Two attempts: pinned haiku (failed), then opus (succeeded).
    assert [model for _, model in calls] == [
        "claude-haiku-4-5-20251001",
        "claude-opus-4-1-20250805",
    ]


def test_ladder_stops_on_non_escalatable_error(monkeypatch):
    # First rung returns a 401 — ladder should NOT retry, should raise.
    def behaviour(*_):
        return RuntimeError("401 Unauthorized invalid api key")

    ladder = [{"provider": "anthropic", "model": "claude-opus-4-1-20250805"}]
    ex, calls = _make_executor(monkeypatch, behaviour, ladder=ladder)

    with pytest.raises(RuntimeError, match="401"):
        asyncio.run(ex._do_inference("test"))

    # Only one attempt — non-escalatable error stopped the walk.
    assert len(calls) == 1


def test_ladder_raises_last_error_when_all_rungs_fail(monkeypatch):
    def behaviour(*_):
        return _FakeRateLimitError("429 across the board")

    ladder = [
        {"provider": "anthropic", "model": "claude-opus-4-1-20250805"},
    ]
    ex, calls = _make_executor(monkeypatch, behaviour, ladder=ladder)

    with pytest.raises(_FakeRateLimitError):
        asyncio.run(ex._do_inference("test"))

    # Both rungs attempted (pinned + one from ladder).
    assert len(calls) == 2


def test_ladder_skips_unknown_provider(monkeypatch):
    # A misconfigured rung with a non-existent provider is logged + skipped;
    # ladder still walks remaining rungs.
    def behaviour(provider, *_):
        if provider == "anthropic":
            return _FakeRateLimitError("first rung rate limit")
        return f"ok from {provider}"

    ladder = [
        {"provider": "totally_made_up", "model": "fake-1"},  # should be skipped
        {"provider": "anthropic", "model": "claude-opus-4-1-20250805"},
    ]
    ex, calls = _make_executor(monkeypatch, behaviour, ladder=ladder)

    # First attempt uses the pinned (haiku) which raises, then skips
    # totally_made_up, then reaches opus. Because behaviour returns ok for
    # provider==anthropic, the opus rung also fails (same provider). Assert
    # the skip happened (call count reflects 2 real attempts, not 3).
    with pytest.raises(_FakeRateLimitError):
        asyncio.run(ex._do_inference("test"))
    assert len(calls) == 2  # pinned + opus (totally_made_up skipped)
