"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useCanvasStore, type WorkspaceNodeData } from "@/store/canvas";
import { api } from "@/lib/api";
import { showToast } from "./Toaster";
import { ConfirmDialog } from "./ConfirmDialog";

interface MenuItem {
  label: string;
  icon: string;
  action: () => void;
  danger?: boolean;
  disabled?: boolean;
  divider?: boolean;
}

export function ContextMenu() {
  const contextMenu = useCanvasStore((s) => s.contextMenu);
  const closeContextMenu = useCanvasStore((s) => s.closeContextMenu);
  const removeNode = useCanvasStore((s) => s.removeNode);
  const updateNodeData = useCanvasStore((s) => s.updateNodeData);
  const selectNode = useCanvasStore((s) => s.selectNode);
  const setPanelTab = useCanvasStore((s) => s.setPanelTab);
  const nestNode = useCanvasStore((s) => s.nestNode);
  const contextNodeId = contextMenu?.nodeId ?? null;
  const hasChildren = useCanvasStore((s) => contextNodeId ? s.nodes.some((n) => n.data.parentId === contextNodeId) : false);
  const ref = useRef<HTMLDivElement>(null);
  const [deleteConfirm, setDeleteConfirm] = useState<{ id: string; name: string } | null>(null);
  const [actionLoading, setActionLoading] = useState(false);

  // Clear orphaned dialog state when context menu closes
  useEffect(() => {
    if (!contextMenu) setDeleteConfirm(null);
  }, [contextMenu]);

  // Close on click outside or Escape
  useEffect(() => {
    if (!contextMenu) return;
    const handleClick = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as HTMLElement)) {
        closeContextMenu();
      }
    };
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") closeContextMenu();
    };
    document.addEventListener("mousedown", handleClick);
    document.addEventListener("keydown", handleKey);
    return () => {
      document.removeEventListener("mousedown", handleClick);
      document.removeEventListener("keydown", handleKey);
    };
  }, [contextMenu, closeContextMenu]);

  const handleExportBundle = useCallback(async () => {
    if (!contextMenu || actionLoading) return;
    setActionLoading(true);
    try {
      const bundle = await api.get<Record<string, unknown>>(`/bundles/export/${contextMenu.nodeId}`);
      const blob = new Blob([JSON.stringify(bundle, null, 2)], { type: "application/json" });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `${(contextMenu.nodeData.name || "workspace").toLowerCase().replace(/\s+/g, "-")}.bundle.json`;
      a.click();
      URL.revokeObjectURL(url);
      showToast("Bundle exported", "success");
    } catch (e) {
      showToast("Export failed", "error");
    } finally {
      setActionLoading(false);
    }
    closeContextMenu();
  }, [contextMenu, closeContextMenu, actionLoading]);

  const handleDuplicate = useCallback(async () => {
    if (!contextMenu || actionLoading) return;
    setActionLoading(true);
    try {
      const bundle = await api.get<Record<string, unknown>>(`/bundles/export/${contextMenu.nodeId}`);
      await api.post("/bundles/import", bundle);
    } catch (e) {
      showToast("Duplicate failed", "error");
    } finally {
      setActionLoading(false);
    }
    closeContextMenu();
  }, [contextMenu, closeContextMenu, actionLoading]);

  const handleRestart = useCallback(async () => {
    if (!contextMenu) return;
    try {
      await api.post(`/workspaces/${contextMenu.nodeId}/restart`, {});
      updateNodeData(contextMenu.nodeId, { status: "provisioning" });
    } catch (e) {
      showToast("Restart failed", "error");
    }
    closeContextMenu();
  }, [contextMenu, updateNodeData, closeContextMenu]);

  const handlePause = useCallback(async () => {
    if (!contextMenu) return;
    const nodeId = contextMenu.nodeId;
    closeContextMenu();
    try {
      await api.post(`/workspaces/${nodeId}/pause`, {});
      updateNodeData(nodeId, { status: "paused" });
    } catch (e) {
      showToast("Pause failed", "error");
    }
  }, [contextMenu, updateNodeData, closeContextMenu]);

  const handleResume = useCallback(async () => {
    if (!contextMenu) return;
    const nodeId = contextMenu.nodeId;
    closeContextMenu();
    try {
      await api.post(`/workspaces/${nodeId}/resume`, {});
      updateNodeData(nodeId, { status: "provisioning" });
    } catch (e) {
      showToast("Resume failed", "error");
    }
  }, [contextMenu, updateNodeData, closeContextMenu]);

  const handleDelete = useCallback(() => {
    if (!contextMenu) return;
    // Don't close context menu yet — keep it mounted so ConfirmDialog renders
    setDeleteConfirm({ id: contextMenu.nodeId, name: contextMenu.nodeData.name });
  }, [contextMenu]);

  const confirmDelete = useCallback(async () => {
    if (!deleteConfirm) return;
    try {
      await api.del(`/workspaces/${deleteConfirm.id}`);
      removeNode(deleteConfirm.id);
    } catch {
      showToast("Delete failed", "error");
    }
    setDeleteConfirm(null);
    closeContextMenu();
  }, [deleteConfirm, removeNode, closeContextMenu]);

  const handleViewDetails = useCallback(() => {
    if (!contextMenu) return;
    selectNode(contextMenu.nodeId);
    setPanelTab("details");
    closeContextMenu();
  }, [contextMenu, selectNode, setPanelTab, closeContextMenu]);

  const handleOpenChat = useCallback(() => {
    if (!contextMenu) return;
    selectNode(contextMenu.nodeId);
    setPanelTab("chat");
    closeContextMenu();
  }, [contextMenu, selectNode, setPanelTab, closeContextMenu]);

  const handleOpenTerminal = useCallback(() => {
    if (!contextMenu) return;
    selectNode(contextMenu.nodeId);
    setPanelTab("terminal");
    closeContextMenu();
  }, [contextMenu, selectNode, setPanelTab, closeContextMenu]);

  const handleExpand = useCallback(async () => {
    if (!contextMenu) return;
    try {
      await api.post(`/workspaces/${contextMenu.nodeId}/expand`, {});
    } catch (e) {
      showToast("Expand failed", "error");
    }
    closeContextMenu();
  }, [contextMenu, closeContextMenu]);

  const handleCollapse = useCallback(async () => {
    if (!contextMenu) return;
    try {
      await api.post(`/workspaces/${contextMenu.nodeId}/collapse`, {});
    } catch (e) {
      showToast("Collapse failed", "error");
    }
    closeContextMenu();
  }, [contextMenu, closeContextMenu]);

  const handleRemoveFromTeam = useCallback(async () => {
    if (!contextMenu) return;
    try {
      await nestNode(contextMenu.nodeId, null);
      showToast("Extracted from team", "success");
    } catch {
      showToast("Extract failed", "error");
    }
    closeContextMenu();
  }, [contextMenu, nestNode, closeContextMenu]);

  if (!contextMenu) return null;

  const isOfflineOrFailed = contextMenu.nodeData.status === "offline" || contextMenu.nodeData.status === "failed";
  const isOnline = contextMenu.nodeData.status === "online";
  const isPaused = contextMenu.nodeData.status === "paused";
  const isChild = !!contextMenu.nodeData.parentId;

  const items: MenuItem[] = [
    { label: "Details", icon: "i", action: handleViewDetails },
    { label: "Chat", icon: "💬", action: handleOpenChat, disabled: !isOnline },
    { label: "Terminal", icon: ">_", action: handleOpenTerminal, disabled: !isOnline },
    { label: "", icon: "", action: () => {}, divider: true },
    { label: "Export Bundle", icon: "📦", action: handleExportBundle },
    { label: "Duplicate", icon: "⧉", action: handleDuplicate },
    ...(isChild
      ? [{ label: "Extract from Team", icon: "⤴", action: handleRemoveFromTeam }]
      : []),
    ...(hasChildren
      ? [{ label: "Collapse Team", icon: "◁", action: handleCollapse }]
      : [{ label: "Expand to Team", icon: "▷", action: handleExpand }]),
    { label: "", icon: "", action: () => {}, divider: true },
    ...(isPaused
      ? [{ label: "Resume", icon: "▶", action: handleResume }]
      : [{ label: "Pause", icon: "⏸", action: handlePause, disabled: !isOnline }]),
    { label: "Restart", icon: "↻", action: handleRestart, disabled: !(isOfflineOrFailed || isPaused) },
    { label: "Delete", icon: "✕", action: handleDelete, danger: true },
  ];

  return (
    <div
      ref={ref}
      className="fixed z-[60] min-w-[200px] bg-zinc-950/95 backdrop-blur-xl border border-zinc-800/60 rounded-xl shadow-2xl shadow-black/60 py-1 overflow-hidden"
      style={{ left: contextMenu.x, top: contextMenu.y }}
    >
      {/* Header */}
      <div className="px-3.5 py-2 border-b border-zinc-800/40 mb-0.5">
        <div className="text-[11px] font-semibold text-zinc-200 truncate">{contextMenu.nodeData.name}</div>
        <div className="flex items-center gap-1.5 mt-0.5">
          <div className={`w-1.5 h-1.5 rounded-full ${
            isOnline ? "bg-emerald-400" : isOfflineOrFailed ? "bg-red-400" : "bg-zinc-500"
          }`} />
          <span className="text-[9px] text-zinc-500">{contextMenu.nodeData.status}</span>
        </div>
      </div>

      {items.map((item, i) => {
        if (item.divider) {
          return <div key={i} className="h-px bg-zinc-800/60 my-1" />;
        }
        return (
          <button
            key={i}
            onClick={item.action}
            disabled={item.disabled}
            className={`w-full px-3.5 py-1.5 flex items-center gap-2.5 text-left text-[11px] transition-colors disabled:opacity-25 disabled:cursor-not-allowed ${
              item.danger
                ? "text-red-400 hover:bg-red-950/40 hover:text-red-300"
                : "text-zinc-300 hover:bg-zinc-800/40 hover:text-zinc-100"
            }`}
          >
            <span className="w-4 text-center text-[10px] shrink-0 opacity-50">{item.icon}</span>
            {item.label}
          </button>
        );
      })}

      {/* Delete confirmation dialog */}
      <ConfirmDialog
        open={!!deleteConfirm}
        title="Delete Workspace"
        message={`Permanently delete "${deleteConfirm?.name}"? This will stop the container and remove all configuration. This action cannot be undone.`}
        confirmLabel="Delete"
        confirmVariant="danger"
        onConfirm={confirmDelete}
        onCancel={() => { setDeleteConfirm(null); closeContextMenu(); }}
      />
    </div>
  );
}
