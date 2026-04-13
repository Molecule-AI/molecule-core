import { test, expect } from "@playwright/test";

const API = process.env.E2E_API_URL ?? "http://localhost:8080";

test.describe("Org template import (PLAN.md §20.3)", () => {
  test("org templates section renders inside the palette", async ({ page }) => {
    // Sanity: platform must serve /org/templates
    const res = await page.request.get(`${API}/org/templates`);
    expect(res.ok()).toBeTruthy();
    const orgs: { dir: string; name: string; workspaces: number }[] = await res.json();
    test.skip(orgs.length === 0, "No org templates configured");

    await page.goto("/", { waitUntil: "networkidle" });

    // The Org Templates section lives in TWO places: inside the EmptyState
    // (visible only when there are 0 workspaces) and inside the
    // TemplatePalette sidebar. Open the palette so the section is reachable
    // regardless of workspace count.
    const paletteToggle = page.getByTitle("Template Palette");
    if (await paletteToggle.isVisible()) {
      await paletteToggle.click({ force: true });
    }

    const section = page.getByTestId("org-templates-section").first();
    await expect(section).toBeVisible({ timeout: 15000 });
    await expect(section.getByText("Org Templates")).toBeVisible();

    // Wait for the API fetch to populate (auto-waits via toBeVisible)
    const first = orgs[0];
    const label = first.name || first.dir;
    await expect(section.getByText(label, { exact: false })).toBeVisible({ timeout: 15000 });
    await expect(section.getByText(`${first.workspaces}w`)).toBeVisible();
    await expect(section.getByRole("button", { name: /Import org/i }).first()).toBeVisible();
  });

  test("import button exists for every org template returned by the API", async ({ page }) => {
    const res = await page.request.get(`${API}/org/templates`);
    const orgs: { dir: string }[] = await res.json();
    test.skip(orgs.length === 0, "No org templates configured");

    await page.goto("/", { waitUntil: "networkidle" });
    const paletteToggle = page.getByTitle("Template Palette");
    if (await paletteToggle.isVisible()) {
      await paletteToggle.click({ force: true });
    }
    const section = page.getByTestId("org-templates-section").first();
    await expect(section).toBeVisible({ timeout: 15000 });
    // Wait for the API result to render (one Import button per org)
    const buttons = section.getByRole("button", { name: /Import org/i });
    await expect(buttons).toHaveCount(orgs.length, { timeout: 15000 });
  });
});
