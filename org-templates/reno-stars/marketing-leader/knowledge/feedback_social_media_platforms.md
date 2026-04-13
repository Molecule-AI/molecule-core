---
name: Social media platform quirks and failure modes
description: Per-platform browser-automation quirks, exact element selectors, gotchas, and rate limits learned posting the Burnaby bathroom before/after video to all 8 Reno Stars accounts on 2026-04-07. Read this before any social media browser work.
type: feedback
---

This memory exists in addition to the `social-media-post` skill at `~/.claude/skills/social-media-post/SKILL.md`. The skill is the playbook (what to do); this memory is the failure-mode index (what NOT to do and why). Read the skill first; come here when you hit something unexpected.

## File path constraint
Playwright-mcp's `file_upload` tool refuses any path outside `/Users/renostars/`. Always copy upload targets to `/Users/renostars/<safe-name>.<ext>` first. Avoid spaces, Chinese characters, and ellipsis in filenames — some upload widgets choke on them. Confirmed broken on TikTok/Instagram with the original Dreamina filename `dreamina-2026-04-06-5364-首帧和尾帧是同一个地方同一个角度，这是装修前后的两个照片，我想要第一张照片里面的....mp4`.

## Caption rules (universal)
- **No promotional CTAs in organic posts or replies.** No "We do X at Reno Stars", no "feel free to reach out", no "happy to help". The account name on the post already attributes the brand. User explicitly flagged this on 2026-04-07 after seeing the first reply land with a CTA.
- **Xiaohongshu prohibits external links, phone numbers, and addresses.** Strip all of those for that platform; use Chinese only.
- **LinkedIn audience is B2B** — drop emoji, use a Challenge / Result framing.
- **X is hard-capped at 280 chars including the URL** (assume 23 chars for URLs via t.co wrapping).

## TikTok ⚠️ most fragile
1. **`document.execCommand('insertText', ...)` BREAKS the description editor** — TikTok uses Lexical, and execCommand triggers `NotFoundError: Failed to execute 'removeChild'` inside React's reconciler. The form crashes to "Something went wrong / Retry" and the upload is lost. **Use `playwright.keyboard` typing instead** — it dispatches React-friendly input events.
2. **Native `beforeunload` dialog blocks navigation mid-flow.** Disable preemptively: `window.onbeforeunload = null; window.addEventListener('beforeunload', e => e.stopImmediatePropagation(), true);`. If it fires, handle with `mcp__playwright__browser_handle_dialog accept=true`.
3. Two automatic dialogs appear after upload: "Turn on automatic content checks?" (click `Turn on`) and "New editing features added" (click `Got it`).
4. File input is hidden; click via `document.querySelector('input[type="file"][accept="video/*"]').click()`.
5. Success URL: `https://www.tiktok.com/tiktokstudio/content`.

## X / Twitter
1. The `Post` button is intercepted by an invisible overlay during normal `browser_click`. Click via JS instead: `document.querySelector('[data-testid="tweetButton"]').click()`.
2. Compose URL: `https://x.com/compose/post`. Add media button → file picker → upload → wait for `Uploaded (100%)` status.
3. Success URL: `https://x.com/home`.

## Instagram
1. New post flow: left-nav "New post" → submenu "Post" → modal with "Select from computer".
2. **"Video posts are now shared as reels" info dialog** appears after upload — click `OK`. Easy to miss in screenshots.
3. Three sequential screens after upload: Crop → Edit → Caption. Click `Next` twice to reach the caption screen.
4. After clicking `Share`, a "Sharing" spinner dialog stays for ~10 seconds — wait it out, don't assume it's hung.
5. Posting to Instagram does NOT auto-cross-post to Facebook even though the accounts are linked. Handle each separately unless using Meta Business Suite's create flow.

## Facebook (Page)
1. **For video, use the "Reel" button on the page composer, NOT "Photo/video".** The Photo/video flow uses a different upload path that fails on longer video content. Reel is the right primitive for any video upload.
2. Two `Next` clicks: first after upload, second after the auto-shown Edit screen, lands on the "Reel settings" form.
3. Description textbox is contenteditable; standard fill works.
4. Page URL: `https://www.facebook.com/profile.php?id=100068876523966` (Reno Stars).

## LinkedIn (company page)
1. Post via company admin: `https://www.linkedin.com/company/103326696/admin/`. Click `Create` → `Start a post`.
2. **Verify the composer header reads "Reno Stars Construction Inc."** — if it shows the user's personal profile (Ryan Zhang), the dropdown defaulted wrong; click the dropdown to switch.
3. Add media → upload → `Next` → text editor → `Post`.
4. Drop emoji, write a brief case study (Challenge / Result framing) — LinkedIn audience is B2B.
5. (Existing memory `feedback_linkedin_automation.md` covers the older personal-profile flow and shadow DOM issues — that's separate from the company page flow above.)

## YouTube Shorts
1. **`execCommand insertText` works fine here** (unlike TikTok). Use it for both title (`aria-label="Add a title that describes your video..."`) and description (`aria-label="Tell viewers about your video..."`). YouTube uses faceplate web components with shadow roots that handle execCommand correctly.
2. "Made for kids" radio is required — click "No, it's not made for kids".
3. May see an "Altered content" notification — click `Close`.
4. Click `Next` 3 times to advance Details → Video elements → Checks → Visibility tabs.
5. On Visibility tab: select `Public` radio → click `Publish`.
6. Success: dialog with "Video published" + URL pattern `https://youtube.com/shorts/<id>`.

## Xiaohongshu / Rednote
1. **Title is hard-capped at 20 characters (CJK-counted).** Error toast `标题最多输入20字哦~` if you exceed. Use the React-setter pattern to set value:
   ```js
   const t = document.querySelector('input[placeholder="填写标题会有更多赞哦"]');
   const setter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value').set;
   setter.call(t, 'TITLE');
   t.dispatchEvent(new Event('input', { bubbles: true }));
   ```
2. Body is contenteditable, max 1000 chars, supports `browser_type` / fill.
3. **NO external links, NO phone numbers, NO addresses, NO English URLs.** All copy must be Chinese. User explicitly said so on 2026-04-07.
4. Click 发布 (Publish) when enabled.
5. Success URL: `https://creator.xiaohongshu.com/publish/success...` and on-page text `发布成功`.
6. There's only one `<input type="file">` on the publish page — click it directly via JS to trigger the picker.

## Reddit ⚠️ paused until 2026-04-21
The `u/Anxious-Owl-9826` account was deleted on 2026-04-07 after a fresh-account shadow ban. **Skip Reddit entirely until then** — a launchd reminder fires April 21 9am. See also `feedback_reddit_new_account.md` and `feedback_reddit_rate_limit.md`.

When the new account exists:
- Wait 48h after creation before any posting.
- Comment-only week one. No links, no Reno Stars mention. Build organic karma + history.
- Username should look human (e.g. `RenoStarsVan`), NOT auto-generated.
- Even on a healthy account, space comment publishes 60–90 seconds apart. 4+ rapid comments triggers a 9–10 minute rate-limit cooldown.

**Reddit failure modes that mean "the account is the problem, not your code"** — don't waste time debugging:
- New Reddit submit: "Hmm, that community doesn't exist. Try checking the spelling." (when posting to your own profile)
- New Reddit settings save: "We had some issues saving your changes" + console "No profile ID for profile settings page"
- Old Reddit settings save: HTTP 500 from `/api/site_admin`
- Old Reddit submit: aggressive reCAPTCHA wall

## Telegram approval flow (cron context)
The `social-media-engage` and `social-media-poster` crons use Telegram for human approval. When you see a short ambiguous Telegram message ("reply all", "approve", "yes", "do it"):

**ALWAYS check `~/.openclaw/workspace/social/pending-replies.json` and `~/reno-star-business-intelligent/data/cron-logs/` BEFORE asking the user for clarification.** Telegram Bot API has no message history — the cron's outbound message lives only on disk in the logs. The user got frustrated on 2026-04-07 when I asked "reply to what?" instead of just checking the cron state. See also `feedback_telegram_cron_context.md`.

## Cleanup
After publishing, remove temp files:
```bash
rm -f /Users/renostars/burnaby-bathroom-before-after.mp4 /Users/renostars/reno-stars-avatar.png /Users/renostars/reno-stars-banner.png
```
