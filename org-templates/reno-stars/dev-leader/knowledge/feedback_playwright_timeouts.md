---
name: Playwright operations need explicit short timeouts
description: User gets frustrated when playwright tools hang on slow pages — wrap operations in browser_run_code with short explicit timeouts instead of using the default 60s
type: feedback
---

The playwright-mcp tools (`browser_navigate`, `browser_click`, `browser_type`, etc.) default to a 60-second timeout per operation, and their JSON schemas don't expose a per-call timeout parameter. On slow or broken pages this means each interaction can hang for up to a minute, and a multi-step flow can stall the conversation for 5+ minutes before failing.

**Why:** The user said on 2026-04-07 "playwright always stuck for ever, can you set a timeout every time we use playwright or something?" — they were frustrated with multi-minute hangs during the Reddit work earlier the same day.

**How to apply:**
- For risky/slow sites (Reddit, Facebook, Instagram admin pages, anything that's been failing in this session), prefer `mcp__playwright__browser_run_code` and explicitly set a short timeout:
  ```js
  async (page) => {
    await page.goto(url, { timeout: 10000, waitUntil: 'domcontentloaded' });
    await page.locator('#foo').click({ timeout: 5000 });
  }
  ```
- 10s is plenty for navigation on any well-behaved site. 5s for clicks. If a step hits the timeout, fail fast and ask the user instead of grinding.
- For routine tools where a snappy site is expected (Gmail, X, Linear, etc.), the default tools are fine.
- If the user has already complained about a specific site being slow earlier in the conversation, ALWAYS use `browser_run_code` with a short timeout for that site for the rest of the session.
- When a destructive or one-shot action would be faster done by the user manually (e.g. clicking through a settings flow they know by heart), offer to hand it off rather than fight the browser.
