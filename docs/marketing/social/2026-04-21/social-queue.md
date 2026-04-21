# Social Queue — 2026-04-21
**Approved by:** Marketing Lead
**Status:** READY TO POST

---

## Campaign 1: Chrome DevTools MCP — Day 1 (POST TODAY)

**Source:** `docs/marketing/campaigns/chrome-devtools-mcp-seo/social-copy.md`
**Blog:** `docs/blog/2026-04-20-chrome-devtools-mcp-seo/index.md` (live on staging)
**Images:** fleet diagram (`marketing/assets/phase30-fleet-diagram.png`)
**Hashtags:** #MCP #AIAgents #AgenticAI #MoleculeAI
**UTM:** `?utm_source=twitter&utm_medium=social&utm_campaign=chrome-devtools-mcp-seo`

### X Thread (5 posts, post 20–30 min apart)

**Post 1 — Hook**
> Your AI agent just made a purchase on your behalf.
> What did it buy? From where? With which account?
> Most agents operate in a black box. Browser DevTools MCP makes the browser a first-class tool — with org-level audit attribution on every action.
> → [link]

**Post 2 — Problem framing**
> Browser automation for AI agents usually means: give the agent your credentials, hope it doesn't go somewhere unexpected, and check the logs after.
> That's not a governance model. That's a trust fall.
> Molecule AI's MCP governance layer for Chrome DevTools MCP gives you: → Which agent accessed which session → What it did (navigate, fill, screenshot, submit) → Audit trail with org API key attribution
> One org API key prefix per integration. Instant revocation.
> → [link]

**Post 3 — Concrete use cases**
> Real things teams use Chrome DevTools MCP for in production:
> • Automated Lighthouse audits on every PR — agent runs the audit, reports the score, flags regressions
> • Visual regression detection — agent screenshots key pages, diffs against baseline, opens tickets on drift
> • Auth scraping — agent reads the authenticated state from an existing browser session
> The governance layer means your security team can see all three in the audit trail.
> → [link]

**Post 4 — Positioning**
> The MCP protocol lets you connect any compatible tool to any compatible agent.
> What's been missing: visibility into what the agent actually *did* with that access.
> Molecule AI's MCP governance layer adds: • Per-action audit logging with org API key attribution • Token-scoped Chrome sessions — no credential sharing across agents • Instant revocation without redeployment
> → [link]

**Post 5 — CTA**
> Chrome DevTools MCP launched April 20 as part of Molecule AI Phase 30.
> If you're running AI agents that interact with web UIs — there's a governance story you need to have ready before your security team asks.
> → [link]

### LinkedIn (post 2h after X thread)
**Title:** Why your AI agent's browser access needs a governance layer

> Your AI agent can use a browser. That's useful. But "useful" isn't a security posture.
> [Full post in source file — see social-copy.md]
> UTM: `?utm_source=linkedin&utm_medium=social&utm_campaign=chrome-devtools-mcp-seo`

---

## Campaign 2: Fly.io Deploy Anywhere — Day 3+ (✅ APPROVED)

**Source:** `docs/marketing/campaigns/fly-deploy-anywhere/social-copy.md` ← canonical, approved 2026-04-21
**Blog:** `docs/blog/2026-04-17-deploy-anywhere/index.md`
**Post day:** 2026-04-23 (Day 3) or Day 5 (2026-04-25) — both blocked on credentials
**Images:** backend-comparison-card.svg, architecture diagram
**Hashtags:** #AIagents #Flyio #SaaS #DeveloperTools #DevOps #MultiTenant
**Status:** ✅ APPROVED by Marketing Lead 2026-04-21 — ready for Social Media Brand once credentials land

### X Thread (5 posts)

**Post 1 — Hook**
> Your infrastructure choice just got decoupled from your agent platform.
> Until this week: Molecule AI workspaces ran on Docker. One backend. One option.
> Now there are three. And switching takes one environment variable.

**Post 2 — What's new**
> Molecule AI now ships three production-ready workspace backends: 🐳 Docker — self-hosted, no external deps 🚀 Fly.io Machines — pay-per-use, scale to zero ☁️ Control Plane API — multi-tenant SaaS, credential isolation built in. Same agent code. Same API surface. Just flip a config flag.

**Post 3 — Security angle**
> If you're building a SaaS product on Molecule AI, you have a Fly API token problem.
> Every tenant platform instance that carries a FLY_API_TOKEN is one misconfiguration away from a credential exposure.
> The fix: CONTAINER_BACKEND=controlplane. Fly credentials live in Molecule AI's control plane — never on the tenant.

**Post 4 — Indie dev angle**
> On Fly.io already? Three env vars and your Molecule AI workspaces are Fly Machines: CONTAINER_BACKEND=flyio / FLY_API_TOKEN=<your-token> / FLY_WORKSPACE_APP=<your-app> Pay for what you use. Scale to zero. No idle Docker host.

**Post 5 — Comparison**
> Quick guide: which backend fits? Self-hosted / local dev → Docker | On Fly, small team → flyio | SaaS, multi-tenant → controlplane
> Picking your backend → deploying your agents. Link in bio.

### LinkedIn (single post)
See full copy in `docs/marketing/campaigns/fly-deploy-anywhere/social-copy.md`
UTM: `?utm_source=linkedin&utm_medium=social&utm_campaign=fly-deploy-anywhere`

---

## Notes
- Chrome DevTools MCP (Day 1): ✅ copy approved — blocked on X API v2 + LinkedIn credentials (manual post required today 2026-04-21)
- Fly.io Deploy Anywhere (Day 3): ✅ copy approved — blocked on credentials
- Org-scoped API Keys (Day 5): ✅ copy approved — blocked on credentials
- Discord adapter Day 2 Reddit + HN: ✅ copy approved in GH #1383 — two versions ready (Community Manager + Social Media Brand), manual browser post required today 2026-04-21
- Fleet diagram: `marketing/assets/phase30-fleet-diagram.png`
- Backend comparison card: `docs/marketing/campaigns/fly-deploy-anywhere/assets/backend-comparison-card.svg`