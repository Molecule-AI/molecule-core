import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { apiCall } from "../api.js";

export async function handleListPluginRegistry() {
  const data = await apiCall("GET", "/plugins");
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleListInstalledPlugins(params: { workspace_id: string }) {
  const data = await apiCall("GET", `/workspaces/${params.workspace_id}/plugins`);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleInstallPlugin(params: { workspace_id: string; source: string }) {
  const { workspace_id, source } = params;
  const data = await apiCall("POST", `/workspaces/${workspace_id}/plugins`, { source });
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleUninstallPlugin(params: { workspace_id: string; name: string }) {
  const { workspace_id, name } = params;
  const data = await apiCall("DELETE", `/workspaces/${workspace_id}/plugins/${name}`);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleListPluginSources() {
  const data = await apiCall("GET", "/plugins/sources");
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleListAvailablePlugins(params: { workspace_id: string }) {
  const data = await apiCall("GET", `/workspaces/${params.workspace_id}/plugins/available`);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleCheckPluginCompatibility(params: {
  workspace_id: string;
  runtime: string;
}) {
  const { workspace_id, runtime } = params;
  const data = await apiCall(
    "GET",
    `/workspaces/${workspace_id}/plugins/compatibility?runtime=${encodeURIComponent(runtime)}`,
  );
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export function registerPluginTools(srv: McpServer) {
  srv.tool("list_plugin_registry", "List all available plugins from the registry", {}, handleListPluginRegistry);

  srv.tool(
    "list_installed_plugins",
    "List plugins installed in a workspace",
    { workspace_id: z.string().describe("Workspace ID") },
    handleListInstalledPlugins
  );

  srv.tool(
    "install_plugin",
    "Install a plugin into a workspace from any registered source (auto-restarts). Use GET /plugins/sources to list schemes.",
    {
      workspace_id: z.string().describe("Workspace ID"),
      source: z
        .string()
        .describe(
          "Source URL: 'local://<name>' for platform registry, 'github://<owner>/<repo>[#<ref>]' for GitHub, or any registered scheme."
        ),
    },
    handleInstallPlugin
  );

  srv.tool(
    "uninstall_plugin",
    "Remove a plugin from a workspace (auto-restarts)",
    {
      workspace_id: z.string().describe("Workspace ID"),
      name: z.string().describe("Plugin name to remove"),
    },
    handleUninstallPlugin
  );

  srv.tool(
    "list_plugin_sources",
    "List registered plugin install-source schemes (e.g. local, github).",
    {},
    handleListPluginSources,
  );

  srv.tool(
    "list_available_plugins",
    "List plugins from the registry filtered to ones supported by this workspace's runtime.",
    { workspace_id: z.string() },
    handleListAvailablePlugins,
  );

  srv.tool(
    "check_plugin_compatibility",
    "Preflight check: which installed plugins would break if this workspace switched runtime to <runtime>?",
    {
      workspace_id: z.string(),
      runtime: z.string().describe("Target runtime (claude-code, deepagents, langgraph, ...)"),
    },
    handleCheckPluginCompatibility,
  );
}
