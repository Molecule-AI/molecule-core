import type { Metadata, Viewport } from "next";
import { headers } from "next/headers";
import "./globals.css";
import { AuthGate } from "@/components/AuthGate";
import { CookieConsent } from "@/components/CookieConsent";

export const metadata: Metadata = {
  title: "Molecule AI",
  description: "AI Org Chart Canvas",
};

export const viewport: Viewport = {
  width: "device-width",
  initialScale: 1,
  maximumScale: 1,
  userScalable: false,
  viewportFit: "cover",
};

export default async function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  // Read the per-request CSP nonce that middleware.ts sets via the
  // `x-nonce` request header. This call is load-bearing for TWO
  // independent reasons:
  //
  //   1. It opts the root layout into dynamic rendering. Without a
  //      `headers()` / `cookies()` / `noStore()` call, Next.js treats
  //      the layout as statically pre-rendered and serves the SAME
  //      HTML for every request — which means the Next.js bootstrap
  //      <script> tags bake into the HTML without any nonce. The
  //      browser then rejects every one with a CSP violation because
  //      the header demands nonce-only script execution.
  //
  //   2. Next.js 15 propagates the nonce to its own generated inline
  //      scripts (the __next_f chunk push frames) ONLY when the header
  //      is actually read via `headers()`. The header's existence on
  //      the request isn't enough — Next.js watches for the read.
  //
  // Keeping the `nonce` variable unused is intentional: we don't need
  // to pass it to any custom <Script nonce={...}> tags right now, the
  // framework takes care of its own bootstrap scripts once the read
  // happens. Destructuring via `await` + `.get()` is the minimum shape
  // Next.js recognizes as "dynamic server-side access".
  await headers();

  return (
    <html lang="en">
      <body className="bg-zinc-950 text-white">
        {/* AuthGate is a client component; it checks the session on mount
            and bounces anonymous users to the control plane's login page
            when running on a tenant subdomain. Non-SaaS hosts (localhost,
            vercel preview URL, apex) pass through unchanged. */}
        <AuthGate>{children}</AuthGate>
        <CookieConsent />
      </body>
    </html>
  );
}
