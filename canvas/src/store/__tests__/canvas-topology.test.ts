import { describe, it, expect } from "vitest";
import { buildNodesAndEdges, extractSkillNames } from "../canvas-topology";
import type { WorkspaceData } from "../socket";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// buildNodesAndEdges
// ---------------------------------------------------------------------------

describe("buildNodesAndEdges – empty array", () => {
  it("returns empty nodes and edges", () => {
    const { nodes, edges } = buildNodesAndEdges([]);
    expect(nodes).toHaveLength(0);
    expect(edges).toHaveLength(0);
  });
});

describe("buildNodesAndEdges – single workspace", () => {
  it("converts one workspace to one node", () => {
    const { nodes, edges } = buildNodesAndEdges([makeWS({ id: "ws-1", name: "Solo", x: 10, y: 20 })]);

    expect(nodes).toHaveLength(1);
    expect(edges).toHaveLength(0);

    const n = nodes[0];
    expect(n.id).toBe("ws-1");
    expect(n.type).toBe("workspaceNode");
    expect(n.position).toEqual({ x: 10, y: 20 });
    expect(n.hidden).toBeFalsy();
  });

  it("maps all workspace fields to node data", () => {
    const ws = makeWS({
      id: "ws-x",
      name: "Test",
      role: "lead",
      tier: 2,
      status: "degraded",
      agent_card: { skills: [] },
      url: "http://test:9000",
      active_tasks: 4,
      last_error_rate: 0.9,
      last_sample_error: "oops",
      collapsed: true,
      current_task: "Doing something",
    });

    const { nodes } = buildNodesAndEdges([ws]);
    const data = nodes[0].data;

    expect(data.name).toBe("Test");
    expect(data.role).toBe("lead");
    expect(data.tier).toBe(2);
    expect(data.status).toBe("degraded");
    expect(data.agentCard).toEqual({ skills: [] });
    expect(data.url).toBe("http://test:9000");
    expect(data.activeTasks).toBe(4);
    expect(data.lastErrorRate).toBe(0.9);
    expect(data.lastSampleError).toBe("oops");
    expect(data.collapsed).toBe(true);
    expect(data.currentTask).toBe("Doing something");
  });

  it("sets needsRestart to false by default", () => {
    const { nodes } = buildNodesAndEdges([makeWS({ id: "ws-1" })]);
    expect(nodes[0].data.needsRestart).toBe(false);
  });

  it("sets node position from x and y", () => {
    const { nodes } = buildNodesAndEdges([makeWS({ id: "a", x: 150, y: 300 })]);
    expect(nodes[0].position).toEqual({ x: 150, y: 300 });
  });
});

describe("buildNodesAndEdges – parent + child workspaces", () => {
  it("creates two nodes and no edges", () => {
    const { nodes, edges } = buildNodesAndEdges([
      makeWS({ id: "parent" }),
      makeWS({ id: "child", parent_id: "parent" }),
    ]);

    expect(nodes).toHaveLength(2);
    // No edges: children render embedded inside WorkspaceNode
    expect(edges).toHaveLength(0);
  });

  it("marks parent as visible and child as hidden", () => {
    const { nodes } = buildNodesAndEdges([
      makeWS({ id: "parent" }),
      makeWS({ id: "child", parent_id: "parent" }),
    ]);

    const parent = nodes.find((n) => n.id === "parent")!;
    const child = nodes.find((n) => n.id === "child")!;

    expect(parent.hidden).toBeFalsy();
    expect(child.hidden).toBe(true);
  });

  it("stores parent_id in child node data as parentId", () => {
    const { nodes } = buildNodesAndEdges([
      makeWS({ id: "parent" }),
      makeWS({ id: "child", parent_id: "parent" }),
    ]);

    const child = nodes.find((n) => n.id === "child")!;
    expect(child.data.parentId).toBe("parent");
  });

  it("root node has parentId null", () => {
    const { nodes } = buildNodesAndEdges([
      makeWS({ id: "parent" }),
      makeWS({ id: "child", parent_id: "parent" }),
    ]);

    const parent = nodes.find((n) => n.id === "parent")!;
    expect(parent.data.parentId).toBeNull();
  });
});

describe("buildNodesAndEdges – deeply nested hierarchy", () => {
  it("handles three levels of nesting", () => {
    const workspaces = [
      makeWS({ id: "root" }),
      makeWS({ id: "mid", parent_id: "root" }),
      makeWS({ id: "leaf", parent_id: "mid" }),
    ];

    const { nodes, edges } = buildNodesAndEdges(workspaces);

    expect(nodes).toHaveLength(3);
    expect(edges).toHaveLength(0);

    expect(nodes.find((n) => n.id === "root")!.hidden).toBeFalsy();
    expect(nodes.find((n) => n.id === "mid")!.hidden).toBe(true);
    expect(nodes.find((n) => n.id === "leaf")!.hidden).toBe(true);

    expect(nodes.find((n) => n.id === "mid")!.data.parentId).toBe("root");
    expect(nodes.find((n) => n.id === "leaf")!.data.parentId).toBe("mid");
  });

  it("handles multiple root-level nodes", () => {
    const workspaces = [
      makeWS({ id: "root-a", x: 0, y: 0 }),
      makeWS({ id: "root-b", x: 200, y: 0 }),
      makeWS({ id: "child-a", parent_id: "root-a" }),
    ];

    const { nodes } = buildNodesAndEdges(workspaces);

    expect(nodes).toHaveLength(3);
    expect(nodes.find((n) => n.id === "root-a")!.hidden).toBeFalsy();
    expect(nodes.find((n) => n.id === "root-b")!.hidden).toBeFalsy();
    expect(nodes.find((n) => n.id === "child-a")!.hidden).toBe(true);
  });
});

describe("buildNodesAndEdges – current_task field", () => {
  it("maps current_task to currentTask", () => {
    const { nodes } = buildNodesAndEdges([makeWS({ id: "ws-1", current_task: "Working hard" })]);
    expect(nodes[0].data.currentTask).toBe("Working hard");
  });

  it("defaults currentTask to empty string when current_task is empty", () => {
    const { nodes } = buildNodesAndEdges([makeWS({ id: "ws-1", current_task: "" })]);
    expect(nodes[0].data.currentTask).toBe("");
  });
});

// ---------------------------------------------------------------------------
// extractSkillNames
// ---------------------------------------------------------------------------

describe("extractSkillNames – null / missing agent card", () => {
  it("returns empty array for null", () => {
    expect(extractSkillNames(null)).toEqual([]);
  });

  it("returns empty array for empty object (no skills key)", () => {
    expect(extractSkillNames({})).toEqual([]);
  });

  it("returns empty array when skills is not an array", () => {
    expect(extractSkillNames({ skills: "not-an-array" })).toEqual([]);
    expect(extractSkillNames({ skills: 42 })).toEqual([]);
    expect(extractSkillNames({ skills: null })).toEqual([]);
  });
});

describe("extractSkillNames – valid agent card with skills", () => {
  it("extracts skill names using the name field", () => {
    const card = {
      skills: [
        { id: "write", name: "Writing" },
        { id: "plan", name: "Planning" },
      ],
    };
    expect(extractSkillNames(card)).toEqual(["Writing", "Planning"]);
  });

  it("falls back to skill id when name is absent", () => {
    const card = {
      skills: [{ id: "code" }, { id: "search" }],
    };
    expect(extractSkillNames(card)).toEqual(["code", "search"]);
  });

  it("prefers name over id when both are present", () => {
    const card = {
      skills: [{ id: "write", name: "Writing" }],
    };
    expect(extractSkillNames(card)).toEqual(["Writing"]);
  });

  it("filters out skills with no name and no id", () => {
    const card = {
      skills: [{ name: "Valid" }, {}, { id: "" }],
    };
    expect(extractSkillNames(card)).toEqual(["Valid"]);
  });
});

describe("extractSkillNames – empty skills array", () => {
  it("returns empty array", () => {
    expect(extractSkillNames({ skills: [] })).toEqual([]);
  });
});

describe("extractSkillNames – mixed valid/invalid skills", () => {
  it("returns only named skills and skips empty ones", () => {
    const card = {
      skills: [
        { id: "code", name: "Coding" },
        { id: "", name: "" },
        { id: "test", name: "Testing" },
      ],
    };
    expect(extractSkillNames(card)).toEqual(["Coding", "Testing"]);
  });
});
