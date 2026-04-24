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
      <div className="relative max-w-2xl w-full rounded-3xl border border-zinc-800/70 bg-zinc-950/80 backdrop-blur-xl px-8 py-8 text-center shadow-2xl shadow-black/40 pointer-events-auto mx-4">
        <div className="absolute inset-x-8 top-0 h-px bg-gradient-to-r from-transparent via-blue-500/50 to-transparent" />

        {/* Logo */}
        <div className="w-16 h-16 mx-auto mb-4 rounded-2xl bg-gradient-to-br from-sky-500/20 via-blue-500/20 to-violet-500/20 border border-blue-500/20 flex items-center justify-center">
          <svg width="28" height="28" viewBox="0 0 28 28" fill="none">
            <rect x="3" y="3" width="10" height="10" rx="2" stroke="#60a5fa" strokeWidth="1.5" opacity="0.65" />
            <rect x="15" y="3" width="10" height="10" rx="2" stroke="#60a5fa" strokeWidth="1.5" opacity="0.65" />
            <rect x="9" y="15" width="10" height="10" rx="2" stroke="#60a5fa" strokeWidth="1.5" opacity="0.65" />
            <path d="M8 13v2M20 13v4M14 13v2" stroke="#60a5fa" strokeWidth="1.5" strokeLinecap="round" opacity="0.45" />
          </svg>
        </div>

        <p className="text-[10px] font-semibold uppercase tracking-[0.28em] text-sky-400/80 mb-2">
          Welcome to Molecule AI
        </p>
        <h2 className="text-xl font-semibold text-zinc-100 mb-1">
          Deploy your first agent
        </h2>
        <p className="text-sm text-zinc-400 mb-6 leading-relaxed">
          Pick a template to get started instantly, or create a blank workspace.
        </p>

        {/* Template grid */}
        {loading ? (
          <div className="flex items-center justify-center gap-2 text-xs text-zinc-400 py-4">
            <Spinner />
            Loading templates...
          </div>
        ) : templates.length > 0 ? (
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-2.5 mb-4 text-left">
            {templates.map((t) => {
              const tierColor = TIER_CONFIG[t.tier]?.border || TIER_CONFIG[1].border;
              return (
                <button
                  type="button"
                  key={t.id}
                  onClick={() => deploy(t)}
                  disabled={!!deploying}
                  className="group rounded-xl border border-zinc-800/60 bg-zinc-900/50 px-3.5 py-3 hover:border-blue-500/40 hover:bg-zinc-900/80 transition-all disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:border-zinc-800/60 disabled:hover:bg-zinc-900/50 text-left focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/70"
                >
                  <div className="flex items-center gap-2 mb-1">
                    <span className="text-sm font-medium text-zinc-200 group-hover:text-zinc-100 truncate">
                      {deploying === t.id ? "Deploying..." : t.name}
                    </span>
                    <span className={`text-[8px] font-mono font-semibold px-1.5 py-0.5 rounded-md border ${tierColor}`}>
                      T{t.tier}
                    </span>
                  </div>
                  <p className="text-[11px] text-zinc-500 line-clamp-2 leading-relaxed">
                    {t.description || "No description"}
                  </p>
                  {t.skill_count > 0 && (
                    <p className="text-[9px] text-zinc-500 mt-1.5">
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
          type="button"
          onClick={createBlank}
          disabled={!!deploying}
          className="w-full rounded-xl border border-dashed border-zinc-700/60 bg-zinc-900/30 px-4 py-3 text-sm text-zinc-400 hover:text-zinc-200 hover:border-zinc-600 hover:bg-zinc-900/50 transition-all disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:text-zinc-400 disabled:hover:border-zinc-700/60 focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/70"
        >
          {deploying === "blank" ? "Creating..." : "+ Create blank workspace"}
        </button>

        {/* Org templates — instantiate a whole team in one click */}
        <div className="mt-4 pt-4 border-t border-zinc-800/50 text-left">
          <OrgTemplatesSection />
        </div>

        {error && (
          <div role="alert" className="mt-3 px-3 py-2 bg-red-950/40 border border-red-800/50 rounded-lg text-xs text-red-400">
            {error}
          </div>
        )}

        {/* Tips */}
        <div className="mt-5 pt-4 border-t border-zinc-800/50">
          <div className="flex items-center justify-center gap-6 text-[10px] text-zinc-400">
            <span>Drag to nest workspaces into teams</span>
            <span className="text-zinc-700">|</span>
            <span>Right-click for actions</span>
            <span className="text-zinc-700">|</span>
            <span>Press <kbd className="px-1 py-0.5 bg-zinc-800 rounded text-zinc-500 font-mono">&#8984;K</kbd> to search</span>
          </div>
        </div>
      </div>
    </div>
  );
}
