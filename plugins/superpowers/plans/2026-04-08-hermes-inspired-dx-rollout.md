# Hermes-Inspired DX Rollout Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the highest-value Hermes-inspired developer experience improvements across CLI, Canvas, and runtime/platform without turning the codebase into a broad refactor.

**Architecture:** Keep the rollout in narrow vertical slices. Start with CLI and onboarding paths that improve default usage immediately, then expose workspace capabilities more clearly, then add a minimal webhook ingress path as a separate backend feature. Each slice should ship independently and leave the repo in a usable state.

**Tech Stack:** Go + Cobra CLI, Go + Gin platform, Next.js 15 + Zustand canvas, Python workspace runtime, Docker-based verification, existing platform HTTP APIs.

---

## File Map

### CLI / Platform

- Modify: `platform/cmd/cli/commands.go`
- Modify: `platform/cmd/cli/cmd_agent.go`
- Modify: `platform/cmd/cli/cmd_chat.go`
- Modify: `platform/cmd/cli/view.go`
- Modify: `platform/cmd/cli/client.go`
- Modify: `platform/cmd/cli/cli_test.go`
- Create or modify: `platform/cmd/cli/cmd_doctor.go`
- Create or modify: `platform/cmd/cli/doctor.go`
- Create or modify: `platform/cmd/cli/doctor_test.go`
- Modify later only if needed: `platform/internal/router/router.go`
- Modify later only if needed: `platform/internal/handlers/*.go`

### Canvas

- Modify: `canvas/src/components/EmptyState.tsx`
- Modify: `canvas/src/components/Toolbar.tsx`
- Modify: `canvas/src/components/SidePanel.tsx`
- Modify: `canvas/src/components/tabs/ChatTab.tsx`
- Modify: `canvas/src/components/tabs/DetailsTab.tsx`
- Modify: `canvas/src/store/canvas.ts`
- Modify if required: `canvas/src/types/activity.ts`
- Add if needed: `canvas/src/components/QuickHelpPopover.tsx`
- Add if needed: `canvas/src/components/CapabilitySummary.tsx`
- Modify tests if present or add: `canvas/src/store/__tests__/canvas.test.ts`

### Runtime / Platform Integration

- Modify: `workspace-template/main.py`
- Modify: `workspace-template/agent.py`
- Modify: `workspace-template/config.py`
- Modify if needed: `workspace-template/tests/test_config.py`
- Modify if needed: `workspace-template/tests/test_prompt.py`
- Modify if needed: `workspace-template/tests/test_a2a_executor.py`
- Modify: `platform/internal/router/router.go`
- Add: `platform/internal/handlers/webhooks.go`
- Add tests: `platform/internal/handlers/webhooks_test.go`
- Modify if needed: `platform/internal/models/workspace.go`

### Docs

- Modify: `README.md`
- Modify: `README.zh-CN.md`
- Modify: `docs/agent-runtime/cli-runtime.md`
- Modify: `docs/frontend/canvas.md`
- Modify: `docs/api-protocol/platform-api.md`
- Modify: `docs/edit-history/2026-04-08.md`

---

## Multi-Agent Execution Strategy

### Parallel lanes

- Lane A: CLI/default-path improvements
- Lane B: Canvas onboarding/help/resume UX
- Lane C: Capability summary plumbing
- Lane D: Webhook ingress backend
- Lane E: Docs pass after each shipped lane

### Shared-state rule

- Lanes A and B can run in parallel after agreeing on copy and naming.
- Lane C depends on whatever backend/runtime fields are already available; start after confirming whether current agent card payload is sufficient.
- Lane D must stay isolated from Lanes A and B. It touches backend API surface and should be implemented and reviewed separately.
- Docs commits should be separate and follow the feature commits they describe.

---

## Chunk 1: CLI Default Path

### Task 1: Finish the doctor command as the stable entry point

**Files:**
- Modify: `platform/cmd/cli/cmd_doctor.go`
- Modify: `platform/cmd/cli/doctor.go`
- Modify: `platform/cmd/cli/doctor_test.go`

- [ ] **Step 1: Write one more failing test for expected doctor output/JSON shape if a gap remains**

Run: `docker run --rm -v /Users/aricredemption/Projects/molecule-monorepo:/workspace -w /workspace/platform golang:1.25.0 go test ./cmd/cli`

Expected: failing test only if behavior is not yet locked.

- [ ] **Step 2: Implement only the missing doctor behavior**

Keep checks limited to the intended scope: health, Postgres, Redis, templates, Docker.

- [ ] **Step 3: Run CLI tests**

Run: `docker run --rm -v /Users/aricredemption/Projects/molecule-monorepo:/workspace -w /workspace/platform golang:1.25.0 go test ./cmd/cli`

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add platform/cmd/cli/cmd_doctor.go platform/cmd/cli/doctor.go platform/cmd/cli/doctor_test.go platform/cmd/cli/commands.go platform/cmd/cli/main.go
git commit -m "feat(cli): add doctor preflight checks"
```

### Task 2: Add the guided CLI quickstart path

**Files:**
- Modify: `platform/cmd/cli/commands.go`
- Modify: `platform/cmd/cli/cmd_agent.go`
- Modify: `platform/cmd/cli/cmd_chat.go`
- Modify: `platform/cmd/cli/view.go`
- Modify: `platform/cmd/cli/cli_test.go`

- [ ] **Step 1: Write a failing test for the new command/help path**

Examples:
- root help should mention `doctor`
- agent help should expose the recommended `spawn -> chat` flow
- optional `molecli quickstart` should render deterministic guidance

- [ ] **Step 2: Run the targeted test**

Run: `docker run --rm -v /Users/aricredemption/Projects/molecule-monorepo:/workspace -w /workspace/platform golang:1.25.0 go test ./cmd/cli -run 'Test.*Quickstart|Test.*Doctor'`

Expected: FAIL

- [ ] **Step 3: Implement the minimum path**

Preferred shape:
- either a dedicated `molecli quickstart` command
- or a stronger root help and `agent` subcommand examples

Do not add a wizard or interactive setup flow.

- [ ] **Step 4: Run CLI tests**

Run: `docker run --rm -v /Users/aricredemption/Projects/molecule-monorepo:/workspace -w /workspace/platform golang:1.25.0 go test ./cmd/cli`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add platform/cmd/cli/commands.go platform/cmd/cli/cmd_agent.go platform/cmd/cli/cmd_chat.go platform/cmd/cli/view.go platform/cmd/cli/cli_test.go
git commit -m "feat(cli): add guided quickstart path"
```

---

## Chunk 2: Canvas Onboarding and Help

### Task 3: Turn the empty state into a real onboarding panel

**Files:**
- Modify: `canvas/src/components/EmptyState.tsx`

- [ ] **Step 1: Add a failing UI test if the repo already has a clear pattern for component testing**

If there is no stable component-test pattern, skip new component tests and rely on build verification for this task.

- [ ] **Step 2: Replace the current generic empty copy with a Hermes-style start path**

Required content:
- start with template palette
- run `molecli doctor`
- create first workspace
- open chat/config after deploy

Do not add new backend dependencies.

- [ ] **Step 3: Run frontend verification**

Run: `npm test -- --runInBand` from `canvas/` only if that command is already healthy, otherwise use `npm run build`.

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add canvas/src/components/EmptyState.tsx
git commit -m "feat(canvas): add guided empty-state onboarding"
```

### Task 4: Add toolbar help and cheatsheet surfacing

**Files:**
- Modify: `canvas/src/components/Toolbar.tsx`
- Add if needed: `canvas/src/components/QuickHelpPopover.tsx`
- Modify if needed: `canvas/src/store/canvas.ts`

- [ ] **Step 1: Define the minimal help surface**

Include only:
- `⌘K`
- template palette
- right-click actions
- chat sessions/resume
- config/secrets location

- [ ] **Step 2: Implement the popover or inline panel**

Keep state local unless a shared store is clearly necessary.

- [ ] **Step 3: Run frontend verification**

Run: `npm run build` from `canvas/`

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add canvas/src/components/Toolbar.tsx canvas/src/components/QuickHelpPopover.tsx canvas/src/store/canvas.ts
git commit -m "feat(canvas): add toolbar quick help"
```

### Task 5: Make chat resume discoverable

**Files:**
- Modify: `canvas/src/components/tabs/ChatTab.tsx`

- [ ] **Step 1: Write a failing test only if session behavior can be covered cheaply**

Otherwise skip to implementation and rely on build verification.

- [ ] **Step 2: Surface resume state explicitly**

Examples:
- banner when `currentTask` exists
- label for the active session being resumed
- clearer wording around session list and continued polling

Do not re-architect chat transport in this task.

- [ ] **Step 3: Run frontend verification**

Run: `npm run build` from `canvas/`

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add canvas/src/components/tabs/ChatTab.tsx
git commit -m "feat(canvas): surface chat resume state"
```

---

## Chunk 3: Capability Summary Surfacing

### Task 6: Expose a compact workspace capability summary in the side panel

**Files:**
- Modify: `canvas/src/components/SidePanel.tsx`
- Modify: `canvas/src/components/tabs/DetailsTab.tsx`
- Add if needed: `canvas/src/components/CapabilitySummary.tsx`

- [ ] **Step 1: Confirm existing fields are enough**

Prefer using:
- agent card skills
- tier
- status
- active task
- URL/runtime hints already present in config/details

Do not add backend fields if current data is sufficient.

- [ ] **Step 2: Implement the summary UI**

Target output:
- what the workspace is
- what it can do now
- where to configure more

Avoid long cards or dense metadata dumps.

- [ ] **Step 3: Run frontend verification**

Run: `npm run build` from `canvas/`

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add canvas/src/components/SidePanel.tsx canvas/src/components/tabs/DetailsTab.tsx canvas/src/components/CapabilitySummary.tsx
git commit -m "feat(canvas): add workspace capability summary"
```

### Task 7: Add backend/runtime capability fields only if the UI needs more than the current agent card

**Files:**
- Modify: `workspace-template/main.py`
- Modify if needed: `platform/internal/handlers/registry.go`
- Modify if needed: `platform/internal/models/workspace.go`
- Modify if needed: `canvas/src/store/canvas.ts`
- Modify tests as needed

- [ ] **Step 1: Write a failing backend/runtime test for the new capability field**

Only do this if UI work proved the current payload is insufficient.

- [ ] **Step 2: Add the minimum new agent-card or workspace field**

Candidate fields:
- runtime name
- enabled tool classes
- webhook support boolean

Do not expose internal implementation noise.

- [ ] **Step 3: Run targeted backend/runtime tests**

Run the smallest relevant test command first, then broaden.

- [ ] **Step 4: Commit**

```bash
git add workspace-template/main.py platform/internal/handlers/registry.go platform/internal/models/workspace.go canvas/src/store/canvas.ts
git commit -m "feat(platform): expose workspace capability metadata"
```

---

## Chunk 4: Webhook Ingress

### Task 8: Add a minimal webhook endpoint on the platform

**Files:**
- Add: `platform/internal/handlers/webhooks.go`
- Add: `platform/internal/handlers/webhooks_test.go`
- Modify: `platform/internal/router/router.go`

- [ ] **Step 1: Write the failing handler test**

Scope the first version narrowly:
- one generic inbound webhook endpoint
- workspace target resolution from path or body
- optional shared-secret verification
- enqueue/proxy a simple task to the target workspace

- [ ] **Step 2: Run the failing test**

Run: `docker run --rm -v /Users/aricredemption/Projects/molecule-monorepo:/workspace -w /workspace/platform golang:1.25.0 go test ./internal/handlers -run TestWebhook`

Expected: FAIL

- [ ] **Step 3: Implement the smallest viable handler**

Keep v1 generic. Do not hardcode GitHub/Jira/Stripe-specific shapes yet.

- [ ] **Step 4: Run handler tests and broad platform tests that cover routing**

Run the smallest passing command first, then expand if safe.

- [ ] **Step 5: Commit**

```bash
git add platform/internal/handlers/webhooks.go platform/internal/handlers/webhooks_test.go platform/internal/router/router.go
git commit -m "feat(platform): add generic webhook ingress"
```

### Task 9: Teach the runtime/canvas to reflect webhook readiness

**Files:**
- Modify if needed: `workspace-template/config.py`
- Modify if needed: `workspace-template/main.py`
- Modify if needed: `canvas/src/components/CapabilitySummary.tsx`
- Modify docs as needed

- [ ] **Step 1: Add a failing test only if the capability signal is new**

- [ ] **Step 2: Surface a simple readiness signal**

Examples:
- `webhooks: enabled`
- `ingress: available`

Do not build webhook management UI in this step.

- [ ] **Step 3: Run relevant tests/build**

- [ ] **Step 4: Commit**

```bash
git add workspace-template/config.py workspace-template/main.py canvas/src/components/CapabilitySummary.tsx
git commit -m "feat(runtime): surface webhook capability"
```

---

## Chunk 5: Documentation

### Task 10: Update operator-facing docs after each shipped chunk

**Files:**
- Modify: `README.md`
- Modify: `README.zh-CN.md`
- Modify: `docs/agent-runtime/cli-runtime.md`
- Modify: `docs/frontend/canvas.md`
- Modify: `docs/api-protocol/platform-api.md`
- Modify: `docs/edit-history/2026-04-08.md`

- [ ] **Step 1: Document `molecli doctor` and the recommended local workflow**

- [ ] **Step 2: Document Canvas onboarding/help/capability summary behavior**

- [ ] **Step 3: Document webhook ingress once the API is stable**

- [ ] **Step 4: Run build or docs-adjacent verification if available**

- [ ] **Step 5: Commit in atomic docs-only slices**

Recommended commit split:

```bash
git commit -m "docs(cli): document doctor and quickstart flow"
git commit -m "docs(canvas): document onboarding and capability summary"
git commit -m "docs(api): document webhook ingress"
```

---

## Recommended Order

1. Chunk 1 Task 2 can start after the existing doctor commit.
2. Chunk 2 Task 3 and Task 4 can run in parallel.
3. Chunk 2 Task 5 depends on the final wording of Task 4 only if they share UX copy; otherwise parallelize.
4. Chunk 3 Task 6 should happen before Task 7.
5. Chunk 4 is isolated and should be done after the UI/CLI work is settled.
6. Chunk 5 follows each completed chunk as docs-only commits.

---

## Atomic Commit Policy

- One user-visible behavior change per commit.
- Do not mix backend API work with Canvas polish in the same commit.
- Do not mix docs with code unless the code is tiny and the docs are inseparable.
- Keep tests in the same commit as the behavior they protect.
- If a task reveals a required refactor, split it:
  - first commit: no-behavior-change refactor
  - second commit: behavior change

---

## Verification Matrix

- CLI work:
  - `docker run --rm -v /Users/aricredemption/Projects/molecule-monorepo:/workspace -w /workspace/platform golang:1.25.0 go test ./cmd/cli`

- Platform handler work:
  - `docker run --rm -v /Users/aricredemption/Projects/molecule-monorepo:/workspace -w /workspace/platform golang:1.25.0 go test ./internal/handlers`

- Canvas work:
  - `cd /Users/aricredemption/Projects/molecule-monorepo/canvas && npm run build`

- Runtime work:
  - Run the smallest relevant `pytest` target inside `workspace-template/` first, then broaden.

---

Plan complete and saved to `docs/superpowers/plans/2026-04-08-hermes-inspired-dx-rollout.md`. Ready to execute?
