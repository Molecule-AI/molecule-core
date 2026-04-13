import type { Node, Edge } from "@xyflow/react";
import type { WorkspaceData } from "./socket";
import type { WorkspaceNodeData } from "./canvas";

/**
 * Converts raw workspace data from the API into React Flow nodes and edges.
 */
export function buildNodesAndEdges(workspaces: WorkspaceData[]): {
  nodes: Node<WorkspaceNodeData>[];
  edges: Edge[];
} {
  // All workspaces become nodes (children are rendered inside parent via WorkspaceNode)
  const nodes: Node<WorkspaceNodeData>[] = workspaces.map((ws) => ({
    id: ws.id,
    type: "workspaceNode",
    position: { x: ws.x, y: ws.y },
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
  }));

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
