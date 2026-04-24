"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { createSecret } from "@/lib/api/secrets";

interface Props {
  open: boolean;
  /** Display name of the org template — headline only. */
  orgName: string;
  /** Total workspace count so the header can read "12 workspaces". */
  workspaceCount: number;
  /** Env vars the server has declared MUST be set as global secrets.
   *  Import is disabled until every entry here is configured. */
  requiredEnv: string[];
  /** Env vars the server suggests — import can proceed without them,
   *  but the user sees them listed so they can decide. */
  recommendedEnv: string[];
  /** Names of env vars already configured globally. Used to strike
   *  through entries the user has already set up in another
   *  session. Passed in rather than queried inside the modal so the
   *  parent can refresh after each save without prop-driven effects. */
  configuredKeys: Set<string>;
  /** Called after a successful secret save so the parent can refresh
   *  `configuredKeys`. */
  onSecretSaved: () => void;
  /** User clicked Import with all required envs satisfied. */
  onProceed: () => void;
  /** User dismissed the modal. Import is NOT fired. */
  onCancel: () => void;
}

interface DraftEntry {
  key: string;
  value: string;
  saving: boolean;
  error: string | null;
}

/**
 * OrgImportPreflightModal
 * -----------------------
 * Two-tier env preflight before POST /org/import:
 *
 *   - REQUIRED section (red, blocking) — every entry MUST be configured
 *     globally before the Import button enables. Matches the server-
 *     side preflight that would 412 the import anyway.
 *
 *   - RECOMMENDED section (yellow, non-blocking) — listed so the user
 *     can add them if they want the full experience, but the Import
 *     button stays enabled regardless.
 *
 * Saving goes to the GLOBAL secrets endpoint (PUT /settings/secrets)
 * because org-level templates deploy shared resources. Per-workspace
 * overrides still work via the Config tab on an individual node
 * after import. The modal does NOT enable Import the moment a key is
 * typed — only after it saves successfully (so a half-entered token
 * can't proceed and then fail at container-start time instead).
 */
export function OrgImportPreflightModal({
  open,
  orgName,
  workspaceCount,
  requiredEnv,
  recommendedEnv,
  configuredKeys,
  onSecretSaved,
  onProceed,
  onCancel,
}: Props) {
  const [drafts, setDrafts] = useState<Record<string, DraftEntry>>({});

  // Seed a draft entry per declared key the first time the modal
  // opens. Entries persist across `configuredKeys` changes so a mid-
  // save recheck doesn't wipe what the user typed.
  //
  // Dep: dervie a STABLE string from the env-name lists rather than
  // the array refs themselves. The parent computes
  // `preflight.org.required_env ?? []`, which produces a fresh []
  // identity on every re-render (e.g. when refreshConfiguredKeys
  // bumps state); depending on the array refs would re-fire the
  // effect on every parent render and mask any future edit that
  // drops the `if (!next[k])` guard as a silent input-reset bug.
  const envKeysSignature = useMemo(
    () => [...requiredEnv, ...recommendedEnv].sort().join("|"),
    [requiredEnv, recommendedEnv],
  );
  useEffect(() => {
    if (!open) return;
    setDrafts((prev) => {
      const next = { ...prev };
      for (const k of [...requiredEnv, ...recommendedEnv]) {
        if (!next[k]) {
          next[k] = { key: k, value: "", saving: false, error: null };
        }
      }
      return next;
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, envKeysSignature]);

  const missingRequired = useMemo(
    () => requiredEnv.filter((k) => !configuredKeys.has(k)),
    [requiredEnv, configuredKeys],
  );
  const missingRecommended = useMemo(
    () => recommendedEnv.filter((k) => !configuredKeys.has(k)),
    [recommendedEnv, configuredKeys],
  );
  const canProceed = missingRequired.length === 0;

  const saveOne = useCallback(
    async (key: string) => {
      // Functional setter throughout so two near-simultaneous saves
      // don't have the second one's call see a stale snapshot captured
      // before the first save's setState landed. Read the current
      // value AND write the `saving` flag in a single transition
      // rather than reading from closure-scoped `drafts`.
      let startValue = "";
      setDrafts((d) => {
        const current = d[key];
        if (!current || !current.value.trim()) return d;
        startValue = current.value;
        return { ...d, [key]: { ...current, saving: true, error: null } };
      });
      if (!startValue.trim()) return;
      try {
        await createSecret("global", key, startValue);
        setDrafts((d) => ({
          ...d,
          [key]: { ...d[key], value: "", saving: false, error: null },
        }));
        // Let the parent refresh configuredKeys so the strike-through
        // updates and canProceed recomputes.
        onSecretSaved();
      } catch (e) {
        setDrafts((d) => ({
          ...d,
          [key]: {
            ...d[key],
            saving: false,
            error: e instanceof Error ? e.message : "Save failed",
          },
        }));
      }
    },
    [onSecretSaved],
  );

  if (!open) return null;

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="org-preflight-title"
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/70"
      onClick={onCancel}
    >
      <div
        className="w-[560px] max-h-[85vh] overflow-auto rounded-xl bg-zinc-900 border border-zinc-700 shadow-2xl"
        onClick={(e) => e.stopPropagation()}
      >
        <header className="px-5 py-4 border-b border-zinc-800">
          <h2 id="org-preflight-title" className="text-sm font-semibold text-zinc-100">
            Deploy {orgName}
          </h2>
          <p className="mt-0.5 text-[11px] text-zinc-500">
            {workspaceCount} workspace{workspaceCount === 1 ? "" : "s"}.
            Review the credentials needed before import.
          </p>
        </header>

        <section className="p-5 space-y-5">
          {requiredEnv.length > 0 && (
            <EnvList
              tone="required"
              title="Required"
              subtitle="Import is blocked until every key below is saved globally."
              entries={requiredEnv}
              configuredKeys={configuredKeys}
              drafts={drafts}
              onChange={(key, value) =>
                setDrafts((d) => ({ ...d, [key]: { ...d[key], value } }))
              }
              onSave={saveOne}
            />
          )}
          {recommendedEnv.length > 0 && (
            <EnvList
              tone="recommended"
              title="Recommended"
              subtitle="Not required, but some features degrade without them. Add them now for the best experience."
              entries={recommendedEnv}
              configuredKeys={configuredKeys}
              drafts={drafts}
              onChange={(key, value) =>
                setDrafts((d) => ({ ...d, [key]: { ...d[key], value } }))
              }
              onSave={saveOne}
            />
          )}
          {requiredEnv.length === 0 && recommendedEnv.length === 0 && (
            <p className="text-[12px] text-zinc-400">
              No additional credentials required for this template.
            </p>
          )}
        </section>

        <footer className="px-5 py-3 border-t border-zinc-800 flex items-center justify-between">
          <button
            type="button"
            onClick={onCancel}
            className="px-3 py-1.5 text-[11px] rounded bg-zinc-800 hover:bg-zinc-700 text-zinc-300"
          >
            Cancel
          </button>
          <div className="flex items-center gap-2">
            {missingRecommended.length > 0 && canProceed && (
              <span className="text-[10px] text-amber-400/90">
                {missingRecommended.length} recommended key
                {missingRecommended.length === 1 ? "" : "s"} still unset
              </span>
            )}
            <button
              type="button"
              onClick={onProceed}
              disabled={!canProceed}
              className="px-4 py-1.5 text-[11px] font-semibold rounded bg-blue-600 hover:bg-blue-500 text-white disabled:bg-zinc-700 disabled:text-zinc-500 disabled:cursor-not-allowed"
            >
              Import
            </button>
          </div>
        </footer>
      </div>
    </div>
  );
}

interface EnvListProps {
  tone: "required" | "recommended";
  title: string;
  subtitle: string;
  entries: string[];
  configuredKeys: Set<string>;
  drafts: Record<string, DraftEntry>;
  onChange: (key: string, value: string) => void;
  onSave: (key: string) => void;
}

function EnvList({
  tone,
  title,
  subtitle,
  entries,
  configuredKeys,
  drafts,
  onChange,
  onSave,
}: EnvListProps) {
  const accent =
    tone === "required"
      ? "border-red-800/60 bg-red-950/20"
      : "border-amber-800/50 bg-amber-950/15";
  const headerColor =
    tone === "required" ? "text-red-300" : "text-amber-300";

  return (
    <div className={`rounded-lg border ${accent} p-3`}>
      <h3 className={`text-[11px] font-semibold uppercase tracking-wide ${headerColor}`}>
        {title}
      </h3>
      <p className="mt-0.5 mb-2 text-[10px] text-zinc-400">{subtitle}</p>
      <ul className="space-y-2">
        {entries.map((k) => {
          const configured = configuredKeys.has(k);
          const d = drafts[k];
          return (
            <li
              key={k}
              className="flex items-center gap-2 rounded bg-zinc-900/70 border border-zinc-800 px-2 py-1.5"
            >
              <code
                className={`text-[11px] font-mono flex-1 ${
                  configured ? "text-zinc-500 line-through" : "text-zinc-200"
                }`}
              >
                {k}
              </code>
              {configured ? (
                <span className="text-[10px] text-emerald-400">✓ set</span>
              ) : (
                <>
                  <input
                    type="password"
                    aria-label={`Value for ${k}`}
                    placeholder="paste value"
                    value={d?.value ?? ""}
                    onChange={(e) => onChange(k, e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter") {
                        e.preventDefault();
                        onSave(k);
                      }
                    }}
                    disabled={d?.saving}
                    className="flex-1 px-2 py-1 rounded bg-zinc-800 border border-zinc-700 text-[11px] text-zinc-200 focus:outline-none focus:border-blue-500 disabled:opacity-50"
                  />
                  <button
                    type="button"
                    onClick={() => onSave(k)}
                    disabled={d?.saving || !d?.value.trim()}
                    className="px-2 py-1 text-[10px] rounded bg-blue-600 hover:bg-blue-500 text-white disabled:opacity-40 disabled:cursor-not-allowed"
                  >
                    {d?.saving ? "…" : "Save"}
                  </button>
                </>
              )}
              {d?.error && (
                <span className="text-[9px] text-red-400 basis-full pl-1">
                  {d.error}
                </span>
              )}
            </li>
          );
        })}
      </ul>
    </div>
  );
}
