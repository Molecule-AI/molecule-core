IMPORTANT: Check Molecule-AI/internal repo for roadmap (PLAN.md), known issues, runbooks before starting work.

Hourly UX audit of the live Molecule AI canvas using the `browser-testing` skill.

Use the `/browser-test` skill (from the browser-automation plugin) to launch a real headless browser and interact with the canvas at `http://host.docker.internal:3000` like a human user.

## What to test each cycle (rotate — pick 2-3 per cycle, cover all within 4 cycles)

1. **Page load** — navigate, measure load time, screenshot initial state
2. **Workspace cards** — click cards, verify detail panel opens, check layout
3. **Create workspace flow** — open modal, fill fields, verify form validation
4. **Drag and drop** — drag workspace cards, verify position updates
5. **Side panel tabs** — click through Config/Logs/Memory tabs, verify content loads
6. **Keyboard navigation** — Tab through elements, Enter to activate, Escape to close
7. **Responsive layout** — test at 1920x1080, 1280x720, 768x1024
8. **Dark theme** — screenshot and check for hardcoded colors, low-contrast text

## How to use the skill

Write a Python script using Playwright (the skill handles setup):

```python
from playwright.sync_api import sync_playwright
import os
os.makedirs("/tmp/ux-audit", exist_ok=True)

with sync_playwright() as p:
    browser = p.chromium.launch(headless=True)
    page = browser.new_page(viewport={"width": 1280, "height": 720})
    page.goto("http://host.docker.internal:3000", timeout=15000)

    # ... interact, screenshot, evaluate ...

    browser.close()
```

## Output

For each issue: file ONE GitHub issue with `[uiux-agent]` tag, screenshot path, steps to reproduce, severity. Report issue numbers to Dev Lead.

If canvas unreachable or Playwright fails, fall back to code review of `canvas/src/components/`. Never produce empty output.
