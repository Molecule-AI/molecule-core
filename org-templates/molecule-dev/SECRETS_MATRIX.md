# Secrets Matrix — Per-Role Least Privilege

The platform supports per-workspace `.env` files (loaded by `org_import.go` and stored encrypted in `workspace_secrets`). Each role gets only the secrets it needs.

**Resolution order:** Org-root `.env` (shared defaults) → per-workspace `<role>/.env` (overrides). Operator-managed; never committed.

---

## Matrix

| Role | Secrets it gets | Scope of action enabled |
|---|---|---|
| **All workspaces** (org-root `.env`) | `CLAUDE_CODE_OAUTH_TOKEN` (or model-specific equivalent: `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`) | Run the LLM. Required for any agent to think. |
| **PM** | `TELEGRAM_BOT_TOKEN`, `TELEGRAM_CHAT_ID` (CEO comms only) | Send Telegram messages to CEO. Max 2-3/day per SHARED_RULES rule 11. |
| **Dev Lead, Core Lead, App Lead, CP Lead, Infra Lead, SDK Lead** | `GH_TOKEN` (write) | `gh pr merge`, `gh issue close`, `gh pr review --approve` on the team's repo. SHARED_RULES rule 9: Leads merge in their domain. |
| **Triage Operator** | `GH_TOKEN` (write, org-wide) | Cross-org triage: close stale, label, escalate. May merge mechanical PRs only. |
| **Engineers** (Backend, Frontend, Full-stack, DevOps, Platform, SRE, etc.) | `GH_TOKEN` with **PR-author scope only** — can `gh pr create`, `gh issue create`, `gh pr comment`. **Cannot merge.** | Raise PRs and respond to review comments. Per SHARED_RULES rule 9: engineers don't merge. |
| **QA Engineer** | `GH_TOKEN` (PR-comment scope) | Run tests + post `[qa-agent] APPROVED` / `CHANGES REQUESTED` comments. Required gate per rule 10. |
| **Security Auditor, Offensive Security Engineer** | `GH_TOKEN` (PR-comment scope) | Post `[security-auditor-agent] APPROVED` / `CHANGES REQUESTED`. Required gate per rule 10. |
| **UIUX Designer** | `GH_TOKEN` (PR-comment scope) | Post `[uiux-agent] APPROVED` / `CHANGES REQUESTED`. Required gate per rule 10. |
| **Marketing Lead** | `LINKEDIN_ACCESS_TOKEN`, `LINKEDIN_ORG_ID`, `X_API_KEY`, `X_API_SECRET`, `X_BEARER_TOKEN`, `BUFFER_API_KEY`, `MAILCHIMP_API_KEY` | Publish content to social channels. Sole publisher. |
| **Content Marketer, Social Media Brand, SEO Analyst** | NO publishing keys — `GH_TOKEN` (PR-author scope only) | Draft content via PRs to landing/docs/marketing repos. Marketing Lead reviews + publishes. |
| **DevRel Engineer** | `GH_TOKEN` (PR-author + comment scope), `DISCORD_BOT_TOKEN` (read-only on community channel) | Code demos via PRs. Read Discord for community questions. Marketing Lead handles outbound posts. |
| **Community Manager** | `SLACK_BOT_TOKEN`, `DISCORD_BOT_TOKEN` (read + post on community channels only) | Respond to community in Slack/Discord. No GitHub write. |
| **Research Lead, Market Analyst, Competitive Intelligence, Tech Researcher** | `GH_TOKEN` (PR-author + issue-create scope), `BRAVE_SEARCH_API_KEY` or `PERPLEXITY_API_KEY` | File research issues + PRs. No merge, no marketing publish. |
| **DevOps Engineer, SRE Engineer, Infra-Runtime-BE** | `GH_TOKEN` (write), `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` (scoped IAM role), `CLOUDFLARE_API_TOKEN` (DNS-only scope), `FLY_API_TOKEN`, `VERCEL_TOKEN` | Deploy + ops. Production access — heaviest scrutiny on changes. |
| **CP-BE, CP-QA, CP-Security** (control-plane) | `GH_TOKEN` (write on molecule-controlplane only), `AWS_ACCESS_KEY_ID/SECRET` (CP IAM role) | Control-plane code. CP Lead merges. |
| **Documentation Specialist, Technical Writer** | `GH_TOKEN` (PR-author scope on docs/landingpage repos) | Doc PRs only. No code-repo write. |
| **Release Manager** | `GH_TOKEN` (write on all repos), `NPM_TOKEN`, `PYPI_TOKEN` | Tag releases + publish packages after Lead-approved PRs land. |

---

## Why this matters

- **Prompt-injection blast radius**: an attacker who exfiltrates a workspace's secrets via prompt injection only gets that role's keys. Engineer compromise ≠ org-wide write. Marketing Compromise ≠ Telegram CEO message.
- **Audit trail**: when something goes wrong, the secret used identifies the role that did it.
- **Operator clarity**: copy `<role>/.env.example` to `<role>/.env`, paste the right keys, don't put production secrets in roles that don't need them.

---

## Operator setup

For each role's `.env.example`, copy to `.env` and fill in real values:

```bash
cd org-templates/molecule-dev
for role in dev-lead marketing-lead infra-lead pm; do
  cp $role/.env.example $role/.env  # then edit $role/.env
done
```

`.env` files are gitignored. The platform encrypts them on import to `workspace_secrets`.

---

## Future hardening (filed in `internal/security/credential-token-backlog.md`)

- Per-agent GitHub Apps (not shared org-wide token) — eliminates blast radius via #7 in backlog
- Egress filtering on workspace networks — limits what an exfiltrated secret can be sent to
- Volume encryption at rest — protects `.env` in workspace volumes from backup leak
- Token issuance audit logging — answers "who fetched the org token at time X?"
