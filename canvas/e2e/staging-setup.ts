/**
 * Playwright global setup for the staging canvas E2E.
 *
 * Provisions a fresh staging org per test run (via POST /cp/orgs against
 * staging CP), waits for the tenant EC2 + cloudflared tunnel + TLS
 * propagation, provisions one hermes workspace on the new tenant, waits
 * for it to reach status=online, then exports:
 *
 *   STAGING_TENANT_URL    — https://<slug>.moleculesai.app
 *   STAGING_WORKSPACE_ID  — UUID of the provisioned hermes workspace
 *   STAGING_SLUG          — org slug (for teardown)
 *
 * staging-teardown.ts consumes STAGING_SLUG to DELETE the org.
 *
 * Required env (set via GH Actions secrets in the workflow):
 *   MOLECULE_CP_URL           default: https://staging-api.moleculesai.app
 *   MOLECULE_SESSION_COOKIE   WorkOS session for the staging test user
 *   MOLECULE_ADMIN_TOKEN      CP admin bearer for teardown (unused in setup
 *                             but checked here so both halves fail fast)
 *
 * Runs only when CANVAS_E2E_STAGING=1 so local `pnpm playwright test` in
 * dev doesn't try to provision against staging by accident.
 */

import type { FullConfig } from "@playwright/test";
import { writeFileSync } from "fs";
import { join } from "path";

const CP_URL = process.env.MOLECULE_CP_URL || "https://staging-api.moleculesai.app";
const SESSION = process.env.MOLECULE_SESSION_COOKIE;
const ADMIN_TOKEN = process.env.MOLECULE_ADMIN_TOKEN;
const STAGING = process.env.CANVAS_E2E_STAGING === "1";

const PROVISION_TIMEOUT_MS = 15 * 60 * 1000; // 15 min cold-boot budget
const WORKSPACE_ONLINE_TIMEOUT_MS = 10 * 60 * 1000;
const TLS_TIMEOUT_MS = 3 * 60 * 1000;

async function jsonFetch(
  url: string,
  init: RequestInit = {},
): Promise<{ status: number; body: any }> {
  const res = await fetch(url, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init.headers || {}),
    },
  });
  let body: any = null;
  try {
    body = await res.json();
  } catch {
    /* non-JSON */
  }
  return { status: res.status, body };
}

async function waitFor<T>(
  op: () => Promise<T | null>,
  deadlineMs: number,
  intervalMs: number,
  desc: string,
): Promise<T> {
  const deadline = Date.now() + deadlineMs;
  while (Date.now() < deadline) {
    const v = await op();
    if (v !== null) return v;
    await new Promise((r) => setTimeout(r, intervalMs));
  }
  throw new Error(`${desc}: timed out after ${Math.round(deadlineMs / 1000)}s`);
}

function makeSlug(): string {
  // Matches CP's ^[a-z][a-z0-9-]{2,31}$. The "e2e-" prefix lets auto-cleanup
  // crons grep-find leftovers from crashed runs.
  const y = new Date().toISOString().slice(0, 10).replace(/-/g, "");
  const rand = Math.random().toString(36).slice(2, 8);
  return `e2e-canvas-${y}-${rand}`.slice(0, 32);
}

export default async function globalSetup(_config: FullConfig): Promise<void> {
  if (!STAGING) {
    console.log("[staging-setup] CANVAS_E2E_STAGING not set, skipping");
    return;
  }

  if (!SESSION) {
    throw new Error("MOLECULE_SESSION_COOKIE required for staging E2E");
  }
  if (!ADMIN_TOKEN) {
    throw new Error(
      "MOLECULE_ADMIN_TOKEN required for staging E2E (teardown needs it)",
    );
  }

  const slug = makeSlug();
  const cookieHeader = `molecule_cp_session=${SESSION}`;
  console.log(`[staging-setup] Using slug=${slug}`);

  // 1. Accept terms (idempotent — already-accepted returns 2xx or 400)
  await jsonFetch(`${CP_URL}/cp/auth/accept-terms`, {
    method: "POST",
    headers: { Cookie: cookieHeader },
    body: JSON.stringify({}),
  }).catch(() => {
    /* best-effort */
  });

  // 2. Create org
  const create = await jsonFetch(`${CP_URL}/cp/orgs`, {
    method: "POST",
    headers: { Cookie: cookieHeader },
    body: JSON.stringify({ slug, name: `E2E Canvas ${slug}` }),
  });
  if (create.status >= 400) {
    throw new Error(
      `POST /cp/orgs returned ${create.status}: ${JSON.stringify(create.body)}`,
    );
  }
  console.log(`[staging-setup] Org created: ${slug}`);

  // 3. Wait for tenant provision (status=running)
  const finalStatus = await waitFor<{ url?: string; status: string }>(
    async () => {
      const r = await jsonFetch(
        `${CP_URL}/cp/orgs/${slug}/provision-status`,
        { headers: { Cookie: cookieHeader } },
      );
      if (r.status !== 200) return null;
      if (r.body?.status === "running") return r.body;
      if (r.body?.status === "failed") {
        throw new Error(`Provisioning failed: ${JSON.stringify(r.body)}`);
      }
      return null;
    },
    PROVISION_TIMEOUT_MS,
    15_000,
    "tenant provision",
  );

  const tenantURL =
    finalStatus.url ||
    `https://${slug}.${CP_URL.includes("staging") ? "moleculesai.app" : "moleculesai.app"}`;
  console.log(`[staging-setup] Tenant URL: ${tenantURL}`);

  // 4. Wait for tenant TLS readiness
  await waitFor<boolean>(
    async () => {
      try {
        const res = await fetch(`${tenantURL}/health`, {
          signal: AbortSignal.timeout(5000),
        });
        return res.ok ? true : null;
      } catch {
        return null;
      }
    },
    TLS_TIMEOUT_MS,
    5_000,
    "tenant TLS",
  );

  // 5. Provision one hermes workspace (cheapest, fastest-booting)
  const ws = await jsonFetch(`${tenantURL}/workspaces`, {
    method: "POST",
    headers: { Cookie: cookieHeader },
    body: JSON.stringify({
      name: "E2E Canvas Test",
      runtime: "hermes",
      tier: 2,
      model: "gpt-4o",
    }),
  });
  if (ws.status >= 400 || !ws.body?.id) {
    throw new Error(
      `Workspace create failed (${ws.status}): ${JSON.stringify(ws.body)}`,
    );
  }
  const workspaceId = ws.body.id as string;
  console.log(`[staging-setup] Workspace created: ${workspaceId}`);

  // 6. Wait for workspace online
  await waitFor<boolean>(
    async () => {
      const r = await jsonFetch(`${tenantURL}/workspaces/${workspaceId}`, {
        headers: { Cookie: cookieHeader },
      });
      if (r.status !== 200) return null;
      if (r.body?.status === "online") return true;
      if (r.body?.status === "failed") {
        throw new Error(
          `Workspace ${workspaceId} failed: ${r.body.last_sample_error || ""}`,
        );
      }
      return null;
    },
    WORKSPACE_ONLINE_TIMEOUT_MS,
    10_000,
    "workspace online",
  );
  console.log(`[staging-setup] Workspace online`);

  // 7. Export via a state file so staging-teardown and the test spec can
  //    pick up the same slug / urls. Playwright's global setup can't
  //    export env to the test subprocess directly in all configurations.
  const stateFile = join(process.cwd(), ".playwright-staging-state.json");
  writeFileSync(
    stateFile,
    JSON.stringify({ slug, tenantURL, workspaceId }, null, 2),
  );
  // Also set env for in-process test reads.
  process.env.STAGING_SLUG = slug;
  process.env.STAGING_TENANT_URL = tenantURL;
  process.env.STAGING_WORKSPACE_ID = workspaceId;
  process.env.STAGING_SESSION_COOKIE = SESSION;

  console.log(`[staging-setup] Ready — ${stateFile}`);
}
