import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { apiCall, toMcpResult } from "../api.js";

export async function handleListPeers(params: { workspace_id: string }) {
  const data = await apiCall("GET", `/registry/${params.workspace_id}/peers`);
  return toMcpResult(data);
}

export async function handleDiscoverWorkspace(params: { workspace_id: string }) {
  const data = await apiCall("GET", `/registry/discover/${params.workspace_id}`);
  return toMcpResult(data);
}

export async function handleCheckAccess(params: { caller_id: string; target_id: string }) {
  const { caller_id, target_id } = params;
  const data = await apiCall("POST", `/registry/check-access`, { caller_id, target_id });
  return toMcpResult(data);
}

export async function handleListEvents(params: { workspace_id?: string }) {
  const path = params.workspace_id ? `/events/${params.workspace_id}` : "/events";
  const data = await apiCall("GET", path);
  return toMcpResult(data);
}

export async function handleListTemplates() {
  const data = await apiCall("GET", "/templates");
  return toMcpResult(data);
}

export async function handleListOrgTemplates() {
  const data = await apiCall("GET", "/org/templates");
  return toMcpResult(data);
}

export async function handleImportOrg(params: { dir: string }) {
  const data = await apiCall("POST", "/org/import", { dir: params.dir });
  return toMcpResult(data);
}

export async function handleImportTemplate(params: { name: string; files: Record<string, string> }) {
  const { name, files } = params;
  const data = await apiCall("POST", `/templates/import`, { name, files });
  return toMcpResult(data);
}

export async function handleExportBundle(params: { workspace_id: string }) {
  const data = await apiCall("GET", `/bundles/export/${params.workspace_id}`);
  return toMcpResult(data);
}

export async function handleImportBundle(params: { bundle: Record<string, unknown> }) {
  const data = await apiCall("POST", `/bundles/import`, params.bundle);
  return toMcpResult(data);
}

export async function handleGetViewport() {
  const data = await apiCall("GET", "/canvas/viewport");
  return toMcpResult(data);
}

export async function handleSetViewport(params: { x: number; y: number; zoom: number }) {
  const data = await apiCall("PUT", "/canvas/viewport", params);
  return toMcpResult(data);
}

export async function handleExpandTeam(params: { workspace_id: string }) {
  const data = await apiCall("POST", `/workspaces/${params.workspace_id}/expand`, {});
  return toMcpResult(data);
}

export async function handleCollapseTeam(params: { workspace_id: string }) {
  const data = await apiCall("POST", `/workspaces/${params.workspace_id}/collapse`, {});
  return toMcpResult(data);
}

export function registerDiscoveryTools(srv: McpServer) {
  srv.tool(
    "list_peers",
    "List reachable peer workspaces (siblings, children, parent)",
    { workspace_id: z.string() },
    handleListPeers
  );

  srv.tool(
    "discover_workspace",
    "Resolve a workspace URL by ID (for A2A communication)",
    { workspace_id: z.string() },
    handleDiscoverWorkspace
  );

  srv.tool(
    "check_access",
    "Check if two workspaces can communicate",
    { caller_id: z.string(), target_id: z.string() },
    handleCheckAccess
  );

  srv.tool(
    "list_events",
    "List structure events (global or per workspace)",
    { workspace_id: z.string().optional().describe("Filter to workspace, or omit for all") },
    handleListEvents
  );

  srv.tool("list_templates", "List available workspace templates", {}, handleListTemplates);

  srv.tool("list_org_templates", "List available org templates", {}, handleListOrgTemplates);

  srv.tool(
    "import_org",
    "Import an org template to create an entire workspace hierarchy",
    { dir: z.string().describe("Org template directory name (e.g., 'molecule-dev')") },
    handleImportOrg
  );

  srv.tool(
    "import_template",
    "Import agent files as a new workspace template",
    {
      name: z.string().describe("Template name"),
      files: z.record(z.string()).describe("Map of file path → content"),
    },
    handleImportTemplate
  );

  srv.tool(
    "export_bundle",
    "Export a workspace as a portable .bundle.json",
    { workspace_id: z.string() },
    handleExportBundle
  );

  srv.tool(
    "import_bundle",
    "Import a workspace from a bundle JSON object",
    { bundle: z.record(z.unknown()).describe("Bundle JSON object") },
    handleImportBundle
  );

  srv.tool(
    "get_canvas_viewport",
    "Get the current canvas viewport (x, y, zoom) persisted per-user.",
    {},
    handleGetViewport,
  );

  srv.tool(
    "set_canvas_viewport",
    "Persist the canvas viewport (x, y, zoom).",
    {
      x: z.number(),
      y: z.number(),
      zoom: z.number(),
    },
    handleSetViewport,
  );

  srv.tool(
    "expand_team",
    "Expand a workspace into a team of sub-workspaces",
    { workspace_id: z.string().describe("Workspace ID to expand") },
    handleExpandTeam
  );

  srv.tool(
    "collapse_team",
    "Collapse a team back to a single workspace",
    { workspace_id: z.string().describe("Workspace ID to collapse") },
    handleCollapseTeam
  );
}
