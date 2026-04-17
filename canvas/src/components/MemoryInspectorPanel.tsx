'use client';

import { useState, useEffect, useCallback } from "react";
import { api } from "@/lib/api";
import { ConfirmDialog } from "@/components/ConfirmDialog";

// ── Types ─────────────────────────────────────────────────────────────────────

interface MemoryEntry {
  key: string;
  value: unknown;
  version: number;
  /** Omitted by the API when there is no TTL (Go omitempty) */
  expires_at?: string;
  updated_at: string;
}

interface WriteResult {
  status: string;
  key: string;
  version: number;
}

interface Props {
  workspaceId: string;
}

// ── Helpers ───────────────────────────────────────────────────────────────────

function formatRelativeTime(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  if (diff < 60_000) return `${Math.floor(diff / 1000)}s`;
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m`;
  if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)}h`;
  return new Date(iso).toLocaleDateString();
}

// ── Component ─────────────────────────────────────────────────────────────────

export function MemoryInspectorPanel({ workspaceId }: Props) {
  const [entries, setEntries] = useState<MemoryEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Expand/edit/delete state — keyed by entry.key (string primitive, no new objects)
  const [expandedKey, setExpandedKey] = useState<string | null>(null);
  const [editingKey, setEditingKey] = useState<string | null>(null);
  const [editValue, setEditValue] = useState("");
  const [editError, setEditError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [pendingDeleteKey, setPendingDeleteKey] = useState<string | null>(null);

  // ── Data loading ────────────────────────────────────────────────────────────

  const loadEntries = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      // API returns MemoryEntry[] (flat array, never wrapped, never null)
      const data = await api.get<MemoryEntry[]>(`/workspaces/${workspaceId}/memory`);
      setEntries(data);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load memory entries");
      setEntries([]);
    } finally {
      setLoading(false);
    }
  }, [workspaceId]);

  useEffect(() => {
    loadEntries();
  }, [loadEntries]);

  // ── Edit handlers ───────────────────────────────────────────────────────────

  const startEdit = useCallback((entry: MemoryEntry) => {
    setEditingKey(entry.key);
    setEditValue(JSON.stringify(entry.value, null, 2));
    setEditError(null);
  }, []);

  const cancelEdit = useCallback(() => {
    setEditingKey(null);
    setEditValue("");
    setEditError(null);
  }, []);

  const saveEdit = useCallback(
    async (entry: MemoryEntry) => {
      // Validate JSON before touching network
      let parsed: unknown;
      try {
        parsed = JSON.parse(editValue);
      } catch {
        setEditError("Invalid JSON — fix the syntax before saving");
        return;
      }

      setSaving(true);
      setEditError(null);

      // Optimistic update — capture rollback snapshot before mutating
      const snapshot = entries;
      setEntries((prev) =>
        prev.map((e) =>
          e.key === entry.key
            ? {
                ...e,
                value: parsed,
                version: e.version + 1,
                updated_at: new Date().toISOString(),
              }
            : e
        )
      );
      setEditingKey(null);
      setEditValue("");

      try {
        await api.post<WriteResult>(`/workspaces/${workspaceId}/memory`, {
          key: entry.key,
          value: parsed,
          if_match_version: entry.version,
        });
      } catch (e) {
        // Roll back optimistic update on any error
        setEntries(snapshot);
        setEditingKey(entry.key);
        setEditValue(JSON.stringify(entry.value, null, 2));
        const msg = e instanceof Error ? e.message : "Save failed";
        if (msg.includes("409") || msg.toLowerCase().includes("mismatch")) {
          setEditError("Version conflict — entry changed elsewhere. Reload to see latest.");
        } else {
          setEditError(msg);
        }
      } finally {
        setSaving(false);
      }
    },
    [entries, editValue, workspaceId]
  );

  // ── Delete handlers ─────────────────────────────────────────────────────────

  const confirmDelete = useCallback(async () => {
    if (!pendingDeleteKey) return;
    const key = pendingDeleteKey;
    setPendingDeleteKey(null);

    // Optimistic removal
    setEntries((prev) => prev.filter((e) => e.key !== key));
    if (expandedKey === key) setExpandedKey(null);

    try {
      await api.del(`/workspaces/${workspaceId}/memory/${encodeURIComponent(key)}`);
    } catch (e) {
      // On failure, reload to restore the true state
      setError(e instanceof Error ? e.message : "Delete failed — reloading...");
      await loadEntries();
    }
  }, [pendingDeleteKey, expandedKey, workspaceId, loadEntries]);

  // ── Render ──────────────────────────────────────────────────────────────────

  if (loading) {
    return (
      <div className="flex items-center justify-center h-32">
        <span className="text-xs text-zinc-500">Loading memory…</span>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full">
      {/* Toolbar */}
      <div className="px-4 py-3 border-b border-zinc-800/40 flex items-center justify-between shrink-0">
        <span className="text-[11px] text-zinc-500">
          {entries.length === 1 ? "1 entry" : `${entries.length} entries`}
        </span>
        <button
          onClick={loadEntries}
          className="px-2 py-1 text-[11px] bg-zinc-800 hover:bg-zinc-700 text-zinc-300 rounded transition-colors"
          aria-label="Refresh memory entries"
        >
          ↻ Refresh
        </button>
      </div>

      {/* Error banner */}
      {error && (
        <div className="mx-4 mt-3 px-3 py-2 bg-red-950/30 border border-red-800/40 rounded text-xs text-red-400">
          {error}
        </div>
      )}

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-4">
        {entries.length === 0 ? (
          /* Empty state */
          <div className="flex flex-col items-center justify-center py-16 gap-3 text-center">
            <span className="text-4xl text-zinc-700" aria-hidden="true">◇</span>
            <p className="text-sm font-medium text-zinc-400">No memory entries yet</p>
            <p className="text-[11px] text-zinc-600 max-w-[200px] leading-relaxed">
              Memory entries will appear here when the workspace writes to its KV store.
            </p>
          </div>
        ) : (
          <div className="space-y-1.5">
            {entries.map((entry) => {
              const isExpanded = expandedKey === entry.key;
              const isEditing = editingKey === entry.key;
              return (
                <MemoryEntryRow
                  key={entry.key}
                  entry={entry}
                  isExpanded={isExpanded}
                  isEditing={isEditing}
                  editValue={editValue}
                  editError={editError}
                  saving={saving}
                  onToggle={() => {
                    const next = isExpanded ? null : entry.key;
                    setExpandedKey(next);
                    if (!next && isEditing) cancelEdit();
                  }}
                  onEditValueChange={setEditValue}
                  onStartEdit={() => startEdit(entry)}
                  onSave={() => saveEdit(entry)}
                  onCancelEdit={cancelEdit}
                  onDelete={() => setPendingDeleteKey(entry.key)}
                />
              );
            })}
          </div>
        )}
      </div>

      {/* Delete confirmation dialog */}
      <ConfirmDialog
        open={pendingDeleteKey !== null}
        title="Delete memory entry"
        message={`Delete key "${pendingDeleteKey}"? This cannot be undone.`}
        confirmLabel="Delete"
        confirmVariant="danger"
        onConfirm={confirmDelete}
        onCancel={() => setPendingDeleteKey(null)}
      />
    </div>
  );
}

// ── MemoryEntryRow sub-component ──────────────────────────────────────────────

interface MemoryEntryRowProps {
  entry: MemoryEntry;
  isExpanded: boolean;
  isEditing: boolean;
  editValue: string;
  editError: string | null;
  saving: boolean;
  onToggle: () => void;
  onEditValueChange: (v: string) => void;
  onStartEdit: () => void;
  onSave: () => void;
  onCancelEdit: () => void;
  onDelete: () => void;
}

function MemoryEntryRow({
  entry,
  isExpanded,
  isEditing,
  editValue,
  editError,
  saving,
  onToggle,
  onEditValueChange,
  onStartEdit,
  onSave,
  onCancelEdit,
  onDelete,
}: MemoryEntryRowProps) {
  return (
    <div className="rounded-lg border border-zinc-800/60 bg-zinc-900/50 overflow-hidden">
      {/* Header row — click to expand/collapse */}
      <button
        className="w-full flex items-center gap-2 px-3 py-2.5 text-left hover:bg-zinc-800/30 transition-colors"
        onClick={onToggle}
        aria-expanded={isExpanded}
      >
        <span className="text-[10px] font-mono text-blue-400 truncate flex-1 min-w-0">
          {entry.key}
        </span>
        <span className="text-[9px] text-zinc-600 shrink-0 font-mono">
          v{entry.version}
        </span>
        <span className="text-[9px] text-zinc-600 shrink-0">
          {formatRelativeTime(entry.updated_at)}
        </span>
        <span className="text-[9px] text-zinc-500 shrink-0" aria-hidden="true">
          {isExpanded ? "▼" : "▶"}
        </span>
      </button>

      {/* Expanded body */}
      {isExpanded && (
        <div className="border-t border-zinc-800/50 px-3 pb-3 pt-2 space-y-2">
          {entry.expires_at && (
            <p className="text-[9px] text-zinc-500">
              Expires: {new Date(entry.expires_at).toLocaleString()}
            </p>
          )}

          {isEditing ? (
            /* Edit mode */
            <div className="space-y-2">
              <textarea
                value={editValue}
                onChange={(e) => onEditValueChange(e.target.value)}
                rows={6}
                aria-label="Edit memory value"
                className="w-full bg-zinc-950 border border-zinc-700 focus:border-blue-500 rounded px-2 py-1.5 text-[11px] font-mono text-zinc-100 focus:outline-none resize-none transition-colors"
              />
              {editError && (
                <p className="text-[10px] text-red-400">{editError}</p>
              )}
              <div className="flex items-center gap-2">
                <button
                  onClick={onSave}
                  disabled={saving}
                  className="px-3 py-1 bg-blue-600 hover:bg-blue-500 disabled:opacity-50 disabled:cursor-not-allowed text-xs rounded text-white transition-colors"
                >
                  {saving ? "Saving…" : "Save"}
                </button>
                <button
                  onClick={onCancelEdit}
                  disabled={saving}
                  className="px-3 py-1 bg-zinc-700 hover:bg-zinc-600 disabled:opacity-50 text-xs rounded text-zinc-300 transition-colors"
                >
                  Cancel
                </button>
              </div>
            </div>
          ) : (
            /* Read mode */
            <div className="space-y-2">
              <pre className="text-[10px] font-mono text-zinc-300 bg-zinc-950 rounded p-2 overflow-x-auto max-h-48 whitespace-pre-wrap break-all">
                {JSON.stringify(entry.value, null, 2)}
              </pre>
              <div className="flex items-center justify-between gap-2">
                <span className="text-[9px] text-zinc-600">
                  Updated: {new Date(entry.updated_at).toLocaleString()}
                </span>
                <div className="flex items-center gap-1.5 shrink-0">
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      onStartEdit();
                    }}
                    aria-label={`Edit ${entry.key}`}
                    className="text-[10px] px-2 py-0.5 bg-zinc-700 hover:bg-zinc-600 rounded text-zinc-300 transition-colors"
                  >
                    Edit
                  </button>
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      onDelete();
                    }}
                    aria-label={`Delete ${entry.key}`}
                    className="text-[10px] px-2 py-0.5 bg-red-950/40 hover:bg-red-900/50 border border-red-900/30 rounded text-red-400 transition-colors"
                  >
                    Delete
                  </button>
                </div>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
