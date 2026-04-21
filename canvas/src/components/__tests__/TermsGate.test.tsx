// @vitest-environment jsdom
/**
 * TermsGate tests — covers terms checking, acceptance flow, error states.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, cleanup, fireEvent, act } from "@testing-library/react";

// ── Mocks ────────────────────────────────────────────────────────────────────

// Mock PLATFORM_URL before importing the component
vi.mock("@/lib/api", () => ({
  PLATFORM_URL: "http://test-platform:8080",
}));

// ── Imports ──────────────────────────────────────────────────────────────────

import { TermsGate } from "../TermsGate";

// ── Helpers ──────────────────────────────────────────────────────────────────

let fetchSpy: ReturnType<typeof vi.fn>;

beforeEach(() => {
  fetchSpy = vi.fn();
  vi.stubGlobal("fetch", fetchSpy);
});

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

// ── Tests ────────────────────────────────────────────────────────────────────

describe("TermsGate — accepted terms", () => {
  it("renders children when terms are accepted", async () => {
    fetchSpy.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({ accepted: true }),
    });

    render(
      <TermsGate>
        <div data-testid="child">Protected content</div>
      </TermsGate>
    );

    await act(async () => {});

    expect(screen.getByTestId("child")).toBeTruthy();
    // Modal should NOT be present
    expect(screen.queryByText("Terms & conditions")).toBeNull();
  });
});

describe("TermsGate — pending terms", () => {
  it("shows terms modal when terms are not accepted", async () => {
    fetchSpy.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({ accepted: false }),
    });

    render(
      <TermsGate>
        <div data-testid="child">Protected content</div>
      </TermsGate>
    );

    await act(async () => {});

    // Children are still rendered (visible behind modal)
    expect(screen.getByTestId("child")).toBeTruthy();
    // Modal should be present
    expect(screen.getByText("Terms & conditions")).toBeTruthy();
    expect(screen.getByText("I agree")).toBeTruthy();
  });

  it("links to Terms of Service and Privacy Policy", async () => {
    fetchSpy.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({ accepted: false }),
    });

    render(
      <TermsGate>
        <div>Content</div>
      </TermsGate>
    );

    await act(async () => {});

    const tosLink = screen.getByText("Terms of Service");
    expect(tosLink.getAttribute("href")).toBe("/legal/terms");
    const privacyLink = screen.getByText("Privacy Policy");
    expect(privacyLink.getAttribute("href")).toBe("/legal/privacy");
  });
});

describe("TermsGate — accept action", () => {
  it("accepts terms and hides modal on success", async () => {
    // First call: terms-status (pending)
    fetchSpy.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({ accepted: false }),
    });

    render(
      <TermsGate>
        <div data-testid="child">Content</div>
      </TermsGate>
    );

    await act(async () => {});

    // Second call: accept-terms (success)
    fetchSpy.mockResolvedValueOnce({
      ok: true,
      status: 200,
      text: async () => "ok",
    });

    const agreeButton = screen.getByText("I agree");
    await act(async () => {
      fireEvent.click(agreeButton);
    });

    // Modal should disappear
    expect(screen.queryByText("Terms & conditions")).toBeNull();
  });
});

describe("TermsGate — 401 (not signed in)", () => {
  it("falls through to accepted state on 401", async () => {
    fetchSpy.mockResolvedValueOnce({
      ok: false,
      status: 401,
    });

    render(
      <TermsGate>
        <div data-testid="child">Content</div>
      </TermsGate>
    );

    await act(async () => {});

    expect(screen.getByTestId("child")).toBeTruthy();
    expect(screen.queryByText("Terms & conditions")).toBeNull();
  });
});

describe("TermsGate — error state", () => {
  it("shows error banner on network failure", async () => {
    fetchSpy.mockRejectedValueOnce(new Error("network timeout"));

    render(
      <TermsGate>
        <div data-testid="child">Content</div>
      </TermsGate>
    );

    await act(async () => {});

    expect(screen.getByText(/Couldn.*t check terms status/)).toBeTruthy();
    expect(screen.getByText(/network timeout/)).toBeTruthy();
  });

  it("shows error banner on non-OK, non-401 response", async () => {
    fetchSpy.mockResolvedValueOnce({
      ok: false,
      status: 500,
    });

    render(
      <TermsGate>
        <div data-testid="child">Content</div>
      </TermsGate>
    );

    await act(async () => {});

    expect(screen.getByText(/terms-status: 500/)).toBeTruthy();
  });
});
