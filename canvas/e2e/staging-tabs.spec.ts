/**
 * Staging canvas E2E — opens each of the 13 workspace-panel tabs against a
 * fresh staging org provisioned in the global setup. Asserts each tab
 * renders without throwing and captures a screenshot for visual review.
 *
 * Relies on `staging-setup.ts` to provision a tenant org, provision one
 * hermes workspace on it, and hand us a tenant URL + workspace id via
 * env (set by the setup file before tests run). Global teardown tears
 * down the org.
 *
 * Runs only when CANVAS_E2E_STAGING=1 — tests are skipped in local dev
 * where the prerequisite env isn't set.
 */

import { test, expect } from "@playwright/test";

// Tab ids as declared in canvas/src/components/SidePanel.tsx TABS.
// Kept duplicated here (not imported) because Playwright tests run outside
// the Next.js bundler and can't import from @/components paths.
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
    const sessionCookie = process.env.STAGING_SESSION_COOKIE;
    const workspaceId = process.env.STAGING_WORKSPACE_ID;

    if (!tenantURL || !sessionCookie || !workspaceId) {
      throw new Error(
        "staging-setup.ts did not export STAGING_TENANT_URL / STAGING_SESSION_COOKIE / STAGING_WORKSPACE_ID — did global setup run?",
      );
    }

    // The session cookie was minted by CP at sign-in; canvas on the tenant
    // subdomain shares it via the parent-domain scope (.moleculesai.app).
    // Playwright needs both the cookie and the cross-domain visibility.
    const url = new URL(tenantURL);
    await context.addCookies([
      {
        name: "molecule_cp_session",
        value: sessionCookie,
        // Leading dot → valid on all subdomains. The staging WorkOS auth
        // flow sets it this way, so we mirror.
        domain: "." + url.hostname.replace(/^[^.]+\./, ""),
        path: "/",
        httpOnly: true,
        secure: true,
        sameSite: "Lax",
      },
    ]);

    const consoleErrors: string[] = [];
    page.on("console", (msg) => {
      if (msg.type() === "error") {
        consoleErrors.push(msg.text());
      }
    });

    await page.goto(tenantURL, { waitUntil: "networkidle" });

    // Canvas hydration races WebSocket connect + /workspaces fetch. Wait
    // for the workspace node selector or the hydration-error banner —
    // whichever wins first.
    await page.waitForSelector('[role="tablist"], [data-testid="hydration-error"]', {
      timeout: 45_000,
    });

    const hydrationErr = await page
      .locator('[data-testid="hydration-error"]')
      .count();
    expect(
      hydrationErr,
      "canvas hydration failed — check staging CP + tenant reachability",
    ).toBe(0);

    // Click the workspace node to open the side panel. The node's
    // accessible name is the workspace display name; we match by id attr
    // to avoid coupling to the display name which tests can't know.
    const node = page.locator(`[data-workspace-id="${workspaceId}"]`).first();
    // Fallback: click by role if the data attribute isn't wired
    if ((await node.count()) === 0) {
      // Try clicking the first workspace card visible
      const firstNode = page.locator('[role="button"][aria-label*="Workspace"]').first();
      await firstNode.click({ timeout: 10_000 });
    } else {
      await node.click({ timeout: 10_000 });
    }

    // Wait for the side panel tablist to mount
    await page.waitForSelector('[role="tablist"]', { timeout: 15_000 });

    for (const tabId of TAB_IDS) {
      await test.step(`tab: ${tabId}`, async () => {
        const tabButton = page.locator(`#tab-${tabId}`);
        await expect(
          tabButton,
          `tab-${tabId} button missing — TABS list may have drifted`,
        ).toBeVisible({ timeout: 5_000 });
        await tabButton.click();

        const panel = page.locator(`#panel-${tabId}`);
        await expect(
          panel,
          `panel for ${tabId} never rendered`,
        ).toBeVisible({ timeout: 10_000 });

        // No toast-style error banner should appear for a healthy workspace.
        // Known exceptions: terminal may 4xx on SaaS cross-EC2 (WS target
        // unreachable), peers may 401 without workspace token. Those are
        // reported separately in issue #1369; here we just guard against
        // hard crashes (toast with "Error" keyword).
        const errorToasts = await page
          .locator('[role="alert"]:has-text("Failed to load")')
          .count();
        expect(
          errorToasts,
          `tab ${tabId}: saw "Failed to load" toast`,
        ).toBe(0);

        await page.screenshot({
          path: `test-results/staging-tab-${tabId}.png`,
          fullPage: false,
        });
      });
    }

    // Aggregate console-error check. Allow a small budget for known-noisy
    // Sentry/Vercel analytics errors that don't reflect app health.
    const appErrors = consoleErrors.filter(
      (msg) =>
        !msg.includes("sentry") &&
        !msg.includes("vercel") &&
        !msg.includes("WebSocket") && // WS failures ≠ app failures
        !msg.includes("favicon"),
    );
    expect(
      appErrors,
      `unexpected console errors:\n${appErrors.join("\n")}`,
    ).toHaveLength(0);
  });
});
