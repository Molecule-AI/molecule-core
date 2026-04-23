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
**File:** `docs/marketing/campaigns/mcp-server-list/social-copy.md` (staging, commit `0d3ad96`)
**Status:** COPY READY — awaiting visual assets + X credentials
**Canonical URL:** `docs.molecule.ai/blog/mcp-server-list`
**Owner:** Social Media Brand | **Day:** Ready once visual assets done

5-post X thread + LinkedIn post. Full copy on staging.

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
