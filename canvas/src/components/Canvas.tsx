"use client";

import { useCallback, useMemo } from "react";
import {
  ReactFlow,
  ReactFlowProvider,
  Background,
  Controls,
  MiniMap,
  type Edge,
  BackgroundVariant,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";

import { useCanvasStore } from "@/store/canvas";
import { A2ATopologyOverlay } from "./A2ATopologyOverlay";
import { WorkspaceNode } from "./WorkspaceNode";
import { SidePanel } from "./SidePanel";
import { CreateWorkspaceButton } from "./CreateWorkspaceDialog";
import { ContextMenu } from "./ContextMenu";
import { TemplatePalette } from "./TemplatePalette";
import { ApprovalBanner } from "./ApprovalBanner";
import { BundleDropZone } from "./BundleDropZone";
import { EmptyState } from "./EmptyState";
import { OnboardingWizard } from "./OnboardingWizard";
import { SearchDialog } from "./SearchDialog";
import { Toaster, showToast } from "./Toaster";
import { Toolbar } from "./Toolbar";
import { ConfirmDialog } from "./ConfirmDialog";
import { api } from "@/lib/api";
import { SettingsPanel, DeleteConfirmDialog } from "./settings";
import { BatchActionBar } from "./BatchActionBar";
import { ProvisioningTimeout } from "./ProvisioningTimeout";

import { DropTargetBadge } from "./canvas/DropTargetBadge";
import { useDragHandlers } from "./canvas/useDragHandlers";
import { useKeyboardShortcuts } from "./canvas/useKeyboardShortcuts";
import { useCanvasViewport } from "./canvas/useCanvasViewport";

const nodeTypes = {
  workspaceNode: WorkspaceNode,
};

const defaultEdgeOptions: Partial<Edge> = {
  animated: true,
  style: {
    stroke: "#3f3f46",
    strokeWidth: 1.5,
  },
};

export function Canvas() {
  return (
    <ReactFlowProvider>
      <CanvasInner />
    </ReactFlowProvider>
  );
}

function CanvasInner() {
  const nodes = useCanvasStore((s) => s.nodes);
  const edges = useCanvasStore((s) => s.edges);
  const a2aEdges = useCanvasStore((s) => s.a2aEdges);
  const showA2AEdges = useCanvasStore((s) => s.showA2AEdges);
  const allEdges = useMemo(
    () => (showA2AEdges ? [...edges, ...a2aEdges] : edges),
    [edges, a2aEdges, showA2AEdges],
  );
  const onNodesChange = useCanvasStore((s) => s.onNodesChange);
  const selectNode = useCanvasStore((s) => s.selectNode);
  const selectedNodeId = useCanvasStore((s) => s.selectedNodeId);

  // Drag / nest lifecycle — handlers, pending-nest state, confirm/cancel.
  const {
    onNodeDragStart,
    onNodeDrag,
    onNodeDragStop,
    pendingNest,
    confirmNest,
    cancelNest,
  } = useDragHandlers();

  // Window-level keyboard shortcuts (Esc, Enter, Shift+Enter, Cmd+]/[, Z).
  useKeyboardShortcuts();

  // Pan-to-node / zoom-to-team CustomEvent listeners + viewport save.
  const { onMoveEnd } = useCanvasViewport();

  // Delete-confirmation lives in the store so the dialog survives ContextMenu
  // unmounting — the prior local-in-ContextMenu state raced with the menu's
  // outside-click handler.
  const pendingDelete = useCanvasStore((s) => s.pendingDelete);
  const setPendingDelete = useCanvasStore((s) => s.setPendingDelete);
  const removeNode = useCanvasStore((s) => s.removeNode);
  const confirmDelete = useCallback(async () => {
    if (!pendingDelete) return;
    const { id } = pendingDelete;
    setPendingDelete(null);
    try {
      await api.del(`/workspaces/${id}?confirm=true`);
      removeNode(id);
    } catch (e) {
      showToast(e instanceof Error ? e.message : "Delete failed", "error");
    }
  }, [pendingDelete, setPendingDelete, removeNode]);

  const onPaneClick = useCallback(() => {
    selectNode(null);
    const state = useCanvasStore.getState();
    state.closeContextMenu();
    state.clearSelection();
  }, [selectNode]);

  const viewport = useCanvasStore((s) => s.viewport);
  const defaultViewport = useMemo(
    () => ({ x: viewport.x, y: viewport.y, zoom: viewport.zoom }),
    // Only use the initial viewport — don't re-render on every save
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [],
  );

  const settingsWorkspaceId = selectedNodeId ?? "global";

  return (
    <>
      <a
        href="#canvas-main"
        className="sr-only focus:not-sr-only focus:absolute focus:top-2 focus:left-2 focus:z-50 focus:px-4 focus:py-2 focus:bg-zinc-900 focus:text-zinc-100 focus:rounded-lg focus:border focus:border-zinc-700"
      >
        Skip to canvas
      </a>
      <main id="canvas-main" className="w-screen h-screen bg-zinc-950">
        <ReactFlow
          colorMode="dark"
          nodes={nodes}
          edges={allEdges}
          onNodesChange={onNodesChange}
          onNodeDragStart={onNodeDragStart}
          onNodeDrag={onNodeDrag}
          onNodeDragStop={onNodeDragStop}
          onPaneClick={onPaneClick}
          onMoveEnd={onMoveEnd}
          nodeTypes={nodeTypes}
          defaultEdgeOptions={defaultEdgeOptions}
          defaultViewport={defaultViewport}
          fitView={viewport.x === 0 && viewport.y === 0 && viewport.zoom === 1}
          minZoom={0.1}
          maxZoom={2}
          proOptions={{ hideAttribution: true }}
          aria-label="Molecule AI workspace canvas"
        >
          <Background
            variant={BackgroundVariant.Dots}
            gap={24}
            size={1}
            color="#27272a"
          />
          <Controls
            className="!bg-zinc-900/90 !border-zinc-700/50 !rounded-lg !shadow-xl !shadow-black/20 [&>button]:!bg-zinc-800 [&>button]:!border-zinc-700/50 [&>button]:!text-zinc-400 [&>button:hover]:!bg-zinc-700 [&>button:hover]:!text-zinc-200"
            showInteractive={false}
          />
          <MiniMap
            className="!bg-zinc-900/90 !border-zinc-700/50 !rounded-lg !shadow-xl !shadow-black/20"
            maskColor="rgba(0, 0, 0, 0.7)"
            nodeColor={(node) => {
              // Parents show as a filled region — hierarchy visible at
              // a glance in the minimap without needing to zoom.
              const hasChildren = nodes.some((n) => n.parentId === node.id);
              if (hasChildren) return "#3b82f6";
              const status = (node.data as Record<string, unknown>)?.status;
              switch (status) {
                case "online":
                  return "#34d399";
                case "offline":
                  return "#52525b";
                case "degraded":
                  return "#fbbf24";
                case "failed":
                  return "#f87171";
                case "provisioning":
                  return "#38bdf8";
                default:
                  return "#3f3f46";
              }
            }}
            nodeStrokeColor={(node) => {
              const hasChildren = nodes.some((n) => n.parentId === node.id);
              return hasChildren ? "#60a5fa" : "transparent";
            }}
            nodeStrokeWidth={2}
            nodeBorderRadius={4}
          />
          <DropTargetBadge />
        </ReactFlow>

        {/* Screen-reader live region: announces workspace count on canvas load or change */}
        <div role="status" aria-live="polite" className="sr-only">
          {nodes.filter((n) => !n.data.parentId).length === 0
            ? "No workspaces on canvas"
            : `${nodes.filter((n) => !n.data.parentId).length} workspace${nodes.filter((n) => !n.data.parentId).length !== 1 ? "s" : ""} on canvas`}
        </div>

        {nodes.length === 0 && <EmptyState />}
        <A2ATopologyOverlay />
        <OnboardingWizard />
        <Toolbar />
        <ApprovalBanner />
        <BundleDropZone />
        <TemplatePalette />
        <SidePanel />
        <ContextMenu />
        <SearchDialog />
        <Toaster />
        <ProvisioningTimeout />
        {!selectedNodeId && <CreateWorkspaceButton />}
        <BatchActionBar />

        <ConfirmDialog
          open={!!pendingNest}
          title={pendingNest?.targetId ? "Nest Workspace" : "Extract Workspace"}
          message={
            pendingNest?.targetId
              ? `Move "${pendingNest.nodeName}" inside "${pendingNest.targetName}"? This changes the org hierarchy — ${pendingNest.nodeName} will become a sub-workspace of ${pendingNest.targetName}.`
              : `Extract "${pendingNest?.nodeName}" from "${pendingNest?.targetName}"? This moves it to the root level.`
          }
          confirmLabel={pendingNest?.targetId ? "Nest" : "Extract"}
          onConfirm={confirmNest}
          onCancel={cancelNest}
        />

        <ConfirmDialog
          open={!!pendingDelete}
          title={pendingDelete?.hasChildren ? "Delete Workspace and Children" : "Delete Workspace"}
          message={pendingDelete?.hasChildren
            ? `⚠️ Deleting "${pendingDelete?.name}" will permanently delete all of its child workspaces and their data. This cannot be undone.`
            : `Permanently delete "${pendingDelete?.name}"? This will stop the container and remove all configuration. This action cannot be undone.`}
          confirmLabel={pendingDelete?.hasChildren ? "Delete All" : "Delete"}
          confirmVariant="danger"
          onConfirm={confirmDelete}
          onCancel={() => setPendingDelete(null)}
        />

        <SettingsPanel workspaceId={settingsWorkspaceId} />
        <DeleteConfirmDialog workspaceId={settingsWorkspaceId} />
      </main>
    </>
  );
}
