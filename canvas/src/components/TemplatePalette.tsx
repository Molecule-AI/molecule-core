"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import { api } from "@/lib/api";
import { useCanvasStore } from "@/store/canvas";
import type { WorkspaceData } from "@/store/socket";
import { checkDeploySecrets, type PreflightResult, type ModelSpec } from "@/lib/deploy-preflight";
import { MissingKeysModal } from "./MissingKeysModal";
import { ConfirmDialog } from "./ConfirmDialog";
import { Spinner } from "./Spinner";
import { showToast } from "./Toaster";
import { TIER_CONFIG } from "@/lib/design-tokens";

interface Template {
  id: string;
  name: string;
  description: string;
  tier: number;
  runtime?: string;
  model: string;
  models?: ModelSpec[];
  /** AND-required env vars declared at runtime_config.required_env. */
  required_env?: string[];
  skills: string[];
  skill_count: number;
}

export interface OrgTemplate {
  dir: string;
  name: string;
  description: string;
  workspaces: number;
}

/** Fetch the list of org templates from the platform. Returns [] on error
 * so the UI shows the empty state instead of crashing. */
export async function fetchOrgTemplates(): Promise<OrgTemplate[]> {
  try {
    return await api.get<OrgTemplate[]>("/org/templates");
  } catch {
    return [];
  }
}

/** Server response from POST /org/import. The handler returns 207
 * (StatusMultiStatus) with a populated `error` field when only some of
 * the workspaces in the tree could be created — the HTTP status alone
 * isn't enough to detect a partial failure. */
interface OrgImportResponse {
  org: string;
  workspaces: Array<{ id: string; name: string }>;
  count: number;
  error?: string;
}

/** Import an org template by directory name. Throws on platform error
 * so the caller can surface the message in its error state. Also throws
 * on 2xx-with-error-body (StatusMultiStatus) — without this check a
 * partial failure (e.g. first workspace INSERT fails, 0 created)
 * appears as a green success toast and the user sees no canvas update.
 *
 * Uses a long timeout because createWorkspaceTree paces sibling DB
 * inserts by `workspaceCreatePacingMs` (2s) to avoid overwhelming
 * Docker — a 15-workspace tree sleeps ~28s in the handler alone,
 * which blows past the default 15s and makes the client report a
 * spurious "signal timed out" error even though the server finished
 * successfully. 2min covers trees up to ~60 workspaces. */
const ORG_IMPORT_TIMEOUT_MS = 120_000;

export async function importOrgTemplate(dir: string): Promise<OrgImportResponse> {
  const resp = await api.post<OrgImportResponse>(
    "/org/import",
    { dir },
    { timeoutMs: ORG_IMPORT_TIMEOUT_MS },
  );
  if (resp && resp.error) {
    throw new Error(`${resp.error} (created ${resp.count ?? 0} workspaces)`);
  }
  return resp;
}

/**
 * Section listing org templates (multi-workspace hierarchies). Click "Import"
 * to instantiate the entire tree via `POST /org/import { dir }`. PLAN.md §20.3.
 *
 * Exported separately so the org import flow has a focused unit-test surface
 * without re-rendering the full palette.
 */
export function OrgTemplatesSection() {
  const [orgs, setOrgs] = useState<OrgTemplate[]>([]);
  const [loading, setLoading] = useState(false);
  const [importing, setImporting] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  // Collapsed by default — org templates are multi-workspace imports
  // that most new users don't reach for first. Keeping them
  // expand-on-demand frees ~400 px of vertical space for the
  // individual workspace templates above, which is the primary
  // deploy path. The count in the header still makes discovery
  // obvious: "Org Templates (4) ▸".
  const [expanded, setExpanded] = useState(false);

  const loadOrgs = useCallback(async () => {
    setLoading(true);
    setOrgs(await fetchOrgTemplates());
    setLoading(false);
  }, []);

  useEffect(() => {
    loadOrgs();
  }, [loadOrgs]);

  const handleImport = async (org: OrgTemplate) => {
    setImporting(org.dir);
    setError(null);
    try {
      await importOrgTemplate(org.dir);
      // Hydrate is the safety net for the "WS is offline" case —
      // without live events the canvas stays empty. But calling it
      // immediately wipes the org-deploy animation (hydrate rebuilds
      // the node array from scratch, dropping the spawn / shimmer
      // classes and position tweens). So:
      //   1. If the number of nodes on the canvas already matches
      //      (or exceeds) the template's workspace count, WS
      //      delivered everything — skip hydrate.
      //   2. Otherwise, wait a short window to let any in-flight WS
      //      events land, then hydrate only if still behind.
      const expectedCount = org.workspaces;
      // Nodes transition through WORKSPACE_REMOVED which physically
      // drops them from the store — there is no "removed" status in
      // WorkspaceNodeData — so a simple length check is enough here.
      const hasAll = () => useCanvasStore.getState().nodes.length >= expectedCount;
      if (!hasAll()) {
        await new Promise((r) => setTimeout(r, 1500));
      }
      if (!hasAll()) {
        try {
          const workspaces = await api.get<WorkspaceData[]>("/workspaces");
          useCanvasStore.getState().hydrate(workspaces);
        } catch {
          // WS (if alive) or the next health-check cycle will
          // eventually pick the new workspaces up.
        }
      }
      showToast(`Imported "${org.name || org.dir}" (${org.workspaces} workspaces)`, "success");
    } catch (e) {
      const msg = e instanceof Error ? e.message : "Import failed";
      setError(msg);
      showToast(`Import failed: ${msg}`, "error");
    } finally {
      setImporting(null);
    }
  };

  return (
    <div className="space-y-2" data-testid="org-templates-section">
      <div className="flex items-center justify-between">
        <button
          type="button"
          onClick={() => setExpanded((v) => !v)}
          aria-expanded={expanded}
          aria-controls="org-templates-body"
          className="flex items-center gap-1.5 text-[10px] uppercase tracking-wide text-zinc-500 hover:text-zinc-300 font-semibold transition-colors"
        >
          <span
            aria-hidden="true"
            className={`inline-block text-[8px] transition-transform duration-150 ${expanded ? "rotate-90" : ""}`}
          >
            ▶
          </span>
          Org Templates
          {orgs.length > 0 && (
            <span className="text-zinc-600 normal-case tracking-normal">
              ({orgs.length})
            </span>
          )}
        </button>
        <button
          onClick={loadOrgs}
          aria-label="Refresh org templates"
          className="text-[10px] text-zinc-500 hover:text-zinc-300"
        >
          ↻
        </button>
      </div>

      {expanded && (
        <div id="org-templates-body" className="space-y-2">
      {loading && (
        <div role="status" aria-live="polite" className="flex items-center gap-1.5 text-[10px] text-zinc-500">
          <Spinner size="sm" />
          Loading…
        </div>
      )}

      {!loading && orgs.length === 0 && (
        <div className="text-[10px] text-zinc-500">
          No org templates in <code>org-templates/</code>
        </div>
      )}

      {error && (
        <div className="px-2 py-1 bg-red-950/40 border border-red-800/50 rounded text-[10px] text-red-400">
          {error}
        </div>
      )}

      {orgs.map((o) => {
        const isImporting = importing === o.dir;
        return (
          <div
            key={o.dir}
            className="bg-zinc-900/50 border border-zinc-800/60 rounded-xl p-3 hover:border-zinc-700/60 transition-all"
          >
            <div className="flex items-center justify-between mb-1">
              <span className="text-[12px] font-semibold text-zinc-200 truncate">
                {o.name || o.dir}
              </span>
              <span className="text-[9px] font-mono text-sky-400 bg-sky-950/40 px-1.5 py-0.5 rounded-md shrink-0">
                {o.workspaces} workspaces
              </span>
            </div>
            {o.description && (
              <p className="text-[10px] text-zinc-500 mb-2.5 line-clamp-2 leading-relaxed">
                {o.description}
              </p>
            )}
            <button
              onClick={() => handleImport(o)}
              disabled={isImporting}
              className="w-full px-2 py-1.5 bg-blue-600/20 hover:bg-blue-600/30 border border-blue-500/30 rounded-lg text-[10px] text-blue-300 font-medium transition-colors disabled:opacity-50"
            >
              {isImporting ? "Importing…" : "Import org"}
            </button>
          </div>
        );
      })}
        </div>
      )}
    </div>
  );
}

const TIER_LABELS = TIER_CONFIG;

function ImportAgentButton({ onImported }: { onImported: () => void }) {
  const [importing, setImporting] = useState(false);
  const [notice, setNotice] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleFiles = async (fileList: FileList) => {
    setImporting(true);
    try {
      const files: Record<string, string> = {};
      let agentName = "";

      for (const file of Array.from(fileList)) {
        // webkitRelativePath gives us "folder/file.md"
        const path = file.webkitRelativePath || file.name;
        // Strip the top-level folder name
        const parts = path.split("/");
        if (!agentName && parts.length > 1) {
          agentName = parts[0];
        }
        const relPath = parts.length > 1 ? parts.slice(1).join("/") : parts[0];

        // Only import text files
        if (file.size > 1_000_000) continue; // skip files > 1MB
        try {
          const content = await file.text();
          files[relPath] = content;
        } catch {
          // Skip binary files
        }
      }

      if (Object.keys(files).length === 0) {
        setNotice("No files found in the selected folder");
        return;
      }

      const name = agentName || "Imported Agent";
      await api.post("/templates/import", { name, files });
      onImported();
    } catch (e) {
      setNotice(e instanceof Error ? e.message : "Import failed");
    } finally {
      setImporting(false);
    }
  };

  return (
    <div>
      <input
        ref={fileInputRef}
        type="file"
        // @ts-expect-error webkitdirectory is non-standard but widely supported
        webkitdirectory=""
        multiple
        className="hidden"
        onChange={(e) => e.target.files && handleFiles(e.target.files)}
      />
      <button
        onClick={() => fileInputRef.current?.click()}
        disabled={importing}
        className="w-full px-3 py-2 bg-blue-600/20 hover:bg-blue-600/30 border border-blue-500/30 rounded-lg text-[11px] text-blue-300 font-medium transition-colors disabled:opacity-50"
      >
        {importing ? "Importing..." : "Import Agent Folder"}
      </button>
      <ConfirmDialog
        open={!!notice}
        title="Import"
        message={notice ?? ""}
        confirmLabel="OK"
        confirmVariant="primary"
        singleButton
        onConfirm={() => setNotice(null)}
        onCancel={() => setNotice(null)}
      />
    </div>
  );
}

export function TemplatePalette() {
  const [open, setOpen] = useState(false);
  // Publish palette-open state to the canvas store so Legend (and any
  // future floating left-bottom UI) can shift right to avoid being
  // hidden behind the 280 px palette drawer.
  const setTemplatePaletteOpen = useCanvasStore((s) => s.setTemplatePaletteOpen);
  useEffect(() => {
    setTemplatePaletteOpen(open);
  }, [open, setTemplatePaletteOpen]);

  const [templates, setTemplates] = useState<Template[]>([]);
  const [loading, setLoading] = useState(false);
  const [creating, setCreating] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Missing keys modal state
  const [missingKeysInfo, setMissingKeysInfo] = useState<{
    template: Template;
    preflight: PreflightResult;
  } | null>(null);

  const loadTemplates = useCallback(async () => {
    setLoading(true);
    try {
      const data = await api.get<Template[]>("/templates");
      setTemplates(data);
    } catch {
      setTemplates([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (open) loadTemplates();
  }, [open, loadTemplates]);

  /** Resolve runtime from template ID (e.g., "langgraph", "claude-code-default" → "claude-code") */
  const resolveRuntime = (templateId: string): string => {
    const runtimeMap: Record<string, string> = {
      langgraph: "langgraph",
      "claude-code-default": "claude-code",
      openclaw: "openclaw",
      deepagents: "deepagents",
      crewai: "crewai",
      autogen: "autogen",
    };
    return runtimeMap[templateId] ?? templateId.replace(/-default$/, "");
  };

  /** Actually execute the deploy API call */
  const executeDeploy = useCallback(async (template: Template) => {
    setCreating(template.id);
    setError(null);
    try {
      await api.post("/workspaces", {
        name: template.name,
        template: template.id,
        tier: template.tier,
        canvas: {
          x: Math.random() * 400 + 100,
          y: Math.random() * 300 + 100,
        },
      });
      setCreating(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to deploy");
      setCreating(null);
    }
  }, []);

  /** Pre-deploy check: validate secrets before deploying */
  const handleDeploy = async (template: Template) => {
    setCreating(template.id);
    setError(null);

    // Prefer the runtime the Go /templates endpoint returned verbatim —
    // resolveRuntime() is a legacy id→runtime fallback for installs whose
    // template summary predates the `runtime` field.
    const runtime = template.runtime ?? resolveRuntime(template.id);
    const preflight = await checkDeploySecrets({
      runtime,
      models: template.models,
      required_env: template.required_env,
    });

    if (!preflight.ok) {
      // Missing keys — show the modal instead of deploying
      setMissingKeysInfo({ template, preflight });
      setCreating(null);
      return;
    }

    // All keys present — deploy directly
    await executeDeploy(template);
  };

  return (
    <>
      {/* Toggle button */}
      <button
        onClick={() => setOpen(!open)}
        className={`fixed top-4 left-4 z-40 w-9 h-9 flex items-center justify-center rounded-lg transition-colors ${
          open
            ? "bg-blue-600 text-white"
            : "bg-zinc-900/90 border border-zinc-700/50 text-zinc-400 hover:text-zinc-200 hover:border-zinc-600"
        }`}
        title="Template Palette"
        aria-label={open ? "Close template palette" : "Open template palette"}
      >
        <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
          <rect x="1" y="1" width="6" height="6" rx="1" stroke="currentColor" strokeWidth="1.5" />
          <rect x="9" y="1" width="6" height="6" rx="1" stroke="currentColor" strokeWidth="1.5" />
          <rect x="1" y="9" width="6" height="6" rx="1" stroke="currentColor" strokeWidth="1.5" />
          <rect x="9" y="9" width="6" height="6" rx="1" stroke="currentColor" strokeWidth="1.5" />
        </svg>
      </button>

      {/* Missing Keys Modal */}
      <MissingKeysModal
        open={!!missingKeysInfo}
        missingKeys={missingKeysInfo?.preflight.missingKeys ?? []}
        providers={missingKeysInfo?.preflight.providers ?? []}
        runtime={missingKeysInfo?.preflight.runtime ?? ""}
        onKeysAdded={() => {
          if (missingKeysInfo) {
            const template = missingKeysInfo.template;
            setMissingKeysInfo(null);
            executeDeploy(template);
          }
        }}
        onCancel={() => setMissingKeysInfo(null)}
      />

      {/* Sidebar */}
      {open && (
        <div className="fixed top-0 left-0 h-full w-[280px] bg-zinc-900/95 backdrop-blur-md border-r border-zinc-800/60 z-30 flex flex-col shadow-2xl shadow-black/40">
          <div className="px-4 pt-14 pb-3 border-b border-zinc-800/60">
            <h2 className="text-sm font-semibold text-zinc-100">Templates</h2>
            <p className="text-[10px] text-zinc-500 mt-0.5">Click to deploy a workspace</p>
          </div>

          <div className="flex-1 overflow-y-auto p-3 space-y-2">
            {/* Org templates live INSIDE the scroll container so an
             *  expanded list (15+ entries) is reachable instead of
             *  overflowing the fixed footer below. */}
            <OrgTemplatesSection />

            {loading && (
              <div role="status" aria-live="polite" className="flex items-center justify-center gap-2 text-xs text-zinc-500 text-center py-8">
                <Spinner />
                Loading…
              </div>
            )}

            {!loading && templates.length === 0 && (
              <div role="status" aria-live="polite" className="text-xs text-zinc-500 text-center py-8">
                No templates found in<br />workspace-configs-templates/
              </div>
            )}

            {error && (
              <div className="px-3 py-1.5 bg-red-950/40 border border-red-800/50 rounded-lg text-xs text-red-400">
                {error}
              </div>
            )}

            {templates.map((t) => {
              const tierCfg = TIER_LABELS[t.tier] || TIER_LABELS[1];
              const isDeploying = creating === t.id;

              return (
                <button
                  key={t.id}
                  onClick={() => handleDeploy(t)}
                  disabled={isDeploying}
                  className="w-full text-left bg-zinc-800/40 hover:bg-zinc-800/70 border border-zinc-700/40 hover:border-zinc-600/50 rounded-xl p-3 transition-all disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:bg-zinc-800/40 disabled:hover:border-zinc-700/40 group focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/70"
                >
                  <div className="flex items-center justify-between mb-1">
                    <span className="text-[12px] font-semibold text-zinc-200 group-hover:text-zinc-100 truncate">
                      {t.name}
                    </span>
                    <span className={`text-[9px] font-mono px-1.5 py-0.5 rounded-md shrink-0 ${tierCfg.color}`}>
                      {tierCfg.label}
                    </span>
                  </div>

                  {t.description && (
                    <p className="text-[10px] text-zinc-500 mb-2 line-clamp-2 leading-relaxed">
                      {t.description}
                    </p>
                  )}

                  {t.skills?.length > 0 && (
                    <div className="flex flex-wrap gap-1">
                      {t.skills.slice(0, 3).map((s) => (
                        <span key={s} className="text-[8px] text-zinc-400 bg-zinc-700/40 px-1.5 py-0.5 rounded">
                          {s}
                        </span>
                      ))}
                      {t.skills.length > 3 && (
                        <span className="text-[8px] text-zinc-500">+{t.skills.length - 3}</span>
                      )}
                    </div>
                  )}

                  {isDeploying && (
                    <div className="text-[10px] text-sky-400 mt-1.5 motion-safe:animate-pulse">Deploying...</div>
                  )}
                </button>
              );
            })}
          </div>

          <div className="px-4 py-3 border-t border-zinc-800/60 space-y-3">
            <ImportAgentButton onImported={loadTemplates} />
            <button
              onClick={loadTemplates}
              className="text-[10px] text-zinc-500 hover:text-zinc-300 transition-colors block"
            >
              Refresh templates
            </button>
          </div>
        </div>
      )}
    </>
  );
}
