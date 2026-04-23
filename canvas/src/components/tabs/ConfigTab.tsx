"use client";

import { useState, useEffect, useCallback, useRef, useId } from "react";
import { api } from "@/lib/api";
import { useCanvasStore } from "@/store/canvas";
import { type ConfigData, DEFAULT_CONFIG, TextInput, NumberInput, Toggle, TagList, Section } from "./config/form-inputs";
import { parseYaml, toYaml } from "./config/yaml-utils";
import { SecretsSection } from "./config/secrets-section";

interface Props {
  workspaceId: string;
}

// --- Agent Card Section ---

function AgentCardSection({ workspaceId }: { workspaceId: string }) {
  const [card, setCard] = useState<Record<string, unknown> | null>(null);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  useEffect(() => {
    api.get<Record<string, unknown>>(`/workspaces/${workspaceId}`)
      .then((ws) => setCard((ws.agent_card as Record<string, unknown>) || null))
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [workspaceId]);

  const handleSave = async () => {
    setError(null);
    let parsed: unknown;
    try { parsed = JSON.parse(draft); } catch { setError("Invalid JSON"); return; }
    setSaving(true);
    try {
      await api.post("/registry/update-card", { workspace_id: workspaceId, agent_card: parsed });
      setCard(parsed as Record<string, unknown>);
      setSuccess(true);
      setEditing(false);
      setTimeout(() => setSuccess(false), 2000);
    } catch (e) { setError(e instanceof Error ? e.message : "Failed to update"); }
    finally { setSaving(false); }
  };

  return (
    <Section title="Agent Card" defaultOpen={false}>
      {loading ? (
        <div className="text-[10px] text-zinc-500">Loading...</div>
      ) : editing ? (
        <div className="space-y-2">
          <textarea
            value={draft} onChange={(e) => setDraft(e.target.value)}
            spellCheck={false} rows={12}
            className="w-full bg-zinc-800 border border-zinc-700 rounded p-2 text-[10px] font-mono text-zinc-200 focus:outline-none focus:border-blue-500 resize-none"
          />
          {error && <div className="px-2 py-1 bg-red-900/30 border border-red-800 rounded text-[10px] text-red-400">{error}</div>}
          <div className="flex gap-2">
            <button onClick={handleSave} disabled={saving}
              className="px-2 py-1 bg-blue-600 hover:bg-blue-500 text-[10px] rounded text-white disabled:opacity-50">
              {saving ? "Saving..." : "Save"}
            </button>
            <button onClick={() => setEditing(false)}
              className="px-2 py-1 bg-zinc-700 hover:bg-zinc-600 text-[10px] rounded text-zinc-300">Cancel</button>
          </div>
        </div>
      ) : (
        <div>
          {card ? (
            <pre className="text-[9px] text-zinc-400 bg-zinc-800/50 rounded p-2 overflow-x-auto max-h-48 border border-zinc-700/50">
              {JSON.stringify(card, null, 2)}
            </pre>
          ) : (
            <div className="text-[10px] text-zinc-500">No agent card</div>
          )}
          {success && <div className="mt-2 px-2 py-1 bg-green-900/30 border border-green-800 rounded text-[10px] text-green-400">Updated</div>}
          <button onClick={() => { setDraft(JSON.stringify(card || {}, null, 2)); setEditing(true); setError(null); setSuccess(false); }}
            className="mt-2 text-[10px] text-blue-400 hover:text-blue-300">Edit Agent Card</button>
        </div>
      )}
    </Section>
  );
}

// --- Main ConfigTab ---

interface ModelSpec {
  id: string;
  name?: string;
  required_env?: string[];
}

function arraysEqual(a: readonly string[], b: readonly string[]): boolean {
  return a.length === b.length && a.every((v, i) => v === b[i]);
}

interface RuntimeOption {
  value: string;
  label: string;
  models: ModelSpec[];
}

// Fallback used when /templates can't be fetched (offline, older backend).
// Keep in sync with manifest.json workspace_templates as a defensive default.
// Model + env suggestions only flow when the backend is reachable.
const FALLBACK_RUNTIME_OPTIONS: RuntimeOption[] = [
  { value: "", label: "LangGraph (default)", models: [] },
  { value: "claude-code", label: "Claude Code", models: [] },
  { value: "crewai", label: "CrewAI", models: [] },
  { value: "autogen", label: "AutoGen", models: [] },
  { value: "deepagents", label: "DeepAgents", models: [] },
  { value: "openclaw", label: "OpenClaw", models: [] },
  { value: "hermes", label: "Hermes", models: [] },
  { value: "gemini-cli", label: "Gemini CLI", models: [] },
];

export function ConfigTab({ workspaceId }: Props) {
  const [config, setConfig] = useState<ConfigData>({ ...DEFAULT_CONFIG });
  const [originalYaml, setOriginalYaml] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);
  const [rawMode, setRawMode] = useState(false);
  const [rawDraft, setRawDraft] = useState("");
  const [runtimeOptions, setRuntimeOptions] = useState<RuntimeOption[]>(FALLBACK_RUNTIME_OPTIONS);
  const successTimerRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  useEffect(() => {
    return () => clearTimeout(successTimerRef.current);
  }, []);

  const loadConfig = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await api.get<{ content: string }>(`/workspaces/${workspaceId}/files/config.yaml`);
      const parsed = parseYaml(res.content);
      setOriginalYaml(res.content);
      setRawDraft(res.content);
      setConfig({ ...DEFAULT_CONFIG, ...parsed } as ConfigData);
    } catch {
      setError("No config.yaml found");
    } finally {
      setLoading(false);
    }
  }, [workspaceId]);

  useEffect(() => {
    loadConfig();
  }, [loadConfig]);

  useEffect(() => {
    let cancelled = false;
    api.get<Array<{ id: string; name?: string; runtime?: string; models?: ModelSpec[] }>>("/templates")
      .then((rows) => {
        if (cancelled || !Array.isArray(rows)) return;
        const byRuntime = new Map<string, RuntimeOption>();
        byRuntime.set("", { value: "", label: "LangGraph (default)", models: [] });
        for (const r of rows) {
          const v = (r.runtime || "").trim();
          if (!v || v === "langgraph") continue;
          // Last template wins if two templates share a runtime — rare, and the
          // one with the richer models list is probably newer.
          const existing = byRuntime.get(v);
          const models = Array.isArray(r.models) ? r.models : [];
          if (!existing || models.length > existing.models.length) {
            byRuntime.set(v, { value: v, label: r.name || v, models });
          }
        }
        if (byRuntime.size > 1) setRuntimeOptions(Array.from(byRuntime.values()));
      })
      .catch(() => { /* keep fallback */ });
    return () => { cancelled = true; };
  }, []);

  // Models + env hints for the currently-selected runtime.
  const selectedRuntime = runtimeOptions.find((o) => o.value === (config.runtime || "")) ?? null;
  const availableModels: ModelSpec[] = selectedRuntime?.models ?? [];
  const currentModelId = config.runtime_config?.model || config.model || "";
  const currentModelSpec = availableModels.find((m) => m.id === currentModelId) ?? null;

  const update = <K extends keyof ConfigData>(key: K, value: ConfigData[K]) => {
    setConfig((prev) => ({ ...prev, [key]: value }));
  };

  const updateNested = <K extends keyof ConfigData>(key: K, subKey: string, value: unknown) => {
    setConfig((prev) => ({
      ...prev,
      [key]: { ...(prev[key] as Record<string, unknown>), [subKey]: value },
    }));
  };

  const handleSave = async (restart: boolean) => {
    setSaving(true);
    setError(null);
    setSuccess(false);
    try {
      const content = rawMode ? rawDraft : toYaml(config);
      await api.put(`/workspaces/${workspaceId}/files/config.yaml`, { content });

      // If runtime changed, update it in the DB so restart uses the correct image
      const newRuntime = rawMode
        ? (parseYaml(rawDraft).runtime as string || "")
        : (config.runtime || "");
      const oldRuntime = (parseYaml(originalYaml).runtime as string || "");
      if (newRuntime && newRuntime !== oldRuntime) {
        await api.patch(`/workspaces/${workspaceId}`, { runtime: newRuntime });
      }

      setOriginalYaml(content);
      if (rawMode) {
        const parsed = parseYaml(content);
        setConfig({ ...DEFAULT_CONFIG, ...parsed } as ConfigData);
      } else {
        setRawDraft(content);
      }
      if (restart) {
        await useCanvasStore.getState().restartWorkspace(workspaceId);
      } else {
        useCanvasStore.getState().updateNodeData(workspaceId, { needsRestart: true });
      }
      setSuccess(true);
      clearTimeout(successTimerRef.current);
      successTimerRef.current = setTimeout(() => setSuccess(false), 2000);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to save");
    } finally {
      setSaving(false);
    }
  };

  // Stable IDs for bare label↔control pairs (WCAG 1.3.1)
  const descriptionId = useId();
  const tierId = useId();
  const runtimeId = useId();
  const effortId = useId();
  const taskBudgetId = useId();
  const sandboxBackendId = useId();

  const isDirty = rawMode ? rawDraft !== originalYaml : toYaml(config) !== originalYaml;

  if (loading) {
    return <div className="p-4 text-xs text-zinc-500">Loading config...</div>;
  }

  return (
    <div className="flex flex-col h-full">
      {/* Mode toggle */}
      <div className="flex items-center justify-between px-3 py-1.5 border-b border-zinc-800/40 bg-zinc-900/30">
        <span className="text-[10px] text-zinc-500">config.yaml</span>
        <label className="flex items-center gap-1.5 cursor-pointer">
          <span className="text-[9px] text-zinc-500">Raw YAML</span>
          <input
            type="checkbox"
            checked={rawMode}
            onChange={(e) => {
              if (e.target.checked) {
                setRawDraft(toYaml(config));
              } else {
                const parsed = parseYaml(rawDraft);
                setConfig({ ...DEFAULT_CONFIG, ...parsed } as ConfigData);
              }
              setRawMode(e.target.checked);
            }}
            className="accent-blue-500"
          />
        </label>
      </div>

      {rawMode ? (
        <div className="flex-1 p-3">
          <textarea
            value={rawDraft}
            onChange={(e) => setRawDraft(e.target.value)}
            spellCheck={false}
            className="w-full h-full min-h-[300px] bg-zinc-800 border border-zinc-700 rounded p-3 text-xs font-mono text-zinc-200 focus:outline-none focus:border-blue-500 resize-none"
          />
        </div>
      ) : (
        <div className="flex-1 overflow-y-auto p-3 space-y-2">
          <Section title="General">
            <TextInput label="Name" value={config.name} onChange={(v) => update("name", v)} />
            <div>
              <label htmlFor={descriptionId} className="text-[10px] text-zinc-500 block mb-1">Description</label>
              <textarea
                id={descriptionId}
                value={config.description}
                onChange={(e) => update("description", e.target.value)}
                rows={3}
                className="w-full bg-zinc-800 border border-zinc-700 rounded px-2 py-1 text-xs text-zinc-200 focus:outline-none focus:border-blue-500 resize-none"
              />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <TextInput label="Version" value={config.version} onChange={(v) => update("version", v)} mono />
              <div>
                <label htmlFor={tierId} className="text-[10px] text-zinc-500 block mb-1">Tier</label>
                <select
                  id={tierId}
                  value={config.tier}
                  onChange={(e) => update("tier", parseInt(e.target.value, 10))}
                  className="w-full bg-zinc-800 border border-zinc-700 rounded px-2 py-1 text-xs text-zinc-200 focus:outline-none focus:border-blue-500"
                >
                  <option value={1}>T1 — Sandboxed</option>
                  <option value={2}>T2 — Standard</option>
                  <option value={3}>T3 — Full Access</option>
                </select>
              </div>
            </div>
          </Section>

          <Section title="Runtime">
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label htmlFor={runtimeId} className="text-[10px] text-zinc-500 block mb-1">Runtime</label>
                <select
                  id={runtimeId}
                  value={config.runtime || ""}
                  onChange={(e) => update("runtime", e.target.value)}
                  className="w-full bg-zinc-800 border border-zinc-700 rounded px-2 py-1 text-xs text-zinc-200 focus:outline-none focus:border-blue-500"
                >
                  {runtimeOptions.map((opt) => (
                    <option key={opt.value} value={opt.value}>{opt.label}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="text-[10px] text-zinc-500 block mb-1">
                  Model
                  {availableModels.length > 0 && (
                    <span className="ml-1 text-zinc-600">({availableModels.length} suggested)</span>
                  )}
                </label>
                <input
                  type="text"
                  list={availableModels.length > 0 ? `${runtimeId}-models` : undefined}
                  value={currentModelId}
                  onChange={(e) => {
                    const v = e.target.value;
                    setConfig((prev) => {
                      // If the new value exactly matches a known modelSpec id,
                      // swap required_env to that spec's list — but only when
                      // the current required_env is empty or was itself
                      // template-driven (i.e. matches the previous modelSpec's
                      // required_env). User-typed envs always win.
                      const nextSpec = availableModels.find((m) => m.id === v) ?? null;
                      const prevModelId = prev.runtime_config?.model || prev.model || "";
                      const prevSpec = availableModels.find((m) => m.id === prevModelId) ?? null;
                      const prevRequired = prev.runtime_config?.required_env ?? [];
                      const wasTemplateDriven =
                        prevRequired.length === 0 ||
                        (prevSpec?.required_env?.length
                          ? prevRequired.length === prevSpec.required_env.length &&
                            prevRequired.every((e, i) => e === prevSpec.required_env![i])
                          : false);
                      const nextRequired =
                        nextSpec?.required_env?.length && wasTemplateDriven
                          ? nextSpec.required_env
                          : prevRequired;
                      if (prev.runtime) {
                        return {
                          ...prev,
                          runtime_config: {
                            ...prev.runtime_config,
                            model: v,
                            ...(nextSpec?.required_env?.length && wasTemplateDriven
                              ? { required_env: nextRequired }
                              : {}),
                          },
                        };
                      }
                      return { ...prev, model: v };
                    });
                  }}
                  placeholder="e.g. anthropic:claude-sonnet-4-6"
                  className="w-full bg-zinc-800 border border-zinc-700 rounded px-2 py-1 text-xs text-zinc-200 font-mono focus:outline-none focus:border-blue-500"
                />
                {availableModels.length > 0 && (
                  <datalist id={`${runtimeId}-models`}>
                    {availableModels.map((m, i) => (
                      <option key={`${m.id}-${i}`} value={m.id}>{m.name || m.id}</option>
                    ))}
                  </datalist>
                )}
              </div>
            </div>
            <TagList
              label={
                currentModelSpec?.required_env?.length &&
                arraysEqual(config.runtime_config?.required_env ?? [], currentModelSpec.required_env)
                  ? "Required Env Var Names (from template)"
                  : "Required Env Var Names"
              }
              values={config.runtime_config?.required_env ?? []}
              onChange={(v) => updateNested("runtime_config" as keyof ConfigData, "required_env", v)}
              placeholder="variable NAME (e.g. ANTHROPIC_API_KEY) — not the value"
            />
            <p className="text-[10px] text-zinc-500 mt-1">
              This declares which env var <em>names</em> the workspace needs.
              Set the actual values in the <strong>Secrets</strong> section
              below — those are encrypted and mounted into the container at
              runtime.
            </p>
            {currentModelSpec?.required_env?.length &&
              !arraysEqual(config.runtime_config?.required_env ?? [], currentModelSpec.required_env) && (
              <div className="text-[10px] text-zinc-500 mt-1 flex items-center gap-2">
                <span>
                  Template suggests{" "}
                  <code className="text-zinc-400">{currentModelSpec.required_env.join(", ")}</code>{" "}
                  for <code className="text-zinc-400">{currentModelSpec.name || currentModelSpec.id}</code>.
                </span>
                <button
                  type="button"
                  onClick={() => updateNested("runtime_config" as keyof ConfigData, "required_env", currentModelSpec.required_env)}
                  className="text-blue-400 hover:text-blue-300 underline"
                >
                  Apply
                </button>
              </div>
            )}
          </Section>

          {/* Claude Settings — shown for claude-code runtime or claude/anthropic model names */}
          {(config.runtime === "claude-code" ||
            (config.runtime_config?.model || config.model || "").toLowerCase().includes("claude") ||
            (config.runtime_config?.model || config.model || "").toLowerCase().includes("anthropic")) && (
            <Section title="Claude Settings" defaultOpen={false}>
              <div>
                <label htmlFor={effortId} className="text-[10px] text-zinc-500 block mb-1">
                  Effort
                  <span className="ml-1 text-zinc-600">(output_config.effort — Opus 4.7+)</span>
                </label>
                <select
                  id={effortId}
                  value={config.effort || ""}
                  onChange={(e) => update("effort", e.target.value)}
                  className="w-full bg-zinc-800 border border-zinc-700 rounded px-2 py-1 text-xs text-zinc-200 focus:outline-none focus:border-blue-500"
                  data-testid="effort-select"
                >
                  <option value="">— unset (model default) —</option>
                  <option value="low">low</option>
                  <option value="medium">medium</option>
                  <option value="high">high</option>
                  <option value="xhigh">xhigh (extended thinking)</option>
                  <option value="max">max — absolute ceiling</option>
                </select>
              </div>
              <div>
                <label htmlFor={taskBudgetId} className="text-[10px] text-zinc-500 block mb-1">
                  Task Budget (tokens)
                  <span className="ml-1 text-zinc-600">(output_config.task_budget.total — 0 = unset)</span>
                </label>
                <input
                  id={taskBudgetId}
                  type="number"
                  min={0}
                  step={1000}
                  value={config.task_budget ?? 0}
                  onChange={(e) => update("task_budget", parseInt(e.target.value, 10) || 0)}
                  placeholder="0"
                  className="w-full bg-zinc-800 border border-zinc-700 rounded px-2 py-1 text-xs text-zinc-200 focus:outline-none focus:border-blue-500 font-mono"
                  data-testid="task-budget-input"
                />
              </div>
            </Section>
          )}

          <Section title="Skills & Tools" defaultOpen={false}>
            <TagList label="Skills" values={config.skills || []} onChange={(v) => update("skills", v)} placeholder="e.g. code-review" />
            <TagList label="Tools" values={config.tools || []} onChange={(v) => update("tools", v)} placeholder="e.g. web_search, filesystem" />
            <TagList label="Prompt Files" values={config.prompt_files || []} onChange={(v) => update("prompt_files", v)} placeholder="e.g. system-prompt.md" />
            <TagList label="Shared Context" values={config.shared_context || []} onChange={(v) => update("shared_context", v)} placeholder="e.g. architecture.md" />
          </Section>

          <Section title="A2A Protocol" defaultOpen={false}>
            <NumberInput label="Port" value={config.a2a?.port ?? 8000} onChange={(v) => updateNested("a2a" as keyof ConfigData, "port", v)} />
            <Toggle label="Streaming" checked={config.a2a?.streaming ?? true} onChange={(v) => updateNested("a2a" as keyof ConfigData, "streaming", v)} />
            <Toggle label="Push Notifications" checked={config.a2a?.push_notifications ?? true} onChange={(v) => updateNested("a2a" as keyof ConfigData, "push_notifications", v)} />
          </Section>

          <Section title="Delegation" defaultOpen={false}>
            <div className="grid grid-cols-2 gap-3">
              <NumberInput label="Retry Attempts" value={config.delegation?.retry_attempts ?? 3} onChange={(v) => updateNested("delegation" as keyof ConfigData, "retry_attempts", v)} min={0} max={10} />
              <NumberInput label="Retry Delay (s)" value={config.delegation?.retry_delay ?? 5} onChange={(v) => updateNested("delegation" as keyof ConfigData, "retry_delay", v)} min={1} />
            </div>
            <NumberInput label="Timeout (s)" value={config.delegation?.timeout ?? 120} onChange={(v) => updateNested("delegation" as keyof ConfigData, "timeout", v)} min={10} />
            <Toggle label="Escalate on failure" checked={config.delegation?.escalate ?? true} onChange={(v) => updateNested("delegation" as keyof ConfigData, "escalate", v)} />
          </Section>

          <Section title="Sandbox" defaultOpen={false}>
            <div>
              <label htmlFor={sandboxBackendId} className="text-[10px] text-zinc-500 block mb-1">Backend</label>
              <select
                id={sandboxBackendId}
                value={config.sandbox?.backend || "docker"}
                onChange={(e) => updateNested("sandbox" as keyof ConfigData, "backend", e.target.value)}
                className="w-full bg-zinc-800 border border-zinc-700 rounded px-2 py-1 text-xs text-zinc-200 focus:outline-none focus:border-blue-500"
              >
                <option value="subprocess">subprocess</option>
                <option value="docker">docker</option>
                <option value="e2b">e2b</option>
              </select>
            </div>
            <div className="grid grid-cols-2 gap-3">
              <TextInput label="Memory Limit" value={config.sandbox?.memory_limit || "256m"} onChange={(v) => updateNested("sandbox" as keyof ConfigData, "memory_limit", v)} mono />
              <NumberInput label="Timeout (s)" value={config.sandbox?.timeout ?? 30} onChange={(v) => updateNested("sandbox" as keyof ConfigData, "timeout", v)} min={5} />
            </div>
          </Section>

          <SecretsSection
            workspaceId={workspaceId}
            requiredEnv={config.runtime_config?.required_env}
          />

          <AgentCardSection workspaceId={workspaceId} />
        </div>
      )}

      {error && (
        <div className="mx-3 mb-2 px-3 py-1.5 bg-red-900/30 border border-red-800 rounded text-xs text-red-400">{error}</div>
      )}
      {success && (
        <div className="mx-3 mb-2 px-3 py-1.5 bg-green-900/30 border border-green-800 rounded text-xs text-green-400">Saved</div>
      )}

      <div className="p-3 border-t border-zinc-800 flex gap-2">
        <button
          onClick={() => handleSave(true)}
          disabled={!isDirty || saving}
          className="px-3 py-1.5 bg-blue-600 hover:bg-blue-500 text-xs rounded text-white disabled:opacity-30 transition-colors"
        >
          {saving ? "Restarting..." : "Save & Restart"}
        </button>
        <button
          onClick={() => handleSave(false)}
          disabled={!isDirty || saving}
          className="px-3 py-1.5 bg-zinc-700 hover:bg-zinc-600 text-xs rounded text-zinc-300 disabled:opacity-30 transition-colors"
        >
          Save
        </button>
        <button
          onClick={loadConfig}
          className="px-3 py-1.5 bg-zinc-700 hover:bg-zinc-600 text-xs rounded text-zinc-300 ml-auto"
        >
          Reload
        </button>
      </div>
    </div>
  );
}
