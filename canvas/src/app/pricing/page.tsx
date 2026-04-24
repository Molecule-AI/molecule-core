import { PricingTable } from "@/components/PricingTable";

/**
 * /pricing — static marketing + plan-selector route.
 *
 * Served from the same canvas deploy as the tenant UI and the apex
 * landing page. Intentionally a server component so the initial HTML
 * renders with full content for SEO; PricingTable is a client
 * component that handles the CTA click + checkout POST.
 *
 * Uses the same dark theme as the canvas so the visual transition
 * from landing → pricing → in-app experience stays cohesive.
 */
export const metadata = {
  title: "Pricing — Molecule AI",
  description:
    "Flat-rate team and org pricing — no per-seat fees. Free to start, $29/month for teams, $99/month for production orgs. Full runtime stack included on every paid tier.",
};

export default function PricingPage() {
  return (
    <main className="min-h-screen bg-zinc-950 text-zinc-100">
      <div className="mx-auto max-w-5xl px-6 pt-20 pb-8 text-center">
        <h1 className="text-5xl font-bold tracking-tight text-white md:text-6xl">
          Pricing
        </h1>
        <p className="mx-auto mt-4 max-w-2xl text-lg text-zinc-300">
          One flat price per org — not per seat. Every paid tier includes the
          full runtime stack. You upgrade for scale, support, and dedicated
          infrastructure.
        </p>
        <p className="mx-auto mt-2 max-w-xl text-sm text-zinc-400">
          5-person team? You pay $29/month — not $200. No seat math, ever.
        </p>
      </div>

      <PricingTable />

      <section className="mx-auto mt-20 max-w-3xl px-6 text-center">
        <h2 className="text-2xl font-semibold text-white">Questions?</h2>
        <p className="mt-2 text-zinc-400">
          We publish the{" "}
          <a
            href="https://github.com/Molecule-AI/molecule-monorepo"
            className="text-blue-400 underline hover:text-blue-300"
          >
            full source on GitHub
          </a>
          {" "}— if something's ambiguous, file an issue or{" "}
          <a
            href="mailto:support@moleculesai.app"
            className="text-blue-400 underline hover:text-blue-300"
          >
            email support
          </a>
          .
        </p>
        <p className="mt-6 text-sm text-zinc-500">
          Prices shown in USD. Flat-rate per org — no per-seat fees on any paid tier.
          Enterprise / self-hosted licensing available — contact us.
        </p>
      </section>

      <footer className="mx-auto mt-20 max-w-5xl border-t border-zinc-800 px-6 py-6 text-center text-sm text-zinc-500">
        <p>
          © {new Date().getFullYear()} Molecule AI, Inc. ·{" "}
          <a href="/legal/terms" className="hover:text-zinc-300">
            Terms
          </a>
          {" "}·{" "}
          <a href="/legal/privacy" className="hover:text-zinc-300">
            Privacy
          </a>
          {" "}·{" "}
          <a href="/legal/dpa" className="hover:text-zinc-300">
            DPA
          </a>
        </p>
      </footer>
    </main>
  );
}
