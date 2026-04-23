# Phase 34 Discord Launch Runbook — April 30, 2026

**Owner:** Community Manager + DevRel
**GA date:** April 30, 2026
**Primary channels:** `#announcements`, `#general`, `#bug-reports`, `#partner-program`

---

## 1. Pre-launch checklist

Complete all items before 09:00 UTC on April 30.

### Docs and links
- [ ] `/docs/api/partner-keys` is live and returns 200
- [ ] `phase-34-community-faq.md` is published and linked from the announcement
- [ ] Phase 34 changelog is live
- [ ] All code examples in the announcement and FAQ have been verified against the production API

### Feature verification
- [ ] `message.metadata.tool_trace` is present in a real A2A response (smoke test in staging, then prod)
- [ ] `PUT /cp/platform-instructions` succeeds with an org admin token in prod
- [ ] `POST /cp/admin/partner-keys` returns a `mol_pk_*` key in prod
- [ ] `DELETE /cp/admin/partner-keys/:id` completes cleanly and billing stops
- [ ] SaaS Fed v2: confirm federated org boundary enforcement is behaving correctly in prod

### Discord setup
- [ ] Announcement message is drafted and reviewed (see Section 2)
- [ ] `#announcements` is set to slow mode or admin-only post for the first 30 minutes
- [ ] Community Manager and at least one DevRel engineer are online and available at 09:00 UTC
- [ ] Dev Lead is reachable (Slack/phone) from 09:00–13:00 UTC
- [ ] `#partner-program` pinned message is up to date with April 30 GA date

### Comms
- [ ] Blog posts for Tool Trace and Platform Instructions are published and URLs are confirmed
- [ ] GitHub Discussions announcement post is drafted and ready to publish
- [ ] Email announcement (if applicable) is scheduled and reviewed

---

## 2. Launch announcement message template

Paste this into `#announcements` at 09:00 UTC on April 30. Edit anything in `[brackets]` before posting.

---

**Phase 34 is GA. Four new features for platform builders, live now.**

- **Tool Trace** — every A2A response now includes `message.metadata.tool_trace`: a full list of every tool your agent called, with inputs and output previews. Parallel calls are handled correctly via `run_id`. Capped at 200 entries. No config needed — it's there already.
- **Platform Instructions** — org admins can now set org-wide system instructions with `PUT /cp/platform-instructions`. Set once, every agent in your org inherits it. Available on all plans.
- **Partner API Keys (`mol_pk_*`) — GA today** — programmatically create and manage Molecule AI orgs via API. Ephemeral orgs per PR, org-scoped keys, automated billing teardown on `DELETE`. Full docs at `/docs/api/partner-keys`.
- **SaaS Federation v2** — improved reliability and org boundary enforcement for federated multi-org deployments. See the changelog for details.
- Questions? Reply here or check the [FAQ link]. Bugs go to `#bug-reports`. Partner program interest goes to `#partner-program`.

---

## 3. Channel monitoring schedule

### 09:00–11:00 UTC (first 2 hours — highest traffic window)

| Channel | What to watch for | Who monitors |
|---|---|---|
| `#announcements` | Questions in thread replies; confusion about Partner Key GA date | Community Manager |
| `#general` | Tool Trace not appearing in responses; plan eligibility questions; "where is X" | DevRel Engineer |
| `#bug-reports` | Any new reports tagged Phase 34 / Tool Trace / Platform Instructions / Partner Keys | DevRel Engineer (triage) + Dev Lead (escalation) |
| `#partner-program` | Access requests for Partner API Keys; integration questions | Partner team / mol-ops |

### 11:00–17:00 UTC (steady state)

| Channel | What to watch for | Who monitors |
|---|---|---|
| `#general` | Ongoing questions; sentiment check | Community Manager (async, check every 60 min) |
| `#bug-reports` | New reports; ensure nothing is going unacknowledged for >30 min | DevRel Engineer |
| `#partner-program` | Access requests accumulating; flag to mol-ops if queue is growing | Community Manager |

### 17:00–24:00 UTC (end of day + late traffic)

| Channel | What to watch for | Who monitors |
|---|---|---|
| `#bug-reports` | Any late-breaking issues; confirm nothing severity-1 is open unacknowledged | On-call DevRel |
| `#general` | Any unresolved threads from earlier | On-call DevRel |

---

## 4. Escalation paths

### Tool Trace issues
**Symptoms:** `tool_trace` missing from responses, incorrect entries, parallel call pairing broken, entries exceeding 200 cap unexpectedly.
**First response:** DevRel Engineer (acknowledge in `#bug-reports`, gather: org ID, request body, response body)
**Escalate to:** Dev Lead if the issue is reproducible in prod and affects more than one org

### Platform Instructions bugs
**Symptoms:** Instructions not applying to all agents, workspace-scope overriding global-scope incorrectly, `PUT /cp/platform-instructions` returning non-200.
**First response:** DevRel Engineer (acknowledge and reproduce)
**Escalate to:** Dev Lead with a reproduction case

### Partner API Keys — access and provisioning questions
**Symptoms:** "I can't create a key", "my `mol_pk_*` key isn't working", "I need early access".
**First response:** Community Manager or DevRel (direct to `/docs/api/partner-keys` and `#partner-program`)
**Escalate to:** mol-ops / partner team for provisioning issues or access grants that need manual intervention

### Performance and uptime issues
**Symptoms:** Elevated latency, 5xx errors across multiple endpoints, federation dropping events at scale.
**First response:** DevRel Engineer (check status page, post acknowledgment in `#general` within 10 minutes)
**Escalate to:** Dev Lead + PM immediately — treat as a Sev-1 incident. Do not wait for multiple reports.

---

## 5. Canned response templates

Copy, personalize lightly (add the user's name or org ID where natural), and post.

---

**CR-1: "Is this feature on my plan?"**

> Great question — Tool Trace, Platform Instructions, and the other Phase 34 features are available on all plans, including the free tier. Partner API Keys are also available on all plans starting today. No upgrade needed. If you're still not seeing something, let us know your org ID and we'll take a look.

---

**CR-2: "I don't see `tool_trace` in my response"**

> `tool_trace` lives at `message.metadata.tool_trace` in the A2A response — it's there by default with no config required. A couple of things to check: (1) make sure you're reading from the `metadata` field of the `Message` object, not the top-level response, and (2) confirm you're making an A2A call (not a direct completions call). If you're doing both of those and still not seeing it, paste a redacted version of your response and we'll dig in.

---

**CR-3: "How do I get a Partner API Key?"**

> Partner API Keys are GA today. Full docs are at `/docs/api/partner-keys` — you'll find the endpoints for creating (`POST /cp/admin/partner-keys`) and deleting (`DELETE /cp/admin/partner-keys/:id`) keys there. If you run into any access issues or have questions about what you can build, drop details in `#partner-program` and the partner team will get back to you.

---

**CR-4: "How do I set up Platform Instructions?"**

> Platform Instructions are set with a single API call using your org admin token:
>
> ```http
> PUT /cp/platform-instructions
> Authorization: Bearer <your-org-admin-token>
> Content-Type: application/json
>
> { "instructions": "Your org-wide instructions here." }
> ```
>
> Once set, every agent in your org inherits them automatically — no changes needed to individual workspace configs. Let us know if you hit any issues.

---

**CR-5: "What's the difference between global and workspace scope for Platform Instructions?"**

> Global scope (set via `PUT /cp/platform-instructions`) applies to your entire org — every workspace and every agent. Workspace scope applies only to a specific workspace. When both are set, they're combined: your global instructions always run alongside whatever workspace-specific instructions are in place. Global is the right home for compliance rules or anything org-wide; workspace scope is for context specific to one team or product area.

---

## 6. Post-launch: 24-hour metrics and feedback routing

### Metrics to capture by 09:00 UTC May 1

**Usage**
- [ ] Number of A2A responses containing a non-empty `tool_trace` (past 24h)
- [ ] Number of orgs that have called `PUT /cp/platform-instructions`
- [ ] Number of `mol_pk_*` keys created on April 30
- [ ] Number of `DELETE /cp/admin/partner-keys/:id` calls (teardown events)

**Support volume**
- [ ] Total `#bug-reports` posts tagged Phase 34 (triage status for each: resolved / open / escalated)
- [ ] Total `#partner-program` access requests received
- [ ] Total `#general` threads that required a response

**Sentiment**
- [ ] Note any recurring confusion themes (e.g., common misunderstanding about where `tool_trace` lives)
- [ ] Note any feature requests that came up more than twice

### Feedback routing

| Source | Route to |
|---|---|
| Bug reports still open after 24h | Dev Lead — create GitHub issues |
| Feature requests (2+ mentions) | PM — add to backlog review queue |
| Partner integration questions unanswered | mol-ops / partner team |
| Positive shoutouts / testimonials worth amplifying | Community Manager — consider quoting in next newsletter or social post |
| Confusion patterns that suggest doc gaps | DevRel — file doc improvement tasks against `phase-34-community-faq.md` |
