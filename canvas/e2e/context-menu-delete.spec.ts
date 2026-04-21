/**
 * Playwright E2E: context-menu → delete confirm flow (issue #1138)
 *
 * Exercises the delete dialog portal interaction that caused the race
 * fixed in PR #1133. Previously, the portal-rendered Confirm button
 * clicked counted as "outside" by the menu's outside-click handler,
 * which killed the dialog before onConfirm could fire.
 *
 * Test flow:
 * 1. Create a test workspace (seed fixture via API)
 * 2. Register it so the node appears on the canvas
 * 3. Right-click the workspace node → context menu opens
 * 4. Click Delete → ConfirmDialog appears
 * 5. Click Confirm → dialog closes + node disappears + API DELETE fires
 * 6. (Bonus) Cancel path — dialog closes, node stays
 *
 * Run: npx playwright test e2e/context-menu-delete.spec.ts
 */
import { test, expect, type Page, type APIRequestContext } from "@playwright/test";

const API_BASE = process.env.E2E_API_URL ?? "http://localhost:8080";
const CANVAS_BASE = process.env.E2E_CANVAS_URL ?? "http://localhost:3000";

interface Workspace {
  id: string;
  name: string;
}

/** Create and register a temporary workspace for testing. */
async function createTestWorkspace(
  request: APIRequestContext,
  name: string
): Promise<Workspace> {
  const created = await request
    .post(`${API_BASE}/workspaces`, {
      data: { name, tier: 1, runtime: "langgraph" },
      headers: { "Content-Type": "application/json" },
    })
    .then((r) => r.json() as Promise<{ id: string; name: string }>);

  await request.post(`${API_BASE}/registry/register`, {
    data: {
      id: created.id,
      url: "http://localhost:9999",
      agent_card: { name: "DeleteTest", skills: [] },
    },
    headers: { "Content-Type": "application/json" },
  });

  return created;
}

/** Delete a workspace (for test cleanup). */
async function deleteWorkspace(
  request: APIRequestContext,
  workspaceId: string
): Promise<void> {
  await request.delete(`${API_BASE}/workspaces/${workspaceId}?confirm=true`);
}

/** Wait for a canvas workspace node with the given name to appear. */
async function waitForCanvasNode(
  page: Page,
  nodeName: string,
  timeout = 10_000
): Promise<void> {
  await page.goto(CANVAS_BASE);
  await page.waitForLoadState("networkidle");
  // Dismiss any onboarding overlay
  const skipBtn = page.getByText(/skip guide/i).first();
  if (await skipBtn.isVisible().catch(() => false)) {
    await skipBtn.click();
    await page.waitForTimeout(500);
  }
  const node = page.getByText(nodeName).first();
  await node.waitFor({ timeout });
}

/** Right-click a canvas node to open the context menu. */
async function openContextMenu(page: Page, nodeName: string): Promise<void> {
  const node = page.getByText(nodeName).first();
  await node.click({ button: "right" });
  await page.waitForTimeout(300); // animation delay
}

test.describe("Context menu → Delete confirm flow", () => {
  let testWs: Workspace;

  test.beforeAll(async ({ request }) => {
    testWs = await createTestWorkspace(request, "DeleteConfirmE2E");
  });

  test.afterAll(async ({ request }) => {
    await deleteWorkspace(request, testWs.id);
  });

  test.beforeEach(async ({ page }) => {
    await waitForCanvasNode(page, testWs.name);
  });

  // ── Happy path: Delete → Confirm → node removed ──────────────────────────

  test("right-click → Delete → Confirm → dialog closes and node disappears", async ({
    page,
    request,
  }) => {
    await openContextMenu(page, testWs.name);

    // Context menu should be visible
    const menu = page.locator('[role="menu"]');
    await expect(menu).toBeVisible({ timeout: 5_000 });

    // Click Delete
    const deleteBtn = page.getByRole("menuitem", { name: /delete/i }).first();
    await deleteBtn.click();
    await page.waitForTimeout(300);

    // ConfirmDialog appears (role=dialog via createPortal)
    const dialog = page.getByRole("dialog", { name: /delete/i });
    await expect(dialog).toBeVisible({ timeout: 5_000 });

    // Capture API call — watch for DELETE /workspaces/:id
    const [deleteRequest] = await Promise.all([
      page.waitForRequest(
        (req) =>
          req.method() === "DELETE" &&
          req.url().includes(`/workspaces/${testWs.id}`),
        { timeout: 10_000 }
      ),
      // Click Confirm (confirmVariant="danger" → "Delete" label in cascade flow,
      // or generic "Confirm" in leaf-node flow)
      dialog.getByRole("button", { name: /^(confirm|delete)$/i }).click(),
    ]);

    // API DELETE fired with ?confirm=true
    expect(deleteRequest.url()).toContain("confirm=true");

    // Dialog closes (removed from DOM via !open)
    await expect(page.getByRole("dialog", { name: /delete/i })).not.toBeVisible();

    // Node disappears from canvas
    const node = page.getByText(testWs.name).first();
    await expect(node).not.toBeVisible({ timeout: 5_000 });
  });

  // ── Cancel path: Delete → Cancel → node remains ──────────────────────────

  test("right-click → Delete → Cancel → dialog closes and node remains", async ({
    page,
    request,
  }) => {
    // Re-register the workspace since the previous test deleted it
    await request.post(`${API_BASE}/registry/register`, {
      data: {
        id: testWs.id,
        url: "http://localhost:9999",
        agent_card: { name: "DeleteTest", skills: [] },
      },
      headers: { "Content-Type": "application/json" },
    });
    await page.reload();
    await waitForCanvasNode(page, testWs.name);
    await openContextMenu(page, testWs.name);

    // Open Delete
    const deleteBtn = page.getByRole("menuitem", { name: /delete/i }).first();
    await deleteBtn.click();
    await page.waitForTimeout(300);

    const dialog = page.getByRole("dialog", { name: /delete/i });
    await expect(dialog).toBeVisible({ timeout: 5_000 });

    // Click Cancel
    dialog.getByRole("button", { name: /^cancel$/i }).click();
    await page.waitForTimeout(300);

    // Dialog closes
    await expect(page.getByRole("dialog", { name: /delete/i })).not.toBeVisible();

    // Node is still on the canvas
    const node = page.getByText(testWs.name).first();
    await expect(node).toBeVisible({ timeout: 3_000 });
  });

  // ── Outside-click closes context menu (no dialog opened) ─────────────────

  test("clicking outside context menu closes it without opening dialog", async ({
    page,
  }) => {
    await openContextMenu(page, testWs.name);

    const menu = page.locator('[role="menu"]');
    await expect(menu).toBeVisible({ timeout: 5_000 });

    // Click the canvas background (outside the menu)
    await page.click("body", { position: { x: 10, y: 10 } });
    await page.waitForTimeout(300);

    // Menu closed, no dialog opened
    await expect(menu).not.toBeVisible();
    await expect(page.getByRole("dialog", { name: /delete/i })).not.toBeVisible();
  });
});