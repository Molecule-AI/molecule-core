"use client";

import { useState, useCallback } from "react";
import { api } from "@/lib/api";

export function BundleDropZone() {
  const [isDragging, setIsDragging] = useState(false);
  const [importing, setImporting] = useState(false);
  const [result, setResult] = useState<{ status: string; name?: string } | null>(null);

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.dataTransfer.types.includes("Files")) {
      setIsDragging(true);
    }
  }, []);

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(false);
  }, []);

  const handleDrop = useCallback(async (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(false);

    const file = Array.from(e.dataTransfer.files).find(
      (f) => f.name.endsWith(".bundle.json")
    );

    if (!file) {
      setResult({ status: "error", name: "Only .bundle.json files are accepted" });
      setTimeout(() => setResult(null), 3000);
      return;
    }

    setImporting(true);
    try {
      const text = await file.text();
      const bundle = JSON.parse(text);
      const res = await api.post<{ workspace_id: string; name: string; status: string }>(
        "/bundles/import",
        bundle
      );
      setResult({ status: "success", name: res.name || bundle.name });
      setTimeout(() => setResult(null), 4000);
    } catch (e) {
      setResult({
        status: "error",
        name: e instanceof Error ? e.message : "Import failed",
      });
      setTimeout(() => setResult(null), 4000);
    } finally {
      setImporting(false);
    }
  }, []);

  return (
    <>
      {/* Invisible drop zone covering the canvas */}
      <div
        className="fixed inset-0 z-10 pointer-events-none"
        style={{ pointerEvents: isDragging ? "auto" : "none" }}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
      />

      {/* Global drag listener to detect file entering the window */}
      <div
        className="fixed inset-0 z-[5]"
        onDragOver={handleDragOver}
        style={{ pointerEvents: "none" }}
      />

      {/* Visual overlay when dragging */}
      {isDragging && (
        <div className="fixed inset-0 z-20 flex items-center justify-center bg-blue-950/40 backdrop-blur-sm border-2 border-dashed border-blue-400/50 pointer-events-none">
          <div className="bg-zinc-900/95 border border-blue-500/50 rounded-2xl px-8 py-6 shadow-2xl text-center">
            <div className="text-3xl mb-2">📦</div>
            <div className="text-sm font-semibold text-zinc-100">Drop Bundle to Import</div>
            <div className="text-xs text-zinc-500 mt-1">.bundle.json files only</div>
          </div>
        </div>
      )}

      {/* Importing spinner */}
      {importing && (
        <div className="fixed bottom-6 left-1/2 -translate-x-1/2 z-50 bg-zinc-900/95 border border-zinc-700/60 rounded-xl px-5 py-3 shadow-2xl flex items-center gap-3">
          <div className="w-4 h-4 border-2 border-sky-400 border-t-transparent rounded-full animate-spin" />
          <span className="text-sm text-zinc-200">Importing bundle...</span>
        </div>
      )}

      {/* Result toast */}
      {result && (
        <div
          className={`fixed bottom-6 left-1/2 -translate-x-1/2 z-50 rounded-xl px-5 py-3 shadow-2xl text-sm ${
            result.status === "success"
              ? "bg-emerald-950/90 border border-emerald-700/50 text-emerald-200"
              : "bg-red-950/90 border border-red-700/50 text-red-200"
          }`}
        >
          {result.status === "success"
            ? `Imported "${result.name}" successfully`
            : result.name}
        </div>
      )}
    </>
  );
}
