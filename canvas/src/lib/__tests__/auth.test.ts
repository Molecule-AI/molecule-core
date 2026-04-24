/**
 * @vitest-environment jsdom
 */
import { describe, it, expect, vi, afterEach } from "vitest";
import { fetchSession, redirectToLogin } from "../auth";

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe("fetchSession", () => {
  it("returns session on 200", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ user_id: "u1", org_id: "o1", email: "a@x.com" }),
    }));
    const s = await fetchSession();
    expect(s).toEqual({ user_id: "u1", org_id: "o1", email: "a@x.com" });
  });

  it("returns null on 401 without throwing", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: false, status: 401 }));
    const s = await fetchSession();
    expect(s).toBeNull();
  });

  it("throws on 500 so transient outages aren't treated as 'anonymous'", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: false, status: 500, statusText: "oops" }));
    await expect(fetchSession()).rejects.toThrow("500");
  });

  it("sends credentials:include for cross-origin cookies", async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: false, status: 401 });
    vi.stubGlobal("fetch", fetchMock);
    await fetchSession();
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining("/cp/auth/me"),
      expect.objectContaining({ credentials: "include" }),
    );
  });
});

describe("redirectToLogin", () => {
  it("sets window.location to cp login URL with return_to", () => {
    const href = "https://acme.moleculesai.app/dashboard";
    Object.defineProperty(window, "location", {
      writable: true,
      value: {
        href,
        pathname: "/dashboard",
        hostname: "acme.moleculesai.app",
        protocol: "https:",
      },
    });
    redirectToLogin("sign-in");
    expect((window.location as unknown as { href: string }).href).toContain("/cp/auth/login");
    expect((window.location as unknown as { href: string }).href).toContain(
      encodeURIComponent(href),
    );
  });

  it("uses signup path for sign-up screenHint", () => {
    Object.defineProperty(window, "location", {
      writable: true,
      value: {
        href: "https://acme.moleculesai.app/",
        pathname: "/",
        hostname: "acme.moleculesai.app",
        protocol: "https:",
      },
    });
    redirectToLogin("sign-up");
    expect((window.location as unknown as { href: string }).href).toContain("/cp/auth/signup");
  });

  // Regression: AuthGate + redirectToLogin mutual recursion on /cp/auth/login
  // caused double-encoded return_to that grew until the URL exceeded 431.
  // Guard: redirectToLogin must NOT set window.location when already on an
  // auth path, otherwise each call adds another encoding layer.
  it("does NOT set window.location when already on /cp/auth/login (redirect loop guard)", () => {
    const loginHref = "https://app.moleculesai.app/cp/auth/login?return_to=https%3A%2F%2Facme.moleculesai.app%2Fdashboard";
    Object.defineProperty(window, "location", {
      writable: true,
      value: {
        href: loginHref,
        pathname: "/cp/auth/login",
        hostname: "app.moleculesai.app",
        protocol: "https:",
      },
    });
    redirectToLogin("sign-in");
    // href must be unchanged — any mutation means the guard is missing
    expect((window.location as unknown as { href: string }).href).toBe(loginHref);
  });

  it("does NOT set window.location when already on /cp/auth/signup (redirect loop guard)", () => {
    const signupHref = "https://app.moleculesai.app/cp/auth/signup";
    Object.defineProperty(window, "location", {
      writable: true,
      value: {
        href: signupHref,
        pathname: "/cp/auth/signup",
        hostname: "app.moleculesai.app",
        protocol: "https:",
      },
    });
    redirectToLogin("sign-up");
    expect((window.location as unknown as { href: string }).href).toBe(signupHref);
  });
});
