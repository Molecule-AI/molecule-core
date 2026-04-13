import { describe, it, expect, beforeEach, vi } from "vitest";
import { handleCanvasEvent } from "../canvas-events";
import type { WSMessage } from "../socket";
import type { WorkspaceNodeData } from "../canvas";
import type { Node, Edge } from "@xyflow/react";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function makeNode(
  id: string,
  overrides: Partial<WorkspaceNodeData> = {}
): Node<WorkspaceNodeData> {
  return {
    id,
    type: "workspaceNode",
    position: { x: 0, y: 0 },
    data: {
      name: `Node-${id}`,
      status: "online",
      tier: 1,
      agentCard: null,
      activeTasks: 0,
      collapsed: false,
      role: "agent",
      lastErrorRate: 0,
      lastSampleError: "",
      url: "http://localhost:9000",
      parentId: null,
      currentTask: "",
      needsRestart: false,
      runtime: "",
      ...overrides,
    },
  };
}

function makeMsg(
  overrides: Partial<WSMessage> & { event: string; workspace_id: string }
): WSMessage {
  return {
    timestamp: new Date().toISOString(),
    payload: {},
    ...overrides,
  };
}

// Build a fresh get/set pair each test
function makeStore(
  nodes: Node<WorkspaceNodeData>[] = [],
  edges: Edge[] = [],
  selectedNodeId: string | null = null,
  agentMessages: Record<string, Array<{ id: string; content: string; timestamp: string }>> = {}
) {
  const state = { nodes, edges, selectedNodeId, agentMessages };
  const get = () => state;
  const set = vi.fn((partial: Record<string, unknown>) => {
    Object.assign(state, partial);
  });
  return { state, get, set };
}

// ---------------------------------------------------------------------------
// WORKSPACE_ONLINE
// ---------------------------------------------------------------------------

describe("handleCanvasEvent – WORKSPACE_ONLINE", () => {
  it("sets status to 'online' for a matching node", () => {
    const node = makeNode("ws-1", { status: "offline" });
    const { state, get, set } = makeStore([node]);

    handleCanvasEvent(makeMsg({ event: "WORKSPACE_ONLINE", workspace_id: "ws-1" }), get, set);

    expect(set).toHaveBeenCalledOnce();
    const updated = (set.mock.calls[0][0] as { nodes: Node<WorkspaceNodeData>[] }).nodes;
    expect(updated.find((n) => n.id === "ws-1")!.data.status).toBe("online");
  });

  it("is a no-op when workspace_id does not match any node", () => {
    const node = makeNode("ws-1");
    const { get, set } = makeStore([node]);

    handleCanvasEvent(makeMsg({ event: "WORKSPACE_ONLINE", workspace_id: "unknown" }), get, set);

    expect(set).not.toHaveBeenCalled();
  });

  it("does not mutate other nodes", () => {
    const nodes = [makeNode("ws-1", { status: "offline" }), makeNode("ws-2", { status: "offline" })];
    const { get, set } = makeStore(nodes);

    handleCanvasEvent(makeMsg({ event: "WORKSPACE_ONLINE", workspace_id: "ws-1" }), get, set);

    const updated = (set.mock.calls[0][0] as { nodes: Node<WorkspaceNodeData>[] }).nodes;
    expect(updated.find((n) => n.id === "ws-2")!.data.status).toBe("offline");
  });
});

// ---------------------------------------------------------------------------
// WORKSPACE_OFFLINE
// ---------------------------------------------------------------------------

describe("handleCanvasEvent – WORKSPACE_OFFLINE", () => {
  it("sets status to 'offline' for a matching node", () => {
    const node = makeNode("ws-1", { status: "online" });
    const { get, set } = makeStore([node]);

    handleCanvasEvent(makeMsg({ event: "WORKSPACE_OFFLINE", workspace_id: "ws-1" }), get, set);

    expect(set).toHaveBeenCalledOnce();
    const updated = (set.mock.calls[0][0] as { nodes: Node<WorkspaceNodeData>[] }).nodes;
    expect(updated.find((n) => n.id === "ws-1")!.data.status).toBe("offline");
  });

  it("still calls set even when workspace_id does not match (maps over all nodes)", () => {
    const node = makeNode("ws-1");
    const { get, set } = makeStore([node]);

    handleCanvasEvent(makeMsg({ event: "WORKSPACE_OFFLINE", workspace_id: "nope" }), get, set);

    // set is called because it maps over all nodes (no early-exit guard)
    expect(set).toHaveBeenCalledOnce();
    const updated = (set.mock.calls[0][0] as { nodes: Node<WorkspaceNodeData>[] }).nodes;
    expect(updated[0].data.status).toBe("online"); // unchanged
  });
});

// ---------------------------------------------------------------------------
// WORKSPACE_DEGRADED
// ---------------------------------------------------------------------------

describe("handleCanvasEvent – WORKSPACE_DEGRADED", () => {
  it("sets status, lastErrorRate, and lastSampleError", () => {
    const node = makeNode("ws-1", { status: "online" });
    const { get, set } = makeStore([node]);

    handleCanvasEvent(
      makeMsg({
        event: "WORKSPACE_DEGRADED",
        workspace_id: "ws-1",
        payload: { error_rate: 0.42, sample_error: "timeout connecting to DB" },
      }),
      get,
      set
    );

    const updated = (set.mock.calls[0][0] as { nodes: Node<WorkspaceNodeData>[] }).nodes;
    const data = updated.find((n) => n.id === "ws-1")!.data;
    expect(data.status).toBe("degraded");
    expect(data.lastErrorRate).toBe(0.42);
    expect(data.lastSampleError).toBe("timeout connecting to DB");
  });

  it("defaults error_rate to 0 and sample_error to '' when missing from payload", () => {
    const node = makeNode("ws-1");
    const { get, set } = makeStore([node]);

    handleCanvasEvent(
      makeMsg({ event: "WORKSPACE_DEGRADED", workspace_id: "ws-1" }),
      get,
      set
    );

    const updated = (set.mock.calls[0][0] as { nodes: Node<WorkspaceNodeData>[] }).nodes;
    const data = updated.find((n) => n.id === "ws-1")!.data;
    expect(data.lastErrorRate).toBe(0);
    expect(data.lastSampleError).toBe("");
  });
});

// ---------------------------------------------------------------------------
// WORKSPACE_PROVISIONING
// ---------------------------------------------------------------------------

describe("handleCanvasEvent – WORKSPACE_PROVISIONING", () => {
  it("creates a new node when workspace_id is unknown", () => {
    const { get, set } = makeStore([]);

    handleCanvasEvent(
      makeMsg({
        event: "WORKSPACE_PROVISIONING",
        workspace_id: "ws-new",
        payload: { name: "Brand New", tier: 3 },
      }),
      get,
      set
    );

    const newNodes = (set.mock.calls[0][0] as { nodes: Node<WorkspaceNodeData>[] }).nodes;
    expect(newNodes).toHaveLength(1);
    const n = newNodes[0];
    expect(n.id).toBe("ws-new");
    expect(n.type).toBe("workspaceNode");
    expect(n.position).toEqual({ x: 0, y: 0 });
    expect(n.data.name).toBe("Brand New");
    expect(n.data.tier).toBe(3);
    expect(n.data.status).toBe("provisioning");
  });

  it("uses defaults for name and tier when payload is sparse", () => {
    const { get, set } = makeStore([]);

    handleCanvasEvent(
      makeMsg({ event: "WORKSPACE_PROVISIONING", workspace_id: "ws-x", payload: {} }),
      get,
      set
    );

    const newNodes = (set.mock.calls[0][0] as { nodes: Node<WorkspaceNodeData>[] }).nodes;
    expect(newNodes[0].data.name).toBe("New Workspace");
    expect(newNodes[0].data.tier).toBe(1);
  });

  it("updates an existing node to provisioning (restart path)", () => {
    const node = makeNode("ws-1", { status: "online", currentTask: "old task", needsRestart: true });
    const { get, set } = makeStore([node]);

    handleCanvasEvent(
      makeMsg({
        event: "WORKSPACE_PROVISIONING",
        workspace_id: "ws-1",
        payload: { name: "PM" },
      }),
      get,
      set
    );

    const updated = (set.mock.calls[0][0] as { nodes: Node<WorkspaceNodeData>[] }).nodes;
    // Must not create a duplicate node
    expect(updated).toHaveLength(1);
    const data = updated[0].data;
    expect(data.status).toBe("provisioning");
    expect(data.needsRestart).toBe(false);
    expect(data.currentTask).toBe("");
  });
});

// ---------------------------------------------------------------------------
// WORKSPACE_REMOVED
// ---------------------------------------------------------------------------

describe("handleCanvasEvent – WORKSPACE_REMOVED", () => {
  it("removes the node from the list", () => {
    const nodes = [makeNode("ws-1"), makeNode("ws-2")];
    const { get, set } = makeStore(nodes);

    handleCanvasEvent(makeMsg({ event: "WORKSPACE_REMOVED", workspace_id: "ws-1" }), get, set);

    const { nodes: updatedNodes } = set.mock.calls[0][0] as { nodes: Node<WorkspaceNodeData>[]; edges: Edge[] };
    expect(updatedNodes.find((n) => n.id === "ws-1")).toBeUndefined();
    expect(updatedNodes.find((n) => n.id === "ws-2")).toBeDefined();
  });

  it("reparents children to the removed node's parent", () => {
    const parent = makeNode("parent");
    const mid = makeNode("mid", { parentId: "parent" });
    const child = makeNode("child", { parentId: "mid" });
    const { get, set } = makeStore([parent, mid, child]);

    // Remove mid — child should be reparented to parent
    handleCanvasEvent(makeMsg({ event: "WORKSPACE_REMOVED", workspace_id: "mid" }), get, set);

    const { nodes: updatedNodes } = set.mock.calls[0][0] as { nodes: Node<WorkspaceNodeData>[] };
    const updatedChild = updatedNodes.find((n) => n.id === "child")!;
    expect(updatedChild.data.parentId).toBe("parent");
    expect(updatedChild.hidden).toBe(true); // still has a parent
  });

  it("reparents children to null when root node is removed", () => {
    const root = makeNode("root");
    const child = makeNode("child", { parentId: "root" });
    const { get, set } = makeStore([root, child]);

    handleCanvasEvent(makeMsg({ event: "WORKSPACE_REMOVED", workspace_id: "root" }), get, set);

    const { nodes: updatedNodes } = set.mock.calls[0][0] as { nodes: Node<WorkspaceNodeData>[] };
    const updatedChild = updatedNodes.find((n) => n.id === "child")!;
    expect(updatedChild.data.parentId).toBeNull();
    expect(updatedChild.hidden).toBe(false);
  });

  it("removes edges connected to the removed workspace", () => {
    const nodes = [makeNode("ws-1"), makeNode("ws-2")];
    const edges: Edge[] = [
      { id: "e1", source: "ws-1", target: "ws-2" },
      { id: "e2", source: "ws-3", target: "ws-1" },
      { id: "e3", source: "ws-2", target: "ws-3" },
    ];
    const { get, set } = makeStore(nodes, edges);

    handleCanvasEvent(makeMsg({ event: "WORKSPACE_REMOVED", workspace_id: "ws-1" }), get, set);

    const { edges: updatedEdges } = set.mock.calls[0][0] as { edges: Edge[] };
    expect(updatedEdges).toHaveLength(1);
    expect(updatedEdges[0].id).toBe("e3");
  });

  it("clears selectedNodeId when the selected node is removed", () => {
    const node = makeNode("ws-1");
    const { get, set } = makeStore([node], [], "ws-1");

    handleCanvasEvent(makeMsg({ event: "WORKSPACE_REMOVED", workspace_id: "ws-1" }), get, set);

    const { selectedNodeId } = set.mock.calls[0][0] as { selectedNodeId: string | null };
    expect(selectedNodeId).toBeNull();
  });

  it("preserves selectedNodeId when a different node is removed", () => {
    const nodes = [makeNode("ws-1"), makeNode("ws-2")];
    const { get, set } = makeStore(nodes, [], "ws-1");

    handleCanvasEvent(makeMsg({ event: "WORKSPACE_REMOVED", workspace_id: "ws-2" }), get, set);

    const { selectedNodeId } = set.mock.calls[0][0] as { selectedNodeId: string | null };
    expect(selectedNodeId).toBe("ws-1");
  });
});

// ---------------------------------------------------------------------------
// AGENT_CARD_UPDATED
// ---------------------------------------------------------------------------

describe("handleCanvasEvent – AGENT_CARD_UPDATED", () => {
  it("sets agentCard from the payload", () => {
    const node = makeNode("ws-1");
    const { get, set } = makeStore([node]);
    const card = { name: "My Agent", skills: [{ id: "code", name: "Coding" }] };

    handleCanvasEvent(
      makeMsg({
        event: "AGENT_CARD_UPDATED",
        workspace_id: "ws-1",
        payload: { agent_card: card },
      }),
      get,
      set
    );

    const updated = (set.mock.calls[0][0] as { nodes: Node<WorkspaceNodeData>[] }).nodes;
    expect(updated.find((n) => n.id === "ws-1")!.data.agentCard).toEqual(card);
  });

  it("sets agentCard to null when payload value is a non-object string", () => {
    const node = makeNode("ws-1");
    const { get, set } = makeStore([node]);

    handleCanvasEvent(
      makeMsg({
        event: "AGENT_CARD_UPDATED",
        workspace_id: "ws-1",
        payload: { agent_card: "bad-value" },
      }),
      get,
      set
    );

    const updated = (set.mock.calls[0][0] as { nodes: Node<WorkspaceNodeData>[] }).nodes;
    expect(updated.find((n) => n.id === "ws-1")!.data.agentCard).toBeNull();
  });

  it("sets agentCard to null when payload value is null", () => {
    const node = makeNode("ws-1", { agentCard: { name: "old" } });
    const { get, set } = makeStore([node]);

    handleCanvasEvent(
      makeMsg({
        event: "AGENT_CARD_UPDATED",
        workspace_id: "ws-1",
        payload: { agent_card: null },
      }),
      get,
      set
    );

    const updated = (set.mock.calls[0][0] as { nodes: Node<WorkspaceNodeData>[] }).nodes;
    expect(updated.find((n) => n.id === "ws-1")!.data.agentCard).toBeNull();
  });
});

// ---------------------------------------------------------------------------
// TASK_UPDATED
// ---------------------------------------------------------------------------

describe("handleCanvasEvent – TASK_UPDATED", () => {
  it("sets currentTask and activeTasks", () => {
    const node = makeNode("ws-1");
    const { get, set } = makeStore([node]);

    handleCanvasEvent(
      makeMsg({
        event: "TASK_UPDATED",
        workspace_id: "ws-1",
        payload: { current_task: "Analysing code", active_tasks: 3 },
      }),
      get,
      set
    );

    const updated = (set.mock.calls[0][0] as { nodes: Node<WorkspaceNodeData>[] }).nodes;
    const data = updated.find((n) => n.id === "ws-1")!.data;
    expect(data.currentTask).toBe("Analysing code");
    expect(data.activeTasks).toBe(3);
  });

  it("defaults to empty string and 0 when payload fields are missing", () => {
    const node = makeNode("ws-1", { currentTask: "old task", activeTasks: 5 });
    const { get, set } = makeStore([node]);

    handleCanvasEvent(
      makeMsg({ event: "TASK_UPDATED", workspace_id: "ws-1", payload: {} }),
      get,
      set
    );

    const updated = (set.mock.calls[0][0] as { nodes: Node<WorkspaceNodeData>[] }).nodes;
    const data = updated.find((n) => n.id === "ws-1")!.data;
    expect(data.currentTask).toBe("");
    expect(data.activeTasks).toBe(0);
  });
});

// ---------------------------------------------------------------------------
// AGENT_MESSAGE
// ---------------------------------------------------------------------------

describe("handleCanvasEvent – AGENT_MESSAGE", () => {
  it("appends a message to agentMessages for the workspace", () => {
    const node = makeNode("ws-1");
    const { get, set } = makeStore([node], [], null, {});

    handleCanvasEvent(
      makeMsg({
        event: "AGENT_MESSAGE",
        workspace_id: "ws-1",
        payload: { message: "Hello from agent!" },
      }),
      get,
      set
    );

    expect(set).toHaveBeenCalledOnce();
    const { agentMessages } = set.mock.calls[0][0] as {
      agentMessages: Record<string, Array<{ id: string; content: string; timestamp: string }>>;
    };
    expect(agentMessages["ws-1"]).toHaveLength(1);
    expect(agentMessages["ws-1"][0].content).toBe("Hello from agent!");
    expect(typeof agentMessages["ws-1"][0].id).toBe("string");
    expect(typeof agentMessages["ws-1"][0].timestamp).toBe("string");
  });

  it("appends to existing messages rather than replacing them", () => {
    const node = makeNode("ws-1");
    const existing = [{ id: "old-id", content: "prior msg", timestamp: "2024-01-01T00:00:00Z" }];
    const { get, set } = makeStore([node], [], null, { "ws-1": existing });

    handleCanvasEvent(
      makeMsg({
        event: "AGENT_MESSAGE",
        workspace_id: "ws-1",
        payload: { message: "second message" },
      }),
      get,
      set
    );

    const { agentMessages } = set.mock.calls[0][0] as {
      agentMessages: Record<string, Array<{ id: string; content: string; timestamp: string }>>;
    };
    expect(agentMessages["ws-1"]).toHaveLength(2);
    expect(agentMessages["ws-1"][0].content).toBe("prior msg");
    expect(agentMessages["ws-1"][1].content).toBe("second message");
  });

  it("is a no-op when message content is empty", () => {
    const node = makeNode("ws-1");
    const { get, set } = makeStore([node]);

    handleCanvasEvent(
      makeMsg({
        event: "AGENT_MESSAGE",
        workspace_id: "ws-1",
        payload: { message: "" },
      }),
      get,
      set
    );

    expect(set).not.toHaveBeenCalled();
  });

  it("is a no-op when message is absent from payload", () => {
    const node = makeNode("ws-1");
    const { get, set } = makeStore([node]);

    handleCanvasEvent(
      makeMsg({ event: "AGENT_MESSAGE", workspace_id: "ws-1", payload: {} }),
      get,
      set
    );

    expect(set).not.toHaveBeenCalled();
  });
});

// ---------------------------------------------------------------------------
// A2A_RESPONSE
// ---------------------------------------------------------------------------

describe("handleCanvasEvent – A2A_RESPONSE", () => {
  it("extracts text from response_body and stores as agentMessage", () => {
    const node = makeNode("ws-1");
    const { get, set } = makeStore([node], [], null, {});

    handleCanvasEvent(
      makeMsg({
        event: "A2A_RESPONSE",
        workspace_id: "ws-1",
        payload: {
          response_body: {
            result: { parts: [{ kind: "text", text: "Here is my analysis" }] },
          },
          method: "message/send",
          duration_ms: 1500,
        },
      }),
      get,
      set
    );

    expect(set).toHaveBeenCalledOnce();
    const { agentMessages } = set.mock.calls[0][0] as {
      agentMessages: Record<string, Array<{ id: string; content: string; timestamp: string }>>;
    };
    expect(agentMessages["ws-1"]).toHaveLength(1);
    expect(agentMessages["ws-1"][0].content).toBe("Here is my analysis");
  });

  it("is a no-op when response_body is missing", () => {
    const node = makeNode("ws-1");
    const { get, set } = makeStore([node]);

    handleCanvasEvent(
      makeMsg({
        event: "A2A_RESPONSE",
        workspace_id: "ws-1",
        payload: { method: "message/send" },
      }),
      get,
      set
    );

    expect(set).not.toHaveBeenCalled();
  });

  it("is a no-op when response text is empty", () => {
    const node = makeNode("ws-1");
    const { get, set } = makeStore([node]);

    handleCanvasEvent(
      makeMsg({
        event: "A2A_RESPONSE",
        workspace_id: "ws-1",
        payload: {
          response_body: { result: { parts: [] } },
        },
      }),
      get,
      set
    );

    expect(set).not.toHaveBeenCalled();
  });
});

// ---------------------------------------------------------------------------
// Unknown event
// ---------------------------------------------------------------------------

describe("handleCanvasEvent – unknown event", () => {
  it("does not crash and does not call set", () => {
    const node = makeNode("ws-1");
    const { get, set } = makeStore([node]);

    expect(() =>
      handleCanvasEvent(
        makeMsg({ event: "TOTALLY_UNKNOWN_EVENT", workspace_id: "ws-1" }),
        get,
        set
      )
    ).not.toThrow();

    expect(set).not.toHaveBeenCalled();
  });

  it("handles an empty event string without crashing", () => {
    const { get, set } = makeStore([]);

    expect(() =>
      handleCanvasEvent(makeMsg({ event: "", workspace_id: "ws-1" }), get, set)
    ).not.toThrow();
  });
});
