# Reno Stars — Operator Setup Notes

This template references operator-specific identity, accounts, and IDs as
**env vars** rather than hardcoding them. Before importing this org, set
the following as `global_secrets` so the platform injects them into every
workspace container.

## Required env vars

| Variable | Example | Where it's referenced |
|----------|---------|------------------------|
| `OPERATOR_EMAIL` | `you@example.com` | user_profile, social-media-poster, seo-builder, pinterest_account |
| `OPERATOR_PHONE` | `555-123-4567` | user_profile (display only) |
| `OPERATOR_TELEGRAM_ID` | `1234567890` | user_profile (Telegram bot DM target) |
| `GADS_MCC_ID` | `123-456-7890` | project_google_ads (Google Ads MCC) |
| `GADS_CUSTOMER_ID` | `987-654-3210` | project_google_ads (Google Ads child account) |
| `GCP_PROJECT_ID` | `my-website-123456` | seo-weekly-report (GCP project for Search Console reporter) |
| `GSC_SERVICE_ACCOUNT` | `gsc-reporter@my-website-123456.iam.gserviceaccount.com` | seo-weekly-report (auto-derived from GCP_PROJECT_ID + service-account name) |

## How to set them

Pick one:

**A. Via the canvas Settings → Secrets tab** (per-workspace) or
   Settings → Global Secrets (platform-wide, recommended for operator info).

**B. Via the API:**
```bash
curl -X PUT http://localhost:8080/settings/secrets \
  -H 'Content-Type: application/json' \
  -d '{"key":"OPERATOR_EMAIL","value":"you@example.com"}'
# repeat for each var
```

**C. Via the MCP server tool** `mcp__molecule__set_global_secret` from any
   Claude Code / Cursor / Codex session connected to the platform.

## Verify

After importing the org, exec into any reno-stars container and check
the env is populated:

```bash
docker exec ws-<id> env | grep -E '^(OPERATOR|GADS|GSC|GCP)_'
```

If a value is missing, the agent will see the literal `${VAR_NAME}` string
in its system prompt — that's the failure mode to watch for.

## Why this exists

The literal values used to be hardcoded in 14 markdown files across this
template. That was fine when the repo was private but leaked operator PII
to the public hackathon repo (phone, email, GCP service account, Google
Ads IDs). The 2026-04-13 scrub moved everything to env vars; the template
shape is unchanged.
