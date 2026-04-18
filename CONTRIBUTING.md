# Contributing to Molecule AI

Thanks for your interest in contributing to Molecule AI! This guide covers the
development workflow, conventions, and how to get your changes merged.

## Getting Started

### Prerequisites

- **Go 1.25+** — platform backend
- **Node.js 20+** — canvas frontend
- **Python 3.11+** — workspace runtime
- **Docker** — infrastructure services (Postgres, Redis)
- **Git** — with hooks path set to `.githooks`

### Setup

```bash
# Clone the repo
git clone https://github.com/Molecule-AI/molecule-monorepo.git
cd molecule-monorepo

# Install git hooks
git config core.hooksPath .githooks

# Start infrastructure (Postgres, Redis, Langfuse, Temporal)
./infra/scripts/setup.sh

# Build and run the platform
cd workspace-server
go run ./cmd/server

# In a separate terminal, run the canvas
cd canvas
npm install
npm run dev
```

### Environment Variables

Copy `.env.example` to `.env` and fill in your values:
```bash
cp .env.example .env
```

See `CLAUDE.md` for a full list of environment variables and their purposes.

## Development Workflow

### Branch Naming

Use prefixed branches:
- `feat/` — new features
- `fix/` — bug fixes
- `chore/` — maintenance, deps, CI
- `docs/` — documentation only

**Never push directly to `main`.** All changes go through pull requests.

### Commits

Write concise commit messages that focus on the "why":
```
fix(canvas): prevent infinite re-render on WebSocket reconnect

The useEffect dependency array included the entire nodes object,
causing a render loop when any node position changed.
```

### Pull Requests

- Keep PRs focused — one concern per PR
- Include a test plan in the PR description
- PRs are merged with **merge commits** (not squash or rebase)

### Running Tests

```bash
# Go (platform)
cd workspace-server && go test -race ./...

# Canvas (Next.js)
cd canvas && npm test

# Workspace runtime (Python)
cd workspace && python -m pytest -v

# E2E API tests (requires running platform)
bash tests/e2e/test_api.sh
```

### Pre-commit Hooks

The `.githooks/pre-commit` hook enforces:
- `'use client'` directive on React hook files
- Dark theme only (no white/light CSS classes)
- No SQL injection patterns (`fmt.Sprintf` with SQL)
- No leaked secrets (`sk-ant-`, `ghp_`, `AKIA`)

Fix violations before committing — the hook will reject the commit.

### CI Pipeline

CI runs on GitHub Actions with a self-hosted runner. External contributors:
PRs from forks will not trigger CI automatically. A maintainer will review
and run CI manually.

| Job | What it checks |
|-----|---------------|
| platform-build | Go build + vet + `go test -race` |
| canvas-build | npm build + vitest |
| python-lint | pytest with coverage |
| e2e-api | Full API test suite (62 tests) |
| shellcheck | Shell script linting |

## Code Style

### Go (Platform)
- Standard `gofmt` formatting
- `go vet` must pass
- No `fmt.Sprintf` in SQL queries (use parameterized queries)
- Prefer function injection over import cycles

### TypeScript (Canvas)
- Strict mode enabled
- No `any` types (use `unknown` or proper types)
- Use `ConfirmDialog` component, never native `confirm/alert/prompt`
- Dark theme only — no white/light CSS classes

### Python (Workspace Runtime)
- Type hints on public functions
- pytest for all tests

## Architecture Overview

See `CLAUDE.md` for detailed architecture documentation, including:
- Component diagram (Platform, Canvas, Workspace Runtime)
- Key architectural patterns
- Database schema and migrations
- API route reference

## Reporting Issues

Use GitHub Issues with a clear title and reproduction steps. Include:
- What you expected
- What actually happened
- Platform/OS version
- Relevant logs or screenshots

## Security

If you discover a security vulnerability, please report it privately via
GitHub Security Advisories rather than opening a public issue.

## License

By contributing, you agree that your contributions will be licensed under the
same [Business Source License 1.1](LICENSE) that covers this project.
