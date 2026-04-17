# molecule-context7

Context7 MCP integration for Molecule AI — fetches up-to-date library
documentation from [mcp.context7.com](https://mcp.context7.com) and injects
it into the agent's context window.

## Tools

| Tool | Description |
|------|-------------|
| `resolve_library_id` | Resolve a library name to its canonical Context7 library ID |
| `query_docs` | Fetch documentation snippets for a library/topic |

## Setup

### Key Management (per-workspace — never global)

Context7 API keys (`CONTEXT7_API_KEY`) are issued and rotated per workspace.
Setting a key as a **global secret** would share one credential across every
workspace in your org — a single compromise would affect all agents and make
audit attribution impossible.

**Always set `CONTEXT7_API_KEY` as a workspace secret:**

```bash
# Via Molecule AI Canvas → Workspace → Secrets tab
CONTEXT7_API_KEY=ctx7_<your-key-here>

# Via API
curl -X POST http://localhost:8080/workspaces/<id>/secrets \
  -H "Authorization: Bearer <token>" \
  -d '{"key":"CONTEXT7_API_KEY","value":"ctx7_<your-key>"}'
```

Generate keys at [context7.com/dashboard → API Keys](https://context7.com/dashboard).

### Rate limiting

Set `CONTEXT7_MAX_CALLS_PER_SESSION` (default `50`) to cap the number of
Context7 API calls per agent session.  Increase only if your use case
requires fetching docs for many libraries in a single task.

### Missing key behaviour

When `CONTEXT7_API_KEY` is absent the tools operate in **mock mode** — they
return a stub response so the agent can still run without network access.
This is intentional for local development and CI environments.

## Security properties

- **Response scrubbing**: any `ctx7_*` token that leaks into a Context7
  response is replaced with `[REDACTED]` before the result reaches the agent.
- **Query validation**: queries longer than 200 characters are rejected, and
  queries containing secret-like patterns (API keys, bearer tokens) are
  refused before reaching `mcp.context7.com`.
- **Session call counter**: `CONTEXT7_MAX_CALLS_PER_SESSION` (default `50`)
  prevents runaway LLM loops from draining quota.
- **Key scope**: `CONTEXT7_API_KEY` must be a workspace secret (see above).

## Example

```python
# Resolve a library name to its Context7 ID
result = await resolve_library_id("react")
library_id = result["library_id"]   # e.g. "/facebook/react"

# Fetch documentation
docs = await query_docs(library_id=library_id, topic="hooks", tokens=5000)
print(docs["content"])
```
