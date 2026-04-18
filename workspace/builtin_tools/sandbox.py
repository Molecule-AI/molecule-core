"""Code sandbox tool for safe code execution.

Executes code in an isolated environment. Three backends are supported:

subprocess (default)
    Runs code locally via asyncio subprocess with a hard timeout.
    Best for Tier 1/2 agents where run_code is lightly used and the
    workspace container itself is the isolation boundary.

docker
    Throwaway Docker-in-Docker container: network disabled, memory capped,
    read-only filesystem. Requires Docker socket access inside the container.
    Best for Tier 3 on-prem deployments.

e2b
    Cloud-hosted microVM sandbox via E2B (https://e2b.dev).
    No local Docker required — code runs in E2B's isolated cloud VMs.
    Supports Python and JavaScript.
    Requires:
      - e2b-code-interpreter Python package (pinned in requirements.txt)
      - E2B_API_KEY workspace secret (set via canvas Secrets panel or API)
    Best for hosted/cloud Molecule AI deployments.

Backend is selected via the SANDBOX_BACKEND env var, which the provisioner
sets from config.yaml → sandbox.backend. Default: "subprocess".
"""

import asyncio
import logging
import os
import tempfile

from langchain_core.tools import tool

logger = logging.getLogger(__name__)

SANDBOX_BACKEND = os.environ.get("SANDBOX_BACKEND", "subprocess")
SANDBOX_TIMEOUT = int(os.environ.get("SANDBOX_TIMEOUT", "30"))
SANDBOX_MEMORY_LIMIT = os.environ.get("SANDBOX_MEMORY_LIMIT", "256m")
MAX_OUTPUT = 10_000

# E2B kernel names differ from internal language names.
_E2B_KERNEL_MAP = {
    "python": "python3",
    "javascript": "js",
    "js": "js",
}


@tool
async def run_code(code: str, language: str = "python") -> dict:
    """Execute code in an isolated sandbox and return the output.

    Args:
        code: The code to execute.
        language: Programming language — python, javascript, or shell.
                  The e2b backend supports python and javascript only.
    """
    if SANDBOX_BACKEND == "docker":
        return await _run_docker(code, language)
    elif SANDBOX_BACKEND == "e2b":
        return await _run_e2b(code, language)
    else:
        return await _run_subprocess(code, language)


async def _run_subprocess(code: str, language: str) -> dict:
    """Fallback: run code in a subprocess with timeout."""
    cmd_map = {
        "python": ["python3", "-c"],
        "javascript": ["node", "-e"],
        "shell": ["sh", "-c"],
        "bash": ["bash", "-c"],
    }

    cmd_prefix = cmd_map.get(language)
    if not cmd_prefix:
        return {"error": f"Unsupported language: {language}", "exit_code": -1}

    try:
        proc = await asyncio.create_subprocess_exec(
            *cmd_prefix, code,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )

        stdout, stderr = await asyncio.wait_for(proc.communicate(), timeout=SANDBOX_TIMEOUT)

        return {
            "exit_code": proc.returncode,
            "stdout": stdout.decode("utf-8", errors="replace")[:MAX_OUTPUT],
            "stderr": stderr.decode("utf-8", errors="replace")[:MAX_OUTPUT],
            "language": language,
            "backend": "subprocess",
        }
    except asyncio.TimeoutError:
        try:
            proc.kill()
            await proc.wait()
        except ProcessLookupError:
            pass
        return {"error": f"Timeout after {SANDBOX_TIMEOUT}s", "exit_code": -1}
    except Exception as e:
        return {"error": str(e), "exit_code": -1}


async def _run_docker(code: str, language: str) -> dict:
    """Run code in a throwaway Docker container via mounted temp file."""
    image_map = {
        "python": ("python:3.11-slim", ["python3", "/sandbox/code.py"]),
        "javascript": ("node:20-slim", ["node", "/sandbox/code.js"]),
        "shell": ("alpine:3.18", ["sh", "/sandbox/code.sh"]),
        "bash": ("alpine:3.18", ["sh", "/sandbox/code.sh"]),
    }

    entry = image_map.get(language)
    if not entry:
        return {"error": f"Unsupported language: {language}", "exit_code": -1}

    image, run_cmd = entry
    code_file = None

    try:
        # Write code to temp file — avoids shell metacharacter injection
        ext = {"python": ".py", "javascript": ".js", "shell": ".sh", "bash": ".sh"}.get(language, ".txt")
        fd, code_file = tempfile.mkstemp(suffix=ext, prefix="sandbox_")
        with os.fdopen(fd, "w") as f:
            f.write(code)

        cmd = [
            "docker", "run", "--rm",
            "--network", "none",
            "--memory", SANDBOX_MEMORY_LIMIT,
            "--cpus", "0.5",
            "--read-only",
            "--tmpfs", "/tmp:size=32m",
            "-v", f"{code_file}:/sandbox/code{ext}:ro",
            image,
        ] + run_cmd

        proc = await asyncio.create_subprocess_exec(
            *cmd,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )

        stdout, stderr = await asyncio.wait_for(proc.communicate(), timeout=SANDBOX_TIMEOUT)

        return {
            "exit_code": proc.returncode,
            "stdout": stdout.decode("utf-8", errors="replace")[:MAX_OUTPUT],
            "stderr": stderr.decode("utf-8", errors="replace")[:MAX_OUTPUT],
            "language": language,
            "backend": "docker",
            "image": image,
        }
    except asyncio.TimeoutError:
        return {"error": f"Timeout after {SANDBOX_TIMEOUT}s", "exit_code": -1}
    except Exception as e:
        return {"error": str(e), "exit_code": -1}
    finally:
        if code_file:
            try:
                os.unlink(code_file)
            except OSError:
                pass


async def _run_e2b(code: str, language: str) -> dict:
    """Run code in an E2B cloud microVM sandbox.

    Requires the e2b-code-interpreter package and an E2B_API_KEY secret.
    Each call creates a fresh sandbox, runs the code, and destroys the sandbox.
    Sandbox lifetime is bounded by SANDBOX_TIMEOUT seconds.

    Supported languages: python, javascript.
    """
    # Import lazily so the package is only required when the e2b backend is
    # actually configured — other backends work without it installed.
    try:
        from e2b_code_interpreter import Sandbox
    except ImportError:
        return {
            "error": (
                "e2b-code-interpreter is not installed. "
                "Add it to requirements.txt or switch to the docker/subprocess backend."
            ),
            "exit_code": -1,
        }

    api_key = os.environ.get("E2B_API_KEY")
    if not api_key:
        return {
            "error": (
                "E2B_API_KEY is not set. "
                "Add it as a workspace secret via the canvas Secrets panel or platform API."
            ),
            "exit_code": -1,
        }

    kernel = _E2B_KERNEL_MAP.get(language)
    if kernel is None:
        return {
            "error": (
                f"Language '{language}' is not supported by the e2b backend. "
                "Supported: python, javascript."
            ),
            "exit_code": -1,
        }

    sandbox = None
    try:
        # Create a fresh sandbox for this execution.
        # timeout controls the sandbox lifetime in seconds.
        sandbox = await asyncio.wait_for(
            asyncio.get_running_loop().run_in_executor(
                None,
                lambda: Sandbox(api_key=api_key, timeout=SANDBOX_TIMEOUT),
            ),
            timeout=SANDBOX_TIMEOUT,
        )

        # Execute code and collect results.
        execution = await asyncio.wait_for(
            asyncio.get_running_loop().run_in_executor(
                None,
                lambda: sandbox.run_code(code, language=kernel),
            ),
            timeout=SANDBOX_TIMEOUT,
        )

        # E2B returns a list of Result objects; collect text/error output.
        stdout_parts = []
        stderr_parts = []

        for result in execution.results:
            # result.text is the primary output (stdout equivalent)
            if hasattr(result, "text") and result.text:
                stdout_parts.append(str(result.text))
            # Some result types expose an error attribute
            if hasattr(result, "error") and result.error:
                stderr_parts.append(str(result.error))

        # Logs are stored separately in execution.logs
        if hasattr(execution, "logs"):
            logs = execution.logs
            if hasattr(logs, "stdout") and logs.stdout:
                stdout_parts.extend(logs.stdout)
            if hasattr(logs, "stderr") and logs.stderr:
                stderr_parts.extend(logs.stderr)

        combined_stdout = "".join(stdout_parts)[:MAX_OUTPUT]
        combined_stderr = "".join(stderr_parts)[:MAX_OUTPUT]

        # Treat any stderr output as a non-zero exit code (e2b doesn't expose
        # a numeric exit code at the sandbox level).
        exit_code = 1 if combined_stderr else 0

        return {
            "exit_code": exit_code,
            "stdout": combined_stdout,
            "stderr": combined_stderr,
            "language": language,
            "backend": "e2b",
        }

    except asyncio.TimeoutError:
        logger.warning("E2B sandbox timed out after %ds", SANDBOX_TIMEOUT)
        return {"error": f"Timeout after {SANDBOX_TIMEOUT}s", "exit_code": -1}
    except Exception as e:
        logger.exception("E2B sandbox error: %s", e)
        return {"error": str(e), "exit_code": -1}
    finally:
        # Always destroy the sandbox to avoid leaking E2B credits.
        if sandbox is not None:
            try:
                await asyncio.get_running_loop().run_in_executor(
                    None, sandbox.kill
                )
            except Exception:
                pass  # Best-effort cleanup
