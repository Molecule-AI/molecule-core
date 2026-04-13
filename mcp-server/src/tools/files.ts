import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { apiCall, toMcpResult, toMcpText } from "../api.js";

export async function handleListFiles(params: { workspace_id: string }) {
  const data = await apiCall("GET", `/workspaces/${params.workspace_id}/files`);
  return toMcpResult(data);
}

export async function handleReadFile(params: { workspace_id: string; path: string }) {
  const { workspace_id, path } = params;
  const data = await apiCall<{ content?: string }>("GET", `/workspaces/${workspace_id}/files/${path}`);
  const fileText = (data as { content?: string } | null)?.content;
  return fileText ? toMcpText(fileText) : toMcpResult(data);
}

export async function handleWriteFile(params: { workspace_id: string; path: string; content: string }) {
  const { workspace_id, path, content } = params;
  const data = await apiCall("PUT", `/workspaces/${workspace_id}/files/${path}`, { content });
  return toMcpResult(data);
}

export async function handleDeleteFile(params: { workspace_id: string; path: string }) {
  const { workspace_id, path } = params;
  const data = await apiCall("DELETE", `/workspaces/${workspace_id}/files/${path}`);
  return toMcpResult(data);
}

export async function handleReplaceAllFiles(params: {
  workspace_id: string;
  files: Record<string, string>;
}) {
  const { workspace_id, files } = params;
  const data = await apiCall("PUT", `/workspaces/${workspace_id}/files`, { files });
  return toMcpResult(data);
}

export async function handleGetConfig(params: { workspace_id: string }) {
  const data = await apiCall("GET", `/workspaces/${params.workspace_id}/config`);
  return toMcpResult(data);
}

export async function handleUpdateConfig(params: { workspace_id: string; config: Record<string, unknown> }) {
  const { workspace_id, config } = params;
  const data = await apiCall("PATCH", `/workspaces/${workspace_id}/config`, config);
  return toMcpResult(data);
}

export function registerFileTools(srv: McpServer) {
  srv.tool(
    "list_files",
    "List workspace config files (skills, prompts, config.yaml)",
    { workspace_id: z.string().describe("Workspace ID") },
    handleListFiles
  );

  srv.tool(
    "read_file",
    "Read a workspace config file",
    {
      workspace_id: z.string().describe("Workspace ID"),
      path: z.string().describe("File path (e.g., system-prompt.md, skills/seo/SKILL.md)"),
    },
    handleReadFile
  );

  srv.tool(
    "write_file",
    "Write or create a workspace config file",
    {
      workspace_id: z.string().describe("Workspace ID"),
      path: z.string().describe("File path"),
      content: z.string().describe("File content"),
    },
    handleWriteFile
  );

  srv.tool(
    "delete_file",
    "Delete a workspace file or folder",
    {
      workspace_id: z.string().describe("Workspace ID"),
      path: z.string().describe("File or folder path"),
    },
    handleDeleteFile
  );

  srv.tool(
    "replace_all_files",
    "Replace all workspace config files at once",
    {
      workspace_id: z.string(),
      files: z.record(z.string()).describe("Map of file path → content"),
    },
    handleReplaceAllFiles
  );

  srv.tool(
    "get_config",
    "Get workspace runtime config as JSON",
    { workspace_id: z.string() },
    handleGetConfig
  );

  srv.tool(
    "update_config",
    "Update workspace runtime config",
    { workspace_id: z.string(), config: z.record(z.unknown()).describe("Config fields to update") },
    handleUpdateConfig
  );
}
