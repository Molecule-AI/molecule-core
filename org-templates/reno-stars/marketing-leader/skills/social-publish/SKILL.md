# Skill: social-publish

Battle-tested helper scripts for publishing video posts to Reno Stars' social accounts. Each `.cjs` script under `scripts/` encapsulates hours of debugging against one platform's real DOM — Lexical editors, Next-button disambiguation, post-publish upsells, iframe scoping on Google Business Profile, and so on.

**Platforms covered:** Facebook (Reel), Instagram (Reel), X/Twitter, LinkedIn (company page), TikTok, YouTube (Shorts), Google Business Profile.

---

## HARD RULE — NEVER FREESTYLE PUPPETEER FOR SOCIAL POSTS

If you find yourself typing `puppeteer.connect`, `document.querySelector('div[role="dialog"]')`, Lexical editor queries, or "Next button" heuristics inside a social-posting task — **stop**. You are re-deriving, wrong, everything these helpers already solved.

Always invoke the helper:

```bash
node org-templates/reno-stars/marketing-leader/skills/social-publish/scripts/<platform>-publish.cjs <video-path> "<caption>"
```

(The helpers are also mirrored in `~/reno-star-business-intelligent/scripts/social-helpers/` on the host — use whichever path resolves in your workspace.)

If a helper fails (non-zero exit), read the exit code below first, screenshot at `/tmp/<platform>-fail.png`, and either:
1. fix the helper in THIS file and commit (so next run benefits),
2. or escalate to the operator via Telegram — **do not** silently fall back to hand-rolled puppeteer.

---

## Pre-flight (all platforms)

1. Chrome must be running with CDP exposed on `http://127.0.0.1:9222`:
   ```bash
   open -na "Google Chrome" --args --user-data-dir="/Users/renostars/.openclaw/chrome-profile" --remote-debugging-port=9222
   ```
2. Video path must be under `/Users/renostars/`, ASCII-only filename, no spaces / CJK / emoji.
3. The relevant platform must already be logged in inside that Chrome profile. (The helpers **connect** to the existing Chrome — they never launch a fresh Chromium, which is why "session expired" false positives disappear.)
4. Chrome window width ≥ 1200px for Facebook Reel (the composer hides the Post button at narrow widths).

---

## Helpers and exit codes

All helpers take `<video-path>` and `<caption>` as positional args. Exit `0` = success; `1` = fatal uncaught error.

### `fb-publish-reel.cjs` — Facebook Page Reel
- `0` composer closed, post committed (still feed-verify)
- `2` viewport <1200px wide
- `3` no on-screen Lexical caption box found
- `4` caption typing produced <50 chars (focus missed)
- `5` Post button not visible / composer never closed

### `ig-publish-reel.cjs` — Instagram Reel
- `0` Share clicked, sharing spinner done
- `3` no file input / no caption box
- `4` caption typing failed
- `5` no Share button

### `x-publish.cjs` — X / Twitter
- `0` posted (URL → x.com/home)
- `3` no file input / no composer
- `4` caption typing missed
- `5` post click intercepted

### `li-publish.cjs` — LinkedIn company page
- `0` posted
- `3` file chooser timeout
- `4` no `.ql-editor`
- `5` caption typing missed (<100 chars)
- `6` no Post button

### `tt-publish.cjs` — TikTok Studio
- `0` posted (URL → tiktokstudio/content)
- `3` no video input
- `4` no caption editor
- `5` caption typing missed (<50 chars)
- `6` Post button never enabled

### `yt-publish.cjs` — YouTube Shorts (studio.youtube.com)
- `0` Publish clicked
- `3` no file input
- `5` Publish button disabled

### `gbp-publish.cjs` — Google Business Profile "Add update"
- `0` Publish clicked
- `3` no GBP iframe found (not logged in as operator account?)

---

## Lessons baked in (do not re-learn)

- **Connect, never launch.** `puppeteer.connect({browserURL, defaultViewport: null})` reuses real Chrome sessions. `puppeteer.launch()` spawns fresh Chromium with no cookies — that is the "all sessions expired" false positive.
- **Facebook Lexical has 4–6 mirror DOM instances**, most off-screen. Pick the one with visible viewport rect, width > 200, and not the comment box.
- **Lexical rejects `execCommand` / clipboard paste.** Use `page.mouse.click(target)` to focus then `page.keyboard.type()` for real keystrokes.
- **Facebook Reel flow is Next → Next → Post**, not Next → Post. First Next advances Upload → Edit; second Next advances Edit → Reel settings; Post button only exists on Settings.
- **After Facebook Post**, Meta shows upsell modals ("Add WhatsApp button", "Boost", etc). Dismiss each or the next navigation triggers a `beforeunload` "Leave site?" dialog that blocks the script. Register `page.on('dialog', d => d.dismiss())` BEFORE clicking Post.
- **Verify success by composer disappearance**, not upsell modal appearance.
- **TikTok description editor is Lexical** — `execCommand insertText` throws `NotFoundError: Failed to execute 'removeChild'` and loses the upload. Click-to-focus + real `keyboard.type()` only.
- **X Post button is covered by an invisible overlay** for normal clicks. Use `document.querySelector('[data-testid="tweetButton"]').click()`.
- **GBP opens in an iframe** at `/local/business/<id>/promote/updates`. Scope every DOM query to that frame; the outer google.com page has a decoy "Add update" in the knowledge panel.
- **YouTube Studio title/description are faceplate-textarea web components** — `execCommand insertText` works fine here (unlike TikTok / FB). Don't over-generalize the Lexical rule.
- **LinkedIn company composer** needs header verification reading "Reno Stars Construction Inc." — if it shows the personal profile, switch accounts first or the post goes to the wrong place.
- **Instagram reels show a "Video posts are now shared as reels" info dialog** — click OK; it is not an error.

---

## When a helper actually breaks

1. Re-run once — transient CDP flake is common.
2. If it fails twice: read `/tmp/<platform>-fail.png`, identify the new DOM pattern, patch the helper in this directory, commit, and re-run.
3. Never replace the helper with a fresh hand-rolled puppeteer block. That path ends in re-discovering every lesson above.
