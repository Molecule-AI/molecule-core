---
id: citation-builder
name: Citation Builder
description: Submit Reno Stars to one business directory per run. Reads a queue, picks the next pending site, attempts signup + listing via Chrome CDP, logs result. Pings Telegram on captcha / email-verify blockers so a human can finish.
tags: [seo, citations, backlinks, browser]
---

# Citation Builder — one directory per run

## Purpose

Replaces the $2K/mo NetSync / Whitespark-style paid services with a daily cron
that submits Reno Stars to business directories ourselves. Each run picks ONE
pending directory from `queue.json` and attempts the full signup + listing flow.

The key rule: **do not batch**. One directory per run. Slow wins. Each site
has unique quirks (Hotfrog is a React SPA with hydration flake, TupaloTupalo does
email-magic-link, Cylex has Cloudflare, BBB requires phone verify). Trying to
submit 10 per run inevitably hits rate limits or cascading failures. Trust the
queue — at 1/day, 20 directories take three weeks of hands-off work.

## Inputs

- `/configs/plugins/.../business-profile.json` — canonical NAP + categories +
  descriptions. The [browser-automation plugin](../../../../plugins/browser-automation/)
  must be installed so Chrome CDP is reachable at `host.docker.internal:9223`.
- `/configs/skills/citation-builder/queue.json` — ordered list of directories.
  Each entry: `{name, url, status, priority, notes?}`.
- `/configs/skills/citation-builder/scripts/<site>.cjs` — per-site adapter
  (optional). If absent, the generic adapter runs and escalates.

## Flow (one run)

1. Read `queue.json`. Pick the first entry with `status: "pending"`. If none,
   log "queue exhausted" + Telegram a completion summary + exit.
2. Load `business-profile.json`. Assemble form data (name, address, phone,
   email, category, description, hours, logo URL).
3. Look for a per-site adapter at `scripts/<site>.cjs`. If present, run it.
   If absent, run the generic adapter (below).
4. Capture outcome into `status` field + append to
   `/configs/skills/citation-builder/log.jsonl`:
   - `live` — listing is visible on the public directory URL (verify-before-commit)
   - `pending_email_verify` — submitted but waiting on email link click
   - `pending_human` — captcha / phone verify / manual step needed
   - `failed` — hard error; include reason
5. If `pending_email_verify`, open Gmail (Chrome profile has it logged in),
   search `from:<site-domain>`, click the verification link, then re-verify the
   listing is live. If it is, update status to `live`.
6. Send Telegram summary — one line per attempted directory this run. Include
   the public URL if live; include "needs human" otherwise.

## Generic adapter (fallback)

```javascript
// scripts/_generic.cjs — invoked when no per-site adapter exists
// 1. Navigate to {url} from queue.json
// 2. Detect form shape: search for inputs matching /business.name|company/i,
//    /phone/, /email/, /address/, /website/, /description/, /categor/i,
//    /city/, /postal|zip/
// 3. Fill what matches; leave others blank
// 4. Click the most-prominent submit button with text matching
//    /submit|continue|register|sign.?up|get.?started|add.my.business/i
// 5. Wait 5s, screenshot, evaluate response body for
//    "success|thank you|check your email|verify your email"
// 6. If match → pending_email_verify. Otherwise → pending_human.
```

Never brute-force a site that rejects the generic adapter. Escalate to
`pending_human` and move on — a per-site adapter can be written later from
the screenshot.

## Adding a per-site adapter

When the generic adapter can't finish a submission, the human (or a follow-up
cron run) can author `scripts/<site>.cjs` that:

- Uses `lib/connect.js` from the `browser-automation` plugin (never
  `puppeteer.launch()` or raw `puppeteer.connect({defaultViewport:<anything>})`).
- Handles site-specific quirks: iframes, Shadow DOM, multi-step wizards,
  conditional fields.
- Exits with `exit 0 + {status: 'live'|'pending_email_verify'|'pending_human'|'failed', reason}`
  on stdout as JSON on the last line.

Refer to the 7 social-media helpers in
[`skills/social-publish/scripts/`](../social-publish/scripts/) for the canonical
pattern (mouse.click + keyboard.type, modal-top-right filters, multi-Lexical
disambiguation).

## Hard rules

- NEVER freestyle puppeteer. Always use the plugin's `lib/connect.js` so
  `defaultViewport: null` is enforced.
- NEVER spam retries on the same site — one attempt per run. If it fails, mark
  `pending_human` and move on.
- NEVER fabricate NAP data. Pull only from `business-profile.json`. If a field
  is missing there, ask (via Telegram), don't invent.
- Photo uploads are OUT OF SCOPE for this skill. Listings with only NAP are fine
  — photos can be added manually later.

## Tracker schema (`queue.json`)

```json
{
  "entries": [
    {
      "name": "Hotfrog",
      "url": "https://admin.hotfrog.ca/login/register",
      "priority": 1,
      "status": "pending",
      "last_attempt": null,
      "public_listing_url": null,
      "notes": "React SPA, hydration flake — may need 2-3 attempts"
    }
  ]
}
```

## Schedule

Once per day, 7:30 AM Vancouver. Paired with SEO Builder (6:17 AM) and SEO
Weekly Report so the whole "SEO loop" runs in a ~90 min window each morning.
