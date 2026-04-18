# Local Development

## Starting the Stack

```bash
docker compose up
```

This starts:

| Service | Port | Description |
|---------|------|-------------|
| Postgres | internal only | Primary database |
| Redis | internal only | Ephemeral state |
| Platform (Go) | `:8080` | Control plane API |
| Canvas (Next.js) | `:3000` | Visual frontend |
| Langfuse web | `:3001` (host) / `:3000` (internal) | Observability UI |
| Langfuse worker | — | Background processing |
| ClickHouse | — | Langfuse dependency |

Each workspace container is provisioned **on demand** by the platform when a user creates or imports one.

Langfuse uses a dedicated `langfuse` Postgres database. The compose stack creates it automatically before starting the Langfuse service, so it does not conflict with the platform's `molecule` schema.

### Infrastructure Only

To start just Postgres, Redis, and Langfuse (no application code):

```bash
docker compose -f docker-compose.infra.yml up
```

### Optional Profiles

```bash
docker compose --profile multi-provider up  # Add LiteLLM proxy (unified LLM API)
docker compose --profile local-models up    # Add Ollama (local LLM models)
```

## Environment Variables

### Platform (Go)

```
DATABASE_URL=postgres://dev:dev@postgres:5432/molecule?sslmode=prefer
REDIS_URL=redis://redis:6379
PORT=8080
SECRETS_ENCRYPTION_KEY=dev-key-change-in-production
WORKSPACE_DIR=/path/to/molecule-monorepo   # Optional global fallback; prefer per-workspace workspace_dir in org.yaml or API
```

### Canvas (Next.js)

```
NEXT_PUBLIC_PLATFORM_URL=http://localhost:8080
NEXT_PUBLIC_WS_URL=ws://localhost:8080/ws
```

### Workspace Runtime

```
WORKSPACE_ID=           # assigned by platform on provision
WORKSPACE_CONFIG_PATH=  # path to config folder inside container
MODEL_PROVIDER=         # e.g. anthropic:claude-sonnet-4-6
TIER=                   # 1, 2, 3, or 4
PLATFORM_URL=           # http://platform:8080
PARENT_ID=              # set by platform during team expansion (empty for top-level)
ANTHROPIC_API_KEY=      # or OPENAI_API_KEY, etc.
LANGFUSE_HOST=          # http://langfuse-web:3000 (internal container port; host-mapped to :3001)
LANGFUSE_PUBLIC_KEY=
LANGFUSE_SECRET_KEY=
LANGSMITH_TRACING=true  # LangGraph reads this to enable tracing
```

## Technology Versions

```
Go              1.25+ (go.mod)
Python          3.11+
Node.js         22+
Next.js         15
React Flow      12   (@xyflow/react)
a2a-sdk         0.3+ (A2A server SDK, install with a2a-sdk[http-server])
langfuse        3.x  (self-hosted Docker)
Postgres        16
Redis           7
Docker Compose  2.x
```

## Running Tests

### Unit Tests

```bash
cd workspace-server && go test -race ./...               # Go tests with race detection (695 tests)
cd canvas && npm test                            # Vitest tests (357 tests)
cd workspace && python -m pytest -v     # Workspace runtime tests (1140 tests)
cd sdk/python && python -m pytest -v             # SDK tests (121 tests)
cd mcp-server && npm test                        # MCP server tests (97 Jest tests)
```

### Integration Tests

```bash
bash tests/e2e/test_api.sh                 # 62 API tests (quick local verify; Phase 30.1 bearer-auth aware; also runs in CI)
bash tests/e2e/test_a2a_e2e.sh             # 22 A2A e2e tests (requires platform + 2 agents)
bash tests/e2e/test_activity_e2e.sh        # 25 activity/task E2E tests (requires platform + 1 agent)
bash tests/e2e/test_comprehensive_e2e.sh   # 67 endpoint/memory/bundle/approval checks
```

All E2E scripts share `tests/e2e/_lib.sh` helpers and are shellcheck-clean (enforced in CI). See [`./testing-e2e.md`](./testing-e2e.md) for auth prerequisites and what CI runs.

### CI Pipeline

GitHub Actions runs automatically on push to `main` and on PRs (`.github/workflows/ci.yml`):
- **platform-build** — Go build, vet, `go test -race` with coverage profiling (25% baseline threshold; setup-go uses module cache)
- **canvas-build** — npm build, `vitest run` (no `--passWithNoTests` -- tests must exist and pass)
- **mcp-server-build** — npm build
- **python-lint** — `pytest --cov=. --cov-report=term-missing` (pytest-cov enabled)
- **e2e-api** (added 2026-04-13) — Postgres + Redis service containers, migrations applied via `docker exec`, `tests/e2e/test_api.sh` must pass 62/62
- **shellcheck** (added 2026-04-13) — lints every `tests/e2e/*.sh`

Postgres and Redis are not exposed to the host -- use `docker compose exec postgres psql` or `docker compose exec redis redis-cli` for direct access.

## Utility Scripts

| Script | Purpose |
|--------|---------|
| `infra/scripts/setup.sh` | Initialize the local environment |
| `infra/scripts/nuke.sh` | Tear down and clean up everything |
| `bundle-compile.sh` | Compile workspace config folders into `.bundle.json` files |
| `test_api.sh` | Run 62 platform API integration tests |
| `test_a2a_e2e.sh` | Run 22 A2A end-to-end tests |
| `test_activity_e2e.sh` | Run 25 activity/task E2E tests |
| `setup-org.sh` | Create default 15-agent org hierarchy (PM + Marketing/Research/Dev teams, all Claude Code) |

## Related Docs

- [Architecture](../architecture/architecture.md) — System overview
- [Observability](./observability.md) — Langfuse details
