import type { Node, Edge } from "@xyflow/react";
import type { WSMessage } from "./socket";
import type { WorkspaceNodeData } from "./canvas";
import { extractResponseText } from "@/components/tabs/chat/message-parser";

/**
 * Standalone event handler extracted from the canvas store.
 * Applies a single WebSocket event to the current node/edge state.
 */
export function handleCanvasEvent(
  msg: WSMessage,
  get: () => {
    nodes: Node<WorkspaceNodeData>[];
    edges: Edge[];
    selectedNodeId: string | null;
    agentMessages: Record<string, Array<{ id: string; content: string; timestamp: string }>>;
  },
  set: (partial: Record<string, unknown>) => void,
): void {
  const { nodes, edges, selectedNodeId } = get();

  switch (msg.event) {
    case "WORKSPACE_ONLINE": {
      const existing = nodes.find((n) => n.id === msg.workspace_id);
      if (existing) {
        set({
          nodes: nodes.map((n) =>
            n.id === msg.workspace_id
              ? { ...n, data: { ...n.data, status: "online" } }
              : n
          ),
        });
      }
      break;
    }

    case "WORKSPACE_OFFLINE": {
      set({
        nodes: nodes.map((n) =>
          n.id === msg.workspace_id
            ? { ...n, data: { ...n.data, status: "offline" } }
            : n
        ),
      });
      break;
    }

    case "WORKSPACE_PAUSED": {
      set({
        nodes: nodes.map((n) =>
          n.id === msg.workspace_id
            ? { ...n, data: { ...n.data, status: "paused", currentTask: "" } }
            : n
        ),
      });
      break;
    }

    case "WORKSPACE_DEGRADED": {
      set({
        nodes: nodes.map((n) =>
          n.id === msg.workspace_id
            ? {
                ...n,
                data: {
                  ...n.data,
                  status: "degraded",
                  lastErrorRate: (msg.payload.error_rate as number) ?? 0,
                  lastSampleError:
                    (msg.payload.sample_error as string) ?? "",
                },
              }
            : n
        ),
      });
      break;
    }

    case "WORKSPACE_PROVISIONING": {
      const exists = nodes.find((n) => n.id === msg.workspace_id);
      if (exists) {
        // Restart — update existing node to provisioning
        set({
          nodes: nodes.map((n) =>
            n.id === msg.workspace_id
              ? { ...n, data: { ...n.data, status: "provisioning", needsRestart: false, currentTask: "" } }
              : n
          ),
        });
      } else {
        // Spread new nodes in a grid so they don't stack at the viewport origin
        const GRID_COLS = 4;
        const COL_SPACING = 320;
        const ROW_SPACING = 160;
        const GRID_ORIGIN_X = 100;
        const GRID_ORIGIN_Y = 100;
        const idx = nodes.length;
        const x = GRID_ORIGIN_X + (idx % GRID_COLS) * COL_SPACING;
        const y = GRID_ORIGIN_Y + Math.floor(idx / GRID_COLS) * ROW_SPACING;

        set({
          nodes: [
            ...nodes,
            {
              id: msg.workspace_id,
              type: "workspaceNode",
              position: { x, y },
              data: {
                name: (msg.payload.name as string) ?? "New Workspace",
                status: "provisioning",
                tier: (msg.payload.tier as number) ?? 1,
                agentCard: null,
                activeTasks: 0,
                collapsed: false,
                role: "",
                lastErrorRate: 0,
                lastSampleError: "",
                url: "",
                parentId: null,
                currentTask: "",
                needsRestart: false,
              },
            },
          ],
        });

        // Pan the canvas to the new node
        if (typeof window !== "undefined") {
          window.dispatchEvent(
            new CustomEvent("molecule:pan-to-node", {
              detail: { nodeId: msg.workspace_id },
            })
          );
        }
      }
      break;
    }

    case "WORKSPACE_REMOVED": {
      const removedNode = nodes.find((n) => n.id === msg.workspace_id);
      const parentOfRemoved = removedNode?.data.parentId ?? null;
      set({
        nodes: nodes
          .filter((n) => n.id !== msg.workspace_id)
          .map((n) =>
            n.data.parentId === msg.workspace_id
              ? {
                  ...n,
                  hidden: !!parentOfRemoved,
                  data: { ...n.data, parentId: parentOfRemoved },
                }
              : n
          ),
        edges: edges.filter(
          (e) =>
            e.source !== msg.workspace_id && e.target !== msg.workspace_id
        ),
        selectedNodeId: selectedNodeId === msg.workspace_id ? null : selectedNodeId,
      });
      break;
    }

    case "AGENT_CARD_UPDATED": {
      const card = msg.payload.agent_card;
      const agentCard = (typeof card === "object" && card !== null ? card : null) as Record<string, unknown> | null;
      set({
        nodes: nodes.map((n) =>
          n.id === msg.workspace_id
            ? { ...n, data: { ...n.data, agentCard } }
            : n
        ),
      });
      break;
    }

    case "TASK_UPDATED": {
      const currentTask = (msg.payload.current_task as string) ?? "";
      const activeTasks = (msg.payload.active_tasks as number) ?? 0;
      set({
        nodes: nodes.map((n) =>
          n.id === msg.workspace_id
            ? { ...n, data: { ...n.data, currentTask, activeTasks } }
            : n
        ),
      });
      break;
    }

    case "AGENT_MESSAGE": {
      const content = (msg.payload.message as string) ?? "";
      if (content) {
        const { agentMessages } = get();
        const existing = agentMessages[msg.workspace_id] || [];
        set({
          agentMessages: {
            ...agentMessages,
            [msg.workspace_id]: [
              ...existing,
              { id: crypto.randomUUID(), content, timestamp: new Date().toISOString() },
            ],
          },
        });
      }
      break;
    }

    case "WORKSPACE_PROVISION_FAILED": {
      const errorMsg = (msg.payload.error as string) ?? "Unknown provisioning error";
      set({
        nodes: nodes.map((n) =>
          n.id === msg.workspace_id
            ? {
                ...n,
                data: {
                  ...n.data,
                  status: "failed",
                  lastSampleError: errorMsg,
                },
              }
            : n
        ),
      });
      break;
    }

    case "A2A_RESPONSE": {
      // A2A proxy completed — extract response text and store as agent message.
      // This gives the ChatTab instant response delivery via WebSocket instead of polling.
      const responseBody = msg.payload.response_body as Record<string, unknown> | undefined;
      if (responseBody) {
        const text = extractResponseText(responseBody);
        if (text) {
          const { agentMessages } = get();
          const existing = agentMessages[msg.workspace_id] || [];
          set({
            agentMessages: {
              ...agentMessages,
              [msg.workspace_id]: [
                ...existing,
                { id: crypto.randomUUID(), content: text, timestamp: new Date().toISOString() },
              ],
            },
          });
        }
      }
      break;
    }

    default:
      break;
  }
}
