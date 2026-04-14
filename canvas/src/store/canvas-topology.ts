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
 */
export function buildNodesAndEdges(
  workspaces: WorkspaceData[],
  layoutOverrides: Map<string, { x: number; y: number }> = new Map()
): {
  nodes: Node<WorkspaceNodeData>[];
  edges: Edge[];
} {
  // All workspaces become nodes (children are rendered inside parent via WorkspaceNode)
  const nodes: Node<WorkspaceNodeData>[] = workspaces.map((ws) => {
    const override = layoutOverrides.get(ws.id);
    const x = override?.x ?? ws.x;
    const y = override?.y ?? ws.y;
    return {
      id: ws.id,
      type: "workspaceNode",
      position: { x, y },
      // Don't set React Flow parentId — children render embedded inside the WorkspaceNode component
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
      },
      // Hide child nodes from canvas — they render inside the parent WorkspaceNode
      hidden: !!ws.parent_id,
    };
  });

  // No parent→child edges — children are embedded inside the parent node.
  // Only create edges between siblings or cross-team connections if needed in future.
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
