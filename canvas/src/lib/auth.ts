/**
 * Canvas-side session detection. Calls /cp/auth/me on the control plane
 * (via same-origin → PLATFORM_URL) and returns the session or null.
 *
 * 401 is the "anonymous" signal and does NOT throw — the caller decides
 * whether to redirect. Network errors do throw so React error boundaries
 * can surface them.
 */
import { PLATFORM_URL } from "./api";

export interface Session {
  user_id: string;
  org_id: string;
  email: string;
}

// Base path prefix for auth endpoints on the control plane.
const AUTH_BASE = "/cp/auth";

/**
 * fetchSession probes /cp/auth/me with the session cookie (credentials:
 * include mandatory cross-origin). Returns the Session on 200, null on
 * 401 (anonymous), throws on anything else so callers don't silently
 * treat a 5xx as "not logged in".
 */
export async function fetchSession(): Promise<Session | null> {
  const res = await fetch(`${PLATFORM_URL}${AUTH_BASE}/me`, {
    credentials: "include",
  });
  if (res.status === 401) return null;
  if (!res.ok) {
    throw new Error(`/cp/auth/me: ${res.status} ${res.statusText}`);
  }
  return res.json();
}

/**
 * redirectToLogin bounces the browser to the control plane's login page
 * with a `return_to` param so the user lands back on the current URL
 * after signup/login completes. Same-origin safety is enforced on the
 * CP side (isSafeReturnTo rejects cross-domain / http / protocol-
 * relative URLs). Uses window.location.href so the full URL including
 * query + hash survives the round trip.
 */
export function redirectToLogin(screenHint: "sign-up" | "sign-in" = "sign-in"): void {
  if (typeof window === "undefined") return;
  // Guard against infinite redirect loop: if we're already on the login
  // page, don't redirect again (each redirect double-encodes return_to
  // until the URL exceeds header limits → 431).
  if (window.location.pathname.startsWith("/cp/auth/")) return;
  const returnTo = window.location.href;
  const path = screenHint === "sign-up" ? "signup" : "login";
  const dest = `${PLATFORM_URL}${AUTH_BASE}/${path}?return_to=${encodeURIComponent(returnTo)}`;
  window.location.href = dest;
}
