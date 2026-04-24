# Phase 34 DevRel Demo Brief: Platform Instructions
**Date:** 2026-04-24
**Owner:** DevRel Engineer
**Priority:** High — publish alongside Apr 29 social post
**Source:** PR #1686, blog post `docs/marketing/blog/2026-04-23-platform-instructions-governance.md`
**Status:** Brief ready — awaiting DevRel execution

---

## Demo Goal

Show an enterprise developer how to enforce governance rules across an entire agent fleet from a single API call — before any agent starts, with no code changes to individual agents.

The "aha moment" is demonstrating that a `global` instruction set once takes effect on every workspace at startup, without touching any individual workspace config.

---

## Target Audience

Enterprise IT / Security / Governance leads, Platform Engineering, compliance-focused buyers.

---

## Demo Script (3–5 minutes)

### Step 1: Frame the problem (30 seconds)
> "If you've ever had to copy a compliance rule into 20 different system prompts — and then update all 20 when the rule changed — Platform Instructions is what you've been missing. One rule. Every workspace. Enforced before the first token."

### Step 2: Set a global instruction (~1 minute)

```bash
# Set an org-wide governance rule
curl -X POST https://api.moleculesai.app/org/instructions \
  -H "Authorization: Bearer $MOLECULE_ORG_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "scope": "global",
    "content": "No agent action may proceed without explicit user confirmation when the request involves: deleting data, issuing credentials, or modifying access controls.",
    "label": "critical-action-gate"
  }'
```

**Response:**
```json
{
  "id": "inst-global-001",
  "scope": "global",
  "label": "critical-action-gate",
  "created_at": "2026-04-29T10:00:00Z"
}
```

> "That instruction is now active for every workspace in the org. I haven't touched a single agent config."

### Step 3: Set a workspace-scoped instruction (~1 minute)

```bash
# Workspace-specific rule for a high-sensitivity workspace
curl -X POST https://api.moleculesai.app/workspaces/ws-finance-agent/instructions \
  -H "Authorization: Bearer $MOLECULE_ORG_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "scope": "workspace",
    "content": "All financial figures must include their data source and a confidence level (high/medium/low). Flag any response involving amounts over $10,000 with [REVIEW REQUIRED].",
    "label": "finance-data-policy"
  }'
```

> "And I can layer workspace-specific rules on top. Global rules apply everywhere; workspace rules add context for specific agents."

### Step 4: Show what the agent sees at startup (~1 minute)

```bash
# This is what the platform resolves at workspace startup
curl https://api.moleculesai.app/workspaces/ws-finance-agent/instructions/resolve \
  -H "Authorization: Bearer $MOLECULE_ORG_TOKEN"
```

**Response:**
```json
{
  "preamble": "No agent action may proceed without explicit user confirmation when the request involves: deleting data, issuing credentials, or modifying access controls.\n\nAll financial figures must include their data source and a confidence level (high/medium/low). Flag any response involving amounts over $10,000 with [REVIEW REQUIRED].",
  "sources": ["inst-global-001", "inst-ws-finance-001"]
}
```

> "This preamble is prepended to the agent's system prompt before the first token is generated. The agent doesn't know it came from a platform layer — it's just part of its context. No middleware. No plugin."

### Step 5: Security properties (30 seconds)
> "Three things worth knowing for enterprise buyers: First, the resolve endpoint is auth-gated per workspace — there's no cross-workspace enumeration. Second, Instructions are capped at 8KB — you can't accidentally fill an agent's entire context window with governance rules. Third, every Instruction records who created it — the audit trail is built in."

---

## Key Demo Points (must land)

1. **Global scope** — set once, enforces on every workspace in the org
2. **Pre-execution** — takes effect before the first token, not after the agent acts
3. **No agent code changes** — platform-layer enforcement
4. **Audit trail** — who created each rule, when

---

## Talking Points (avoid)

- Do NOT position as a replacement for RBAC or IAM — this is complementary
- Do NOT say "GA" — use "available now / beta"
- Do NOT show the `router.go:376` implementation detail — too internal for a demo
- Do NOT promise Canvas UI is live — it's in development

---

## Comparison angle (use if asked)

> "Most platforms let you put governance in the prompt. Platform Instructions puts governance in the platform. The difference: a developer can edit a prompt. They can't edit the platform."

---

## Assets Needed

- [ ] Screencast / GIF: the curl-based flow (create global → create workspace → resolve)
- [ ] Optional: side-by-side showing the preamble prepended to a real system prompt

---

## Publish Checklist

- [ ] Code samples tested against current API build
- [ ] Blog post link: `docs/marketing/blog/2026-04-23-platform-instructions-governance.md`
- [ ] Social copy aligned: `docs/marketing/social/2026-04-29-platform-instructions/social-copy.md`
