# Canvas test regression: orgs-page error state test failure (Issue #1268)

## Status
- **Issue**: #1268
- **Type**: Bug / Test Regression
- **Owner**: Core-FE
- **Unfixed since**: Round 19
- **GH auth blocked**: 401 — filed as local markdown on 2026-04-21

## Problem

The orgs-page error state test is failing in CI. The test uses
`vi.advanceTimersByTimeAsync(50)` which does not guarantee React re-render has
settled before the assertion runs.

`advanceTimersByTimeAsync` only advances the timer queue — it does not wait for
React's scheduler to flush microtasks and complete the render pass. This causes
a race condition where assertions fire before the component tree reflects the
intended state.

## Root Cause

Introduced by PR #1243 which replaced `waitFor` polling with fake timers for
performance, but did not wrap the async timer advance in `act()` for the
error-state test path.

## Fix Needed

Restore `act()` wrapper (or `waitFor`) to ensure the render settles before
asserting:

```ts
// Before (flaky):
await vi.advanceTimersByTimeAsync(50);
// assertion runs before React render settles

// After (correct):
await act(async () => { await vi.advanceTimersByTimeAsync(50); });
await waitFor(() => expect(screen.getByText(/Error:/)).toBeInTheDocument());
```

**File**: `canvas/src/app/__tests__/orgs-page.test.tsx`

**Existing fix branch**: `fix/orgs-page-fake-timers-1345` (commit `3e3c02d`:
"fix(canvas/test): restore waitFor in orgs-page error test + add getState mock")

## Test Coverage Gap

The error-state render path needs a test that:
1. Uses `vi.useFakeTimers()`
2. Advances time past the error-threshold
3. Wraps the timer advance in `act()` or follows with `waitFor()`
4. Asserts the error UI is present

## Notes

- Fix already exists on `fix/orgs-page-fake-timers-1345` (not yet merged to staging)
- Second timer issue: between-test isolation — add `await vi.useRealTimers()` cleanup
- Related: `fix/flaky-orgs-page-tests-1207` branch covers the between-test isolation fix
