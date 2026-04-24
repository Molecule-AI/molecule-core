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
import {
  defaultChildSlot,
  CHILD_DEFAULT_HEIGHT,
  CHILD_DEFAULT_WIDTH,
} from "@/store/canvas-topology";
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
import { Toaster } from "./Toaster";
import { Toolbar } from "./Toolbar";
import { ConfirmDialog } from "./ConfirmDialog";
import { api } from "@/lib/api";
import { showToast } from "./Toaster";
// Phase 20 components
import { SettingsPanel, DeleteConfirmDialog } from "./settings";
// Phase 20.3 batch operations
import { BatchActionBar } from "./BatchActionBar";
import { ProvisioningTimeout } from "./ProvisioningTimeout";

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

// Hysteresis: detach-on-drop only fires once the child has moved far
// enough outside the parent that the intent is unambiguous. We pick 20%
// of the overlapping dimension as the threshold (Miro behaves similarly
// at ~20-30%). A slightly-past-edge drag commits a MOVE, not a detach.
const DETACH_FRACTION = 0.2;

/** Floating "Drop into: <name>" label that tracks the current drag
 *  target. Mural-style affordance — colour alone is ambiguous on dense
 *  canvases, so we spell out the target by name. Mounted inside the
 *  ReactFlowProvider subtree so it can read positionAbsolute. */
function DropTargetBadge() {
  const dragOverNodeId = useCanvasStore((s) => s.dragOverNodeId);
  const targetName = useCanvasStore((s) => {
    if (!s.dragOverNodeId) return null;
    const n = s.nodes.find((nn) => nn.id === s.dragOverNodeId);
    return (n?.data as WorkspaceNodeData | undefined)?.name ?? null;
  });
  const childCount = useCanvasStore((s) =>
    !s.dragOverNodeId
      ? 0
      : s.nodes.filter((n) => n.parentId === s.dragOverNodeId).length,
  );
  const { getInternalNode, flowToScreenPosition } = useReactFlow();
  if (!dragOverNodeId || !targetName) return null;
  const internal = getInternalNode(dragOverNodeId);
  if (!internal) return null;
  const abs = internal.internals.positionAbsolute;
  const w = internal.measured?.width ?? 220;
  const h = internal.measured?.height ?? 120;
  const badge = flowToScreenPosition({ x: abs.x + w / 2, y: abs.y });

  // Ghost preview: dashed outline at the next default grid slot inside
  // the target parent. Whimsical-style affordance so the user sees
  // exactly where the dropped card will land.
  const slot = defaultChildSlot(childCount);
  const slotTL = flowToScreenPosition({ x: abs.x + slot.x, y: abs.y + slot.y });
  const slotBR = flowToScreenPosition({
    x: abs.x + slot.x + CHILD_DEFAULT_WIDTH,
    y: abs.y + slot.y + CHILD_DEFAULT_HEIGHT,
  });
  // Clip the ghost to the parent's visible bounds so it doesn't spill
  // out when the parent is smaller than the slot.
  const parentTL = flowToScreenPosition({ x: abs.x, y: abs.y });
  const parentBR = flowToScreenPosition({ x: abs.x + w, y: abs.y + h });
  const ghostVisible =
    slotBR.x > parentTL.x &&
    slotTL.x < parentBR.x &&
    slotBR.y > parentTL.y &&
    slotTL.y < parentBR.y;

  return (
    <>
      {ghostVisible && (
        <div
          className="pointer-events-none absolute z-40 rounded-lg border-2 border-dashed border-emerald-400/70 bg-emerald-500/10"
          style={{
            left: slotTL.x,
            top: slotTL.y,
            width: slotBR.x - slotTL.x,
            height: slotBR.y - slotTL.y,
          }}
        />
      )}
      <div
        className="pointer-events-none absolute z-50 -translate-x-1/2 -translate-y-full rounded-md bg-emerald-500 px-2 py-0.5 text-[11px] font-medium text-emerald-50 shadow-lg shadow-emerald-950/40"
        style={{ left: badge.x, top: badge.y - 6 }}
      >
        Drop into: {targetName}
      </div>
    </>
  );
}

/** Snap a child back so its bbox is fully inside the parent's bounds.
 *  Called on drag-stop when the user drifted slightly past the edge
 *  without holding Alt or Cmd — the canvas treats the gesture as a
 *  plain move rather than an un-nest. */
function clampChildIntoParent(
  childId: string,
  parentId: string,
  getInternalNode: (id: string) => ReturnType<ReturnType<typeof useReactFlow>["getInternalNode"]>,
) {
  const c = getInternalNode(childId);
  const p = getInternalNode(parentId);
  if (!c || !p) return;
  const cw = c.measured?.width ?? c.width ?? 220;
  const ch = c.measured?.height ?? c.height ?? 120;
  const pw = p.measured?.width ?? p.width ?? 220;
  const ph = p.measured?.height ?? p.height ?? 120;
  const { nodes } = useCanvasStore.getState();
  const cur = nodes.find((n) => n.id === childId);
  if (!cur) return;
  const rel = cur.position;
  const clampedX = Math.max(0, Math.min(rel.x, pw - cw));
  const clampedY = Math.max(0, Math.min(rel.y, ph - ch));
  if (clampedX === rel.x && clampedY === rel.y) return;
  useCanvasStore.setState({
    nodes: nodes.map((n) =>
      n.id === childId ? { ...n, position: { x: clampedX, y: clampedY } } : n,
    ),
  });
}

function shouldDetach(
  childId: string,
  parentId: string,
  getInternalNode: (id: string) => ReturnType<ReturnType<typeof useReactFlow>["getInternalNode"]>,
): boolean {
  const c = getInternalNode(childId);
  const p = getInternalNode(parentId);
  if (!c || !p) return true; // If we can't measure, fall back to the old behavior.
  const cw = c.measured?.width ?? c.width ?? 220;
  const ch = c.measured?.height ?? c.height ?? 120;
  const pw = p.measured?.width ?? p.width ?? 220;
  const ph = p.measured?.height ?? p.height ?? 120;
  const cx = c.internals.positionAbsolute;
  const px = p.internals.positionAbsolute;
  const overlapW =
    Math.max(0, Math.min(cx.x + cw, px.x + pw) - Math.max(cx.x, px.x));
  const overlapH =
    Math.max(0, Math.min(cx.y + ch, px.y + ph) - Math.max(cx.y, px.y));
  const outsideFractionX = 1 - overlapW / cw;
  const outsideFractionY = 1 - overlapH / ch;
  return outsideFractionX > DETACH_FRACTION || outsideFractionY > DETACH_FRACTION;
}

function CanvasInner() {
  const nodes = useCanvasStore((s) => s.nodes);
  const edges = useCanvasStore((s) => s.edges);
  const a2aEdges = useCanvasStore((s) => s.a2aEdges);
  const showA2AEdges = useCanvasStore((s) => s.showA2AEdges);
  // Merge topology edges with A2A overlay edges via useMemo (no new object in selector)
  const allEdges = useMemo(
    () => (showA2AEdges ? [...edges, ...a2aEdges] : edges),
    [edges, a2aEdges, showA2AEdges]
  );
  const onNodesChange = useCanvasStore((s) => s.onNodesChange);
  const savePosition = useCanvasStore((s) => s.savePosition);
  const selectNode = useCanvasStore((s) => s.selectNode);
  const selectedNodeId = useCanvasStore((s) => s.selectedNodeId);
  const setDragOverNode = useCanvasStore((s) => s.setDragOverNode);
  const nestNode = useCanvasStore((s) => s.nestNode);
  const isDescendant = useCanvasStore((s) => s.isDescendant);
  const dragStartParentRef = useRef<string | null>(null);
  const dragModifiersRef = useRef<{ alt: boolean; meta: boolean }>({ alt: false, meta: false });
  const { getInternalNode } = useReactFlow();

  // Track Alt / Cmd-Meta during the whole drag so onNodeDrag and
  // onNodeDragStop see the same modifier state. (React Flow's drag event
  // only fires mousemove events — we attach a window-level keyboard
  // listener while a drag is in progress.)
  const onNodeDragStart: OnNodeDrag<Node<WorkspaceNodeData>> = useCallback(
    (event, node) => {
      dragStartParentRef.current = (node.data as WorkspaceNodeData).parentId;
      dragModifiersRef.current = {
        alt: event.altKey,
        meta: event.metaKey || event.ctrlKey,
      };
    },
    [],
  );

  // Absolute-bounds hit test. Returns the **best** drop target among the
  // candidates whose measured bbox contains `point`. Tiebreakers, in
  // order (matches Figma / tldraw / xyflow issue #2827 community fix):
  //
  //   1. DEEPEST tree depth first — dropping onto a nested grandchild
  //      lands on the grandchild, not its outermost ancestor.
  //   2. Highest zIndex second — when nested parents overlap with equal
  //      depth (siblings of each other), the one rendered above wins.
  //   3. Smallest area last — visually-tightest match otherwise.
  //
  // Self + descendants are excluded (can't nest something under itself).
  const findDropTarget = useCallback(
    (draggedId: string, point: { x: number; y: number }): string | null => {
      const all = useCanvasStore.getState().nodes;
      // Tree depth for each node — depth = ancestor count.
      const depthOf = (id: string | null | undefined): number => {
        let d = 0;
        let cursor: string | null | undefined = id;
        while (cursor) {
          const n = all.find((nn) => nn.id === cursor);
          if (!n) break;
          cursor = n.data.parentId;
          d += 1;
        }
        return d;
      };
      let best:
        | { id: string; depth: number; zIndex: number; area: number }
        | null = null;
      for (const n of all) {
        if (n.id === draggedId || isDescendant(draggedId, n.id)) continue;
        const internal = getInternalNode(n.id);
        if (!internal) continue;
        const abs = internal.internals.positionAbsolute;
        const w = internal.measured?.width ?? n.width ?? 220;
        const h = internal.measured?.height ?? n.height ?? 120;
        if (point.x < abs.x || point.x > abs.x + w) continue;
        if (point.y < abs.y || point.y > abs.y + h) continue;
        const depth = depthOf(n.id);
        const z = n.zIndex ?? 0;
        const area = w * h;
        if (
          !best ||
          depth > best.depth ||
          (depth === best.depth && z > best.zIndex) ||
          (depth === best.depth && z === best.zIndex && area < best.area)
        ) {
          best = { id: n.id, depth, zIndex: z, area };
        }
      }
      return best?.id ?? null;
    },
    [getInternalNode, isDescendant],
  );

  const onNodeDrag: OnNodeDrag<Node<WorkspaceNodeData>> = useCallback(
    (event, node) => {
      dragModifiersRef.current = {
        alt: event.altKey,
        meta: event.metaKey || event.ctrlKey,
      };
      const internal = getInternalNode(node.id);
      if (!internal) {
        setDragOverNode(null);
        return;
      }
      const abs = internal.internals.positionAbsolute;
      const w = internal.measured?.width ?? 220;
      const h = internal.measured?.height ?? 120;
      const center = { x: abs.x + w / 2, y: abs.y + h / 2 };
      setDragOverNode(findDropTarget(node.id, center));
    },
    [findDropTarget, getInternalNode, setDragOverNode],
  );

  // Confirmation dialog state for structure changes
  const [pendingNest, setPendingNest] = useState<{ nodeId: string; targetId: string | null; nodeName: string; targetName: string } | null>(null);
  // Delete-confirmation lives in the store so the dialog survives ContextMenu
  // unmounting — the prior local-in-ContextMenu state raced with the menu's
  // outside-click handler (the portal-rendered Confirm button counted as
  // "outside" and closed the menu, killing the dialog mid-click).
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

  // Cascade guard: include child count in the warning message when the workspace
  // has children, so the user understands the blast radius before clicking Delete All.
  const cascadeMessage = pendingDelete?.hasChildren
    ? `⚠️ Deleting "${pendingDelete.name}" will permanently delete all child workspaces and their data. This cannot be undone.`
    : null;

  const onNodeDragStop: OnNodeDrag<Node<WorkspaceNodeData>> = useCallback(
    (event, node) => {
      const { dragOverNodeId, nodes: allNodes } = useCanvasStore.getState();
      setDragOverNode(null);

      const nodeName = (node.data as WorkspaceNodeData).name;
      const currentParentId = (node.data as WorkspaceNodeData).parentId;
      const altHeld = event.altKey || dragModifiersRef.current.alt;
      const forceDetach =
        event.metaKey || event.ctrlKey || dragModifiersRef.current.meta;

      // Soft clamp: without a modifier, a child dropped just past its
      // parent's edge is snapped back inside (Alt-drag escapes this to
      // allow re-parenting). The explicit nest gesture (drop inside
      // another parent) always wins over the clamp.
      const droppingIntoAnotherParent =
        !!dragOverNodeId && dragOverNodeId !== currentParentId;
      if (
        currentParentId &&
        !altHeld &&
        !forceDetach &&
        !droppingIntoAnotherParent &&
        shouldDetach(node.id, currentParentId, getInternalNode)
      ) {
        clampChildIntoParent(node.id, currentParentId, getInternalNode);
      }

      // The drag-stop offers several possible intents. Hysteresis
      // (Miro/tldraw pattern) keeps a child nested unless it's clearly
      // outside the parent — a twitchy release 1px past the edge no
      // longer un-nests. Cmd / Ctrl (forceDetach) or Alt (escape)
      // bypass the clamp.
      if (droppingIntoAnotherParent) {
        const targetNode = allNodes.find((n) => n.id === dragOverNodeId);
        const targetName = targetNode?.data.name || "Unknown";
        setPendingNest({ nodeId: node.id, targetId: dragOverNodeId, nodeName, targetName });
      } else if (
        currentParentId &&
        (forceDetach || (altHeld && shouldDetach(node.id, currentParentId, getInternalNode)))
      ) {
        const parentNode = allNodes.find((n) => n.id === currentParentId);
        const parentName = parentNode?.data.name || "Unknown";
        setPendingNest({ nodeId: node.id, targetId: null, nodeName, targetName: parentName });
      }

      // savePosition expects ABSOLUTE coords. When node is a child, its
      // `position` is relative to its parent, so translate through the
      // measured absolute position React Flow tracks.
      const internal = getInternalNode(node.id);
      const abs = internal?.internals.positionAbsolute ?? node.position;
      savePosition(node.id, abs.x, abs.y);
      // Commit-on-release grow: run the parent auto-grow pass once now
      // that the drag has settled. Cheap and deterministic vs running
      // grow on every drag tick (avoids tldraw's edge-chase artifact).
      useCanvasStore.getState().growParentsToFitChildren();
    },
    [getInternalNode, savePosition, setDragOverNode],
  );

  const batchNest = useCanvasStore((s) => s.batchNest);
  const confirmNest = useCallback(() => {
    if (!pendingNest) return;
    const state = useCanvasStore.getState();
    // If the primary dragged node is part of a batch selection, apply
    // the same nest target to every selected node — preserves the
    // selection's inter-node spacing (Lucidchart pattern).
    if (
      state.selectedNodeIds.size > 1 &&
      state.selectedNodeIds.has(pendingNest.nodeId)
    ) {
      batchNest(Array.from(state.selectedNodeIds), pendingNest.targetId);
    } else {
      nestNode(pendingNest.nodeId, pendingNest.targetId);
    }
    setPendingNest(null);
  }, [pendingNest, nestNode, batchNest]);

  const cancelNest = useCallback(() => {
    setPendingNest(null);
  }, []);

  const onPaneClick = useCallback(() => {
    selectNode(null);
    const state = useCanvasStore.getState();
    state.closeContextMenu();
    state.clearSelection();
  }, [selectNode]);

  // Team zoom-in: double-click a team node to zoom to its children
  const { fitBounds, fitView } = useReactFlow();

  // Pan to newly deployed workspace.
  // Uses fitView({ nodes }) so the viewport adapts to any current zoom level
  // instead of forcing zoom=1 (which was jarring when the user was zoomed out).
  const panTimerRef = useRef<ReturnType<typeof setTimeout>>(undefined);
  useEffect(() => {
    const handler = (e: Event) => {
      const { nodeId } = (e as CustomEvent<{ nodeId: string }>).detail;
      // Small delay so ReactFlow has time to measure the newly rendered node
      clearTimeout(panTimerRef.current);
      panTimerRef.current = setTimeout(() => {
        fitView({ nodes: [{ id: nodeId }], duration: 400, padding: 0.3 });
      }, 100);
    };
    window.addEventListener("molecule:pan-to-node", handler);
    return () => {
      window.removeEventListener("molecule:pan-to-node", handler);
      clearTimeout(panTimerRef.current);
    };
  }, [fitView]);
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
      const tag = (e.target as HTMLElement).tagName;
      const inInput =
        tag === "INPUT" ||
        tag === "TEXTAREA" ||
        tag === "SELECT" ||
        (e.target as HTMLElement).isContentEditable;

      if (e.key === "Escape") {
        const state = useCanvasStore.getState();
        if (state.contextMenu) {
          state.closeContextMenu();
        } else if (state.selectedNodeIds.size > 0) {
          state.clearSelection();
        } else if (state.selectedNodeId) {
          state.selectNode(null);
        }
      }

      // Figma-style hierarchy navigation. Enter descends to the first
      // child of the selected node; Shift+Enter ascends to its parent;
      // Cmd+]/[ re-orders siblings (z-index up/down). Skipped when the
      // user is typing into an input — Enter should commit the form.
      if (!inInput && (e.key === "Enter" || e.key === "NumpadEnter")) {
        e.preventDefault();
        const state = useCanvasStore.getState();
        const id = state.selectedNodeId;
        if (!id) return;
        if (e.shiftKey) {
          const sel = state.nodes.find((n) => n.id === id);
          const parentId = sel?.data.parentId ?? null;
          if (parentId) state.selectNode(parentId);
        } else {
          const firstChild = state.nodes.find((n) => n.data.parentId === id);
          if (firstChild) state.selectNode(firstChild.id);
        }
      }

      if (
        !inInput &&
        (e.metaKey || e.ctrlKey) &&
        (e.key === "]" || e.key === "[")
      ) {
        e.preventDefault();
        const state = useCanvasStore.getState();
        const id = state.selectedNodeId;
        if (!id) return;
        state.bumpZOrder(id, e.key === "]" ? 1 : -1);
      }

      // Z — keyboard equivalent for double-click zoom-to-team (WCAG 2.1.1)
      if (!inInput && (e.key === "z" || e.key === "Z")) {
        const state = useCanvasStore.getState();
        const selectedId = state.selectedNodeId;
        if (!selectedId) return;
        const hasChildren = state.nodes.some((n) => n.data.parentId === selectedId);
        if (hasChildren) {
          window.dispatchEvent(
            new CustomEvent("molecule:zoom-to-team", { detail: { nodeId: selectedId } })
          );
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

      {/* Screen-reader live region: announces workspace count when canvas loads or changes */}
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

      {/* Confirmation dialog for workspace delete — driven by store */}
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

      {/* Settings Panel — global secrets management drawer */}
      <SettingsPanel workspaceId={settingsWorkspaceId} />
      <DeleteConfirmDialog workspaceId={settingsWorkspaceId} />
      </main>
    </>
  );
}
