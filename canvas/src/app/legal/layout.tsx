import Link from "next/link";

/**
 * Shared layout for /legal/* pages (privacy, terms, DPA, etc.).
 *
 * Renders a centered, readable column on the dark zinc-950 background
 * with a back-link to the home page. All legal pages are server
 * components — no client JS needed.
 */
export default function LegalLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <main className="min-h-screen bg-zinc-950 text-zinc-100">
      <div className="mx-auto max-w-3xl px-4 sm:px-6 pt-8 sm:pt-12 pb-16 sm:pb-20">
        <nav className="mb-10">
          <Link
            href="/"
            className="inline-flex items-center gap-1.5 text-sm text-zinc-400 hover:text-zinc-200 transition-colors"
          >
            <span aria-hidden="true">&larr;</span> Back to Molecule AI
          </Link>
        </nav>

        <article className="prose-invert prose-zinc max-w-none">
          {children}
        </article>

        <footer className="mt-16 border-t border-zinc-800 pt-6 text-center text-sm text-zinc-500">
          <p>
            &copy; {new Date().getFullYear()} Molecule AI, Inc. &middot;{" "}
            <Link href="/legal/terms" className="hover:text-zinc-300">
              Terms
            </Link>{" "}
            &middot;{" "}
            <Link href="/legal/privacy" className="hover:text-zinc-300">
              Privacy
            </Link>
          </p>
        </footer>
      </div>
    </main>
  );
}
