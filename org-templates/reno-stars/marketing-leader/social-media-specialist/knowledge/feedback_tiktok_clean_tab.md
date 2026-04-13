---
name: TikTok CAPTCHA avoidance — use clean tabs
description: TikTok CAPTCHA puzzle triggers when reusing browser tabs with accumulated state. Fresh tabs via Target.createTarget avoid it entirely.
type: feedback
---

When posting TikTok comments via Chrome CDP, always create a **fresh tab** using `Target.createTarget` from the browser-level debugger instead of reusing an existing tab.

**Why:** Reusing tabs that have browsed multiple pages accumulates tracking state that triggers TikTok's slider CAPTCHA ("Drag the slider to fit the puzzle"). Fresh tabs start with a clean slate and bypass it.

**How to apply:**
1. Connect to `ws://localhost:9222/devtools/browser/...` (browser endpoint, not page)
2. `Target.createTarget({url: 'about:blank'})` to get a new target ID
3. Find the new tab's page-level websocket from `/json`
4. Navigate to TikTok video URL from the clean tab
5. The comment input uses DraftEditor — click "Add comment..." text, then focus `.public-DraftEditor-content[contenteditable]`
6. Post button is `[data-e2e="comment-post"]` (not text "Post" — it's an arrow icon)

Also: TikTok's keyboard shortcuts overlay blocks the comments panel. Close it by finding the SVG close button inside the panel DOM (not by coordinates — the panel is in the right sidebar at x>1000).
