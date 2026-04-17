// @vitest-environment jsdom
import { describe, it, expect, vi, afterEach, beforeEach } from "vitest";
import { render, cleanup, act } from "@testing-library/react";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

// ── Mocks (defined before dynamic import of component) ───────────────────────
// Use a function type so TypeScript accepts the mock as callable in vi.mock factories.
// ReturnType<typeof vi.fn> resolves to Mock<Procedure|Constructable> in newer Vitest
// type defs, which TS no longer considers directly callable. Casting to a plain
// function type avoids the TS2348 error while keeping full mock API (mockReturnValue etc.).
let mockFetchSession: ((...args: unknown[]) => unknown) & ReturnType<typeof vi.fn>;
let mockRedirectToLogin: ((...args: unknown[]) => unknown) & ReturnType<typeof vi.fn>;
let mockGetTenantSlug: ((...args: unknown[]) => unknown) & ReturnType<typeof vi.fn>;

beforeEach(() => {
  mockFetchSession = vi.fn() as typeof mockFetchSession;
  mockRedirectToLogin = vi.fn() as typeof mockRedirectToLogin;
  mockGetTenantSlug = vi.fn(() => null) as typeof mockGetTenantSlug;
});

vi.mock("@/lib/auth", () => ({
  fetchSession: (...args: unknown[]) => mockFetchSession(...args),
  redirectToLogin: (...args: unknown[]) => mockRedirectToLogin(...args),
}));

vi.mock("@/lib/tenant", () => ({
  getTenantSlug: (...args: unknown[]) => mockGetTenantSlug(...args),
}));

// Import after mocks are set up
import { AuthGate } from "../AuthGate";

// ─────────────────────────────────────────────────────────────────────────────

describe("AuthGate — loading state", () => {
  it("renders a blank overlay while session fetch is in-flight (prevents flash of unauth'd content)", () => {
    // getTenantSlug returns a slug so the session fetch is triggered
    mockGetTenantSlug.mockReturnValue("acme");
    // fetchSession never resolves — keeps the component in loading state
    mockFetchSession.mockReturnValue(new Promise(() => {}));

    const { container } = render(
      <AuthGate>
        <div data-testid="child">Protected content</div>
      </AuthGate>
    );

    const overlay = container.querySelector(".bg-zinc-950.fixed.inset-0");
    expect(overlay).not.toBeNull();
    expect(overlay?.getAttribute("aria-hidden")).toBe("true");
  });

  it("does not render children while in loading state", () => {
    mockGetTenantSlug.mockReturnValue("acme");
    mockFetchSession.mockReturnValue(new Promise(() => {}));

    const { queryByTestId } = render(
      <AuthGate>
        <div data-testid="child">Protected content</div>
      </AuthGate>
    );

    expect(queryByTestId("child")).toBeNull();
  });
});

describe("AuthGate — non-SaaS / pass-through mode", () => {
  it("renders children immediately when there is no tenant slug", async () => {
    mockGetTenantSlug.mockReturnValue(null);

    let result: ReturnType<typeof render>;
    await act(async () => {
      result = render(
        <AuthGate>
          <div data-testid="child">Protected content</div>
        </AuthGate>
      );
    });

    expect(result!.getByTestId("child")).toBeTruthy();
  });
});

describe("AuthGate — authenticated state", () => {
  it("renders children after a successful session fetch", async () => {
    mockGetTenantSlug.mockReturnValue("acme");
    mockFetchSession.mockResolvedValue({ userId: "u1", email: "a@b.com" });

    let result: ReturnType<typeof render>;
    await act(async () => {
      result = render(
        <AuthGate>
          <div data-testid="child">Protected content</div>
        </AuthGate>
      );
    });

    expect(result!.getByTestId("child")).toBeTruthy();
  });
});

describe("AuthGate — anonymous / redirect state", () => {
  it("calls redirectToLogin when session fetch returns null", async () => {
    mockGetTenantSlug.mockReturnValue("acme");
    mockFetchSession.mockResolvedValue(null);

    await act(async () => {
      render(
        <AuthGate>
          <div data-testid="child">Protected content</div>
        </AuthGate>
      );
    });

    expect(mockRedirectToLogin).toHaveBeenCalledWith("sign-in");
  });
});
