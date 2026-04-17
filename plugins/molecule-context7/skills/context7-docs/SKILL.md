---
name: Context7 Docs
description: >
  Fetch up-to-date library documentation from Context7 (mcp.context7.com) and
  inject it into your context window. Resolves library IDs and queries specific
  topics or API references without hallucinating outdated APIs.
tags: [docs, context7, libraries, llm-context, research]
examples:
  - "What are the hooks available in React 18?"
  - "Show me the FastAPI router documentation"
  - "Fetch the latest LangChain tool interface docs"
---

# Context7 Docs

Provides two tools for fetching accurate, up-to-date documentation:

- **resolve_library_id** — Resolve a human-friendly library name (e.g. `"react"`,
  `"fastapi"`) to its canonical Context7 library ID.
- **query_docs** — Fetch documentation snippets for a library and optional topic.

## Usage

```
1. Call resolve_library_id("fastapi") → get library_id "/tiangolo/fastapi"
2. Call query_docs(library_id="/tiangolo/fastapi", topic="dependency injection", tokens=5000)
3. Use the returned content to answer the user's question accurately
```

## Rate limits

Each agent session is capped at `CONTEXT7_MAX_CALLS_PER_SESSION` calls
(default 50). If you hit the limit, batch your queries or ask the user to
start a new session.
