"use client";

import { useState } from "react";
import { createPortal } from "react-dom";
import { useCanvasStore } from "@/store/canvas";
import { ConfirmDialog } from "./ConfirmDialog";
import { showToast } from "./Toaster";

type BatchAction = "restart" | "pause" | "delete" | null;

export function BatchActionBar() {
  const selectedNodeIds = useCanvasStore((s) => s.selectedNodeIds);
  const clearSelection = useCanvasStore((s) => s.clearSelection);
  const batchRestart = useCanvasStore((s) => s.batchRestart);
  const batchPause = useCanvasStore((s) => s.batchPause);
  const batchDelete = useCanvasStore((s) => s.batchDelete);

  const [pending, setPending] = useState<BatchAction>(null);
  const [busy, setBusy] = useState(false);

  const count = selectedNodeIds.size;
  if (count < 2) return null;

  const confirmMessages: Record<NonNullable<BatchAction>, string> = {
    restart: `Restart ${count} workspace${count !== 1 ? "s" : ""}? Each will briefly go offline while it restarts.`,
    pause:   `Pause ${count} workspace${count !== 1 ? "s" : ""}? Their containers will be stopped.`,
    delete:  `Permanently delete ${count} workspace${count !== 1 ? "s" : ""}? This cannot be undone.`,
  };

  const confirmLabels: Record<NonNullable<BatchAction>, string> = {
    restart: "Restart All",
    pause:   "Pause All",
    delete:  "Delete All",
  };

  async function execute() {
    if (!pending) return;
    setBusy(true);
    try {
      if (pending === "restart") await batchRestart();
      if (pending === "pause")   await batchPause();
      if (pending === "delete")  await batchDelete();
      showToast(`${pending.charAt(0).toUpperCase() + pending.slice(1)} applied to ${count} workspace${count !== 1 ? "s" : ""}`, "success");
      clearSelection();
    } catch {
      showToast(`Batch ${pending} failed`, "error");
    } finally {
      setBusy(false);
      setPending(null);
    }
  }

  const bar = (
    <div
      role="toolbar"
      aria-label="Batch workspace actions"
      className="fixed bottom-6 left-1/2 -translate-x-1/2 z-[200] flex items-center gap-3 px-4 py-2.5 rounded-2xl bg-zinc-900/95 border border-zinc-700/70 shadow-2xl shadow-black/50 backdrop-blur-md"
    >
      {/* Selection count badge */}
      <span className="text-[12px] font-semibold text-zinc-100 bg-blue-600/80 px-2.5 py-0.5 rounded-full tabular-nums">
        {count} selected
      </span>

      <div className="w-px h-5 bg-zinc-700/60" aria-hidden="true" />

      {/* Action buttons */}
      <button
        disabled={busy}
        onClick={() => setPending("restart")}
        className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-[12px] font-medium text-sky-300 bg-sky-900/30 hover:bg-sky-800/50 border border-sky-700/30 hover:border-sky-600/50 transition-colors disabled:opacity-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-sky-500/70"
      >
        <span aria-hidden="true">↻</span>
        Restart All
      </button>

      <button
        disabled={busy}
        onClick={() => setPending("pause")}
        className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-[12px] font-medium text-amber-300 bg-amber-900/30 hover:bg-amber-800/50 border border-amber-700/30 hover:border-amber-600/50 transition-colors disabled:opacity-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-amber-500/70"
      >
        <span aria-hidden="true">⏸</span>
        Pause All
      </button>

      <button
        disabled={busy}
        onClick={() => setPending("delete")}
        className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-[12px] font-medium text-red-300 bg-red-900/30 hover:bg-red-800/50 border border-red-700/30 hover:border-red-600/50 transition-colors disabled:opacity-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-500/70"
      >
        <span aria-hidden="true">✕</span>
        Delete All
      </button>

      <div className="w-px h-5 bg-zinc-700/60" aria-hidden="true" />

      {/* Deselect */}
      <button
        disabled={busy}
        onClick={clearSelection}
        aria-label="Clear selection"
        title="Clear selection (Escape)"
        className="p-1.5 rounded-lg text-[12px] text-zinc-400 hover:text-zinc-200 hover:bg-zinc-700/50 transition-colors disabled:opacity-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-zinc-500/70"
      >
        ✕
      </button>
    </div>
  );

  return (
    <>
      {typeof window !== "undefined" ? createPortal(bar, document.body) : null}

      <ConfirmDialog
        open={!!pending}
        title={pending ? confirmLabels[pending] : ""}
        message={pending ? confirmMessages[pending] : ""}
        confirmLabel={pending ? confirmLabels[pending] : "Confirm"}
        confirmVariant={pending === "delete" ? "danger" : pending === "pause" ? "warning" : "primary"}
        onConfirm={execute}
        onCancel={() => setPending(null)}
      />
    </>
  );
}
