import { describe, it, expect } from "vitest";
import { buildNodesAndEdges, extractSkillNames, computeAutoLayout } from "../canvas-topology";
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
    budget_limit: null,
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

  it("binds child to parent via React Flow's native parentId", () => {
    // Children are first-class nodes now (rendered as full cards inside
    // their parent via RF's parentId). No `hidden` flag anymore — the
    // nesting is visual, not hide-and-show.
    const { nodes } = buildNodesAndEdges([
      makeWS({ id: "parent" }),
      makeWS({ id: "child", parent_id: "parent" }),
    ]);

    const parent = nodes.find((n) => n.id === "parent")!;
    const child = nodes.find((n) => n.id === "child")!;

    expect(parent.hidden).toBeFalsy();
    expect(child.hidden).toBeFalsy();
    expect(parent.parentId).toBeUndefined();
    expect(child.parentId).toBe("parent");
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

    expect(nodes.find((n) => n.id === "root")!.parentId).toBeUndefined();
    expect(nodes.find((n) => n.id === "mid")!.parentId).toBe("root");
    expect(nodes.find((n) => n.id === "leaf")!.parentId).toBe("mid");

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
    expect(nodes.find((n) => n.id === "root-a")!.parentId).toBeUndefined();
    expect(nodes.find((n) => n.id === "root-b")!.parentId).toBeUndefined();
    expect(nodes.find((n) => n.id === "child-a")!.parentId).toBe("root-a");
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

// ---------------------------------------------------------------------------
// computeAutoLayout
// ---------------------------------------------------------------------------

describe("computeAutoLayout – all nodes already positioned", () => {
  it("returns empty map when all nodes have non-zero positions", () => {
    const workspaces = [
      makeWS({ id: "a", x: 100, y: 200 }),
      makeWS({ id: "b", x: 400, y: 200 }),
    ];
    const overrides = computeAutoLayout(workspaces);
    expect(overrides.size).toBe(0);
  });
});

describe("computeAutoLayout – empty workspace list", () => {
  it("returns empty map", () => {
    const overrides = computeAutoLayout([]);
    expect(overrides.size).toBe(0);
  });
});

describe("computeAutoLayout – single zero-position root node", () => {
  it("assigns a position to the zero node", () => {
    const workspaces = [makeWS({ id: "ws-1", x: 0, y: 0 })];
    const overrides = computeAutoLayout(workspaces);
    expect(overrides.has("ws-1")).toBe(true);
    const pos = overrides.get("ws-1")!;
    expect(typeof pos.x).toBe("number");
    expect(typeof pos.y).toBe("number");
  });
});

describe("computeAutoLayout – multiple zero-position root nodes", () => {
  it("spreads siblings horizontally (distinct x values)", () => {
    const workspaces = [
      makeWS({ id: "a", x: 0, y: 0 }),
      makeWS({ id: "b", x: 0, y: 0 }),
      makeWS({ id: "c", x: 0, y: 0 }),
    ];
    const overrides = computeAutoLayout(workspaces);
    const positions = ["a", "b", "c"].map((id) => overrides.get(id)!);
    const xs = positions.map((p) => p.x);
    // All x values should be unique (nodes spread horizontally)
    const uniqueXs = new Set(xs);
    expect(uniqueXs.size).toBe(3);
    // All at depth 0 → y should be 0
    for (const p of positions) {
      expect(p.y).toBe(0);
    }
  });
});

describe("computeAutoLayout – parent with zero-position children", () => {
  it("places child at greater y than parent", () => {
    const workspaces = [
      makeWS({ id: "parent", x: 0, y: 0 }),
      makeWS({ id: "child", parent_id: "parent", x: 0, y: 0 }),
    ];
    const overrides = computeAutoLayout(workspaces);
    const parentPos = overrides.get("parent")!;
    const childPos = overrides.get("child")!;
    expect(childPos.y).toBeGreaterThan(parentPos.y);
  });
});

describe("computeAutoLayout – anchored node not overridden", () => {
  it("does not include already-positioned node in overrides", () => {
    const workspaces = [
      makeWS({ id: "anchored", x: 500, y: 300 }),
      makeWS({ id: "zero", x: 0, y: 0 }),
    ];
    const overrides = computeAutoLayout(workspaces);
    expect(overrides.has("anchored")).toBe(false);
    expect(overrides.has("zero")).toBe(true);
  });
});

describe("buildNodesAndEdges – layoutOverrides applied", () => {
  it("uses override position instead of ws.x/ws.y for zero-position nodes", () => {
    const workspaces = [makeWS({ id: "ws-1", x: 0, y: 0 })];
    const overrides = new Map([["ws-1", { x: 150, y: 250 }]]);
    const { nodes } = buildNodesAndEdges(workspaces, overrides);
    expect(nodes[0].position).toEqual({ x: 150, y: 250 });
  });

  it("leaves non-overridden node at its own position", () => {
    const workspaces = [makeWS({ id: "ws-2", x: 100, y: 200 })];
    const overrides = new Map<string, { x: number; y: number }>();
    const { nodes } = buildNodesAndEdges(workspaces, overrides);
    expect(nodes[0].position).toEqual({ x: 100, y: 200 });
  });
});
