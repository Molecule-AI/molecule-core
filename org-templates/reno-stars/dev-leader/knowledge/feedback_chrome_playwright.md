---
name: Chrome/Playwright Usage Notes
description: Lessons learned about browser automation with Chrome CDP and Playwright
type: feedback
---

Chrome CDP (port 9222) gets overloaded after many Playwright connections in one session. Connections start timing out after ~5-6 uses.

**Why:** The debug port doesn't cleanly release connections. Accumulated sessions degrade performance.

**How to apply:**
- Kill Chrome between heavy Playwright sessions: `pkill -f "remote-debugging-port=9222"` then relaunch
- For Google Ads UI: Angular app requires stealth injector to bypass false "ad blocker detected"
- CDP keyboard input into Angular forms is unreliable — use Runtime.evaluate with native value setter + InputEvent dispatch
- Railway GraphQL API works via authenticated page context (cookie-based auth with `credentials: 'include'`)
