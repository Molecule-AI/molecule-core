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
import { buildNodesAndEdges, computeAutoLayout } from "./canvas-topology";

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
    const currentParentId = nodes.find((n) => n.id === draggedId)?.data.parentId ?? null;

    // No change needed
    if (currentParentId === targetId) return;

    // Optimistic update:
    // - Set parentId in data
    // - Hide child nodes (they render inside parent WorkspaceNode)
    // - Remove all edges involving the dragged node
    const newEdges = edges.filter(
      (e) => e.source !== draggedId && e.target !== draggedId
    );

    set({
      nodes: nodes.map((n) =>
        n.id === draggedId
          ? {
              ...n,
              hidden: !!targetId, // Hide if becoming a child, show if un-nesting
              data: { ...n.data, parentId: targetId },
            }
          : n
      ),
      edges: newEdges,
    });

    // Persist to API
    try {
      await api.patch(`/workspaces/${draggedId}`, { parent_id: targetId });
    } catch {
      // Revert on failure
      set({
        nodes: get().nodes.map((n) =>
          n.id === draggedId
            ? {
                ...n,
                hidden: !!currentParentId,
                data: { ...n.data, parentId: currentParentId },
              }
            : n
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
    // Re-parent children to the deleted node's parent (or root)
    const deletedNode = nodes.find((n) => n.id === id);
    const parentOfDeleted = deletedNode?.data.parentId ?? null;
    set({
      nodes: nodes
        .filter((n) => n.id !== id)
        .map((n) =>
          n.data.parentId === id
            ? {
                ...n,
                hidden: !!parentOfDeleted,
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
    set({
      nodes: applyNodeChanges(changes, get().nodes),
    });
  },

  savePosition: async (nodeId: string, x: number, y: number) => {
    try {
      await api.patch(`/workspaces/${nodeId}`, { x, y });
    } catch {
      // Non-critical — position save failure doesn't block user
    }
  },
}));
