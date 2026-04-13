import type { Metadata } from "next";
import "./globals.css";

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
      <body className="bg-zinc-950 text-white">{children}</body>
    </html>
  );
}
