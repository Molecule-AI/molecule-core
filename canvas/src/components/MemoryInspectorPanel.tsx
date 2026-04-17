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
  /**
   * Semantic similarity score (0–1). Only present when the API is queried
   * with ?q=<query> and the pgvector backend has been deployed (issue #776).
   * Absent on plain list fetches — renders gracefully without a badge.
   */
  similarity_score?: number;
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

// ── Skeleton rows — shown during re-fetches when entries already exist ────────

function MemorySkeletonRows() {
  return (
    <div className="space-y-1.5" aria-busy="true" aria-label="Loading entries">
      {Array.from({ length: 3 }).map((_, i) => (
        <div
          key={i}
          className="rounded-lg border border-zinc-800/60 bg-zinc-900/50 px-3 py-3 animate-pulse"
        >
          <div className="flex items-center gap-2">
            <div className="h-2 rounded bg-zinc-700/50 flex-1" />
            <div className="h-2 rounded bg-zinc-700/50 w-8" />
            <div className="h-2 rounded bg-zinc-700/50 w-6" />
            <div className="h-2 rounded bg-zinc-700/50 w-10" />
          </div>
        </div>
      ))}
    </div>
  );
}

// ── Component ─────────────────────────────────────────────────────────────────

export function MemoryInspectorPanel({ workspaceId }: Props) {
  const [entries, setEntries] = useState<MemoryEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // ── Search state ────────────────────────────────────────────────────────────
  /** Raw input value — updated on every keystroke. */
  const [searchQuery, setSearchQuery] = useState("");
  /**
   * Debounced value — drives the API fetch.
   * Lags searchQuery by 300 ms to avoid hammering the endpoint on every key.
   */
  const [debouncedQuery, setDebouncedQuery] = useState("");

  // 300 ms debounce: cancel previous timer whenever searchQuery changes.
  useEffect(() => {
    const timer = setTimeout(
      () => setDebouncedQuery(searchQuery.trim()),
      300
    );
    return () => clearTimeout(timer);
  }, [searchQuery]);

  // ── Expand/edit/delete state (keyed by entry.key — primitives, no new objects)

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
      const url = debouncedQuery
        ? `/workspaces/${workspaceId}/memory?q=${encodeURIComponent(debouncedQuery)}`
        : `/workspaces/${workspaceId}/memory`;
      const data = await api.get<MemoryEntry[]>(url);
      // When a semantic query is active, sort by similarity_score descending.
      // Entries without a score (older backend) fall to the end gracefully.
      const sorted = debouncedQuery
        ? [...data].sort(
            (a, b) => (b.similarity_score ?? 0) - (a.similarity_score ?? 0)
          )
        : data;
      setEntries(sorted);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load memory entries");
      setEntries([]);
    } finally {
      setLoading(false);
    }
  }, [workspaceId, debouncedQuery]);

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
          setEditError(
            "Version conflict — entry changed elsewhere. Reload to see latest."
          );
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
      await api.del(
        `/workspaces/${workspaceId}/memory/${encodeURIComponent(key)}`
      );
    } catch (e) {
      setError(e instanceof Error ? e.message : "Delete failed — reloading...");
      await loadEntries();
    }
  }, [pendingDeleteKey, expandedKey, workspaceId, loadEntries]);

  // ── Render ──────────────────────────────────────────────────────────────────

  // Full-screen loader — only on the very first fetch (no entries cached yet).
  if (loading && entries.length === 0 && !error) {
    return (
      <div className="flex items-center justify-center h-32">
        <span className="text-xs text-zinc-500">Loading memory…</span>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full">
      {/* Search bar */}
      <div className="px-4 pt-3 pb-2 border-b border-zinc-800/40 shrink-0">
        <div className="relative flex items-center">
          {/* Magnifying glass icon */}
          <svg
            width="12"
            height="12"
            viewBox="0 0 16 16"
            fill="none"
            className="absolute left-2.5 text-zinc-500 pointer-events-none shrink-0"
            aria-hidden="true"
          >
            <circle cx="7" cy="7" r="4.5" stroke="currentColor" strokeWidth="1.5" />
            <path d="M11 11l3 3" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
          </svg>
          <input
            type="search"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Semantic search…"
            aria-label="Search memory entries"
            className="w-full bg-zinc-900 border border-zinc-700/60 focus:border-blue-500/60 rounded-lg pl-8 pr-7 py-1.5 text-[11px] text-zinc-200 placeholder-zinc-600 focus:outline-none transition-colors"
          />
          {/* Clear button — only shown when there is a query */}
          {searchQuery && (
            <button
              onClick={() => {
                setSearchQuery("");
                // Skip the debounce delay for clear — reset immediately
                setDebouncedQuery("");
              }}
              aria-label="Clear search"
              className="absolute right-2 text-zinc-500 hover:text-zinc-200 transition-colors text-sm leading-none"
            >
              ×
            </button>
          )}
        </div>
      </div>

      {/* Toolbar */}
      <div className="px-4 py-2.5 border-b border-zinc-800/40 flex items-center justify-between shrink-0">
        <span className="text-[11px] text-zinc-500">
          {debouncedQuery
            ? `${entries.length} result${entries.length !== 1 ? "s" : ""}`
            : entries.length === 1
            ? "1 entry"
            : `${entries.length} entries`}
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
        <div
          role="alert"
          aria-live="assertive"
          className="mx-4 mt-3 px-3 py-2 bg-red-950/30 border border-red-800/40 rounded text-xs text-red-400 shrink-0"
        >
          {error}
        </div>
      )}

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-4">
        {loading ? (
          /* Skeleton rows — visible during search-transition re-fetches */
          <MemorySkeletonRows />
        ) : entries.length === 0 ? (
          debouncedQuery ? (
            /* Search-specific empty state */
            <div className="flex flex-col items-center justify-center py-16 gap-3 text-center">
              <span className="text-4xl text-zinc-700" aria-hidden="true">◇</span>
              <p className="text-sm font-medium text-zinc-400">
                No memories match your search
              </p>
              <p className="text-[11px] text-zinc-600 max-w-[200px] leading-relaxed">
                Try a different query or{" "}
                <button
                  onClick={() => {
                    setSearchQuery("");
                    setDebouncedQuery("");
                  }}
                  className="text-blue-500 hover:text-blue-400 underline transition-colors"
                >
                  clear the search
                </button>
                .
              </p>
            </div>
          ) : (
            /* Default empty state */
            <div className="flex flex-col items-center justify-center py-16 gap-3 text-center">
              <span className="text-4xl text-zinc-700" aria-hidden="true">◇</span>
              <p className="text-sm font-medium text-zinc-400">No memory entries yet</p>
              <p className="text-[11px] text-zinc-600 max-w-[200px] leading-relaxed">
                Memory entries will appear here when the workspace writes to its KV
                store.
              </p>
            </div>
          )
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
        {/* Similarity score badge — only rendered when backend provides a score */}
        {entry.similarity_score != null && (
          <span
            className="text-[9px] text-zinc-500 shrink-0 font-mono tabular-nums"
            title={`Similarity: ${(entry.similarity_score * 100).toFixed(1)}%`}
            data-testid="similarity-badge"
          >
            {Math.round(entry.similarity_score * 100)}%
          </span>
        )}
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
                <p role="alert" aria-live="assertive" className="text-[10px] text-red-400">
                  {editError}
                </p>
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
