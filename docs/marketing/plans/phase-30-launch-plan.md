# Phase 30 Launch Plan — Chrome DevTools MCP SEO Campaign

**Owner:** Marketing Lead
**Status:** Active — Day 1 execution pending social credentials
**Last updated:** 2026-04-21

---

## Campaign Status

| Deliverable | Owner | Status |
|-------------|-------|--------|
| SEO brief | Marketing Lead | ✅ Complete |
| Blog post | Marketing Lead | ✅ LIVE on main (689d82d) |
| Keywords (P0/P1) | Marketing Lead | ✅ Confirmed — all P0/P1 integrated |
| Social copy | Marketing Lead | ✅ APPROVED (PR #1504) |
| Backlinks outreach | Marketing Lead | ✅ APPROVED (PR #1504) |
| Social queue Day 1–5 | Marketing Lead | ✅ APPROVED — executing when credentials land |
| SEO pre-launch | SEO Analyst | ✅ COMPLETE — all P0/P1 keywords integrated, frontmatter + structured data fixed |
| SEO indexing | SEO Analyst | ⏳ Lighthouse audit opens 2026-04-22 (~15h) |
| Social distribution | Social Media Brand | ⏳ BLOCKED — X/LinkedIn credentials not provisioned |

## Social Queue (Approved)

| Day | Date | Campaign | Status |
|-----|------|----------|--------|
| Day 1 | Apr 21 | Chrome DevTools MCP | ✅ Ready — blocked on credentials |
| Day 2 | Apr 22 | Discord Adapter | ✅ Ready |
| Day 3 | Apr 23 | Org API Keys | ✅ Ready |
| Day 4 | Apr 24 | EC2 Console Output | ⚠️ Draft pending ML approval — share copy to approve |
| Day 5 | Apr 25 | Cloudflare Artifacts | ✅ APPROVED — "sub-100ms" softened to "fast edge-based clone times" |
| Day 5+ | Apr 25+ | Org-Scoped API Keys | ✅ Approved | |

---

## Confirmed Content

- **Brief:** `docs/marketing/briefs/2026-04-20-chrome-devtools-mcp-seo-brief.md`
- **Blog post:** `docs/marketing/blog/2026-04-20-how-to-add-browser-automation-to-ai-agents-with-mcp.md`
- **P0 keywords:** "MCP browser automation", "Chrome DevTools MCP"
- **P1 keywords:** "AI agent browser control", "MCP protocol tutorial"

---

## Pending Actions

### Social Credentials (BLOCKER — CEO action required)
**Owner:** CEO / whoever has access to developer.twitter.com and linkedin.com/developers
**Action required:**
1. Create X API v2 app → generate Bearer Token
2. Create LinkedIn API app → generate Client ID + Secret
3. Provision both to Social Media Brand workspace (`a0ddb78e-72b3-4597-b945-daa3314478c6`)
**Status:** Blocking all 5 days of approved social content.

### SEO Indexing
**Owner:** SEO Analyst
**Status:** Lighthouse audit pending post-deploy. Schedule 48h post-GA audit.
**Action required:** Run Lighthouse + verify P0 keywords are indexed in Google Search Console.

### EC2 Console Output (Day 4)
**Owner:** Social Media Brand
**Status:** Draft pending Marketing Lead approval
**Action required:** Share draft copy for approval before Apr 24.

---

## Open Questions

1. **Social credentials:** Timeline for provisioning?
2. **EC2 Console Output:** Share draft copy for Marketing Lead approval.
3. **Phase 30 GA date:** Confirmed shipped Apr 20 ✅

---

## Next Checkpoint

Social Media Brand will execute Day 1–5 as soon as credentials land. No other blockers from marketing side.
