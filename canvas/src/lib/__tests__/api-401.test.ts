// @vitest-environment jsdom
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

// Dedicated file for the 401 → login-redirect tests because they need
// `window.location.hostname` (jsdom), while the rest of api.test.ts
// runs happily in node. Splitting keeps the node tests fast.

// ---------------------------------------------------------------------------
// 401 handling — gated on SaaS-tenant hostname
// ---------------------------------------------------------------------------
//
// Before fix/quickstart-bugless, any 401 from any endpoint triggered
// `redirectToLogin()`, navigating to `/cp/auth/login`. That route
// exists only on SaaS (mounted by cp_proxy when CP_UPSTREAM_URL is
// set). On localhost / self-hosted / Vercel preview it 404s, so the
// user lands on a broken login page instead of seeing the actual error.
//
// These tests lock in:
//   - SaaS tenant hostname (*.moleculesai.app) → 401 still redirects.
//   - non-SaaS hostname (localhost, LAN IP, apex) → 401 throws, no
//     redirect, so the caller renders a real error affordance.

const mockFetch = vi.fn();
globalThis.fetch = mockFetch;

function mockFailure(status: number, text: string) {
  mockFetch.mockResolvedValueOnce({
    ok: false,
    status,
    json: () => Promise.reject(new Error("no json")),
    text: () => Promise.resolve(text),
  } as unknown as Response);
}

function setHostname(host: string) {
  Object.defineProperty(window, "location", {
    configurable: true,
    value: { ...window.location, hostname: host },
  });
}

describe("api 401 handling", () => {
  let redirectSpy: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    vi.clearAllMocks();
    vi.resetModules();
    redirectSpy = vi.fn();
    vi.doMock("../auth", () => ({
      redirectToLogin: redirectSpy,
      // Stub siblings so any other import of ../auth in the chain
      // (AuthGate, TermsGate, etc.) still resolves.
      fetchSession: vi.fn().mockResolvedValue(null),
    }));
  });

  afterEach(() => {
    vi.doUnmock("../auth");
    vi.resetModules();
  });

  it("redirects to login on SaaS tenant hostname", async () => {
    setHostname("acme.moleculesai.app");
    mockFailure(401, '{"error":"admin auth required"}');

    const { api } = await import("../api");
    await expect(api.get("/workspaces")).rejects.toThrow(/Session expired/);
    expect(redirectSpy).toHaveBeenCalledWith("sign-in");
  });

  it("does NOT redirect on localhost — throws a real error instead", async () => {
    setHostname("localhost");
    mockFailure(401, '{"error":"admin auth required"}');

    const { api } = await import("../api");
    await expect(api.get("/workspaces")).rejects.toThrow(/401/);
    expect(redirectSpy).not.toHaveBeenCalled();
  });

  it("does NOT redirect on a LAN hostname", async () => {
    setHostname("192.168.1.74");
    mockFailure(401, '{"error":"missing workspace auth token"}');

    const { api } = await import("../api");
    await expect(api.get("/workspaces/abc/activity")).rejects.toThrow(/401/);
    expect(redirectSpy).not.toHaveBeenCalled();
  });

  it("does NOT redirect on reserved subdomains (app.moleculesai.app)", async () => {
    // `app` is in reservedSubdomains — getTenantSlug returns "" there.
    // Users landing on app.moleculesai.app (pre-tenant-selection) must
    // see the real 401 error rather than loop on login.
    setHostname("app.moleculesai.app");
    mockFailure(401, '{"error":"admin auth required"}');

    const { api } = await import("../api");
    await expect(api.get("/workspaces")).rejects.toThrow(/401/);
    expect(redirectSpy).not.toHaveBeenCalled();
  });
});
