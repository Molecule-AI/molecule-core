# molecule-github-mcp

Wraps the official GitHub MCP server (`@modelcontextprotocol/server-github`),
giving agents access to GitHub tools (issues, PRs, code search, repositories,
and more) via the Model Context Protocol (MCP) over stdio.

## When to use

Install this plugin on any workspace that needs to:
- Create, update, or read GitHub issues and pull requests
- Search code or repositories on GitHub
- Manage repository content (read/write files, create commits)
- Query GitHub Actions workflow runs
- Interact with the GitHub API through natural language

## Prerequisites

1. **Node.js + npx** must be available in the workspace container (`npx` is
   the launch command for `@modelcontextprotocol/server-github`).
2. A **GitHub personal access token** (or GitHub App token) with the scopes
   required by the tools you intend to use. At minimum:
   - `repo` — full repository access (issues, PRs, contents)
   - `read:org` — organisation membership (for org-scoped searches)

## Configuration

Set the `GITHUB_TOKEN` secret on the workspace:

```bash
# via the Molecule AI API
curl -X POST http://localhost:8080/workspaces/<id>/secrets \
  -H "Content-Type: application/json" \
  -d '{"key": "GITHUB_TOKEN", "value": "ghp_..."}'
```

Or via the Canvas UI: open the workspace → Secrets tab → add `GITHUB_TOKEN`.

The plugin's `plugin.yaml` declares `env: { GITHUB_TOKEN: "${GITHUB_TOKEN}" }`,
so the `MCPServerAdaptor` resolves the token from the workspace environment at
install time and writes it into `mcp-servers.json` for the executor to use.

## How it works

When installed, `MCPServerAdaptor.install()` reads the `mcp_servers:` list
from this plugin's `plugin.yaml` and writes an entry to
`<configs_dir>/mcp-servers.json`:

```json
{
  "github": {
    "name": "github",
    "command": "npx",
    "args": ["-y", "@modelcontextprotocol/server-github"],
    "env": { "GITHUB_TOKEN": "<resolved-value>" },
    "plugin": "molecule-github-mcp"
  }
}
```

Executors that support MCP (Claude Code, Hermes, etc.) read this file at
startup and launch the `npx` server as a subprocess, making all of its
tools available to the agent automatically.

## Uninstalling

```bash
curl -X DELETE http://localhost:8080/workspaces/<id>/plugins/molecule-github-mcp
```

`MCPServerAdaptor.uninstall()` removes the `github` entry from
`mcp-servers.json` so the server is no longer launched on next restart.

## Security notes

- The `MCPServerAdaptor` never passes `shell=True` to subprocess.
- `GITHUB_TOKEN` is resolved from `os.environ` at install time; the raw
  `${GITHUB_TOKEN}` placeholder is never written to disk.
- Env keys that could override security-sensitive variables (`PATH`,
  `LD_PRELOAD`, `PYTHONPATH`, etc.) are rejected at validation time.
- Stderr from the MCP server process is captured and sanitised; it is never
  forwarded verbatim to the agent.
