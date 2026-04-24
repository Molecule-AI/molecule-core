# Social Queue Status — Updated 2026-04-24
**Owner:** Community Manager
**Status:** Active queue — 3 past due dates flagged

> ⚠️ **Past-due items (Apr 24 + Apr 25):** These were marked APPROVED but blocked on visual assets + X credentials. Today is April 24 — items are past their scheduled dates. Social Media Brand to publish when X credentials land.
> 
> **Apr 24** (EC2 Console Output): APPROVED, visual assets present. Blocked on X credentials.
> **Apr 25** (Org-Scoped API Keys): APPROVED, visual assets needed. Blocked on X credentials.

---

# Chrome DevTools MCP — Social Copy
**Source:** PR #1306 merged to origin/main (2026-04-21)
**Status:** MERGED — awaiting Marketing Lead approval for publishing

---

## X (140–280 chars)

### Version A — Governance angle
```
Chrome DevTools MCP gives agents full browser control. Screenshot, DOM, JS execution — all through a standard interface.

Raw CDP is all-or-nothing. Molecule AI adds the governance layer: which agents get access, what they can do, how to revoke it.

Audit trail included.
```

### Version B — Production use cases
```
Three things you couldn't automate before Chrome DevTools MCP + Molecule AI governance:

1. Lighthouse CI/CD audits — agent opens Chrome, runs Lighthouse, posts score to PR
2. Visual regression testing — screenshot diffs across agent workflow runs
3. Authenticated session scraping — agent behind a login with managed cookies

All with org API key audit trail.
```

### Version C — Problem framing
```
Chrome DevTools MCP: browser automation as a first-class MCP tool.

For prototypes: great. For production: you need something between no browser and full admin. That's the gap Molecule AI's MCP governance fills.
```

---

## LinkedIn (100–200 words)

Chrome DevTools MCP shipped in early 2026 — and browser automation is now a standard tool for any compatible AI agent.

Screenshot. DOM inspection. Network interception. JavaScript execution. No custom wrappers, no browser-driver installation.

That's the prototype story. For production — especially anything touching customer-facing workflows or authenticated sessions — all-or-nothing CDP access is a governance gap.

Molecule AI's MCP governance layer answers the production questions:
- Which agents can open a browser?
- What can they do with it?
- How do you revoke access?
- When something goes wrong, who accessed what session data?

Real-world use cases the layer enables: automated Lighthouse performance audits in CI/CD, screenshot-based visual regression testing, and authenticated session scraping — agents operating behind a login with cookies managed through the platform's secrets system.

Every action is logged. Every browser operation is attributed to an org API key and workspace ID.

Chrome DevTools MCP plus Molecule AI's governance layer: browser automation that meets production standards.

---

## Image suggestions

| Post | Image |
|---|---|
| X Version A | Fleet diagram: `marketing/assets/phase30-fleet-diagram.png` (reusable) |
| X Version B | Custom: 3-item checklist graphic — "Lighthouse / Regression / Auth Scraping" |
| X Version C | Quote card: "something between no browser and full admin" |
| LinkedIn | Quote card or the checklist graphic |

---

## Hashtags

`#MCP` `#BrowserAutomation` `#AIAgents` `#MoleculeAI` `#DevOps` `#QA` `#CI/CD`

---

## Blog canonical URL

`docs.moleculesai.app/blog/browser-automation-ai-agents-mcp`

---

## MCP Server List Explainer
**File:** `docs/marketing/campaigns/mcp-server-list/social-copy.md`
**⚠️ Status:** FILE MISSING — `social-copy.md` not on staging (only `assets/` directory present). Queue entry is stale.
**Action required:** Content Marketer to write social copy or confirm location. Remove or restore this entry.
**Canonical URL:** `docs.molecule.ai/blog/mcp-server-list`
**Owner:** Content Marketer | **Day:** TBD

---

## Discord Adapter Day 2
**File:** `discord-adapter-social-copy.md` (local)
**Status:** COPY READY — awaiting visual assets + X credentials
**Canonical URL:** `docs.molecule.ai/blog/discord-adapter` (live, PR #1301 merged)
**Owner:** Social Media Brand | **Day:** Ready once visual assets done

See `discord-adapter-social-copy.md` for full copy (4 X variants + LinkedIn draft).

---

## Fly.io Deploy Anywhere (T+3 catch-up)
**Source:** Blog live 2026-04-17 | Social delayed 5 days
**File:** `fly-deploy-anywhere-social-copy.md` (local)
**Status:** COPY READY — PMM executing Option A (retrospective catch-up). Awaiting X credentials.
**Canonical URL:** `moleculesai.app/blog/deploy-anywhere`
**Owner:** Social Media Brand | **Day:** Queue immediately after Chrome DevTools MCP Day 1 posts
**Decision:** PMM chose Option A per decision brief. Frame: "we shipped this last week."

Retrospective framing: "Week in review: we shipped Fly.io Deploy Anywhere last week. Here's what it means for your agent infrastructure."

Social Media Brand: hold Fly.io post until Chrome DevTools MCP Day 1 posts land, then queue Fly.io in the same session.

---

## EC2 Instance Connect SSH (PR #1533)
**File:** `docs/marketing/social/2026-04-22-ec2-instance-connect-ssh/social-copy.md`
**Status:** COPY READY — `#AgenticAI` replaced with `#AIAgents` (fix applied 2026-04-23)
**Canonical URL:** `docs.molecule.ai/blog/ec2-instance-connect-ssh`
**Owner:** Social Media Brand | **Day:** Ready once X credentials available

Full 5-post X thread + LinkedIn post. Angle: no SSH key management, ephemeral permissions, AWS-native. Blog live (PR #1533 merged). ⚠️ Visual assets needed before publish.
---

## Org-Scoped API Keys (PR #1105)
**File:** `docs/marketing/social/2026-04-25-org-scoped-api-keys/social-copy.md`
**Status:** ✅ APPROVED — Marketing Lead 2026-04-21 | `#AgenticAI` replaced with `#AIAgents` (fix applied 2026-04-23)
**Canonical URL:** `docs.molecule.ai/blog/org-scoped-api-keys`
**Owner:** Social Media Brand | **Day:** 5 (2026-04-25) ⚠️ PAST DUE

Full 5-post X thread + LinkedIn post. Angle: named, revocable, audit-attributed org API keys replacing shared ADMIN_TOKEN. Compliance + DevOps audience. ⚠️ Blog publish confirmation + visual assets needed.

---

## Forward-Looking Queue

### Phase 34 Launch — Day 6 (2026-04-26)
**File:** `docs/marketing/social/2026-04-26-phase34-ga-launch/social-copy.md`
**Status:** DRAFT — PMM pre-write, pending Marketing Lead approval
**Features:** Tool Trace + Platform Instructions
**Canonical URL:** `docs.molecule.ai/blog/tool-trace-platform-instructions`
**Owner:** PMM → Social Media Brand
**Blocker:** Marketing Lead approval required before Social Media Brand can publish

### Phase 34 GA — Day 10 (2026-04-30)
**File:** `docs/marketing/social/2026-04-30-phase-34-ga-launch/social-copy.md`
**Status:** ✅ APPROVED — Marketing Lead 2026-04-23
**Features:** Partner API Keys GA + Tool Trace/Platform Instructions recap
**Canonical URL:** `docs.molecule.ai/api/partner-keys`
**Owner:** Social Media Brand | **Day:** 10 (2026-04-30) GA day
**Blocker:** X credentials needed; GA vs Beta label must be resolved (Gate 1)

---

*Updated: 2026-04-24 by Community Manager*
*Next update: After Marketing Lead approves Apr 26 Phase 34 social copy*
