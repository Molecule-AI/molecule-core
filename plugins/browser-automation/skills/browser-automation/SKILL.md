---
id: browser-automation
name: browser-automation
description: Connect to Chrome via CDP proxy to automate web interactions — posting, scraping, form filling. Uses puppeteer-core (no bundled Chromium).
tags: [browser, puppeteer, cdp]
---

# Browser Automation via Chrome CDP

Connect to the host Chrome browser via the CDP proxy to automate web interactions.

## Connection — ALWAYS use the helper

**DO NOT call `puppeteer.connect()` directly.** Use `./lib/connect.js`:

```javascript
const { connect } = require('/configs/plugins/browser-automation/skills/browser-automation/lib/connect');
const browser = await connect();
const page = (await browser.pages())[0];
// ... do work ...
await browser.disconnect();  // NEVER browser.close() (kills shared Chrome)
```

The helper enforces two settings that broke social-media automation repeatedly on 2026-04-15:

1. **`defaultViewport: null`** — use real Chrome window dims (NOT puppeteer's 800×600 default).
2. **Host auto-detection** — Docker (`host.docker.internal:9223`) vs host script (`127.0.0.1:9222`).

If you absolutely cannot use the helper (one-off debug, no plugin path), the rule is still inviolable — paste this verbatim:

```javascript
const browser = await puppeteer.connect({
  browserURL: 'http://127.0.0.1:9222',  // or browserWSEndpoint with proxy host
  defaultViewport: null,                 // ← MANDATORY, NEVER omit
});
```

**Why `defaultViewport: null` is non-negotiable:** without it, puppeteer overrides Chrome's reported size to 800×600. The browser visually still renders at the user's actual size, but `window.innerWidth/Height` returns `800/600`. All click coords, on-screen filters, and `getBoundingClientRect()` checks become wrong. Symptoms: agent reports "session expired" / "button not found" / "caption typed nowhere" — but visually everything looks fine to the user. This was the root of the 2026-04-15 social-media-poster runs that bailed claiming all sessions were expired (~3h debug).

## Key Patterns

- **Tab listing:** `http://host.docker.internal:9223/json`
- **Navigate:** `await page.goto(url, {waitUntil: 'networkidle2'})`
- **Disconnect (don't close):** `browser.disconnect()` — never `browser.close()` (that kills the shared Chrome)

## Host setup (one-time, per machine)

The plugin ships a **host bridge** at `plugins/browser-automation/host-bridge/`
that keeps a CDP proxy alive on the user's machine so any container with this
plugin can reach their Chrome. Install once — it survives reboots:

```bash
# from the molecule-monorepo repo root:
bash plugins/browser-automation/host-bridge/install-host-bridge.sh
```

This registers a launchd agent (macOS) or systemd user unit (Linux) that runs
`cdp-proxy.cjs` on `0.0.0.0:9223` forever. Then launch Chrome with the debug
port (once per reboot is enough; the proxy reconnects):

```bash
open -na "Google Chrome" --args --remote-debugging-port=9222 \
  --user-data-dir="$HOME/.chrome-molecule" --profile-directory=Default
```

Verify: `curl http://127.0.0.1:9223/json/version` returns JSON. If it doesn't,
the proxy is running but Chrome isn't — launch Chrome and re-check. No
workspace-side changes needed — `lib/connect.js` already points at
`host.docker.internal:9223`.

To uninstall: `bash plugins/browser-automation/host-bridge/install-host-bridge.sh uninstall`.

## Available Accounts

The Chrome profile has active sessions for:
- YouTube, Instagram, Facebook, X/Twitter, LinkedIn, TikTok
- Gmail, InvoiceSimple, Google Search Console
- Manta, TrustedPros, Foursquare, Pinterest, Medium
