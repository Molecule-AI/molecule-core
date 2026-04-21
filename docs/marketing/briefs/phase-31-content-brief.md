# Phase 31 Launch — Content Brief: Molecule AI Cloud GA

> **Owner:** Marketing Lead + PMM + DevRel
> **Status:** Draft — for Marketing Lead review
> **Target launch:** TBD (pending Stripe Atlas + hardening completion)
> **Branch:** `content/phase-32-saas-launch-blog` (merged to main as of ~2026-04-15)

---

## Executive Summary

Phase 31 marks the transition of Molecule AI from a self-hosted developer platform to a **multi-tenant cloud SaaS** available at `moleculesai.app`. Existing users can continue self-hosting; new users can sign up, provision an org, and deploy agents in minutes without managing infrastructure.

**Core positioning shift:**
- Before: "Molecule AI is an OSS agent orchestration platform you run yourself"
- After: "Molecule AI is the OSS platform you run yourself — and the cloud service you don't have to"

This matters because it removes the biggest friction point for individual developers and small teams: the ops burden of self-hosting.

---

## What Changed in Phase 31

| Component | Before (self-hosted) | After (Molecule AI Cloud) |
|---|---|---|
| Provisioning | Manual Docker/K8s setup | One-click org creation via `moleculesai.app` |
| Auth | Manual JWT / session config | WorkOS AuthKit — signup/login/SSO |
| Database | Self-managed Postgres | Neon serverless Postgres (branch-per-org) |
| Cache | Self-managed Redis | Upstash Redis |
| Billing | N/A | Stripe (subscription + usage metering) |
| Container isolation | Docker socket (self-hosted) | Firecracker microVMs via Fly Machines |
| Control plane | Self-hosted | `https://molecule-cp.fly.dev` (managed) |

---

## Target Audiences

### Primary: Individual developers / indie hackers
- **Job to be done:** Run a personal AI agent or small fleet without managing a server
- **Pain:** Don't want to run Docker, configure DNS, manage secrets, or maintain infra
- **Entry point:** `moleculesai.app` → sign up → deploy first agent in <5 min
- **Content need:** Quickstart guide (cloud-first), 1-page explainer, explainer video (60s)

### Secondary: Small teams (2–10 engineers)
- **Job to be done:** Collaborate on agent workflows, share context across team members
- **Pain:** Self-hosting multi-user is ops-heavy; no billing separation
- **Entry point:** Org-level workspaces, team memory scopes, per-workspace tokens
- **Content need:** Team onboarding guide, org management docs, comparison with self-hosted

### Tertiary: Enterprise / CTO ( Phase 32G+ hardening)
- **Job to be done:** Evaluate for production use, understand security and compliance posture
- **Pain:** Trust, data residency, SLA, compliance docs
- **Entry point:** Security + compliance page, security whitepaper (post-launch)
- **Content need:** Security overview, GDPR/SOC 2 posture, enterprise inquiry flow

---

## Content Inventory (Phase 31)

### Already shipped on `content/phase-32-saas-launch-blog` (merged)

| Asset | File | Status | Notes |
|---|---|---|---|
| Blog post: SaaS launch | `docs/blog/2026-04-XX-phase-32-saas-launch/index.md` | ✅ SHIPPED | Merger as of ~2026-04-15 |
| Blog post: Skills vs Bundled Tools | `docs/blog/2026-04-XX-skills-vs-bundled-tools/index.md` | ✅ SHIPPED | |
| Org-scoped API keys blog | `docs/blog/2026-04-XX-org-api-keys-launch/index.md` | ✅ SHIPPED | |

### DevRel deliverable: Molecule AI Cloud quickstart guide

**Why this is needed:** The existing `docs/guides/remote-workspaces.md` and quickstart are self-hosted-first. A cloud-first quickstart is the primary conversion asset for the primary audience.

**Target:** ~800 words, 8 steps, no ops required.

**Outline:**
1. Sign up at `moleculesai.app` (30s, email or GitHub OAuth)
2. Create your org (auto-named from email domain)
3. Launch Canvas at `app.moleculesai.app` (auto-redirect)
4. Register your first agent (5-min `python3 run.py` from any machine)
5. Verify agent appears in Canvas with REMOTE badge
6. Share workspace with teammates (invite via email)
7. Explore TEAM-scoped memory (shared knowledge layer)
8. Deploy to production: mint an org API key

**SEO target:** `get started with AI agents`, `AI agent platform quickstart`, `self-hosted vs cloud AI agents`

**TTS:** Generate 45s VO narration for explainer video (follow-on deliverable)

---

### DevRel deliverable: Pricing explainer page

**Why this is needed:** Stripe billing scaffold is deployed but not live; pricing page needs developer-friendly framing before launch.

**Angle:** "Start free. Scale as your agent fleet grows." Tiered by agent-hours or workspace-count, not by seat.

**Outline:**
- Free tier: 2 agents, 5 workspaces, 100 MB memory snapshots
- Pro tier: $29/mo — unlimited agents, priority A2A routing, audit logs
- Team tier: $99/mo — org-scoped API keys, SAML SSO, per-workspace RBAC
- Enterprise: custom — dedicated infra, SLA, compliance docs

**Notes:**
- Stripe Atlas is the blocker for live payments (est. ~2 week lead time)
- Pricing page should go up as a placeholder before billing is live
- DevRel should write the copy; PMM owns the tier boundaries and pricing

---

### DevRel deliverable: Phase 31 explainer video (60s)

**Why this needed:** Primary conversion asset alongside the quickstart guide.

**Concept:** "The fastest way to run AI agents."

**Storyboard approach (terminal + browser, no voiceover):**
1. Sign up screen — email field, "Create free account" button (0:00–0:05)
2. Canvas dashboard — empty org, "Add your first agent" prompt (0:05–0:10)
3. Terminal — `curl https://moleculesai.app/install | sh` (simplified) or `pip install molecule-sdk` + `python3 run.py` (0:10–0:20)
4. Canvas — agent registers, REMOTE badge appears in real time (0:20–0:30)
5. Split screen — team member invites, org workspace list (0:30–0:45)
6. End card — `moleculesai.app` + "Start free" CTA (0:45–0:60)

**Format:** 1080p H.264, 30fps. Dark zinc theme. SF Mono terminal. No VO.

---

## Messaging Framework

### Tagline candidates (for PMM to decide)
- "AI agents that run where you need them" (reframe of Phase 30)
- "From self-hosted to SaaS in one click"
- "The AI agent platform you don't have to run"

### Position against competition

| Competitor | Molecule AI Cloud advantage |
|---|---|
| Cursor / Windsurf | Not a code editor — orchestration layer; works with any agent runtime |
| LangChain Agents | Open source, no vendor lock-in, self-host or cloud |
| OpenAI Assistants API | Multi-vendor — orchestrates across models, not just GPT |
| AWS Bedrock / Agentic AI | OSS core, no AWS dependency, simpler mental model |

### Key proof points (needs PMM verification)
- Deploy an agent in 5 minutes from any machine
- No credit card required for free tier
- Org-scoped API keys for CI/CD pipelines
- TEAM and GLOBAL memory scopes for multi-agent coordination
- Open-core: self-host or use cloud, same API surface

---

## SEO Strategy

### Phase 31 keyword targets

| Keyword | Intent | Target |
|---|---|---|
| `AI agent platform` | Informational / Comparison | Quickstart guide H1 |
| `self-hosted AI agents` | Informational / Comparison | Comparison section of quickstart |
| `AI agent collaboration` | Informational | Team onboarding guide |
| `Molecule AI pricing` | Commercial | Pricing page |
| `AI agent team workspace` | Informational | Org management docs |

### SEO approach
- Quickstart guide is the primary ranking asset — optimize for "how to get started with AI agents"
- Pricing page captures commercial-intent traffic
- Do NOT try to rank for "AI agents" generally — too broad, too competitive
- Focus on compound phrases: "AI agent team workspace", "AI agent platform self-hosted"

---

## Pre-launch Checklist

- [ ] **Pricing page copy** — DevRel drafts; PMM sets tier boundaries
- [ ] **Quickstart guide** — DevRel authors; Doc Specialist reviews (cloud-first)
- [ ] **Explainer video** — DevRel produces (60s, no VO, terminal + browser)
- [ ] **Blog post: Molecule AI Cloud GA** — Content Marketer authors; PMM approves positioning; DevRel provides technical accuracy
- [ ] **Blog post: Self-hosted vs Cloud comparison** — Content Marketer authors; DevRel provides feature table
- [ ] **Social copy** — Social Media Brand drafts; CM schedules (X + LinkedIn)
- [ ] **Email drip sequence** — Marketing Lead coordinates (Day 1 announcement, Day 3 feature highlight, Day 7 case study)
- [ ] **Community posts** — Discord + Reddit (Community Manager)
- [ ] **HN launch** — Marketing Lead or CEO posts (template in `community/hacker-news-launch.md` — adapt for cloud launch angle)
- [ ] **Press release** — Content Marketer authors; CEO reviews

---

## Blocker: Stripe Atlas

Stripe Atlas is the gating item for live billing. Estimated ~2-week lead time. Content can ship before billing is live — pricing page is a placeholder, free tier can go live first, Stripe integration is additive.

**Recommendation:** Ship the quickstart guide, explainer video, and blog post before Stripe is live. Pricing page ships at the same time as billing activation.

---

*DevRel authored. PMM to add: pricing tier boundaries, official tagline, competitive battlecard for cloud offering.*