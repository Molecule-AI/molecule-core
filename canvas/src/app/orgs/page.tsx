"use client";

// /orgs — the post-signup landing page.
//
// The control plane's Callback handler (authorized via WorkOS) redirects
// every new session to APP_URL/orgs after login/signup succeeds. Before
// this route existed that redirect 404'd and new users were stranded.
// Now:
//   - Signed-out browsers are bounced back to /cp/auth/login
//   - Zero-org users see a slug-picker → POST /cp/orgs → refresh
//   - `awaiting_payment` orgs get a "Complete payment" CTA → /pricing
//   - `running` orgs show a link to the tenant URL
//   - `provisioning` / `failed` surface the state so the user knows
//     why their tenant isn't available yet
//
// Everything here is intentionally server-light: one GET /cp/orgs,
// zero WebSocket, no canvas store hydration — the whole point is a
// quick bounce between signup and either Checkout or the tenant UI.

import { useEffect, useState } from "react";
import { fetchSession, redirectToLogin, type Session } from "@/lib/auth";
import { PLATFORM_URL } from "@/lib/api";

type OrgStatus = "awaiting_payment" | "provisioning" | "running" | "failed" | string;

interface Org {
  id: string;
  slug: string;
  name: string;
  plan: string;
  status: OrgStatus;
  created_at: string;
  updated_at: string;
}

export default function OrgsPage() {
  const [session, setSession] = useState<Session | null | "loading">("loading");
  const [orgs, setOrgs] = useState<Org[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [justCheckedOut, setJustCheckedOut] = useState(false);

  useEffect(() => {
    // URLSearchParams is safe on the first render because this component
    // is "use client" — window exists. Clear the flag from the URL so
    // reloading the page doesn't keep showing the banner indefinitely.
    if (typeof window !== "undefined") {
      const params = new URLSearchParams(window.location.search);
      if (params.get("checkout") === "success") {
        setJustCheckedOut(true);
        window.history.replaceState({}, "", window.location.pathname);
      }
    }
  }, []);

  useEffect(() => {
    let cancelled = false;
    let pollTimer: ReturnType<typeof setTimeout> | null = null;

    const fetchOrgs = async () => {
      try {
        const sess = await fetchSession();
        if (cancelled) return;
        if (!sess) {
          redirectToLogin();
          return;
        }
        setSession(sess);
        const res = await fetch(`${PLATFORM_URL}/cp/orgs`, {
          credentials: "include",
          signal: AbortSignal.timeout(15_000),
        });
        if (!res.ok) {
          throw new Error(`GET /cp/orgs: ${res.status}`);
        }
        const body = (await res.json()) as { orgs?: Org[] } | Org[];
        const list = Array.isArray(body) ? body : body.orgs ?? [];
        if (cancelled) return;
        setOrgs(list);

        // Poll while anything is still moving so the user sees the
        // status flip live after a Stripe Checkout. 5s is frequent
        // enough to feel responsive, slow enough to not DoS the CP.
        const stillMoving = list.some(
          (o) => o.status === "provisioning" || o.status === "awaiting_payment"
        );
        if (stillMoving) {
          pollTimer = setTimeout(fetchOrgs, 5_000);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : String(err));
        }
      }
    };

    fetchOrgs();
    return () => {
      cancelled = true;
      if (pollTimer) clearTimeout(pollTimer);
    };
  }, []);

  if (session === "loading" || (orgs === null && error === null)) {
    return <Shell><p className="text-zinc-400">Loading…</p></Shell>;
  }
  if (error) {
    return (
      <Shell>
        <p className="text-red-400">Error: {error}</p>
        <button
          onClick={() => window.location.reload()}
          className="mt-4 rounded bg-zinc-800 px-4 py-2 text-sm text-zinc-200 hover:bg-zinc-700"
        >
          Retry
        </button>
      </Shell>
    );
  }
  if (!orgs || orgs.length === 0) {
    return <EmptyState banner={justCheckedOut ? <CheckoutBanner /> : null} />;
  }
  return (
    <Shell>
      {justCheckedOut && <CheckoutBanner />}
      <ul className="space-y-3">
        {orgs.map((o) => (
          <OrgRow key={o.id} org={o} />
        ))}
      </ul>
      <div className="mt-8 border-t border-zinc-800 pt-6">
        <CreateOrgForm
          onCreated={(slug) => {
            // Refresh the list so the new org appears + its CTA fires.
            window.location.reload();
            void slug;
          }}
        />
      </div>
    </Shell>
  );
}

function CheckoutBanner() {
  return (
    <div className="mb-6 rounded-lg border border-emerald-700 bg-emerald-950 p-4">
      <p className="text-sm text-emerald-200">
        ✓ Payment confirmed. Your workspace is spinning up now — this page
        refreshes automatically when it&apos;s ready.
      </p>
    </div>
  );
}

function Shell({ children }: { children: React.ReactNode }) {
  return (
    <main className="min-h-screen bg-zinc-950 text-zinc-100">
      <div className="mx-auto max-w-2xl px-6 pt-20 pb-12">
        <h1 className="text-3xl font-bold text-white">Your organizations</h1>
        <p className="mt-2 text-zinc-400">
          Each org is an isolated Molecule workspace.
        </p>
        <div className="mt-8">{children}</div>
      </div>
    </main>
  );
}

function OrgRow({ org }: { org: Org }) {
  return (
    <li className="rounded-lg border border-zinc-800 bg-zinc-900 p-4">
      <div className="flex items-center justify-between">
        <div>
          <div className="font-medium text-white">{org.name}</div>
          <div className="text-sm text-zinc-400">
            {org.slug} · <StatusLabel status={org.status} /> · {org.plan || "free"}
          </div>
        </div>
        <OrgCTA org={org} />
      </div>
    </li>
  );
}

function StatusLabel({ status }: { status: OrgStatus }) {
  const cls =
    status === "running"
      ? "text-emerald-400"
      : status === "awaiting_payment"
      ? "text-amber-400"
      : status === "failed"
      ? "text-red-400"
      : "text-sky-400";
  const label =
    status === "awaiting_payment"
      ? "awaiting payment"
      : status;
  return <span className={cls}>{label}</span>;
}

function OrgCTA({ org }: { org: Org }) {
  if (org.status === "running") {
    const host = typeof window !== "undefined" ? window.location.hostname : "moleculesai.app";
    const appDomain = host.endsWith(".moleculesai.app")
      ? host.split(".").slice(-2).join(".")
      : "moleculesai.app";
    const href = `https://${org.slug}.${appDomain}`;
    return (
      <a
        href={href}
        className="rounded bg-emerald-600 px-4 py-2 text-sm font-medium text-white hover:bg-emerald-500"
      >
        Open
      </a>
    );
  }
  if (org.status === "awaiting_payment") {
    return (
      <a
        href={`/pricing?org=${encodeURIComponent(org.slug)}`}
        className="rounded bg-amber-600 px-4 py-2 text-sm font-medium text-white hover:bg-amber-500"
      >
        Complete payment
      </a>
    );
  }
  if (org.status === "failed") {
    return (
      <a
        href="mailto:support@moleculesai.app"
        className="rounded bg-zinc-700 px-4 py-2 text-sm font-medium text-zinc-200 hover:bg-zinc-600"
      >
        Contact support
      </a>
    );
  }
  // provisioning / unknown — non-interactive
  return <span className="text-sm text-zinc-500">{org.status}…</span>;
}

function EmptyState({ banner }: { banner?: React.ReactNode }) {
  return (
    <Shell>
      {banner}
      <p className="text-zinc-300">
        You don&apos;t have any organizations yet. Create one to get started — your
        workspace spins up automatically once billing is set up.
      </p>
      <div className="mt-6">
        <CreateOrgForm
          onCreated={() => {
            window.location.reload();
          }}
        />
      </div>
    </Shell>
  );
}

function CreateOrgForm({ onCreated }: { onCreated: (slug: string) => void }) {
  const [slug, setSlug] = useState("");
  const [name, setName] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setSubmitting(true);
    setErr(null);
    try {
      const res = await fetch(`${PLATFORM_URL}/cp/orgs`, {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ slug, name }),
        signal: AbortSignal.timeout(15_000),
      });
      if (!res.ok) {
        const body = await res.text();
        throw new Error(`${res.status}: ${body}`);
      }
      onCreated(slug);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
      setSubmitting(false);
    }
  }

  return (
    <form onSubmit={submit} className="space-y-3">
      <label className="block">
        <span className="text-sm text-zinc-300">Slug (URL)</span>
        <input
          value={slug}
          onChange={(e) => setSlug(e.target.value.toLowerCase())}
          pattern="^[a-z][a-z0-9-]{2,31}$"
          placeholder="acme"
          required
          className="mt-1 w-full rounded border border-zinc-700 bg-zinc-800 px-3 py-2 text-sm text-zinc-100"
        />
      </label>
      <label className="block">
        <span className="text-sm text-zinc-300">Display name</span>
        <input
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="Acme Corp"
          required
          className="mt-1 w-full rounded border border-zinc-700 bg-zinc-800 px-3 py-2 text-sm text-zinc-100"
        />
      </label>
      {err && <p className="text-sm text-red-400">{err}</p>}
      <button
        type="submit"
        disabled={submitting}
        className="rounded bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-500 disabled:opacity-50"
      >
        {submitting ? "Creating…" : "Create organization"}
      </button>
    </form>
  );
}
