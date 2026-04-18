// @vitest-environment jsdom
/**
 * Tests the molecule:pan-to-node CustomEvent dispatch from canvas-events.ts.
 * Runs in jsdom because window.dispatchEvent is required.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { handleCanvasEvent, resetProvisioningSequence } from "../canvas-events";
import type { WSMessage } from "../socket";
import type { WorkspaceNodeData } from "../canvas";
import type { Node, Edge } from "@xyflow/react";

// ── Helpers (copied from canvas-events.test.ts) ──────────────────────────────

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
      budgetLimit: null,
      ...overrides,
    },
  };
}

function makeMsg(
  overrides: Partial<WSMessage> & { event: string; workspace_id: string }
): WSMessage {
  return { timestamp: new Date().toISOString(), payload: {}, ...overrides };
}

function makeStore(
  nodes: Node<WorkspaceNodeData>[] = [],
  edges: Edge[] = []
) {
  const state = { nodes, edges, selectedNodeId: null, agentMessages: {} };
  const get = () => state;
  const set = vi.fn((partial: Record<string, unknown>) => { Object.assign(state, partial); });
  return { state, get, set };
}

// ─────────────────────────────────────────────────────────────────────────────

describe("canvas-events – molecule:pan-to-node dispatch", () => {
  beforeEach(() => {
    resetProvisioningSequence();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("dispatches molecule:pan-to-node with the new nodeId for a NEW provision", () => {
    const { get, set } = makeStore([]);
    const dispatched: Event[] = [];
    const spy = vi.spyOn(window, "dispatchEvent").mockImplementation((e) => {
      dispatched.push(e);
      return true;
    });

    handleCanvasEvent(
      makeMsg({ event: "WORKSPACE_PROVISIONING", workspace_id: "ws-new", payload: {} }),
      get,
      set
    );

    expect(dispatched).toHaveLength(1);
    expect(dispatched[0].type).toBe("molecule:pan-to-node");
    expect((dispatched[0] as CustomEvent).detail?.nodeId).toBe("ws-new");
  });

  it("does NOT dispatch molecule:pan-to-node when restarting an existing node", () => {
    const { get, set } = makeStore([makeNode("ws-existing")]);
    const dispatched: Event[] = [];
    const spy = vi.spyOn(window, "dispatchEvent").mockImplementation((e) => {
      dispatched.push(e);
      return true;
    });

    handleCanvasEvent(
      makeMsg({ event: "WORKSPACE_PROVISIONING", workspace_id: "ws-existing", payload: {} }),
      get,
      set
    );

    expect(dispatched).toHaveLength(0);
  });
});
