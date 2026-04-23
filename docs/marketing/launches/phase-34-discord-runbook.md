# Phase 34 — Discord Launch Runbook
**Campaign:** Phase 34 GA (April 30, 2026)
**Owner:** Community Manager
**Use:** Step-by-step execution guide for launch day
**Date:** 2026-04-23

---

## Launch day overview

| Time (UTC) | Action |
|------------|--------|
| -15 min (08:45) | Pre-launch checklist |
| -5 min (08:55) | Final link verification |
| 09:00 | Post in `#announcements` |
| 09:00 | Pin FAQ in `#faq` |
| 09:00–11:00 | Monitor `#general` + `#feedback` (2h active watch, 30-min SLA) |
| Ongoing | Route inbound to correct channel / team |
| Apr 30 ~16:00 (Day 2) | Reddit r/MachineLearning + HN Show HN |

---

## Pre-launch checklist (complete by 08:45 UTC)

### Blog posts verified live
- [ ] `docs.moleculesai.app/blog/ai-agent-observability-without-overhead` (Tool Trace)
- [ ] `docs.moleculesai.app/blog/platform-instructions-governance` (Platform Instructions)
- [ ] `docs.moleculesai.app/blog/partner-api-keys` (Partner API Keys — GA April 30, may show "coming soon" before launch)
- [ ] `docs.moleculesai.app/guides/external-workspace-quickstart` (SaaS Fed v2) ⚠️ PM VERIFICATION NEEDED — tutorial file not found in codebase, treat as unconfirmed
- [ ] `docs.moleculesai.app/blog/tool-trace-platform-instructions` (combined overview)

### API endpoints responding
- [ ] `PUT /cp/platform-instructions` — test in staging (org admin endpoint)
- [ ] Partner API Keys endpoint — confirm not responding before GA (expected)
- [ ] `GET /cp/platform-instructions` — confirm accessible in staging

### Docs updated
- [ ] `docs/architecture/partner-api-keys.md` — reflects `mol_pk_*` key format and scopes
- [ ] `docs/api-protocol/a2a-protocol.md` — mentions `tool_trace` in `Message.metadata`
- [ ] `docs/guides/external-workspace-quickstart.md` — ⚠️ PM VERIFICATION NEEDED — do not check as "done" until PM confirms what SaaS Fed v2 actually shipped

### Announcement file verified
- [ ] `docs/marketing/launches/phase-34-community-announcement.md` — on `staging` or `main`, CTA links accurate
- [ ] No design partner names in copy
- [ ] Partner API Keys framed as "GA April 30" — not "available now"

### X credentials status (issue #1865)
- [ ] If `X_API_KEY` + `X_API_SECRET` provided by mol-ops → note in Discord post thread
- [ ] If not provided → Reddit/HN posts go Day 2, no external social Apr 30

### Teams on standby
- [ ] DevRel confirmed monitoring `#devrel` or DM for how-to routing
- [ ] Platform team monitoring `#bug-reports`
- [ ] Marketing Lead aware of launch timing

### Escalation path clear
- [ ] DM list for emergencies: DevRel, Security, Marketing Lead
- [ ] Bug-report channel confirmed active

---

## Step 1 — Post announcement in `#announcements` (09:00 UTC)

**Source:** `docs/marketing/launches/phase-34-community-announcement.md`

Post as plain text, emoji-formatted. Discord message limit is 2000 chars — split across multiple messages if needed. Keep the block separators (`━━`) intact.

**Before posting — final checks:**
1. Run `curl -s -o /dev/null -w "%{http_code}" docs.moleculesai.app/blog/ai-agent-observability-without-overhead` and confirm 200
2. Confirm no "available now" framing for Partner API Keys
3. Confirm no design partner names

**Post announcement, then immediately reply in thread:**
```
📋 FAQ — answers to the top community questions:
[paste link or mirror the FAQ content here]

Full blog coverage:
docs.moleculesai.app/blog/tool-trace-platform-instructions
docs.moleculesai.app/blog/ai-agent-observability-without-overhead
docs.moleculesai.app/blog/platform-instructions-governance
docs.moleculesai.app/blog/partner-api-keys
docs.moleculesai.app/guides/external-workspace-quickstart
```

---

## Step 2 — Pin FAQ in `#faq` (09:00 UTC)

1. Open `#faq`
2. Paste full content from `docs/marketing/launches/phase-34-community-faq.md`
3. Right-click → Pin Message
4. Confirm pin is visible
5. If channel has stale pins, clear old ones and pin Phase 34 FAQ

**FAQ should be pinned before the announcement goes out** so people can find it immediately when they arrive.
**Day 2 update:** After the announcement settles, consider pinning the announcement itself in `#announcements` so it stays at the top of the channel.

---

## Step 3 — Monitor `#general` and `#feedback` (09:00–11:00 UTC)

**SLA: respond within 30 minutes of any Phase 34 reply.**

Set a 30-minute repeating reminder to check both channels.

**Response template — question you can answer:**
```
Hey [name] — good question. [1-2 sentence answer]. Full details in our docs: [link]. Let me know if that doesn't cover it!
```

**Response template — question you can't answer:**
```
Great question — I need to loop in the platform team and get back to you. Tagging @devrel for a closer look.
```

**Response template — feature request:**
```
Love this idea — tagging @pm so this gets into the backlog. You can also open a GitHub issue with the label "enhancement" to track it formally.
```

---

## Step 4 — Monitor `#bugs`, `#partner-program` (ongoing)

### `#bugs` channel
- Tool Trace bugs → tag `@devrel` in `#devrel` or DM directly
- Platform Instructions bugs → tag `@dev-lead` in `#devrel` or DM directly
- Partner API Keys issues (post-GA) → tag `@mol-ops` or DM

### `#partner-program` channel
- Partner API Keys early access requests → acknowledge, DM with next steps
- Integration questions → route to DevRel if technical, Marketing Lead if strategic

### `#general`
- How-to questions → answer directly or tag `@devrel`
- "Is this available now?" → check against GA date, redirect to docs
- Security concerns → do not respond publicly. DM Security team immediately.

---

## Step 5 — Escalation paths

| Issue type | Route to | How |
|-----------|----------|-----|
| Tool Trace unexpected behavior | DevRel | DM or tag in `#devrel` |
| Platform Instructions not applying | DevRel / Dev Lead | DM or tag in `#devrel` |
| Partner API Keys access / billing issues | mol-ops | DM or tag in `#partner-program` |
| SaaS Fed v2 isolation concern | Security / Dev Lead | DM Security, tag Dev Lead |
| Security vulnerability | Security team | **DM only — do not post in any channel** |
| Press / media inquiry | Marketing Lead | **Do not engage publicly — DM Marketing Lead immediately** |

**For any toxic or spam thread:**
- Do not engage
- Screenshot thread
- DM Marketing Lead with link + screenshot

---

## Response templates — common questions

**Q: "Is Partner API Keys available now?"**
A: "Partner API Keys ship on April 30, 2026. Until then the API isn't live. If you want early access for a concrete integration use case, DM me and I'll connect you with the team."

**Q: "How is Tool Trace different from Langfuse/Helicone?"**
A: "Tool Trace captures A2A-level agent behavior — tool calls, inputs, output previews. Langfuse/Helicone capture LLM API calls. They measure different layers. If you're running agents on Molecule, Tool Trace is zero-config and free. If you need cross-platform multi-model observability, Langfuse is still a great complement."

**Q: "Can I use Platform Instructions to enforce a policy across my org?"**
A: "Yes — set a global instruction via PUT /cp/platform-instructions (scope: global). It applies to every workspace in your org at startup. Rules prepend to each agent's system prompt — workspace users can't override them by editing config.yaml."

**Q: "What's the rate limit on Partner API Keys?"**
A: "Default is 60 requests/minute per key, configurable at key creation time. For high-volume CI pipelines, request a higher limit when you apply for a partner key."

**Q: "My agent isn't producing tool_trace in responses."**
A: "Tool Trace is on by default for all workspaces. Make sure activity logging is enabled on your workspace. If it is, and you're still not seeing traces, open a bug in #bug-reports and tag @devrel."

**Q: "Where is the migration guide for Phase 34?"**
A: "Phase 34 features are additive — no migration required for existing agents. Tool Trace starts appearing automatically, Platform Instructions are opt-in per org admin. Migration docs at docs.moleculesai.app/guides/phase-34-migration — live on April 30."

---

## Post-launch (24h): metrics and feedback

### Engagement metrics to capture
- `#announcements` — reply count, reaction count (first 4h)
- `#faq` — pin view count if available, question count
- `#general` — Phase 34 thread volume
- GitHub Discussions — new discussions opened, response time
- Reddit/HN (Day 2) — post score, comment count, avg time to first reply

### Feedback to route to PM
- Questions that surfaced unexpected complexity in the features
- Feature requests that multiple community members asked about
- Any confusion about what shipped vs what's coming April 30
- Partner API Keys early access requests — log use case + org name

### Day 2 — Reddit + HN

**Reddit r/MachineLearning** (~09:00 PT / 16:00 UTC):
- Source: `docs/marketing/community/phase34-reddit-post.md`
- Title: "Built agent execution tracing into the platform — no SDK, no sidecar, no sampling"
- Monitor for 2h, reply to top-level comments within 30 min
- Do not name design partners

**HackerNews Show HN** (~09:00 PT / 16:00 UTC):
- Source: `docs/marketing/community/phase34-hn-post.md`
- Title: "Show HN: Molecule AI's approach to platform-native agent observability + governance"
- First reply (pinned): code snippet from the tool_trace example
- Monitor for 3h, reply to every top-level comment

---

## Files reference

| File | Purpose |
|------|---------|
| `docs/marketing/launches/phase-34-community-announcement.md` | Announce in `#announcements` |
| `docs/marketing/launches/phase-34-community-faq.md` | Pin in `#faq` |
| `docs/marketing/community/phase34-reddit-post.md` | Reddit Day 2 |
| `docs/marketing/community/phase34-hn-post.md` | HN Day 2 |
| `docs/marketing/briefs/phase34-positioning.md` | PMM-approved positioning |
| `docs/marketing/launches/phase-34-discord-runbook.md` | This file |

**Last updated:** 2026-04-23
