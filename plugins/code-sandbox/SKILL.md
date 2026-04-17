# code-sandbox

Run arbitrary code in a sandboxed environment using the `run_code` tool.

## When to use

Call `run_code` whenever you need to:
- Execute a Python script and capture its output
- Run JavaScript/Node snippets inline
- Execute shell or bash commands safely
- Validate logic that is hard to reason about statically (math, string transforms, file ops)

## Tool signature

```python
run_code(code: str, language: str = "python") -> dict
```

Returns:
```json
{
  "exit_code": 0,
  "stdout": "...",
  "stderr": "...",
  "language": "python",
  "backend": "subprocess"
}
```

A non-zero `exit_code` means the code raised an exception or the interpreter
returned an error — always check it before using `stdout`.

## Supported languages

| `language` value | Interpreter | subprocess | docker | e2b |
|-----------------|-------------|:----------:|:------:|:---:|
| `python`        | python3     | ✓ | ✓ | ✓ |
| `javascript`    | node        | ✓ | ✓ | ✓ |
| `shell`         | sh          | ✓ | ✓ | — |
| `bash`          | bash        | ✓ | ✓ | — |

## Backend selection

The active backend is set by the `SANDBOX_BACKEND` environment variable
(configured via `config.yaml → sandbox.backend` or the Secrets panel).

| Backend | How to enable | Notes |
|---------|--------------|-------|
| `subprocess` | default | No extra deps. Timeout = `SANDBOX_TIMEOUT` (default 30 s). |
| `docker` | `SANDBOX_BACKEND=docker` | Requires Docker socket in container. Network disabled, `--memory 256m`, `--cpus 0.5`, read-only FS + tmpfs. |
| `e2b` | `SANDBOX_BACKEND=e2b` | Requires `E2B_API_KEY` secret and `e2b-code-interpreter` package. Python + JavaScript only. Cloud microVM; fresh sandbox per call. |

## Usage examples

**Quick Python computation:**
```python
result = await run_code("print(2 ** 32)")
# result["stdout"] == "4294967296\n"
```

**Multi-line script with error check:**
```python
result = await run_code("""
import json, sys
data = {"key": "value"}
print(json.dumps(data))
""", language="python")

if result["exit_code"] != 0:
    raise RuntimeError(f"Script failed: {result['stderr']}")
output = result["stdout"]
```

**Shell one-liner:**
```python
result = await run_code("echo hello && ls /tmp", language="shell")
```

**JavaScript:**
```python
result = await run_code("console.log([1,2,3].map(x => x*2).join(','))", language="javascript")
# result["stdout"] == "2,4,6\n"
```

## Resource limits & timeouts

- `SANDBOX_TIMEOUT` (default `30`): hard wall-clock timeout in seconds for
  all backends. Exceeded calls return `{"exit_code": -1, "stderr": "timeout"}`.
- `SANDBOX_MEMORY_LIMIT` (default `"256m"`): Docker/e2b memory cap.

## Security notes

- **subprocess**: inherits the workspace container's isolation (network,
  filesystem). Suitable for trusted code or when the container is already
  an isolated tier.
- **docker**: network disabled at the Docker layer; tmpfs for `/tmp`; no
  access to host paths. Recommended for untrusted user-supplied code.
- **e2b**: each call runs in a fresh cloud microVM with no persistence
  between calls. Safest for multi-tenant or public-facing agents.
