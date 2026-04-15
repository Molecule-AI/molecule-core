import type { Metadata } from "next";
import "./globals.css";
import { AuthGate } from "@/components/AuthGate";

export const metadata: Metadata = {
  title: "Molecule AI",
  description: "AI Org Chart Canvas",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body className="bg-zinc-950 text-white">
        {/* AuthGate is a client component; it checks the session on mount
            and bounces anonymous users to the control plane's login page
            when running on a tenant subdomain. Non-SaaS hosts (localhost,
            vercel preview URL, apex) pass through unchanged. */}
        <AuthGate>{children}</AuthGate>
      </body>
    </html>
  );
}
