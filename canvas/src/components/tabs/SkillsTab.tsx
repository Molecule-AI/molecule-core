"use client";

import { useMemo, useState, useEffect, useCallback, useRef } from "react";
import { api } from "@/lib/api";
import { useCanvasStore, summarizeWorkspaceCapabilities, type WorkspaceNodeData } from "@/store/canvas";
import { showToast } from "../Toaster";

interface Props {
  data: WorkspaceNodeData;
}

interface SkillEntry {
  id: string;
  name: string;
  description: string;
  tags: string[];
  examples: string[];
}

interface PluginInfo {
  name: string;
  version: string;
  description: string;
  author: string;
  tags: string[];
  skills: string[];
  // Declared supported runtimes (e.g. ["claude_code", "deepagents"]).
  // Empty / absent = "unspecified, try it".
  runtimes?: string[];
  // Only present on /workspaces/:id/plugins responses — true if the
  // plugin declared support for the workspace's current runtime (or
  // declared no runtimes at all). Lets us grey out inert installs.
  supported_on_runtime?: boolean;
}

interface SourceSchemesResponse {
  schemes: string[];
}

// Delay before reloading installed plugins after install/uninstall (workspace restarts)
const PLUGIN_RELOAD_DELAY_MS = 15_000;

export function SkillsTab({ data }: Props) {
  const capability = summarizeWorkspaceCapabilities(data);
  const skills = useMemo(() => extractSkills(data.agentCard), [data.agentCard]);
  const setPanelTab = useCanvasStore((s) => s.setPanelTab);
  const promotionTask = data.currentTask.startsWith("Skill promotion:");

  const [registry, setRegistry] = useState<PluginInfo[]>([]);
  const [installed, setInstalled] = useState<PluginInfo[]>([]);
  const [sourceSchemes, setSourceSchemes] = useState<string[]>([]);
  const [installing, setInstalling] = useState<string | null>(null);
  const [uninstalling, setUninstalling] = useState<string | null>(null);
  const [showRegistry, setShowRegistry] = useState(false);
  const [customSource, setCustomSource] = useState("");
  const mountedRef = useRef(true);
  const reloadTimerRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  useEffect(() => {
    return () => {
      mountedRef.current = false;
      clearTimeout(reloadTimerRef.current);
    };
  }, []);

  const workspaceId = data.id;

  const loadInstalled = useCallback(async () => {
    try {
      const result = await api.get<PluginInfo[]>(`/workspaces/${workspaceId}/plugins`);
      if (mountedRef.current) setInstalled(result);
    } catch { /* ignore */ }
  }, [workspaceId]);

  const loadRegistry = useCallback(async () => {
    try {
      const result = await api.get<PluginInfo[]>("/plugins");
      if (mountedRef.current) setRegistry(result);
    } catch { /* ignore */ }
  }, []);

  const loadSourceSchemes = useCallback(async () => {
    try {
      const result = await api.get<SourceSchemesResponse>("/plugins/sources");
      if (mountedRef.current) setSourceSchemes(result.schemes ?? []);
    } catch { /* ignore — falls back to "local only" UX */ }
  }, []);

  useEffect(() => {
    loadInstalled();
    loadRegistry();
    loadSourceSchemes();
  }, [loadInstalled, loadRegistry, loadSourceSchemes]);

  const installedNames = useMemo(() => new Set(installed.map((p) => p.name)), [installed]);

  // Install always goes through the source-based API. For registry
  // plugins we build the local:// source on the fly; custom sources
  // (github://, clawhub://, …) are typed into the input below.
  const installFromSource = async (source: string, labelOverride?: string) => {
    const label = labelOverride ?? source;
    setInstalling(label);
    try {
      await api.post(`/workspaces/${workspaceId}/plugins`, { source });
      showToast(`Installed ${label} — restarting workspace`, "success");
      reloadTimerRef.current = setTimeout(() => loadInstalled(), PLUGIN_RELOAD_DELAY_MS);
    } catch (e) {
      showToast(e instanceof Error ? e.message : "Install failed", "error");
    } finally {
      setInstalling(null);
    }
  };

  const handleInstall = (pluginName: string) => installFromSource(`local://${pluginName}`, pluginName);

  const handleInstallCustom = async () => {
    const source = customSource.trim();
    if (!source) return;
    await installFromSource(source);
    setCustomSource("");
  };

  const handleUninstall = async (pluginName: string) => {
    setUninstalling(pluginName);
    try {
      await api.del(`/workspaces/${data.id}/plugins/${pluginName}`);
      showToast(`Removed ${pluginName} — restarting workspace`, "success");
      setInstalled((prev) => prev.filter((p) => p.name !== pluginName));
      reloadTimerRef.current = setTimeout(() => loadInstalled(), PLUGIN_RELOAD_DELAY_MS);
    } catch (e) {
      showToast(e instanceof Error ? e.message : "Uninstall failed", "error");
    } finally {
      setUninstalling(null);
    }
  };

  return (
    <div className="p-4 space-y-4">
      {/* Plugins section */}
      <div className="rounded-xl border border-zinc-800 bg-zinc-900/70 p-3">
        <div className="flex items-center justify-between gap-3">
          <div>
            <div className="text-[10px] uppercase tracking-[0.22em] text-zinc-500">Plugins</div>
            <h3 className="mt-1 text-sm font-semibold text-zinc-100">
              {installed.length} installed
            </h3>
          </div>
          <button
            onClick={() => setShowRegistry(!showRegistry)}
            className="rounded-full border border-violet-700/50 bg-violet-950/30 px-3 py-1 text-[10px] text-violet-200 hover:bg-violet-900/40 transition-colors"
          >
            {showRegistry ? "Hide Registry" : "+ Install Plugin"}
          </button>
        </div>

        {/* Installed plugins */}
        {installed.length > 0 && (
          <div className="mt-3 space-y-1.5">
            {installed.map((p) => {
              // Plugin was installed but does NOT declare support for
              // the workspace's current runtime — grey it out so users
              // see it's inert. Happens after a runtime change or when
              // someone installs a runtime-specific plugin on a wrong
              // workspace.
              const inert = p.supported_on_runtime === false;
              return (
                <div
                  key={p.name}
                  className={`flex items-center justify-between gap-2 rounded-lg border px-3 py-2 ${
                    inert
                      ? "border-amber-800/40 bg-amber-950/10 opacity-70"
                      : "border-zinc-800/60 bg-zinc-950/40"
                  }`}
                >
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="text-[11px] font-medium text-zinc-200">{p.name}</span>
                      {p.version && <span className="text-[9px] text-zinc-600">v{p.version}</span>}
                      {inert && (
                        <span className="rounded-full border border-amber-700/50 bg-amber-950/30 px-1.5 py-0.5 text-[8px] text-amber-300">
                          inert on this runtime
                        </span>
                      )}
                    </div>
                    {p.description && <div className="text-[10px] text-zinc-500 truncate">{p.description}</div>}
                    {p.skills && p.skills.length > 0 && (
                      <div className="mt-1 flex flex-wrap gap-1">
                        {p.skills.slice(0, 4).map((s) => (
                          <span key={s} className="rounded-full bg-zinc-800/60 px-1.5 py-0.5 text-[8px] text-zinc-400">{s}</span>
                        ))}
                        {p.skills.length > 4 && (
                          <span className="text-[8px] text-zinc-600">+{p.skills.length - 4}</span>
                        )}
                      </div>
                    )}
                  </div>
                  <button
                    onClick={() => handleUninstall(p.name)}
                    disabled={uninstalling === p.name}
                    className="shrink-0 rounded-full border border-red-800/40 bg-red-950/20 px-2 py-0.5 text-[9px] text-red-400 hover:bg-red-900/30 disabled:opacity-30"
                  >
                    {uninstalling === p.name ? "..." : "Remove"}
                  </button>
                </div>
              );
            })}
          </div>
        )}

        {/* Plugin registry (expandable) */}
        {showRegistry && (
          <div className="mt-3 border-t border-zinc-800/40 pt-3">
            {/* Install from any source (github://, clawhub://, …) */}
            <div className="mb-3 rounded-lg border border-zinc-800/60 bg-zinc-950/40 p-2.5">
              <div className="flex items-center justify-between gap-2 mb-1.5">
                <div className="text-[9px] uppercase tracking-[0.2em] text-zinc-600">
                  Install from source
                </div>
                {sourceSchemes.length > 0 && (
                  <div className="flex flex-wrap gap-1">
                    {sourceSchemes.map((s) => (
                      <span
                        key={s}
                        className="rounded-full border border-zinc-700/50 bg-zinc-900/50 px-1.5 py-0.5 text-[8px] text-zinc-500"
                      >
                        {s}://
                      </span>
                    ))}
                  </div>
                )}
              </div>
              <div className="flex items-center gap-1.5">
                <input
                  type="text"
                  value={customSource}
                  onChange={(e) => setCustomSource(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" && !installing) handleInstallCustom();
                  }}
                  placeholder="e.g. github://owner/repo#v1.0"
                  spellCheck={false}
                  className="flex-1 rounded border border-zinc-700 bg-zinc-950 px-2 py-1 text-[10px] text-zinc-200 placeholder:text-zinc-600 focus:border-violet-600 focus:outline-none"
                />
                <button
                  onClick={handleInstallCustom}
                  disabled={!customSource.trim() || installing !== null}
                  className="shrink-0 rounded-full border border-violet-700/50 bg-violet-950/30 px-2.5 py-1 text-[9px] text-violet-300 hover:bg-violet-900/40 disabled:opacity-30"
                >
                  {installing === customSource.trim() ? "Installing..." : "Install"}
                </button>
              </div>
              <div className="mt-1 text-[9px] text-zinc-600">
                Local registry plugins below; paste any scheme URL above for GitHub or other sources.
              </div>
            </div>
            <div className="text-[9px] uppercase tracking-[0.2em] text-zinc-600 mb-2">Available plugins</div>
            {registry.length === 0 ? (
              <div className="text-[10px] text-zinc-600">No plugins in registry</div>
            ) : (
              <div className="space-y-1.5">
                {registry.map((p) => {
                  const isInstalled = installedNames.has(p.name);
                  return (
                    <div key={p.name} className="flex items-center justify-between gap-2 rounded-lg border border-zinc-800/40 bg-zinc-950/30 px-3 py-2">
                      <div className="min-w-0">
                        <div className="flex items-center gap-2">
                          <span className="text-[11px] text-zinc-300">{p.name}</span>
                          {p.version && <span className="text-[9px] text-zinc-600">v{p.version}</span>}
                        </div>
                        {p.description && <div className="text-[10px] text-zinc-500 truncate">{p.description}</div>}
                        {p.tags && p.tags.length > 0 && (
                          <div className="mt-1 flex flex-wrap gap-1">
                            {p.tags.map((t) => (
                              <span key={t} className="rounded-full border border-zinc-700/40 px-1.5 py-0.5 text-[8px] text-zinc-500">{t}</span>
                            ))}
                          </div>
                        )}
                        {p.runtimes && p.runtimes.length > 0 && (
                          <div className="mt-1 flex flex-wrap gap-1">
                            {p.runtimes.map((r) => (
                              <span key={r} className="rounded-full border border-blue-800/40 bg-blue-950/20 px-1.5 py-0.5 text-[8px] text-blue-300">{r}</span>
                            ))}
                          </div>
                        )}
                      </div>
                      {isInstalled ? (
                        <span className="shrink-0 text-[9px] text-emerald-500">Installed</span>
                      ) : (
                        <button
                          onClick={() => handleInstall(p.name)}
                          disabled={installing === p.name}
                          className="shrink-0 rounded-full border border-violet-700/50 bg-violet-950/30 px-2.5 py-0.5 text-[9px] text-violet-300 hover:bg-violet-900/40 disabled:opacity-30"
                        >
                          {installing === p.name ? "Installing..." : "Install"}
                        </button>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        )}
      </div>

      {/* Skills section */}
      <div className="rounded-xl border border-zinc-800 bg-zinc-900/70 p-3">
        <div className="flex items-center justify-between gap-3">
          <div>
            <div className="text-[10px] uppercase tracking-[0.22em] text-zinc-500">Workspace skills</div>
            <h3 className="mt-1 text-sm font-semibold text-zinc-100">Installed skills</h3>
          </div>
          <div className="flex flex-wrap gap-2">
            <MetaPill label="Count" value={String(capability.skillCount)} />
            <MetaPill label="Runtime" value={capability.runtime || "unknown"} />
          </div>
        </div>
        <p className="mt-2 text-[11px] leading-5 text-zinc-500">
          Live skill directory from the Agent Card — updates when the workspace hot-reloads skills.
        </p>
        <div className="mt-3 flex flex-wrap gap-2">
          <button
            onClick={() => setPanelTab("config")}
            className="rounded-full border border-zinc-700 bg-zinc-950 px-3 py-1 text-[10px] text-zinc-300 hover:bg-zinc-900"
          >
            Open Config
          </button>
          <button
            onClick={() => setPanelTab("files")}
            className="rounded-full border border-zinc-700 bg-zinc-950 px-3 py-1 text-[10px] text-zinc-300 hover:bg-zinc-900"
          >
            Open Files
          </button>
        </div>
      </div>

      {promotionTask && (
        <div className="rounded-xl border border-violet-800/30 bg-violet-950/20 p-3 text-[11px] text-violet-200/90">
          A skill promotion is currently in flight. The workspace is compressing a repeatable workflow into
          a new skill package.
        </div>
      )}

      {skills.length === 0 ? (
        <div className="rounded-xl border border-dashed border-zinc-800 bg-zinc-900/40 p-6 text-center">
          <div className="text-sm text-zinc-100">No skills loaded</div>
          <p className="mt-2 text-[11px] leading-5 text-zinc-500">
            Add skills from the Config tab, install a plugin above, or let the runtime hot-load them.
          </p>
        </div>
      ) : (
        <div className="space-y-2">
          {skills.map((skill) => (
            <div key={skill.id} className="rounded-xl border border-zinc-800 bg-zinc-900/60 p-3">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <div className="text-xs font-semibold text-zinc-100">{skill.name}</div>
                  <div className="mt-0.5 text-[10px] font-mono text-zinc-500">{skill.id}</div>
                </div>
                {skill.tags.length > 0 && (
                  <div className="flex flex-wrap justify-end gap-1.5">
                    {skill.tags.slice(0, 4).map((tag) => (
                      <span
                        key={tag}
                        className="rounded-full border border-zinc-700 bg-zinc-900 px-2 py-0.5 text-[9px] text-zinc-400"
                      >
                        {tag}
                      </span>
                    ))}
                  </div>
                )}
              </div>

              {skill.description && (
                <p className="mt-2 text-[11px] leading-5 text-zinc-400">{skill.description}</p>
              )}

              {skill.examples.length > 0 && (
                <div className="mt-2">
                  <div className="text-[9px] uppercase tracking-[0.2em] text-zinc-600">Examples</div>
                  <div className="mt-1 space-y-1">
                    {skill.examples.slice(0, 2).map((example, index) => (
                      <div
                        key={`${skill.id}-${index}`}
                        className="rounded-md border border-zinc-800 bg-zinc-950/60 px-2 py-1 text-[10px] text-zinc-300"
                      >
                        {example}
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function extractSkills(agentCard: Record<string, unknown> | null): SkillEntry[] {
  if (!agentCard) return [];
  const rawSkills = agentCard.skills;
  if (!Array.isArray(rawSkills)) return [];

  return rawSkills
    .map((skill: Record<string, unknown>) => ({
      id: String(skill.id || skill.name || ""),
      name: String(skill.name || skill.id || "Unnamed skill"),
      description: String(skill.description || ""),
      tags: Array.isArray(skill.tags) ? skill.tags.map((tag) => String(tag)) : [],
      examples: Array.isArray(skill.examples) ? skill.examples.map((example) => String(example)) : [],
    }))
    .filter((skill) => skill.id.length > 0);
}

function MetaPill({ label, value }: { label: string; value: string }) {
  return (
    <span className="inline-flex items-center gap-1 rounded-full border border-zinc-700/60 bg-zinc-950/60 px-2 py-1 text-[9px] text-zinc-300">
      <span className="uppercase tracking-[0.18em] text-[8px] text-zinc-500">{label}</span>
      <span className="font-medium">{value}</span>
    </span>
  );
}
