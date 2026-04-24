/**
 * Playwright global setup for the staging canvas E2E.
 *
 * Provisions a fresh staging org per run (POST /cp/admin/orgs), fetches
 * the per-tenant admin token, provisions one hermes workspace, waits
 * for online, then exports:
 *
 *   STAGING_TENANT_URL     https://<slug>.moleculesai.app
 *   STAGING_WORKSPACE_ID   UUID of the hermes workspace
 *   STAGING_TENANT_TOKEN   per-tenant admin bearer (for spec requests)
 *   STAGING_SLUG           org slug (used by teardown)
 *
 * Required env:
 *   MOLECULE_CP_URL        default: https://staging-api.moleculesai.app
 *   MOLECULE_ADMIN_TOKEN   CP admin bearer (Railway staging
 *                          CP_ADMIN_API_TOKEN). Drives provision +
 *                          tenant-token retrieval + teardown via a
 *                          single credential.
 */

import type { FullConfig } from "@playwright/test";
import { writeFileSync } from "fs";
import { join } from "path";

const CP_URL = process.env.MOLECULE_CP_URL || "https://staging-api.moleculesai.app";
const ADMIN_TOKEN = process.env.MOLECULE_ADMIN_TOKEN;
const STAGING = process.env.CANVAS_E2E_STAGING === "1";

// Tenant cold boot on staging regularly takes 12-15 min when the
// workspace-server Docker image isn't already cached on the AMI. Raised
// to 20 min to match tests/e2e/test_staging_full_saas.sh (PR #1930)
// after repeated "tenant provision: timed out after 900s" flakes
// were blocking staging→main syncs on 2026-04-24.
const PROVISION_TIMEOUT_MS = 20 * 60 * 1000;
const WORKSPACE_ONLINE_TIMEOUT_MS = 20 * 60 * 1000;
const TLS_TIMEOUT_MS = 3 * 60 * 1000;

async function jsonFetch(
  url: string,
  init: RequestInit = {},
): Promise<{ status: number; body: any }> {
  const res = await fetch(url, {
    ...init,
    headers: { "Content-Type": "application/json", ...(init.headers || {}) },
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
  const y = new Date().toISOString().slice(0, 10).replace(/-/g, "");
  const rand = Math.random().toString(36).slice(2, 8);
  return `e2e-canvas-${y}-${rand}`.slice(0, 32);
}

export default async function globalSetup(_config: FullConfig): Promise<void> {
  if (!STAGING) {
    console.log("[staging-setup] CANVAS_E2E_STAGING not set, skipping");
    return;
  }
  if (!ADMIN_TOKEN) {
    throw new Error(
      "MOLECULE_ADMIN_TOKEN required (Railway staging CP_ADMIN_API_TOKEN)",
    );
  }

  const slug = makeSlug();
  const adminAuth = { Authorization: `Bearer ${ADMIN_TOKEN}` };
  console.log(`[staging-setup] Using slug=${slug}`);

  // 1. Create org via admin endpoint — no WorkOS session needed
  const create = await jsonFetch(`${CP_URL}/cp/admin/orgs`, {
    method: "POST",
    headers: adminAuth,
    body: JSON.stringify({
      slug,
      name: `E2E Canvas ${slug}`,
      owner_user_id: `e2e-runner:${slug}`,
    }),
  });
  if (create.status >= 400) {
    throw new Error(
      `POST /cp/admin/orgs ${create.status}: ${JSON.stringify(create.body)}`,
    );
  }
  console.log(`[staging-setup] Org created: ${slug}`);

  // 2. Wait for tenant running (admin-orgs list is the status source)
  await waitFor<boolean>(
    async () => {
      const r = await jsonFetch(`${CP_URL}/cp/admin/orgs`, { headers: adminAuth });
      if (r.status !== 200) return null;
      const row = (r.body?.orgs || []).find((o: any) => o.slug === slug);
      if (!row) return null;
      if (row.status === "running") return true;
      if (row.status === "failed") throw new Error(`provision failed: ${slug}`);
      return null;
    },
    PROVISION_TIMEOUT_MS,
    15_000,
    "tenant provision",
  );
  console.log(`[staging-setup] Tenant running`);

  // 3. Fetch per-tenant admin token
  const tokRes = await jsonFetch(
    `${CP_URL}/cp/admin/orgs/${slug}/admin-token`,
    { headers: adminAuth },
  );
  if (tokRes.status !== 200 || !tokRes.body?.admin_token) {
    throw new Error(
      `tenant-token fetch ${tokRes.status}: ${JSON.stringify(tokRes.body)}`,
    );
  }
  const tenantToken: string = tokRes.body.admin_token;
  const tenantURL = `https://${slug}.moleculesai.app`;
  console.log(`[staging-setup] Tenant URL: ${tenantURL}`);

  // 4. TLS readiness
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

  // 5. Provision workspace
  const tenantAuth = { Authorization: `Bearer ${tenantToken}` };
  const ws = await jsonFetch(`${tenantURL}/workspaces`, {
    method: "POST",
    headers: tenantAuth,
    body: JSON.stringify({
      name: "E2E Canvas Test",
      runtime: "hermes",
      tier: 2,
      model: "gpt-4o",
    }),
  });
  if (ws.status >= 400 || !ws.body?.id) {
    throw new Error(`Workspace create ${ws.status}: ${JSON.stringify(ws.body)}`);
  }
  const workspaceId = ws.body.id as string;
  console.log(`[staging-setup] Workspace created: ${workspaceId}`);

  // 6. Wait for workspace online
  await waitFor<boolean>(
    async () => {
      const r = await jsonFetch(`${tenantURL}/workspaces/${workspaceId}`, {
        headers: tenantAuth,
      });
      if (r.status !== 200) return null;
      if (r.body?.status === "online") return true;
      if (r.body?.status === "failed") {
        throw new Error(`Workspace failed: ${r.body.last_sample_error || ""}`);
      }
      return null;
    },
    WORKSPACE_ONLINE_TIMEOUT_MS,
    10_000,
    "workspace online",
  );
  console.log(`[staging-setup] Workspace online`);

  // 7. Hand state off to tests + teardown
  const stateFile = join(process.cwd(), ".playwright-staging-state.json");
  writeFileSync(
    stateFile,
    JSON.stringify({ slug, tenantURL, workspaceId, tenantToken }, null, 2),
  );
  process.env.STAGING_SLUG = slug;
  process.env.STAGING_TENANT_URL = tenantURL;
  process.env.STAGING_WORKSPACE_ID = workspaceId;
  process.env.STAGING_TENANT_TOKEN = tenantToken;
  console.log(`[staging-setup] Ready — ${stateFile}`);
}
