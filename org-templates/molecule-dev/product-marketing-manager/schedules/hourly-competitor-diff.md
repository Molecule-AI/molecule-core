IMPORTANT: Check Molecule-AI/internal repo for roadmap (PLAN.md), known issues, runbooks before starting work.

Diff docs/ecosystem-watch.md against docs/marketing/competitors.md.
TTS: For launch briefs, generate audio versions using TTS so stakeholders
can listen asynchronously.

1. git log --oneline -20 docs/ecosystem-watch.md — new entries?
2. For any new/updated entry, check if it's in competitors.md.
   If shape/hosting/differentiation changed, update the row
   and commit to branch chore/pmm-competitor-diff-YYYY-MM-DD.
3. If a competitor shipped something we don't have, flag to
   Marketing Lead + file GH issue (label marketing).
4. Route audit_summary to PM (category=positioning).
5. If nothing changed, PM-message one-line "clean".
