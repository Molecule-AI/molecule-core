# Social Media Engage — Reno Stars

Search platforms for relevant posts about renovation and Vancouver. Draft replies for approval, then publish approved replies.

## HONESTY RULE (CRITICAL)
All engagement replies must be truthful. Only reference real data from the website, database, or owner-provided information.
- Do NOT guess prices, timelines, or project details. If you don't have real data, say "it varies" or "hard to say without seeing the space".
- Do NOT claim specific project counts ("we've done 100+ kitchens") unless verified from the DB.
- Do NOT fabricate case studies, testimonials, or statistics.
- When sharing tips or advice, only share what Reno Stars actually does — don't make up practices or policies.
- It's OK to share general renovation knowledge, but never attribute specific numbers to Reno Stars without verifying.

## Config
Read `/Users/renostars/reno-star-business-intelligent/config/env.json` for credentials.

```bash
BOT_TOKEN=$(jq -r '.telegram.bot_token' /Users/renostars/reno-star-business-intelligent/config/env.json)
CHAT_ID="-5219630660"
```

## Pending Replies File
`/Users/renostars/.openclaw/workspace/social/pending-replies.json`
```json
{"pending": [], "last_telegram_update_id": 0}
```
(Create if it doesn't exist)

---

## PHASE 1: Publish Approved Replies

Load `pending-replies.json`. For each item with `status: "approved"`:
1. Navigate to the original post URL
2. Post the reply (platform-specific method below)
3. Update status to `"published"`
4. Send Telegram confirmation: `✅ Reply posted on [platform]: "[reply preview]"`

---

## PHASE 2: Search for Relevant Posts

Connect to Chrome CDP at `http://127.0.0.1:9222` using puppeteer-core at `/opt/homebrew/lib/node_modules/puppeteer-core`.
Launch if needed: `open -na "Google Chrome" --args --user-data-dir="/Users/renostars/.openclaw/chrome-profile" --remote-debugging-port=9222`
Remove dialogs after each navigation: `document.querySelectorAll('[role=dialog],[aria-modal=true]').forEach(el => el.remove())`

**Target: find 5-8 posts across all platforms to engage with per run.**

### What to look for (prioritized by lead potential):

**🔴 HIGH INTENT — these are potential customers (prioritize these):**
- "Does anyone know a good contractor/renovator in Vancouver/BC?"
- "Looking for recommendations for [kitchen/bathroom/basement] reno"
- "Just bought a house, need to renovate..."
- "Our contractor ghosted us mid-project" / "Need someone to finish a project"
- Posts showing DIY disasters, water damage, or "is this normal?"
- Comments under reno content: "I wish I could do this to my kitchen" / "How much would this cost?"

**🟡 MEDIUM — good for visibility and network building:**
- Sharing a renovation project (compliment, ask a genuine question)
- Discussing costs, timelines, contractor experiences
- Interior designers, real estate agents, property managers posting about properties
- Before/after transformations (react naturally)

**🟢 LOW — casual engagement for algorithm presence:**
- Satisfying renovation/construction videos (quick genuine reaction)
- Home design inspiration content

**Skip:** competitor promotions, political/controversial topics, anything off-topic.

### How to reply (the Help → Relate → Be Available framework):

1. **Lead with genuine help or reaction** — answer their question or react to their content
2. **Add a personal touch if natural** — "we ran into this exact thing last month..." or "this is so satisfying to watch"
3. **Soft availability on HIGH INTENT posts only** — "happy to answer any other questions" (NOT "DM me for a quote")
4. **On recommendation requests** — give genuinely useful hiring advice (check insurance, pull permits, get references). Your profile does the selling. People click through, see your work, and DM YOU.

**NEVER:**
- Drop phone number or website in comments
- Say "We can help! DM us" — fastest way to get ignored
- Copy-paste the same reply across posts (flagged as spam)
- Pitch in someone else's thread
- Use every comment as a teaching moment (be human first)

### TikTok
Search `https://www.tiktok.com/search?q=vancouver+renovation` and `https://www.tiktok.com/search?q=bathroom+renovation+before+after`.
Look for videos showing renovation projects or asking for advice. React naturally — compliment, laugh, relate. Under 100 chars.

### YouTube
Search `https://www.youtube.com/results?search_query=vancouver+renovation+2026` and `https://www.youtube.com/results?search_query=home+renovation+cost+breakdown`.
Look for videos where you can add a genuine reaction or relate to the content. Under 200 chars.

### Instagram ⭐ HIGH PRIORITY
Search and engage on these accounts/hashtags:
- `https://www.instagram.com/explore/tags/vancouverrenovation/` — recent posts tagged with Vancouver renovation
- `https://www.instagram.com/explore/tags/beforeandafter/` — transformation posts
- `https://www.instagram.com/explore/tags/kitchenrenovation/` — kitchen content
- `https://www.instagram.com/explore/tags/bathroomrenovation/` — bathroom content
- Local Vancouver home/design accounts — interior designers, real estate agents, home stagers

**Instagram engagement rules:**
- Comment on 3-5 posts per run
- Be genuinely impressed, ask real questions, relate to the content
- NO "great work, check us out!" — that's spam
- Good: "the backsplash choice is everything 😍" / "how long did this take? looks incredible"
- Engage with LOCAL Vancouver/BC content first (builds local network)
- Like + comment together (signals genuine engagement to the algorithm)

### LinkedIn ⭐ ADDED
Search `https://www.linkedin.com/search/results/content/?keywords=vancouver%20renovation&datePosted=past-24h` and `https://www.linkedin.com/search/results/content/?keywords=commercial%20renovation%20vancouver&datePosted=past-24h`

**LinkedIn engagement rules:**
- Comment on posts from: real estate agents, property managers, commercial developers, architects, interior designers in Metro Vancouver
- Tone: professional but conversational. Share a genuine insight or experience.
- Good: "We see this a lot with pre-sale renos — the ROI on kitchen updates is consistently the highest in the Lower Mainland market"
- Good: "Interesting perspective. The permit timeline in Vancouver is definitely the hidden cost most people don't budget for"
- NO sales pitch. Position as a knowledgeable industry peer, not a vendor.
- 2-3 comments per run

### Reddit — PAUSED until 2026-04-21

The Reno Stars Reddit account was deleted on 2026-04-07 after a fresh-account shadow ban.
Skip Reddit entirely until a new account is created. The user will be reminded on 2026-04-21.

### X / Twitter
Search `https://x.com/search?q=vancouver+renovation+contractor&f=live` and `https://x.com/search?q=bathroom+renovation+vancouver&f=live`
React to recent tweets about renovation experiences. Keep it casual.

### Facebook
Search `https://www.facebook.com/search/posts/?q=vancouver+renovation+contractor`
Look for posts in public groups asking for contractor recommendations. Share helpful answers, no pitch.

### Xiaohongshu — ⚠️ PAUSED (platform warning 2026-04-09)
**SKIP Xiaohongshu in all engage runs.** Do not search, draft, or post replies on Xiaohongshu until user re-enables.

---

## PHASE 3: Draft Replies

For each relevant post found (max 5 total per run), draft a reply.

**Reply rules:**
- Sound like a REAL PERSON casually scrolling, not an expert dispensing advice
- React naturally — laugh, compliment, be impressed, joke around
- Keep it SHORT. TikTok: 1-2 sentences max. YouTube: 2-3 sentences max.
- NEVER mention Reno Stars, services, phone numbers, or website. The account name already shows who we are.
- No "pro tips", no "key things to watch", no numbered advice lists
- Only share a genuine insight if it flows naturally from the conversation — don't force it
- Match the platform's energy (TikTok = casual/fun, YouTube = slightly more detailed, Reddit = conversational)
- Max reply length: TikTok 100 chars, YouTube 200 chars, Reddit 200 words

**Reply tone examples:**
- Good (TikTok): "that transformation is insane 🔥"
- Good (TikTok): "the before made me physically uncomfortable lol"
- Good (YouTube): "This is exactly what my kitchen looked like before we gutted it. The difference is night and day 👏"
- Good (YouTube): "The tile choices are so clean. How long did the bathroom take start to finish?"
- Bad: "Pro tip: always seal the edges with silicone so moisture can't get behind them 💧"
- Bad: "One thing I'd add: check your plumbing before starting any bathroom reno..."
- Bad: "We do this at Reno Stars — feel free to reach out"

---

## PHASE 4: Save Drafts and Send for Approval

Generate a unique reply ID: `reply_YYYYMMDD_HHMMSS_N`

Append each draft to `pending-replies.json`:
```json
{
  "id": "reply_20260406_120000_1",
  "created_at": "<ISO>",
  "status": "pending_approval",
  "platform": "reddit",
  "post_url": "<original post URL>",
  "post_title": "<original post title>",
  "post_preview": "<first 100 chars of original post>",
  "reply_draft": "<the reply text>",
  "subreddit": "vancouver",
  "telegram_message_id": null
}
```

Send a single consolidated Telegram message with all drafts:

```
💬 ENGAGEMENT DRAFTS — [date]

[For each draft:]
━━━━━━━━━━━━━━━━━━
🟠 REDDIT r/[subreddit] — [reply_id]
Original: "[post title]"
URL: [post_url]

Draft reply:
"[reply_draft]"

Reply: REPLY [reply_id] to approve
━━━━━━━━━━━━━━━━━━

APPROVE ALL: REPLY ALL to approve everything above
```

---

## PHASE 5: Post Approved Replies (when publishing)

> **READ FIRST**: `~/.claude/skills/social-media-post/SKILL.md` and memory `feedback_social_media_platforms.md` for platform-specific quirks and failure modes. The notes below are reply-flow-specific.

**Universal reply rules:**
- **NO promotional CTAs.** No "We do X at Reno Stars", no "feel free to reach out", no "happy to help". The account name attributes the brand. The user will be upset if you add CTAs (confirmed 2026-04-07).
- **Pace publishes 60–90 seconds apart on Reddit.** Posting 4+ comments back-to-back triggers a `Rate limit exceeded` cooldown of ~9–10 minutes. Spread the load across the run.
- **Reddit account is PAUSED until 2026-04-21** — see top of file. Do not draft or attempt Reddit replies.
- **Disable beforeunload** preemptively on TikTok/Xiaohongshu/Facebook tabs before typing into composers.

### Reddit ⚠️ PAUSED
Account `u/Anxious-Owl-9826` was deleted 2026-04-07. Skip entirely until 2026-04-21.

When the new account exists: navigate to the post URL → click the comment box at the bottom (`comment-composer-host` element on shreddit, focus via `getByLabel('').click()`) → type reply via browser_type → click the `Comment` submit button.

### X
Navigate to the tweet URL → click "Reply" → type via browser_type → click `Reply` button. Same overlay-intercept issue as posting; if click fails, use:
`document.querySelector('[data-testid="tweetButtonInline"]').click()`

### LinkedIn
Navigate to post URL → click "Comment" → type reply → click Post.

### Facebook
Navigate to post URL → find comment box → type reply → press Enter or click Post.

### Xiaohongshu
Navigate to note URL → find comment input → type Chinese reply → submit. **No external links / phone / address** in the reply.

### TikTok
Navigate to video URL → find the comment input at the bottom → type reply (max 150 chars) → post. **Use `playwright.keyboard` for typing, NEVER `execCommand insertText`** (TikTok uses Lexical and execCommand crashes the editor — see skill).

### YouTube
Navigate to video URL → find the comment input below the video → type reply → click "Comment".

---

## Self-Improvement (every run, end of run)

Same loop as the poster cron's PHASE 6 — if you encountered something new that isn't already in:
- `~/.claude/skills/social-media-post/SKILL.md`, OR
- `~/.claude/projects/-Users-renostars/memory/feedback_social_media_platforms.md`

…and it's a real recurring pattern (not a one-off), use `Edit` to add it surgically (additive, dated `(YYYY-MM-DD)`) and notify the user via Telegram:
```
📚 Skill update from social-media-engage: <one line>
File: <path>
```

If you're unsure whether it's worth a skill update, append an observation to `/Users/renostars/reno-star-business-intelligent/data/social-media-observations.jsonl` for the user to triage:
```json
{"timestamp":"<ISO>","platform":"<name>","observation":"<what you saw>","action_suggestion":"<what to do>"}
```

---

## Log

Append to `/Users/renostars/reno-star-business-intelligent/data/cron-logs/social-media-engage.jsonl`:
```json
{"timestamp":"<ISO>","job":"social-media-engage","draftsCreated":<n>,"repliesPublished":<n>,"platforms":["reddit","x"],"error":null,"phase6_action":"none"|"skill_updated"|"observation_logged"}
```
