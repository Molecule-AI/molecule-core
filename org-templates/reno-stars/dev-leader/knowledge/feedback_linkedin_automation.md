---
name: LinkedIn Automation via Chrome CDP
description: What works and what doesn't for posting to LinkedIn via puppeteer-core CDP
type: feedback
---

## What Works (UPDATED 2026-04-08 — VIDEO UPLOAD ON COMPANY PAGE)

**The reliable path is to post FROM THE COMPANY ADMIN, not the personal feed:**

1. Connect via puppeteer-core CDP: `browserURL: 'http://127.0.0.1:9222'`
2. Navigate: `page.goto('https://www.linkedin.com/company/103326696/admin/dashboard/', { waitUntil: 'load' })`
3. Click the page's "Create" button (find via `button.innerText === 'Create'`)
4. In the popup menu, click the "Start a post" item — find via the text node walk:
   ```js
   const candidates = Array.from(document.querySelectorAll('*')).filter(e => {
     for (const node of e.childNodes) if (node.nodeType === 3 && /^Start a post$/i.test(node.textContent.trim())) return true;
     return false;
   });
   // Walk up to nearest A/BUTTON/[role=button] and click()
   ```
5. The COMPANY composer modal opens (header: "Reno Stars Construction Inc." + "Post to Anyone")
6. **Find the modal's "Add media" button** — `aria-label="Add media"`. As of 2026-04-08, this is a real `<button>` that triggers a LEGACY file picker.
   - **Critical:** filter to buttons INSIDE the modal. The personal-feed share box has stale "Add a video" buttons at viewport y~309 that are covered by the modal overlay — clicking them does nothing because the modal intercepts pointer events. The MODAL's media button is at roughly (505, 519) with `aria-label="Add media"`, NOT "Add a video".
7. Use `page.waitForFileChooser({ timeout: 8000 })` BEFORE clicking, then click via `page.mouse.click(x, y)`. The FileChooser event fires reliably; `chooser.accept([filePath])` sends the file.
8. Wait for the "Next" button to appear (upload progress completes — usually <30s for ~10MB video).
9. Click Next → caption step. Type caption into `.ql-editor` via `page.keyboard.type`.
10. Click the modal's "Post" button (`b.innerText.trim() === 'Post' && !b.disabled`).
11. Verify success: the page navigates to `/company/<id>/admin/page-posts/published/` AND a notification "Post successful. View post" appears in body text.

**Posts as the company page (Reno Stars Construction Inc.) — no account switching needed.**

## What Doesn't Work for Video Upload

- **Personal feed `https://www.linkedin.com/feed/` Video button** — the "Video" button at viewport (424, 164) on the personal feed uses `window.showOpenFilePicker()` (modern File System Access API). Puppeteer's `waitForFileChooser` does not fire. Even direct `osascript` clicks at the screen-coordinate equivalent (431, 426) don't trigger the OS file dialog reliably. **Skip the personal feed for video posts entirely** — use the company admin path above.
- **Meta Business Suite Composer for Facebook+Instagram** — same FSA API issue. Workaround documented in `feedback_file_system_access_api_blocks_upload.md`.

## What Doesn't Work for Other Things

## What Doesn't Work

- `document.querySelectorAll('[contenteditable], [role=textbox]')` — **returns 0 results**. The compose modal is rendered in shadow DOM / web components, not accessible via regular DOM queries.
- Switching to Reno Stars company account ("Posting as" dialog): clicking "Ryan Zhang ▼" opens Post Settings, the "▶" arrow to go to "Posting as" never responds to coordinate clicks. **Account switching is broken via automation.**
- Link preview removal via DOM: the dismiss button IS accessible (`button[aria-label*="dismiss"]`) but must be clicked BEFORE resizing viewport.
- `window.scrollBy()` — doesn't affect the fixed modal overlay. Use `page.mouse.wheel()` instead.
- `waitUntil: 'domcontentloaded'` — LinkedIn feed navigation times out. Use `waitUntil: 'load'` with `.catch(() => {})`.
- `waitUntil: 'networkidle2'` — always times out on LinkedIn.
- Keeping separate script connections between steps — the modal closes when puppeteer disconnects. **Do everything in one script.**

## Key Insight

The compose modal is a shadow DOM overlay. All interaction must be via:
- Mouse coordinates (page.mouse.click, page.mouse.wheel)
- Keyboard (page.keyboard.type, page.keyboard.press)
- Puppeteer never connects to disconnect between steps or modal closes

**Why:** LinkedIn uses React with complex shadow DOM components that aren't queryable via standard selectors.

## How to Apply

When the `social-media-poster` cron posts to LinkedIn, use this exact coordinate-based flow in a single puppeteer script without disconnecting mid-flow.
