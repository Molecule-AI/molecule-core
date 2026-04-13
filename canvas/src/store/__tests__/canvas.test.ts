import { describe, it, expect, beforeEach, vi } from "vitest";

// Mock fetch globally before importing the store (api.ts uses fetch)
global.fetch = vi.fn(() =>
  Promise.resolve({ ok: true, json: () => Promise.resolve({}) } as Response)
);

import { useCanvasStore, summarizeWorkspaceCapabilities } from "../canvas";
import type { WorkspaceData, WSMessage } from "../socket";

// Helper to build a WorkspaceData object with sensible defaults
function makeWS(overrides: Partial<WorkspaceData> & { id: string }): WorkspaceData {
  return {
    name: "WS",
    role: "agent",
    tier: 1,
    status: "online",
    agent_card: null,
    url: "http://localhost:9000",
    parent_id: null,
    active_tasks: 0,
    last_error_rate: 0,
    last_sample_error: "",
    uptime_seconds: 60,
    current_task: "",
    x: 0,
    y: 0,
    collapsed: false,
    runtime: "",
    ...overrides,
  };
}

function makeMsg(overrides: Partial<WSMessage> & { event: string; workspace_id: string }): WSMessage {
  return {
    timestamp: new Date().toISOString(),
    payload: {},
    ...overrides,
  };
}

beforeEach(() => {
  // Reset to initial state before each test
  useCanvasStore.setState({
    nodes: [],
    edges: [],
    selectedNodeId: null,
    panelTab: "details",
    dragOverNodeId: null,
    contextMenu: null,
    searchOpen: false,
    viewport: { x: 0, y: 0, zoom: 1 },
  });
  vi.clearAllMocks();
});

// ---------- selectNode ----------

describe("selectNode", () => {
  it("sets selectedNodeId", () => {
    useCanvasStore.getState().selectNode("ws-1");
    expect(useCanvasStore.getState().selectedNodeId).toBe("ws-1");
  });

  it("deselects when passed null", () => {
    useCanvasStore.getState().selectNode("ws-1");
    useCanvasStore.getState().selectNode(null);
    expect(useCanvasStore.getState().selectedNodeId).toBeNull();
  });
});

// ---------- hydrate ----------

describe("hydrate", () => {
  it("converts WorkspaceData[] to nodes", () => {
    const workspaces = [
      makeWS({ id: "a", name: "Alpha", x: 10, y: 20 }),
      makeWS({ id: "b", name: "Beta", x: 30, y: 40 }),
    ];

    useCanvasStore.getState().hydrate(workspaces);
    const { nodes, edges } = useCanvasStore.getState();

    expect(nodes).toHaveLength(2);
    expect(nodes[0].id).toBe("a");
    expect(nodes[0].data.name).toBe("Alpha");
    expect(nodes[0].position).toEqual({ x: 10, y: 20 });
    expect(nodes[0].type).toBe("workspaceNode");
    expect(nodes[1].id).toBe("b");
    // No parent-child edges
    expect(edges).toHaveLength(0);
  });

  it("sets hidden=true for nodes with parent_id", () => {
    const workspaces = [
      makeWS({ id: "parent", name: "Parent" }),
      makeWS({ id: "child", name: "Child", parent_id: "parent" }),
    ];

    useCanvasStore.getState().hydrate(workspaces);
    const { nodes } = useCanvasStore.getState();

    const parent = nodes.find((n) => n.id === "parent")!;
    const child = nodes.find((n) => n.id === "child")!;

    expect(parent.hidden).toBeFalsy();
    expect(child.hidden).toBe(true);
    expect(child.data.parentId).toBe("parent");
  });

  it("maps all WorkspaceData fields into node data", () => {
    const ws = makeWS({
      id: "x",
      name: "Test",
      role: "lead",
      tier: 2,
      status: "degraded",
      agent_card: { skills: ["code"] },
      url: "http://test:9000",
      active_tasks: 3,
      last_error_rate: 0.75,
      last_sample_error: "timeout",
      collapsed: true,
    });

    useCanvasStore.getState().hydrate([ws]);
    const data = useCanvasStore.getState().nodes[0].data;

    expect(data.name).toBe("Test");
    expect(data.role).toBe("lead");
    expect(data.tier).toBe(2);
    expect(data.status).toBe("degraded");
    expect(data.agentCard).toEqual({ skills: ["code"] });
    expect(data.url).toBe("http://test:9000");
    expect(data.activeTasks).toBe(3);
    expect(data.lastErrorRate).toBe(0.75);
    expect(data.lastSampleError).toBe("timeout");
    expect(data.collapsed).toBe(true);
  });

  it("maps current_task into currentTask", () => {
    const ws = makeWS({ id: "x", current_task: "Processing request" });
    useCanvasStore.getState().hydrate([ws]);
    expect(useCanvasStore.getState().nodes[0].data.currentTask).toBe("Processing request");
  });

  it("defaults currentTask to empty string when missing", () => {
    const ws = makeWS({ id: "x" });
    // current_task is "" from makeWS default
    useCanvasStore.getState().hydrate([ws]);
    expect(useCanvasStore.getState().nodes[0].data.currentTask).toBe("");
  });
});

describe("summarizeWorkspaceCapabilities", () => {
  it("derives runtime, skills, and resume state from node data", () => {
    const summary = summarizeWorkspaceCapabilities({
      name: "Echo",
      status: "online",
      tier: 2,
      agentCard: {
        runtime: "claude-code",
        skills: [{ id: "write", name: "Writing" }, { id: "plan" }],
      },
      activeTasks: 1,
      collapsed: false,
      role: "agent",
      lastErrorRate: 0,
      lastSampleError: "",
      url: "http://localhost:9000",
      parentId: null,
      currentTask: "Reviewing docs",
      needsRestart: false,
      runtime: "claude-code",
    });

    expect(summary.runtime).toBe("claude-code");
    expect(summary.skills).toEqual(["Writing", "plan"]);
    expect(summary.skillCount).toBe(2);
    expect(summary.currentTask).toBe("Reviewing docs");
    expect(summary.hasActiveTask).toBe(true);
  });

  it("handles missing agent card and whitespace-only task", () => {
    const summary = summarizeWorkspaceCapabilities({
      name: "Echo",
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
      currentTask: "   ",
      needsRestart: false,
      runtime: "",
    });

    expect(summary.runtime).toBeNull();
    expect(summary.skills).toEqual([]);
    expect(summary.skillCount).toBe(0);
    expect(summary.currentTask).toBe("");
    expect(summary.hasActiveTask).toBe(false);
  });
});

// ---------- applyEvent ----------

describe("applyEvent", () => {
  beforeEach(() => {
    useCanvasStore.getState().hydrate([
      makeWS({ id: "ws-1", name: "One", status: "online" }),
      makeWS({ id: "ws-2", name: "Two", status: "online", parent_id: "ws-1" }),
    ]);
  });

  it("WORKSPACE_ONLINE sets status to online", () => {
    // First set it to something else
    useCanvasStore.getState().updateNodeData("ws-1", { status: "offline" });

    useCanvasStore.getState().applyEvent(
      makeMsg({ event: "WORKSPACE_ONLINE", workspace_id: "ws-1" })
    );

    const node = useCanvasStore.getState().nodes.find((n) => n.id === "ws-1")!;
    expect(node.data.status).toBe("online");
  });

  it("WORKSPACE_ONLINE is a no-op for unknown workspace", () => {
    const before = useCanvasStore.getState().nodes.length;
    useCanvasStore.getState().applyEvent(
      makeMsg({ event: "WORKSPACE_ONLINE", workspace_id: "unknown" })
    );
    expect(useCanvasStore.getState().nodes.length).toBe(before);
  });

  it("WORKSPACE_OFFLINE sets status to offline", () => {
    useCanvasStore.getState().applyEvent(
      makeMsg({ event: "WORKSPACE_OFFLINE", workspace_id: "ws-1" })
    );

    const node = useCanvasStore.getState().nodes.find((n) => n.id === "ws-1")!;
    expect(node.data.status).toBe("offline");
  });

  it("WORKSPACE_DEGRADED sets status and error fields", () => {
    useCanvasStore.getState().applyEvent(
      makeMsg({
        event: "WORKSPACE_DEGRADED",
        workspace_id: "ws-1",
        payload: { error_rate: 0.8, sample_error: "connection refused" },
      })
    );

    const node = useCanvasStore.getState().nodes.find((n) => n.id === "ws-1")!;
    expect(node.data.status).toBe("degraded");
    expect(node.data.lastErrorRate).toBe(0.8);
    expect(node.data.lastSampleError).toBe("connection refused");
  });

  it("WORKSPACE_PROVISIONING creates a new node", () => {
    useCanvasStore.getState().applyEvent(
      makeMsg({
        event: "WORKSPACE_PROVISIONING",
        workspace_id: "ws-new",
        payload: { name: "Fresh", tier: 2 },
      })
    );

    const { nodes } = useCanvasStore.getState();
    expect(nodes).toHaveLength(3);

    const newNode = nodes.find((n) => n.id === "ws-new")!;
    expect(newNode).toBeDefined();
    expect(newNode.data.name).toBe("Fresh");
    expect(newNode.data.tier).toBe(2);
    expect(newNode.data.status).toBe("provisioning");
    // Position is offset by existing node count * 40
    expect(newNode.position.x).toBeGreaterThanOrEqual(0);
    expect(newNode.position.y).toBeGreaterThanOrEqual(0);
  });

  it("WORKSPACE_PROVISIONING updates existing node status on restart", () => {
    // ws-1 exists as "online" — a restart should set it to "provisioning"
    useCanvasStore.getState().applyEvent(
      makeMsg({
        event: "WORKSPACE_PROVISIONING",
        workspace_id: "ws-1",
        payload: { name: "PM" },
      })
    );

    const { nodes } = useCanvasStore.getState();
    expect(nodes).toHaveLength(2); // no duplication
    const node = nodes.find((n) => n.id === "ws-1")!;
    expect(node.data.status).toBe("provisioning");
    expect(node.data.needsRestart).toBe(false);
    expect(node.data.currentTask).toBe("");
  });

  it("WORKSPACE_PROVISIONING uses defaults when payload is sparse", () => {
    useCanvasStore.getState().applyEvent(
      makeMsg({
        event: "WORKSPACE_PROVISIONING",
        workspace_id: "ws-default",
        payload: {},
      })
    );

    const node = useCanvasStore.getState().nodes.find((n) => n.id === "ws-default")!;
    expect(node.data.name).toBe("New Workspace");
    expect(node.data.tier).toBe(1);
  });

  it("WORKSPACE_REMOVED removes node and reparents children", () => {
    // ws-2 is a child of ws-1. Removing ws-1 should reparent ws-2 to null (root)
    useCanvasStore.getState().applyEvent(
      makeMsg({ event: "WORKSPACE_REMOVED", workspace_id: "ws-1" })
    );

    const { nodes } = useCanvasStore.getState();
    expect(nodes).toHaveLength(1);
    expect(nodes[0].id).toBe("ws-2");
    expect(nodes[0].data.parentId).toBeNull();
    expect(nodes[0].hidden).toBe(false);
  });

  it("WORKSPACE_REMOVED clears selectedNodeId if removed", () => {
    useCanvasStore.getState().selectNode("ws-1");
    useCanvasStore.getState().applyEvent(
      makeMsg({ event: "WORKSPACE_REMOVED", workspace_id: "ws-1" })
    );
    expect(useCanvasStore.getState().selectedNodeId).toBeNull();
  });

  it("WORKSPACE_REMOVED keeps selectedNodeId if different node removed", () => {
    useCanvasStore.getState().selectNode("ws-1");
    useCanvasStore.getState().applyEvent(
      makeMsg({ event: "WORKSPACE_REMOVED", workspace_id: "ws-2" })
    );
    expect(useCanvasStore.getState().selectedNodeId).toBe("ws-1");
  });

  it("AGENT_CARD_UPDATED sets agentCard", () => {
    const card = { name: "Echo Agent", skills: [{ id: "echo" }] };
    useCanvasStore.getState().applyEvent(
      makeMsg({
        event: "AGENT_CARD_UPDATED",
        workspace_id: "ws-1",
        payload: { agent_card: card },
      })
    );

    const node = useCanvasStore.getState().nodes.find((n) => n.id === "ws-1")!;
    expect(node.data.agentCard).toEqual(card);
  });

  it("AGENT_CARD_UPDATED sets null for non-object card", () => {
    useCanvasStore.getState().applyEvent(
      makeMsg({
        event: "AGENT_CARD_UPDATED",
        workspace_id: "ws-1",
        payload: { agent_card: "invalid" },
      })
    );

    const node = useCanvasStore.getState().nodes.find((n) => n.id === "ws-1")!;
    expect(node.data.agentCard).toBeNull();
  });

  it("TASK_UPDATED sets currentTask and activeTasks", () => {
    useCanvasStore.getState().applyEvent(
      makeMsg({
        event: "TASK_UPDATED",
        workspace_id: "ws-1",
        payload: { current_task: "Analyzing data", active_tasks: 2 },
      })
    );

    const node = useCanvasStore.getState().nodes.find((n) => n.id === "ws-1")!;
    expect(node.data.currentTask).toBe("Analyzing data");
    expect(node.data.activeTasks).toBe(2);
  });

  it("TASK_UPDATED clears currentTask when empty", () => {
    // First set a task
    useCanvasStore.getState().updateNodeData("ws-1", { currentTask: "Working" });

    useCanvasStore.getState().applyEvent(
      makeMsg({
        event: "TASK_UPDATED",
        workspace_id: "ws-1",
        payload: { current_task: "", active_tasks: 0 },
      })
    );

    const node = useCanvasStore.getState().nodes.find((n) => n.id === "ws-1")!;
    expect(node.data.currentTask).toBe("");
    expect(node.data.activeTasks).toBe(0);
  });

  it("TASK_UPDATED is a no-op for unknown workspace", () => {
    const nodesBefore = [...useCanvasStore.getState().nodes];
    useCanvasStore.getState().applyEvent(
      makeMsg({
        event: "TASK_UPDATED",
        workspace_id: "unknown",
        payload: { current_task: "task", active_tasks: 1 },
      })
    );
    // Nodes unchanged (same length, same data for ws-1)
    const node = useCanvasStore.getState().nodes.find((n) => n.id === "ws-1")!;
    expect(node.data.currentTask).toBe("");
  });

  it("unknown event is a no-op", () => {
    const nodesBefore = useCanvasStore.getState().nodes;
    useCanvasStore.getState().applyEvent(
      makeMsg({ event: "UNKNOWN_EVENT", workspace_id: "ws-1" })
    );
    expect(useCanvasStore.getState().nodes).toEqual(nodesBefore);
  });
});

// ---------- removeNode ----------

describe("removeNode", () => {
  beforeEach(() => {
    useCanvasStore.getState().hydrate([
      makeWS({ id: "root" }),
      makeWS({ id: "mid", parent_id: "root" }),
      makeWS({ id: "leaf", parent_id: "mid" }),
    ]);
  });

  it("removes the node from the list", () => {
    useCanvasStore.getState().removeNode("leaf");
    const ids = useCanvasStore.getState().nodes.map((n) => n.id);
    expect(ids).toEqual(["root", "mid"]);
  });

  it("reparents children to deleted node's parent", () => {
    // Removing mid: leaf should be reparented to root
    useCanvasStore.getState().removeNode("mid");

    const leaf = useCanvasStore.getState().nodes.find((n) => n.id === "leaf")!;
    expect(leaf.data.parentId).toBe("root");
    expect(leaf.hidden).toBe(true); // still has a parent
  });

  it("reparents children to null when root is deleted", () => {
    useCanvasStore.getState().removeNode("root");

    const mid = useCanvasStore.getState().nodes.find((n) => n.id === "mid")!;
    expect(mid.data.parentId).toBeNull();
    expect(mid.hidden).toBe(false);
  });

  it("clears selection if removed node was selected", () => {
    useCanvasStore.getState().selectNode("mid");
    useCanvasStore.getState().removeNode("mid");
    expect(useCanvasStore.getState().selectedNodeId).toBeNull();
  });

  it("preserves selection if a different node is removed", () => {
    useCanvasStore.getState().selectNode("root");
    useCanvasStore.getState().removeNode("leaf");
    expect(useCanvasStore.getState().selectedNodeId).toBe("root");
  });
});

// ---------- isDescendant ----------

describe("isDescendant", () => {
  beforeEach(() => {
    useCanvasStore.getState().hydrate([
      makeWS({ id: "a" }),
      makeWS({ id: "b", parent_id: "a" }),
      makeWS({ id: "c", parent_id: "b" }),
      makeWS({ id: "d" }), // unrelated root
    ]);
  });

  it("returns true for direct child", () => {
    expect(useCanvasStore.getState().isDescendant("a", "b")).toBe(true);
  });

  it("returns true for grandchild", () => {
    expect(useCanvasStore.getState().isDescendant("a", "c")).toBe(true);
  });

  it("returns false for ancestor (reverse direction)", () => {
    expect(useCanvasStore.getState().isDescendant("c", "a")).toBe(false);
  });

  it("returns false for unrelated nodes", () => {
    expect(useCanvasStore.getState().isDescendant("a", "d")).toBe(false);
  });

  it("returns false for same node", () => {
    expect(useCanvasStore.getState().isDescendant("a", "a")).toBe(false);
  });

  it("returns false for non-existent nodeId", () => {
    expect(useCanvasStore.getState().isDescendant("a", "nope")).toBe(false);
  });
});

// ---------- updateNodeData ----------

describe("updateNodeData", () => {
  beforeEach(() => {
    useCanvasStore.getState().hydrate([makeWS({ id: "ws-1", name: "Old" })]);
  });

  it("merges partial data into the node", () => {
    useCanvasStore.getState().updateNodeData("ws-1", { name: "New", tier: 3 });
    const data = useCanvasStore.getState().nodes[0].data;
    expect(data.name).toBe("New");
    expect(data.tier).toBe(3);
    // Unaffected fields preserved
    expect(data.status).toBe("online");
  });

  it("is a no-op for unknown id (no crash)", () => {
    useCanvasStore.getState().updateNodeData("nope", { name: "X" });
    expect(useCanvasStore.getState().nodes).toHaveLength(1);
    expect(useCanvasStore.getState().nodes[0].data.name).toBe("Old");
  });
});

// ---------- openContextMenu / closeContextMenu ----------

describe("context menu", () => {
  const menu = {
    x: 100,
    y: 200,
    nodeId: "ws-1",
    nodeData: {
      name: "Test",
      status: "online",
      tier: 1,
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
      runtime: "",
    },
  };

  it("openContextMenu sets state", () => {
    useCanvasStore.getState().openContextMenu(menu);
    expect(useCanvasStore.getState().contextMenu).toEqual(menu);
  });

  it("closeContextMenu clears state", () => {
    useCanvasStore.getState().openContextMenu(menu);
    useCanvasStore.getState().closeContextMenu();
    expect(useCanvasStore.getState().contextMenu).toBeNull();
  });
});

// ---------- setPanelTab ----------

describe("setPanelTab", () => {
  it("sets the active panel tab", () => {
    useCanvasStore.getState().setPanelTab("chat");
    expect(useCanvasStore.getState().panelTab).toBe("chat");
  });

  it("can switch between tabs", () => {
    useCanvasStore.getState().setPanelTab("terminal");
    useCanvasStore.getState().setPanelTab("config");
    expect(useCanvasStore.getState().panelTab).toBe("config");
  });

  it("can switch to skills tab", () => {
    useCanvasStore.getState().setPanelTab("skills");
    expect(useCanvasStore.getState().panelTab).toBe("skills");
  });
});

// ---------- getSelectedNode ----------

describe("getSelectedNode", () => {
  beforeEach(() => {
    useCanvasStore.getState().hydrate([makeWS({ id: "ws-1", name: "Alpha" })]);
  });

  it("returns null when nothing selected", () => {
    expect(useCanvasStore.getState().getSelectedNode()).toBeNull();
  });

  it("returns the selected node", () => {
    useCanvasStore.getState().selectNode("ws-1");
    const node = useCanvasStore.getState().getSelectedNode();
    expect(node).not.toBeNull();
    expect(node!.data.name).toBe("Alpha");
  });

  it("returns null when selected id does not match any node", () => {
    useCanvasStore.getState().selectNode("nonexistent");
    expect(useCanvasStore.getState().getSelectedNode()).toBeNull();
  });
});

// ---------- savePosition ----------

describe("savePosition", () => {
  it("calls API to persist position", async () => {
    await useCanvasStore.getState().savePosition("ws-1", 42, 99);
    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/workspaces/ws-1"),
      expect.objectContaining({ method: "PATCH" })
    );
  });
});

// ---------- saveViewport ----------

describe("saveViewport", () => {
  it("updates local viewport and calls API", async () => {
    await useCanvasStore.getState().saveViewport(10, 20, 1.5);
    expect(useCanvasStore.getState().viewport).toEqual({ x: 10, y: 20, zoom: 1.5 });
    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/canvas/viewport"),
      expect.objectContaining({ method: "PUT" })
    );
  });
});

// ---------- nestNode ----------

describe("nestNode", () => {
  beforeEach(() => {
    useCanvasStore.getState().hydrate([
      makeWS({ id: "a", name: "A" }),
      makeWS({ id: "b", name: "B" }),
    ]);
  });

  it("optimistically updates parentId and hidden", async () => {
    await useCanvasStore.getState().nestNode("b", "a");

    const b = useCanvasStore.getState().nodes.find((n) => n.id === "b")!;
    expect(b.data.parentId).toBe("a");
    expect(b.hidden).toBe(true);
  });

  it("un-nesting sets parentId to null and shows node", async () => {
    // First nest
    await useCanvasStore.getState().nestNode("b", "a");
    // Then un-nest
    await useCanvasStore.getState().nestNode("b", null);

    const b = useCanvasStore.getState().nodes.find((n) => n.id === "b")!;
    expect(b.data.parentId).toBeNull();
    expect(b.hidden).toBe(false);
  });

  it("skips when parentId is already the target", async () => {
    await useCanvasStore.getState().nestNode("b", "a");
    vi.clearAllMocks();
    await useCanvasStore.getState().nestNode("b", "a");
    // No API call since no change
    expect(global.fetch).not.toHaveBeenCalled();
  });

  it("reverts on API failure", async () => {
    (global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
      ok: false,
      status: 500,
      text: () => Promise.resolve("internal error"),
    });

    await useCanvasStore.getState().nestNode("b", "a");

    // Should revert to original state (no parent)
    const b = useCanvasStore.getState().nodes.find((n) => n.id === "b")!;
    expect(b.data.parentId).toBeNull();
    expect(b.hidden).toBe(false);
  });
});

// ---------- misc state setters ----------

describe("misc state setters", () => {
  it("setDragOverNode", () => {
    useCanvasStore.getState().setDragOverNode("ws-1");
    expect(useCanvasStore.getState().dragOverNodeId).toBe("ws-1");
    useCanvasStore.getState().setDragOverNode(null);
    expect(useCanvasStore.getState().dragOverNodeId).toBeNull();
  });

  it("setSearchOpen", () => {
    useCanvasStore.getState().setSearchOpen(true);
    expect(useCanvasStore.getState().searchOpen).toBe(true);
    useCanvasStore.getState().setSearchOpen(false);
    expect(useCanvasStore.getState().searchOpen).toBe(false);
  });

  it("setViewport", () => {
    useCanvasStore.getState().setViewport({ x: 5, y: 10, zoom: 2 });
    expect(useCanvasStore.getState().viewport).toEqual({ x: 5, y: 10, zoom: 2 });
  });

  it("setPanelTab to activity", () => {
    useCanvasStore.getState().setPanelTab("activity");
    expect(useCanvasStore.getState().panelTab).toBe("activity");
  });
});

// ---------- ACTIVITY_LOGGED event ----------

describe("ACTIVITY_LOGGED event", () => {
  beforeEach(() => {
    useCanvasStore.getState().hydrate([
      makeWS({ id: "ws-1", name: "Agent" }),
    ]);
  });

  it("does not crash the store (no-op)", () => {
    // ACTIVITY_LOGGED is handled by ActivityTab polling, not the store
    useCanvasStore.getState().applyEvent(
      makeMsg({
        event: "ACTIVITY_LOGGED",
        workspace_id: "ws-1",
        payload: { activity_type: "a2a_receive", method: "message/send" },
      })
    );

    // Store unchanged
    const node = useCanvasStore.getState().nodes.find((n) => n.id === "ws-1")!;
    expect(node.data.status).toBe("online");
    expect(node.data.name).toBe("Agent");
  });
});

// ---------- TASK_UPDATED edge cases ----------

describe("TASK_UPDATED edge cases", () => {
  beforeEach(() => {
    useCanvasStore.getState().hydrate([
      makeWS({ id: "ws-1", name: "Agent", current_task: "Initial task" }),
    ]);
  });

  it("handles missing current_task in payload (defaults to empty)", () => {
    useCanvasStore.getState().applyEvent(
      makeMsg({
        event: "TASK_UPDATED",
        workspace_id: "ws-1",
        payload: { active_tasks: 0 },
      })
    );

    const node = useCanvasStore.getState().nodes.find((n) => n.id === "ws-1")!;
    expect(node.data.currentTask).toBe("");
    expect(node.data.activeTasks).toBe(0);
  });

  it("handles missing active_tasks in payload (defaults to 0)", () => {
    useCanvasStore.getState().applyEvent(
      makeMsg({
        event: "TASK_UPDATED",
        workspace_id: "ws-1",
        payload: { current_task: "New task" },
      })
    );

    const node = useCanvasStore.getState().nodes.find((n) => n.id === "ws-1")!;
    expect(node.data.currentTask).toBe("New task");
    expect(node.data.activeTasks).toBe(0);
  });

  it("preserves other node data when task changes", () => {
    useCanvasStore.getState().applyEvent(
      makeMsg({
        event: "TASK_UPDATED",
        workspace_id: "ws-1",
        payload: { current_task: "New task", active_tasks: 3 },
      })
    );

    const node = useCanvasStore.getState().nodes.find((n) => n.id === "ws-1")!;
    expect(node.data.name).toBe("Agent");
    expect(node.data.status).toBe("online");
    expect(node.data.currentTask).toBe("New task");
  });

  it("does not affect other nodes when task updates", () => {
    useCanvasStore.getState().hydrate([
      makeWS({ id: "ws-1", name: "A", current_task: "Task A" }),
      makeWS({ id: "ws-2", name: "B", current_task: "Task B" }),
    ]);

    useCanvasStore.getState().applyEvent(
      makeMsg({
        event: "TASK_UPDATED",
        workspace_id: "ws-1",
        payload: { current_task: "Updated A", active_tasks: 1 },
      })
    );

    const ws1 = useCanvasStore.getState().nodes.find((n) => n.id === "ws-1")!;
    const ws2 = useCanvasStore.getState().nodes.find((n) => n.id === "ws-2")!;
    expect(ws1.data.currentTask).toBe("Updated A");
    expect(ws2.data.currentTask).toBe("Task B"); // unchanged
  });
});
