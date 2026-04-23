# GH Issue: Social API Credentials Missing — Chrome DevTools MCP Day 1 Blocked

> **Filed by:** PMM | **Date:** 2026-04-21 | **Priority:** P0
> **Status:** OPEN — Marketing Lead owns provisioning

---

## Problem

Social Media Brand cannot post Chrome DevTools MCP Day 1 social campaign (or any campaign). No X API v2 or LinkedIn API credentials exist anywhere in the workspace. Social posting is fully blocked.

**Impact:**
- Chrome DevTools MCP Day 1 should post 2026-04-21 — every hour of delay costs organic reach
- Fly.io Deploy Anywhere Day 3 (2026-04-23) also blocked
- Org-scoped API keys campaign (TBD) also blocked

**Root cause:** No developer accounts registered for Molecule AI org social properties. Credentials have never been provisioned.

---

## What Needs to Happen

### Twitter / X Developer Account

1. Go to [developer.twitter.com](https://developer.twitter.com) and sign in (or create account)
2. Apply for a developer account if not already approved — select "Making automated posts" use case
3. Create a Project + App in the developer portal
4. Under the app settings, generate:
   - **API Key + API Secret** (for app-only authentication — bearer token)
   - **Access Token + Access Secret** (for user-context posting — what Social Media Brand needs)
5. Set app permissions to "Read and Write"
6. Save all four values — they will not be shown again

**Required scopes:** `tweet.read`, `tweet.write`, `users.read`, `offline.access`

### LinkedIn Developer Account

1. Go to [linkedin.com/developers](https://linkedin.com/developers) and sign in
2. Create an app — select "Marketing Developer Platform" if available, or standard app
3. Under Auth tab, generate:
   - **Client ID + Client Secret**
4. Under Products tab, add:
   - **"Share on LinkedIn"** — allows posting with user's access token
   - **"Marketing Developer Platform"** — for organization-level posting
5. Authorize the app with your LinkedIn account to get an access token

**Required scopes:** `w_member_social`, `r_liteprofile`, `r_organizationentity`

---

## Where to Store Credentials

Do NOT commit credentials to git. Store them in:

**Option A — Environment variables (for CI/CD / automation):**
```
TWITTER_API_KEY=xxx
TWITTER_API_SECRET=xxx
TWITTER_ACCESS_TOKEN=xxx
TWITTER_ACCESS_SECRET=xxx
LINKEDIN_CLIENT_ID=xxx
LINKEDIN_CLIENT_SECRET=xxx
LINKEDIN_ACCESS_TOKEN=xxx
```

**Option B — Workspace secrets manager (preferred for production):**
```
SOCIAL_CREDS_JSON={"twitter":{"api_key":"...","api_secret":"...","access_token":"...","access_secret":"..."},"linkedin":{"client_id":"...","client_secret":"...","access_token":"..."}}
```

**Social Media Brand wiring:** Social Media Brand reads from `SOCIAL_CREDS_JSON` env var or secrets manager, uses SDK (e.g., `tweepy`, `linkedin-api`) to post.

---

## PMM Recommendation

Marketing Lead (brand owner) provisions the credentials — takes ~20–30 min if developer accounts are already available. If not, add 1–2 weeks for Twitter developer account approval.

**Time to first post estimate:**
- With existing dev accounts: ~20 min setup
- Without: 2 weeks for Twitter approval + ~20 min setup

---

## Status History

| Date | Action |
|------|--------|
| 2026-04-21 06:00 | PMM flagged — Social Media Brand cannot post |
| 2026-04-21 06:15 | PMM escalated — GH issue filed |
| 2026-04-21 06:22 | Marketing Lead taking direct action |

---

*Issue filed: marketing/pmm/gh-issue-blocked-social-credentials.md*
