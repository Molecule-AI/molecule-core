---
name: Modern File System Access API blocks puppeteer file uploads on Meta + LinkedIn
description: Meta Business Suite Composer and LinkedIn use window.showOpenFilePicker() instead of legacy <input type=file>. Puppeteer's waitForFileChooser and uploadFile cannot intercept these.
type: feedback
---

When automating file uploads via puppeteer-core / playwright on certain platforms, the upload button does NOT trigger a hidden `<input type="file">` — it calls `window.showOpenFilePicker()` from the modern File System Access API. Puppeteer's `waitForFileChooser` event ONLY fires for the legacy input flow, so the click goes through without any interception possible.

**Affected platforms (confirmed 2026-04-08):**
- **Meta Business Suite Composer** (`business.facebook.com/latest/composer`) — both photo + video upload
- **LinkedIn feed composer** (the "Start a post" → Video button)

**Workaround that works:**
- **Facebook**: skip Business Suite, use the legacy page composer at `facebook.com/profile.php?id=<page_id>` directly. The page profile composer DOES use `<input type="file">` (2 inputs in the DOM at all times — one for photos, one for video+image with `accept` containing `video/*`). Find the video-accepting one, call `inputElement.uploadFile(file)` directly — no need to click any button. The modal opens automatically once a file is set.
- **Instagram**: skip Business Suite, use `instagram.com` directly. Click the "New post" SVG (find via `svg[aria-label="New post"]`), then click the "Post" sub-menu option. After that the file input appears in the DOM and can be uploaded to.
- **LinkedIn**: NO known programmatic workaround. The LinkedIn web composer is fully File System Access API. Either use the LinkedIn API (requires OAuth + posting permissions) or hand the caption to the user for manual paste.

**Detection signal:**
When you see `dialog.querySelectorAll('input[type=file]').length === 0` after clicking a visible upload button, AND `window.showOpenFilePicker` is defined, that's the smoking gun. Don't waste time monkey-patching `createElement` or hooking `HTMLInputElement.prototype` — the page never creates a file input.

**Deeper hack (not yet tried):**
Override `window.showOpenFilePicker` BEFORE the click to return a synthetic `FileSystemFileHandle`. Requires constructing a fake handle that satisfies the page's expected interface (`getFile()`, `kind`, `name`). Risky — the page may sniff the handle's prototype chain. Save for a focused investigation; not worth doing inline during a normal post run.

**For social-media-post skill:** when posting video to Meta, ALWAYS use the legacy facebook.com page composer + a separate instagram.com upload. Do NOT route through Meta Business Suite. For LinkedIn video posts, drop the caption into Telegram for manual user action and continue with the other platforms.
