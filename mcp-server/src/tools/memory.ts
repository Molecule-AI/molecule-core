import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { apiCall, toMcpResult } from "../api.js";

export async function handleCommitMemory(params: {
  workspace_id: string;
  content: string;
  scope: "LOCAL" | "TEAM" | "GLOBAL";
}) {
  const { workspace_id, content, scope } = params;
  const data = await apiCall("POST", `/workspaces/${workspace_id}/memories`, { content, scope });
  return toMcpResult(data);
}

export async function handleSearchMemory(params: {
  workspace_id: string;
  query?: string;
  scope?: "LOCAL" | "TEAM" | "GLOBAL" | "";
}) {
  const { workspace_id, query, scope } = params;
  const urlParams = new URLSearchParams();
  if (query) urlParams.set("q", query);
  if (scope) urlParams.set("scope", scope);
  const data = await apiCall("GET", `/workspaces/${workspace_id}/memories?${urlParams}`);
  return toMcpResult(data);
}

export async function handleDeleteMemory(params: { workspace_id: string; memory_id: string }) {
  const { workspace_id, memory_id } = params;
  const data = await apiCall("DELETE", `/workspaces/${workspace_id}/memories/${memory_id}`);
  return toMcpResult(data);
}

export async function handleSessionSearch(params: {
  workspace_id: string;
  q?: string;
  limit?: number;
}) {
  const { workspace_id, q, limit } = params;
  const qs = new URLSearchParams();
  if (q) qs.set("q", q);
  if (limit) qs.set("limit", String(limit));
  const suffix = qs.toString() ? `?${qs.toString()}` : "";
  const data = await apiCall("GET", `/workspaces/${workspace_id}/session-search${suffix}`);
  return toMcpResult(data);
}

export async function handleGetSharedContext(params: { workspace_id: string }) {
  const data = await apiCall("GET", `/workspaces/${params.workspace_id}/shared-context`);
  return toMcpResult(data);
}

export async function handleSetKV(params: {
  workspace_id: string;
  key: string;
  value: string;
  ttl_seconds?: number;
}) {
  const { workspace_id, ...body } = params;
  const data = await apiCall("POST", `/workspaces/${workspace_id}/memory`, body);
  return toMcpResult(data);
}

export async function handleGetKV(params: { workspace_id: string; key: string }) {
  const data = await apiCall(
    "GET",
    `/workspaces/${params.workspace_id}/memory/${encodeURIComponent(params.key)}`,
  );
  return toMcpResult(data);
}

export async function handleListKV(params: { workspace_id: string }) {
  const data = await apiCall("GET", `/workspaces/${params.workspace_id}/memory`);
  return toMcpResult(data);
}

export async function handleDeleteKV(params: { workspace_id: string; key: string }) {
  const data = await apiCall(
    "DELETE",
    `/workspaces/${params.workspace_id}/memory/${encodeURIComponent(params.key)}`,
  );
  return toMcpResult(data);
}

export function registerMemoryTools(srv: McpServer) {
  srv.tool(
    "commit_memory",
    "Store a fact in workspace memory (LOCAL, TEAM, or GLOBAL scope)",
    {
      workspace_id: z.string().describe("Workspace ID"),
      content: z.string().describe("Fact to remember"),
      scope: z.enum(["LOCAL", "TEAM", "GLOBAL"]).default("LOCAL").describe("Memory scope"),
    },
    handleCommitMemory
  );

  srv.tool(
    "search_memory",
    "Search workspace memories",
    {
      workspace_id: z.string().describe("Workspace ID"),
      query: z.string().optional().describe("Search query"),
      scope: z.enum(["LOCAL", "TEAM", "GLOBAL", ""]).optional().describe("Filter by scope"),
    },
    handleSearchMemory
  );

  srv.tool(
    "delete_memory",
    "Delete a specific memory entry",
    { workspace_id: z.string(), memory_id: z.string() },
    handleDeleteMemory
  );

  srv.tool(
    "session_search",
    "Search a workspace's recent session activity and memory (FTS). Useful for 'did I tell you about X'.",
    {
      workspace_id: z.string(),
      q: z.string().optional(),
      limit: z.number().optional(),
    },
    handleSessionSearch,
  );

  srv.tool(
    "get_shared_context",
    "Get the shared-context blob for a workspace (persistent cross-turn context).",
    { workspace_id: z.string() },
    handleGetSharedContext,
  );

  srv.tool(
    "memory_set",
    "Set a key-value memory entry with optional TTL. Distinct from commit_memory which uses HMA scopes.",
    {
      workspace_id: z.string(),
      key: z.string(),
      value: z.string(),
      ttl_seconds: z.number().optional(),
    },
    handleSetKV,
  );

  srv.tool(
    "memory_get",
    "Read a single K/V memory entry.",
    { workspace_id: z.string(), key: z.string() },
    handleGetKV,
  );

  srv.tool(
    "memory_list",
    "List all K/V memory entries for a workspace.",
    { workspace_id: z.string() },
    handleListKV,
  );

  srv.tool(
    "memory_delete_kv",
    "Delete a single K/V memory entry.",
    { workspace_id: z.string(), key: z.string() },
    handleDeleteKV,
  );
}
