"use client";

import { useState, useEffect, useCallback } from "react";
import { api } from "@/lib/api";
import { getKeyLabel } from "@/lib/deploy-preflight";

interface Props {
  open: boolean;
  missingKeys: string[];
  runtime: string;
  /** Called when user adds all keys and wants to proceed with deploy. */
  onKeysAdded: () => void;
  /** Called when user cancels the deploy. */
  onCancel: () => void;
  /** Called when user wants to open the Settings Panel (Config tab → Secrets). */
  onOpenSettings?: () => void;
  /** Optional workspace ID — if provided, secrets are saved at workspace scope. */
  workspaceId?: string;
}

interface KeyEntry {
  key: string;
  label: string;
  value: string;
  saved: boolean;
  saving: boolean;
  error: string | null;
}

export function MissingKeysModal({
  open,
  missingKeys,
  runtime,
  onKeysAdded,
  onCancel,
  onOpenSettings,
  workspaceId,
}: Props) {
  const [entries, setEntries] = useState<KeyEntry[]>([]);
  const [globalError, setGlobalError] = useState<string | null>(null);

  // Initialize entries when modal opens or missingKeys change
  useEffect(() => {
    if (!open) return;
    setEntries(
      missingKeys.map((key) => ({
        key,
        label: getKeyLabel(key),
        value: "",
        saved: false,
        saving: false,
        error: null,
      })),
    );
    setGlobalError(null);
  }, [open, missingKeys]);

  // Keyboard handler
  useEffect(() => {
    if (!open) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") onCancel();
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [open, onCancel]);

  const updateEntry = useCallback(
    (index: number, updates: Partial<KeyEntry>) => {
      setEntries((prev) =>
        prev.map((entry, i) => (i === index ? { ...entry, ...updates } : entry)),
      );
    },
    [],
  );

  const handleSaveKey = useCallback(
    async (index: number) => {
      const entry = entries[index];
      if (!entry.value.trim()) return;

      updateEntry(index, { saving: true, error: null });

      try {
        // Save to global scope by default (available to all workspaces)
        if (workspaceId) {
          await api.put(`/workspaces/${workspaceId}/secrets`, {
            key: entry.key,
            value: entry.value.trim(),
          });
        } else {
          await api.put("/settings/secrets", {
            key: entry.key,
            value: entry.value.trim(),
          });
        }
        updateEntry(index, { saved: true, saving: false });
      } catch (e) {
        updateEntry(index, {
          saving: false,
          error: e instanceof Error ? e.message : "Failed to save",
        });
      }
    },
    [entries, updateEntry, workspaceId],
  );

  const handleAddKeysAndDeploy = useCallback(() => {
    const anySaving = entries.some((e) => e.saving);
    if (anySaving) {
      setGlobalError("Please wait for all keys to finish saving.");
      return;
    }
    const allSaved = entries.every((e) => e.saved);
    if (!allSaved) {
      setGlobalError("Please save all required keys before deploying.");
      return;
    }
    onKeysAdded();
  }, [entries, onKeysAdded]);

  if (!open) return null;

  const allSaved = entries.every((e) => e.saved);
  const anySaving = entries.some((e) => e.saving);
  const runtimeLabel = runtime.replace(/[-_]/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/70 backdrop-blur-sm"
        onClick={onCancel}
      />

      {/* Dialog */}
      <div className="relative bg-zinc-900 border border-zinc-700 rounded-xl shadow-2xl shadow-black/50 max-w-[440px] w-full mx-4 overflow-hidden">
        {/* Header */}
        <div className="px-5 py-4 border-b border-zinc-800">
          <div className="flex items-center gap-2 mb-1">
            <div className="w-5 h-5 rounded-md bg-amber-600/20 border border-amber-500/30 flex items-center justify-center">
              <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
                <path
                  d="M6 1L11 10H1L6 1Z"
                  stroke="#fbbf24"
                  strokeWidth="1.2"
                  strokeLinejoin="round"
                />
                <path d="M6 5V7" stroke="#fbbf24" strokeWidth="1.2" strokeLinecap="round" />
                <circle cx="6" cy="8.5" r="0.5" fill="#fbbf24" />
              </svg>
            </div>
            <h3 className="text-sm font-semibold text-zinc-100">
              Missing API Keys
            </h3>
          </div>
          <p className="text-[12px] text-zinc-400 leading-relaxed">
            The <span className="text-amber-300 font-medium">{runtimeLabel}</span> runtime
            requires the following keys to be configured before deploying.
          </p>
        </div>

        {/* Body — key list */}
        <div className="px-5 py-4 space-y-3 max-h-[50vh] overflow-y-auto">
          {entries.map((entry, index) => (
            <div
              key={entry.key}
              className="bg-zinc-800/50 rounded-lg px-3 py-2.5 border border-zinc-700/50"
            >
              <div className="flex items-center justify-between mb-1">
                <div>
                  <div className="text-[11px] text-zinc-300 font-medium">
                    {entry.label}
                  </div>
                  <div className="text-[9px] font-mono text-zinc-600">
                    {entry.key}
                  </div>
                </div>
                {entry.saved && (
                  <span className="text-[9px] text-emerald-400 bg-emerald-900/30 px-1.5 py-0.5 rounded flex items-center gap-1">
                    <svg width="8" height="8" viewBox="0 0 8 8" fill="none">
                      <path d="M1.5 4L3.5 6L6.5 2" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" strokeLinejoin="round" />
                    </svg>
                    Saved
                  </span>
                )}
              </div>

              {!entry.saved && (
                <div className="flex gap-2 mt-2">
                  <input
                    value={entry.value}
                    onChange={(e) => updateEntry(index, { value: e.target.value.trimStart() })}
                    placeholder={entry.key.includes("API_KEY") ? "sk-..." : "Enter value"}
                    type="password"
                    autoFocus={index === 0}
                    onKeyDown={(e) => {
                      if (e.key === "Enter" && entry.value.trim()) {
                        handleSaveKey(index);
                      }
                    }}
                    className="flex-1 bg-zinc-900 border border-zinc-600 rounded px-2 py-1.5 text-[11px] text-zinc-100 font-mono focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500/20 transition-colors"
                  />
                  <button
                    onClick={() => handleSaveKey(index)}
                    disabled={!entry.value.trim() || entry.saving}
                    className="px-3 py-1.5 bg-blue-600 hover:bg-blue-500 text-[11px] rounded text-white disabled:opacity-30 transition-colors shrink-0"
                  >
                    {entry.saving ? "..." : "Save"}
                  </button>
                </div>
              )}

              {entry.error && (
                <div className="mt-1.5 text-[10px] text-red-400">{entry.error}</div>
              )}
            </div>
          ))}

          {globalError && (
            <div className="px-3 py-2 bg-red-950/40 border border-red-800/50 rounded-lg text-[11px] text-red-400">
              {globalError}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="px-5 py-3 border-t border-zinc-800 bg-zinc-950/50 flex items-center justify-between gap-2">
          <div>
            {onOpenSettings && (
              <button
                onClick={onOpenSettings}
                className="text-[11px] text-blue-400 hover:text-blue-300 transition-colors"
              >
                Open Settings Panel
              </button>
            )}
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={onCancel}
              className="px-3.5 py-1.5 text-[12px] text-zinc-400 hover:text-zinc-200 bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 rounded-lg transition-colors"
            >
              Cancel Deploy
            </button>
            <button
              onClick={handleAddKeysAndDeploy}
              disabled={!allSaved || anySaving}
              className="px-3.5 py-1.5 text-[12px] bg-blue-600 hover:bg-blue-500 text-white rounded-lg transition-colors disabled:opacity-40"
            >
              {anySaving ? "Saving..." : allSaved ? "Deploy" : "Add Keys"}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
