/**
 * Molecule AI tenant proxy — Cloudflare Worker
 *
 * Routes *.moleculesai.app requests to the correct EC2 tenant instance.
 * Replaces per-tenant DNS records with a single wildcard + edge routing.
 *
 * Cache strategy (3-tier):
 *   L1: in-memory Map (60s TTL, per-isolate)
 *   L2: Workers KV (5 min TTL, stale-while-revalidate)
 *   L3: CP API — GET /cp/orgs/:slug/instance
 *   Fallback: serve stale KV when CP is unreachable
 */

export interface Env {
  TENANT_CACHE: KVNamespace;
  CP_API_URL: string;
}

interface TenantInfo {
  slug: string;
  status: string; // "running" | "provisioning" | "failed"
  ip: string | null;
  org_id: string;
  admin_token?: string;
}

// L1: in-memory cache (per-isolate, 60s TTL)
const memCache = new Map<string, { data: TenantInfo; expires: number }>();
const MEM_TTL_MS = 60_000;
const KV_TTL_S = 300; // 5 min

// Subdomains that are NOT tenants — handled by explicit DNS records
const RESERVED = new Set(["api", "app", "www", "docs", "doc", "status", "staging-api", "tunneltest"]);

// Routes that go to platform (:8080) vs canvas (:3000)
const API_PREFIXES = [
  "/health", "/metrics", "/workspaces", "/registry", "/templates",
  "/org", "/settings", "/plugins", "/events", "/bundles", "/channels",
  "/webhooks", "/approvals", "/admin", "/canvas", "/ws",
];

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    const url = new URL(request.url);
    const host = url.hostname;

    // Extract slug from hostname: "acme.moleculesai.app" → "acme"
    const slug = host.replace(".moleculesai.app", "");
    if (!slug || slug === host || RESERVED.has(slug) || slug.includes(".")) {
      // Pass through to origin (tunnel CNAME or explicit DNS record).
      // slug.includes(".") catches multi-level subdomains like
      // "foo.staging.moleculesai.app" which are routed via CF Tunnel.
      return fetch(request);
    }

    // Lookup tenant backend
    const tenant = await resolveTenant(slug, env);

    if (!tenant) {
      return notFoundPage(slug);
    }

    if (tenant.status === "provisioning" || !tenant.ip) {
      return provisioningPage(slug);
    }

    if (tenant.status === "failed") {
      return errorPage(slug);
    }

    // Route ALL traffic to :8080 (Go platform). The platform proxies non-API
    // routes to Canvas internally via CANVAS_PROXY_URL. We don't split traffic
    // between :8080 and :3000 because Canvas may bind to 127.0.0.1 only
    // (not externally reachable) while the platform is always on 0.0.0.0.
    const backendUrl = `http://${tenant.ip}:8080${url.pathname}${url.search}`;

    // WebSocket upgrade
    if (request.headers.get("Upgrade") === "websocket") {
      return fetch(backendUrl, request);
    }

    // Proxy the request
    const headers = new Headers(request.headers);
    headers.set("X-Molecule-Org-Id", tenant.org_id);
    headers.set("Origin", `https://${slug}.moleculesai.app`);
    headers.set("X-Forwarded-For", request.headers.get("CF-Connecting-IP") || "");
    headers.set("X-Forwarded-Proto", "https");
    headers.set("Host", `${slug}.moleculesai.app`);
    // Inject ADMIN_TOKEN for AdminAuth — the tenant platform validates this
    // as a dedicated admin credential (not a workspace token).
    if (tenant.admin_token) {
      headers.set("Authorization", `Bearer ${tenant.admin_token}`);
    }

    const proxyReq = new Request(backendUrl, {
      method: request.method,
      headers,
      body: request.body,
      redirect: "manual",
    });

    try {
      const resp = await fetch(proxyReq);
      // Strip backend hop headers, pass everything else through
      const respHeaders = new Headers(resp.headers);
      respHeaders.delete("transfer-encoding");
      return new Response(resp.body, {
        status: resp.status,
        statusText: resp.statusText,
        headers: respHeaders,
      });
    } catch {
      return new Response("Backend unavailable", { status: 502 });
    }
  },
};

// ---------------------------------------------------------------------------
// 3-tier cache resolution
// ---------------------------------------------------------------------------

async function resolveTenant(
  slug: string,
  env: Env,
): Promise<TenantInfo | null> {
  // L1: in-memory
  const mem = memCache.get(slug);
  if (mem && Date.now() < mem.expires) {
    return mem.data;
  }

  // L2: KV (stale-while-revalidate)
  let kvData: TenantInfo | null = null;
  try {
    const kvRaw = await env.TENANT_CACHE.get(slug);
    if (kvRaw) {
      kvData = JSON.parse(kvRaw) as TenantInfo;
      // Populate L1 from KV
      memCache.set(slug, { data: kvData, expires: Date.now() + MEM_TTL_MS });
    }
  } catch { /* KV read failure — continue to L3 */ }

  // L3: CP API
  try {
    const resp = await fetch(
      `${env.CP_API_URL}/cp/orgs/${encodeURIComponent(slug)}/instance`,
      { headers: { "User-Agent": "molecule-tenant-proxy/1.0" } },
    );

    if (resp.status === 404) {
      // Org doesn't exist — cache the miss briefly to avoid hammering CP
      memCache.set(slug, {
        data: { slug, status: "not_found", ip: null, org_id: "" },
        expires: Date.now() + 10_000, // 10s negative cache
      });
      return null;
    }

    if (resp.ok) {
      const data = (await resp.json()) as TenantInfo;
      // Update both caches
      memCache.set(slug, { data, expires: Date.now() + MEM_TTL_MS });
      await env.TENANT_CACHE.put(slug, JSON.stringify(data), {
        expirationTtl: KV_TTL_S,
      }).catch(() => {}); // KV write failure is non-fatal
      return data;
    }
  } catch {
    // CP unreachable — fall back to stale KV
  }

  // Fallback: stale KV data (any age) is better than an error
  return kvData;
}

// ---------------------------------------------------------------------------
// Static response pages
// ---------------------------------------------------------------------------

function provisioningPage(slug: string): Response {
  return new Response(
    `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <meta http-equiv="refresh" content="5">
  <title>${slug} - Setting up | Molecule AI</title>
  <style>
    *{margin:0;padding:0;box-sizing:border-box}
    body{background:#09090b;color:#f4f4f5;font-family:-apple-system,BlinkMacSystemFont,sans-serif;
         display:flex;align-items:center;justify-content:center;min-height:100vh}
    .card{text-align:center;max-width:420px;padding:3rem 2rem}
    .spinner{width:48px;height:48px;border:3px solid #27272a;border-top-color:#3b82f6;
             border-radius:50%;animation:spin 1s linear infinite;margin:0 auto 1.5rem}
    @keyframes spin{to{transform:rotate(360deg)}}
    h1{font-size:1.25rem;font-weight:600;margin-bottom:.5rem}
    p{font-size:.875rem;color:#a1a1aa;line-height:1.6}
    .hint{margin-top:1.5rem;font-size:.75rem;color:#52525b}
  </style>
</head>
<body>
  <div class="card">
    <div class="spinner"></div>
    <h1>Setting up your workspace</h1>
    <p>Your cloud instance is starting up. This usually takes 2-3 minutes.</p>
    <p class="hint">This page refreshes automatically.</p>
  </div>
</body>
</html>`,
    {
      status: 202,
      headers: {
        "Content-Type": "text/html;charset=utf-8",
        "Cache-Control": "no-cache",
        "Retry-After": "5",
      },
    },
  );
}

function notFoundPage(slug: string): Response {
  return new Response(
    `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>Not Found | Molecule AI</title>
  <style>
    *{margin:0;padding:0;box-sizing:border-box}
    body{background:#09090b;color:#f4f4f5;font-family:-apple-system,BlinkMacSystemFont,sans-serif;
         display:flex;align-items:center;justify-content:center;min-height:100vh}
    .card{text-align:center;max-width:420px;padding:3rem 2rem}
    h1{font-size:1.25rem;font-weight:600;margin-bottom:.5rem}
    p{font-size:.875rem;color:#a1a1aa;line-height:1.6}
    a{color:#3b82f6;text-decoration:none}a:hover{text-decoration:underline}
  </style>
</head>
<body>
  <div class="card">
    <h1>Organization not found</h1>
    <p><strong>${slug}.moleculesai.app</strong> doesn't exist.</p>
    <p style="margin-top:1rem"><a href="https://app.moleculesai.app">Go to Molecule AI</a></p>
  </div>
</body>
</html>`,
    { status: 404, headers: { "Content-Type": "text/html;charset=utf-8" } },
  );
}

function errorPage(slug: string): Response {
  return new Response(
    `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>Error | Molecule AI</title>
  <style>
    *{margin:0;padding:0;box-sizing:border-box}
    body{background:#09090b;color:#f4f4f5;font-family:-apple-system,BlinkMacSystemFont,sans-serif;
         display:flex;align-items:center;justify-content:center;min-height:100vh}
    .card{text-align:center;max-width:420px;padding:3rem 2rem}
    h1{font-size:1.25rem;font-weight:600;margin-bottom:.5rem;color:#ef4444}
    p{font-size:.875rem;color:#a1a1aa;line-height:1.6}
    a{color:#3b82f6;text-decoration:none}a:hover{text-decoration:underline}
  </style>
</head>
<body>
  <div class="card">
    <h1>Provisioning failed</h1>
    <p>Something went wrong setting up <strong>${slug}</strong>.</p>
    <p style="margin-top:1rem"><a href="https://app.moleculesai.app">Return to dashboard</a></p>
  </div>
</body>
</html>`,
    { status: 503, headers: { "Content-Type": "text/html;charset=utf-8" } },
  );
}
