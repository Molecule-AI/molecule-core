"""Tests for denylist-based env sanitization — safe_env.py (issue #826 / #827).

Covers:
  (a) SMOLAGENTS_ENV_DENYLIST keys are stripped
  (b) *_API_KEY suffix keys are stripped
  (c) *_TOKEN suffix keys are stripped
  (d) Non-secret keys (PATH, HOME, …) are preserved
  (e) safe_send_message label, truncation, and HTML escaping
"""

from __future__ import annotations

import os
from unittest.mock import MagicMock, patch

import pytest

from adapters.smolagents.safe_env import (
    SMOLAGENTS_ENV_DENYLIST,
    make_safe_env,
)
from adapters.smolagents.send_message_wrapper import safe_send_message


# ---------------------------------------------------------------------------
# make_safe_env — denylist-based
# ---------------------------------------------------------------------------


class TestMakeSafeEnvDenylist:
    """(a) Explicit denylist keys are removed."""

    @pytest.mark.parametrize("key", sorted(SMOLAGENTS_ENV_DENYLIST))
    def test_denylist_key_stripped(self, key: str):
        with patch.dict(os.environ, {key: "secret-value"}, clear=False):
            result = make_safe_env()
        assert key not in result, f"Denylist key {key!r} must be stripped"

    def test_all_denylist_keys_stripped_simultaneously(self):
        secrets = {k: "secret" for k in SMOLAGENTS_ENV_DENYLIST}
        with patch.dict(os.environ, secrets, clear=False):
            result = make_safe_env()
        for key in SMOLAGENTS_ENV_DENYLIST:
            assert key not in result


class TestMakeSafeEnvApiKeySuffix:
    """(b) Keys ending with _API_KEY are stripped."""

    def test_openai_api_key(self):
        with patch.dict(os.environ, {"OPENAI_API_KEY": "sk-openai"}, clear=False):
            assert "OPENAI_API_KEY" not in make_safe_env()

    def test_custom_api_key_suffix(self):
        with patch.dict(os.environ, {"MY_CUSTOM_SERVICE_API_KEY": "abc123"}, clear=False):
            assert "MY_CUSTOM_SERVICE_API_KEY" not in make_safe_env()

    def test_arbitrary_api_key_suffix(self):
        with patch.dict(os.environ, {"FOOBAR_API_KEY": "secret"}, clear=False):
            assert "FOOBAR_API_KEY" not in make_safe_env()


class TestMakeSafeEnvTokenSuffix:
    """(c) Keys ending with _TOKEN are stripped."""

    def test_gh_token(self):
        with patch.dict(os.environ, {"GH_TOKEN": "ghp_secret"}, clear=False):
            assert "GH_TOKEN" not in make_safe_env()

    def test_github_token(self):
        with patch.dict(os.environ, {"GITHUB_TOKEN": "ghp_secret"}, clear=False):
            assert "GITHUB_TOKEN" not in make_safe_env()

    def test_custom_token_suffix(self):
        with patch.dict(os.environ, {"MY_SERVICE_TOKEN": "tok_abc"}, clear=False):
            assert "MY_SERVICE_TOKEN" not in make_safe_env()

    def test_arbitrary_token_suffix(self):
        with patch.dict(os.environ, {"INTERNAL_ACCESS_TOKEN": "secret"}, clear=False):
            assert "INTERNAL_ACCESS_TOKEN" not in make_safe_env()


class TestMakeSafeEnvPreservesNonSecrets:
    """(d) Non-secret keys are preserved."""

    def test_preserves_path(self):
        with patch.dict(os.environ, {"PATH": "/usr/bin:/bin"}, clear=False):
            result = make_safe_env()
        assert result.get("PATH") == "/usr/bin:/bin"

    def test_preserves_home(self):
        with patch.dict(os.environ, {"HOME": "/home/agent"}, clear=False):
            result = make_safe_env()
        assert result.get("HOME") == "/home/agent"

    def test_preserves_workspace_id(self):
        with patch.dict(os.environ, {"WORKSPACE_ID": "ws-abc123"}, clear=False):
            result = make_safe_env()
        assert result.get("WORKSPACE_ID") == "ws-abc123"

    def test_preserves_pythonpath(self):
        with patch.dict(os.environ, {"PYTHONPATH": "/app"}, clear=False):
            result = make_safe_env()
        assert result.get("PYTHONPATH") == "/app"

    def test_preserves_lang(self):
        with patch.dict(os.environ, {"LANG": "en_US.UTF-8"}, clear=False):
            result = make_safe_env()
        assert result.get("LANG") == "en_US.UTF-8"

    def test_does_not_mutate_os_environ(self):
        """make_safe_env must never write back to os.environ."""
        with patch.dict(
            os.environ,
            {"ANTHROPIC_API_KEY": "sk-ant-secret", "PATH": "/usr/bin"},
            clear=False,
        ):
            before = dict(os.environ)
            make_safe_env()
            after = dict(os.environ)
        assert before == after

    def test_returns_dict(self):
        assert isinstance(make_safe_env(), dict)


# ---------------------------------------------------------------------------
# safe_send_message — label, truncation, HTML escaping
# ---------------------------------------------------------------------------


class TestSafeSendMessage:
    def _capture(self):
        """Return a mock send_fn and its captured calls."""
        fn = MagicMock()
        return fn

    def test_label_prefix_added(self):
        fn = self._capture()
        safe_send_message("hello", fn)
        fn.assert_called_once()
        payload = fn.call_args[0][0]
        assert payload.startswith("[smolagents]"), f"Missing label: {payload!r}"

    def test_label_prefix_followed_by_content(self):
        fn = self._capture()
        safe_send_message("world", fn)
        payload = fn.call_args[0][0]
        assert "world" in payload

    def test_truncates_at_2000_chars(self):
        fn = self._capture()
        long_text = "a" * 3000
        safe_send_message(long_text, fn)
        payload = fn.call_args[0][0]
        # The user content portion must be capped; label adds a few chars on top
        # Total len = len("[smolagents] ") + 2000
        assert len(payload) <= len("[smolagents] ") + 2000

    def test_short_message_not_truncated(self):
        fn = self._capture()
        safe_send_message("short", fn)
        payload = fn.call_args[0][0]
        assert "short" in payload

    def test_html_entities_escaped(self):
        fn = self._capture()
        safe_send_message("<script>alert('xss')</script>", fn)
        payload = fn.call_args[0][0]
        assert "<script>" not in payload
        assert "&lt;script&gt;" in payload

    def test_ampersand_escaped(self):
        fn = self._capture()
        safe_send_message("a & b", fn)
        payload = fn.call_args[0][0]
        assert "&amp;" in payload

    def test_double_quote_escaped(self):
        fn = self._capture()
        safe_send_message('say "hello"', fn)
        payload = fn.call_args[0][0]
        assert "&quot;" in payload

    def test_non_str_coerced(self):
        """Non-string input must be coerced to str, not raise."""
        fn = self._capture()
        safe_send_message(42, fn)
        fn.assert_called_once()
        payload = fn.call_args[0][0]
        assert "42" in payload

    def test_send_fn_called_exactly_once(self):
        fn = self._capture()
        safe_send_message("msg", fn)
        assert fn.call_count == 1

    def test_empty_string_sends_label_only(self):
        fn = self._capture()
        safe_send_message("", fn)
        payload = fn.call_args[0][0]
        assert payload.strip() == "[smolagents]"
