# Pre-Launch Blog QA Notes — Phase 34

**Reviewed by:** Marketing Lead  
**Date:** 2026-04-23  
**Scope:** Three new blog posts written this session for Phase 34 launch week  

---

## Post 1: A2A Enterprise Deep-Dive

**File:** `docs/marketing/blog/2026-04-23-a2a-enterprise-deep-dive.md`  
**Slug:** `a2a-enterprise-deep-dive`  
**Status:** ✅ CLEAN — approved for publish

**Checks passed:**
- Auth guardrail present: "every A2A call is token-authenticated. There is no unauthenticated path." ✓
- VPN guardrail present: "no VPN tunnel required for the control plane" ✓
- Code sample correct: Bearer token + `X-Workspace-ID` + `org_key_prefix` ✓
- P0 keywords present: `enterprise AI agent platform`, `multi-cloud AI agent orchestration`, `agent delegation audit trail` ✓
- LangGraph comparison: framed as ADR-layer validation, no PR numbers cited ✓
- Platform Instructions: correctly stated as all plans ✓
- No enterprise-only gating errors ✓
- Internal links: Tool Trace observability post cross-linked ✓

**No issues found.**

---

## Post 2: MCP Server List

**File:** `docs/marketing/blog/2026-04-23-mcp-server-list.md`  
**Slug:** `mcp-server-list`  
**Status:** ✅ APPROVED with minor link fixes noted below

**Checks passed:**
- Catalogue accurate: Chrome DevTools MCP, Playwright, Cloudflare Artifacts, EC2 Instance Connect, Sandbox (Node/Python/Bash), WriteFile/ReadFile/Glob/Grep, Slack, Discord, Custom/Community ✓
- Governance section correctly describes Platform Instructions as all-plans (no enterprise gate) ✓
- Tool Trace cross-link to observability post present ✓
- Custom MCP registration code sample syntax correct ✓
- Partner API Keys (`mol_pk_*`) governance description accurate ✓

**Minor issues (non-blocking, fix before final publish):**

1. **Relative link format on lines 28 and 39** — internal blog cross-links use bare relative paths without leading slash:
   - Line 28: `docs/blog/2026-04-20-chrome-devtools-mcp/` → should be `/docs/blog/2026-04-20-chrome-devtools-mcp/`
   - Line 39: `docs/blog/2026-04-21-cloudflare-artifacts/` → should be `/docs/blog/2026-04-21-cloudflare-artifacts/`

2. **Double "docs" in partner program URL (lines 119, 122)**:
   - Current: `https://docs.molecule.ai/docs/guides/partner-onboarding`
   - Likely correct: `https://docs.molecule.ai/guides/partner-onboarding`
   - Verify against live docs.molecule.ai URL structure before publish

These are link-formatting issues only. No factual errors or positioning violations.

---

## Post 3: EC2 Instance Connect SSH Terminal

**File:** `docs/marketing/blog/2026-04-23-ec2-instance-connect-ssh.md`  
**Slug:** `ec2-instance-connect-ssh`  
**Status:** ✅ APPROVED with one unverified claim flagged for PM confirmation

**Checks passed:**
- No sub-second timing claims — "roughly three seconds" is appropriately qualified ✓
- "No public IP" — accurate, EICE routes through AWS internal network ✓
- "No internet egress required" — accurate for EICE ✓
- IAM as control plane — correct, EICE is an AWS API call ✓
- CloudTrail audit trail — accurate ✓
- 60-second TTL on ephemeral key — matches documented EICE behavior ✓
- "Zero per-user configuration" — correct, Terminal tab appears automatically ✓
- Scope correct: "CP-provisioned EC2 workspaces" (not all EC2 workspaces) ✓
- PR #1533 cited ✓

**All items resolved — no open questions.**

- **Phase attribution (line 17):** "Phase 30" confirmed ✓ — `docs/marketing/launches/pr-1533-ec2-instance-connect-ssh.md` (positioning brief) explicitly places this feature in the "Phase 30 remote workspaces narrative" (brief line 89, 120).
- **Internal link on line 52** (`../../infra/workspace-terminal.md`): `docs/infra/workspace-terminal.md` confirmed present in repo ✓ — shipped in PR #1533 per brief line 118.

---

## Summary

| Post | Status | Blocking Issues | Notes |
|------|--------|----------------|-------|
| A2A Enterprise Deep-Dive | ✅ Publish-ready | None | All guardrails clean |
| MCP Server List | ✅ Publish-ready | None | 2 minor link fixes recommended before final publish |
| EC2 SSH Terminal | ✅ Publish-ready | None | Phase 30 confirmed ✓; workspace-terminal.md link confirmed ✓ |

All three posts are clear of factual errors on Platform Instructions plan gating (confirmed all-plans by code review of `instructions.go`). No enterprise-only positioning errors. No unqualified sub-second timing claims. Auth and VPN guardrails honored where relevant.

---

*Marketing Lead QA — 2026-04-23. PMM quality gate delegation (6a6bf951) queued but workers saturated; review handled directly.*
