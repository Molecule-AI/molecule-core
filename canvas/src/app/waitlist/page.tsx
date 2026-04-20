"use client";

// /waitlist — the page shown to users whose email isn't on the
// private-beta allowlist. The CP auth callback redirects here (no
// session cookie set) after rejecting the sign-in.
//
// The page offers a contact form that POSTs to
// /cp/waitlist/request with the user's email + optional name and
// use-case. The CP stores the row in beta_requests; ops triages
// via GET /cp/admin/beta-requests and moves approved emails over
// to beta_allowlist manually.
//
// No session required — the whole point is that the user isn't
// authenticated yet. Per CLAUDE.md privacy rule, we don't read the
// email from a URL query param; user re-enters it into the form.

import { useState } from "react";
import { PLATFORM_URL } from "@/lib/api";

type SubmitState = "idle" | "submitting" | "success" | "dedup" | "error";

interface SubmitResponse {
  ok: boolean;
  id?: string;
  dedup?: boolean;
  error?: string;
}

export default function WaitlistPage() {
  const [email, setEmail] = useState("");
  const [name, setName] = useState("");
  const [useCase, setUseCase] = useState("");
  const [state, setState] = useState<SubmitState>("idle");
  const [errorMsg, setErrorMsg] = useState("");

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    // Client-side sanity. CP enforces the same checks — these exist
    // so the user gets instant feedback without a round trip.
    const trimmed = email.trim();
    if (!trimmed || !trimmed.includes("@")) {
      setState("error");
      setErrorMsg("Please enter a valid email address.");
      return;
    }
    setState("submitting");
    setErrorMsg("");
    try {
      const res = await fetch(`${PLATFORM_URL}/cp/waitlist/request`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          email: trimmed,
          name: name.trim(),
          use_case: useCase.trim(),
        }),
      });
      const body = (await res.json().catch(() => ({}))) as SubmitResponse;
      if (!res.ok || !body.ok) {
        setState("error");
        setErrorMsg(body.error || `Request failed (${res.status}). Please try again.`);
        return;
      }
      // Backend returns dedup=true when this email was already
      // submitted within the last hour. Same 200, softer message.
      setState(body.dedup ? "dedup" : "success");
    } catch (err) {
      setState("error");
      setErrorMsg(err instanceof Error ? err.message : "Network error. Please try again.");
    }
  }

  return (
    <main className="min-h-screen bg-zinc-950 text-zinc-100">
      <div className="mx-auto max-w-xl px-6 py-20">
        <h1 className="text-4xl font-bold tracking-tight text-white md:text-5xl">
          You&rsquo;re on the waitlist
        </h1>
        <p className="mt-4 text-lg text-zinc-300">
          Molecule AI is in private beta while we harden the platform. Tell us
          a bit about yourself and we&rsquo;ll reach out when a spot opens.
        </p>

        {state === "success" && (
          <div
            role="status"
            className="mt-8 rounded-lg border border-emerald-800 bg-emerald-950/50 p-4 text-emerald-200"
          >
            Thanks — your request is in. We&rsquo;ll email{" "}
            <span className="font-mono">{email}</span> when access opens up.
          </div>
        )}

        {state === "dedup" && (
          <div
            role="status"
            className="mt-8 rounded-lg border border-sky-800 bg-sky-950/50 p-4 text-sky-200"
          >
            We already have your request on file for{" "}
            <span className="font-mono">{email}</span>. No need to resubmit —
            we&rsquo;ll be in touch.
          </div>
        )}

        {state !== "success" && state !== "dedup" && (
          <form className="mt-8 space-y-5" onSubmit={onSubmit}>
            <div>
              <label htmlFor="email" className="block text-sm font-medium text-zinc-200">
                Email <span className="text-rose-400">*</span>
              </label>
              <input
                id="email"
                type="email"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="mt-1 block w-full rounded-md border border-zinc-700 bg-zinc-900 px-3 py-2 text-zinc-100 placeholder-zinc-500 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                placeholder="you@company.com"
                autoComplete="email"
              />
            </div>

            <div>
              <label htmlFor="name" className="block text-sm font-medium text-zinc-200">
                Name
              </label>
              <input
                id="name"
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="mt-1 block w-full rounded-md border border-zinc-700 bg-zinc-900 px-3 py-2 text-zinc-100 placeholder-zinc-500 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                placeholder="How should we address you?"
                autoComplete="name"
                maxLength={200}
              />
            </div>

            <div>
              <label htmlFor="use_case" className="block text-sm font-medium text-zinc-200">
                What would you build with this?
              </label>
              <textarea
                id="use_case"
                rows={4}
                value={useCase}
                onChange={(e) => setUseCase(e.target.value)}
                className="mt-1 block w-full rounded-md border border-zinc-700 bg-zinc-900 px-3 py-2 text-zinc-100 placeholder-zinc-500 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                placeholder="Research assistant, customer support automation, internal ops agent…"
                maxLength={500}
              />
              <p className="mt-1 text-xs text-zinc-500">
                Helps us prioritize who to let in first.
              </p>
            </div>

            {state === "error" && errorMsg && (
              <div
                role="alert"
                className="rounded-md border border-rose-800 bg-rose-950/50 px-3 py-2 text-sm text-rose-200"
              >
                {errorMsg}
              </div>
            )}

            <button
              type="submit"
              disabled={state === "submitting"}
              className="inline-flex items-center justify-center rounded-md bg-blue-600 px-5 py-2.5 font-medium text-white transition hover:bg-blue-500 disabled:cursor-not-allowed disabled:bg-blue-900 disabled:text-blue-300"
            >
              {state === "submitting" ? "Submitting…" : "Request access"}
            </button>
          </form>
        )}

        <p className="mt-12 text-sm text-zinc-500">
          Questions? Email{" "}
          <a
            href="mailto:support@moleculesai.app"
            className="text-blue-400 underline hover:text-blue-300"
          >
            support@moleculesai.app
          </a>
          .
        </p>
      </div>
    </main>
  );
}
