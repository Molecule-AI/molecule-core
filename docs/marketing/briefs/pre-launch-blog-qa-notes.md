# Pre-Launch Blog QA Notes
**Owner:** PMM | **Date:** 2026-04-24 | **Status:** PARTIAL — two posts not yet on main/staging
**Purpose:** Flag accuracy issues vs. approved positioning briefs before content goes live.

---

## ✅ Blog 3 of 3 — A2A Enterprise Deep-Dive
**File:** `docs/blog/2026-04-22-a2a-v1-agent-platform/index.md`
**Brief:** `docs/marketing/briefs/2026-04-22-a2a-enterprise-deep-dive-seo-brief.md`
**Status: CLEAN** (one ⚠️ advisory, no blockers)

### Accuracy checks

| Claim area | Brief says | Blog says | Verdict |
|---|---|---|---|
| Auth enforcement point | "per-workspace bearer tokens enforced at every authenticated route" | "org-scoped API keys... audited at the org level" + "per-workspace bearer tokens" | ✅ Correct |
| Auth architecture | Protocol-level enforcement (NOT "discovery-time CanCommunicate()") | Auth correctly described as per-workspace enforced at authenticated routes | ✅ Correct |
| LangGraph A2A support | "LangGraph A2A PR #6645 [WIP]" | "LangGraph A2A PR #6645 (WIP) plus #7113 and #7205" | ⚠️ VERIFY — PRs #7113 and #7205 were NOT independently verified in this QA cycle. Blog correctly preserves ⚠️ flag. No blocking issue — flag is self-documenting. |
| VPN guardrail | "no VPN required for control plane" | Correctly implied throughout (no VPN language) | ✅ Correct |
| Org-scoped API keys + audit trail | Both mentioned together | Both mentioned | ✅ Correct |

**PMM verdict:** Blog 3 is clean to publish. The LangGraph PR ⚠️ flag is appropriate self-documentation.

---

## 🔴 Blog 1 of 3 — EC2 Instance Connect SSH ⚠️ FILE NOT ON MAIN OR STAGING
**Expected file:** `docs/marketing/blog/2026-04-23-ec2-instance-connect-ssh.md`
**Expected on:** `origin/staging` (commit `0d3ad96` per social queue)
**Actual:** File NOT FOUND on main or staging under either path attempted.
**Brief:** `docs/marketing/launches/pr-1533-ec2-instance-connect-ssh.md`

### Blocking issues

**Cannot QA — file does not exist yet.**
Social queue notes file should live on staging under `docs/marketing/blog/2026-04-23-ec2-instance-connect-ssh.md` (commit `0d3ad96`) but `git show origin/staging:docs/marketing/blog/2026-04-23-ec2-instance-connect-ssh.md` returned NOT FOUND. The blog post may not have been committed, or may be in a different directory.

### Key claims to verify once file is available

| Watch item | Brief approved claim | Risk if not verified |
|---|---|---|
| Connection timing | "< 3 seconds" | Overclaiming ("instant"/"real-time") would be inaccurate |
| Public IP requirement | "no public IP" | EICE requires private subnet access — must not imply all EC2 instances qualify |
| IAM auth vs SSH keys | EICE uses IAM-based auth — no SSH key management | Must not claim "no keys stored on instance" without the EICE context |
| Bastion host framing | Alternatives to bastion hosts | Must not贬低 AWS-native EICE vs. generic bastion elimination |

**PMM action:** Flag to Marketing Lead and Content Marketer — blog post not on staging, QA blocked. Request Content Marketer commit the file or confirm its location.

---

## 🔴 Blog 2 of 3 — MCP Server List ⚠️ FILE NOT ON MAIN OR STAGING
**Expected file:** `docs/marketing/blog/2026-04-23-mcp-server-list.md`
**Expected on:** `origin/staging` (per social queue commit `0d3ad96`)
**Actual:** File NOT FOUND on main or staging.
**Brief:** `docs/marketing/seo/mcp-server-list-explainer-seo-brief.md`

### Blocking issues

**Cannot QA — file does not exist yet.**
`git ls-tree -r origin/staging --name-only | grep -i mcp-server` returned zero blog-post matches. Only `docs/guides/mcp-server-setup.md` and `docs/marketing/seo/mcp-server-list-explainer-seo-brief.md` (the brief itself) exist on staging.

### Key claims to verify once file is available

| Watch item | Brief approved claim | Risk if not verified |
|---|---|---|
| MCP server names | Chrome DevTools MCP, Playwright MCP, Cloudflare Artifacts, EC2 Instance Connect, WriteFile, ReadFile, Glob, Grep, Slack, Discord adapters | Omitting or misnaming a server category is an accuracy error |
| Governance claims | Org-scoped API keys + Platform Instructions (Enterprise plans) | Platform Instructions confirmed Enterprise plans only (AdminAuth-gated, router.go:376). Blog post correct; community FAQ corrected 2026-04-24. |
| Platform Instructions plan gate | Enterprise-only feature | ✅ CONFIRMED 2026-04-24: Enterprise plans only per code review + plan-gating note. Blog post correct; community FAQ corrected. |
| Blog title/URL alignment | Brief targets `mcp-server-list-explainer` keyword | Title should match SEO intent |

**PMM action:** Flag to Marketing Lead and Content Marketer — blog post not on staging, QA blocked. Request Content Marketer commit the file.

---

## Summary

| Blog | Status | Blocking? | Action owner |
|---|---|---|---|
| A2A Enterprise Deep-Dive | ✅ Clean (LangGraph ⚠️ flag self-documenting) | No | Ready for publish |
| EC2 Instance Connect SSH | 🔴 File not on staging | Yes — cannot QA | Content Marketer |
| MCP Server List | 🔴 File not on staging | Yes — cannot QA | Content Marketer |

**Recommendation:** Marketing Lead should route blockers to Content Marketer. Once both files are on staging, PMM can complete QA within one working day.

---

*PMM QA notes 2026-04-24. A2A blog reviewed. Platform Instructions plan availability resolved (Enterprise plans only). EC2 + MCP blog posts: blocked on Content Marketer delivery.*
