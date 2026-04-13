---
name: review-loop
description: "Orchestrate a multi-round implementation + review cycle. Use when coordinating a feature that requires implementation (FE/BE), design review (UIUX), security review, and QA verification. Ensures QA findings get routed back for fixes until clean."
---

# Review Loop

Orchestrate implementation through multiple rounds until QA reports zero issues.
This prevents the one-shot delegation problem where QA finds bugs but nobody
fixes them.

## When to Use

Use this when you're a coordinator (Dev Lead, PM) assigning a feature that
involves multiple specialists.

## The Loop

### Round 1: Design + Implementation (parallel where possible)

1. **Identify all stakeholders** — before delegating, ask: who needs to be involved?
   - UI work → UIUX Designer reviews interaction design FIRST
   - Credentials / auth / secrets → Security Auditor reviews
   - API changes → Backend Engineer + Frontend Engineer coordinate
   - Everything → QA Engineer is the final gate

2. **Delegate design review first** (if UI work):
   ```
   delegate_task_async → UIUX Designer: "Review the interaction design for [feature]"
   ```

3. **Delegate implementation** (after design, or parallel if non-UI):
   ```
   delegate_task_async → Frontend Engineer: "Implement [feature] following UIUX spec"
   delegate_task_async → Backend Engineer: "Add [endpoint]" (if needed)
   delegate_task_async → Security Auditor: "Review [feature] for [specific concerns]"
   ```

4. **Delegate QA** (can start in parallel — QA reads existing code while FE works):
   ```
   delegate_task_async → QA Engineer: "Review [feature], run full test suite, write missing tests, grep for convention violations"
   ```

5. **Collect all results** via `check_task_status` on each delegation.

### Round 2: Fix QA Findings (if any issues found)

If QA reported issues:

1. **Send QA's findings back to the implementer:**
   ```
   delegate_task → Frontend Engineer: "QA found these issues in your implementation:
   [paste QA's specific findings with file:line references]
   Fix all of them and report back."
   ```

2. **Re-run QA on the fixes:**
   ```
   delegate_task → QA Engineer: "FE applied fixes for your findings. Re-verify:
   [paste the specific issues that were fixed]
   Run the test suite again. Report if any issues remain."
   ```

3. **If QA still finds issues → repeat Round 2.**

### Round 3: Final Sign-off

When QA reports zero issues:
- Compile the full report: what was implemented, what was fixed, test results
- Report to PM / CEO with substance, not just "done"

## Key Rules

- **Never skip QA.** Even if FE says "I tested it." QA verifies independently.
- **Never skip Security for credential-related features.** A secrets panel without security review is a liability.
- **QA findings are not optional.** If QA found it, it gets fixed. Period.
- **Use parallel delegation.** `delegate_task_async` to all specialists at once, then collect with `check_task_status`. Don't serialize what can be concurrent.
- **Ask side questions.** If FE needs to know the API shape, FE should `delegate_task` directly to BE — don't relay through the lead.
