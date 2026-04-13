# Tests

This repo uses the standard monorepo testing convention: **unit tests live with their package, cross-component E2E tests live here.**

## Where to find tests

| Scope | Location |
|---|---|
| Go unit + integration (platform, CLI, handlers) | `platform/**/*_test.go` — run with `cd platform && go test -race ./...` |
| TypeScript unit (canvas components, hooks, store) | `canvas/src/**/__tests__/` — run with `cd canvas && npm test -- --run` |
| TypeScript unit (MCP server handlers) | `mcp-server/src/__tests__/` — run with `cd mcp-server && npx jest` |
| Python unit (workspace runtime, adapters) | `workspace-template/tests/` — run with `cd workspace-template && python3 -m pytest` |
| Python unit (SDK: plugin + remote agent) | `sdk/python/tests/` — run with `cd sdk/python && python3 -m pytest` |
| **Cross-component E2E** (spans platform + runtime + HTTP) | `tests/e2e/` ← **you are here** |

## Why split this way

- **Go** requires co-located `_test.go` files to access unexported symbols.
- **Per-package test commands** keep the inner loop fast — changing canvas doesn't re-run Go tests.
- **`tests/e2e/`** covers scenarios that no single package owns: a full workspace lifecycle, A2A across two provisioned agents, delegation chains, bundle round-trips.

## Running E2E

Every E2E script here assumes the platform is running at `localhost:8080` and (where noted) provisioned agents are online. See the header comment of each `.sh` for specifics.
