import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { apiCall } from "../api.js";

export async function handleListWorkspaces() {
  const data = await apiCall("GET", "/workspaces");
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleCreateWorkspace(params: {
  name: string;
  role?: string;
  template?: string;
  tier?: number;
  parent_id?: string;
  runtime?: string;
  workspace_dir?: string;
  workspace_access?: "none" | "read_only" | "read_write";
}) {
  const { name, role, template, tier, parent_id, runtime, workspace_dir, workspace_access } = params;
  const data = await apiCall("POST", "/workspaces", {
    name, role, template, tier, parent_id, runtime,
    workspace_dir, workspace_access,
    canvas: { x: Math.random() * 400 + 100, y: Math.random() * 300 + 100 },
  });
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleGetWorkspace(params: { workspace_id: string }) {
  const data = await apiCall("GET", `/workspaces/${params.workspace_id}`);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleDeleteWorkspace(params: { workspace_id: string }) {
  const data = await apiCall("DELETE", `/workspaces/${params.workspace_id}?confirm=true`);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleRestartWorkspace(params: { workspace_id: string }) {
  const data = await apiCall("POST", `/workspaces/${params.workspace_id}/restart`, {});
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleUpdateWorkspace(params: {
  workspace_id: string;
  name?: string;
  role?: string;
  tier?: number;
  parent_id?: string | null;
  workspace_dir?: string;
  workspace_access?: "none" | "read_only" | "read_write";
}) {
  const { workspace_id, ...fields } = params;
  const data = await apiCall("PATCH", `/workspaces/${workspace_id}`, fields);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handlePauseWorkspace(params: { workspace_id: string }) {
  const data = await apiCall("POST", `/workspaces/${params.workspace_id}/pause`, {});
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleResumeWorkspace(params: { workspace_id: string }) {
  const data = await apiCall("POST", `/workspaces/${params.workspace_id}/resume`, {});
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export function registerWorkspaceTools(srv: McpServer) {
  srv.tool("list_workspaces", "List all workspaces with their status, skills, and hierarchy", {}, handleListWorkspaces);

  srv.tool(
    "create_workspace",
    "Create a new workspace node on the canvas",
    {
      name: z.string().describe("Workspace name"),
      role: z.string().optional().describe("Role description"),
      template: z.string().optional().describe("Template name from workspace-configs-templates/"),
      tier: z.number().min(1).max(4).default(1).describe("Tier (1=basic, 2=browser, 3=desktop, 4=VM)"),
      parent_id: z.string().optional().describe("Parent workspace ID for nesting"),
    },
    handleCreateWorkspace
  );

  srv.tool(
    "get_workspace",
    "Get detailed information about a specific workspace",
    { workspace_id: z.string().describe("Workspace ID") },
    handleGetWorkspace
  );

  srv.tool(
    "delete_workspace",
    "Delete a workspace (cascades to children)",
    { workspace_id: z.string().describe("Workspace ID") },
    handleDeleteWorkspace
  );

  srv.tool(
    "restart_workspace",
    "Restart an offline or failed workspace",
    { workspace_id: z.string().describe("Workspace ID") },
    handleRestartWorkspace
  );

  srv.tool(
    "update_workspace",
    "Update workspace fields (name, role, tier, parent_id, position)",
    {
      workspace_id: z.string(),
      name: z.string().optional(),
      role: z.string().optional(),
      tier: z.number().optional(),
      parent_id: z.string().nullable().optional().describe("Set parent for nesting, null to un-nest"),
    },
    handleUpdateWorkspace
  );

  srv.tool(
    "pause_workspace",
    "Pause a workspace (stops container, preserves config)",
    { workspace_id: z.string().describe("Workspace ID") },
    handlePauseWorkspace
  );

  srv.tool(
    "resume_workspace",
    "Resume a paused workspace",
    { workspace_id: z.string().describe("Workspace ID") },
    handleResumeWorkspace
  );
}
