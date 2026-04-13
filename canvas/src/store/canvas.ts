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
import { buildNodesAndEdges } from "./canvas-topology";

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
}

export type PanelTab = "details" | "skills" | "chat" | "terminal" | "config" | "schedule" | "channels" | "files" | "memory" | "traces" | "events" | "activity";

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
  searchOpen: boolean;
  setSearchOpen: (open: boolean) => void;
  viewport: { x: number; y: number; zoom: number };
  setViewport: (v: { x: number; y: number; zoom: number }) => void;
  saveViewport: (x: number, y: number, zoom: number) => void;
  /** Agent-pushed messages keyed by workspace ID. ChatTab consumes and clears these. */
  agentMessages: Record<string, Array<{ id: string; content: string; timestamp: string }>>;
  consumeAgentMessages: (workspaceId: string) => Array<{ id: string; content: string; timestamp: string }>;
}

export const useCanvasStore = create<CanvasState>((set, get) => ({
  nodes: [],
  edges: [],
  selectedNodeId: null,
  panelTab: "chat",
  dragOverNodeId: null,
  contextMenu: null,

  viewport: { x: 0, y: 0, zoom: 1 },

  selectNode: (id) => set({ selectedNodeId: id }),
  openContextMenu: (menu) => set({ contextMenu: menu }),
  closeContextMenu: () => set({ contextMenu: null }),
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
    const { nodes, edges } = buildNodesAndEdges(workspaces);
    set({ nodes, edges });
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
