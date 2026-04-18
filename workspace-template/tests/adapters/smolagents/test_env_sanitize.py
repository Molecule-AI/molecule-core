"""Tests for allowlist-based env sanitization (issue #826 — C3 CRITICAL).

All tests patch os.environ directly — the module under test must never
mutate the real process env outside of SafeLocalPythonExecutor.__call__,
and even there it must restore the original env on exit.
"""

from __future__ import annotations

import os
import threading
from typing import Any
from unittest.mock import MagicMock, patch

import pytest

# Import directly from submodule to avoid any sys.modules stub side-effects
from adapters.smolagents.env_sanitize import (
    SafeLocalPythonExecutor,
    _BANNED_IMPORTS,
    _BASELINE_SAFE_IMPORTS,
    _SAFE_ENV_ALLOWLIST,
    make_safe_env,
)


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


class _MockInner:
    """Captures the code string passed to it; returns a configurable result."""

    def __init__(self, return_value: Any = None):
        self.calls: list[str] = []
        self.return_value = return_value

    def __call__(self, code: str, *args: Any, **kwargs: Any) -> Any:
        self.calls.append(code)
        return self.return_value


# ---------------------------------------------------------------------------
# make_safe_env() — pure function tests (os.environ never mutated)
# ---------------------------------------------------------------------------


class TestMakeSafeEnv:
    def test_strips_anthropic_api_key(self):
        with patch.dict(os.environ, {"ANTHROPIC_API_KEY": "sk-ant-secret"}, clear=False):
            result = make_safe_env()
        assert "ANTHROPIC_API_KEY" not in result

    def test_strips_gh_token(self):
        with patch.dict(os.environ, {"GH_TOKEN": "ghp_secret"}, clear=False):
            result = make_safe_env()
        assert "GH_TOKEN" not in result

    def test_strips_openai_api_key(self):
        with patch.dict(os.environ, {"OPENAI_API_KEY": "sk-openai"}, clear=False):
            result = make_safe_env()
        assert "OPENAI_API_KEY" not in result

    def test_strips_database_url(self):
        with patch.dict(os.environ, {"DATABASE_URL": "postgres://secret"}, clear=False):
            result = make_safe_env()
        assert "DATABASE_URL" not in result

    def test_strips_redis_url(self):
        with patch.dict(os.environ, {"REDIS_URL": "redis://secret"}, clear=False):
            result = make_safe_env()
        assert "REDIS_URL" not in result

    def test_strips_aws_access_key(self):
        with patch.dict(os.environ, {"AWS_ACCESS_KEY_ID": "AKIAIOSFODNN7EXAMPLE"}, clear=False):
            result = make_safe_env()
        assert "AWS_ACCESS_KEY_ID" not in result

    def test_strips_slack_token(self):
        with patch.dict(os.environ, {"SLACK_BOT_TOKEN": "xoxb-secret"}, clear=False):
            result = make_safe_env()
        assert "SLACK_BOT_TOKEN" not in result

    def test_strips_generic_password(self):
        with patch.dict(os.environ, {"DB_PASSWORD": "hunter2"}, clear=False):
            result = make_safe_env()
        assert "DB_PASSWORD" not in result

    def test_strips_generic_secret(self):
        with patch.dict(os.environ, {"JWT_SECRET": "supersecret"}, clear=False):
            result = make_safe_env()
        assert "JWT_SECRET" not in result

    def test_passes_path(self):
        with patch.dict(os.environ, {"PATH": "/usr/bin:/bin"}, clear=False):
            result = make_safe_env()
        assert result.get("PATH") == "/usr/bin:/bin"

    def test_passes_home(self):
        with patch.dict(os.environ, {"HOME": "/root"}, clear=False):
            result = make_safe_env()
        assert result.get("HOME") == "/root"

    def test_passes_lang(self):
        with patch.dict(os.environ, {"LANG": "en_US.UTF-8"}, clear=False):
            result = make_safe_env()
        assert result.get("LANG") == "en_US.UTF-8"

    def test_passes_pythonpath(self):
        with patch.dict(os.environ, {"PYTHONPATH": "/app"}, clear=False):
            result = make_safe_env()
        assert result.get("PYTHONPATH") == "/app"

    def test_passes_workspace_id(self):
        with patch.dict(os.environ, {"WORKSPACE_ID": "ws-123"}, clear=False):
            result = make_safe_env()
        assert result.get("WORKSPACE_ID") == "ws-123"

    def test_passes_workspace_name(self):
        with patch.dict(os.environ, {"WORKSPACE_NAME": "my-agent"}, clear=False):
            result = make_safe_env()
        assert result.get("WORKSPACE_NAME") == "my-agent"

    def test_passes_platform_url(self):
        with patch.dict(os.environ, {"PLATFORM_URL": "http://platform:8080"}, clear=False):
            result = make_safe_env()
        assert result.get("PLATFORM_URL") == "http://platform:8080"

    def test_does_not_mutate_os_environ(self):
        """make_safe_env() must be a pure read — os.environ unchanged after call."""
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
        result = make_safe_env()
        assert isinstance(result, dict)

    def test_extra_allowed_via_parameter(self):
        with patch.dict(os.environ, {"MY_SAFE_VAR": "value"}, clear=False):
            result = make_safe_env(extra_allowed=["MY_SAFE_VAR"])
        assert result.get("MY_SAFE_VAR") == "value"

    def test_extra_allowed_via_env_var(self):
        with patch.dict(
            os.environ,
            {
                "SMOLAGENTS_ENV_EXTRA_ALLOWLIST": "REGION,CLUSTER_NAME",
                "REGION": "us-east-1",
                "CLUSTER_NAME": "prod",
                "ANTHROPIC_API_KEY": "sk-ant-secret",
            },
            clear=False,
        ):
            result = make_safe_env()
        assert result.get("REGION") == "us-east-1"
        assert result.get("CLUSTER_NAME") == "prod"
        assert "ANTHROPIC_API_KEY" not in result

    def test_extra_allowed_env_var_is_case_normalized(self):
        """Names in SMOLAGENTS_ENV_EXTRA_ALLOWLIST are uppercased automatically."""
        with patch.dict(
            os.environ,
            {"SMOLAGENTS_ENV_EXTRA_ALLOWLIST": "my_safe_var", "MY_SAFE_VAR": "hello"},
            clear=False,
        ):
            result = make_safe_env()
        assert result.get("MY_SAFE_VAR") == "hello"


# ---------------------------------------------------------------------------
# SafeLocalPythonExecutor — allowlist enforcement during execution
# ---------------------------------------------------------------------------


class TestSafeLocalPythonExecutorAllowlist:
    """Core security guarantee: secrets absent from os.environ during execution."""

    def test_secret_absent_during_execution_anthropic(self):
        """Injected ANTHROPIC_API_KEY must not be visible to executed code."""
        captured_env: dict = {}

        def _mock_inner(code: str, *args, **kwargs):
            # Simulate what agent code would see via os.environ
            captured_env.update(os.environ.copy())
            return ""

        executor = SafeLocalPythonExecutor(_inner=_mock_inner)

        with patch.dict(os.environ, {"ANTHROPIC_API_KEY": "sk-ant-secret"}, clear=False):
            executor("import os; os.environ.get('ANTHROPIC_API_KEY', '')")

        assert "ANTHROPIC_API_KEY" not in captured_env

    def test_secret_absent_during_execution_gh_token(self):
        captured_env: dict = {}

        def _mock_inner(code: str, *args, **kwargs):
            captured_env.update(os.environ.copy())
            return ""

        executor = SafeLocalPythonExecutor(_inner=_mock_inner)

        with patch.dict(os.environ, {"GH_TOKEN": "ghp_secret"}, clear=False):
            executor("import os; os.environ.get('GH_TOKEN', '')")

        assert "GH_TOKEN" not in captured_env

    def test_secret_absent_during_execution_database_url(self):
        captured_env: dict = {}

        def _mock_inner(code: str, *args, **kwargs):
            captured_env.update(os.environ.copy())
            return ""

        executor = SafeLocalPythonExecutor(_inner=_mock_inner)

        with patch.dict(os.environ, {"DATABASE_URL": "postgres://secret"}, clear=False):
            executor("code")

        assert "DATABASE_URL" not in captured_env

    def test_secret_absent_during_execution_openai_key(self):
        captured_env: dict = {}

        def _mock_inner(code: str, *args, **kwargs):
            captured_env.update(os.environ.copy())

        executor = SafeLocalPythonExecutor(_inner=_mock_inner)

        with patch.dict(os.environ, {"OPENAI_API_KEY": "sk-openai"}, clear=False):
            executor("code")

        assert "OPENAI_API_KEY" not in captured_env

    def test_multiple_secrets_all_absent(self):
        """All secrets must be stripped simultaneously, not just one."""
        captured_env: dict = {}

        def _mock_inner(code: str, *args, **kwargs):
            captured_env.update(os.environ.copy())

        executor = SafeLocalPythonExecutor(_inner=_mock_inner)

        secrets = {
            "ANTHROPIC_API_KEY": "sk-ant",
            "GH_TOKEN": "ghp_",
            "OPENAI_API_KEY": "sk-open",
            "DATABASE_URL": "postgres://",
            "REDIS_URL": "redis://",
            "SLACK_BOT_TOKEN": "xoxb-",
            "JWT_SECRET": "secret",
            "DB_PASSWORD": "pass",
        }

        with patch.dict(os.environ, secrets, clear=False):
            executor("code")

        for key in secrets:
            assert key not in captured_env, f"{key!r} was visible during execution"

    def test_safe_vars_present_during_execution(self):
        """Allowlisted variables must remain visible during execution."""
        captured_env: dict = {}

        def _mock_inner(code: str, *args, **kwargs):
            captured_env.update(os.environ.copy())

        executor = SafeLocalPythonExecutor(_inner=_mock_inner)

        with patch.dict(
            os.environ,
            {
                "PATH": "/usr/bin:/bin",
                "WORKSPACE_ID": "ws-abc",
                "PYTHONPATH": "/app",
                "ANTHROPIC_API_KEY": "sk-ant-secret",
            },
            clear=False,
        ):
            executor("code")

        assert captured_env.get("PATH") == "/usr/bin:/bin"
        assert captured_env.get("WORKSPACE_ID") == "ws-abc"
        assert captured_env.get("PYTHONPATH") == "/app"

    def test_env_restored_after_execution(self):
        """os.environ must be fully restored after __call__ returns."""
        executor = SafeLocalPythonExecutor(_inner=_MockInner())

        with patch.dict(
            os.environ,
            {"ANTHROPIC_API_KEY": "sk-ant-secret", "PATH": "/usr/bin"},
            clear=False,
        ):
            env_before = dict(os.environ)
            executor("code")
            env_after = dict(os.environ)

        assert env_before == env_after

    def test_env_restored_after_exception(self):
        """os.environ must be restored even if the inner executor raises."""

        def _raises(code: str, *args, **kwargs):
            raise RuntimeError("boom")

        executor = SafeLocalPythonExecutor(_inner=_raises)

        with patch.dict(
            os.environ,
            {"ANTHROPIC_API_KEY": "sk-ant-secret"},
            clear=False,
        ):
            env_before = dict(os.environ)
            with pytest.raises(RuntimeError, match="boom"):
                executor("code")
            env_after = dict(os.environ)

        assert env_before == env_after

    def test_returns_inner_result(self):
        mock_inner = _MockInner(return_value="hello world")
        executor = SafeLocalPythonExecutor(_inner=mock_inner)
        result = executor("some code")
        assert result == "hello world"

    def test_passes_code_to_inner(self):
        mock_inner = _MockInner()
        executor = SafeLocalPythonExecutor(_inner=mock_inner)
        executor("print('hi')")
        assert mock_inner.calls == ["print('hi')"]


# ---------------------------------------------------------------------------
# SafeLocalPythonExecutor — import restrictions
# ---------------------------------------------------------------------------


class TestSafeLocalPythonExecutorImports:
    def test_banned_imports_removed_from_authorized(self):
        """Banned imports must not appear in the authorized list regardless of what caller passes."""
        executor = SafeLocalPythonExecutor(
            additional_imports=["subprocess", "socket", "math"],
            _inner=_MockInner(),
        )
        for banned in _BANNED_IMPORTS:
            assert banned not in executor._authorized_imports, (
                f"{banned!r} must not be in authorized imports"
            )

    def test_safe_imports_present(self):
        executor = SafeLocalPythonExecutor(_inner=_MockInner())
        for safe in ["math", "json", "re", "datetime"]:
            assert safe in executor._authorized_imports

    def test_additional_safe_import_added(self):
        executor = SafeLocalPythonExecutor(
            additional_imports=["numpy"],
            _inner=_MockInner(),
        )
        assert "numpy" in executor._authorized_imports

    def test_banned_list_coverage(self):
        """Verify the built-in banned list covers expected attack vectors."""
        expected_banned = {"subprocess", "socket", "ctypes", "importlib", "importlib.util"}
        assert expected_banned.issubset(_BANNED_IMPORTS)


# ---------------------------------------------------------------------------
# SafeLocalPythonExecutor — thread safety
# ---------------------------------------------------------------------------


class TestSafeLocalPythonExecutorThreadSafety:
    def test_concurrent_calls_restore_env_correctly(self):
        """Two concurrent executions must not corrupt each other's env view."""
        results: list[bool] = []
        errors: list[Exception] = []

        def _run(secret_key: str, secret_value: str):
            captured_env: dict = {}

            def _inner(code: str, *args, **kwargs):
                captured_env.update(os.environ.copy())

            executor = SafeLocalPythonExecutor(_inner=_inner)
            try:
                with patch.dict(os.environ, {secret_key: secret_value}, clear=False):
                    executor("code")
                # Secret must not be visible during execution
                results.append(secret_key not in captured_env)
            except Exception as exc:
                errors.append(exc)

        threads = [
            threading.Thread(target=_run, args=(f"SECRET_{i}", f"value_{i}"))
            for i in range(10)
        ]
        for t in threads:
            t.start()
        for t in threads:
            t.join()

        assert not errors, f"Threads raised: {errors}"
        assert all(results), "Some threads saw a secret that should have been stripped"


# ---------------------------------------------------------------------------
# Allowlist contents
# ---------------------------------------------------------------------------


class TestAllowlistContents:
    def test_core_vars_in_allowlist(self):
        """Spot-check that expected safe vars are on the allowlist."""
        required = {"PATH", "HOME", "LANG", "PYTHONPATH", "WORKSPACE_ID", "WORKSPACE_NAME", "PLATFORM_URL"}
        for var in required:
            assert var in _SAFE_ENV_ALLOWLIST, f"{var!r} missing from _SAFE_ENV_ALLOWLIST"

    def test_secrets_not_in_allowlist(self):
        """Known secret names must NOT appear on the allowlist."""
        forbidden = {
            "ANTHROPIC_API_KEY",
            "GH_TOKEN",
            "GITHUB_TOKEN",
            "OPENAI_API_KEY",
            "DATABASE_URL",
            "REDIS_URL",
            "SLACK_BOT_TOKEN",
            "JWT_SECRET",
            "DB_PASSWORD",
            "AWS_SECRET_ACCESS_KEY",
            "AWS_ACCESS_KEY_ID",
        }
        for var in forbidden:
            assert var not in _SAFE_ENV_ALLOWLIST, (
                f"{var!r} must NOT be in _SAFE_ENV_ALLOWLIST — it's a secret"
            )
