# Ephemeral Test Orgs (CI/CD Deep-Dive) — Social Copy
**Publish day:** 2026-04-28
**Status:** APPROVED — Marketing Lead 2026-04-23
**Campaign:** Phase 34 launch run-up

---

## X / Twitter Thread (4 tweets)

---

**Tweet 1 — Problem**

Your AI agent integration tests share one staging org.

That means shared state, contaminated data, and flaky pipelines — especially when multiple PRs merge the same day.

Here's the fix. 🧵

---

**Tweet 2 — Solution**

Partner API Keys give you a clean org lifecycle per pipeline run:

`POST /cp/admin/partner-keys` — provision a fresh org
→ run your integration tests against real infrastructure
`DELETE /cp/admin/partner-keys/:id` — tear down, stop billing

Clean billing. Clean state. No leftover artifacts from the last run. No cross-PR contamination.

Each PR gets exactly the isolation it deserves.

---

**Tweet 3 — GitHub Actions Pattern**

What this looks like in a GitHub Actions workflow:

```yaml
- name: Provision test org
  run: |
    ORG_ID=$(curl -s -X POST https://api.molecule.ai/cp/admin/partner-keys \
      -H "Authorization: Bearer $MOL_PK_TOKEN" \
      -d '{"scope":"orgs:create"}' | jq -r '.org_id')
    echo "ORG_ID=$ORG_ID" >> $GITHUB_ENV

- name: Run integration tests
  run: npm run test:integration
  env:
    MOLECULE_ORG_ID: ${{ env.ORG_ID }}

- name: Teardown test org
  if: always()
  run: |
    curl -s -X DELETE \
      https://api.molecule.ai/cp/admin/partner-keys/$ORG_ID \
      -H "Authorization: Bearer $MOL_PK_TOKEN"
```

Create → test → teardown. The `if: always()` ensures cleanup even on failure.

Full CI/CD example in the partner onboarding guide: https://doc.moleculesai.app/blog/tool-trace-platform-instructions

---

**Tweet 4 — CTA**

Partner API Keys go GA April 30.

If your team tests against a shared staging org today, this is worth looking at before your next sprint planning.

Docs: https://doc.moleculesai.app/blog/tool-trace-platform-instructions
Partner Discord: join the conversation before launch day.

---

## LinkedIn Post (~200 words)

**Why your agent integration tests are flaky — and how ephemeral test orgs fix it.**

If your team builds on Molecule AI and runs integration tests, there's a good chance your pipeline shares one staging org across all test runs. That means:

- PR A creates test data. PR B runs while PR A's data is still there.
- A failed teardown from yesterday leaves artifacts that break today's tests.
- Debugging a flaky test means untangling whether the failure is your code or someone else's leftover state.

This is a solved problem in databases (transactions, per-test rollbacks) and containers (ephemeral environments per PR). Agent orchestration infrastructure deserves the same pattern.

**With Partner API Keys (GA April 30), the pattern is:**

1. `POST /cp/admin/partner-keys` — provision a clean org at the start of the pipeline run
2. Run integration tests against that org in full isolation
3. `DELETE /cp/admin/partner-keys/:id` — tear down the org and stop billing when the run finishes

Each PR gets a fresh org. No shared state. No contamination. The `DELETE` is idempotent — wire it into your cleanup step with `if: always()` and it runs even on failure.

The full GitHub Actions example is in the partner onboarding guide, live April 30.

Partner Discord is open now if you want to talk through the pattern before launch day.
