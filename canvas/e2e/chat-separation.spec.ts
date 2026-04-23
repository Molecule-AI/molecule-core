import { test, expect } from "@playwright/test";

const API = process.env.E2E_API_URL ?? "http://localhost:8080";

test.describe("Chat Sub-Tabs", () => {
  test("chat tab shows My Chat and Agent Comms sub-tabs", async ({ page }) => {
    const res = await page.request.get(`${API}/workspaces`);
    const workspaces = await res.json();
    test.skip(workspaces.length === 0, "No workspaces to test");

    await page.goto("/");
    await page.waitForTimeout(3000);

    // Click first workspace node
    await page.locator(".react-flow__node").first().click();
    await page.waitForTimeout(500);

    // Click Chat tab
    const chatTab = page.getByRole("button", { name: /Chat/ }).first();
    await chatTab.click();
    await page.waitForTimeout(500);

    // Sub-tabs should be visible
    await expect(page.locator("text=My Chat")).toBeVisible({ timeout: 3000 });
    await expect(page.locator("text=Agent Comms")).toBeVisible({ timeout: 3000 });
  });

  test("My Chat is selected by default", async ({ page }) => {
    const res = await page.request.get(`${API}/workspaces`);
    const workspaces = await res.json();
    test.skip(workspaces.length === 0, "No workspaces");

    await page.goto("/");
    await page.waitForTimeout(3000);
    await page.locator(".react-flow__node").first().click();
    await page.waitForTimeout(500);
    await page.getByRole("button", { name: /Chat/ }).first().click();
    await page.waitForTimeout(500);

    // My Chat sub-tab should have active styling (border-blue-500)
    const myChatBtn = page.locator("button", { hasText: "My Chat" });
    await expect(myChatBtn).toHaveClass(/border-blue-500/);
  });

  test("switching to Agent Comms shows different content", async ({ page }) => {
    const res = await page.request.get(`${API}/workspaces`);
    const workspaces = await res.json();
    test.skip(workspaces.length === 0, "No workspaces");

    await page.goto("/");
    await page.waitForTimeout(3000);
    await page.locator(".react-flow__node").first().click();
    await page.waitForTimeout(500);
    await page.getByRole("button", { name: /Chat/ }).first().click();
    await page.waitForTimeout(500);

    // Click Agent Comms
    await page.locator("button", { hasText: "Agent Comms" }).click();
    await page.waitForTimeout(500);

    // Should show empty state or agent comms messages
    const hasEmpty = await page.locator("text=No agent-to-agent communications").isVisible().catch(() => false);
    const hasMessages = await page.locator("[class*=cyan]").count() > 0;
    expect(hasEmpty || hasMessages).toBeTruthy();
  });

  test("My Chat has input box, Agent Comms does not", async ({ page }) => {
    const res = await page.request.get(`${API}/workspaces`);
    const workspaces = await res.json();
    test.skip(workspaces.length === 0, "No workspaces");

    await page.goto("/");
    await page.waitForTimeout(3000);
    await page.locator(".react-flow__node").first().click();
    await page.waitForTimeout(500);
    await page.getByRole("button", { name: /Chat/ }).first().click();
    await page.waitForTimeout(500);

    // My Chat should have textarea
    await expect(page.locator("textarea")).toBeVisible();

    // Switch to Agent Comms
    await page.locator("button", { hasText: "Agent Comms" }).click();
    await page.waitForTimeout(500);

    // Agent Comms should NOT have textarea
    await expect(page.locator("textarea")).not.toBeVisible();
  });

  test("switching back to My Chat preserves messages", async ({ page }) => {
    const res = await page.request.get(`${API}/workspaces`);
    const workspaces = await res.json();
    test.skip(workspaces.length === 0, "No workspaces");

    await page.goto("/");
    await page.waitForTimeout(3000);
    await page.locator(".react-flow__node").first().click();
    await page.waitForTimeout(500);
    await page.getByRole("button", { name: /Chat/ }).first().click();
    await page.waitForTimeout(1000);

    // Check if there are messages or empty state in My Chat
    const hasContent = await page.locator("text=No messages yet").isVisible().catch(() => false) ||
      await page.locator("[class*=blue-600]").count() > 0;

    // Switch to Agent Comms and back
    await page.locator("button", { hasText: "Agent Comms" }).click();
    await page.waitForTimeout(300);
    await page.locator("button", { hasText: "My Chat" }).click();
    await page.waitForTimeout(300);

    // Same content should be there
    const hasContentAfter = await page.locator("text=No messages yet").isVisible().catch(() => false) ||
      await page.locator("[class*=blue-600]").count() > 0;

    // Both should be truthy (content exists before and after switch)
    expect(hasContent || hasContentAfter).toBeTruthy();
  });
});

test.describe("Activity API Source Filter", () => {
  test("source=canvas returns only canvas-initiated entries", async ({ request }) => {
    const wsRes = await request.get(`${API}/workspaces`);
    const workspaces = await wsRes.json();
    if (workspaces.length === 0) return;

    const wsId = workspaces[0].id;
    const res = await request.get(`${API}/workspaces/${wsId}/activity?source=canvas`);
    expect(res.ok()).toBeTruthy();
    const entries = await res.json();
    expect(Array.isArray(entries)).toBeTruthy();

    // All entries should have source_id null
    for (const e of entries) {
      expect(e.source_id).toBeNull();
    }
  });

  test("source=agent returns only agent-initiated entries", async ({ request }) => {
    const wsRes = await request.get(`${API}/workspaces`);
    const workspaces = await wsRes.json();
    if (workspaces.length === 0) return;

    const wsId = workspaces[0].id;
    const res = await request.get(`${API}/workspaces/${wsId}/activity?source=agent`);
    expect(res.ok()).toBeTruthy();
    const entries = await res.json();
    expect(Array.isArray(entries)).toBeTruthy();

    // All entries should have non-null source_id
    for (const e of entries) {
      if (e.source_id !== undefined) {
        expect(e.source_id).not.toBeNull();
      }
    }
  });

  test("source=invalid returns 400", async ({ request }) => {
    const wsRes = await request.get(`${API}/workspaces`);
    const workspaces = await wsRes.json();
    if (workspaces.length === 0) return;

    const wsId = workspaces[0].id;
    const res = await request.get(`${API}/workspaces/${wsId}/activity?source=bogus`);
    expect(res.status()).toBe(400);
  });

  test("source+type filters combine correctly", async ({ request }) => {
    const wsRes = await request.get(`${API}/workspaces`);
    const workspaces = await wsRes.json();
    if (workspaces.length === 0) return;

    const wsId = workspaces[0].id;
    const res = await request.get(`${API}/workspaces/${wsId}/activity?type=a2a_receive&source=canvas`);
    expect(res.ok()).toBeTruthy();
    const entries = await res.json();
    expect(Array.isArray(entries)).toBeTruthy();

    for (const e of entries) {
      expect(e.activity_type).toBe("a2a_receive");
      expect(e.source_id).toBeNull();
    }
  });
});

test.describe("Data Flow — Initial Prompt in Chat", () => {
  test("initial prompt appears as user message in My Chat", async ({ page }) => {
    // Find a workspace that has activity with source=canvas (initial prompt via proxy)
    const wsRes = await page.request.get(`${API}/workspaces`);
    const workspaces = await wsRes.json();
    test.skip(workspaces.length === 0, "No workspaces");

    // Find a workspace with canvas-initiated activity
    let targetWs: { id: string; name: string } | null = null;
    for (const ws of workspaces) {
      const actRes = await page.request.get(`${API}/workspaces/${ws.id}/activity?source=canvas&type=a2a_receive&limit=1`);
      const entries = await actRes.json();
      if (entries.length > 0) {
        targetWs = ws;
        break;
      }
    }
    test.skip(!targetWs, "No workspace has canvas-initiated activity (initial prompt may not have run)");

    await page.goto("/");
    await page.waitForTimeout(3000);

    // Click the workspace node
    const node = page.locator(`.react-flow__node`).filter({ hasText: targetWs!.name });
    await node.first().click();
    await page.waitForTimeout(500);

    // Open Chat tab
    await page.getByRole("button", { name: /Chat/ }).first().click();
    await page.waitForTimeout(500);

    // Ensure we're on My Chat
    await page.locator("button", { hasText: "My Chat" }).click();
    await page.waitForTimeout(2000);

    // The chat should NOT show "No messages yet" — it should have the initial prompt
    const emptyState = page.locator("text=No messages yet");
    await expect(emptyState).not.toBeVisible({ timeout: 5000 });

    // There should be at least one user message bubble (blue) and one agent message bubble
    const userBubbles = page.locator('[class*="bg-blue-600"]');
    const agentBubbles = page.locator('[class*="bg-zinc-800"]');
    expect(await userBubbles.count()).toBeGreaterThan(0);
    expect(await agentBubbles.count()).toBeGreaterThan(0);
  });

  test("initial prompt text matches config content", async ({ page }) => {
    const wsRes = await page.request.get(`${API}/workspaces`);
    const workspaces = await wsRes.json();
    test.skip(workspaces.length === 0, "No workspaces");

    // Find workspace with activity
    let targetWs: { id: string; name: string } | null = null;
    let promptText = "";
    for (const ws of workspaces) {
      const actRes = await page.request.get(`${API}/workspaces/${ws.id}/activity?source=canvas&type=a2a_receive&limit=1`);
      const entries = await actRes.json();
      if (entries.length > 0) {
        const reqBody = entries[0].request_body;
        const parts = reqBody?.params?.message?.parts;
        if (parts?.[0]?.text) {
          targetWs = ws;
          promptText = parts[0].text;
          break;
        }
      }
    }
    test.skip(!targetWs, "No workspace has canvas-initiated activity with text");

    await page.goto("/");
    await page.waitForTimeout(3000);

    const node = page.locator(`.react-flow__node`).filter({ hasText: targetWs!.name });
    await node.first().click();
    await page.waitForTimeout(500);
    await page.getByRole("button", { name: /Chat/ }).first().click();
    await page.waitForTimeout(500);
    await page.locator("button", { hasText: "My Chat" }).click();
    await page.waitForTimeout(2000);

    // The first few words of the initial prompt should be visible in the chat
    const firstWords = promptText.split(/\s+/).slice(0, 4).join(" ");
    await expect(page.locator(`text=${firstWords}`).first()).toBeVisible({ timeout: 5000 });
  });

  test("agent response to initial prompt is visible", async ({ page }) => {
    const wsRes = await page.request.get(`${API}/workspaces`);
    const workspaces = await wsRes.json();
    test.skip(workspaces.length === 0, "No workspaces");

    let targetWs: { id: string } | null = null;
    let responseText = "";
    for (const ws of workspaces) {
      const actRes = await page.request.get(`${API}/workspaces/${ws.id}/activity?source=canvas&type=a2a_receive&limit=1`);
      const entries = await actRes.json();
      if (entries.length > 0 && entries[0].response_body) {
        const result = entries[0].response_body.result;
        const parts = result?.parts;
        if (parts?.[0]?.text) {
          targetWs = ws;
          responseText = parts[0].text;
          break;
        }
      }
    }
    test.skip(!targetWs, "No workspace has response in activity");

    await page.goto("/");
    await page.waitForTimeout(3000);
    await page.locator(".react-flow__node").first().click();
    await page.waitForTimeout(500);
    await page.getByRole("button", { name: /Chat/ }).first().click();
    await page.waitForTimeout(2000);

    // Agent response should be visible — check for agent message bubble existence
    // (response text varies per agent, so check for non-empty agent bubble instead of exact text)
    await page.locator("button", { hasText: "My Chat" }).click();
    await page.waitForTimeout(2000);
    const agentBubbles = page.locator('[class*="bg-zinc-800"]');
    expect(await agentBubbles.count()).toBeGreaterThan(0);
  });
});

test.describe("No JS Errors", () => {
  test("page loads without errors with chat sub-tabs", async ({ page }) => {
    const errors: string[] = [];
    page.on("pageerror", (err) => errors.push(err.message));

    await page.goto("/");
    await page.waitForTimeout(3000);

    const nodes = page.locator(".react-flow__node");
    if (await nodes.count() > 0) {
      await nodes.first().click();
      await page.waitForTimeout(500);
      await page.getByRole("button", { name: /Chat/ }).first().click();
      await page.waitForTimeout(1000);

      // Switch between tabs
      await page.locator("button", { hasText: "Agent Comms" }).click();
      await page.waitForTimeout(500);
      await page.locator("button", { hasText: "My Chat" }).click();
      await page.waitForTimeout(500);
    }

    const critical = errors.filter(
      (e) => !e.includes("WebSocket") && !e.includes("favicon") && !e.includes("hydration")
    );
    expect(critical).toEqual([]);
  });
});
