"""Tests for the sandbox run_code tool — subprocess, docker-routing, and e2b backends.

The e2b backend tests use a fully mocked e2b_code_interpreter to avoid
requiring a real E2B_API_KEY or network access in CI.

Design notes:
- sandbox.py lives in tools/ alongside other tool modules.
- conftest.py stubs sys.modules["tools"] so a plain `import builtin_tools.sandbox`
  would hit the stub. We load sandbox.py via its file path instead.
- SANDBOX_BACKEND is captured as a module-level constant on load, so
  _load_sandbox() must be called with it set.
- E2B_API_KEY and e2b_code_interpreter are read at call-time inside
  _run_e2b(), so they must be present in os.environ / sys.modules during
  the actual async call (use monkeypatch or patch.dict).
"""

import asyncio
import importlib.util
import os
import sys
from pathlib import Path
from unittest.mock import MagicMock, patch

import pytest

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
_SANDBOX_PATH = Path(__file__).parent.parent / "builtin_tools" / "sandbox.py"


def _load_sandbox(sandbox_backend: str = "subprocess", extra_env: dict | None = None):
    """
    Load (or reload) tools/sandbox.py from its real file path.
    Only SANDBOX_BACKEND needs to be set at load time — it's a module-level
    constant. Other env vars (E2B_API_KEY etc.) are read at call-time and
    should be set by the caller via monkeypatch or patch.dict.
    """
    # Evict any previously cached copy.
    for key in list(sys.modules.keys()):
        if "sandbox_mod" in key:
            del sys.modules[key]

    saved = os.environ.get("SANDBOX_BACKEND")
    os.environ["SANDBOX_BACKEND"] = sandbox_backend

    for k, v in (extra_env or {}).items():
        os.environ[k] = v
    try:
        spec = importlib.util.spec_from_file_location("sandbox_mod", _SANDBOX_PATH)
        mod = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(mod)
    finally:
        if saved is None:
            os.environ.pop("SANDBOX_BACKEND", None)
        else:
            os.environ["SANDBOX_BACKEND"] = saved
        for k in (extra_env or {}):
            os.environ.pop(k, None)

    return mod


def _make_e2b_mock(stdout_text: str = "hello e2b\n", stderr_text: str = ""):
    """Build a mock e2b Sandbox that returns a plausible execution result."""
    result_obj = MagicMock()
    result_obj.text = stdout_text
    result_obj.error = None

    logs_obj = MagicMock()
    logs_obj.stdout = []
    logs_obj.stderr = [stderr_text] if stderr_text else []

    exec_obj = MagicMock()
    exec_obj.results = [result_obj]
    exec_obj.logs = logs_obj

    sandbox_instance = MagicMock()
    sandbox_instance.run_code.return_value = exec_obj
    sandbox_instance.kill.return_value = None

    sandbox_cls = MagicMock(return_value=sandbox_instance)
    return sandbox_cls, sandbox_instance


def _run_sync(coro):
    return asyncio.run(coro)


# ---------------------------------------------------------------------------
# subprocess backend
# ---------------------------------------------------------------------------

class TestSubprocessBackend:
    def test_python_hello(self):
        sb = _load_sandbox("subprocess")
        result = _run_sync(sb._run_subprocess('print("hello subprocess")', "python"))
        assert result["exit_code"] == 0
        assert "hello subprocess" in result["stdout"]
        assert result["backend"] == "subprocess"

    def test_stderr_nonzero_exit(self):
        sb = _load_sandbox("subprocess")
        result = _run_sync(sb._run_subprocess("import sys; sys.exit(2)", "python"))
        assert result["exit_code"] == 2

    def test_unsupported_language(self):
        sb = _load_sandbox("subprocess")
        result = _run_sync(sb._run_subprocess("code", "cobol"))
        assert result["exit_code"] == -1
        assert "Unsupported" in result["error"]

    def test_syntax_error_captured_in_stderr(self):
        sb = _load_sandbox("subprocess")
        result = _run_sync(sb._run_subprocess("def broken(:", "python"))
        assert result["exit_code"] != 0

    def test_timeout(self):
        sb = _load_sandbox("subprocess", {"SANDBOX_TIMEOUT": "1"})
        # Manually set the module-level constant that was captured at load time
        sb.SANDBOX_TIMEOUT = 1
        result = _run_sync(sb._run_subprocess("import time; time.sleep(10)", "python"))
        assert result["exit_code"] == -1
        assert "Timeout" in result["error"]


# ---------------------------------------------------------------------------
# E2B backend
# ---------------------------------------------------------------------------

class TestE2BBackend:
    """
    All tests mock e2b_code_interpreter to avoid real network calls.
    E2B_API_KEY must be present in os.environ for the duration of _run_e2b
    (it's read at call-time, not module-load time).
    """

    def _call_e2b(self, code: str, language: str, sandbox_cls, api_key: str = "test-key"):
        sb = _load_sandbox("e2b")
        mock_mod = MagicMock()
        mock_mod.Sandbox = sandbox_cls
        with patch.dict(os.environ, {"E2B_API_KEY": api_key}):
            with patch.dict("sys.modules", {"e2b_code_interpreter": mock_mod}):
                return _run_sync(sb._run_e2b(code, language)), sb, sandbox_cls

    def test_python_success(self):
        sandbox_cls, sandbox_instance = _make_e2b_mock(stdout_text="42\n")
        result, _, _ = self._call_e2b("print(6 * 7)", "python", sandbox_cls)

        assert result["exit_code"] == 0
        assert result["backend"] == "e2b"
        assert result["language"] == "python"
        assert result["stdout"] == "42\n"
        sandbox_instance.kill.assert_called_once()

    def test_javascript_success(self):
        sandbox_cls, sandbox_instance = _make_e2b_mock(stdout_text="hello js\n")
        result, _, _ = self._call_e2b('console.log("hi")', "javascript", sandbox_cls)

        assert result["exit_code"] == 0
        assert result["language"] == "javascript"
        # E2B kernel must be remapped: "javascript" → "js"
        call_args = sandbox_instance.run_code.call_args
        called_kernel = (
            call_args.kwargs.get("language")
            or (call_args.args[1] if len(call_args.args) > 1 else None)
        )
        assert called_kernel == "js", f"Expected kernel 'js', got {called_kernel!r}"

    def test_stderr_produces_nonzero_exit(self):
        sandbox_cls, _ = _make_e2b_mock(
            stdout_text="", stderr_text="NameError: name 'x' is not defined"
        )
        result, _, _ = self._call_e2b("print(x)", "python", sandbox_cls)

        assert result["exit_code"] == 1
        assert "NameError" in result["stderr"]

    def test_missing_api_key_returns_error(self):
        sb = _load_sandbox("e2b")
        sandbox_cls, _ = _make_e2b_mock()
        mock_mod = MagicMock()
        mock_mod.Sandbox = sandbox_cls
        # Do NOT set E2B_API_KEY
        with patch.dict("sys.modules", {"e2b_code_interpreter": mock_mod}):
            with patch.dict(os.environ, {}, clear=False):
                os.environ.pop("E2B_API_KEY", None)
                result = _run_sync(sb._run_e2b("print(1)", "python"))

        assert result["exit_code"] == -1
        assert "E2B_API_KEY" in result["error"]

    def test_missing_package_returns_error(self):
        sb = _load_sandbox("e2b")
        with patch.dict(os.environ, {"E2B_API_KEY": "key"}):
            # Simulate ImportError by putting None in sys.modules
            with patch.dict("sys.modules", {"e2b_code_interpreter": None}):
                result = _run_sync(sb._run_e2b("print(1)", "python"))

        assert result["exit_code"] == -1
        assert "e2b-code-interpreter" in result["error"]

    def test_unsupported_language_returns_error(self):
        sandbox_cls, _ = _make_e2b_mock()
        result, _, _ = self._call_e2b("echo hi", "shell", sandbox_cls)

        assert result["exit_code"] == -1
        assert "not supported by the e2b backend" in result["error"]

    def test_sandbox_always_killed_on_exception(self):
        """sandbox.kill() is called even when run_code raises."""
        sandbox_instance = MagicMock()
        sandbox_instance.run_code.side_effect = RuntimeError("network error")
        sandbox_instance.kill.return_value = None
        sandbox_cls = MagicMock(return_value=sandbox_instance)

        result, _, _ = self._call_e2b("print(1)", "python", sandbox_cls)

        assert result["exit_code"] == -1
        assert "network error" in result["error"]
        sandbox_instance.kill.assert_called_once()

    def test_output_truncated_at_max_output(self):
        big = "x" * 20_000
        sandbox_cls, _ = _make_e2b_mock(stdout_text=big)
        result, sb, _ = self._call_e2b("print('x' * 20000)", "python", sandbox_cls)

        assert "stdout" in result
        assert len(result["stdout"]) <= sb.MAX_OUTPUT

    def test_api_key_forwarded_to_constructor(self):
        """E2B_API_KEY from env is passed to Sandbox(api_key=...)."""
        sandbox_cls, _ = _make_e2b_mock()
        _, _, used_cls = self._call_e2b("print(1)", "python", sandbox_cls, api_key="my-secret")

        call_kwargs = used_cls.call_args.kwargs
        assert call_kwargs.get("api_key") == "my-secret"

    def test_timeout_forwarded_to_constructor(self):
        """SANDBOX_TIMEOUT is forwarded as the sandbox timeout kwarg."""
        sandbox_cls, _ = _make_e2b_mock()
        sb = _load_sandbox("e2b", {"SANDBOX_TIMEOUT": "45"})
        sb.SANDBOX_TIMEOUT = 45

        mock_mod = MagicMock()
        mock_mod.Sandbox = sandbox_cls
        with patch.dict(os.environ, {"E2B_API_KEY": "key"}):
            with patch.dict("sys.modules", {"e2b_code_interpreter": mock_mod}):
                _run_sync(sb._run_e2b("print(1)", "python"))

        call_kwargs = sandbox_cls.call_args.kwargs
        assert call_kwargs.get("timeout") == 45


# ---------------------------------------------------------------------------
# Dispatcher routing — verify SANDBOX_BACKEND selects the right function
# ---------------------------------------------------------------------------

class TestRunCodeDispatcher:
    def test_subprocess_backend_dispatched(self):
        sb = _load_sandbox("subprocess")
        assert sb.SANDBOX_BACKEND == "subprocess"
        result = _run_sync(sb._run_subprocess("1 + 1", "python"))
        assert result["exit_code"] == 0

    def test_e2b_backend_dispatched(self):
        """run_code routes to _run_e2b when SANDBOX_BACKEND=e2b."""
        sb = _load_sandbox("e2b")
        assert sb.SANDBOX_BACKEND == "e2b"

        called_with = []

        async def fake_e2b(code, language):
            called_with.append((code, language))
            return {"exit_code": 0, "stdout": "ok", "backend": "e2b"}

        with patch.object(sb, "_run_e2b", fake_e2b):
            # conftest mocks @tool as identity, so run_code is the raw async fn
            result = _run_sync(sb.run_code("print(1)", "python"))

        assert called_with == [("print(1)", "python")]
        assert result["backend"] == "e2b"

    def test_docker_backend_dispatched(self):
        """run_code routes to _run_docker when SANDBOX_BACKEND=docker."""
        sb = _load_sandbox("docker")
        assert sb.SANDBOX_BACKEND == "docker"

        called_with = []

        async def fake_docker(code, language):
            called_with.append((code, language))
            return {"exit_code": 0, "stdout": "ok", "backend": "docker"}

        with patch.object(sb, "_run_docker", fake_docker):
            result = _run_sync(sb.run_code("echo hi", "shell"))

        assert called_with == [("echo hi", "shell")]
        assert result["backend"] == "docker"

    def test_subprocess_backend_routes_to_run_subprocess(self):
        """run_code with SANDBOX_BACKEND=subprocess calls _run_subprocess."""
        sb = _load_sandbox("subprocess")

        called_with = []

        async def fake_subprocess(code, language):
            called_with.append((code, language))
            return {"exit_code": 0, "stdout": "ok", "backend": "subprocess"}

        with patch.object(sb, "_run_subprocess", fake_subprocess):
            result = _run_sync(sb.run_code("print(1)", "python"))

        assert called_with == [("print(1)", "python")]
        assert result["backend"] == "subprocess"


# ---------------------------------------------------------------------------
# Additional subprocess backend edge-cases
# ---------------------------------------------------------------------------

class TestSubprocessEdgeCases:

    def test_process_lookup_error_on_kill(self):
        """ProcessLookupError during proc.kill() after timeout is silently ignored."""
        sb = _load_sandbox("subprocess")
        sb.SANDBOX_TIMEOUT = 1

        # We need the real timeout path but with proc.kill() raising ProcessLookupError.
        # Patch asyncio.wait_for to raise TimeoutError then patch proc.kill to raise.
        import asyncio as _asyncio

        original_create = _asyncio.create_subprocess_exec

        async def fake_create(*args, **kwargs):
            proc = MagicMock()
            proc.returncode = None

            async def _communicate():
                raise _asyncio.TimeoutError()

            proc.communicate = _communicate

            def _kill():
                raise ProcessLookupError("no such process")

            proc.kill = _kill

            async def _wait():
                pass

            proc.wait = _wait
            return proc

        with patch("asyncio.create_subprocess_exec", fake_create):
            result = _run_sync(sb._run_subprocess("import time; time.sleep(100)", "python"))

        assert result["exit_code"] == -1
        assert "Timeout" in result["error"]

    def test_general_exception_in_subprocess_exec(self):
        """Exception from asyncio.create_subprocess_exec is caught and returned."""
        sb = _load_sandbox("subprocess")

        async def fake_create(*args, **kwargs):
            raise OSError("no such executable")

        with patch("asyncio.create_subprocess_exec", fake_create):
            result = _run_sync(sb._run_subprocess("print(1)", "python"))

        assert result["exit_code"] == -1
        assert "no such executable" in result["error"]


# ---------------------------------------------------------------------------
# Docker backend
# ---------------------------------------------------------------------------

class TestDockerBackend:

    def _make_docker_proc(self, stdout: bytes = b"", stderr: bytes = b"", returncode: int = 0):
        """Return a fake asyncio subprocess-like object."""
        proc = MagicMock()
        proc.returncode = returncode

        async def _communicate():
            return (stdout, stderr)

        proc.communicate = _communicate
        return proc

    def test_run_docker_unsupported_language(self):
        sb = _load_sandbox("docker")
        result = _run_sync(sb._run_docker("code", "cobol"))
        assert result["exit_code"] == -1
        assert "Unsupported" in result["error"]

    def test_run_docker_success(self):
        """_run_docker returns exit_code=0 and correct stdout on success."""
        import asyncio as _asyncio

        sb = _load_sandbox("docker")
        fake_proc = self._make_docker_proc(stdout=b"hello docker\n", stderr=b"")

        async def fake_wait_for(coro, timeout):
            return await coro

        async def fake_create(*args, **kwargs):
            return fake_proc

        with patch("asyncio.create_subprocess_exec", fake_create), \
             patch("asyncio.wait_for", fake_wait_for):
            result = _run_sync(sb._run_docker('print("hello docker")', "python"))

        assert result["exit_code"] == 0
        assert "hello docker" in result["stdout"]
        assert result["backend"] == "docker"
        assert result["language"] == "python"

    def test_run_docker_timeout(self):
        """asyncio.wait_for TimeoutError → returns timeout error dict."""
        import asyncio as _asyncio

        sb = _load_sandbox("docker")
        sb.SANDBOX_TIMEOUT = 1

        async def fake_create(*args, **kwargs):
            proc = MagicMock()
            return proc

        async def fake_wait_for(coro, timeout):
            raise _asyncio.TimeoutError()

        with patch("asyncio.create_subprocess_exec", fake_create), \
             patch("asyncio.wait_for", fake_wait_for):
            result = _run_sync(sb._run_docker("code", "python"))

        assert result["exit_code"] == -1
        assert "Timeout" in result["error"]

    def test_run_docker_general_exception(self):
        """Generic exception in create_subprocess_exec → returns error dict."""
        sb = _load_sandbox("docker")

        async def fake_create(*args, **kwargs):
            raise RuntimeError("docker not available")

        with patch("asyncio.create_subprocess_exec", fake_create):
            result = _run_sync(sb._run_docker("code", "python"))

        assert result["exit_code"] == -1
        assert "docker not available" in result["error"]

    def test_run_docker_cleanup_on_success(self, tmp_path, monkeypatch):
        """Temp file is removed after successful run."""
        import asyncio as _asyncio
        import tempfile
        import os

        sb = _load_sandbox("docker")

        created_files = []
        original_mkstemp = tempfile.mkstemp

        def fake_mkstemp(suffix="", prefix="", dir=None, text=False):
            fd, path = original_mkstemp(suffix=suffix, prefix=prefix)
            created_files.append(path)
            return fd, path

        fake_proc = self._make_docker_proc(stdout=b"done\n", stderr=b"")

        async def fake_wait_for(coro, timeout):
            return await coro

        async def fake_create(*args, **kwargs):
            return fake_proc

        with patch("tempfile.mkstemp", fake_mkstemp), \
             patch("asyncio.create_subprocess_exec", fake_create), \
             patch("asyncio.wait_for", fake_wait_for):
            result = _run_sync(sb._run_docker("print('done')", "python"))

        assert result["exit_code"] == 0
        for f in created_files:
            assert not os.path.exists(f), f"temp file {f} was not cleaned up"

    def test_run_docker_cleanup_on_exception(self, tmp_path, monkeypatch):
        """Temp file is removed even when an exception is raised."""
        import tempfile
        import os

        sb = _load_sandbox("docker")

        created_files = []
        original_mkstemp = tempfile.mkstemp

        def fake_mkstemp(suffix="", prefix="", dir=None, text=False):
            fd, path = original_mkstemp(suffix=suffix, prefix=prefix)
            created_files.append(path)
            return fd, path

        async def fake_create(*args, **kwargs):
            raise RuntimeError("crash")

        with patch("tempfile.mkstemp", fake_mkstemp), \
             patch("asyncio.create_subprocess_exec", fake_create):
            result = _run_sync(sb._run_docker("print(1)", "python"))

        assert result["exit_code"] == -1
        for f in created_files:
            assert not os.path.exists(f), f"temp file {f} was not cleaned up after exception"

    def test_run_docker_cleanup_oserror_swallowed(self, tmp_path):
        """Lines 165-166: os.unlink raises OSError in finally block — swallowed, result still returned."""
        import tempfile
        import os

        sb = _load_sandbox("docker")
        fake_proc = self._make_docker_proc(stdout=b"ok\n", stderr=b"")

        created_files = []
        original_mkstemp = tempfile.mkstemp

        def fake_mkstemp(suffix="", prefix="", dir=None, text=False):
            fd, path = original_mkstemp(suffix=suffix, prefix=prefix)
            created_files.append(path)
            return fd, path

        async def fake_wait_for(coro, timeout):
            return await coro

        async def fake_create(*args, **kwargs):
            return fake_proc

        original_unlink = os.unlink
        unlink_calls = []

        def raising_unlink(path):
            unlink_calls.append(path)
            raise OSError("permission denied")

        with patch("tempfile.mkstemp", fake_mkstemp), \
             patch("asyncio.create_subprocess_exec", fake_create), \
             patch("asyncio.wait_for", fake_wait_for), \
             patch("os.unlink", raising_unlink):
            result = _run_sync(sb._run_docker("print('ok')", "python"))

        # OSError is swallowed; result is still returned
        assert result["exit_code"] == 0
        assert len(unlink_calls) > 0


# ---------------------------------------------------------------------------
# Gap 4: E2B backend — additional coverage paths
# ---------------------------------------------------------------------------

class TestE2BBackendGapCoverage:
    """Cover lines 242, 248, 268-269, 280-281 in _run_e2b."""

    def _call_e2b(self, code, language, mock_e2b_mod, api_key="test-key"):
        sb = _load_sandbox("e2b")
        with patch.dict(os.environ, {"E2B_API_KEY": api_key}):
            with patch.dict("sys.modules", {"e2b_code_interpreter": mock_e2b_mod}):
                return _run_sync(sb._run_e2b(code, language)), sb

    def test_result_error_attribute_captured(self):
        """Line 242: result.error in execution.results → captured in stderr."""
        result_obj = MagicMock()
        result_obj.text = None
        result_obj.error = "NameError: x not defined"

        logs_obj = MagicMock()
        logs_obj.stdout = []
        logs_obj.stderr = []

        exec_obj = MagicMock()
        exec_obj.results = [result_obj]
        exec_obj.logs = logs_obj

        sandbox_instance = MagicMock()
        sandbox_instance.run_code.return_value = exec_obj
        sandbox_instance.kill.return_value = None
        sandbox_cls = MagicMock(return_value=sandbox_instance)

        mock_mod = MagicMock()
        mock_mod.Sandbox = sandbox_cls

        result, _ = self._call_e2b("print(x)", "python", mock_mod)

        assert result["exit_code"] == 1
        assert "NameError" in result["stderr"]

    def test_logs_stdout_captured(self):
        """Line 248: execution.logs.stdout → appended to stdout_parts."""
        result_obj = MagicMock()
        result_obj.text = None
        result_obj.error = None

        logs_obj = MagicMock()
        logs_obj.stdout = ["hello from logs\n"]
        logs_obj.stderr = []

        exec_obj = MagicMock()
        exec_obj.results = [result_obj]
        exec_obj.logs = logs_obj

        sandbox_instance = MagicMock()
        sandbox_instance.run_code.return_value = exec_obj
        sandbox_instance.kill.return_value = None
        sandbox_cls = MagicMock(return_value=sandbox_instance)

        mock_mod = MagicMock()
        mock_mod.Sandbox = sandbox_cls

        result, _ = self._call_e2b("print('hello from logs')", "python", mock_mod)

        assert result["exit_code"] == 0
        assert "hello from logs" in result["stdout"]

    def test_e2b_timeout_returns_error(self):
        """Lines 268-269: asyncio.TimeoutError raised → returns timeout error dict."""
        import asyncio as _asyncio

        # Sandbox constructor itself raises TimeoutError via wait_for
        sandbox_instance = MagicMock()
        sandbox_cls = MagicMock(return_value=sandbox_instance)

        mock_mod = MagicMock()
        mock_mod.Sandbox = sandbox_cls

        sb = _load_sandbox("e2b")

        original_wait_for = _asyncio.wait_for

        call_count = {"n": 0}

        async def raising_wait_for(coro, timeout):
            call_count["n"] += 1
            if call_count["n"] == 1:
                raise _asyncio.TimeoutError()
            return await original_wait_for(coro, timeout)

        with patch.dict(os.environ, {"E2B_API_KEY": "test-key"}):
            with patch.dict("sys.modules", {"e2b_code_interpreter": mock_mod}):
                with patch("asyncio.wait_for", raising_wait_for):
                    result = _run_sync(sb._run_e2b("print(1)", "python"))

        assert result["exit_code"] == -1
        assert "Timeout" in result["error"]

    def test_e2b_cleanup_exception_swallowed(self):
        """Lines 280-281: sandbox.kill raises in finally → exception swallowed."""
        result_obj = MagicMock()
        result_obj.text = "42\n"
        result_obj.error = None

        logs_obj = MagicMock()
        logs_obj.stdout = []
        logs_obj.stderr = []

        exec_obj = MagicMock()
        exec_obj.results = [result_obj]
        exec_obj.logs = logs_obj

        sandbox_instance = MagicMock()
        sandbox_instance.run_code.return_value = exec_obj
        # Make kill raise an exception
        sandbox_instance.kill.side_effect = RuntimeError("kill failed")
        sandbox_cls = MagicMock(return_value=sandbox_instance)

        mock_mod = MagicMock()
        mock_mod.Sandbox = sandbox_cls

        result, _ = self._call_e2b("print(42)", "python", mock_mod)

        # Result is still returned despite kill() failing
        assert result["exit_code"] == 0
        assert "42" in result["stdout"]
