/**
 * Staging canvas E2E — opens each of the 13 workspace-panel tabs against a
 * fresh staging org provisioned in the global setup. Asserts each tab
 * renders without throwing and captures a screenshot for visual review.
 *
 * Auth model: the tenant platform's AdminAuth middleware accepts a bearer
 * token OR a WorkOS session cookie. Playwright can't mint a WorkOS
 * session, so we feed the per-tenant admin token (fetched in global
 * setup via GET /cp/admin/orgs/:slug/admin-token) as an Authorization:
 * Bearer header via context.setExtraHTTPHeaders(). Every browser
 * request inherits the header.
 *
 * Known SaaS gaps — documented in #1369 and allowed to render errored
 * content without failing the test (the gate is "no hard crash, no
 * 'Failed to load' toast"):
 *   - Files tab: empty (platform can't docker exec into a remote EC2)
 *   - Terminal tab: WS connect fails
 *   - Peers tab: 401 without workspace-scoped token
 */

import { test, expect } from "@playwright/test";

// Tab ids as declared in canvas/src/components/SidePanel.tsx TABS.
const TAB_IDS = [
  "chat",
  "activity",
  "details",
  "skills",
  "terminal",
  "config",
  "schedule",
  "channels",
  "files",
  "memory",
  "traces",
  "events",
  "audit",
] as const;

const STAGING = process.env.CANVAS_E2E_STAGING === "1";

test.skip(!STAGING, "CANVAS_E2E_STAGING not set — skipping staging-only tests");

test.describe("staging canvas tabs", () => {
  test("each workspace-panel tab renders without error", async ({
    page,
    context,
  }) => {
    const tenantURL = process.env.STAGING_TENANT_URL;
    const tenantToken = process.env.STAGING_TENANT_TOKEN;
    const workspaceId = process.env.STAGING_WORKSPACE_ID;

    if (!tenantURL || !tenantToken || !workspaceId) {
      throw new Error(
        "staging-setup.ts did not export STAGING_TENANT_URL / STAGING_TENANT_TOKEN / STAGING_WORKSPACE_ID — did global setup run?",
      );
    }

    // Attach the per-tenant admin bearer to every outbound request.
    // The tenant platform's AdminAuth middleware accepts this; no
    // WorkOS session needed.
    await context.setExtraHTTPHeaders({
      Authorization: `Bearer ${tenantToken}`,
    });

    // canvas/src/components/AuthGate.tsx fetches /cp/auth/me on mount
    // and redirects to the login page on 401. The bearer header above
    // is for platform API calls — it does NOT satisfy /cp/auth/me,
    // which is cookie-based (WorkOS session). Without this mock, the
    // canvas page mounts AuthGate, sees 401 from /cp/auth/me, and
    // redirects away from the tenant URL before the React Flow root
    // ever renders. The [aria-label] selector wait then times out.
    //
    // Intercept /cp/auth/me + return a fake Session shape so AuthGate
    // resolves to "authenticated" and renders {children}. The session
    // contents are cosmetic — the canvas only inspects org_id/user_id
    // in a few places that don't fail when these are dummy values.
    await context.route("**/cp/auth/me", (route) =>
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          user_id: `e2e-test-user-${workspaceId}`,
          org_id: "e2e-test-org",
          email: "e2e@test.local",
        }),
      }),
    );

    // Universal 401 → empty-200 fallback for any fetch.
    //
    // The narrow first pass (#2073, scoped to /workspaces/<id>/*) didn't
    // catch all the redirect triggers — SkillsTab.tsx alone fetches
    // /plugins and /plugins/sources outside the /workspaces/ tree, and
    // each of those 401s with the tenant admin bearer in SaaS mode.
    // canvas/src/lib/api.ts:62-74 calls `redirectToLogin` on ANY 401,
    // so a single non-workspace-scoped 401 yanks the page off the
    // tenant origin and breaks every locator that runs after.
    //
    // Broaden the route to ALL fetches: pass-through real responses,
    // swap 401s for 200 + empty body. Skip `/cp/auth/me` and the
    // tenant-origin HTML/JS bundle requests (resourceType !== fetch);
    // those are already handled or shouldn't be intercepted.
    //
    // For tab-render tests we don't need real data — the gate is
    // "panel mounts without crashing, no Failed-to-load toast". Body
    // shape is best-effort by URL: list endpoints (paths not ending
    // in a UUID-shaped segment) get `[]`; single-resource endpoints
    // get `{}`. Both are valid JSON; well-written panels render an
    // empty state for either rather than throwing.
    await context.route("**", async (route, request) => {
      if (request.resourceType() !== "fetch") {
        return route.fallback();
      }
      // /cp/auth/me is mocked above with a fixed Session shape — let
      // that handler win without us round-tripping the network.
      if (request.url().includes("/cp/auth/me")) {
        return route.fallback();
      }
      let resp;
      try {
        resp = await route.fetch();
      } catch {
        return route.fallback();
      }
      if (resp.status() !== 401) {
        return route.fulfill({ response: resp });
      }
      const lastSeg =
        new URL(request.url()).pathname.split("/").filter(Boolean).pop() || "";
      const looksLikeList = !/^[0-9a-f-]{8,}$/.test(lastSeg);
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: looksLikeList ? "[]" : "{}",
      });
    });

    const consoleErrors: string[] = [];
    page.on("console", (msg) => {
      if (msg.type() === "error") {
        consoleErrors.push(msg.text());
      }
    });

    // Capture the URL of any failed network request so a "Failed to load
    // resource: 404" console message we filter out below leaves a
    // breadcrumb. Browser console messages for resource-load failures
    // omit the URL, so we'd otherwise be flying blind. Logged to the
    // test's stdout (visible in the workflow log under the failed step).
    page.on("requestfailed", (req) => {
      console.log(`[e2e/requestfailed] ${req.method()} ${req.url()}: ${req.failure()?.errorText ?? "?"}`);
    });
    page.on("response", (res) => {
      if (res.status() >= 400) {
        console.log(`[e2e/response-${res.status()}] ${res.request().method()} ${res.url()}`);
      }
    });

    // waitUntil="networkidle" is wrong here — the canvas keeps a
    // WebSocket open + polls /events and /workspaces every few
    // seconds, so the network is *never* idle for 500ms. page.goto
    // would hang until its 45s default timeout. "domcontentloaded"
    // returns as soon as the HTML is parsed; React hydration + the
    // selector wait below is what actually gates ready-for-interaction.
    await page.goto(tenantURL, { waitUntil: "domcontentloaded" });

    // Canvas hydration races WebSocket connect + /workspaces fetch.
    // Wait for the React Flow canvas wrapper (always present once
    // hydrated, even with zero workspaces) or the hydration-error
    // banner — whichever wins first. Previous version of this wait
    // used `[role="tablist"]`, but that selector only appears AFTER
    // a workspace node is clicked (which happens below at L100), so
    // the wait would always time out at 45s before any meaningful
    // failure surfaced.
    await page.waitForSelector(
      '[aria-label="Molecule AI workspace canvas"], [data-testid="hydration-error"]',
      { timeout: 45_000 },
    );

    const hydrationErr = await page
      .locator('[data-testid="hydration-error"]')
      .count();
    expect(
      hydrationErr,
      "canvas hydration failed — check staging CP + tenant reachability",
    ).toBe(0);

    // Click the workspace node to open the side panel. Try a data
    // attribute first, fall back to a generic role-based selector so
    // the test doesn't break when the node-card markup changes.
    const byDataAttr = page.locator(`[data-workspace-id="${workspaceId}"]`).first();
    if ((await byDataAttr.count()) > 0) {
      await byDataAttr.click({ timeout: 10_000 });
    } else {
      const firstNode = page
        .locator('[role="button"][aria-label*="Workspace" i]')
        .first();
      await firstNode.click({ timeout: 10_000 });
    }

    await page.waitForSelector('[role="tablist"]', { timeout: 15_000 });

    for (const tabId of TAB_IDS) {
      await test.step(`tab: ${tabId}`, async () => {
        const tabButton = page.locator(`#tab-${tabId}`);
        // The TABS bar is `overflow-x-auto` (SidePanel.tsx:~tabs
        // wrapper) — tabs after position ~3 are clipped behind the
        // right-edge fade gradient on smaller viewports. Playwright's
        // `toBeVisible()` returns false for clipped elements, so a
        // bare visibility check fails on `skills` and later tabs in
        // CI. scrollIntoViewIfNeeded brings the button into view
        // before the visibility check, mirroring what SidePanel's own
        // keyboard handler does on arrow-key navigation.
        await tabButton.scrollIntoViewIfNeeded({ timeout: 5_000 });
        await expect(
          tabButton,
          `tab-${tabId} button missing — TABS list may have drifted`,
        ).toBeVisible({ timeout: 5_000 });
        await tabButton.click();

        const panel = page.locator(`#panel-${tabId}`);
        await expect(panel, `panel for ${tabId} never rendered`).toBeVisible({
          timeout: 10_000,
        });

        // "Failed to load" toast = hard crash. Known SaaS-mode gaps
        // (Files empty, Terminal disconnected, Peers 401) surface as
        // in-panel content, not toasts.
        const errorToasts = await page
          .locator('[role="alert"]:has-text("Failed to load")')
          .count();
        expect(errorToasts, `tab ${tabId}: "Failed to load" toast`).toBe(0);

        await page.screenshot({
          path: `test-results/staging-tab-${tabId}.png`,
          fullPage: false,
        });
      });
    }

    // Aggregate console-error budget. Known-noisy sources whitelisted:
    // Sentry, Vercel analytics, WS reconnects (expected on SaaS
    // terminal), favicon 404 (cosmetic), and the browser's generic
    // "Failed to load resource: ... 404" message which never includes
    // the URL — uninformative on its own and impossible to filter
    // meaningfully without a URL. The page.on('requestfailed') +
    // page.on('response>=400') logging above captures the actual URLs
    // so a real bug still leaves a breadcrumb in the workflow log;
    // a real exception (panel crash, JS error) surfaces as a typed
    // error with file path which the filter still catches.
    const appErrors = consoleErrors.filter(
      (msg) =>
        !msg.includes("sentry") &&
        !msg.includes("vercel") &&
        !msg.includes("WebSocket") &&
        !msg.includes("favicon") &&
        !msg.includes("molecule-icon.png") && // cosmetic 404
        !msg.includes("Failed to load resource"),
    );
    expect(
      appErrors,
      `unexpected console errors:\n${appErrors.join("\n")}`,
    ).toHaveLength(0);
  });
});
