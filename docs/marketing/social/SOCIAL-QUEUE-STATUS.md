# Marketing Social Queue — Status Tracker
**Owner:** PMM | **Last updated:** 2026-04-23
**Purpose:** Single source of truth for all social copy status across campaigns.

---

## Phase 34 GA Launch (April 30, 2026)

### 2026-04-26 — Phase 34 GA: Tool Trace + Platform Instructions ✅ DRAFT (PMM)
- **File:** `2026-04-26-phase34-ga-launch/social-copy.md`
- **Status:** PMM pre-write complete (commit 5f020914). Awaiting Marketing Lead approval → approved queue.
- **Content:** 6-post X thread + LinkedIn post
- **Owner:** PMM → Social Media Brand
- **Blocking:** Marketing Lead approval

---

## Phase 30 Social Campaign (days 1–5, April 21–25)

### 2026-04-21 — Chrome DevTools MCP ✅ MERGED (awaiting ML publish)
- **File:** `2026-04-21-chrome-devtools-mcp/social-copy.md`
- **Status:** Copy complete, organized. Awaiting Marketing Lead publish approval.
- **Content:** 3 X variants (governance / production use cases / developer) + LinkedIn
- **Owner:** PMM → Social Media Brand
- **Blocking:** Marketing Lead publish approval + X credentials
- **Visual assets needed:** 3-item checklist graphic (Lighthouse/Regression/Auth) + fleet diagram (reusable)

### 2026-04-21 — Cloudflare Artifacts ✅ DRAFT (PMM, catch-up)
- **File:** `2026-04-21-cloudflare-artifacts/social-copy.md`
- **Status:** PMM pre-write complete (commit 56ea6375). Awaiting Marketing Lead approval → approved queue.
- **Content:** 5-post X thread + LinkedIn + Reddit + HN copy
- **Blog:** Live 2026-04-21 (`moleculesai.app/blog/cloudflare-artifacts-molecule-ai`)
- **Angle:** "Git for agents" — pain story first, technology as answer
- **Owner:** PMM → Social Media Brand
- **Blocking:** Marketing Lead approval + X credentials
- **Visual assets needed:** Artifacts repo attach flow screenshot + git commit terminal output
- **Note:** Blog shipped April 21; social copy delayed, now catching up. Cloudflare Artifacts is in beta — do not claim GA.

### 2026-04-22 — EC2 Instance Connect SSH ✅ APPROVED (PMM positioning)
- **File:** `2026-04-22-ec2-instance-connect-ssh/social-copy.md`
- **Status:** PMM positioning approved (GH #1637). Social Media Brand unblocked for Versions A + D.
- **Content:** 5-post X thread + LinkedIn
- **Owner:** PMM → Social Media Brand → DevRel (screenshot)
- **Blocking:** DevRel terminal screenshot (PR #1545) + Content Marketer blog (#1546) + X credentials
- **Note:** PR #1686 DevRel demo package (PR #1878) may supersede the original screenshot requirement

### 2026-04-24 — EC2 Console Output ✅ APPROVED (Marketing Lead)
- **File:** `2026-04-24-ec2-console-output/social-copy.md`
- **Status:** Approved by Marketing Lead 2026-04-22. Ready for Social Media Brand execution.
- **Content:** 4-post X thread + LinkedIn
- **Owner:** Social Media Brand
- **Blocking:** X credentials + visual asset (`ec2-console-output-canvas.png`, 1200×800 dark mode)
- **Campaign position:** Day 4 of Phase 30 social campaign

### 2026-04-25 — Org-Scoped API Keys ✅ APPROVED (Marketing Lead)
- **File:** `2026-04-25-org-scoped-api-keys/social-copy.md`
- **Status:** Approved by Marketing Lead 2026-04-21. Ready for Social Media Brand execution.
- **Content:** 5-post X thread + LinkedIn
- **Owner:** Social Media Brand
- **Blocking:** X credentials + visual assets (Canvas UI screenshot, before/after credential model, audit log terminal output)
- **Campaign position:** Day 5 of Phase 30 social campaign

---

## Staged on `origin/staging` (unreviewed by PMM)

### MCP Server List — Day 1 ✅ COPY READY
- **File:** `docs/marketing/campaigns/mcp-server-list/social-copy.md` (on staging, commit `0d3ad96`)
- **Status:** Copy complete. Awaiting visual assets + X credentials.
- **Content:** 5-post X thread + LinkedIn
- **Canonical URL:** `docs.molecule.ai/blog/mcp-server-list`
- **Owner:** Social Media Brand
- **Blocking:** Visual assets + X credentials

### Discord Adapter — Day 2 ✅ COPY READY
- **File:** `discord-adapter-social-copy.md` (on staging)
- **Status:** Copy complete. Awaiting Marketing Lead Day 2 approval + X credentials + visual assets.
- **Content:** 4 X variants + LinkedIn + Reddit + HN copy
- **Canonical URL:** `docs.molecule.ai/blog/discord-adapter` (live, PR #1301 merged)
- **Owner:** Social Media Brand → Marketing Lead (Day 2 approval)

---

## Held / Pending Decision

### Fly.io Deploy Anywhere — Stale (T+5)
- **File:** `fly-deploy-anywhere-social-copy.md`
- **Status:** 5 days stale (blog shipped April 17). PMM selected Option A (retrospective framing).
- **Campaign position:** Phase 30 social campaign catch-up
- **Decision needed:** Marketing Lead confirmation on Option A framing
- **Blocking:** Marketing Lead decision + X credentials

### Phase 30 (original) — MERGED
- **File:** `phase30-social-copy.md`
- **Status:** MERGED to origin/main. Awaiting Marketing Lead publish approval.
- **Owner:** PMM → Social Media Brand
- **Blocking:** Marketing Lead publish approval + X credentials

---

## Cross-Cutting Blockers (all human-gated)

| Blocker | Affects |
|---|---|
| X credentials (`X_ACCESS_TOKEN` + `X_ACCESS_TOKEN_SECRET`) | ALL posts — Social Media Brand cannot publish anything |
| Marketing Lead publish approvals | Chrome DevTools MCP, Phase 30, Fly.io, Discord Day 2 |
| DevRel terminal screenshot (PR #1545) | EC2 Instance Connect SSH |
| Content Marketer blog post (#1546) | EC2 Instance Connect SSH |
| Visual assets (Canvas screenshots, diagrams) | EC2 Console Output, Org-Scoped API Keys, Chrome DevTools MCP |

---

## Assets: Visual Requirements by Post

| Post | Asset | Source | Status |
|---|---|---|---|
| Chrome DevTools MCP | 3-item checklist graphic | Custom (Lighthouse/Regression/Auth) | Needed |
| Chrome DevTools MCP | Fleet diagram | Reuse `marketing/assets/phase30-fleet-diagram.png` | Ready |
| EC2 Instance Connect SSH | Canvas terminal screenshot | DevRel (PR #1545) | Blocked |
| EC2 Console Output | Canvas screenshot (dark, 1200×800) | Custom | Needed |
| Org-Scoped API Keys | Canvas Org API Keys UI screenshot | Custom | Needed |
| Org-Scoped API Keys | Before/after credential model graphic | Custom | Needed |
| Org-Scoped API Keys | Audit log terminal output | Custom | Needed |
| MCP Server List | Campaign visual | Custom | Needed |
| Discord Adapter | Multi-channel diagram | Custom | Needed |

---

*PMM compiled 2026-04-23. Consolidated from multiple inline social-queue files into single status tracker.*
*Marketing Lead: approve queue items to unblock Social Media Brand for execution once X credentials are restored.*