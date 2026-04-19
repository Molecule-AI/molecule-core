// @vitest-environment jsdom
/**
 * Tests for /orgs — the post-signup landing page (PR #992 feat/canvas-orgs-landing
 * plus #994 feat/canvas-post-checkout-redirect).
 *
 * The page is the only route the control-plane Callback hands a new session to,
 * so bugs here strand new users. Covers:
 *   - Signed-out → redirectToLogin
 *   - Failed /cp/orgs → error state + retry button
 *   - Empty org list → EmptyState w/ CreateOrgForm
 *   - `running` org → Open button links to `{slug}.{appDomain}`
 *   - `awaiting_payment` org → "Complete payment" → /pricing?org=<slug>
 *   - `failed` org → mailto support link
 *   - `?checkout=success` param → CheckoutBanner renders + URL is scrubbed
 *   - Polling: provisioning orgs schedule a 5s refresh (fake timers)
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor, cleanup, fireEvent } from "@testing-library/react";

// ── Hoisted mocks ────────────────────────────────────────────────────────────
// vi.mock factories are hoisted above imports; any captured references must
// come from vi.hoisted() or the factory would see "undefined before init".

const { mockFetchSession, mockRedirectToLogin } = vi.hoisted(() => ({
  mockFetchSession: vi.fn(),
  mockRedirectToLogin: vi.fn(),
}));

vi.mock("@/lib/auth", () => ({
  fetchSession: mockFetchSession,
  redirectToLogin: mockRedirectToLogin,
}));

// api module provides PLATFORM_URL; page imports it as a constant
vi.mock("@/lib/api", () => ({
  PLATFORM_URL: "https://cp.test",
}));

const mockFetch = vi.fn();
globalThis.fetch = mockFetch as unknown as typeof fetch;

// Import page AFTER mocks are declared
import OrgsPage from "../../app/orgs/page";

// ── Helpers ──────────────────────────────────────────────────────────────────

function okJson(body: unknown, status = 200) {
  return {
    ok: true,
    status,
    json: () => Promise.resolve(body),
    text: () => Promise.resolve(JSON.stringify(body)),
  } as unknown as Response;
}

function notOk(status: number, text = "boom") {
  return {
    ok: false,
    status,
    json: () => Promise.reject(new Error("no json")),
    text: () => Promise.resolve(text),
  } as unknown as Response;
}

function setLocation(href: string) {
  // jsdom allows window.location replacement via pushState rather than assign;
  // the component only reads `search` + `hostname` + `pathname`.
  const url = new URL(href);
  window.history.pushState({}, "", url.pathname + url.search);
  Object.defineProperty(window, "location", {
    configurable: true,
    value: {
      ...window.location,
      hostname: url.hostname,
      search: url.search,
      pathname: url.pathname,
    },
  });
}

beforeEach(() => {
  vi.clearAllMocks();
  setLocation("https://moleculesai.app/orgs");
});

afterEach(() => {
  cleanup();
});

// ── Tests ────────────────────────────────────────────────────────────────────

describe("/orgs — auth guard", () => {
  it("redirects to login when session is null", async () => {
    mockFetchSession.mockResolvedValueOnce(null);
    render(<OrgsPage />);
    await waitFor(() => expect(mockRedirectToLogin).toHaveBeenCalled());
    // Must not attempt to fetch /cp/orgs before auth is established
    expect(mockFetch).not.toHaveBeenCalledWith(
      expect.stringContaining("/cp/orgs"),
      expect.anything()
    );
  });
});

describe("/orgs — error state", () => {
  it("shows error + Retry button when /cp/orgs fails", async () => {
    mockFetchSession.mockResolvedValueOnce({ userId: "u-1" });
    mockFetch.mockResolvedValueOnce(notOk(500, "db down"));
    render(<OrgsPage />);
    await waitFor(() => expect(screen.getByText(/Error:/)).toBeTruthy());
    expect(screen.getByRole("button", { name: /retry/i })).toBeTruthy();
  });
});

describe("/orgs — empty list", () => {
  it("renders EmptyState with CreateOrgForm when user has zero orgs", async () => {
    mockFetchSession.mockResolvedValueOnce({ userId: "u-1" });
    mockFetch.mockResolvedValueOnce(okJson({ orgs: [] }));
    render(<OrgsPage />);
    await waitFor(() => expect(screen.getByText(/don't have any organizations/i)).toBeTruthy());
    expect(screen.getByRole("button", { name: /create organization/i })).toBeTruthy();
  });
});

describe("/orgs — CTAs by status", () => {
  const session = { userId: "u-1" };

  it("running → Open link targets {slug}.moleculesai.app", async () => {
    mockFetchSession.mockResolvedValueOnce(session);
    mockFetch.mockResolvedValueOnce(
      okJson({
        orgs: [
          {
            id: "o-1",
            slug: "acme",
            name: "Acme",
            plan: "pro",
            status: "running",
            created_at: "",
            updated_at: "",
          },
        ],
      })
    );
    render(<OrgsPage />);
    const link = (await screen.findByRole("link", { name: /open/i })) as HTMLAnchorElement;
    expect(link.href).toBe("https://acme.moleculesai.app/");
  });

  it("awaiting_payment → Complete payment link to /pricing?org=<slug>", async () => {
    mockFetchSession.mockResolvedValueOnce(session);
    mockFetch.mockResolvedValueOnce(
      okJson({
        orgs: [
          {
            id: "o-2",
            slug: "beta-co",
            name: "Beta",
            plan: "",
            status: "awaiting_payment",
            created_at: "",
            updated_at: "",
          },
        ],
      })
    );
    render(<OrgsPage />);
    const link = (await screen.findByRole("link", {
      name: /complete payment/i,
    })) as HTMLAnchorElement;
    expect(link.getAttribute("href")).toBe("/pricing?org=beta-co");
  });

  it("failed → mailto support link", async () => {
    mockFetchSession.mockResolvedValueOnce(session);
    mockFetch.mockResolvedValueOnce(
      okJson({
        orgs: [
          {
            id: "o-3",
            slug: "boom",
            name: "Boom",
            plan: "",
            status: "failed",
            created_at: "",
            updated_at: "",
          },
        ],
      })
    );
    render(<OrgsPage />);
    const link = (await screen.findByRole("link", {
      name: /contact support/i,
    })) as HTMLAnchorElement;
    expect(link.getAttribute("href")).toBe("mailto:support@moleculesai.app");
  });
});

describe("/orgs — post-checkout banner", () => {
  it("renders CheckoutBanner when ?checkout=success and scrubs the URL", async () => {
    setLocation("https://moleculesai.app/orgs?checkout=success");
    const replaceState = vi.spyOn(window.history, "replaceState");
    mockFetchSession.mockResolvedValueOnce({ userId: "u-1" });
    mockFetch.mockResolvedValueOnce(
      okJson({
        orgs: [
          {
            id: "o-1",
            slug: "acme",
            name: "Acme",
            plan: "pro",
            status: "running",
            created_at: "",
            updated_at: "",
          },
        ],
      })
    );
    render(<OrgsPage />);
    expect(await screen.findByText(/Payment confirmed/i)).toBeTruthy();
    // URL must be rewritten to drop the ?checkout flag so reload doesn't re-show the banner
    expect(replaceState).toHaveBeenCalled();
    const callArgs = replaceState.mock.calls[0];
    expect(callArgs[2]).toBe("/orgs");
  });

  it("does NOT render CheckoutBanner without ?checkout=success", async () => {
    mockFetchSession.mockResolvedValueOnce({ userId: "u-1" });
    mockFetch.mockResolvedValueOnce(okJson({ orgs: [] }));
    render(<OrgsPage />);
    await waitFor(() =>
      expect(screen.getByText(/don't have any organizations/i)).toBeTruthy()
    );
    expect(screen.queryByText(/Payment confirmed/i)).toBeNull();
  });
});

describe("/orgs — fetch includes credentials + timeout signal", () => {
  it("/cp/orgs fetch is called with credentials:include and an AbortSignal", async () => {
    mockFetchSession.mockResolvedValueOnce({ userId: "u-1" });
    mockFetch.mockResolvedValueOnce(okJson({ orgs: [] }));
    render(<OrgsPage />);
    await waitFor(() => expect(mockFetch).toHaveBeenCalled());
    const callArgs = mockFetch.mock.calls.find((c) =>
      String(c[0]).includes("/cp/orgs")
    );
    expect(callArgs).toBeDefined();
    expect(callArgs![1]).toMatchObject({ credentials: "include" });
    expect(callArgs![1].signal).toBeInstanceOf(AbortSignal);
  });
});

// ── Polling ──────────────────────────────────────────────────────────────────
// page.tsx line 83-88: if any org is `provisioning` OR `awaiting_payment`,
// schedule a 5s refresh so the user sees the state flip live after Stripe
// Checkout returns. Cleanup must clear the timer on unmount; otherwise a
// fast-nav-away leaves the interval firing against the CP indefinitely.

describe("/orgs — polling of in-flight orgs", () => {
  it("schedules a 5s refetch when at least one org is provisioning", async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    try {
      mockFetchSession.mockResolvedValue({ userId: "u-1" });
      mockFetch.mockResolvedValueOnce(
        okJson({
          orgs: [
            {
              id: "o-1",
              slug: "acme",
              name: "Acme",
              plan: "pro",
              status: "provisioning",
              created_at: "",
              updated_at: "",
            },
          ],
        })
      );
      // Second fetch (the poll refresh) returns a running org so we can
      // observe the state flip — and to let the test stop re-scheduling.
      mockFetch.mockResolvedValueOnce(
        okJson({
          orgs: [
            {
              id: "o-1",
              slug: "acme",
              name: "Acme",
              plan: "pro",
              status: "running",
              created_at: "",
              updated_at: "",
            },
          ],
        })
      );

      render(<OrgsPage />);
      // First fetch resolves
      await vi.waitFor(() => expect(mockFetch).toHaveBeenCalledTimes(1));
      // Advance past the 5s scheduled refresh
      await vi.advanceTimersByTimeAsync(5_100);
      // Second fetch is the poll refresh
      await vi.waitFor(() => expect(mockFetch).toHaveBeenCalledTimes(2));
    } finally {
      vi.useRealTimers();
    }
  });

  it("does NOT schedule a refetch when all orgs are running", async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    try {
      mockFetchSession.mockResolvedValue({ userId: "u-1" });
      mockFetch.mockResolvedValueOnce(
        okJson({
          orgs: [
            {
              id: "o-1",
              slug: "acme",
              name: "Acme",
              plan: "pro",
              status: "running",
              created_at: "",
              updated_at: "",
            },
          ],
        })
      );
      render(<OrgsPage />);
      await vi.waitFor(() => expect(mockFetch).toHaveBeenCalledTimes(1));
      // Advance well past the 5s poll window — no second fetch must fire
      await vi.advanceTimersByTimeAsync(10_000);
      expect(mockFetch).toHaveBeenCalledTimes(1);
    } finally {
      vi.useRealTimers();
    }
  });

  it("clears the poll timer on unmount — no fetch after unmount", async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    try {
      mockFetchSession.mockResolvedValue({ userId: "u-1" });
      mockFetch.mockResolvedValueOnce(
        okJson({
          orgs: [
            {
              id: "o-1",
              slug: "acme",
              name: "Acme",
              plan: "pro",
              status: "awaiting_payment",
              created_at: "",
              updated_at: "",
            },
          ],
        })
      );
      const { unmount } = render(<OrgsPage />);
      await vi.waitFor(() => expect(mockFetch).toHaveBeenCalledTimes(1));
      // Tear down BEFORE the 5s timer fires
      unmount();
      await vi.advanceTimersByTimeAsync(10_000);
      // Fetch count must stay at 1 — the cleanup cleared the timer
      expect(mockFetch).toHaveBeenCalledTimes(1);
    } finally {
      vi.useRealTimers();
    }
  });
});
