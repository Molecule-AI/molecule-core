# PR #1714 — Canvas: Require Hermes Model at Create
**Source:** PR #1714 merged to `origin/main` (2026-04-23)
**Status:** CHANGELOG — no marketing campaign warranted
**Type:** Bug fix / UX improvement

## Summary
Canvas workspace creation dialog now requires a model selection before submitting. Previously, omitting the model caused a silent Anthropic 401 error — the workspace would fail without a clear user-facing error. Now the dialog enforces model selection at create time and sends the model to the control plane.

**Files changed:** `canvas/src/components/CreateWorkspaceDialog.tsx` (+90, -17)

## Marketing action
None. Internal bug fix. Add to release notes / changelog only.

## Content angle
N/A — bug fix, not a feature

