"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { api } from "@/lib/api";

interface Props {
  workspaceId: string;
}

interface MemoryEntry {
  key: string;
  value: unknown;
  expires_at: string | null;
  updated_at: string;
}

const AWARENESS_BASE_URL =
  process.env.NEXT_PUBLIC_AWARENESS_URL || "http://localhost:37800";

export function MemoryTab({ workspaceId }: Props) {
  const [entries, setEntries] = useState<MemoryEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [showAwareness, setShowAwareness] = useState(true);
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [expanded, setExpanded] = useState<string | null>(null);
  const [showAdd, setShowAdd] = useState(false);
  const [newKey, setNewKey] = useState("");
  const [newValue, setNewValue] = useState("");
  const [newTTL, setNewTTL] = useState("");
  const [error, setError] = useState<string | null>(null);

  const awarenessUrl = useMemo(() => {
    try {
      const url = new URL(AWARENESS_BASE_URL);
      url.searchParams.set("workspaceId", workspaceId);
      return url.toString();
    } catch {
      return AWARENESS_BASE_URL;
    }
  }, [workspaceId]);

  const awarenessStatus = useMemo(() => {
    try {
      const url = new URL(AWARENESS_BASE_URL);
      return url.origin.includes("localhost") ? "local" : url.hostname;
    } catch {
      return "unavailable";
    }
  }, []);

  const loadMemory = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await api.get<MemoryEntry[]>(`/workspaces/${workspaceId}/memory`);
      setEntries(data);
    } catch (e) {
      setEntries([]);
      setError(e instanceof Error ? e.message : "Failed to load memory");
    } finally {
      setLoading(false);
    }
  }, [workspaceId]);

  useEffect(() => {
    loadMemory();
  }, [loadMemory]);

  const handleAdd = async () => {
    setError(null);
    if (!newKey.trim()) {
      setError("Key is required");
      return;
    }

    let parsedValue: unknown;
    try {
      parsedValue = JSON.parse(newValue);
    } catch {
      parsedValue = newValue;
    }

    const body: Record<string, unknown> = { key: newKey, value: parsedValue };
    if (newTTL) {
      const ttl = parseInt(newTTL);
      if (!Number.isNaN(ttl) && ttl > 0) body.ttl_seconds = ttl;
    }

    try {
      await api.post(`/workspaces/${workspaceId}/memory`, body);
      setNewKey("");
      setNewValue("");
      setNewTTL("");
      setShowAdd(false);
      loadMemory();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to add");
    }
  };

  const handleDelete = async (key: string) => {
    setError(null);
    try {
      await api.del(`/workspaces/${workspaceId}/memory/${encodeURIComponent(key)}`);
      setEntries((prev) => prev.filter((e) => e.key !== key));
      if (expanded === key) setExpanded(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to delete entry");
    }
  };

  const openAwareness = () => {
    window.open(awarenessUrl, "_blank", "noopener,noreferrer");
  };

  if (loading) {
    return <div className="p-4 text-xs text-zinc-500">Loading memory...</div>;
  }

  return (
    <div className="p-4 space-y-4">
      {error && !showAdd && (
        <div className="px-3 py-1.5 bg-red-900/30 border border-red-800 rounded text-xs text-red-400">
          {error}
        </div>
      )}

      <section className="space-y-3">
        <div className="flex items-center justify-between gap-3">
          <div>
            <div className="text-xs font-medium text-zinc-200">Awareness dashboard</div>
            <p className="text-[10px] text-zinc-500">
              Embedded view for the local Awareness memory UI. The current workspace id is appended to the URL for workspace-scoped routing or future filtering.
            </p>
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setShowAwareness((prev) => !prev)}
              className="shrink-0 px-2 py-1 bg-zinc-700 hover:bg-zinc-600 text-[10px] rounded text-zinc-200"
            >
              {showAwareness ? "Collapse" : "Expand"}
            </button>
            <button
              onClick={openAwareness}
              className="shrink-0 px-2 py-1 bg-zinc-700 hover:bg-zinc-600 text-[10px] rounded text-zinc-200"
            >
              Open
            </button>
          </div>
        </div>

        {showAwareness ? (
          AWARENESS_BASE_URL ? (
            <div className="overflow-hidden rounded-xl border border-zinc-800 bg-zinc-900/70 shadow-[0_0_0_1px_rgba(255,255,255,0.02)]">
              <iframe
                title="Awareness dashboard"
                src={awarenessUrl}
                className="h-[520px] w-full border-0"
                loading="lazy"
              />
            </div>
          ) : (
            <div className="rounded-xl border border-dashed border-zinc-800 bg-zinc-900/40 p-4 text-xs text-zinc-500">
              Set <code className="font-mono text-zinc-300">NEXT_PUBLIC_AWARENESS_URL</code> to embed the Awareness dashboard here.
            </div>
          )
        ) : (
          <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 px-4 py-3 flex items-center justify-between gap-3">
            <div className="min-w-0">
              <p className="text-xs text-zinc-200">Awareness dashboard is collapsed</p>
              <p className="text-[10px] text-zinc-500 truncate">
                Workspace context stays linked through <span className="font-mono text-zinc-400">{workspaceId}</span>.
              </p>
            </div>
            <button
              onClick={() => setShowAwareness(true)}
              className="shrink-0 px-2 py-1 bg-blue-600 hover:bg-blue-500 text-[10px] rounded text-white"
            >
              Expand
            </button>
          </div>
        )}

        <div className="grid gap-2 rounded-xl border border-zinc-800 bg-zinc-950/40 px-3 py-2 text-[10px] text-zinc-400 sm:grid-cols-3">
          <div className="flex items-center justify-between gap-2">
            <span className="uppercase tracking-[0.18em] text-zinc-500">Status</span>
            <span className="font-medium text-emerald-300">Connected</span>
          </div>
          <div className="flex items-center justify-between gap-2">
            <span className="uppercase tracking-[0.18em] text-zinc-500">Mode</span>
            <span className="font-medium text-zinc-200">{awarenessStatus}</span>
          </div>
          <div className="flex items-center justify-between gap-2 min-w-0">
            <span className="uppercase tracking-[0.18em] text-zinc-500">Workspace</span>
            <span className="font-mono text-zinc-300 truncate">{workspaceId}</span>
          </div>
        </div>
      </section>

      <section className="space-y-3 border-t border-zinc-800/60 pt-4">
        <div className="flex items-center justify-between">
          <div>
            <div className="text-xs font-medium text-zinc-200">Workspace KV memory</div>
            <p className="text-[10px] text-zinc-500">
              Native platform key-value memory for workspace <span className="font-mono text-zinc-400">{workspaceId}</span>.
            </p>
          </div>
          <div className="flex gap-2">
            <button
              onClick={() => setShowAdvanced((prev) => !prev)}
              className="px-2 py-1 bg-zinc-700 hover:bg-zinc-600 text-[10px] rounded text-zinc-300"
            >
              {showAdvanced ? "Hide Advanced" : "Advanced"}
            </button>
            <button
              onClick={loadMemory}
              className="px-2 py-1 bg-zinc-700 hover:bg-zinc-600 text-[10px] rounded text-zinc-300"
            >
              Refresh
            </button>
            <button
              onClick={() => setShowAdd(!showAdd)}
              className="px-2 py-1 bg-blue-600 hover:bg-blue-500 text-[10px] rounded text-white"
            >
              + Add
            </button>
          </div>
        </div>

        {showAdvanced && showAdd && (
          <div className="bg-zinc-800 rounded p-3 space-y-2 border border-zinc-700">
            <input
              value={newKey}
              onChange={(e) => setNewKey(e.target.value)}
              placeholder="Key"
              className="w-full bg-zinc-900 border border-zinc-600 rounded px-2 py-1 text-xs text-zinc-100 focus:outline-none focus:border-blue-500"
            />
            <textarea
              value={newValue}
              onChange={(e) => setNewValue(e.target.value)}
              placeholder='Value (JSON or plain text)'
              rows={3}
              className="w-full bg-zinc-900 border border-zinc-600 rounded px-2 py-1 text-xs font-mono text-zinc-100 focus:outline-none focus:border-blue-500 resize-none"
            />
            <input
              value={newTTL}
              onChange={(e) => setNewTTL(e.target.value)}
              placeholder="TTL in seconds (optional)"
              className="w-full bg-zinc-900 border border-zinc-600 rounded px-2 py-1 text-xs text-zinc-100 focus:outline-none focus:border-blue-500"
            />
            {error && <div className="text-xs text-red-400">{error}</div>}
            <div className="flex gap-2">
              <button
                onClick={handleAdd}
                className="px-3 py-1 bg-blue-600 hover:bg-blue-500 text-xs rounded text-white"
              >
                Save
              </button>
              <button
                onClick={() => {
                  setShowAdd(false);
                  setError(null);
                }}
                className="px-3 py-1 bg-zinc-700 hover:bg-zinc-600 text-xs rounded text-zinc-300"
              >
                Cancel
              </button>
            </div>
          </div>
        )}

        {showAdvanced ? (
          entries.length === 0 ? (
            <p className="text-xs text-zinc-500 text-center py-4">No memory entries</p>
          ) : (
            <div className="space-y-1">
              {entries.map((entry) => (
                <div key={entry.key} className="bg-zinc-800 rounded border border-zinc-700">
                  <button
                    onClick={() => setExpanded(expanded === entry.key ? null : entry.key)}
                    className="w-full flex items-center justify-between px-3 py-2 text-left"
                  >
                    <span className="text-xs font-mono text-blue-400">{entry.key}</span>
                    <div className="flex items-center gap-2">
                      {entry.expires_at && (
                        <span className="text-[9px] text-zinc-500">
                          TTL {new Date(entry.expires_at).toLocaleString()}
                        </span>
                      )}
                      <span className="text-[10px] text-zinc-500">
                        {expanded === entry.key ? "▼" : "▶"}
                      </span>
                    </div>
                  </button>

                  {expanded === entry.key && (
                    <div className="px-3 pb-2 space-y-2">
                      <pre className="text-[10px] text-zinc-300 bg-zinc-900 rounded p-2 overflow-x-auto max-h-40">
                        {JSON.stringify(entry.value, null, 2)}
                      </pre>
                      <div className="flex items-center justify-between">
                        <span className="text-[9px] text-zinc-500">
                          Updated: {new Date(entry.updated_at).toLocaleString()}
                        </span>
                        <button
                          onClick={() => handleDelete(entry.key)}
                          className="text-[10px] text-red-400 hover:text-red-300"
                        >
                          Delete
                        </button>
                      </div>
                    </div>
                  )}
                </div>
              ))}
            </div>
          )
        ) : (
          <div className="rounded-xl border border-zinc-800 bg-zinc-950/30 px-4 py-3 flex items-center justify-between gap-3">
            <div className="min-w-0">
              <p className="text-xs text-zinc-200">Advanced workspace memory is hidden</p>
              <p className="text-[10px] text-zinc-500 truncate">
                KV entries remain available if you need the raw platform store.
              </p>
            </div>
            <button
              onClick={() => setShowAdvanced(true)}
              className="shrink-0 px-2 py-1 bg-blue-600 hover:bg-blue-500 text-[10px] rounded text-white"
            >
              Show
            </button>
          </div>
        )}
      </section>
    </div>
  );
}
