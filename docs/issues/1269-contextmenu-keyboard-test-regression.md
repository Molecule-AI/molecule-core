# Canvas test regression: ContextMenu keyboard test failure (Issue #1269)

## Status
- **Issue**: #1269
- **Type**: Bug / Test Regression
- **Owner**: Core-FE
- **Unfixed since**: Round 20
- **GH auth blocked**: 401 — filed as local markdown on 2026-04-21

## Problem

The ContextMenu keyboard test (`ContextMenu.keyboard.test.tsx`) expects
`setPendingDelete` to be called with `{id, name}` but PR #1243 changed the
actual call to include `hasChildren: false`.

The test now fails because its assertion does not account for the new field.

## Root Cause

PR #1243 added `hasChildren: false` to the `setPendingDelete` call in
`ContextMenu.tsx` to enable the parent-warns-before-delete UX, but did not
update the corresponding test expectation.

## Affected Files

1. `canvas/src/components/__tests__/ContextMenu.keyboard.test.tsx` — **test needs update**
2. `canvas/src/components/ContextMenu.tsx` — already correct (no change needed)

## Fix Needed

Update the test expectation to include `hasChildren: false`:

```ts
// Before (failing):
expect(mockStore.setPendingDelete).toHaveBeenCalledWith({
  id: "ws-1",
  name: "Alpha Workspace",
});

// After (correct):
expect(mockStore.setPendingDelete).toHaveBeenCalledWith({
  id: "ws-1",
  name: "Alpha Workspace",
  hasChildren: false,
});
```

**Existing fix branch**: `fix/canvas-test-regressions-pr1243` (commit `f8a6dae`:
"fix(canvas/test): patch regressed tests from PR #1243 orgs-page flakiness fix")

## Notes

- Fix already exists on `fix/canvas-test-regressions-pr1243` (not yet merged to staging)
- The `hasChildren: false` field was intentionally added by PR #1243 to support
  cascade-delete UX warning — the test needs to catch up, not the implementation
