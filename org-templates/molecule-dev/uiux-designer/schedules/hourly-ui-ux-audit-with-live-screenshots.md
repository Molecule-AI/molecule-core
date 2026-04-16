Hourly UX audit of the live Molecule AI canvas. Take real screenshots
and analyse actual user flows. The runtime discovered a working Chromium
path that bypasses the missing-libglib issue; use it rather than the
bundled `playwright install --with-deps` path (which fails in our sandbox).

1. SETUP BROWSER (proven-working recipe from Run 6, 2026-04-14):
   # Install @sparticuz/chromium + puppeteer-core via npm if not present
   # and reuse the NSS/NSPR libs bundled with Playwright's Firefox binary.
   cd /tmp && [ -d uiux-browser ] || (mkdir uiux-browser && cd uiux-browser && \
     npm init -y >/dev/null && npm install --quiet @sparticuz/chromium puppeteer-core 2>&1 | tail -3)
   # Ensure Playwright's firefox is present (ships libnss3.so, libnspr4.so)
   npx playwright install firefox 2>/dev/null || true
   FIREFOX_LIBS=$(ls -d /home/agent/.cache/ms-playwright/firefox-*/firefox 2>/dev/null | head -1)
   [ -z "$FIREFOX_LIBS" ] && FIREFOX_LIBS=$(ls -d /root/.cache/ms-playwright/firefox-*/firefox 2>/dev/null | head -1)

2. TAKE SCREENSHOTS against http://host.docker.internal:3000:
   Write a small puppeteer script capturing: home/empty state, create-workspace
   modal, full canvas, help dropdown, settings panel (open + detail), template
   palette, mobile 375px, responsive 1280px. Save to /tmp/ux-screenshots/.
   Invoke with:
      LD_LIBRARY_PATH="$FIREFOX_LIBS" node /tmp/uiux-browser/capture.cjs
   Then Read each PNG in /tmp/ux-screenshots/ to analyse with vision.
   If the browser still won't launch, fall back to curl+HTML and note it.

3. HTML / CSS ANALYSIS (always runs):
   - curl http://host.docker.internal:3000 — verify build ID / HTML size
   - Grep shipped JS chunks for 'window.alert|window.confirm|window.prompt'
     (should be 0 — ConfirmDialog replaces them)
   - cd /workspace/repo/canvas && grep-check: every .tsx using hooks has
     'use client' as its first line
   - Inspect any recently-changed .css / .tsx for light-theme regressions
     (hard zinc-900/950 bg mandate — no #fff, #f4f4f5 backgrounds)

4. USER-FLOW SANITY:
   - Workspace creation modal fields + submit path
   - Canvas node positioning and edges
   - Side-panel chat input and send
   - Toolbar tooltips
   - Responsive layout at 1280px

=== FINAL STEP — DELIVERABLE ROUTING (MANDATORY every cycle) ===

a. For each CRITICAL (broken flow, inaccessible control, theme regression):
   FILE A GITHUB ISSUE:
   - Dedupe: gh issue list --repo Molecule-AI/molecule-monorepo --search "ui OR ux OR theme" --state open
   - gh issue create --title "ui: <short>" --body with file:line, screenshot link (if available),
     expected vs actual, dark-theme rule cited.

b. delegate_task to PM with summary: build ID audited, screenshots count,
   violation counts by severity, new issue numbers, top 3 recommended
   improvements. PM routes to Frontend Engineer.

c. If clean: delegate_task to PM with "ui clean on build <X>" so the audit
   is observable.

d. Save to memory key 'uiux-audit-latest' as a secondary record only.
