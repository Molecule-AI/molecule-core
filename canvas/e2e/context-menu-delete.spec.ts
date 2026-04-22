import { test, expect } from "@playwright/test";

/**
 * Playwright E2E for context-menu → delete confirm flow.
 * Regression test for the portal/race bug fixed in PR #1133:
 * clicking "Delete" in the context menu did nothing because the
 * portal-rendered ConfirmDialog was closed by the menu's outside-click
 * handler before onConfirm could fire.
 *
 * The fix hoists dialog state to the canvas store via `setPendingDelete`,
 * which survives ContextMenu unmount. This test exercises the full
 * interaction in a real browser environment.
 *
 * Requires: platform on :8080, canvas on :3000.
 */
const API = process.env.E2E_API_URL ?? "http://localhost:8080";

test.describe("Context Menu → Delete Confirm", () => {
  test.beforeEach(async ({ request }) => {
    // Ensure at least one workspace exists so the menu can be triggered
    const res = await request.get(`${API}/workspaces`);
    const workspaces = (await res.json()) as Array<{ id: string; name: string }>;
    if (workspaces.length === 0) {
      test.skip("No workspaces on canvas — cannot test context menu");
    }
  });

  test("Delete button opens ConfirmDialog and clicking Confirm deletes the workspace", async ({
    page,
    request,
  }) => {
    // 1. Create a workspace to delete (leaf node — no children, no cascade)
    const create = await request.post(`${API}/workspaces`, {
      data: { name: "E2E Delete Test", tier: 1, runtime: "claude-code" },
      headers: { "Content-Type": "application/json" },
    });
    const workspace = (await create.json()) as { id: string; name: string };
    const wsId = workspace.id;

    // Register so the node appears online on the canvas
    await request.post(`${API}/registry/register`, {
      data: {
        id: wsId,
        url: `http://localhost:9999`,
        agent_card: { name: "E2E Delete Test", skills: [] },
      },
      headers: { "Content-Type": "application/json" },
    });

    // 2. Open the canvas and wait for the workspace node
    await page.goto("/", { waitUntil: "networkidle" });
    await page.waitForTimeout(2000); // allow WS to appear

    // Find the workspace node on the canvas
    const node = page.locator(`.react-flow__node`).filter({ hasText: "E2E Delete Test" }).first();
    await expect(node).toBeVisible({ timeout: 10000 });

    // 3. Right-click to open context menu
    await node.click({ button: "right" });
    const menu = page.locator('[role="menu"]').first();
    await expect(menu).toBeVisible({ timeout: 3000 });
    await expect(menu).toHaveAttribute("aria-label", /E2E Delete Test/i);

    // 4. Click "Delete" — should open the ConfirmDialog (not close silently)
    const deleteBtn = menu.getByRole("menuitem").filter({ hasText: /Delete/i });
    await expect(deleteBtn).toBeVisible();
    await deleteBtn.click();

    // 5. ConfirmDialog should appear (portal renders into document.body)
    const dialog = page.locator('[role="dialog"]');
    await expect(dialog).toBeVisible({ timeout: 3000 });
    await expect(dialog).toContainText(/delete/i);
    await expect(dialog.getByRole("button", { name: /confirm|delete/i })).toBeVisible();

    // 6. Click Confirm — workspace should be deleted
    await dialog.getByRole("button", { name: /confirm|delete/i }).first().click();

    // 7. Dialog should close
    await expect(dialog).not.toBeVisible({ timeout: 3000 });

    // 8. Node should disappear from canvas
    await expect(
      page.locator(`.react-flow__node`).filter({ hasText: "E2E Delete Test" })
    ).not.toBeVisible({ timeout: 5000 });

    // 9. API confirms workspace is gone
    const getRes = await request.get(`${API}/workspaces/${wsId}`);
    expect(getRes.status()).toBeGreaterThanOrEqual(400); // 404 or similar
  });

  test("Cancel closes the dialog and the workspace remains", async ({ page, request }) => {
    const res = await request.get(`${API}/workspaces`);
    const workspaces = (await res.json()) as Array<{ id: string; name: string }>;
    if (workspaces.length === 0) {
      test.skip("No workspaces");
    }

    const ws = workspaces[0];

    // Register if not already
    await request.post(`${API}/registry/register`, {
      data: { id: ws.id, url: `http://localhost:9999`, agent_card: { name: ws.name, skills: [] } },
      headers: { "Content-Type": "application/json" },
    });

    await page.goto("/", { waitUntil: "networkidle" });
    await page.waitForTimeout(2000);

    const node = page.locator(`.react-flow__node`).filter({ hasText: ws.name }).first();
    await node.click({ button: "right" });

    const menu = page.locator('[role="menu"]').first();
    await expect(menu).toBeVisible();

    // Get workspace name before we click Delete (can't easily look it up after)
    const wsName = ws.name;

    await menu.getByRole("menuitem").filter({ hasText: /Delete/i }).click();
    const dialog = page.locator('[role="dialog"]');
    await expect(dialog).toBeVisible({ timeout: 3000 });

    // Cancel
    await dialog.getByRole("button", { name: /cancel/i }).first().click();
    await expect(dialog).not.toBeVisible({ timeout: 3000 });

    // Node still on canvas
    await expect(
      page.locator(`.react-flow__node`).filter({ hasText: wsName }).first()
    ).toBeVisible({ timeout: 5000 });
  });
});