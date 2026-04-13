# Hermes Borrowing Roadmap Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bring the highest-leverage Hermes-style improvements into Molecule AI: a clearer local startup path, better capability/onboarding discovery in Canvas, and a minimal external event ingress story.

**Architecture:** Keep changes staged and independently shippable. First strengthen the CLI/docs path, then expose capability and onboarding affordances in Canvas, then add runtime/platform primitives for webhook ingress and richer capability visibility. Avoid broad refactors; build on existing handlers, stores, and agent card publication.

**Tech Stack:** Go + Cobra CLI, Go/Gin platform handlers, Next.js 15 + Zustand canvas, Python workspace runtime, Docker-based verification.

---

## File Map

### CLI / Docs
- Modify: `README.md`
- Modify: `README.zh-CN.md`
- Modify: `platform/cmd/cli/commands.go`
- Modify: `platform/cmd/cli/cmd_doctor.go`
- Modify: `platform/cmd/cli/doctor.go`
- Modify: `platform/cmd/cli/doctor_test.go`

### Canvas UX
- Modify: `canvas/src/components/EmptyState.tsx`
- Modify: `canvas/src/components/Toolbar.tsx`
- Modify: `canvas/src/components/tabs/ChatTab.tsx`
- Modify: `canvas/src/components/SidePanel.tsx`
- Modify: `canvas/src/store/canvas.ts`
- Modify: `canvas/src/types/activity.ts` if capability summary types need extraction
- Test: `canvas/src/store/__tests__/canvas.test.ts`
- Modify: `docs/frontend/canvas.md`

### Capability Visibility / Platform
- Modify: `workspace-template/main.py`
- Modify: `workspace-template/config.py`
- Modify: `workspace-template/agent.py`
- Add: `workspace-template/preflight.py`
- Add: `workspace-template/tests/test_preflight.py`
- Modify: `platform/internal/models/workspace.go`
- Modify: `platform/internal/handlers/templates.go`
- Modify: `platform/internal/router/router.go`
- Add: `platform/internal/handlers/webhooks.go`
- Add: `platform/internal/handlers/webhooks_test.go`
- Modify: `docs/agent-runtime/cli-runtime.md`
- Modify: `docs/agent-runtime/config-format.md`
- Modify: `docs/api-protocol/platform-api.md`

---

## Chunk 1: Tighten the Local Startup Path

### Task 1: Expand `doctor` to cover the real local path

**Files:**
- Modify: `platform/cmd/cli/doctor.go`
- Modify: `platform/cmd/cli/cmd_doctor.go`
- Test: `platform/cmd/cli/doctor_test.go`

- [ ] **Step 1: Write the failing tests for the next doctor checks**

Add tests for:
- `migrations` directory discovery
- `workspace-configs-templates` warning vs fail behavior
- JSON output shape for `--json`

- [ ] **Step 2: Run the CLI tests to verify they fail**

Run:
```bash
docker run --rm -v /Users/aricredemption/Projects/molecule-monorepo:/workspace -w /workspace/platform golang:1.25.0 go test ./cmd/cli
```

Expected: FAIL in the new doctor test cases.

- [ ] **Step 3: Implement the smallest useful doctor additions**

Add:
- migrations directory check
- optional `--json` coverage verification if output formatting needs adjustment
- keep checks flat and synchronous; do not introduce a plugin framework

- [ ] **Step 4: Run the CLI tests to verify they pass**

Run:
```bash
docker run --rm -v /Users/aricredemption/Projects/molecule-monorepo:/workspace -w /workspace/platform golang:1.25.0 go test ./cmd/cli
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add platform/cmd/cli/cmd_doctor.go platform/cmd/cli/doctor.go platform/cmd/cli/doctor_test.go
git commit -m "feat(cli): expand doctor startup checks"
```

### Task 2: Make the quickstart path explicit in docs

**Files:**
- Modify: `README.md`
- Modify: `README.zh-CN.md`

- [ ] **Step 1: Write the docs-first delta**

Add a short "recommended path" section:
- `./infra/scripts/setup.sh`
- `molecli doctor`
- `go run ./cmd/server`
- `npm run dev`
- deploy a template from Canvas

- [ ] **Step 2: Verify the docs are accurate against existing commands**

Run:
```bash
rg -n "setup.sh|molecli doctor|go run ./cmd/server|npm run dev" README.md README.zh-CN.md
```

Expected: the new quickstart path appears in both READMEs.

- [ ] **Step 3: Commit**

```bash
git add README.md README.zh-CN.md
git commit -m "docs: add explicit local quickstart path"
```

---

## Chunk 2: Surface Onboarding and Capability Discovery in Canvas

### Task 3: Upgrade the empty state from hint list to startup flow

**Files:**
- Modify: `canvas/src/components/EmptyState.tsx`
- Modify: `docs/frontend/canvas.md`

- [ ] **Step 1: Write the expected content and behavior**

Target:
- a short 3-step startup flow
- one clear primary action
- references to template palette, search, and drag-to-nest

- [ ] **Step 2: Implement the empty-state refresh**

Keep it static first. No new API calls.

- [ ] **Step 3: Verify the app still builds**

Run:
```bash
cd canvas && npm run build
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add canvas/src/components/EmptyState.tsx docs/frontend/canvas.md
git commit -m "feat(canvas): turn empty state into onboarding flow"
```

### Task 4: Add a toolbar quick-actions / cheatsheet surface

**Files:**
- Modify: `canvas/src/components/Toolbar.tsx`
- Optionally modify: `canvas/src/components/Tooltip.tsx`

- [ ] **Step 1: Write the interaction expectations**

Support:
- visible help affordance in toolbar
- quick reminders for `⌘K`, template palette, right-click, resume chat, config location

- [ ] **Step 2: Implement the smallest UI surface**

Prefer a compact popover/panel over a full modal. Do not add routing.

- [ ] **Step 3: Verify the app still builds**

Run:
```bash
cd canvas && npm run build
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add canvas/src/components/Toolbar.tsx
git commit -m "feat(canvas): add quick actions help surface"
```

### Task 5: Make chat resume and capability visibility discoverable

**Files:**
- Modify: `canvas/src/components/tabs/ChatTab.tsx`
- Modify: `canvas/src/components/SidePanel.tsx`
- Modify: `canvas/src/store/canvas.ts`
- Test: `canvas/src/store/__tests__/canvas.test.ts`

- [ ] **Step 1: Write failing store or rendering expectations**

Cover:
- resumed task state is visible as such
- capability summary can be derived from workspace data / agent card without extra fetches

- [ ] **Step 2: Run the canvas tests to verify they fail**

Run:
```bash
cd canvas && npm test -- --runInBand
```

Expected: FAIL in new capability/resume expectations.

- [ ] **Step 3: Implement minimal resume and capability summary UI**

Target:
- show "resume current run" or equivalent when `currentTask` exists
- expose a concise capability summary near the panel header or chat tab
- use existing `agentCard`, `tier`, `status`, `currentTask`; do not add a new API yet

- [ ] **Step 4: Re-run tests and build**

Run:
```bash
cd canvas && npm test -- --runInBand
cd canvas && npm run build
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add canvas/src/components/tabs/ChatTab.tsx canvas/src/components/SidePanel.tsx canvas/src/store/canvas.ts canvas/src/store/__tests__/canvas.test.ts
git commit -m "feat(canvas): surface resume state and capability summary"
```

---

## Chunk 3: Strengthen Runtime Capability Metadata and Preflight

### Task 6: Add runtime preflight as a reusable Python primitive

**Files:**
- Add: `workspace-template/preflight.py`
- Modify: `workspace-template/main.py`
- Modify: `workspace-template/config.py`
- Add: `workspace-template/tests/test_preflight.py`

- [ ] **Step 1: Write the failing Python tests**

Cover:
- config-level preflight for required env / runtime prerequisites
- minimal capability snapshot generation from runtime config

- [ ] **Step 2: Run the workspace tests to verify they fail**

Run:
```bash
cd workspace-template && pytest tests/test_preflight.py -q
```

Expected: FAIL

- [ ] **Step 3: Implement the smallest preflight layer**

Scope:
- no new CLI yet
- reusable function that validates config/runtime assumptions before startup
- emits a compact capability/preflight summary for later publication

- [ ] **Step 4: Run the focused tests**

Run:
```bash
cd workspace-template && pytest tests/test_preflight.py -q
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add workspace-template/preflight.py workspace-template/main.py workspace-template/config.py workspace-template/tests/test_preflight.py
git commit -m "feat(runtime): add workspace preflight primitives"
```

### Task 7: Publish richer capability metadata from runtime to platform

**Files:**
- Modify: `workspace-template/main.py`
- Modify: `workspace-template/agent.py`
- Modify: `platform/internal/models/workspace.go`
- Possibly modify: `platform/internal/handlers/registry.go`
- Possibly modify: `canvas/src/store/canvas.ts`

- [ ] **Step 1: Write failing tests where coverage exists**

Cover:
- agent card / capability metadata shape
- store handling if any new fields are added to workspace payloads

- [ ] **Step 2: Implement metadata expansion**

Add only compact, durable fields such as:
- runtime kind
- tool modes
- session continuity support
- sandbox/backend hints
- preflight warnings count if appropriate

Do not publish deep provider internals or secrets.

- [ ] **Step 3: Verify focused tests**

Run:
```bash
docker run --rm -v /Users/aricredemption/Projects/molecule-monorepo:/workspace -w /workspace/platform golang:1.25.0 go test ./internal/handlers ./internal/router
cd workspace-template && pytest tests/test_preflight.py tests/test_prompt.py -q
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add workspace-template/main.py workspace-template/agent.py platform/internal/models/workspace.go platform/internal/handlers/registry.go canvas/src/store/canvas.ts
git commit -m "feat(runtime): publish richer workspace capability metadata"
```

---

## Chunk 4: Add a Minimal External Event Ingress

### Task 8: Introduce webhook endpoint scaffolding in platform

**Files:**
- Add: `platform/internal/handlers/webhooks.go`
- Add: `platform/internal/handlers/webhooks_test.go`
- Modify: `platform/internal/router/router.go`
- Modify: `docs/api-protocol/platform-api.md`

- [ ] **Step 1: Write failing handler tests**

Cover:
- accepts a basic signed or token-protected webhook request
- validates target workspace
- stores or forwards a normalized event payload
- rejects malformed or unauthorized requests

- [ ] **Step 2: Run the focused Go tests to verify they fail**

Run:
```bash
docker run --rm -v /Users/aricredemption/Projects/molecule-monorepo:/workspace -w /workspace/platform golang:1.25.0 go test ./internal/handlers -run Webhook -v
```

Expected: FAIL

- [ ] **Step 3: Implement minimal ingress**

Scope:
- one generic endpoint such as `POST /workspaces/:id/webhooks/events`
- one normalization path
- simple authentication guard
- no provider-specific adapters yet

- [ ] **Step 4: Re-run focused tests**

Run:
```bash
docker run --rm -v /Users/aricredemption/Projects/molecule-monorepo:/workspace -w /workspace/platform golang:1.25.0 go test ./internal/handlers -run Webhook -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add platform/internal/handlers/webhooks.go platform/internal/handlers/webhooks_test.go platform/internal/router/router.go docs/api-protocol/platform-api.md
git commit -m "feat(platform): add generic webhook ingress endpoint"
```

### Task 9: Connect webhook ingress to runtime-facing task handling

**Files:**
- Modify: `workspace-template/main.py`
- Modify: `workspace-template/config.py`
- Modify: `docs/agent-runtime/cli-runtime.md`
- Modify: `docs/agent-runtime/config-format.md`

- [ ] **Step 1: Define the smallest runtime contract**

Support:
- webhook event arrives at platform
- platform forwards a normalized task payload to workspace A2A or activity/task path
- runtime can distinguish webhook-originated work from chat-originated work if needed

- [ ] **Step 2: Implement only the minimum required runtime/config hooks**

Do not add provider-specific webhook logic. Keep the runtime generic.

- [ ] **Step 3: Verify focused tests and docs**

Run:
```bash
cd workspace-template && pytest tests/test_config.py tests/test_a2a_executor.py -q
docker run --rm -v /Users/aricredemption/Projects/molecule-monorepo:/workspace -w /workspace/platform golang:1.25.0 go test ./internal/handlers -run Webhook -v
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add workspace-template/main.py workspace-template/config.py docs/agent-runtime/cli-runtime.md docs/agent-runtime/config-format.md
git commit -m "feat(runtime): wire webhook ingress into workspace tasks"
```

---

## Dependencies and Order

1. Expand `doctor`
2. Update quickstart docs
3. Refresh Canvas empty state
4. Add toolbar help
5. Add chat resume + capability summary
6. Add runtime preflight primitives
7. Publish richer capability metadata
8. Add generic webhook ingress
9. Wire webhook ingress into runtime task handling

Reasoning:
- Steps 1-5 improve discoverability without increasing backend coupling.
- Steps 6-7 give us a stable capability/preflight contract before Canvas or integrations rely on richer metadata.
- Steps 8-9 add the external ingress path only after visibility and runtime metadata are in place.

## Atomic Commit Rules

- Each commit must change one user-visible concern only.
- No mixed docs + platform + canvas + runtime commit unless the docs only describe the code introduced in the same commit.
- Every commit must have at least one focused verification command run before moving on.
- If a task unexpectedly spans two subsystems, split by boundary and commit the provider-side primitive before the consumer-side UI.

## Verification Matrix

- CLI:
```bash
docker run --rm -v /Users/aricredemption/Projects/molecule-monorepo:/workspace -w /workspace/platform golang:1.25.0 go test ./cmd/cli
```

- Platform handlers/router:
```bash
docker run --rm -v /Users/aricredemption/Projects/molecule-monorepo:/workspace -w /workspace/platform golang:1.25.0 go test ./internal/handlers ./internal/router
```

- Canvas:
```bash
cd canvas && npm test -- --runInBand
cd canvas && npm run build
```

- Runtime:
```bash
cd workspace-template && pytest -q
```
