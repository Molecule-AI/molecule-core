"use client";

import { useCallback, useRef, useMemo, useEffect, useState } from "react";
import {
  ReactFlow,
  ReactFlowProvider,
  Background,
  Controls,
  MiniMap,
  useReactFlow,
  type OnNodeDrag,
  type Node,
  type Edge,
  BackgroundVariant,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";

import { useCanvasStore, type WorkspaceNodeData } from "@/store/canvas";
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
import { Toaster } from "./Toaster";
import { Toolbar } from "./Toolbar";
import { ConfirmDialog } from "./ConfirmDialog";
// Phase 20 components
import { SettingsPanel, DeleteConfirmDialog } from "./settings";
// import { ProvisioningTimeout } from "./ProvisioningTimeout";

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
  const onNodesChange = useCanvasStore((s) => s.onNodesChange);
  const savePosition = useCanvasStore((s) => s.savePosition);
  const selectNode = useCanvasStore((s) => s.selectNode);
  const selectedNodeId = useCanvasStore((s) => s.selectedNodeId);
  const setDragOverNode = useCanvasStore((s) => s.setDragOverNode);
  const nestNode = useCanvasStore((s) => s.nestNode);
  const isDescendant = useCanvasStore((s) => s.isDescendant);
  const dragStartParentRef = useRef<string | null>(null);
  const { getIntersectingNodes } = useReactFlow();

  const onNodeDragStart: OnNodeDrag<Node<WorkspaceNodeData>> = useCallback(
    (_event, node) => {
      dragStartParentRef.current = (node.data as WorkspaceNodeData).parentId;
    },
    []
  );

  const onNodeDrag: OnNodeDrag<Node<WorkspaceNodeData>> = useCallback(
    (_event, node) => {
      const intersecting = getIntersectingNodes(node);
      const target = intersecting.find(
        (n) => n.id !== node.id && !isDescendant(node.id, n.id)
      );
      setDragOverNode(target?.id ?? null);
    },
    [getIntersectingNodes, isDescendant, setDragOverNode]
  );

  // Confirmation dialog state for structure changes
  const [pendingNest, setPendingNest] = useState<{ nodeId: string; targetId: string | null; nodeName: string; targetName: string } | null>(null);

  const onNodeDragStop: OnNodeDrag<Node<WorkspaceNodeData>> = useCallback(
    (_event, node) => {
      const { dragOverNodeId, nodes: allNodes } = useCanvasStore.getState();
      setDragOverNode(null);

      const nodeName = (node.data as WorkspaceNodeData).name;

      if (dragOverNodeId) {
        const targetNode = allNodes.find((n) => n.id === dragOverNodeId);
        const targetName = targetNode?.data.name || "Unknown";
        setPendingNest({ nodeId: node.id, targetId: dragOverNodeId, nodeName, targetName });
      } else {
        const currentParentId = (node.data as WorkspaceNodeData).parentId;
        if (currentParentId) {
          const parentNode = allNodes.find((n) => n.id === currentParentId);
          const parentName = parentNode?.data.name || "Unknown";
          setPendingNest({ nodeId: node.id, targetId: null, nodeName, targetName: parentName });
        }
      }

      savePosition(node.id, node.position.x, node.position.y);
    },
    [savePosition, setDragOverNode]
  );

  const confirmNest = useCallback(() => {
    if (pendingNest) {
      nestNode(pendingNest.nodeId, pendingNest.targetId);
      setPendingNest(null);
    }
  }, [pendingNest, nestNode]);

  const cancelNest = useCallback(() => {
    setPendingNest(null);
  }, []);

  const onPaneClick = useCallback(() => {
    selectNode(null);
    useCanvasStore.getState().closeContextMenu();
  }, [selectNode]);

  // Team zoom-in: double-click a team node to zoom to its children
  const { fitBounds, setCenter } = useReactFlow();

  // Pan to newly deployed workspace
  const panTimerRef = useRef<ReturnType<typeof setTimeout>>(undefined);
  useEffect(() => {
    const handler = (e: Event) => {
      const { nodeId } = (e as CustomEvent<{ nodeId: string }>).detail;
      // Small delay so ReactFlow has time to lay out the node
      clearTimeout(panTimerRef.current);
      panTimerRef.current = setTimeout(() => {
        const node = useCanvasStore.getState().nodes.find((n) => n.id === nodeId);
        if (node) {
          setCenter(node.position.x + 130, node.position.y + 60, { zoom: 1, duration: 500 });
        }
      }, 100);
    };
    window.addEventListener("molecule:pan-to-node", handler);
    return () => {
      window.removeEventListener("molecule:pan-to-node", handler);
      clearTimeout(panTimerRef.current);
    };
  }, [setCenter]);
  useEffect(() => {
    const handler = (e: Event) => {
      const { nodeId } = (e as CustomEvent).detail;
      const state = useCanvasStore.getState();
      const children = state.nodes.filter((n) => n.data.parentId === nodeId);
      if (children.length === 0) return;

      const parent = state.nodes.find((n) => n.id === nodeId);
      const allNodes = parent ? [parent, ...children] : children;

      let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity;
      for (const n of allNodes) {
        minX = Math.min(minX, n.position.x);
        minY = Math.min(minY, n.position.y);
        maxX = Math.max(maxX, n.position.x + 260);
        maxY = Math.max(maxY, n.position.y + 120);
      }

      fitBounds(
        { x: minX - 50, y: minY - 50, width: maxX - minX + 100, height: maxY - minY + 100 },
        { padding: 0.2, duration: 500 }
      );
    };
    window.addEventListener("molecule:zoom-to-team", handler);
    return () => window.removeEventListener("molecule:zoom-to-team", handler);
  }, [fitBounds]);

  // Keyboard shortcuts
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        const state = useCanvasStore.getState();
        if (state.contextMenu) {
          state.closeContextMenu();
        } else if (state.selectedNodeId) {
          state.selectNode(null);
        }
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, []);

  const saveViewport = useCanvasStore((s) => s.saveViewport);
  const viewport = useCanvasStore((s) => s.viewport);
  const saveTimerRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  // Cleanup debounced save timer on unmount
  useEffect(() => {
    return () => clearTimeout(saveTimerRef.current);
  }, []);

  const onMoveEnd = useCallback(
    (_event: unknown, vp: { x: number; y: number; zoom: number }) => {
      // Debounce viewport saves to avoid spamming the API
      clearTimeout(saveTimerRef.current);
      saveTimerRef.current = setTimeout(() => {
        saveViewport(vp.x, vp.y, vp.zoom);
      }, 1000);
    },
    [saveViewport]
  );

  const defaultViewport = useMemo(
    () => ({ x: viewport.x, y: viewport.y, zoom: viewport.zoom }),
    // Only use the initial viewport — don't re-render on every save
    // eslint-disable-next-line react-hooks/exhaustive-deps
    []
  );

  // Determine which workspace ID to use for global settings.
  // Fall back to "global" when no specific node is selected.
  const settingsWorkspaceId = selectedNodeId ?? "global";

  return (
    <div className="w-screen h-screen bg-zinc-950">
      <ReactFlow
        colorMode="dark"
        nodes={nodes}
        edges={edges}
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
          nodeStrokeWidth={0}
          nodeBorderRadius={4}
        />
      </ReactFlow>

      {nodes.length === 0 && <EmptyState />}
      <OnboardingWizard />
      <Toolbar />
      <ApprovalBanner />
      <BundleDropZone />
      <TemplatePalette />
      <SidePanel />
      <ContextMenu />
      <SearchDialog />
      <Toaster />
      {/* <ProvisioningTimeout /> */}
      {!selectedNodeId && <CreateWorkspaceButton />}

      {/* Confirmation dialog for structure changes */}
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

      {/* Settings Panel — global secrets management drawer */}
      <SettingsPanel workspaceId={settingsWorkspaceId} />
      <DeleteConfirmDialog workspaceId={settingsWorkspaceId} />
    </div>
  );
}
