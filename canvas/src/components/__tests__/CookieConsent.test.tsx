// @vitest-environment jsdom
import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, cleanup } from "@testing-library/react";
import { CookieConsent, hasConsent } from "../CookieConsent";

const STORAGE_KEY = "molecule_cookie_consent";

// These tests lock the privacy-preserving default: the banner appears on
// first visit, clicking either button records a decision, and subsequent
// renders skip the banner until the policy version changes.

beforeEach(() => {
  window.localStorage.clear();
});

afterEach(() => {
  cleanup();
  window.localStorage.clear();
});

describe("CookieConsent", () => {
  it("renders the banner when no decision is stored", () => {
    render(<CookieConsent />);
    expect(screen.getByRole("dialog")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Accept all" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Necessary only" })).toBeTruthy();
  });

  it("stores 'accepted' and hides the banner when user clicks Accept all", () => {
    render(<CookieConsent />);
    fireEvent.click(screen.getByRole("button", { name: "Accept all" }));
    expect(screen.queryByRole("dialog")).toBeNull();

    const raw = window.localStorage.getItem(STORAGE_KEY);
    expect(raw).not.toBeNull();
    const parsed = JSON.parse(raw!);
    expect(parsed.decision).toBe("accepted");
    expect(parsed.version).toBe(1);
    expect(typeof parsed.decidedAt).toBe("string");
  });

  it("stores 'rejected' and hides the banner when user clicks Necessary only", () => {
    render(<CookieConsent />);
    fireEvent.click(screen.getByRole("button", { name: "Necessary only" }));
    expect(screen.queryByRole("dialog")).toBeNull();

    const parsed = JSON.parse(window.localStorage.getItem(STORAGE_KEY)!);
    expect(parsed.decision).toBe("rejected");
  });

  it("does NOT render the banner when a current-version decision is already stored", () => {
    window.localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({ decision: "accepted", decidedAt: new Date().toISOString(), version: 1 }),
    );
    render(<CookieConsent />);
    expect(screen.queryByRole("dialog")).toBeNull();
  });

  it("re-prompts when the stored decision is on an older policy version", () => {
    window.localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({ decision: "accepted", decidedAt: new Date().toISOString(), version: 0 }),
    );
    render(<CookieConsent />);
    expect(screen.getByRole("dialog")).toBeTruthy();
  });

  it("re-prompts when localStorage contains invalid JSON", () => {
    window.localStorage.setItem(STORAGE_KEY, "{not json");
    render(<CookieConsent />);
    expect(screen.getByRole("dialog")).toBeTruthy();
  });

  it("exposes a privacy-policy link with target='_blank'", () => {
    render(<CookieConsent />);
    const link = screen.getByRole("link", { name: /privacy policy/i });
    expect(link).toBeTruthy();
    expect(link.getAttribute("target")).toBe("_blank");
    expect(link.getAttribute("rel")).toContain("noreferrer");
  });

  it("uses role=dialog with aria-labelledby and aria-describedby for screen readers", () => {
    render(<CookieConsent />);
    const dialog = screen.getByRole("dialog");
    expect(dialog.getAttribute("aria-labelledby")).toBe("cookie-consent-title");
    expect(dialog.getAttribute("aria-describedby")).toBe("cookie-consent-body");
  });
});

describe("hasConsent", () => {
  it("returns false when no decision is stored (privacy-preserving default)", () => {
    expect(hasConsent()).toBe(false);
  });

  it("returns true only when the stored decision is 'accepted'", () => {
    window.localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({ decision: "accepted", decidedAt: new Date().toISOString(), version: 1 }),
    );
    expect(hasConsent()).toBe(true);
  });

  it("returns false when stored decision is 'rejected'", () => {
    window.localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({ decision: "rejected", decidedAt: new Date().toISOString(), version: 1 }),
    );
    expect(hasConsent()).toBe(false);
  });

  it("returns false when stored decision is from an older policy version", () => {
    window.localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({ decision: "accepted", decidedAt: new Date().toISOString(), version: 0 }),
    );
    expect(hasConsent()).toBe(false);
  });
});
