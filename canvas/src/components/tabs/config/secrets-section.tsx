"use client";

import { useState, useEffect, useCallback } from "react";
import { api } from "@/lib/api";
import { Section } from "./form-inputs";
import { markAllWorkspacesNeedRestart } from "@/lib/canvas-actions";

interface SecretEntry {
  key: string;
  has_value: boolean;
  created_at: string;
  updated_at: string;
  scope?: "global" | "workspace";
}

const COMMON_KEYS = [
  { key: "ANTHROPIC_API_KEY", label: "Anthropic API Key" },
  { key: "OPENAI_API_KEY", label: "OpenAI API Key" },
  { key: "GOOGLE_API_KEY", label: "Google AI API Key" },
  { key: "SERP_API_KEY", label: "SERP API Key" },
  { key: "MODEL_PROVIDER", label: "Model Override (e.g. anthropic:claude-sonnet-4-6)" },
];

function ScopeBadge({ scope }: { scope: "global" | "workspace" | "override" }) {
  if (scope === "global") {
    return <span className="text-[8px] text-amber-400 bg-amber-900/30 px-1.5 py-0.5 rounded" title="Inherited from global secrets">Global</span>;
  }
  if (scope === "override") {
    return <span className="text-[8px] text-purple-400 bg-purple-900/30 px-1.5 py-0.5 rounded" title="Overrides global secret">Override</span>;
  }
  return null;
}

function SecretRow({ label, secretKey, isSet, scope, globalMode, onSave, onDelete }: {
  label: string; secretKey: string; isSet: boolean;
  scope?: "global" | "workspace" | "override";
  globalMode?: boolean;
  onSave: (value: string) => void; onDelete: () => void;
}) {
  const [editing, setEditing] = useState(false);
  const [value, setValue] = useState("");

  const actionLabel = (): string => {
    if (editing) return "Cancel";
    if (!isSet) return "Set";
    if (globalMode) return "Update";
    if (scope === "global") return "Override";
    return "Update";
  };

  const isPlaintext = secretKey === "MODEL_PROVIDER";

  return (
    <div className="bg-zinc-800/50 rounded px-3 py-2 border border-zinc-700/50">
      <div className="flex items-center justify-between">
        <div className="min-w-0">
          <div className="text-[10px] text-zinc-300">{label}</div>
          <div className="flex items-center gap-2 mt-0.5">
            <span className="text-[9px] font-mono text-zinc-600">{secretKey}</span>
            {isSet && (
              <span className="text-[9px] font-mono text-zinc-500 tracking-widest" title="Value is set (encrypted)">
                •••••
              </span>
            )}
          </div>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          {isSet && <span className="text-[8px] text-green-500 bg-green-900/30 px-1.5 py-0.5 rounded">Set</span>}
          {scope && <ScopeBadge scope={scope} />}
          {!editing && isSet && (globalMode || scope !== "global") && (
            <button onClick={onDelete} className="text-[9px] text-red-400 hover:text-red-300">Remove</button>
          )}
          <button onClick={() => setEditing(!editing)} className="text-[9px] text-blue-400 hover:text-blue-300">
            {actionLabel()}
          </button>
        </div>
      </div>
      {editing && (
        <div className="flex gap-2 mt-2">
          <input
            value={value} onChange={(e) => setValue(e.target.value)}
            placeholder={isPlaintext ? "anthropic:claude-sonnet-4-6" : "sk-..."}
            type={isPlaintext ? "text" : "password"} autoFocus
            className="flex-1 bg-zinc-900 border border-zinc-600 rounded px-2 py-1 text-[10px] text-zinc-100 font-mono focus:outline-none focus:border-blue-500"
          />
          <button
            onClick={() => { onSave(value); setEditing(false); setValue(""); }}
            disabled={!value}
            className="px-2 py-1 bg-blue-600 hover:bg-blue-500 text-[10px] rounded text-white disabled:opacity-30"
          >Save</button>
        </div>
      )}
    </div>
  );
}

function CustomSecretRow({ secretKey, scope, globalMode, onSave, onDelete }: {
  secretKey: string;
  scope: "global" | "workspace" | "override";
  globalMode?: boolean;
  onSave: (value: string) => void;
  onDelete: () => void;
}) {
  const [editing, setEditing] = useState(false);
  const [value, setValue] = useState("");

  const canDelete = globalMode || scope !== "global";
  const showOverride = !globalMode && scope === "global";

  return (
    <div className="py-1.5 px-2">
      <div className="flex items-center justify-between">
        <div className="min-w-0">
          <span className={`text-[10px] font-mono ${globalMode ? "text-amber-400" : scope === "global" ? "text-zinc-400" : "text-blue-400"}`}>
            {secretKey}
          </span>
          <span className="text-[9px] font-mono text-zinc-500 tracking-widest ml-2">•••••</span>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          <span className="text-[8px] text-green-500">Set</span>
          {!globalMode && <ScopeBadge scope={scope} />}
          {canDelete && !editing && (
            <button onClick={onDelete} className="text-[9px] text-red-400 hover:text-red-300">Remove</button>
          )}
          {(canDelete || showOverride) && (
            <button onClick={() => setEditing(!editing)} className="text-[9px] text-blue-400 hover:text-blue-300">
              {editing ? "Cancel" : showOverride ? "Override" : "Update"}
            </button>
          )}
        </div>
      </div>
      {editing && (
        <div className="flex gap-2 mt-1.5">
          <input
            value={value} onChange={(e) => setValue(e.target.value)}
            placeholder="New value" type="password" autoFocus
            className="flex-1 bg-zinc-900 border border-zinc-600 rounded px-2 py-1 text-[10px] text-zinc-100 font-mono focus:outline-none focus:border-blue-500"
          />
          <button
            onClick={() => { onSave(value); setEditing(false); setValue(""); }}
            disabled={!value}
            className="px-2 py-1 bg-blue-600 hover:bg-blue-500 text-[10px] rounded text-white disabled:opacity-30"
          >Save</button>
        </div>
      )}
    </div>
  );
}

export function SecretsSection({ workspaceId }: { workspaceId: string }) {
  const [mergedSecrets, setMergedSecrets] = useState<SecretEntry[]>([]);
  const [globalSecrets, setGlobalSecrets] = useState<SecretEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showAdd, setShowAdd] = useState(false);
  const [newKey, setNewKey] = useState("");
  const [newValue, setNewValue] = useState("");
  const [globalMode, setGlobalMode] = useState(false);

  const loadSecrets = useCallback(async () => {
    setLoading(true);
    try {
      const [merged, global] = await Promise.all([
        api.get<SecretEntry[]>(`/workspaces/${workspaceId}/secrets`).catch(() => []),
        api.get<SecretEntry[]>("/settings/secrets").catch(() => []),
      ]);
      setMergedSecrets(merged);
      setGlobalSecrets(global);
    } catch {
      setMergedSecrets([]);
      setGlobalSecrets([]);
    } finally {
      setLoading(false);
    }
  }, [workspaceId]);

  useEffect(() => { loadSecrets(); }, [loadSecrets]);

  const handleSave = async (key: string, value: string) => {
    setError(null);
    try {
      if (globalMode) {
        await api.put("/settings/secrets", { key, value });
        markAllWorkspacesNeedRestart();
      } else {
        await api.put(`/workspaces/${workspaceId}/secrets`, { key, value });
      }
      setNewKey(""); setNewValue(""); setShowAdd(false);
      loadSecrets();
    } catch (e) { setError(e instanceof Error ? e.message : "Failed to save"); }
  };

  const handleDelete = async (key: string) => {
    setError(null);
    try {
      if (globalMode) {
        await api.del(`/settings/secrets/${encodeURIComponent(key)}`);
        markAllWorkspacesNeedRestart();
      } else {
        await api.del(`/workspaces/${workspaceId}/secrets/${encodeURIComponent(key)}`);
      }
      loadSecrets();
    } catch (e) { setError(e instanceof Error ? e.message : "Failed to delete"); }
  };

  // Build lookup sets from the merged view
  const mergedByKey = new Map(mergedSecrets.map((s) => [s.key, s]));
  const globalKeys = new Set(globalSecrets.map((s) => s.key));

  /** Determine scope badge for the workspace (non-global) view */
  const getScope = (entry: SecretEntry): "global" | "workspace" | "override" => {
    if (entry.scope === "workspace" && globalKeys.has(entry.key)) return "override";
    if (entry.scope === "global") return "global";
    return "workspace";
  };

  // For workspace view: use merged secrets from the backend (includes inherited globals)
  // For global view: use global secrets only
  const activeSecrets = globalMode ? globalSecrets : mergedSecrets;

  // Split into common keys and custom keys
  const commonKeySet = new Set(COMMON_KEYS.map((c) => c.key));
  const customSecrets = activeSecrets.filter((s) => !commonKeySet.has(s.key));

  return (
    <Section title="Secrets & API Keys" defaultOpen={false}>
      {loading ? (
        <div className="text-[10px] text-zinc-500">Loading secrets...</div>
      ) : (
        <div className="space-y-2">
          {error && <div className="px-2 py-1 bg-red-900/30 border border-red-800 rounded text-[10px] text-red-400">{error}</div>}

          {/* Scope toggle */}
          <div className="flex items-center gap-2 pb-1">
            <button
              onClick={() => setGlobalMode(false)}
              className={`text-[10px] px-2 py-0.5 rounded transition-colors ${
                !globalMode ? "bg-blue-600/20 text-blue-300 border border-blue-500/30" : "text-zinc-500 hover:text-zinc-300"
              }`}
            >
              This Workspace
            </button>
            <button
              onClick={() => setGlobalMode(true)}
              className={`text-[10px] px-2 py-0.5 rounded transition-colors ${
                globalMode ? "bg-amber-600/20 text-amber-300 border border-amber-500/30" : "text-zinc-500 hover:text-zinc-300"
              }`}
            >
              Global (All Workspaces)
            </button>
          </div>

          {globalMode && (
            <div className="px-2 py-1.5 bg-amber-950/20 border border-amber-800/30 rounded text-[10px] text-amber-400/80 leading-relaxed">
              Global keys apply to all workspaces. Workspace-level keys override globals with the same name.
            </div>
          )}

          {/* Common keys */}
          {COMMON_KEYS.map(({ key, label }) => {
            const entry = globalMode
              ? globalSecrets.find((s) => s.key === key)
              : mergedByKey.get(key);
            const isSet = !!entry?.has_value;
            const scope = globalMode ? undefined : (entry ? getScope(entry) : undefined);
            return (
              <SecretRow key={key} label={label} secretKey={key}
                isSet={isSet}
                scope={scope}
                globalMode={globalMode}
                onSave={(v) => handleSave(key, v)} onDelete={() => handleDelete(key)} />
            );
          })}

          {/* Custom secrets */}
          {customSecrets.map((s) => {
            const scope = globalMode ? ("global" as const) : getScope(s);
            return (
              <CustomSecretRow key={s.key} secretKey={s.key}
                scope={scope}
                globalMode={globalMode}
                onSave={(v) => handleSave(s.key, v)} onDelete={() => handleDelete(s.key)} />
            );
          })}

          {/* Add new */}
          {showAdd ? (
            <div className="bg-zinc-800/50 rounded p-2 space-y-1.5 border border-zinc-700/50">
              <input value={newKey} onChange={(e) => setNewKey(e.target.value.toUpperCase())} placeholder="KEY_NAME"
                className="w-full bg-zinc-900 border border-zinc-600 rounded px-2 py-1 text-[10px] font-mono text-zinc-100 focus:outline-none focus:border-blue-500" />
              <input value={newValue} onChange={(e) => setNewValue(e.target.value)} placeholder="Value" type="password"
                className="w-full bg-zinc-900 border border-zinc-600 rounded px-2 py-1 text-[10px] text-zinc-100 focus:outline-none focus:border-blue-500" />
              <div className="flex gap-2">
                <button onClick={() => { if (newKey && newValue) handleSave(newKey, newValue); }} disabled={!newKey || !newValue}
                  className="px-2 py-1 bg-blue-600 hover:bg-blue-500 text-[10px] rounded text-white disabled:opacity-30">
                  Save{globalMode ? " (Global)" : ""}
                </button>
                <button onClick={() => { setShowAdd(false); setNewKey(""); setNewValue(""); }}
                  className="px-2 py-1 bg-zinc-700 hover:bg-zinc-600 text-[10px] rounded text-zinc-300">Cancel</button>
              </div>
            </div>
          ) : (
            <button onClick={() => setShowAdd(true)} className="text-[10px] text-blue-400 hover:text-blue-300">
              + Add {globalMode ? "Global " : ""}Variable
            </button>
          )}

          <div className="text-[9px] text-zinc-600 pt-1">
            Values are encrypted and never exposed to the browser.
            {globalMode
              ? " Global keys are shared across all workspaces. Restart workspaces to apply changes."
              : " Global keys are inherited; workspace keys override globals with the same name."}
          </div>
        </div>
      )}
    </Section>
  );
}
