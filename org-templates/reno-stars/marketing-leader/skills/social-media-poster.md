# Social Media Poster — Reno Stars

Draft and queue social media posts for approval, then publish approved posts.

## Config
Read `/Users/renostars/reno-star-business-intelligent/config/env.json` for DB connection and Telegram credentials.

## Pending Posts File
`/Users/renostars/.openclaw/workspace/social/pending-posts.json`
```json
{"pending": [], "last_telegram_update_id": 0}
```

## Active Platforms
- **Facebook**: Business Page https://www.facebook.com/profile.php?id=100068876523966
- **Instagram**: https://www.instagram.com/renostarsvancouver/ (linked to Facebook account)
- **X (Twitter)**: @Renostars_ca — https://x.com/Renostars_ca
- **LinkedIn**: https://www.linkedin.com/ (logged in)
- **Xiaohongshu**: PAUSED — platform warning received 2026-04-09. Do NOT auto-post. Skip Xiaohongshu in all cron runs until user explicitly re-enables.
- **TikTok**: https://www.tiktok.com/ (logged in) — use Photo Mode (slideshow) since no video yet
- **YouTube**: https://www.youtube.com/ (logged in) — use Community Posts (image + text)
- **Google Business Profile**: Post via Google Search panel (search "Reno Stars Local Renovation Company Richmond BC", click "Add update" in the business panel). Logged in as ${OPERATOR_EMAIL}. Posts appear on Google Search + Maps knowledge panel.
- **Reddit**: PAUSED until 2026-04-21 — account was deleted on 2026-04-07 after fresh-account shadow ban. Skip Reddit entirely until then. Do not draft Reddit content, do not include "reddit" in platforms array, do not navigate to reddit.com.

---

## PHASE 0: Trend Research (once per day, first morning run)

**Goal:** Keep our content fresh and aligned with what's actually working in the renovation/home-design space right now. Cached for 24h to avoid burning tokens on every 6h run.

**When to run this phase:**
1. Check `mtime` of `/Users/renostars/reno-star-business-intelligent/data/trend-insights.md`.
2. If the file doesn't exist OR was last modified more than 22 hours ago, run the research below.
3. Otherwise, **skip Phase 0** and use the existing cached insights when drafting in Phase 2.

**Research checklist** (do all of these in parallel using web search; don't get bogged down in any one):
1. **Current renovation trends (last 30 days)** — search WebSearch for queries like:
   - "bathroom renovation trends 2026"
   - "kitchen design trends Vancouver"
   - "home renovation Instagram top posts"
   - Look at Houzz, Architectural Digest, Apartment Therapy, BC Living
2. **What's hot on r/HomeImprovement and r/HomeDecorating** — top posts of the past week. Use the public JSON endpoint:
   `curl -s -A "Mozilla/5.0" "https://www.reddit.com/r/HomeImprovement/top.json?t=week&limit=10" | jq '.data.children[].data | {title, score, num_comments}'`
   Same for r/Renovation, r/centuryhomes, r/InteriorDesign. Note recurring themes and what tone the high-engagement posts use.
3. **Vancouver-specific signals** — search for "Vancouver real estate renovation" news, recent home-pricing articles. Local context lifts engagement on Vancouver-targeted posts.
4. **Hashtag trends** — check what hashtags are trending on Instagram for #renovation, #homereno, #vancouverhomes. (Browse via the Instagram explore tab if web search is thin.)
5. **Competitor accounts** — quick scan of 2–3 well-followed Vancouver renovation companies on Instagram/TikTok (e.g. search "Vancouver renovation contractor instagram"). Note their hook formulas, post cadence, format (before/after, time-lapse, walkthrough, designer-talk).

**Output:** append (don't overwrite — keep history) to `/Users/renostars/reno-star-business-intelligent/data/trend-insights.md`. Format:

```markdown
## YYYY-MM-DD trend snapshot

**What's working right now:**
- [bullet] (e.g., "before/after vertical reels still dominate IG saves; bathroom > kitchen this week")
- [bullet]

**Trending hashtags / keywords:**
- [list]

**Hook formulas to try this cycle:**
- "[paste actual high-performing hook from competitor or top post]" — why it works
- "[another]"

**Topics to lean into:**
- [bullet — e.g. "small-bathroom space-saving tricks; r/HomeImprovement engagement is 3x normal this week"]

**Topics to avoid:**
- [bullet — e.g. "AI-generated designs got mocked in top comments; don't lead with that"]

**Vancouver-local angles:**
- [bullet — e.g. "BC strata bylaw changes for renos coming 2026 — relevant to townhouse projects"]

---
```

Cap the file at the **most recent 30 snapshots** — if longer, drop the oldest entries when appending.

**Then in PHASE 2 STEP 2** (draft generation), read the most recent snapshot from this file and use the hook formulas / topics / hashtags to inform the drafts. Reference specific insights in the draft notes so the user can see them in the Telegram approval.

---

## Mode Override: PUBLISH_ONLY

If the env var `POSTER_MODE=publish_only` is set, OR an `[OVERRIDE: PUBLISH_ONLY]` line appears anywhere in this prompt, **skip Phase 0 (trend research) and Phase 2 (draft new content) entirely**. Run only Phase 1 (publish approved posts) and Phase 6 (self-improvement). This mode is used when a human approves a pending draft via chat and wants the publishing to happen immediately without spawning yet another approval cycle.

To check: `printenv POSTER_MODE` — if it equals `publish_only`, jump straight to Phase 1 and exit after Phase 6.

---

## PHASE 0.7: Video Day Check (every 2 days)

The cron runs once a day at 9:30 AM Vancouver. **Every other run** (i.e. when the previous video post was ≥ 2 days ago) the post should be a Dreamina before/after morph video instead of static images.

### Decide whether today is a video day

```bash
HISTORY=/Users/renostars/reno-star-business-intelligent/data/dreamina-video-history.jsonl
LAST_VIDEO_TS=$(tail -1 "$HISTORY" | jq -r '.used_at // empty' 2>/dev/null)
NOW_TS=$(date -u +%s)
LAST_TS=$(date -j -u -f "%Y-%m-%dT%H:%M:%SZ" "$LAST_VIDEO_TS" +%s 2>/dev/null || echo 0)
HOURS_SINCE=$(( (NOW_TS - LAST_TS) / 3600 ))
echo "hours since last video: $HOURS_SINCE"
# >= 47 = it's been ~2 days, do a video day. <47 = skip video, do regular photo post.
```

If hours_since ≥ 47 (allow 1h drift): **do a video day**, follow the flow below. Otherwise skip to PHASE 1 / PHASE 2 photo post logic.

### Video day flow

1. **Pick a fresh project image pair from the DB.** Connect to Neon (config → services.neon_db) and run:
   ```sql
   SELECT pip.id AS pair_id, pip.before_image_url, pip.after_image_url,
          pip.title_en, p.slug AS project_slug, p.title_en AS project_title,
          p.location_city, p.budget_range, p.duration_en
   FROM project_image_pairs pip
   JOIN projects p ON p.id = pip.project_id
   WHERE pip.before_image_url IS NOT NULL
     AND pip.after_image_url IS NOT NULL
     AND p.is_published = true
     AND p.slug NOT IN (
       -- exclude already-used projects from history
       <comma-separated list of project_slug values from dreamina-video-history.jsonl>
     )
   ORDER BY p.created_at DESC
   LIMIT 5;
   ```
   Pick the **first row** (most recent unused project). Save the chosen `pair_id`, `project_slug`, `before_image_url`, `after_image_url` for the rest of the flow.

2. **Generate the video on Dreamina.** Open a Chrome tab via puppeteer-core (NOT playwright MCP — see skill memory), navigate to:
   ```
   https://dreamina.capcut.com/ai-tool/home?type=video&model=dreamina_seedance_40_pro
   ```
   The Dreamina interface accepts a "first frame" + "last frame" image pair plus a text prompt. Upload `before_image_url` as the first frame and `after_image_url` as the last frame. Use this exact prompt:
   ```
   首帧和尾帧是同一个地方同一个角度，这是装修前后的两个照片，我想要第一张照片里面的设施都向图片外滑走，后面新的设备在滑入, 这个期间镜头慢慢转移动到最终位置
   ```
   Click Generate. Wait for the video to render (typically 60-180 seconds). Download the resulting mp4.
   **See `~/.claude/skills/social-media-post/SKILL.md` → "Dreamina before/after morph video generation" section for the exact selectors and the download flow.**

3. **Save the video to a clean ASCII path** under `/Users/renostars/`:
   ```
   /Users/renostars/dreamina-<project_slug>-<YYYYMMDD>.mp4
   ```
   Avoid spaces, Chinese characters, or emoji in the filename — every social platform handles ASCII paths reliably; some choke on Unicode.

4. **Append to the history file BEFORE posting** (so an interrupted run doesn't try the same pair again):
   ```bash
   cat >> /Users/renostars/reno-star-business-intelligent/data/dreamina-video-history.jsonl <<EOF
   {"used_at":"$(date -u +%Y-%m-%dT%H:%M:%SZ)","project_slug":"<slug>","pair_id":"<pair_id>","video_path":"<path>","posted_post_id":null}
   EOF
   ```

5. **Build the post draft** with the chosen project's metadata (title, location, budget, duration). Caption pattern: same as a normal photo post but lead with the "before → after" hook (e.g. "Same room, same angle — before and after 🏠"). Set `video_url` in the pending-posts.json entry instead of `tiktok_images`.

6. **Send for approval via Telegram** (PHASE 2 STEP 3 normal flow), with the note that this is a video day and the file path.

7. **After publishing**, update the history entry's `posted_post_id` field with the actual post id.

### Failure handling

- If Dreamina generation fails (UI error, rate limit, image too large): log the failure to `data/social-media-observations.jsonl` and fall through to a normal photo post for today. Don't write to dreamina-video-history.jsonl on failure (so the next run retries the same pair).
- If no unused project pairs remain (all 125+ pairs used): log a Telegram alert "exhausted all before/after pairs" and fall back to photo post.
- If the chosen project's image URLs return 404: skip that pair, try the next one in the SQL result.

---

## PHASE 1: Publish Any Approved Posts

**First**, check `pending-posts.json` for items with `status: "approved"`. For each one, publish to its platform(s) and update to `status: "published"`. See STEP 3 for platform-specific posting instructions.

After publishing: update the item in pending-posts.json to `status: "published"`, then INSERT into `social_media_posts` DB table (see STEP 4).

---

## PHASE 2: Draft New Content

### STEP 1: Pick Content

Connect to Neon DB (config → services.neon_db). Find the best unposted content:

```sql
-- Projects not yet posted to any platform
SELECT p.id, p.slug, p.title_en, p.excerpt_en, p.location_city,
  p.budget_range, p.service_type, p.hero_image_url, p.solution_en,
  p.space_type_en, p.duration_en, p.created_at
FROM projects p
WHERE p.is_published = true
  AND p.hero_image_url IS NOT NULL
  AND p.id NOT IN (
    SELECT project_id FROM social_media_posts
    WHERE project_id IS NOT NULL AND status = 'published'
  )
ORDER BY p.created_at DESC
LIMIT 5;
```

Fall back to blog posts if no unposted projects:
```sql
SELECT b.id, b.slug, b.title_en, b.excerpt_en, b.featured_image_url,
  b.reading_time_minutes, b.created_at
FROM blog_posts b
WHERE b.is_published = true
  AND b.id NOT IN (
    SELECT blog_post_id FROM social_media_posts
    WHERE blog_post_id IS NOT NULL AND status = 'published'
  )
ORDER BY b.created_at DESC
LIMIT 5;
```

Pick the first result. Note its type (project/blog), id, and slug.

Also fetch before/after image pairs for the selected project (used by TikTok and YouTube):
```sql
SELECT before_image_url, after_image_url, before_alt_text_en, after_alt_text_en
FROM project_image_pairs
WHERE project_id = $selected_project_id
ORDER BY display_order ASC
LIMIT 6;
```
If it's a blog post (no project_id), skip this query — TikTok/YouTube will use `featured_image_url` only.

---

### STEP 2: Generate Drafts Per Platform

**Before drafting:** read the most recent snapshot from `/Users/renostars/reno-star-business-intelligent/data/trend-insights.md` (the bottom-most `## YYYY-MM-DD trend snapshot` block). Use its **hook formulas**, **topics to lean into**, and **trending hashtags** to inform the drafts. If the snapshot says "before/after vertical reels are dominating IG", lean the IG draft toward that. If it says "small-bathroom space-saving is hot this week", emphasize space-saving angles when the project allows.

In the Telegram approval message (STEP 3), include a one-line `Trend angle:` note showing which insight from the snapshot the drafts are leaning into. Example: `Trend angle: leveraging this week's "small-bathroom space-saving" surge on r/HomeImprovement`.

Write platform-specific drafts from the real data. No fabrication — use only fields from the DB. Trend insights guide tone and emphasis only — never invent project details that aren't in the database.

### CONTENT STRATEGY: SHARE, DON'T ADVERTISE

This is a long-term brand building strategy. We are NOT running ads — we are sharing content that people genuinely want to see, save, and send to friends.

**The 80/20 rule:** 4 out of 5 posts should teach, entertain, or show personality. Only 1 in 5 can mention services/contact info. If the feed looks like a sales flyer, reach drops.

**What drives algorithm distribution (ranked):**
1. Shares/sends — content people forward to friends ("you need to see this kitchen")
2. Saves — content people bookmark to reference later ("how to choose countertops")
3. Comments — content that sparks opinions ("what would you do with this space?")
4. Watch time — videos people watch to the end (keep under 45 sec)

**Content types to rotate through:**
1. **Process/satisfying clips** (15-30s) — tile being laid, paint rolling, demo day. Oddly satisfying = shares.
2. **Before/after with story** — NOT just glamour shots. Narrate WHY: "The homeowner wanted X but we suggested Y because..."
3. **Quick tips that feel like insider knowledge** — "3 signs your contractor is cutting corners", "Why we never skip waterproofing"
4. **Opinion/poll content** — Show a problem and ask "what would you do?" Drives comments.
5. **Team/personality** — crew intros, jobsite humor, real moments. People hire people they like.

**What to NEVER do:**
- End every post with "Call for a free estimate" / phone number / CTA
- Post only finished glamour shots with no context
- Sound like a brochure — write like a real person talking to a friend
- Use generic stock-photo graphics

**Caption rules:**
- Hook in first line (question, surprising fact, or "watch this...")
- Tell the story behind the project, not just what it looks like
- Phone/website link only on 1 out of every 5 posts — and even then, put it casually at the end, not as the focus
- On video posts: add captions always (80%+ watch muted)

#### FACEBOOK (max 500 chars body)
```
[Hook — question or story opener, NOT "we did a renovation in..."]

[2-3 sentences telling the story: what the homeowner was dealing with, what changed, how it turned out]
[A genuine detail that makes it real — timeline, a challenge we solved, a decision point]

[Only every 5th post: casual link to project page — no phone number, no "call us"]
```

#### INSTAGRAM (max 300 chars body + hashtags on new lines)
```
[Punchy hook that makes people stop scrolling — one line]

[The story in 2 sentences — what was the problem, what's the result]

[An insight or opinion that makes this more than just eye candy]

#VancouverRenovation #BeforeAndAfter #HomeRenovation #[city]Renovation
```
NO phone number. NO "link in bio". NO "DM for quote". The account name IS the branding.

#### X / TWITTER (max 250 chars total including URL)
```
[One punchy thought or reaction about the project — like you're texting a friend]

[Optional: link to project page, but only every 3-4 posts]
```

#### LINKEDIN (professional, 150-400 chars)
```
[Insight or lesson from the project — what went wrong, what we learned, what surprised us]

[2-3 sentences: the real story, not the marketing version. Be honest about challenges.]

[CTA: Free consultation → ${OPERATOR_PHONE} | reno-stars.com]

#Renovation #Vancouver #HomeImprovement #ContractorLife
```

#### XIAOHONGSHU — PAUSED (skip draft generation)

#### GOOGLE POSTS (max 1500 chars, include CTA button)
Google Business Profile posts appear on Google Search + Maps. Write as a brief business update:
```
[Project name] — Before & After ✨

[2-3 sentences about the project, location, scope, and result]

📞 Free consultation: ${OPERATOR_PHONE}
🌐 reno-stars.com
```
Keep it short and professional — these posts show in the knowledge panel next to reviews. Include a CTA like "Call now" or "Learn more".

#### TIKTOK (Photo Mode slideshow — max 35 images, max 2200 chars caption)
TikTok supports posting a series of images as a swipeable slideshow.
Select images in this order: before_1, after_1, before_2, after_2, ... (interleaved before/after for impact).
Hook in first 2 seconds of caption. Keep under 45 seconds for video. Add captions always.

Caption format:
```
[Hook that makes people stop — "Watch this 1970s kitchen disappear" / "Same room. Same angle. 6 weeks apart." / "The homeowner almost didn't do this..."]

[1-2 lines: the STORY, not the specs. What was the homeowner dealing with? What changed their mind?]

[Optional: one genuine insight — "We almost went with white tile but the grey changed everything"]

#BeforeAndAfter #HomeRenovation #[city]Renovation #RenovationLife
```
NO phone number. NO "link in bio". NO "free quote". Let the transformation speak for itself.

#### YOUTUBE (Community Post — image + text, max 5000 chars)
YouTube Community Posts work like social media posts — image + text, appear in subscribers' feeds.
Use the hero_image_url as the image. If image pairs exist, use the best after shot.

Caption format:
```
[Conversational opener — share a thought, lesson, or behind-the-scenes moment from this project]

[2-3 sentences: the real story. What was challenging? What decision made the biggest difference? What would you do differently?]

[End with a question to drive comments: "Would you have gone with the darker tile? 🤔" / "What's the one thing you'd change in your kitchen?"]

#Renovation #VancouverRenovation #HomeImprovement #BeforeAndAfter
```
NO phone number. NO "subscribe" CTA. NO "free quote". Build community through conversation.

#### REDDIT
Find the most relevant subreddit for this content:
- Kitchen/bathroom/basement reno → r/HomeImprovement or r/Renovation
- Vancouver-specific → r/vancouver or r/BritishColumbia
- General → r/DIY (framed as project showcase, not ad)

Write a helpful post — share the project story as a contractor case study. Frame it as educational/informative, not promotional. Keep the business name subtle (end of post only). No direct "hire us" language.

```
Title: [Specific, descriptive — e.g. "Completed a [space type] reno in [city] — here's what we learned about [specific challenge]"]

Body: [Project context, challenge, solution, outcome. 2-3 paragraphs. Factual.
       Mention contractor name once at the end: "— Reno Stars, Vancouver"]
```

---

### STEP 3: Save Draft and Send for Approval

Generate a unique post ID: `post_YYYYMMDD_HHMMSS`

Append to `pending-posts.json`:
```json
{
  "id": "post_20260406_200000",
  "created_at": "<ISO timestamp>",
  "status": "pending_approval",
  "content_type": "project" | "blog",
  "content_id": <db_id>,
  "content_slug": "<slug>",
  "image_url": "<hero_image_url or featured_image_url>",
  "platforms": ["facebook", "instagram", "x", "linkedin", "tiktok", "youtube", "google_posts"],
  "drafts": {
    "facebook": "<facebook draft text>",
    "instagram": "<instagram draft text>",
    "x": "<x draft text>",
    "linkedin": "<linkedin draft text>",
    "tiktok": "<tiktok caption>",
    "youtube": "<youtube community post text>",
    "tiktok_images": ["<before_url_1>", "<after_url_1>", "<before_url_2>", "<after_url_2>"],
    "reddit": {
      "subreddit": "HomeImprovement",
      "title": "<reddit post title>",
      "body": "<reddit post body>"
    }
  },
  "telegram_message_id": null
}
```

Send Telegram notification:
```bash
BOT_TOKEN=$(jq -r '.telegram.bot_token' /Users/renostars/reno-star-business-intelligent/config/env.json)
CHAT_ID="-5219630660"
```

Message format:
```
📋 NEW POST DRAFT — [content_type]: [title]

📘 FACEBOOK:
[facebook draft]

📸 INSTAGRAM:
[instagram draft]

🐦 X:
[x draft]

💼 LINKEDIN:
[linkedin draft]

🟠 REDDIT (r/[subreddit]):
[reddit title]
[reddit body first 200 chars]...

🖼 Image: [image_url]

Reply: APPROVE [post_id] to publish all platforms
Or: APPROVE [post_id] facebook,instagram to publish specific platforms only
```

---

## STEP 3: Platform Posting (when publishing approved posts)

> **READ FIRST**: `~/.claude/skills/social-media-post/SKILL.md` — comprehensive playbook for every platform with the quirks learned the hard way on 2026-04-07. Also see memory `feedback_social_media_platforms.md` for the failure-mode index. The instructions below are the cron-specific shortcuts; the skill is the source of truth.

**Universal pre-flight (do BEFORE touching any platform):**
- Files for upload MUST be under `/Users/renostars/`. Copy first if elsewhere.
- Avoid spaces / Chinese / ellipsis in filenames — some upload widgets choke.
- Disable beforeunload preemptively on long forms (TikTok especially): `await page.evaluate(() => { window.onbeforeunload = null; window.addEventListener('beforeunload', e => e.stopImmediatePropagation(), true); });`
- **NO promotional CTAs** in any post — no "We do X at Reno Stars", no "feel free to reach out". The account name attributes the brand.
- Wrap risky/slow site calls in `mcp__playwright__browser_run_code` with explicit short timeouts (10s nav, 5s clicks).

Connect to Chrome CDP at `http://host.docker.internal:9223` using puppeteer-core at `puppeteer-core`.
Launch Chrome if needed: `# Chrome runs on host — connect via host.docker.internal:9223 (CDP proxy)`
Wait 4s. Remove dialogs: `document.querySelectorAll('[role=dialog],[aria-modal=true]').forEach(el => el.remove())`

If the post has an image_url, download it first:
```javascript
const https = require('https'), fs = require('fs');
const ext = image_url.match(/\.(jpg|jpeg|png|webp)/i)?.[1] || 'jpg';
const localPath = `/tmp/social-post-image.${ext}`;
// stream image_url to localPath
```

### Facebook (Page)
1. Navigate to `https://www.facebook.com/profile.php?id=100068876523966`
2. **For VIDEO**: click the **Reel** button in the composer row (NOT "Photo/video" — that path fails on long videos).
   - "Create reel" dialog → "Add video or drag and drop" → file picker → upload.
   - Wait for upload, click `Next` to advance to Edit.
   - Click `Next` again to advance to "Reel settings".
   - Fill the description textbox (contenteditable, plain fill works).
   - Click `Post`. Toast: "Your Post is successfully shared with EVERYONE".
3. **For IMAGE/TEXT only**: click "What's on your mind?" → verify composer shows "Reno Stars" (not personal name) → type via `execCommand insertText` → if image, click "Photo/video" → upload → "Next" → "Post". Dismiss any boost/CTA dialog with "Not now".

### Instagram
1. Navigate to `https://www.instagram.com/`
2. Click the "New post" link in the left nav, then "Post" in the popout submenu
3. "Create new post" modal → "Select from computer" → upload
4. **VIDEO** triggers a "Video posts are now shared as reels" info dialog → click `OK`
5. Three sequential screens: Crop → Edit → Caption. Click `Next` twice to reach the caption screen.
6. Fill the "Write a caption..." textbox.
7. Click `Share`. A "Sharing" spinner dialog stays for ~10s — wait it out, don't assume it's hung.
8. **NOTE**: Instagram does NOT auto-cross-post to Facebook even though accounts are linked. Post to each separately.

### X (Twitter)
1. Navigate to `https://x.com/compose/post`
2. Click "Add photos or video" button → upload → wait for `Uploaded (100%)` status
3. Fill the "Post text" textbox (use browser_type / fill).
4. **Post button click is intercepted by an invisible overlay** in normal browser_click. Click via JS:
   `document.querySelector('[data-testid="tweetButton"]').click()`
5. Success: navigates to `https://x.com/home`. Caption max 280 chars including the URL (assume 23 chars for URLs via t.co).

### LinkedIn (Company Page)
1. Navigate to `https://www.linkedin.com/company/103326696/admin/` (Reno Stars Construction Inc.)
2. Click `Create` → "Start a post" in the dialog.
3. **Verify the composer header reads "Reno Stars Construction Inc."** — if wrong (showing personal profile), click the dropdown to switch.
4. Click "Add media" → upload video → wait for preview → click `Next`.
5. Fill the "Text editor for creating content" textbox.
6. Click `Post`.
7. **Tone**: drop emoji, write a brief case study (Challenge / Result framing) — LinkedIn audience is B2B.

### Google Business Profile (Google Posts)
1. Search Google for "Reno Stars Local Renovation Company Richmond BC" (must be logged in as ${OPERATOR_EMAIL}).
2. In the "Your business on Google" panel, click **"Add update"** (or "Posts" → "Add update").
3. An iframe opens at `/local/business/<id>/posts/create`. Select **"Add update"** post type (not Offer or Event).
4. Type the post text in the description field. Max 1500 chars.
5. **Add a photo**: click "Add photo" and upload the project's hero image via the file input.
6. **Add a CTA button**: select "Call now" or "Learn more" with URL `https://www.reno-stars.com/en/projects/<slug>/`.
7. Click **"Post"** / **"Publish"**.
8. **Success signal**: post appears in the Posts tab of the business panel.
9. Google Posts expire after 7 days (they stop showing prominently) — this is why the cron should post regularly.

### Xiaohongshu / Rednote — ⚠️ PAUSED (platform warning 2026-04-09)
**SKIP in cron runs.** Recipe kept for reference — see SKILL.md for full details.

### TikTok ⚠️ MOST FRAGILE
1. Navigate to `https://www.tiktok.com/tiktokstudio/upload?lang=en`
2. **Disable beforeunload IMMEDIATELY** before any other action — see pre-flight notes above.
3. File input is hidden — click via JS: `document.querySelector('input[type="file"][accept="video/*"]').click()` → upload.
4. After upload, two dialogs auto-appear:
   - "Turn on automatic content checks?" → click `Turn on`
   - "New editing features added" → click `Got it`
5. **DO NOT use `execCommand insertText` on the description editor.** TikTok uses Lexical, and execCommand triggers `NotFoundError: Failed to execute 'removeChild'` which crashes the form to "Something went wrong / Retry" and you lose the upload. Instead:
   1. Click into the description editor to focus.
   2. `playwright.keyboard.press('ControlOrMeta+a')` then `playwright.keyboard.press('Backspace')` to clear the auto-filled filename.
   3. Use `mcp__playwright__browser_type` (NOT slowly mode) to type the caption.
6. Click `Post`. Success: URL → `https://www.tiktok.com/tiktokstudio/content`.

### YouTube Shorts (for video content)
1. Navigate to `https://studio.youtube.com/`
2. Click the "Upload videos" / `+` icon top right.
3. Click "Select files" → upload. Vertical videos auto-detected as Shorts.
4. **Title and description**: faceplate-textarea-input web components — `execCommand insertText` works fine here (unlike TikTok). Use:
   ```js
   const t = document.querySelector('[aria-label="Add a title that describes your video (type @ to mention a channel)"]');
   t.focus(); document.execCommand('selectAll', false, null); document.execCommand('insertText', false, 'TITLE');
   ```
5. Click "No, it's not made for kids" radio (required).
6. Dismiss "Altered content" notification if it appears (click `Close`).
7. Click `Next` 3 times to advance Details → Video elements → Checks → Visibility tabs.
8. On Visibility tab: click `Public` radio → click `Publish`.
9. Success: dialog with URL `https://youtube.com/shorts/<id>`.

### YouTube (Community Post — for image+text only, no video)
1. Navigate to YouTube Studio → `https://studio.youtube.com/` → Community tab → Create post
2. Click the image icon to attach the hero image (download to `/tmp/yt-community-image.[ext]` first)
3. Type the caption in the text field
4. Click "Post"

### Reddit
1. Navigate to `https://www.reddit.com/r/[subreddit]/submit`
2. Select "Text" post type
3. Fill title and body from draft
4. Click "Post"
5. Do NOT add any images (Reddit prefers text posts for contractor content)

---

## STEP 4: Save to DB

After each successful platform post:
```sql
INSERT INTO social_media_posts (
  title_en, facebook_caption_en, instagram_caption_en,
  selected_image_urls,
  project_id, blog_post_id, status, published_at, notes, created_at, updated_at
) VALUES (
  $title, $facebook_text, $instagram_text,
  ARRAY[$image_url]::text[],
  $project_id, $blog_post_id,
  'published', NOW(),
  'Platforms: [list of platforms posted to]',
  NOW(), NOW()
);
```

---

## STEP 5: Log

Append to `/Users/renostars/reno-star-business-intelligent/data/cron-logs/social-media-posts.jsonl`:
```json
{"timestamp":"<ISO>","job":"social-media-poster","status":"success"|"error","phase":"draft"|"publish","platforms":["facebook","instagram"],"contentType":"project"|"blog","contentId":<id>,"contentSlug":"<slug>","summary":"<first 60 chars>","error":null}
```

---

## PHASE 6: Self-Improvement (every run, end of run)

**Goal:** the cron should get better with experience. If something unexpected happened during this run — a new platform quirk, a UI change, a failure mode that isn't already documented in the skill or memory — capture it so the next run doesn't repeat the mistake.

**Decision tree:**

1. **Did anything go wrong or surprise you during this run?** (uploaded fine but description didn't save? new dialog appeared? element selector changed? rate limit hit?)
   - **No** → skip Phase 6 entirely. Log a brief `phase6: clean run` line and stop.
   - **Yes** → continue.

2. **Is the issue ALREADY documented in the skill (`~/.claude/skills/social-media-post/SKILL.md`) or the memory (`~/.claude/projects/-Users-renostars/memory/feedback_social_media_platforms.md`)?**
   - **Yes** → it's a known issue, no update needed. Just log it.
   - **No** → continue.

3. **Is the issue a one-off / transient** (e.g. network blip, page took longer than usual to load, single broken upload that worked on retry)?
   - **Yes** → skip the skill update; just log it. Don't pollute the skill with noise.
   - **No, it's a real new pattern** → continue.

4. **Update the right file:**
   - **New element selector / new dialog / new UI flow** → use the `Edit` tool to add/update the relevant section in `~/.claude/skills/social-media-post/SKILL.md`. Add the new step in the platform recipe AND add a row to the "Failure modes worth memorizing" table at the bottom.
   - **New failure mode that needs deeper explanation** → add to `~/.claude/projects/-Users-renostars/memory/feedback_social_media_platforms.md` instead.
   - **New rate limit / pacing rule** → add to both: skill recipe + memory.
   - Keep edits **surgical** — don't rewrite sections, add/modify only the relevant lines.
   - At the top of any new entry, prefix with the date in `(YYYY-MM-DD)` so future readers can spot recent additions.

5. **Notify the user via Telegram** that the skill/memory was updated:
   ```
   📚 Skill update: <one line: what changed and why>
   File: <which file you edited>
   ```
   Use the `mcp__reno-stars-hub__telegram_send` MCP tool with `chat_id: -5219630660`.

6. **Log it** to the JSONL log:
   ```json
   {"timestamp":"<ISO>","job":"social-media-poster","phase":"phase6","action":"skill_updated"|"memory_updated"|"none","summary":"<60 char description>","filesChanged":["<path>"]}
   ```

**What NOT to do in Phase 6:**
- Don't update the skill for issues that are already documented (re-read the skill before editing).
- Don't add speculative "might be" entries — only document what you actually observed.
- Don't rewrite existing sections — additive edits only.
- Don't update the skill if you're not sure — better to log the observation in `data/social-media-observations.jsonl` for the user to triage manually:
  ```json
  {"timestamp":"<ISO>","platform":"<name>","observation":"<what you saw>","action_suggestion":"<what to do about it>"}
  ```
