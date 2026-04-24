import type { Node, Edge } from "@xyflow/react";
import type { WorkspaceData } from "./socket";
import type { WorkspaceNodeData } from "./canvas";

const H_SPACING = 320;
const V_SPACING = 200;

// Default card footprint we use when we don't yet have a measured size
// (first render, before React Flow reports dimensions). These match the
// min-width / min-height that WorkspaceNode.tsx sets, so a parent built
// from them will never start too small for its children on first paint.
/**
 * Re-orders a React Flow node array so parents always appear BEFORE
 * their children. React Flow requires this ordering; when it's
 * violated RF logs "Parent node ... not found" and renders the child
 * at canvas-absolute coords (losing the parent-relative transform).
 *
 * We call this every time nestNode / batchNest mutates parentId —
 * without a re-sort a freshly-nested child can appear AFTER its new
 * parent in the array, which breaks the next drag.
 */
export function sortParentsBeforeChildren<T extends { id: string; parentId?: string }>(
  nodes: T[],
): T[] {
  const byId = new Map(nodes.map((n) => [n.id, n]));
  const visited = new Set<string>();
  const out: T[] = [];
  const visit = (n: T) => {
    if (visited.has(n.id)) return;
    if (n.parentId) {
      const parent = byId.get(n.parentId);
      if (parent && !visited.has(parent.id)) visit(parent);
    }
    visited.add(n.id);
    out.push(n);
  };
  for (const n of nodes) visit(n);
  return out;
}

export const CHILD_DEFAULT_WIDTH = 260;
export const CHILD_DEFAULT_HEIGHT = 140;
export const PARENT_HEADER_PADDING = 60; // room for the parent's own header
export const PARENT_SIDE_PADDING = 20;
export const PARENT_BOTTOM_PADDING = 20;
export const CHILD_GUTTER = 20;


/**
 * A deterministic grid slot for the n-th child inside a parent, counted
 * left-to-right then top-to-bottom. Used to lay out org-imported teams
 * and to rescue children whose stored position puts them outside the
 * parent's bounding box. 2-column grid is wide enough to read but
 * narrow enough to keep the parent card from becoming a widescreen.
 */
export function defaultChildSlot(index: number): { x: number; y: number } {
  const col = index % 2;
  const row = Math.floor(index / 2);
  const x = PARENT_SIDE_PADDING + col * (CHILD_DEFAULT_WIDTH + CHILD_GUTTER);
  const y =
    PARENT_HEADER_PADDING + row * (CHILD_DEFAULT_HEIGHT + CHILD_GUTTER);
  return { x, y };
}

/**
 * Minimum parent size that still fits `childCount` children laid out via
 * defaultChildSlot. Never shrinks below the leaf-card min.
 */
export function parentMinSize(childCount: number): { width: number; height: number } {
  if (childCount <= 0) {
    return { width: 210, height: 120 };
  }
  const cols = Math.min(2, childCount);
  const rows = Math.ceil(childCount / 2);
  const width =
    PARENT_SIDE_PADDING * 2 +
    cols * CHILD_DEFAULT_WIDTH +
    (cols - 1) * CHILD_GUTTER;
  const height =
    PARENT_HEADER_PADDING +
    rows * CHILD_DEFAULT_HEIGHT +
    (rows - 1) * CHILD_GUTTER +
    PARENT_BOTTOM_PADDING;
  return { width, height };
}

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

  // Count children per parent so we can size parents to fit their team
  // before any runtime measurement comes back.
  const childCounts = new Map<string, number>();
  for (const ws of workspaces) {
    if (ws.parent_id) {
      childCounts.set(ws.parent_id, (childCounts.get(ws.parent_id) ?? 0) + 1);
    }
  }

  // Track each parent's initial size so we can reset children that land
  // outside those bounds. Parents without children fall back to the leaf
  // default; parents with children get the grid-derived minimum.
  const parentSize = new Map<string, { width: number; height: number }>();
  for (const ws of workspaces) {
    const n = childCounts.get(ws.id) ?? 0;
    parentSize.set(ws.id, n > 0 ? parentMinSize(n) : { width: 260, height: 140 });
  }

  // Running index of children already placed per parent — used to hand
  // out default grid slots for children whose stored position is outside
  // the parent's computed box.
  const nextChildIndex = new Map<string, number>();

  // Depth per node so children always render above parents (and above
  // parent's root-level siblings). React Flow uses a flat zIndex, so a
  // child inherits zIndex = parent.zIndex + 1 — xyflow issue #4012.
  const depthById = new Map<string, number>();
  for (const ws of sorted) {
    const d = ws.parent_id ? (depthById.get(ws.parent_id) ?? 0) + 1 : 0;
    depthById.set(ws.id, d);
  }

  // Mark each node as hidden if any ancestor is collapsed. Walk from
  // the root so children inherit the flag efficiently. (Parents stay
  // visible; only descendants are hidden so the parent renders as a
  // compact header-only card.)
  const hiddenById = new Map<string, boolean>();
  for (const ws of sorted) {
    if (!ws.parent_id) {
      hiddenById.set(ws.id, false);
      continue;
    }
    const parent = byId.get(ws.parent_id);
    const parentHidden = hiddenById.get(ws.parent_id) ?? false;
    hiddenById.set(ws.id, parentHidden || !!parent?.collapsed);
  }

  const nodes: Node<WorkspaceNodeData>[] = sorted.map((ws) => {
    const abs = absPos.get(ws.id)!;
    const hasParent = !!ws.parent_id && byId.has(ws.parent_id);
    let position = abs;
    if (hasParent) {
      const pa = absPos.get(ws.parent_id!)!;
      position = { x: abs.x - pa.x, y: abs.y - pa.y };

      // Auto-rescue on load: fires only when the child's bounding box
      // is FULLY outside the parent's computed bbox by at least one
      // child-width/height. Two real failure modes this covers:
      //
      //   - Legacy data: a child whose stored absolute coords predate
      //     the nesting assignment, so abs→rel produces a huge offset
      //     far past any parent edge.
      //   - Corrupt org-imports with positions in a different
      //     coordinate space.
      //
      // Rejected heuristics we deliberately avoid:
      //   - `position.x < 0` alone — catches legitimate drift when the
      //     user drags the parent past a child that had small positive
      //     stored coords (child's relative goes mildly negative, but
      //     the layout is still recoverable).
      //   - Raw magnitude like `> 8000` — doesn't scale with parent
      //     size; a user who resized the parent huge could legitimately
      //     place a child at 9000px.
      //
      // Children slightly past the initial min-size (user had resized
      // the parent larger on a previous session) are NEVER rescued —
      // the bbox-overlap test gives them room. The manual "Arrange
      // Children" context command is still the escape hatch for that
      // bucket of data.
      // Pure bbox-overlap test — self-calibrating without a magic
      // margin. Rescue iff the child's bbox has ZERO overlap with the
      // parent's bbox (the child would render completely detached).
      //   drift case (position.x = -40, CHILD_WIDTH = 260):
      //     child.right = 220, overlaps parent.left = 0 → kept
      //   screenshot case (position.x = -420, CHILD_WIDTH = 260):
      //     child.right = -160, doesn't overlap parent.left = 0 → rescued
      //   user resized larger (parent.width now 800, position.x = 500):
      //     child.left = 500 < parent.right = 800 → overlaps → kept
      //   legacy huge positive (position.x = 50000):
      //     child.left = 50000 >= parent.right → no overlap → rescued
      const psize = parentSize.get(ws.parent_id!)!;
      const overlapsX =
        position.x + CHILD_DEFAULT_WIDTH > 0 && position.x < psize.width;
      const overlapsY =
        position.y + CHILD_DEFAULT_HEIGHT > 0 && position.y < psize.height;
      if (!overlapsX || !overlapsY) {
        const idx = nextChildIndex.get(ws.parent_id!) ?? 0;
        nextChildIndex.set(ws.parent_id!, idx + 1);
        position = defaultChildSlot(idx);
      }
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
    // Stack children above their ancestors (xyflow #4012).
    node.zIndex = depthById.get(ws.id) ?? 0;
    // Collapse: descendants of a collapsed parent get hidden so the
    // parent renders as a compact header-only card.
    if (hiddenById.get(ws.id)) {
      node.hidden = true;
    }
    // Give parents a measured-ish starting size so NodeResizer has a
    // baseline and child positions have somewhere to live. Without this,
    // parents start at React Flow's default min size (well under a
    // single child) and children render visually outside their parent
    // until the next resize measurement settles.
    if ((childCounts.get(ws.id) ?? 0) > 0) {
      const size = parentSize.get(ws.id)!;
      node.width = size.width;
      node.height = size.height;
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
