# A2A v1 Blog — Staging-Stranded
**Finding:** 2026-04-23 — Marketing Lead pulse
**Status:** STRANDED — write token dead, cannot merge to main

## What is stranded
`docs/blog/2026-04-22-a2a-v1-agent-platform/index.md` on `origin/staging`
- A2A v1 deep-dive: ~1,370 words, technical
- LangGraph governance-gap ADR comparison (enterprise positioning)
- External agent registration Python code example
- Org-scoped API key delegation attribution
- Tied to social copy: `docs/marketing/campaigns/a2a-enterprise-deep-dive/social-copy.md` (also staging-only)

## Why this matters
- Phase 30's most technically substantive content is not on main
- LangGraph A2A GA targeting Q2-Q3 — window to establish Molecule AI as canonical reference is NOW
- SEO: "A2A protocol" and "agent-to-agent protocol" keywords unclaimed
- 72h urgency noted in brief — window closes when LangGraph ships

## What is needed
GitHub write token restored → create staging→main PR → merge

## Local frontmatter additions (not yet on staging)
- `tags: [a2a, agent-protocol, multi-agent, governance, enterprise, platform]`
- `og_image: /assets/blog/2026-04-20-chrome-devtools-mcp-og.png`

## Full staging-stranded blog audit (2026-04-23)

| Blog | Staging path | Main status | Why it matters |
|---|---|---|---|
| A2A v1 Enterprise Deep-Dive | `2026-04-22-a2a-v1-agent-platform/` | ❌ NOT on main | LangGraph comparison, Phase 30 flagship |
| Org-scoped API keys (updated) | `2026-04-22-ai-agents-org-scoped-keys/` | ⚠️ Older version on main | PMM revision, Day 5 social copy references |
| Cloudflare Tunnel Migration (Phase 33) | `2026-04-22-cloudflare-tunnel-migration/` | ❌ NOT on main | Phase 33 launch, social copy written |
| Remote Workspaces (updated) | `2026-04-22-remote-workspaces/` | ⚠️ Older version on main | Phase 30 Day 4 social copy references |

**All 4 require GitHub write token to merge to main.**
