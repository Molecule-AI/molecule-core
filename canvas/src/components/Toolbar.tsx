"use client";

import { useMemo, useState, useCallback, useEffect, useRef } from "react";
import { api } from "@/lib/api";
import { useCanvasStore } from "@/store/canvas";
import { SettingsButton } from "@/components/settings/SettingsButton";
import { settingsGearRef } from "@/components/settings/SettingsPanel";
import { ConfirmDialog } from "@/components/ConfirmDialog";
import { showToast } from "@/components/Toaster";
import { statusDotClass } from "@/lib/design-tokens";

export function Toolbar() {
  const nodes = useCanvasStore((s) => s.nodes);
  const wsStatus = useCanvasStore((s) => s.wsStatus);
  const showA2AEdges = useCanvasStore((s) => s.showA2AEdges);
  const setShowA2AEdges = useCanvasStore((s) => s.setShowA2AEdges);
  const selectedNodeId = useCanvasStore((s) => s.selectedNodeId);
  const setPanelTab = useCanvasStore((s) => s.setPanelTab);

  const [stopping, setStopping] = useState(false);
  const [restartingAll, setRestartingAll] = useState(false);
  const [restartConfirmOpen, setRestartConfirmOpen] = useState(false);
  const [helpOpen, setHelpOpen] = useState(false);
  const helpRef = useRef<HTMLDivElement>(null);

  // Suppress toast on the very first connect at page load; only fire on reconnects.
  const mountedRef = useRef(false);
  useEffect(() => {
    const t = setTimeout(() => { mountedRef.current = true; }, 2000);
    return () => clearTimeout(t);
  }, []);

  const prevWsStatus = useRef<string>("connecting");
  useEffect(() => {
    if (prevWsStatus.current === "connecting" && wsStatus === "connected") {
      if (mountedRef.current) {
        showToast("Live updates restored", "success");
      }
    }
    prevWsStatus.current = wsStatus;
  }, [wsStatus]);

  const counts = useMemo(() => {
    const c = { total: nodes.length, roots: 0, children: 0, online: 0, offline: 0, failed: 0, provisioning: 0, activeTasks: 0 };
    for (const n of nodes) {
      if (n.data.parentId) c.children++; else c.roots++;
      const s = n.data.status;
      if (s === "online") c.online++;
      else if (s === "offline") c.offline++;
      else if (s === "failed") c.failed++;
      else if (s === "provisioning") c.provisioning++;
      if ((n.data.activeTasks as number) > 0) c.activeTasks++;
    }
    return c;
  }, [nodes]);

  const stopAll = useCallback(async () => {
    setStopping(true);
    const active = nodes.filter((n) => (n.data.activeTasks as number) > 0);
    await Promise.all(
      active.map((n) =>
        api.post(`/workspaces/${n.id}/restart`).catch(() => {})
      )
    );
    setStopping(false);
  }, [nodes]);

  // Workspaces flagged as needing restart (config edited, global secret changed, etc.)
  const needsRestartNodes = useMemo(
    () => nodes.filter((n) => n.data.needsRestart),
    [nodes]
  );

  const restartAll = useCallback(async () => {
    setRestartConfirmOpen(false);
    setRestartingAll(true);
    const targets = needsRestartNodes;
    const results = await Promise.allSettled(
      targets.map((n) => api.post(`/workspaces/${n.id}/restart`))
    );
    const failed = results.filter((r) => r.status === "rejected").length;
    setRestartingAll(false);
    // Clear needsRestart on successfully-restarted workspaces
    const store = useCanvasStore.getState();
    targets.forEach((n, i) => {
      if (results[i].status === "fulfilled") {
        store.updateNodeData(n.id, { needsRestart: false });
      }
    });
    if (failed === 0) {
      showToast(`Restarted ${targets.length} workspace${targets.length === 1 ? "" : "s"}`, "success");
    } else if (failed === targets.length) {
      showToast(`Failed to restart any workspaces`, "error");
    } else {
      showToast(`Restarted ${targets.length - failed} of ${targets.length} (${failed} failed)`, "error");
    }
  }, [needsRestartNodes]);

  useEffect(() => {
    const onPointerDown = (event: MouseEvent) => {
      if (helpRef.current && !helpRef.current.contains(event.target as Node)) {
        setHelpOpen(false);
      }
    };
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setHelpOpen(false);
      }
    };
    window.addEventListener("pointerdown", onPointerDown);
    window.addEventListener("keydown", onKeyDown);
    return () => {
      window.removeEventListener("pointerdown", onPointerDown);
      window.removeEventListener("keydown", onKeyDown);
    };
  }, []);

  return (
    <div className="fixed top-3 left-1/2 -translate-x-1/2 z-20 flex items-center gap-3 sm:gap-4 glass-panel rounded-2xl px-3 sm:px-5 py-2.5 shadow-premium max-w-[calc(100vw-1rem)] overflow-x-auto border-b border-b-transparent" style={{ borderImage: "linear-gradient(90deg, transparent, rgba(57,229,140,0.2), rgba(34,209,238,0.2), transparent) 1" }}>
      {/* Logo / Title */}
      <div className="flex items-center gap-2.5 pr-3 sm:pr-4 border-r border-white/[0.06] shrink-0">
        <img src="/molecule-icon.png" alt="Molecule AI" className="w-5 h-5" />
        <span className="text-[12px] font-semibold text-gradient-mint-cyan tracking-wide hidden sm:inline">Molecule AI</span>
      </div>

      {/* Status counts */}
      <div className="flex items-center gap-3">
        <StatusPill color={statusDotClass("online")} count={counts.online} label="online" />
        {counts.offline > 0 && (
          <StatusPill color={statusDotClass("offline")} count={counts.offline} label="offline" />
        )}
        {counts.provisioning > 0 && (
          <StatusPill color={statusDotClass("provisioning")} count={counts.provisioning} label="starting" />
        )}
        {counts.failed > 0 && (
          <StatusPill color={statusDotClass("failed")} count={counts.failed} label="failed" />
        )}
      </div>

      {/* Total */}
      <div className="pl-3 sm:pl-4 border-l border-white/[0.06] hidden sm:block">
        <span className="text-[11px] text-slate-400">
          {counts.roots} workspace{counts.roots !== 1 ? "s" : ""}
          {counts.children > 0 && <span className="text-slate-500"> + {counts.children} sub</span>}
        </span>
      </div>

      {/* WebSocket connection status */}
      <div className="pl-3 sm:pl-4 border-l border-white/[0.06] shrink-0">
        <WsStatusPill status={wsStatus} />
      </div>

      {/* Stop All — visible when agents have active tasks */}
      {counts.activeTasks > 0 && (
        <button
          onClick={stopAll}
          disabled={stopping}
          className="flex items-center gap-1.5 px-2.5 py-1 bg-red-950/50 hover:bg-red-900/60 border border-red-800/40 rounded-lg transition-colors disabled:opacity-50"
          title={`Stop all running tasks (${counts.activeTasks} active)`}
          aria-label={stopping ? "Stopping all running tasks" : `Stop all running tasks (${counts.activeTasks} active)`}
        >
          <svg width="10" height="10" viewBox="0 0 16 16" fill="currentColor" className="text-red-400">
            <rect x="2" y="2" width="12" height="12" rx="2" />
          </svg>
          <span className="text-[10px] text-red-300 font-medium">
            {stopping ? "Stopping..." : `Stop All (${counts.activeTasks})`}
          </span>
        </button>
      )}

      {/* Restart All — only shows when workspaces are flagged as needsRestart */}
      {needsRestartNodes.length > 0 && (
        <button
          onClick={() => setRestartConfirmOpen(true)}
          disabled={restartingAll}
          className="flex items-center gap-1.5 px-2.5 py-1 bg-amber-950/40 hover:bg-amber-900/50 border border-amber-800/40 rounded-lg transition-colors disabled:opacity-50"
          title={`Restart ${needsRestartNodes.length} workspace${needsRestartNodes.length === 1 ? "" : "s"} that need to pick up config or secret changes`}
          aria-label={restartingAll ? "Restarting workspaces" : `Restart ${needsRestartNodes.length} workspace${needsRestartNodes.length === 1 ? "" : "s"} pending config or secret changes`}
        >
          <svg width="10" height="10" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.8" className="text-amber-400">
            <path d="M2 8a6 6 0 1 1 1.76 4.24M2 13v-3h3" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
          <span className="text-[10px] text-amber-300 font-medium">
            {restartingAll ? "Restarting..." : `Restart Pending (${needsRestartNodes.length})`}
          </span>
        </button>
      )}

      {/* A2A topology overlay toggle */}
      <button
        onClick={() => setShowA2AEdges(!showA2AEdges)}
        aria-pressed={showA2AEdges}
        aria-label={showA2AEdges ? "Hide A2A edges" : "Show A2A edges"}
        title={showA2AEdges ? "Hide A2A delegation edges" : "Show A2A delegation edges (last 60 min)"}
        className={`flex items-center gap-1.5 px-3 py-1.5 border rounded-lg transition-all hover:scale-[1.02] ${
          showA2AEdges
            ? "bg-molecule-accent-mint/10 hover:bg-molecule-accent-mint/15 border-molecule-accent-mint/30 text-molecule-accent-mint glow-mint"
            : "bg-molecule-surface-800/60 hover:bg-molecule-surface-700/60 border-white/[0.06] text-slate-400 hover:text-slate-200"
        }`}
      >
        {/* Mesh / network icon */}
        <svg
          width="12"
          height="12"
          viewBox="0 0 16 16"
          fill="none"
          className="shrink-0"
          aria-hidden="true"
        >
          <circle cx="3" cy="3" r="2" stroke="currentColor" strokeWidth="1.4" />
          <circle cx="13" cy="3" r="2" stroke="currentColor" strokeWidth="1.4" />
          <circle cx="8" cy="13" r="2" stroke="currentColor" strokeWidth="1.4" />
          <path
            d="M5 3h6M3.7 5l3.3 6M12.3 5l-3.3 6"
            stroke="currentColor"
            strokeWidth="1.3"
            strokeLinecap="round"
          />
        </svg>
        <span className="text-[11px] font-medium hidden sm:inline">A2A</span>
      </button>

      {/* Audit trail shortcut — switches selected workspace's panel to the Audit tab */}
      <button
        onClick={() => {
          if (selectedNodeId) {
            setPanelTab("audit");
          } else {
            showToast("Select a workspace to view its audit trail", "info");
          }
        }}
        aria-label="Open audit trail for selected workspace"
        title="View audit ledger for the selected workspace"
        className="flex items-center gap-1.5 px-3 py-1.5 bg-molecule-surface-800/60 hover:bg-molecule-surface-700/60 border border-white/[0.06] rounded-lg transition-all hover:scale-[1.02] text-slate-400 hover:text-slate-200"
      >
        {/* Scroll / ledger icon */}
        <svg
          width="12"
          height="12"
          viewBox="0 0 16 16"
          fill="none"
          className="shrink-0"
          aria-hidden="true"
        >
          <rect x="3" y="2" width="10" height="12" rx="1.5" stroke="currentColor" strokeWidth="1.4" />
          <path d="M6 5.5h4M6 8h4M6 10.5h2.5" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" />
        </svg>
        <span className="text-[11px] font-medium hidden sm:inline">Audit</span>
      </button>

      {/* Search shortcut */}
      <button
        onClick={() => useCanvasStore.getState().setSearchOpen(true)}
        className="flex items-center gap-1.5 px-3 py-1.5 bg-molecule-surface-800/60 hover:bg-molecule-surface-700/60 border border-white/[0.06] rounded-lg transition-all hover:scale-[1.02] shrink-0"
      >
        <svg width="12" height="12" viewBox="0 0 16 16" fill="none" className="text-slate-400">
          <circle cx="7" cy="7" r="5" stroke="currentColor" strokeWidth="1.5" />
          <path d="M11 11l3 3" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
        </svg>
        <span className="text-[11px] text-slate-400 hidden sm:inline">Search</span>
        <kbd className="text-[9px] text-slate-500 bg-molecule-bg-900/60 px-1.5 py-0.5 rounded border border-white/[0.06] hidden sm:inline">&#8984;K</kbd>
      </button>

      {/* Quick help */}
      <div ref={helpRef} className="relative">
        <button
          onClick={() => setHelpOpen((open) => !open)}
          className="flex items-center gap-1.5 px-3 py-1.5 bg-molecule-surface-800/60 hover:bg-molecule-surface-700/60 border border-white/[0.06] rounded-lg transition-all hover:scale-[1.02]"
          aria-expanded={helpOpen}
          aria-label="Open quick help"
        >
          <svg width="12" height="12" viewBox="0 0 16 16" fill="none" className="text-slate-400">
            <path d="M8 12v.5M6.5 6.3A1.9 1.9 0 1 1 9 8.1c-.7.4-1 .8-1 1.7" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
            <circle cx="8" cy="8" r="6" stroke="currentColor" strokeWidth="1.2" />
          </svg>
          <span className="text-[11px] text-slate-400 hidden sm:inline">Help</span>
        </button>

        {helpOpen && (
          <div className="absolute right-0 top-full mt-2 w-72 max-w-[calc(100vw-2rem)] rounded-xl glass-panel p-4 shadow-premium-lg">
            <div className="mb-2 flex items-center justify-between">
              <span className="text-[11px] font-semibold uppercase tracking-[0.24em] text-gradient-mint-cyan">Quick start</span>
              <button
                onClick={() => setHelpOpen(false)}
                className="text-[11px] text-slate-500 hover:text-slate-200 transition-colors"
              >
                Close
              </button>
            </div>
            <div className="space-y-2">
              <HelpRow shortcut="⌘K" text="Search workspaces and jump straight into Details or Chat." />
              <HelpRow shortcut="Palette" text="Open the template palette to deploy a new workspace." />
              <HelpRow shortcut="Right-click" text="Use node actions for expand, duplicate, export, restart, or delete." />
              <HelpRow shortcut="Chat" text="If a task is still running, the chat tab resumes that session automatically." />
              <HelpRow shortcut="Config" text="Use the Config tab for skills, model, secrets, and runtime settings." />
              <HelpRow shortcut="Dbl-click / Z" text="Zoom canvas to fit a team node and all its sub-workspaces." />
            </div>
          </div>
        )}
      </div>

      {/* Settings gear icon */}
      <SettingsButton ref={settingsGearRef} />

      <ConfirmDialog
        open={restartConfirmOpen}
        title={`Restart ${needsRestartNodes.length} workspace${needsRestartNodes.length === 1 ? "" : "s"}?`}
        message="These workspaces have pending config or secret changes that need a restart to take effect."
        confirmLabel="Restart"
        confirmVariant="warning"
        onConfirm={restartAll}
        onCancel={() => setRestartConfirmOpen(false)}
      />
    </div>
  );
}

function StatusPill({ color, count, label }: { color: string; count: number; label: string }) {
  return (
    <div className="flex items-center gap-1.5" title={`${count} ${label}`} aria-label={`${count} ${label}`}>
      <div className={`w-2 h-2 rounded-full ${color}`} style={{ boxShadow: label === "online" ? "0 0 8px rgba(57, 229, 140, 0.4)" : undefined }} aria-hidden="true" />
      <span className="text-[11px] text-slate-300 tabular-nums font-medium" aria-hidden="true">{count}</span>
    </div>
  );
}

function WsStatusPill({ status }: { status: "connected" | "connecting" | "disconnected" }) {
  if (status === "connected") {
    return (
      <div className="flex items-center gap-1.5" title="Real-time updates: connected" aria-label="Real-time updates: connected">
        <div className={`w-2 h-2 rounded-full ${statusDotClass("online")}`} style={{ boxShadow: "0 0 8px rgba(57, 229, 140, 0.4)" }} aria-hidden="true" />
        <span className="text-[11px] text-molecule-accent-mint/80 font-medium" aria-hidden="true">Live</span>
      </div>
    );
  }
  if (status === "connecting") {
    return (
      <div className="flex items-center gap-1.5" title="Real-time updates: reconnecting…" aria-label="Real-time updates: reconnecting">
        <div className="w-2 h-2 rounded-full bg-amber-400 motion-safe:animate-pulse" aria-hidden="true" />
        <span className="text-[11px] text-slate-400" aria-hidden="true">Reconnecting</span>
      </div>
    );
  }
  return (
    <div className="flex items-center gap-1.5" title="Real-time updates: disconnected" aria-label="Real-time updates: disconnected">
      <div className={`w-2 h-2 rounded-full ${statusDotClass("failed")}`} aria-hidden="true" />
      <span className="text-[11px] text-slate-400" aria-hidden="true">Offline</span>
    </div>
  );
}

function HelpRow({ shortcut, text }: { shortcut: string; text: string }) {
  return (
    <div className="flex items-start gap-3 rounded-lg border border-white/[0.05] bg-molecule-surface-900/60 px-3 py-2.5">
      <span className="shrink-0 rounded-md border border-molecule-accent-mint/20 bg-molecule-accent-mint/5 px-2 py-0.5 text-[10px] font-medium uppercase tracking-[0.18em] text-molecule-accent-mint/70">
        {shortcut}
      </span>
      <p className="text-[12px] leading-relaxed text-slate-400">{text}</p>
    </div>
  );
}
