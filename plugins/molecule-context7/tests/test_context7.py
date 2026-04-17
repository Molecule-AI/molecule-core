"""Tests for plugins/molecule-context7/skills/context7-docs/scripts/context7.py.

Coverage targets (≥80%)
-----------------------
- _redact_secrets (all 5 patterns) — imported from builtin_tools is re-tested
  in test_memory_redact.py; here we test the local _scrub_response copy.
- _validate_query — length cap, secret-pattern rejection, clean queries.
- Session call counter — increment, cap enforcement, env override, reset.
- resolve_library_id — empty name, mock backend, ToolError propagation.
- query_docs — empty library_id, topic validation, mock backend, scrubbing,
  ToolError propagation.
"""

from __future__ import annotations

import importlib.util
import sys
from pathlib import Path
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

# ---------------------------------------------------------------------------
# Load the module under test by path so we can call private helpers directly.
# ---------------------------------------------------------------------------

_CTX7_PATH = (
    Path(__file__).resolve().parents[1]
    / "skills"
    / "context7-docs"
    / "scripts"
    / "context7.py"
)


def _load_ctx7(monkeypatch=None):
    """Fresh module load — unregisters any cached version first."""
    # Remove stale cached module so each test that calls this gets a clean state.
    for key in list(sys.modules.keys()):
        if "context7_tools" in key:
            del sys.modules[key]

    spec = importlib.util.spec_from_file_location("context7_tools", _CTX7_PATH)
    mod = importlib.util.module_from_spec(spec)
    sys.modules["context7_tools"] = mod
    spec.loader.exec_module(mod)
    return mod


@pytest.fixture()
def ctx7(monkeypatch):
    """Load a fresh context7 module with no API key and reset counter."""
    monkeypatch.delenv("CONTEXT7_API_KEY", raising=False)
    monkeypatch.delenv("CONTEXT7_MAX_CALLS_PER_SESSION", raising=False)
    mod = _load_ctx7()
    mod._reset_counter()
    yield mod
    mod._reset_counter()


# ---------------------------------------------------------------------------
# _scrub_response — response secret scrubbing (C1)
# ---------------------------------------------------------------------------


class TestScrubResponse:
    def test_redacts_ctx7_token(self, ctx7):
        text = "Key: ctx7_abcDEF12345678"
        assert ctx7._scrub_response(text) == "Key: [REDACTED]"

    def test_redacts_sk_openai_key(self, ctx7):
        text = "Authorization: sk-abcdefghij1234567890xyz"
        assert ctx7._scrub_response(text) == "Authorization: [REDACTED]"

    def test_redacts_github_pat(self, ctx7):
        text = f"Token: ghp_{'A' * 36}"
        assert ctx7._scrub_response(text) == "Token: [REDACTED]"

    def test_redacts_bearer_token(self, ctx7):
        text = "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.payload"
        result = ctx7._scrub_response(text)
        assert "[REDACTED]" in result
        assert "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9" not in result

    def test_redacts_env_style_api_key(self, ctx7):
        text = "export OPENAI_API_KEY=sk123abc456def789ghi"
        result = ctx7._scrub_response(text)
        assert "[REDACTED]" in result

    def test_clean_text_unchanged(self, ctx7):
        text = "React hooks documentation: useState, useEffect, useContext"
        assert ctx7._scrub_response(text) == text

    def test_empty_string(self, ctx7):
        assert ctx7._scrub_response("") == ""

    def test_multiple_secrets_all_redacted(self, ctx7):
        text = f"ctx7_abcdef12 and ghp_{'B' * 36}"
        result = ctx7._scrub_response(text)
        assert result.count("[REDACTED]") == 2


# ---------------------------------------------------------------------------
# _validate_query — input validation (C4)
# ---------------------------------------------------------------------------


class TestValidateQuery:
    def test_clean_short_query_passes(self, ctx7):
        ctx7._validate_query("React hooks overview")  # should not raise

    def test_empty_string_passes(self, ctx7):
        ctx7._validate_query("")  # empty is valid (topic is optional)

    def test_exactly_200_chars_passes(self, ctx7):
        ctx7._validate_query("x" * 200)

    def test_201_chars_raises(self, ctx7):
        with pytest.raises(ctx7.ToolError, match="too long"):
            ctx7._validate_query("x" * 201)

    def test_ctx7_key_in_query_rejected(self, ctx7):
        with pytest.raises(ctx7.ToolError, match="secret-like"):
            ctx7._validate_query("ctx7_abcDEF123456789 hooks")

    def test_bearer_token_in_query_rejected(self, ctx7):
        with pytest.raises(ctx7.ToolError, match="secret-like"):
            ctx7._validate_query("Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.stuff")

    def test_sk_key_in_query_rejected(self, ctx7):
        with pytest.raises(ctx7.ToolError, match="secret-like"):
            ctx7._validate_query("sk-abcdefghijklmnopqrstu hooks for React")

    def test_env_key_in_query_rejected(self, ctx7):
        with pytest.raises(ctx7.ToolError, match="secret-like"):
            ctx7._validate_query("OPENAI_API_KEY=sk123abcdefghijklmnop hooks")


# ---------------------------------------------------------------------------
# Session call counter (C5)
# ---------------------------------------------------------------------------


class TestSessionCallCounter:
    def test_first_call_succeeds(self, ctx7, monkeypatch):
        monkeypatch.setenv("CONTEXT7_MAX_CALLS_PER_SESSION", "3")
        ctx7._reset_counter()
        ctx7._increment_and_check()  # call 1 — should not raise

    def test_at_cap_succeeds(self, ctx7, monkeypatch):
        monkeypatch.setenv("CONTEXT7_MAX_CALLS_PER_SESSION", "2")
        ctx7._reset_counter()
        ctx7._increment_and_check()  # call 1
        ctx7._increment_and_check()  # call 2 — at cap, still ok

    def test_over_cap_raises(self, ctx7, monkeypatch):
        monkeypatch.setenv("CONTEXT7_MAX_CALLS_PER_SESSION", "2")
        ctx7._reset_counter()
        ctx7._increment_and_check()
        ctx7._increment_and_check()
        with pytest.raises(ctx7.ToolError, match="session call limit"):
            ctx7._increment_and_check()  # call 3 — exceeds cap of 2

    def test_reset_allows_new_calls(self, ctx7, monkeypatch):
        monkeypatch.setenv("CONTEXT7_MAX_CALLS_PER_SESSION", "1")
        ctx7._reset_counter()
        ctx7._increment_and_check()  # call 1
        # would raise on call 2 — but after reset it should pass
        ctx7._reset_counter()
        ctx7._increment_and_check()  # call 1 again

    def test_default_cap_is_50(self, ctx7, monkeypatch):
        monkeypatch.delenv("CONTEXT7_MAX_CALLS_PER_SESSION", raising=False)
        assert ctx7._max_calls() == 50

    def test_env_override_parsed(self, ctx7, monkeypatch):
        monkeypatch.setenv("CONTEXT7_MAX_CALLS_PER_SESSION", "10")
        assert ctx7._max_calls() == 10

    def test_invalid_env_falls_back_to_default(self, ctx7, monkeypatch):
        monkeypatch.setenv("CONTEXT7_MAX_CALLS_PER_SESSION", "not-a-number")
        assert ctx7._max_calls() == 50


# ---------------------------------------------------------------------------
# resolve_library_id
# ---------------------------------------------------------------------------


class TestResolveLibraryId:
    @pytest.mark.asyncio
    async def test_empty_name_returns_error(self, ctx7):
        result = await ctx7.resolve_library_id("")
        assert "error" in result

    @pytest.mark.asyncio
    async def test_whitespace_only_returns_error(self, ctx7):
        result = await ctx7.resolve_library_id("   ")
        assert "error" in result

    @pytest.mark.asyncio
    async def test_mock_backend_used_without_key(self, ctx7, monkeypatch):
        monkeypatch.delenv("CONTEXT7_API_KEY", raising=False)
        result = await ctx7.resolve_library_id("react")
        assert result["mock"] is True
        assert "react" in result["library_id"]

    @pytest.mark.asyncio
    async def test_mock_backend_normalises_name(self, ctx7, monkeypatch):
        monkeypatch.delenv("CONTEXT7_API_KEY", raising=False)
        result = await ctx7.resolve_library_id("My Library")
        assert "my-library" in result["library_id"]

    @pytest.mark.asyncio
    async def test_counter_incremented(self, ctx7, monkeypatch):
        monkeypatch.delenv("CONTEXT7_API_KEY", raising=False)
        ctx7._reset_counter()
        await ctx7.resolve_library_id("react")
        assert ctx7._session_call_count == 1

    @pytest.mark.asyncio
    async def test_counter_raises_when_exceeded(self, ctx7, monkeypatch):
        monkeypatch.delenv("CONTEXT7_API_KEY", raising=False)
        monkeypatch.setenv("CONTEXT7_MAX_CALLS_PER_SESSION", "1")
        ctx7._reset_counter()
        await ctx7.resolve_library_id("react")  # call 1
        # Call 2 should breach the cap.
        with pytest.raises(ctx7.ToolError, match="session call limit"):
            await ctx7.resolve_library_id("fastapi")

    @pytest.mark.asyncio
    async def test_live_response_scrubbed(self, ctx7, monkeypatch):
        """ctx7_* tokens in a live response must be redacted."""
        monkeypatch.setenv("CONTEXT7_API_KEY", "ctx7_testkey12345678")
        mock_result = {
            "library_id": "ctx7_leaked_abcdef12345",
            "name": "react",
        }
        with patch.object(ctx7, "_live_resolve", new=AsyncMock(return_value=mock_result)):
            result = await ctx7.resolve_library_id("react")
        assert "[REDACTED]" in result["library_id"]
        assert "ctx7_" not in result["library_id"]

    @pytest.mark.asyncio
    async def test_live_exception_returns_error(self, ctx7, monkeypatch):
        monkeypatch.setenv("CONTEXT7_API_KEY", "ctx7_testkey12345678")
        with patch.object(ctx7, "_live_resolve", new=AsyncMock(side_effect=RuntimeError("network"))):
            result = await ctx7.resolve_library_id("react")
        assert "error" in result


# ---------------------------------------------------------------------------
# query_docs
# ---------------------------------------------------------------------------


class TestQueryDocs:
    @pytest.mark.asyncio
    async def test_empty_library_id_returns_error(self, ctx7):
        result = await ctx7.query_docs("", topic="hooks")
        assert "error" in result

    @pytest.mark.asyncio
    async def test_whitespace_library_id_returns_error(self, ctx7):
        result = await ctx7.query_docs("   ")
        assert "error" in result

    @pytest.mark.asyncio
    async def test_topic_too_long_returns_error(self, ctx7):
        result = await ctx7.query_docs("/facebook/react", topic="x" * 201)
        assert "error" in result
        assert "long" in result["error"].lower()

    @pytest.mark.asyncio
    async def test_topic_with_secret_returns_error(self, ctx7):
        result = await ctx7.query_docs(
            "/facebook/react",
            topic="ctx7_abcdef12345678 hooks",
        )
        assert "error" in result
        assert "secret" in result["error"].lower()

    @pytest.mark.asyncio
    async def test_mock_backend_used_without_key(self, ctx7, monkeypatch):
        monkeypatch.delenv("CONTEXT7_API_KEY", raising=False)
        result = await ctx7.query_docs("/facebook/react", topic="hooks")
        assert result["mock"] is True
        assert result["library_id"] == "/facebook/react"

    @pytest.mark.asyncio
    async def test_mock_backend_empty_topic(self, ctx7, monkeypatch):
        monkeypatch.delenv("CONTEXT7_API_KEY", raising=False)
        result = await ctx7.query_docs("/tiangolo/fastapi")
        assert result["mock"] is True

    @pytest.mark.asyncio
    async def test_counter_incremented(self, ctx7, monkeypatch):
        monkeypatch.delenv("CONTEXT7_API_KEY", raising=False)
        ctx7._reset_counter()
        await ctx7.query_docs("/facebook/react")
        assert ctx7._session_call_count == 1

    @pytest.mark.asyncio
    async def test_counter_raises_when_exceeded(self, ctx7, monkeypatch):
        monkeypatch.delenv("CONTEXT7_API_KEY", raising=False)
        monkeypatch.setenv("CONTEXT7_MAX_CALLS_PER_SESSION", "1")
        ctx7._reset_counter()
        await ctx7.query_docs("/facebook/react")  # call 1
        with pytest.raises(ctx7.ToolError, match="session call limit"):
            await ctx7.query_docs("/tiangolo/fastapi")  # call 2

    @pytest.mark.asyncio
    async def test_live_content_scrubbed(self, ctx7, monkeypatch):
        """ctx7_* tokens in API response content must be redacted."""
        monkeypatch.setenv("CONTEXT7_API_KEY", "ctx7_testkey12345678")
        mock_result = {
            "library_id": "/facebook/react",
            "topic": "hooks",
            "tokens_used": 100,
            "content": "See ctx7_leaked_abcdef12345 for more info.",
        }
        with patch.object(ctx7, "_live_query", new=AsyncMock(return_value=mock_result)):
            result = await ctx7.query_docs("/facebook/react", topic="hooks")
        assert "[REDACTED]" in result["content"]
        assert "ctx7_" not in result["content"]

    @pytest.mark.asyncio
    async def test_live_exception_returns_error(self, ctx7, monkeypatch):
        monkeypatch.setenv("CONTEXT7_API_KEY", "ctx7_testkey12345678")
        with patch.object(ctx7, "_live_query", new=AsyncMock(side_effect=RuntimeError("timeout"))):
            result = await ctx7.query_docs("/facebook/react")
        assert "error" in result

    @pytest.mark.asyncio
    async def test_default_tokens_is_5000(self, ctx7, monkeypatch):
        monkeypatch.delenv("CONTEXT7_API_KEY", raising=False)
        captured: list = []

        original_mock = ctx7._mock_query

        def _capture(library_id, topic, tokens):
            captured.append(tokens)
            return original_mock(library_id, topic, tokens)

        with patch.object(ctx7, "_mock_query", side_effect=_capture):
            await ctx7.query_docs("/facebook/react")
        assert captured == [5000]
