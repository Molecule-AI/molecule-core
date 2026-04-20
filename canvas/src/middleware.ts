import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

/**
 * Build a Content-Security-Policy header value.
 *
 * Production — strict, nonce-based policy:
 *   • script-src uses 'nonce-{nonce}' + 'strict-dynamic': eliminates both
 *     'unsafe-inline' and 'unsafe-eval' (the two directives flagged in #450).
 *     'strict-dynamic' propagates trust to dynamically-loaded Next.js chunks
 *     without needing to enumerate every chunk URL.
 *   • style-src retains 'unsafe-inline': React Flow positions nodes via
 *     element-level style="" attributes which cannot be nonce'd; CSS injection
 *     is significantly lower risk than script injection and is acceptable here.
 *   • object-src / base-uri / frame-ancestors locked to 'none'/'self'.
 *   • upgrade-insecure-requests forces HTTPS on mixed-content.
 *
 * Development — permissive policy:
 *   Next.js HMR and fast-refresh rely on eval() and inline scripts; a strict
 *   nonce policy breaks the dev server. In dev we preserve 'unsafe-inline' and
 *   'unsafe-eval' so the developer experience is unchanged.
 *
 * Exported for unit testing.
 */
export function buildCsp(nonce: string, isDev: boolean): string {
  if (isDev) {
    return [
      "default-src 'self'",
      "script-src 'self' 'unsafe-inline' 'unsafe-eval'",
      "style-src 'self' 'unsafe-inline'",
      "img-src 'self' blob: data:",
      "font-src 'self'",
      "connect-src *",
      "worker-src 'self' blob:",
    ].join("; ") + ";";
  }

  // Canvas makes cross-origin fetches to the control plane for
  // /cp/auth/*, /cp/orgs/*, /cp/billing/* — PLATFORM_URL points at
  // it (baked in at build time via NEXT_PUBLIC_PLATFORM_URL). CSP
  // has to whitelist that origin in connect-src or the browser
  // refuses the fetch with "Refused to connect because it violates
  // the document's Content Security Policy."
  //
  // Self-hosted deployments override PLATFORM_URL at build time and
  // the CSP adjusts automatically — no hardcoded hostname here.
  const platformURL = process.env.NEXT_PUBLIC_PLATFORM_URL ?? "";
  const connectSrcParts = ["'self'", "wss:"];
  if (platformURL) {
    connectSrcParts.push(platformURL);
    // Also allow the wss:// sibling of PLATFORM_URL explicitly.
    // `wss:` scheme-wildcard covers it today but making the exact
    // origin explicit survives a future CSP tightening without
    // silently breaking auth.
    connectSrcParts.push(platformURL.replace(/^http/, "ws"));
  }

  return [
    "default-src 'self'",
    // Nonce-based: no unsafe-inline, no unsafe-eval.
    // 'strict-dynamic' propagates trust from nonce'd bootstrap script to
    // all dynamically-imported Next.js chunks.
    `script-src 'self' 'nonce-${nonce}' 'strict-dynamic'`,
    // unsafe-inline kept for inline style="" attributes used by React Flow.
    "style-src 'self' 'unsafe-inline'",
    "img-src 'self' blob: data:",
    "font-src 'self'",
    "object-src 'none'",
    "base-uri 'self'",
    "form-action 'self'",
    "frame-ancestors 'none'",
    `connect-src ${connectSrcParts.join(" ")}`,
    "worker-src 'self' blob:",
    "upgrade-insecure-requests",
  ].join("; ") + ";";
}

export function middleware(request: NextRequest) {
  // Redirect /en, /zh, etc. locale prefixes back to root
  const pathname = request.nextUrl.pathname;
  const locales =
    /^\/(en|zh|ja|ko|fr|de|es|pt|it|ru|ar|hi|th|vi|nl|sv|da|nb|fi|pl|cs|tr|uk|he|id|ms)(\/|$)/;
  if (locales.test(pathname)) {
    return NextResponse.redirect(new URL("/", request.url));
  }

  // Generate a fresh, per-request nonce.
  // Buffer.from(uuid).toString('base64') gives a URL-safe-ish base64 string
  // that is unique per request and safe to embed in the CSP header value.
  const nonce = Buffer.from(crypto.randomUUID()).toString("base64");
  const isDev = process.env.NODE_ENV === "development" || process.env.CSP_DEV_MODE === "1";
  const csp = buildCsp(nonce, isDev);

  // Forward the nonce to Server Components via a request header so the root
  // layout can pass it to any <Script nonce={nonce}> or <style nonce={nonce}>
  // elements it renders (see app/layout.tsx).
  const requestHeaders = new Headers(request.headers);
  requestHeaders.set("x-nonce", nonce);
  requestHeaders.set("Content-Security-Policy", csp);

  const response = NextResponse.next({
    request: { headers: requestHeaders },
  });
  response.headers.set("Content-Security-Policy", csp);

  return response;
}

export const config = {
  matcher: ["/((?!api|_next/static|_next/image|favicon.ico).*)"],
};
