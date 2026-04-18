# GDPR Art. 17 hard-delete cascade

Operational reference for the "delete my org" flow in `the private control-plane repo`.
Skim this before replying to an erasure request, answering a DPA (Data
Processing Addendum) audit, or debugging a failed purge.

## What Art. 17 actually requires

The EU General Data Protection Regulation, Article 17 ("right to erasure" /
"right to be forgotten") says: when a user asks us to delete their personal
data, we must do so within 30 days and destroy **every copy** we control —
including copies held by our sub-processors (Stripe, Fly, Neon, Upstash,
Vercel, WorkOS). Soft-delete is not compliant. A database row with
`deleted_at IS NOT NULL` still counts as "data we process" under the GDPR
definition.

## What the cascade actually does

`DELETE /cp/orgs/:slug` triggers `handlers.executeOrgPurge`, which walks
four steps in order:

| # | Step | Action | Idempotent? |
|---|------|--------|-------------|
| 1 | `stripe` | List every subscription on `cus_*`, DELETE each, then DELETE the customer record. Stripe retains deleted customers for ~30 days per their internal policy — we do not control that window | Yes (404 → success) |
| 2 | `redis` | Upstash REST `scan` + `del` against pattern `<org_slug>:*` | Yes (empty scan = no-op) |
| 3 | `infra` | `Provisioner.DeprovisionInstance` → Fly Machine destroy + Neon branch delete + Vercel subdomain removal | Yes at each sub-step |
| 4 | `db_rows` | One transaction: `DELETE FROM org_instances` → `DELETE FROM org_members` → `DELETE FROM organizations` | Atomic |

Each successful step writes `org_purges.last_step`. The orchestrator reads
`last_step` on entry and **skips every step at or before it** — so a retried
DELETE resumes from the first unfinished step instead of repeating Stripe
cancellations or Redis scans.

The `org_purges` audit row outlives the deleted org on purpose — `org_id` is
NOT a foreign key. Auditors or support staff can still answer "when was
acme.moleculesai.app deleted and did it succeed?" three months later.

## When a purge fails mid-cascade

The API returns `500` with a JSON body:

```json
{
  "error":    "purge cascade failed; retry the request to resume",
  "purge_id": "<uuid>"
}
```

What to do:

1. **Inspect the audit row** — `SELECT status, last_step, last_error, attempts
   FROM org_purges WHERE id = '<purge_id>'`. That tells you which step blew
   up and why.
2. **Fix the underlying cause** if it's ours (Stripe API key rotation,
   Upstash network blip, Fly API 500).
3. **Re-issue the DELETE** — the handler picks up from `last_step + 1`. No
   manual DB surgery is needed in the happy path.
4. **If the step that failed is `db_rows`** — the transaction rolled back, so
   the org is still fully intact. Retry is safe.
5. **If the step that failed is `infra`** — check Fly + Neon + Vercel
   dashboards before retrying. A half-destroyed Fly Machine won't block the
   retry (DeprovisionInstance is idempotent), but it's worth confirming the
   resource actually went away.

## 30-day deadline

GDPR gives us one calendar month to complete erasure from the request date.
The cascade runs synchronously and typically finishes in <15 seconds, so
latency is not the concern — **unattended failure** is. If an `org_purges`
row sits in `status='failed'` for more than 24h, that's the operator's cue
to intervene. A future Phase H task will add a cron that pings Slack when
any purge row is older than 48h without hitting `completed`.

## What this cascade does NOT do

- **It does not delete WorkOS user records.** WorkOS Users are org-scoped
  (a user can belong to multiple orgs), and we don't own enough lifecycle
  signal to decide when to purge the underlying user account. When the last
  org containing a user is erased, the WorkOS user will be orphaned. Phase
  H.2 adds a sweep to reconcile.
- **It does not delete LLM provider history.** Agent conversations that
  used OpenAI / Anthropic / OpenRouter may still appear in the provider's
  own retention window. Our DPAs with those vendors cap that at 30 days; we
  do not expose a hook to accelerate it.
- **It does not delete Langfuse traces** for self-hosted Langfuse. In
  production we forward traces to Langfuse Cloud which has its own
  retention policy — check `LANGFUSE_HOST` in the env before claiming
  compliance.

## Testing the cascade

See the test plan in [PR #29](https://github.com/Molecule-AI/the private control-plane repo/pull/29)
for the staging checklist. The unit tests cover the orchestrator logic
(happy path, resume-from-step, Stripe failure, no-customer); end-to-end
proof requires a real Stripe test-mode customer + provisioned Fly Machine
because the failure modes that matter are transport errors, not logic.

## Related

- `docs/runbooks/saas-secrets.md` — if a cascade fails with "invalid API
  key" the relevant secret probably rotated
- `docs/runbooks/admin-auth.md` — `DELETE /cp/orgs/:slug` is behind
  session-cookie auth in controlplane, not the workspace bearer-token
  middleware documented there
- `the private control-plane repo/internal/handlers/purge.go` — the orchestrator
- `the private control-plane repo/migrations/006_org_purges.*.sql` — audit schema
