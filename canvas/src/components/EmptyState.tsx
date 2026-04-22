"use client";

import { useState, useEffect } from "react";
import { api } from "@/lib/api";
import { useCanvasStore } from "@/store/canvas";
import { OrgTemplatesSection } from "./TemplatePalette";
import { Spinner } from "./Spinner";
import { TIER_CONFIG } from "@/lib/design-tokens";

interface Template {
  id: string;
  name: string;
  description: string;
  tier: number;
  model: string;
  skills: string[];
  skill_count: number;
}

export function EmptyState() {
  const [templates, setTemplates] = useState<Template[]>([]);
  const [loading, setLoading] = useState(true);
  const [deploying, setDeploying] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api
      .get<Template[]>("/templates")
      .then((t) => setTemplates(t))
      .catch(() => setTemplates([]))
      .finally(() => setLoading(false));
  }, []);

  const deploy = async (template: Template) => {
    setDeploying(template.id);
    setError(null);
    try {
      const ws = await api.post<{ id: string }>("/workspaces", {
        name: template.name,
        template: template.id,
        tier: template.tier,
        canvas: { x: 200, y: 150 },
      });
      // Auto-select the new workspace and open chat
      setTimeout(() => {
        useCanvasStore.getState().selectNode(ws.id);
        useCanvasStore.getState().setPanelTab("chat");
      }, 500);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Deploy failed");
    } finally {
      setDeploying(null);
    }
  };

  const createBlank = async () => {
    setDeploying("blank");
    setError(null);
    try {
      const ws = await api.post<{ id: string }>("/workspaces", {
        name: "My First Agent",
        tier: 2,
        canvas: { x: 200, y: 150 },
      });
      setTimeout(() => {
        useCanvasStore.getState().selectNode(ws.id);
        useCanvasStore.getState().setPanelTab("chat");
      }, 500);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Create failed");
    } finally {
      setDeploying(null);
    }
  };

  return (
    <div className="absolute inset-0 flex items-start justify-center pointer-events-none z-[1] overflow-y-auto py-8">
      {/* Radial gradient glow behind the card */}
      <div className="fixed inset-0 pointer-events-none" aria-hidden="true">
        <div className="absolute top-1/4 left-1/2 -translate-x-1/2 w-[600px] h-[400px] bg-gradient-radial from-molecule-accent-mint/[0.04] via-molecule-accent-cyan/[0.02] to-transparent rounded-full blur-3xl" />
      </div>
      <div className="relative max-w-2xl w-full rounded-3xl border border-white/[0.06] bg-molecule-bg-900/80 backdrop-blur-2xl px-4 sm:px-8 py-6 sm:py-10 text-center shadow-premium-lg pointer-events-auto mx-4">
        <div className="absolute inset-x-8 top-0 h-px bg-gradient-to-r from-transparent via-molecule-accent-mint/40 to-transparent" />

        {/* Logo */}
        <div className="w-20 h-20 mx-auto mb-5 rounded-2xl bg-gradient-to-br from-molecule-accent-mint/15 via-molecule-accent-cyan/10 to-transparent border border-molecule-accent-mint/15 flex items-center justify-center shadow-glow-mint">
          <svg width="36" height="36" viewBox="0 0 28 28" fill="none">
            <rect x="3" y="3" width="10" height="10" rx="2" stroke="#39e58c" strokeWidth="1.5" opacity="0.7" />
            <rect x="15" y="3" width="10" height="10" rx="2" stroke="#22d1ee" strokeWidth="1.5" opacity="0.7" />
            <rect x="9" y="15" width="10" height="10" rx="2" stroke="#39e58c" strokeWidth="1.5" opacity="0.7" />
            <path d="M8 13v2M20 13v4M14 13v2" stroke="#22d1ee" strokeWidth="1.5" strokeLinecap="round" opacity="0.5" />
          </svg>
        </div>

        <p className="text-[11px] font-semibold uppercase tracking-[0.28em] text-gradient-mint-cyan mb-3">
          Welcome to Molecule AI
        </p>
        <h2 className="text-2xl font-semibold text-slate-100 mb-1.5">
          Deploy your first agent
        </h2>
        <p className="text-sm text-slate-400 mb-8 leading-relaxed">
          Pick a template to get started instantly, or create a blank workspace.
        </p>

        {/* Template grid */}
        {loading ? (
          <div className="flex items-center justify-center gap-2 text-xs text-zinc-400 py-4">
            <Spinner />
            Loading templates...
          </div>
        ) : templates.length > 0 ? (
          <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-3 mb-5 text-left">
            {templates.map((t) => {
              const tierColor = TIER_CONFIG[t.tier]?.border || TIER_CONFIG[1].border;
              const accentBorder = t.tier >= 3 ? "border-l-molecule-accent-mint" : t.tier === 2 ? "border-l-molecule-accent-cyan" : "border-l-slate-600";
              return (
                <button
                  key={t.id}
                  onClick={() => deploy(t)}
                  disabled={!!deploying}
                  className={`group rounded-xl border border-white/[0.06] border-l-2 ${accentBorder} bg-molecule-surface-900/50 px-4 py-3.5 hover:border-molecule-accent-mint/30 hover:bg-molecule-surface-800/60 hover:-translate-y-0.5 transition-all disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:border-white/[0.06] disabled:hover:bg-molecule-surface-900/50 disabled:hover:translate-y-0 text-left focus:outline-none focus-visible:ring-2 focus-visible:ring-molecule-accent-mint/60 shadow-sm hover:shadow-premium`}
                >
                  <div className="flex items-center gap-2 mb-1.5">
                    <span className="text-sm font-medium text-slate-200 group-hover:text-slate-50 truncate">
                      {deploying === t.id ? "Deploying..." : t.name}
                    </span>
                    <span className={`text-[9px] font-mono font-semibold px-1.5 py-0.5 rounded-md border ${tierColor}`}>
                      T{t.tier}
                    </span>
                  </div>
                  <p className="text-[12px] text-slate-400 line-clamp-2 leading-relaxed">
                    {t.description || "No description"}
                  </p>
                  {t.skill_count > 0 && (
                    <p className="text-[10px] text-slate-500 mt-2">
                      {t.skill_count} skill{t.skill_count !== 1 ? "s" : ""}
                      {t.model ? ` · ${t.model}` : ""}
                    </p>
                  )}
                </button>
              );
            })}
          </div>
        ) : null}

        {/* Create blank */}
        <button
          onClick={createBlank}
          disabled={!!deploying}
          className="w-full rounded-xl border border-dashed border-white/[0.08] bg-molecule-surface-900/30 px-4 py-3.5 text-sm text-slate-400 hover:text-slate-200 hover:border-molecule-accent-mint/30 hover:bg-molecule-surface-800/40 transition-all disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:text-slate-400 disabled:hover:border-white/[0.08] focus:outline-none focus-visible:ring-2 focus-visible:ring-molecule-accent-mint/60"
        >
          {deploying === "blank" ? "Creating..." : "+ Create blank workspace"}
        </button>

        {/* Org templates — instantiate a whole team in one click */}
        <div className="mt-5 pt-5 border-t border-white/[0.06] text-left">
          <OrgTemplatesSection />
        </div>

        {error && (
          <div role="alert" className="mt-3 px-3 py-2 bg-red-950/40 border border-red-800/50 rounded-lg text-xs text-red-400">
            {error}
          </div>
        )}

        {/* Tips */}
        <div className="mt-6 pt-5 border-t border-white/[0.06]">
          <div className="flex flex-wrap items-center justify-center gap-x-6 gap-y-2 text-[11px] text-slate-500">
            <span>Drag to nest workspaces into teams</span>
            <span className="text-slate-700 hidden sm:inline">|</span>
            <span>Right-click for actions</span>
            <span className="text-slate-700 hidden sm:inline">|</span>
            <span>Press <kbd className="px-1.5 py-0.5 bg-molecule-surface-800 rounded text-slate-400 font-mono border border-white/[0.06]">&#8984;K</kbd> to search</span>
          </div>
        </div>
      </div>
    </div>
  );
}
