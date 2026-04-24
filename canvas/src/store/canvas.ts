import { create } from "zustand";
import {
  type Node,
  type Edge,
  applyNodeChanges,
  type NodeChange,
} from "@xyflow/react";
import { api } from "@/lib/api";
import type { WorkspaceData, WSMessage } from "./socket";
import { handleCanvasEvent } from "./canvas-events";
import {
  buildNodesAndEdges,
  computeAutoLayout,
  defaultChildSlot,
  CHILD_DEFAULT_HEIGHT,
  CHILD_DEFAULT_WIDTH,
  PARENT_BOTTOM_PADDING,
  PARENT_SIDE_PADDING,
} from "./canvas-topology";

/**
 * Walk every parent node and bump its width/height (if explicitly set)
 * so the union of its children's relative bboxes plus padding fits. A
 * parent's size never shrinks via this path — only grows — because
 * shrinking on resize would fight the user's own NodeResizer drag.
 */
function growParentsToFitChildren<T extends Record<string, unknown>>(
  nodes: Node<T>[],
): Node<T>[] {
  // Index children by parentId so the scan is O(n).
  const childrenByParent = new Map<string, Node<T>[]>();
  for (const n of nodes) {
    if (!n.parentId) continue;
    const arr = childrenByParent.get(n.parentId) ?? [];
    arr.push(n);
    childrenByParent.set(n.parentId, arr);
  }
  let changed = false;
  const out = nodes.map((n) => {
    const kids = childrenByParent.get(n.id);
    if (!kids || kids.length === 0) return n;
    // Collapsed parents intentionally render compact — skip the grow
    // pass so their size isn't pushed back out by their hidden kids.
    const nData = n.data as unknown as WorkspaceNodeData | undefined;
    if (nData?.collapsed) return n;
    let maxRight = 0;
    let maxBottom = 0;
    for (const k of kids) {
      const w = (k.measured?.width ?? k.width ?? CHILD_DEFAULT_WIDTH) as number;
      const h = (k.measured?.height ?? k.height ?? CHILD_DEFAULT_HEIGHT) as number;
      maxRight = Math.max(maxRight, k.position.x + w);
      maxBottom = Math.max(maxBottom, k.position.y + h);
    }
    const requiredW = maxRight + PARENT_SIDE_PADDING;
    const requiredH = maxBottom + PARENT_BOTTOM_PADDING;
    const currentW = (n.measured?.width ?? n.width ?? 0) as number;
    const currentH = (n.measured?.height ?? n.height ?? 0) as number;
    if (requiredW <= currentW && requiredH <= currentH) return n;
    changed = true;
    return {
      ...n,
      width: Math.max(currentW, requiredW),
      height: Math.max(currentH, requiredH),
    };
  });
  return changed ? out : nodes;
}

// Re-export extracted types and functions so existing imports from "@/store/canvas" keep working
export { summarizeWorkspaceCapabilities } from "./canvas-capabilities";
export type { WorkspaceCapabilitySummary } from "./canvas-capabilities";

export interface WorkspaceNodeData extends Record<string, unknown> {
  name: string;
  status: string;
  tier: number;
  agentCard: Record<string, unknown> | null;
  activeTasks: number;
  collapsed: boolean;
  role: string;
  lastErrorRate: number;
  lastSampleError: string;
  url: string;
  parentId: string | null;
  currentTask: string;
  runtime: string;
  needsRestart: boolean;
  /** USD spend ceiling set by the user; null = unlimited. Added by issue #541. */
  budgetLimit: number | null;
  /** Cumulative USD spend. Present when the platform tracks spend (issue #541). */
  budgetUsed?: number | null;
}

export type PanelTab = "details" | "skills" | "chat" | "terminal" | "config" | "schedule" | "channels" | "files" | "memory" | "traces" | "events" | "activity" | "audit";

export interface ContextMenuState {
  x: number;
  y: number;
  nodeId: string;
  nodeData: WorkspaceNodeData;
}

interface CanvasState {
  nodes: Node<WorkspaceNodeData>[];
  edges: Edge[];
  selectedNodeId: string | null;
  panelTab: PanelTab;
  dragOverNodeId: string | null;
  contextMenu: ContextMenuState | null;
  // Live width of the SidePanel in pixels. Only meaningful when
  // selectedNodeId is non-null (panel visible). The Toolbar reads this
  // to stay centred on the remaining canvas area instead of the full
  // viewport, so the "Audit" / "Search" / "Settings" buttons don't get
  // hidden behind the panel when a workspace is selected.
  sidePanelWidth: number;
  setSidePanelWidth: (w: number) => void;
  // Whether the TemplatePalette left-drawer is open. Consumed by the
  // Legend so it can shift right and avoid being hidden under the
  // palette. Set by TemplatePalette's toggle button.
  templatePaletteOpen: boolean;
  setTemplatePaletteOpen: (open: boolean) => void;
  hydrate: (workspaces: WorkspaceData[]) => void;
  applyEvent: (msg: WSMessage) => void;
  onNodesChange: (changes: NodeChange<Node<WorkspaceNodeData>>[]) => void;
  savePosition: (nodeId: string, x: number, y: number) => void;
  selectNode: (id: string | null) => void;
  setPanelTab: (tab: PanelTab) => void;
  getSelectedNode: () => Node<WorkspaceNodeData> | null;
  updateNodeData: (id: string, data: Partial<WorkspaceNodeData>) => void;
  restartWorkspace: (id: string) => Promise<void>;
  removeNode: (id: string) => void;
  setDragOverNode: (id: string | null) => void;
  nestNode: (draggedId: string, targetId: string | null) => Promise<void>;
  isDescendant: (ancestorId: string, nodeId: string) => boolean;
  /** Re-order siblings in z-index space. `direction = +1` sends the node
   *  one step forward among its parent's children (or among canvas
   *  roots); -1 sends it one step back. Figma Cmd+]/[ parity. */
  bumpZOrder: (nodeId: string, direction: 1 | -1) => void;
  /** Re-parent many nodes at once, preserving each node's absolute
   *  position. Lucidchart pattern: drag a selection into a frame and
   *  the inter-node layout stays intact. Used when the primary dragged
   *  node of a multi-select drag triggers a nest confirmation. */
  batchNest: (nodeIds: string[], targetId: string | null) => Promise<void>;
  /** Run the parent auto-grow pass once. Canvas.onNodeDragStop calls
   *  this so a drag that pushed a child past the parent edge commits
   *  the parent grow on release (commit-on-release pattern). */
  growParentsToFitChildren: () => void;
  /** Re-layout a parent's children to the default 2-column grid. Used
   *  by the "Arrange children" context-menu command so users can rescue
   *  out-of-bounds children on demand — topology no longer does it
   *  automatically (P3.12 opt-in rescue). */
  arrangeChildren: (parentId: string) => void;
  /** Toggle the collapsed flag on a parent and hide/show every
   *  descendant so the card renders as a compact header-only frame.
   *  Miro "frame outline view" analog. */
  setCollapsed: (parentId: string, collapsed: boolean) => void;
  openContextMenu: (menu: ContextMenuState) => void;
  closeContextMenu: () => void;
  // Pending delete confirmation — lives in the store (not inside ContextMenu's
  // local state) so the confirm dialog survives ContextMenu unmounting. The
  // ContextMenu's portal-rendered dialog used to race with its outside-click
  // handler: clicking Confirm registered as "outside", closed the menu, and
  // unmounted the dialog before its onClick fired. Hoisting the state fixes
  // that — see fix/context-menu-delete-race.
  pendingDelete:
    | { id: string; name: string; hasChildren: boolean; children: { id: string; name: string }[] }
    | null;
  setPendingDelete: (
    v: { id: string; name: string; hasChildren: boolean; children: { id: string; name: string }[] } | null
  ) => void;
  searchOpen: boolean;
  setSearchOpen: (open: boolean) => void;
  viewport: { x: number; y: number; zoom: number };
  setViewport: (v: { x: number; y: number; zoom: number }) => void;
  saveViewport: (x: number, y: number, zoom: number) => void;
  // ── Batch selection (Phase 20.3) ─────────────────────────────────────────
  selectedNodeIds: Set<string>;
  toggleNodeSelection: (id: string) => void;
  clearSelection: () => void;
  batchRestart: () => Promise<void>;
  batchPause: () => Promise<void>;
  batchDelete: () => Promise<void>;
  /** Agent-pushed messages keyed by workspace ID. ChatTab consumes and clears these. */
  agentMessages: Record<string, Array<{ id: string; content: string; timestamp: string }>>;
  consumeAgentMessages: (workspaceId: string) => Array<{ id: string; content: string; timestamp: string }>;
  /** WebSocket connection status — drives the live indicator in the Toolbar. */
  wsStatus: "connected" | "connecting" | "disconnected";
  setWsStatus: (status: "connected" | "connecting" | "disconnected") => void;
  /** Hydration error message — set when initial canvas load fails. Null when no error. */
  hydrationError: string | null;
  setHydrationError: (error: string | null) => void;
  // ── A2A topology overlay (issue #744) ─────────────────────────────────────
  /** Directed delegation edges shown as an overlay on the canvas (separate from topology edges). */
  a2aEdges: Edge[];
  setA2AEdges: (edges: Edge[]) => void;
  /** Whether the A2A topology overlay is visible. Persisted to localStorage. Default: true. */
  showA2AEdges: boolean;
  setShowA2AEdges: (show: boolean) => void;
}

export const useCanvasStore = create<CanvasState>((set, get) => ({
  nodes: [],
  edges: [],
  selectedNodeId: null,
  panelTab: "chat",
  dragOverNodeId: null,
  contextMenu: null,
  sidePanelWidth: 480, // matches SIDEPANEL_DEFAULT_WIDTH in SidePanel.tsx
  setSidePanelWidth: (w) => set({ sidePanelWidth: w }),
  templatePaletteOpen: false,
  setTemplatePaletteOpen: (open) => set({ templatePaletteOpen: open }),
  // Batch selection
  selectedNodeIds: new Set<string>(),
  toggleNodeSelection: (id) => {
    const prev = get().selectedNodeIds;
    const next = new Set(prev);
    if (next.has(id)) {
      next.delete(id);
    } else {
      next.add(id);
    }
    set({ selectedNodeIds: next });
  },
  clearSelection: () => set({ selectedNodeIds: new Set<string>() }),
  batchRestart: async () => {
    const ids = Array.from(get().selectedNodeIds);
    const results = await Promise.allSettled(
      ids.map((id) => api.post(`/workspaces/${id}/restart`))
    );
    const failed: string[] = [];
    results.forEach((r, i) => {
      if (r.status === "fulfilled") {
        get().updateNodeData(ids[i], { needsRestart: false });
      } else {
        failed.push(ids[i]);
      }
    });
    // Keep failed IDs selected so the user can retry; drop the successful ones.
    set({ selectedNodeIds: new Set(failed) });
    if (failed.length > 0) {
      throw new Error(`${failed.length}/${ids.length} restart(s) failed`);
    }
  },
  batchPause: async () => {
    const ids = Array.from(get().selectedNodeIds);
    const results = await Promise.allSettled(
      ids.map((id) => api.post(`/workspaces/${id}/pause`))
    );
    const failed: string[] = [];
    results.forEach((r, i) => {
      if (r.status !== "fulfilled") failed.push(ids[i]);
    });
    set({ selectedNodeIds: new Set(failed) });
    if (failed.length > 0) {
      throw new Error(`${failed.length}/${ids.length} pause(s) failed`);
    }
  },
  batchDelete: async () => {
    const ids = Array.from(get().selectedNodeIds);
    const results = await Promise.allSettled(
      ids.map((id) => api.del(`/workspaces/${id}`))
    );
    const failed: string[] = [];
    results.forEach((r, i) => {
      if (r.status === "fulfilled") {
        get().removeNode(ids[i]);
      } else {
        failed.push(ids[i]);
      }
    });
    // Keep failed IDs selected so the user can retry; the successful ones
    // were already removed from `nodes` above.
    set({ selectedNodeIds: new Set(failed) });
    if (failed.length > 0) {
      throw new Error(`${failed.length}/${ids.length} delete(s) failed`);
    }
  },
  wsStatus: "connecting",
  setWsStatus: (status) => set({ wsStatus: status }),
  hydrationError: null,
  setHydrationError: (error) => set({ hydrationError: error }),
  // A2A overlay — default on, persisted to localStorage
  a2aEdges: [],
  setA2AEdges: (edges) => set({ a2aEdges: edges }),
  showA2AEdges:
    typeof window !== "undefined"
      ? localStorage.getItem("molecule:show-a2a-edges") !== "false"
      : true,
  setShowA2AEdges: (show) => {
    set({ showA2AEdges: show });
    if (typeof window !== "undefined") {
      localStorage.setItem("molecule:show-a2a-edges", String(show));
    }
  },

  viewport: { x: 0, y: 0, zoom: 1 },

  selectNode: (id) => set({ selectedNodeId: id }),
  openContextMenu: (menu) => set({ contextMenu: menu }),
  closeContextMenu: () => set({ contextMenu: null }),
  pendingDelete: null,
  setPendingDelete: (v) => set({ pendingDelete: v }),
  searchOpen: false,
  setSearchOpen: (open) => set({ searchOpen: open }),
  agentMessages: {},
  consumeAgentMessages: (workspaceId) => {
    const msgs = get().agentMessages[workspaceId] || [];
    if (msgs.length > 0) {
      const { agentMessages } = get();
      const { [workspaceId]: _, ...rest } = agentMessages;
      set({ agentMessages: rest });
    }
    return msgs;
  },
  setViewport: (v) => set({ viewport: v }),
  saveViewport: async (x, y, zoom) => {
    set({ viewport: { x, y, zoom } });
    try {
      await api.put(`/canvas/viewport`, { x, y, zoom });
    } catch {
      // Non-critical — viewport save failure doesn't block user
    }
  },
  setPanelTab: (tab) => set({ panelTab: tab }),
  setDragOverNode: (id) => set({ dragOverNodeId: id }),

  batchNest: async (nodeIds, targetId) => {
    if (nodeIds.length === 0) return;
    if (nodeIds.length === 1) {
      await get().nestNode(nodeIds[0], targetId);
      return;
    }
    // Run sequentially so each nestNode's absolute-position calc sees
    // the previous update committed. Not a hot path — multi-select
    // re-parents rarely touch more than a handful of nodes.
    for (const id of nodeIds) {
      await get().nestNode(id, targetId);
    }
  },

  bumpZOrder: (nodeId, direction) => {
    const { nodes } = get();
    const target = nodes.find((n) => n.id === nodeId);
    if (!target) return;
    // Siblings = nodes sharing the same parent (null for roots).
    const siblings = nodes.filter(
      (n) => n.data.parentId === target.data.parentId,
    );
    if (siblings.length < 2) return;
    // React Flow uses a flat zIndex; we keep children above parents
    // (+1 per depth) so any nudge here stays within the sibling tier.
    // Reorder in zIndex space by adjusting the target +/- 1.
    const current = target.zIndex ?? 0;
    const newZ = current + direction;
    set({
      nodes: nodes.map((n) =>
        n.id === nodeId ? { ...n, zIndex: newZ } : n,
      ),
    });
  },

  isDescendant: (ancestorId, nodeId) => {
    const { nodes } = get();
    let current = nodes.find((n) => n.id === nodeId);
    while (current?.data.parentId) {
      if (current.data.parentId === ancestorId) return true;
      current = nodes.find((n) => n.id === current?.data.parentId);
    }
    return false;
  },

  nestNode: async (draggedId, targetId) => {
    const { nodes, edges } = get();
    const dragged = nodes.find((n) => n.id === draggedId);
    if (!dragged) return;
    const currentParentId = dragged.data.parentId;
    if (currentParentId === targetId) return;

    // Compute each ancestor's absolute position by walking up the
    // parentId chain. We need this to translate the dragged node's
    // `position` (relative to its current parent when nested) between
    // the old and new coordinate spaces so the card doesn't visually
    // jump on nest/unnest.
    const absOf = (id: string | null): { x: number; y: number } => {
      let sum = { x: 0, y: 0 };
      let cursor: string | null = id;
      while (cursor) {
        const n = nodes.find((nn) => nn.id === cursor);
        if (!n) break;
        sum = { x: sum.x + n.position.x, y: sum.y + n.position.y };
        cursor = n.data.parentId;
      }
      return sum;
    };
    const oldParentAbs = absOf(currentParentId);
    const newParentAbs = absOf(targetId);
    const draggedAbs = {
      x: dragged.position.x + oldParentAbs.x,
      y: dragged.position.y + oldParentAbs.y,
    };
    const newRelative = {
      x: draggedAbs.x - newParentAbs.x,
      y: draggedAbs.y - newParentAbs.y,
    };

    const newEdges = edges.filter(
      (e) => e.source !== draggedId && e.target !== draggedId,
    );

    // Depth walk so zIndex gets bumped correctly on nest/unnest
    // (children render above their new ancestor chain).
    const depthOf = (id: string | null | undefined): number => {
      let d = 0;
      let cursor: string | null | undefined = id;
      while (cursor) {
        const n = nodes.find((nn) => nn.id === cursor);
        if (!n) break;
        cursor = n.data.parentId;
        d += 1;
      }
      return d;
    };
    const newDepth = depthOf(targetId) + (targetId ? 1 : 0);

    set({
      nodes: nodes.map((n) =>
        n.id === draggedId
          ? {
              ...n,
              position: newRelative,
              parentId: targetId ?? undefined,
              zIndex: newDepth,
              data: { ...n.data, parentId: targetId },
            }
          : n,
      ),
      edges: newEdges,
    });

    try {
      await api.patch(`/workspaces/${draggedId}`, { parent_id: targetId });
      // Persist absolute position as DB canonical (matches what
      // savePosition writes elsewhere); keeps reloads stable regardless
      // of which parent the child was under at save time.
      await api.patch(`/workspaces/${draggedId}`, { x: draggedAbs.x, y: draggedAbs.y });
    } catch {
      set({
        nodes: get().nodes.map((n) =>
          n.id === draggedId
            ? {
                ...n,
                position: dragged.position,
                parentId: currentParentId ?? undefined,
                data: { ...n.data, parentId: currentParentId },
              }
            : n,
        ),
        edges,
      });
    }
  },

  getSelectedNode: () => {
    const { nodes, selectedNodeId } = get();
    if (!selectedNodeId) return null;
    return nodes.find((n) => n.id === selectedNodeId) ?? null;
  },

  updateNodeData: (id, data) => {
    set({
      nodes: get().nodes.map((n) =>
        n.id === id ? { ...n, data: { ...n.data, ...data } } : n
      ),
    });
  },

  restartWorkspace: async (id) => {
    await api.post(`/workspaces/${id}/restart`);
    get().updateNodeData(id, { needsRestart: false });
  },

  removeNode: (id) => {
    const { nodes, edges, selectedNodeId } = get();
    // Re-parent children to the deleted node's parent (or root).
    // Children are first-class RF nodes now — we just re-point their
    // parentId (both RF's native field and our data mirror). No hidden
    // flag is toggled because cards are always visible.
    const deletedNode = nodes.find((n) => n.id === id);
    const parentOfDeleted = deletedNode?.data.parentId ?? null;
    set({
      nodes: nodes
        .filter((n) => n.id !== id)
        .map((n) =>
          n.data.parentId === id
            ? {
                ...n,
                parentId: parentOfDeleted ?? undefined,
                data: { ...n.data, parentId: parentOfDeleted },
              }
            : n
        ),
      edges: edges.filter((e) => e.source !== id && e.target !== id),
      selectedNodeId: selectedNodeId === id ? null : selectedNodeId,
    });
  },

  hydrate: (workspaces: WorkspaceData[]) => {
    const layoutOverrides = computeAutoLayout(workspaces);
    const { nodes, edges } = buildNodesAndEdges(workspaces, layoutOverrides);
    set({ nodes, edges });
    for (const [nodeId, { x, y }] of layoutOverrides) {
      api.patch(`/workspaces/${nodeId}`, { x, y }).catch(() => {});
    }
  },

  applyEvent: (msg: WSMessage) => {
    handleCanvasEvent(msg, get, set);
  },

  onNodesChange: (changes) => {
    const next = applyNodeChanges(changes, get().nodes);
    // Parent auto-grow is intentionally conservative. Running
    // growParentsToFitChildren on every change (including the dozens of
    // position updates emitted during a single drag) caused the
    // "edge-chase" artifact tldraw documented — as the parent grows in
    // response to the child near its edge, the child's relative
    // position becomes valid again and the grow stops mid-drag, only to
    // resume on the next tick. Commit-on-release: only run grow when a
    // change set contains a `dimensions` change (NodeResizer commit),
    // not on pure `position` changes. Drag-stop grow is handled
    // explicitly in Canvas.onNodeDragStop via growOnce().
    const hasDimensionChange = changes.some((c) => c.type === "dimensions");
    set({ nodes: hasDimensionChange ? growParentsToFitChildren(next) : next });
  },

  growParentsToFitChildren: () => {
    set({ nodes: growParentsToFitChildren(get().nodes) });
  },

  setCollapsed: (parentId, collapsed) => {
    const { nodes } = get();
    // Find all descendant ids via BFS.
    const descendantIds = new Set<string>();
    const queue = [parentId];
    while (queue.length) {
      const id = queue.shift()!;
      for (const n of nodes) {
        if (n.data.parentId === id && !descendantIds.has(n.id)) {
          descendantIds.add(n.id);
          queue.push(n.id);
        }
      }
    }
    set({
      nodes: nodes.map((n) => {
        if (n.id === parentId) {
          return { ...n, data: { ...n.data, collapsed } };
        }
        if (descendantIds.has(n.id)) {
          return { ...n, hidden: collapsed };
        }
        return n;
      }),
    });
  },

  arrangeChildren: (parentId) => {
    const { nodes } = get();
    const kids = nodes
      .filter((n) => n.parentId === parentId)
      .sort((a, b) => (a.data.name || "").localeCompare(b.data.name || ""));
    if (kids.length === 0) return;
    const slotByKid = new Map<string, { x: number; y: number }>();
    kids.forEach((k, i) => slotByKid.set(k.id, defaultChildSlot(i)));
    set({
      nodes: nodes.map((n) => {
        const slot = slotByKid.get(n.id);
        return slot ? { ...n, position: slot } : n;
      }),
    });
    // Persist the new positions so they survive reload.
    for (const k of kids) {
      const slot = slotByKid.get(k.id)!;
      const parent = nodes.find((nn) => nn.id === parentId);
      if (!parent) continue;
      const absX = slot.x + parent.position.x;
      const absY = slot.y + parent.position.y;
      api.patch(`/workspaces/${k.id}`, { x: absX, y: absY }).catch(() => {});
    }
  },

  savePosition: async (nodeId: string, x: number, y: number) => {
    try {
      await api.patch(`/workspaces/${nodeId}`, { x, y });
    } catch {
      // Non-critical — position save failure doesn't block user
    }
  },
}));
