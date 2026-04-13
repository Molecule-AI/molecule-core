# Workspace Awareness Integration Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add workspace-scoped awareness access so every newly created workspace gets its own isolated awareness namespace while reusing the existing memory tool surface.

**Architecture:** Keep awareness as a shared backend service, not one service per workspace. The platform creates and stores a workspace awareness namespace during provisioning, injects awareness connection settings into the workspace container, and the runtime maps its existing memory tools onto that namespace. This preserves the current agent-facing contract while giving each workspace isolated memory and a clean upgrade path to stricter tenancy later.

**Tech Stack:** Go platform handlers/provisioner, Python workspace runtime, existing workspace memory tools, Postgres-backed workspace metadata, awareness MCP/service integration.

---

## Chunk 1: Define Workspace Awareness Metadata and Provisioning Inputs

This chunk gives the platform a durable awareness identity for every workspace and makes sure the container receives it at startup.

### Task 1: Extend the workspace create flow to assign an awareness namespace

**Files:**
- Modify: `platform/internal/handlers/workspace.go`
- Modify: `platform/internal/models/workspace.go`
- Modify: `platform/internal/handlers/handlers_test.go`

- [ ] **Step 1: Write the failing test**

Add a handler test that creates a workspace and asserts the response or DB state contains a stable awareness namespace derived from the new workspace ID.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./platform/internal/handlers -run TestWorkspaceCreate_AssignsAwarenessNamespace -v`
Expected: FAIL because the namespace field does not exist yet.

- [ ] **Step 3: Write minimal implementation**

Generate a namespace from the new workspace ID in `Create`, persist it with the workspace record, and return it in the created workspace payload if the API already exposes workspace metadata.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./platform/internal/handlers -run TestWorkspaceCreate_AssignsAwarenessNamespace -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add platform/internal/handlers/workspace.go platform/internal/models/workspace.go platform/internal/handlers/handlers_test.go
git commit -m "feat(platform): assign awareness namespace per workspace"
```

### Task 2: Inject awareness settings into workspace provisioning

**Files:**
- Modify: `platform/internal/provisioner/provisioner.go`
- Modify: `platform/internal/handlers/workspace.go`
- Modify: `platform/internal/handlers/handlers_test.go`

- [ ] **Step 1: Write the failing test**

Add a provisioner test that asserts the container env includes `AWARENESS_URL` and `AWARENESS_NAMESPACE` for a workspace start request.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./platform/internal/provisioner -run TestStart_InjectsAwarenessEnv -v`
Expected: FAIL because those env vars are not present yet.

- [ ] **Step 3: Write minimal implementation**

Add awareness URL and namespace to `WorkspaceConfig`, pass them from the workspace create handler, and inject them into the container environment in `Start`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./platform/internal/provisioner -run TestStart_InjectsAwarenessEnv -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add platform/internal/provisioner/provisioner.go platform/internal/handlers/workspace.go platform/internal/handlers/handlers_test.go
git commit -m "feat(platform): inject awareness config into workspaces"
```

## Chunk 2: Add Awareness Backend Wiring in the Workspace Runtime

This chunk keeps the agent-facing tools stable and swaps the backend behind them.

### Task 3: Add an awareness client abstraction to the runtime

**Files:**
- Create: `workspace-template/builtin_tools/awareness_client.py`
- Modify: `workspace-template/builtin_tools/memory.py`
- Modify: `workspace-template/main.py`
- Modify: `workspace-template/tests/test_memory.py` or a new awareness-focused test file

- [ ] **Step 1: Write the failing test**

Add unit tests that verify `commit_memory` and `search_memory` call the awareness client when `AWARENESS_URL` and `AWARENESS_NAMESPACE` are present, and fall back cleanly when they are absent.

- [ ] **Step 2: Run test to verify it fails**

Run: `pytest workspace-template/tests -k awareness -v`
Expected: FAIL because the client module and branch logic do not exist yet.

- [ ] **Step 3: Write minimal implementation**

Create a tiny client wrapper that reads awareness env vars, exposes `commit` and `search`, and let `memory.py` delegate through it while preserving the current tool signatures.

- [ ] **Step 4: Run test to verify it passes**

Run: `pytest workspace-template/tests -k awareness -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add workspace-template/builtin_tools/awareness_client.py workspace-template/builtin_tools/memory.py workspace-template/main.py workspace-template/tests/test_memory.py
git commit -m "feat(runtime): route memory tools through awareness client"
```

### Task 4: Preserve the local fallback path for non-aware workspaces

**Files:**
- Modify: `workspace-template/builtin_tools/memory.py`
- Modify: `workspace-template/tests/test_memory.py`

- [ ] **Step 1: Write the failing test**

Add tests covering the no-awareness case so older or partially provisioned workspaces still behave safely.

- [ ] **Step 2: Run test to verify it fails**

Run: `pytest workspace-template/tests -k memory -v`
Expected: FAIL until fallback behavior is implemented or verified.

- [ ] **Step 3: Write minimal implementation**

Ensure the tool either uses the platform-backed awareness service or, if unavailable, returns a clear error or existing fallback behavior instead of crashing.

- [ ] **Step 4: Run test to verify it passes**

Run: `pytest workspace-template/tests -k memory -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add workspace-template/builtin_tools/memory.py workspace-template/tests/test_memory.py
git commit -m "fix(runtime): keep memory tools resilient without awareness"
```

## Chunk 3: Document the Contract and Validate End-to-End

This chunk makes the design visible to future work and proves the full flow.

### Task 5: Update the memory architecture docs

**Files:**
- Modify: `docs/architecture/memory.md`
- Modify: `docs/agent-runtime/workspace-runtime.md`
- Modify: `docs/agent-runtime/cli-runtime.md`

- [ ] **Step 1: Write the failing review check**

Review the docs for any remaining wording that implies per-workspace instances instead of shared service plus namespace isolation.

- [ ] **Step 2: Run doc sanity check**

Run: `rg -n "per workspace|shared memory|awareness|namespace" docs/architecture/memory.md docs/agent-runtime/workspace-runtime.md docs/agent-runtime/cli-runtime.md`
Expected: The docs should clearly describe workspace-scoped awareness.

- [ ] **Step 3: Write minimal documentation update**

Explain the namespace model, the environment variables, and the fact that agent-facing tools stay stable while the backend changes.

- [ ] **Step 4: Run doc sanity check again**

Run: `rg -n "per workspace|shared memory|awareness|namespace" docs/architecture/memory.md docs/agent-runtime/workspace-runtime.md docs/agent-runtime/cli-runtime.md`
Expected: Wording matches the shared-service design.

- [ ] **Step 5: Commit**

```bash
git add docs/architecture/memory.md docs/agent-runtime/workspace-runtime.md docs/agent-runtime/cli-runtime.md
git commit -m "docs(memory): describe workspace-scoped awareness"
```

### Task 6: Verify workspace creation through runtime startup

**Files:**
- Modify: `workspace-template/tests/test_main.py` or add a focused startup test
- Potentially modify: `platform/internal/handlers/handlers_test.go`

- [ ] **Step 1: Write the failing test**

Add an integration-style test that creates a workspace, inspects the injected env/config, and confirms the runtime can start with awareness configured.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./platform/internal/handlers -run TestWorkspaceCreate_WithAwarenessConfig -v` and/or `pytest workspace-template/tests -k startup -v`
Expected: FAIL until the whole chain is wired.

- [ ] **Step 3: Write minimal implementation**

Close the gap between workspace creation, provisioning, and runtime startup so the awareness config is present end to end.

- [ ] **Step 4: Run test to verify it passes**

Run the same targeted Go and Python tests again.
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add platform/internal/handlers/handlers_test.go workspace-template/tests/test_main.py
git commit -m "test(workspace): cover awareness startup path"
```

## Final Verification

After all chunks are complete:

- Run the workspace-targeted Go tests
- Run the workspace-template Python tests
- Create a new workspace through the platform API
- Confirm the new workspace receives its own awareness namespace
- Confirm `commit_memory` and `search_memory` remain usable from the agent runtime
- Confirm docs match the implemented behavior

