"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import { useCanvasStore } from "@/store/canvas";

export function SearchDialog() {
  const open = useCanvasStore((s) => s.searchOpen);
  const setOpen = useCanvasStore((s) => s.setSearchOpen);
  const [query, setQuery] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);
  const nodes = useCanvasStore((s) => s.nodes);
  const selectNode = useCanvasStore((s) => s.selectNode);
  const setPanelTab = useCanvasStore((s) => s.setPanelTab);

  // Cmd+K to open
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "k") {
        e.preventDefault();
        setOpen(true);
        setQuery("");
      }
      if (e.key === "Escape" && open) {
        setOpen(false);
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [open, setOpen]);

  useEffect(() => {
    if (open) {
      requestAnimationFrame(() => inputRef.current?.focus());
    }
  }, [open]);

  const filtered = nodes.filter((n) => {
    if (!query) return true;
    const q = query.toLowerCase();
    return (
      n.data.name.toLowerCase().includes(q) ||
      (n.data.role || "").toLowerCase().includes(q) ||
      n.data.status.toLowerCase().includes(q)
    );
  });

  const handleSelect = useCallback(
    (nodeId: string) => {
      selectNode(nodeId);
      setPanelTab("details");
      setOpen(false);
    },
    [selectNode, setPanelTab]
  );

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-[70] flex items-start justify-center pt-[20vh] bg-black/50 backdrop-blur-sm" onClick={() => setOpen(false)}>
      <div
        className="w-[420px] bg-zinc-950/95 backdrop-blur-xl border border-zinc-800/60 rounded-2xl shadow-2xl shadow-black/50 overflow-hidden"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Search input */}
        <div className="flex items-center gap-3 px-4 py-3 border-b border-zinc-800/40">
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none" className="shrink-0 text-zinc-500">
            <circle cx="7" cy="7" r="5.5" stroke="currentColor" strokeWidth="1.5" />
            <path d="M11 11l3.5 3.5" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
          </svg>
          <input
            ref={inputRef}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search workspaces..."
            className="flex-1 bg-transparent text-sm text-zinc-100 placeholder-zinc-600 focus:outline-none"
          />
          <kbd className="text-[9px] text-zinc-600 bg-zinc-800/60 px-1.5 py-0.5 rounded border border-zinc-700/40">ESC</kbd>
        </div>

        {/* Results */}
        <div className="max-h-[300px] overflow-y-auto py-1">
          {filtered.length === 0 ? (
            <div className="px-4 py-6 text-center text-xs text-zinc-600">
              {query ? "No workspaces match" : "No workspaces yet"}
            </div>
          ) : (
            filtered.map((node) => (
              <button
                key={node.id}
                onClick={() => handleSelect(node.id)}
                className="w-full px-4 py-2.5 flex items-center gap-3 text-left hover:bg-zinc-800/40 transition-colors"
              >
                <div className={`w-2 h-2 rounded-full shrink-0 ${
                  node.data.status === "online" ? "bg-emerald-400" :
                  node.data.status === "failed" ? "bg-red-400" :
                  node.data.status === "provisioning" ? "bg-sky-400 animate-pulse" :
                  "bg-zinc-500"
                }`} />
                <div className="min-w-0 flex-1">
                  <div className="text-sm text-zinc-200 truncate">{node.data.name}</div>
                  {node.data.role && (
                    <div className="text-[10px] text-zinc-500 truncate">{node.data.role}</div>
                  )}
                </div>
                <span className="text-[9px] font-mono text-zinc-600">T{node.data.tier}</span>
              </button>
            ))
          )}
        </div>

        {/* Footer */}
        <div className="px-4 py-2 border-t border-zinc-800/40 flex items-center justify-between">
          <span className="text-[9px] text-zinc-600">{filtered.length} workspace{filtered.length !== 1 ? "s" : ""}</span>
          <div className="flex gap-2">
            <kbd className="text-[9px] text-zinc-600 bg-zinc-800/60 px-1.5 py-0.5 rounded border border-zinc-700/40">↵ select</kbd>
          </div>
        </div>
      </div>
    </div>
  );
}
