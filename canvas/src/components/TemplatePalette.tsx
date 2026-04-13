"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import { api } from "@/lib/api";
import { checkDeploySecrets, type PreflightResult } from "@/lib/deploy-preflight";
import { MissingKeysModal } from "./MissingKeysModal";

interface Template {
  id: string;
  name: string;
  description: string;
  tier: number;
  model: string;
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

/** Import an org template by directory name. Throws on platform error so the
 * caller can surface the message in its error state. */
export async function importOrgTemplate(dir: string): Promise<void> {
  await api.post("/org/import", { dir });
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
    } catch (e) {
      setError(e instanceof Error ? e.message : "Import failed");
    } finally {
      setImporting(null);
    }
  };

  return (
    <div className="space-y-2" data-testid="org-templates-section">
      <div className="flex items-center justify-between">
        <h3 className="text-[10px] uppercase tracking-wide text-zinc-500 font-semibold">
          Org Templates
        </h3>
        <button
          onClick={loadOrgs}
          className="text-[9px] text-zinc-500 hover:text-zinc-300"
        >
          ↻
        </button>
      </div>

      {loading && <div className="text-[10px] text-zinc-500">Loading…</div>}

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
            className="bg-zinc-800/30 border border-zinc-700/40 rounded-lg p-2.5"
          >
            <div className="flex items-center justify-between mb-0.5">
              <span className="text-[11px] font-semibold text-zinc-200 truncate">
                {o.name || o.dir}
              </span>
              <span className="text-[9px] font-mono text-violet-400 bg-violet-950/40 px-1.5 py-0.5 rounded shrink-0">
                {o.workspaces}w
              </span>
            </div>
            {o.description && (
              <p className="text-[10px] text-zinc-500 mb-2 line-clamp-2 leading-relaxed">
                {o.description}
              </p>
            )}
            <button
              onClick={() => handleImport(o)}
              disabled={isImporting}
              className="w-full px-2 py-1 bg-violet-600/20 hover:bg-violet-600/30 border border-violet-500/30 rounded text-[10px] text-violet-300 font-medium transition-colors disabled:opacity-50"
            >
              {isImporting ? "Importing…" : "Import org"}
            </button>
          </div>
        );
      })}
    </div>
  );
}

const TIER_LABELS: Record<number, { label: string; color: string }> = {
  1: { label: "T1", color: "text-zinc-400 bg-zinc-800/60" },
  2: { label: "T2", color: "text-sky-400 bg-sky-950/40" },
  3: { label: "T3", color: "text-violet-400 bg-violet-950/40" },
  4: { label: "T4", color: "text-amber-400 bg-amber-950/40" },
};

function ImportAgentButton({ onImported }: { onImported: () => void }) {
  const [importing, setImporting] = useState(false);
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
        alert("No files found in the selected folder");
        return;
      }

      const name = agentName || "Imported Agent";
      await api.post("/templates/import", { name, files });
      onImported();
    } catch (e) {
      alert(e instanceof Error ? e.message : "Import failed");
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
    </div>
  );
}

export function TemplatePalette() {
  const [open, setOpen] = useState(false);
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

    const runtime = resolveRuntime(template.id);
    const preflight = await checkDeploySecrets(runtime);

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
            {loading && (
              <div className="text-xs text-zinc-500 text-center py-8">Loading...</div>
            )}

            {!loading && templates.length === 0 && (
              <div className="text-xs text-zinc-500 text-center py-8">
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
                  className="w-full text-left bg-zinc-800/40 hover:bg-zinc-800/70 border border-zinc-700/40 hover:border-zinc-600/50 rounded-xl p-3 transition-all disabled:opacity-50 group"
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
                    <div className="text-[10px] text-sky-400 mt-1.5 animate-pulse">Deploying...</div>
                  )}
                </button>
              );
            })}
          </div>

          <div className="px-4 py-3 border-t border-zinc-800/60 space-y-3">
            <OrgTemplatesSection />
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
