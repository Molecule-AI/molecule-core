---
name: cross-vendor-review
description: Run an adversarial code review against a non-Claude model (Codex / GPT / Gemini) and surface disagreements with Claude's own review. Use ONLY for noteworthy PRs (auth, billing, data-deletion, irreversible migration, large-blast-radius). Inspired by gstack's /codex command.
---

# cross-vendor-review

Two LLMs catch bugs one doesn't. Claude has blind spots; so does GPT-5; so does Gemini. For high-stakes PRs the cost of a second model is dwarfed by the cost of a missed defect.

## When to invoke

ALWAYS for PRs touching:
- Authentication, authorization, session, or token handling
- Billing / payments / Stripe / metering
- Destructive operations (delete cascades, mass-update, drop)
- Database migrations (schema changes, data backfills)
- Cross-tenant isolation logic
- Cryptographic primitives

OPTIONAL for:
- Large refactors (>500 LOC)
- Performance-sensitive changes
- Anything where the cron's standard code-review skill returned conflicting signals

NEVER for:
- Docs, templates, CI tweaks, dependency bumps, test-only changes

## How to invoke

1. Pull the diff: `gh pr diff N --repo OWNER/REPO`
2. Run Claude's own code-review skill first; capture its findings
3. Send the SAME diff + the SAME rubric to a second model:
   - Preferred order: GPT-5 (via Codex CLI or API), Gemini Pro 2.5, Llama 3.3 70B
   - One-shot prompt; no conversation
   - Instruct the second model to be ADVERSARIAL: assume the diff has at least one bug and find it
4. Compare the two reports. For each finding:
   - Both flag it → real, must address
   - Only Claude → likely real, address or justify dismissal
   - Only second model → may be real, investigate
   - Both clean → ok to merge

## Output format

```
## Cross-vendor review for PR #N

| Finding | Claude | <2nd model> | Verdict |
|---|---|---|---|
| Token compared with == not constant-time | 🔴 | 🔴 | MUST FIX |
| ctx not propagated through goroutine | 🟡 | — | SHOULD FIX |
| — | — | 🟡 stale jwt cache on revoke | INVESTIGATE |

## Disagreements
- Claude said X; <model> said Y. Resolution: ...

## Verdict
- ☐ Merge (both clean)
- ☐ Address findings then re-review
- ☐ Escalate to CEO (irreconcilable models)
```

## Cost guard

Cross-vendor calls cost real money. Cap:
- One pass per PR per session
- Skip if the noteworthy-flag is uncertain (default: no second model)
- Log per-tick spend in the cron telemetry channel

## Why this exists

gstack's `/codex` showed that single-model review misses ~15-30% of real findings catchable by a different vendor. Auth bugs are precisely the class where blind spots are catastrophic. This skill formalizes the pattern.
