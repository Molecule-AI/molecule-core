---
id: code-sandbox
name: Code Sandbox
description: Execute code snippets safely in an isolated subprocess, Docker container, or E2B cloud sandbox. Returns stdout, stderr, and exit code.
tags: [code, execution, sandbox, python, javascript, bash]
---

## When to Use
Use when an agent needs to run untrusted or user-provided code and observe the output without risking the host filesystem or network.

## How It Works
Wraps `workspace-template/sandbox.py`. Selects backend based on `SANDBOX_BACKEND` env var:
- `subprocess` (default) — runs in a restricted subprocess on the host
- `docker` — runs in an isolated Docker container (requires Docker socket)
- `e2b` — runs in an E2B cloud sandbox (requires E2B_API_KEY)

## Examples

```python
async with Sandbox(language="python") as sb:
    result = await sb.run("print('hello world')")
    print(result.stdout)  # "hello world\n"

async with Sandbox(language="javascript") as sb:
    result = await sb.run("console.log(2 + 2)")
    print(result.stdout)  # "4\n"
```

## Security Notes
- subprocess backend: no network by default, /tmp only
- docker backend: no --privileged, read-only mounts
- e2b backend: full isolation, 30s timeout enforced
