import type { Node, Edge } from "@xyflow/react";
import type { WorkspaceData } from "./socket";
import type { WorkspaceNodeData } from "./canvas";

const H_SPACING = 320;
const V_SPACING = 200;

/**
 * Computes auto-layout positions for workspaces that have no persisted position
 * (x === 0 AND y === 0). Workspaces with an existing non-zero position are used
 * as anchors and are never moved.
 *
 * Returns a Map of workspace IDs → {x, y} for every workspace that was assigned
 * a computed position (i.e. only the ones that were at 0,0). Callers should
 * persist these back to the API so the positions survive reload.
 */
export function computeAutoLayout(
  workspaces: WorkspaceData[]
): Map<string, { x: number; y: number }> {
  const overrides = new Map<string, { x: number; y: number }>();

  // Separate anchored (already positioned) from zero-position workspaces
  const anchored = new Set<string>();
  for (const ws of workspaces) {
    if (ws.x !== 0 || ws.y !== 0) {
      anchored.add(ws.id);
    }
  }

  // If every workspace is already positioned, nothing to do
  const needsLayout = workspaces.filter((ws) => !anchored.has(ws.id));
  if (needsLayout.length === 0) return overrides;

  // Build parent→children map
  const children = new Map<string | null, WorkspaceData[]>();
  for (const ws of workspaces) {
    const pid = ws.parent_id ?? null;
    if (!children.has(pid)) children.set(pid, []);
    children.get(pid)!.push(ws);
  }

  // Sort children by name for deterministic layout
  for (const list of children.values()) {
    list.sort((a, b) => a.name.localeCompare(b.name));
  }

  // Assigned positions (includes anchors from the original data + computed overrides)
  const assigned = new Map<string, { x: number; y: number }>();
  for (const ws of workspaces) {
    if (anchored.has(ws.id)) {
      assigned.set(ws.id, { x: ws.x, y: ws.y });
    }
  }

  // BFS from root nodes that need layout
  // Track the next X offset per depth row to spread siblings horizontally
  const rowNextX = new Map<number, number>();

  // Enqueue root-level nodes that need layout
  const queue: Array<{ ws: WorkspaceData; depth: number }> = [];
  const rootsNeedingLayout = (children.get(null) ?? []).filter(
    (ws) => !anchored.has(ws.id)
  );
  for (const ws of rootsNeedingLayout) {
    queue.push({ ws, depth: 0 });
  }

  while (queue.length > 0) {
    const { ws, depth } = queue.shift()!;

    // Skip if already assigned (e.g. anchored)
    if (assigned.has(ws.id)) {
      // Still enqueue its unpositioned children
      const kids = (children.get(ws.id) ?? []).filter(
        (c) => !anchored.has(c.id) && !assigned.has(c.id)
      );
      for (const kid of kids) {
        queue.push({ ws: kid, depth: depth + 1 });
      }
      continue;
    }

    // Find parent's x as the center hint for this node
    const parentPos = ws.parent_id ? assigned.get(ws.parent_id) : undefined;
    const parentX = parentPos?.x ?? 0;

    // Place node at the next available slot in this row
    const currentRowX = rowNextX.get(depth) ?? (parentX - H_SPACING / 2);
    const x = Math.max(currentRowX, parentX);
    const y = depth * V_SPACING;

    assigned.set(ws.id, { x, y });
    overrides.set(ws.id, { x, y });
    rowNextX.set(depth, x + H_SPACING);

    // Enqueue children that need layout
    const kids = (children.get(ws.id) ?? []).filter(
      (c) => !anchored.has(c.id) && !assigned.has(c.id)
    );
    for (const kid of kids) {
      queue.push({ ws: kid, depth: depth + 1 });
    }
  }

  return overrides;
}

/**
 * Converts raw workspace data from the API into React Flow nodes and edges.
 * Accepts an optional layoutOverrides map (from computeAutoLayout) to override
 * positions for workspaces that were at 0,0.
 *
 * Parent/child rendering model: every workspace is a first-class React Flow
 * node (full card). When a workspace has parent_id set, its RF `parentId` is
 * set to the parent's id and its position is stored RELATIVE to the parent
 * origin — React Flow renders the child inside the parent's coordinate space,
 * so moving the parent automatically moves all children. The DB keeps
 * absolute x/y; the abs→rel conversion happens here on load, and the
 * reverse translation happens in savePosition.
 */
export function buildNodesAndEdges(
  workspaces: WorkspaceData[],
  layoutOverrides: Map<string, { x: number; y: number }> = new Map()
): {
  nodes: Node<WorkspaceNodeData>[];
  edges: Edge[];
} {
  // React Flow requires parent nodes to appear before children in the nodes
  // array. Topological-sort by depth-first walk from roots so children come
  // after their parent regardless of the order the API returned them.
  const byId = new Map(workspaces.map((w) => [w.id, w]));
  const visited = new Set<string>();
  const sorted: WorkspaceData[] = [];
  function visit(ws: WorkspaceData) {
    if (visited.has(ws.id)) return;
    if (ws.parent_id && byId.has(ws.parent_id) && !visited.has(ws.parent_id)) {
      visit(byId.get(ws.parent_id)!);
    }
    visited.add(ws.id);
    sorted.push(ws);
  }
  workspaces.forEach(visit);

  // Resolve each workspace's absolute position (apply layout override if any).
  const absPos = new Map<string, { x: number; y: number }>();
  for (const ws of workspaces) {
    const o = layoutOverrides.get(ws.id);
    absPos.set(ws.id, { x: o?.x ?? ws.x, y: o?.y ?? ws.y });
  }

  const nodes: Node<WorkspaceNodeData>[] = sorted.map((ws) => {
    const abs = absPos.get(ws.id)!;
    const hasParent = !!ws.parent_id && byId.has(ws.parent_id);
    let position = abs;
    if (hasParent) {
      const pa = absPos.get(ws.parent_id!)!;
      position = { x: abs.x - pa.x, y: abs.y - pa.y };
    }
    const node: Node<WorkspaceNodeData> = {
      id: ws.id,
      type: "workspaceNode",
      position,
      data: {
        name: ws.name,
        status: ws.status,
        tier: ws.tier,
        agentCard: ws.agent_card,
        activeTasks: ws.active_tasks,
        collapsed: ws.collapsed,
        role: ws.role,
        lastErrorRate: ws.last_error_rate,
        lastSampleError: ws.last_sample_error,
        url: ws.url,
        parentId: ws.parent_id,
        currentTask: ws.current_task || "",
        runtime: ws.runtime || "",
        needsRestart: false,
        budgetLimit: ws.budget_limit ?? null,
        budgetUsed: ws.budget_used ?? null,
      },
    };
    if (hasParent) {
      // React Flow native parent binding: children render inside parent's
      // coordinate space and move with the parent. No `extent: 'parent'` —
      // the user can drag a child out to un-nest (handled in Canvas.tsx
      // onNodeDragStop with a bbox hit test).
      node.parentId = ws.parent_id!;
    }
    return node;
  });

  // Edges stay empty — the visual parent/child cue is the enclosing card.
  const edges: Edge[] = [];

  return { nodes, edges };
}

/**
 * Extracts skill names from an agent card's skills array.
 */
export function extractSkillNames(agentCard: Record<string, unknown> | null): string[] {
  if (!agentCard) return [];
  const skills = agentCard.skills;
  if (!Array.isArray(skills)) return [];
  return skills
    .map((skill: Record<string, unknown>) => String(skill.name || skill.id || ""))
    .filter(Boolean);
}
