# Browser Automation Rules

- Chrome CDP is available at `host.docker.internal:9223` (proxy to host Chrome on port 9222)
- Always use `browserWSEndpoint` with URL rewrite (`localhost:9222` → `host.docker.internal:9223`)
- Never use `browserURL` — it resolves to an unreachable localhost address
- Never call `browser.close()` — use `browser.disconnect()` to release without killing Chrome
- Set `NODE_PATH=/usr/lib/node_modules` if `require('puppeteer-core')` fails
- The Chrome profile is shared — all agents see the same logged-in sessions
