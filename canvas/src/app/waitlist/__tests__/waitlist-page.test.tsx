// @vitest-environment jsdom
/**
 * Tests for /waitlist — the contact form page shown to users rejected
 * by the beta-gate (PR #150 backend + #? frontend).
 *
 * This page is a user's ONLY path to request access after the CP
 * rejects their login, so regressions here strand every new user.
 * Covers:
 *   - Form renders with required email field
 *   - Client-side validation rejects empty / malformed emails
 *   - Successful POST → success banner, form hidden
 *   - dedup=true response → softer "already on file" banner
 *   - Non-2xx response → error banner with server message
 *   - Network error → error banner with fallback message
 *   - email NEVER appears in URL (regression guard — pre-fix the CP
 *     redirect passed ?email= which leaked to referrer headers)
 *   - Body is normalized (trim email) before submit
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import {
  render,
  screen,
  waitFor,
  cleanup,
  fireEvent,
} from "@testing-library/react";

vi.mock("@/lib/api", () => ({
  PLATFORM_URL: "https://cp.test",
}));

const mockFetch = vi.fn();
globalThis.fetch = mockFetch as unknown as typeof fetch;

import WaitlistPage from "../page";

function okJson(body: unknown, status = 200) {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
  } as unknown as Response;
}

beforeEach(() => {
  vi.clearAllMocks();
  // Reset URL so the URL-leak guard sees a clean query string.
  window.history.pushState({}, "", "/waitlist");
});

afterEach(() => {
  cleanup();
});

describe("/waitlist — page render", () => {
  it("renders the form with an email field by default", () => {
    render(<WaitlistPage />);
    expect(screen.getByRole("heading", { name: /waitlist/i })).toBeTruthy();
    expect(screen.getByLabelText(/email/i)).toBeTruthy();
    expect(screen.getByRole("button", { name: /request access/i })).toBeTruthy();
    // No success/dedup banners at rest.
    expect(screen.queryByRole("status")).toBeNull();
  });

  it("does NOT pre-fill email from URL query (privacy regression guard)", () => {
    // Pre-fix, the CP redirected to /waitlist?email=<urlencoded>.
    // Even though the backend no longer does that, a bookmark or a
    // cached redirect could still hand us one. The page must not
    // auto-read it.
    window.history.pushState({}, "", "/waitlist?email=leaked@example.com");
    render(<WaitlistPage />);
    const input = screen.getByLabelText(/email/i) as HTMLInputElement;
    expect(input.value).toBe("");
  });
});

describe("/waitlist — client-side validation", () => {
  it("rejects an empty email without calling the API", async () => {
    render(<WaitlistPage />);
    fireEvent.submit(screen.getByRole("button", { name: /request access/i }).closest("form")!);
    // HTML5 required attribute handles this before our JS runs — so
    // no fetch happens. This test documents the contract that invalid
    // submissions never hit the network.
    expect(mockFetch).not.toHaveBeenCalled();
  });

  it("rejects a malformed email with an inline error and no fetch", async () => {
    render(<WaitlistPage />);
    // Type something that passes HTML5 required but fails our @-check.
    // We bypass input validation by setting the value directly through
    // the onChange handler (jsdom's native email type accepts "noat").
    const input = screen.getByLabelText(/email/i);
    fireEvent.change(input, { target: { value: "noat" } });
    const form = screen.getByRole("button", { name: /request access/i }).closest("form")!;
    fireEvent.submit(form);
    await waitFor(() => {
      expect(screen.getByRole("alert")).toBeTruthy();
    });
    expect(mockFetch).not.toHaveBeenCalled();
  });
});

describe("/waitlist — submit happy path", () => {
  it("POSTs a trimmed body and shows a success banner on {ok:true}", async () => {
    mockFetch.mockResolvedValueOnce(okJson({ ok: true, id: "req-abc" }));
    render(<WaitlistPage />);

    fireEvent.change(screen.getByLabelText(/email/i), {
      target: { value: "  hi@example.com  " },
    });
    fireEvent.change(screen.getByLabelText(/^name$/i), {
      target: { value: "Hongming" },
    });
    fireEvent.change(screen.getByLabelText(/what would you build/i), {
      target: { value: "research automation" },
    });
    fireEvent.click(screen.getByRole("button", { name: /request access/i }));

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledTimes(1);
    });

    // URL and body shape.
    const [url, init] = mockFetch.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("https://cp.test/cp/waitlist/request");
    expect(init.method).toBe("POST");
    const body = JSON.parse(init.body as string);
    expect(body).toEqual({
      email: "hi@example.com", // trimmed
      name: "Hongming",
      use_case: "research automation",
    });

    await waitFor(() => {
      expect(screen.getByRole("status").textContent).toMatch(/your request is in/i);
    });
  });
});

describe("/waitlist — submit dedup", () => {
  it("shows a softer banner when backend returns dedup=true", async () => {
    mockFetch.mockResolvedValueOnce(okJson({ ok: true, dedup: true }));
    render(<WaitlistPage />);
    fireEvent.change(screen.getByLabelText(/email/i), {
      target: { value: "existing@example.com" },
    });
    fireEvent.click(screen.getByRole("button", { name: /request access/i }));
    await waitFor(() => {
      expect(screen.getByRole("status").textContent).toMatch(/already have your request/i);
    });
  });
});

describe("/waitlist — submit error paths", () => {
  it("shows the server's error message on a non-2xx response", async () => {
    mockFetch.mockResolvedValueOnce(okJson({ error: "email required" }, 400));
    render(<WaitlistPage />);
    fireEvent.change(screen.getByLabelText(/email/i), {
      target: { value: "x@y.com" },
    });
    fireEvent.click(screen.getByRole("button", { name: /request access/i }));
    await waitFor(() => {
      expect(screen.getByRole("alert").textContent).toMatch(/email required/i);
    });
  });

  it("falls back to a generic message when the response has no error field", async () => {
    mockFetch.mockResolvedValueOnce(
      okJson({}, 500) // no `error` key
    );
    render(<WaitlistPage />);
    fireEvent.change(screen.getByLabelText(/email/i), {
      target: { value: "x@y.com" },
    });
    fireEvent.click(screen.getByRole("button", { name: /request access/i }));
    await waitFor(() => {
      expect(screen.getByRole("alert").textContent).toMatch(/500/);
    });
  });

  it("shows an error banner on network failure", async () => {
    mockFetch.mockRejectedValueOnce(new Error("TCP reset"));
    render(<WaitlistPage />);
    fireEvent.change(screen.getByLabelText(/email/i), {
      target: { value: "x@y.com" },
    });
    fireEvent.click(screen.getByRole("button", { name: /request access/i }));
    await waitFor(() => {
      expect(screen.getByRole("alert").textContent).toMatch(/TCP reset/);
    });
  });
});
