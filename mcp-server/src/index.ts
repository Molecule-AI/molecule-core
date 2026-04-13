#!/usr/bin/env node
/**
 * Molecule AI MCP Server
 *
 * Exposes Molecule AI platform operations as MCP tools so any AI coding agent
 * (Claude Code, Cursor, Codex, OpenCode) can manage workspaces, agents,
 * skills, and memory.
 *
 * Transport: stdio (for local CLI integration)
 */

import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { z } from "zod";

// Prefer MOLECULE_URL (the canonical MCP env var), fall back to PLATFORM_URL
// (what the workspace runtime already injects for heartbeat/register), and
// only then to localhost:8080. Injecting MOLECULE_URL at container provision
// is handled by platform/internal/provisioner/provisioner.go; this fallback
// chain protects older containers and host-side users alike. Fixes #67.
export const PLATFORM_URL =
  process.env.MOLECULE_URL ||
  process.env.PLATFORM_URL ||
  "http://localhost:8080";

export async function apiCall(method: string, path: string, body?: unknown) {
  try {
    const res = await fetch(`${PLATFORM_URL}${path}`, {
      method,
      headers: { "Content-Type": "application/json" },
      body: body ? JSON.stringify(body) : undefined,
    });
    if (!res.ok) {
      const text = await res.text();
      return { error: `HTTP ${res.status}`, detail: text };
    }
    const text = await res.text();
    try {
      return JSON.parse(text);
    } catch {
      return { raw: text, status: res.status };
    }
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err);
    console.error(`Molecule AI API error (${method} ${path}): ${msg}`);
    return { error: `Platform unreachable at ${PLATFORM_URL}`, detail: msg };
  }
}

// ============================================================
// Tool handler functions (exported for unit testing)
// ============================================================

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
  workspace_access?: "none" | "read_only" | "read_write"; // #65
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

export async function handleChatWithAgent(params: { workspace_id: string; message: string }) {
  const { workspace_id, message } = params;
  const data = await apiCall("POST", `/workspaces/${workspace_id}/a2a`, {
    method: "message/send",
    params: {
      message: { role: "user", parts: [{ type: "text", text: message }] },
    },
  });
  // Extract text from response
  const parts = data?.result?.parts || [];
  const text = parts
    .filter((p: { kind?: string }) => p.kind === "text")
    .map((p: { text?: string }) => p.text || "")
    .join("\n");
  return { content: [{ type: "text" as const, text: text || JSON.stringify(data, null, 2) }] };
}

export async function handleAssignAgent(params: { workspace_id: string; model: string }) {
  const { workspace_id, model } = params;
  const data = await apiCall("POST", `/workspaces/${workspace_id}/agent`, { model });
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleSetSecret(params: { workspace_id: string; key: string; value: string }) {
  const { workspace_id, key, value } = params;
  const data = await apiCall("POST", `/workspaces/${workspace_id}/secrets`, { key, value });
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleListSecrets(params: { workspace_id: string }) {
  const data = await apiCall("GET", `/workspaces/${params.workspace_id}/secrets`);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleListFiles(params: { workspace_id: string }) {
  const data = await apiCall("GET", `/workspaces/${params.workspace_id}/files`);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleReadFile(params: { workspace_id: string; path: string }) {
  const { workspace_id, path } = params;
  const data = await apiCall("GET", `/workspaces/${workspace_id}/files/${path}`);
  return { content: [{ type: "text" as const, text: data?.content || JSON.stringify(data) }] };
}

export async function handleWriteFile(params: { workspace_id: string; path: string; content: string }) {
  const { workspace_id, path, content } = params;
  const data = await apiCall("PUT", `/workspaces/${workspace_id}/files/${path}`, { content });
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleDeleteFile(params: { workspace_id: string; path: string }) {
  const { workspace_id, path } = params;
  const data = await apiCall("DELETE", `/workspaces/${workspace_id}/files/${path}`);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleCommitMemory(params: {
  workspace_id: string;
  content: string;
  scope: "LOCAL" | "TEAM" | "GLOBAL";
}) {
  const { workspace_id, content, scope } = params;
  const data = await apiCall("POST", `/workspaces/${workspace_id}/memories`, { content, scope });
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
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
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleListTemplates() {
  const data = await apiCall("GET", "/templates");
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleExpandTeam(params: { workspace_id: string }) {
  const data = await apiCall("POST", `/workspaces/${params.workspace_id}/expand`, {});
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleCollapseTeam(params: { workspace_id: string }) {
  const data = await apiCall("POST", `/workspaces/${params.workspace_id}/collapse`, {});
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleListPendingApprovals() {
  const data = await apiCall("GET", "/approvals/pending");
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleDecideApproval(params: {
  workspace_id: string;
  approval_id: string;
  decision: "approved" | "denied";
}) {
  const { workspace_id, approval_id, decision } = params;
  const data = await apiCall(
    "POST",
    `/workspaces/${workspace_id}/approvals/${approval_id}/decide`,
    { decision, decided_by: "mcp-client" }
  );
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleUpdateWorkspace(params: {
  workspace_id: string;
  name?: string;
  role?: string;
  tier?: number;
  parent_id?: string | null;
  workspace_dir?: string;
  workspace_access?: "none" | "read_only" | "read_write"; // #65
}) {
  const { workspace_id, ...fields } = params;
  const data = await apiCall("PATCH", `/workspaces/${workspace_id}`, fields);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleReplaceAgent(params: { workspace_id: string; model: string }) {
  const { workspace_id, model } = params;
  const data = await apiCall("PATCH", `/workspaces/${workspace_id}/agent`, { model });
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleRemoveAgent(params: { workspace_id: string }) {
  const data = await apiCall("DELETE", `/workspaces/${params.workspace_id}/agent`);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleMoveAgent(params: { workspace_id: string; target_workspace_id: string }) {
  const { workspace_id, target_workspace_id } = params;
  const data = await apiCall("POST", `/workspaces/${workspace_id}/agent/move`, { target_workspace_id });
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleDeleteSecret(params: { workspace_id: string; key: string }) {
  const { workspace_id, key } = params;
  const data = await apiCall("DELETE", `/workspaces/${workspace_id}/secrets/${encodeURIComponent(key)}`);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleGetConfig(params: { workspace_id: string }) {
  const data = await apiCall("GET", `/workspaces/${params.workspace_id}/config`);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleUpdateConfig(params: { workspace_id: string; config: Record<string, unknown> }) {
  const { workspace_id, config } = params;
  const data = await apiCall("PATCH", `/workspaces/${workspace_id}/config`, config);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleListPeers(params: { workspace_id: string }) {
  const data = await apiCall("GET", `/registry/${params.workspace_id}/peers`);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleDiscoverWorkspace(params: { workspace_id: string }) {
  const data = await apiCall("GET", `/registry/discover/${params.workspace_id}`);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleCheckAccess(params: { caller_id: string; target_id: string }) {
  const { caller_id, target_id } = params;
  const data = await apiCall("POST", `/registry/check-access`, { caller_id, target_id });
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleListEvents(params: { workspace_id?: string }) {
  const path = params.workspace_id ? `/events/${params.workspace_id}` : "/events";
  const data = await apiCall("GET", path);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleExportBundle(params: { workspace_id: string }) {
  const data = await apiCall("GET", `/bundles/export/${params.workspace_id}`);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleImportBundle(params: { bundle: Record<string, unknown> }) {
  const data = await apiCall("POST", `/bundles/import`, params.bundle);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleImportTemplate(params: { name: string; files: Record<string, string> }) {
  const { name, files } = params;
  const data = await apiCall("POST", `/templates/import`, { name, files });
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleReplaceAllFiles(params: {
  workspace_id: string;
  files: Record<string, string>;
}) {
  const { workspace_id, files } = params;
  const data = await apiCall("PUT", `/workspaces/${workspace_id}/files`, { files });
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleListTraces(params: { workspace_id: string }) {
  const data = await apiCall("GET", `/workspaces/${params.workspace_id}/traces`);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleListActivity(params: {
  workspace_id: string;
  type?: "a2a_receive" | "a2a_send" | "task_update" | "agent_log" | "error";
  limit?: number;
}) {
  const { workspace_id, type, limit } = params;
  const urlParams = new URLSearchParams();
  if (type) urlParams.set("type", type);
  if (limit) urlParams.set("limit", String(limit));
  const qs = urlParams.toString() ? `?${urlParams.toString()}` : "";
  const data = await apiCall("GET", `/workspaces/${workspace_id}/activity${qs}`);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleDeleteMemory(params: { workspace_id: string; memory_id: string }) {
  const { workspace_id, memory_id } = params;
  const data = await apiCall("DELETE", `/workspaces/${workspace_id}/memories/${memory_id}`);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleGetModel(params: { workspace_id: string }) {
  const data = await apiCall("GET", `/workspaces/${params.workspace_id}/model`);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleCreateApproval(params: {
  workspace_id: string;
  action: string;
  reason?: string;
}) {
  const { workspace_id, action, reason } = params;
  const data = await apiCall("POST", `/workspaces/${workspace_id}/approvals`, { action, reason });
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleGetWorkspaceApprovals(params: { workspace_id: string }) {
  const data = await apiCall("GET", `/workspaces/${params.workspace_id}/approvals`);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

// === PLUGINS ===

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

// === GLOBAL SECRETS ===

export async function handleListGlobalSecrets() {
  const data = await apiCall("GET", "/settings/secrets");
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleSetGlobalSecret(params: { key: string; value: string }) {
  const { key, value } = params;
  const data = await apiCall("PUT", "/settings/secrets", { key, value });
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleDeleteGlobalSecret(params: { key: string }) {
  const data = await apiCall("DELETE", `/settings/secrets/${params.key}`);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

// === PAUSE / RESUME ===

export async function handlePauseWorkspace(params: { workspace_id: string }) {
  const data = await apiCall("POST", `/workspaces/${params.workspace_id}/pause`, {});
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleResumeWorkspace(params: { workspace_id: string }) {
  const data = await apiCall("POST", `/workspaces/${params.workspace_id}/resume`, {});
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

// === ORG TEMPLATES ===

export async function handleListOrgTemplates() {
  const data = await apiCall("GET", "/org/templates");
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleImportOrg(params: { dir: string }) {
  const data = await apiCall("POST", "/org/import", { dir: params.dir });
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

// === ASYNC DELEGATION ===

// === CHANNEL HANDLERS ===

export async function handleListChannelAdapters() {
  const data = await apiCall("GET", `/channels/adapters`);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleListChannels(params: { workspace_id: string }) {
  const data = await apiCall("GET", `/workspaces/${params.workspace_id}/channels`);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleAddChannel(params: {
  workspace_id: string;
  channel_type: string;
  config: string;
  allowed_users?: string;
}) {
  let config: unknown;
  try { config = JSON.parse(params.config); } catch { return { content: [{ type: "text" as const, text: "Error: config is not valid JSON" }] }; }
  const allowed_users = params.allowed_users ? params.allowed_users.split(",").map((s) => s.trim()).filter(Boolean) : [];
  const data = await apiCall("POST", `/workspaces/${params.workspace_id}/channels`, {
    channel_type: params.channel_type,
    config,
    allowed_users,
  });
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleUpdateChannel(params: {
  workspace_id: string;
  channel_id: string;
  config?: string;
  enabled?: boolean;
  allowed_users?: string;
}) {
  const body: Record<string, unknown> = {};
  if (params.config) {
    try { body.config = JSON.parse(params.config); } catch { return { content: [{ type: "text" as const, text: "Error: config is not valid JSON" }] }; }
  }
  if (params.enabled !== undefined) body.enabled = params.enabled;
  if (params.allowed_users !== undefined) {
    body.allowed_users = params.allowed_users.split(",").map((s) => s.trim()).filter(Boolean);
  }
  const data = await apiCall("PATCH", `/workspaces/${params.workspace_id}/channels/${params.channel_id}`, body);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleRemoveChannel(params: { workspace_id: string; channel_id: string }) {
  const data = await apiCall("DELETE", `/workspaces/${params.workspace_id}/channels/${params.channel_id}`);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleSendChannelMessage(params: {
  workspace_id: string;
  channel_id: string;
  text: string;
}) {
  const data = await apiCall("POST", `/workspaces/${params.workspace_id}/channels/${params.channel_id}/send`, {
    text: params.text,
  });
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleTestChannel(params: { workspace_id: string; channel_id: string }) {
  const data = await apiCall("POST", `/workspaces/${params.workspace_id}/channels/${params.channel_id}/test`, {});
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

// === DELEGATION HANDLERS ===

export async function handleAsyncDelegate(params: {
  workspace_id: string;
  target_id: string;
  task: string;
}) {
  const { workspace_id, target_id, task } = params;
  const data = await apiCall("POST", `/workspaces/${workspace_id}/delegate`, { target_id, task });
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleCheckDelegations(params: { workspace_id: string }) {
  const data = await apiCall("GET", `/workspaces/${params.workspace_id}/delegations`);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

// ============================================================
// Coverage completion — additions to reach 100% platform parity
// ============================================================

// --- Delegations (#64: Record/UpdateStatus) ---

export async function handleRecordDelegation(params: {
  workspace_id: string;
  target_id: string;
  task: string;
  delegation_id: string;
}) {
  const { workspace_id, ...body } = params;
  const data = await apiCall("POST", `/workspaces/${workspace_id}/delegations/record`, body);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleUpdateDelegationStatus(params: {
  workspace_id: string;
  delegation_id: string;
  status: "completed" | "failed";
  error?: string;
  response_preview?: string;
}) {
  const { workspace_id, delegation_id, ...body } = params;
  const data = await apiCall(
    "POST",
    `/workspaces/${workspace_id}/delegations/${delegation_id}/update`,
    body,
  );
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

// --- Activity (POST /activity, POST /notify) ---

export async function handleReportActivity(params: {
  workspace_id: string;
  activity_type: string;
  method?: string;
  summary?: string;
  status?: string;
  error_detail?: string;
  request_body?: unknown;
  response_body?: unknown;
  duration_ms?: number;
}) {
  const { workspace_id, ...body } = params;
  const data = await apiCall("POST", `/workspaces/${workspace_id}/activity`, body);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleNotifyUser(params: {
  workspace_id: string;
  type: string;
  [k: string]: unknown;
}) {
  const { workspace_id, ...body } = params;
  const data = await apiCall("POST", `/workspaces/${workspace_id}/notify`, body);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

// --- Canvas viewport ---

export async function handleGetViewport() {
  const data = await apiCall("GET", "/canvas/viewport");
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleSetViewport(params: { x: number; y: number; zoom: number }) {
  const data = await apiCall("PUT", "/canvas/viewport", params);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

// --- Channels (platform-wide discover) ---

export async function handleDiscoverChannelChats(params: {
  type: string;
  config: Record<string, unknown>;
}) {
  const data = await apiCall("POST", "/channels/discover", params);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

// --- Plugins (sources, available, compatibility) ---

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

// --- Schedules (cron) ---

export async function handleListSchedules(params: { workspace_id: string }) {
  const data = await apiCall("GET", `/workspaces/${params.workspace_id}/schedules`);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleCreateSchedule(params: {
  workspace_id: string;
  name: string;
  cron_expr: string;
  prompt: string;
  timezone?: string;
  enabled?: boolean;
}) {
  const { workspace_id, ...body } = params;
  const data = await apiCall("POST", `/workspaces/${workspace_id}/schedules`, body);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleUpdateSchedule(params: {
  workspace_id: string;
  schedule_id: string;
  name?: string;
  cron_expr?: string;
  prompt?: string;
  timezone?: string;
  enabled?: boolean;
}) {
  const { workspace_id, schedule_id, ...body } = params;
  const data = await apiCall(
    "PATCH",
    `/workspaces/${workspace_id}/schedules/${schedule_id}`,
    body,
  );
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleDeleteSchedule(params: {
  workspace_id: string;
  schedule_id: string;
}) {
  const data = await apiCall(
    "DELETE",
    `/workspaces/${params.workspace_id}/schedules/${params.schedule_id}`,
  );
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleRunSchedule(params: {
  workspace_id: string;
  schedule_id: string;
}) {
  const data = await apiCall(
    "POST",
    `/workspaces/${params.workspace_id}/schedules/${params.schedule_id}/run`,
  );
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleGetScheduleHistory(params: {
  workspace_id: string;
  schedule_id: string;
}) {
  const data = await apiCall(
    "GET",
    `/workspaces/${params.workspace_id}/schedules/${params.schedule_id}/history`,
  );
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

// --- Session search + shared context ---

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
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleGetSharedContext(params: { workspace_id: string }) {
  const data = await apiCall("GET", `/workspaces/${params.workspace_id}/shared-context`);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

// --- K/V memory (per-workspace key-value, separate from HMA scope memories) ---

export async function handleSetKV(params: {
  workspace_id: string;
  key: string;
  value: string;
  ttl_seconds?: number;
}) {
  const { workspace_id, ...body } = params;
  const data = await apiCall("POST", `/workspaces/${workspace_id}/memory`, body);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleGetKV(params: { workspace_id: string; key: string }) {
  const data = await apiCall(
    "GET",
    `/workspaces/${params.workspace_id}/memory/${encodeURIComponent(params.key)}`,
  );
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleListKV(params: { workspace_id: string }) {
  const data = await apiCall("GET", `/workspaces/${params.workspace_id}/memory`);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleDeleteKV(params: { workspace_id: string; key: string }) {
  const data = await apiCall(
    "DELETE",
    `/workspaces/${params.workspace_id}/memory/${encodeURIComponent(params.key)}`,
  );
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

// ============================================================
// Phase 30 — Remote agent management handlers
// ============================================================

// Fetch the workspace list, filter to runtime='external'. The platform
// has no dedicated /remote-agents endpoint — we filter client-side
// because the workspace list is small (tens to low-hundreds, never
// pagination scale) and adding a server endpoint would be a separate PR.
export async function handleListRemoteAgents() {
  const data = await apiCall("GET", "/workspaces");
  if (!Array.isArray(data)) {
    return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
  }
  const remote = data
    .filter((w: { runtime?: string }) => w.runtime === "external")
    .map((w: Record<string, unknown>) => ({
      id: w.id,
      name: w.name,
      status: w.status,
      url: w.url,
      last_heartbeat_at: w.last_heartbeat_at,
      uptime_seconds: w.uptime_seconds,
      tier: w.tier,
    }));
  return { content: [{ type: "text" as const, text: JSON.stringify({ count: remote.length, agents: remote }, null, 2) }] };
}

// Phase 30.4 — token-gated; from MCP we don't have a workspace bearer
// (we're an operator surface), so we hit the lightweight unauthenticated
// /workspaces/:id endpoint and project the same shape. Still useful as
// a focused tool that doesn't dump the full workspace blob.
export async function handleGetRemoteAgentState(params: { workspace_id: string }) {
  const data = await apiCall("GET", `/workspaces/${params.workspace_id}`);
  if (data && typeof data === "object" && "error" in data) {
    return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
  }
  const w = data as Record<string, unknown>;
  const projected = {
    workspace_id: w.id,
    status: w.status,
    paused: w.status === "paused",
    deleted: w.status === "removed",
    runtime: w.runtime,
    last_heartbeat_at: w.last_heartbeat_at,
  };
  return { content: [{ type: "text" as const, text: JSON.stringify(projected, null, 2) }] };
}

export async function handleGetRemoteAgentSetupCommand(params: {
  workspace_id: string;
  platform_url_override?: string;
}) {
  // Verify the workspace exists and is runtime='external' before generating
  // the command — saves the operator from pasting a bash line that will
  // fail because the workspace was a Docker workspace they typed by mistake.
  const ws = await apiCall("GET", `/workspaces/${params.workspace_id}`);
  if (ws && typeof ws === "object" && "error" in ws) {
    return { content: [{ type: "text" as const, text: JSON.stringify(ws, null, 2) }] };
  }
  const w = ws as { id: string; name: string; runtime?: string };
  if (w.runtime !== "external") {
    return {
      content: [{
        type: "text" as const,
        text: JSON.stringify({
          error: "workspace is not external; setup command only applies to runtime='external'",
          workspace_id: w.id,
          actual_runtime: w.runtime,
        }, null, 2),
      }],
    };
  }

  // The MCP server's PLATFORM_URL is whatever Claude Desktop / the host
  // injected — usually localhost when an operator runs us locally. That
  // URL is useless inside a remote-agent shell on a different machine.
  // If the caller passes platform_url_override we use it; otherwise we
  // detect localhost and surface a warning so the operator knows to
  // substitute the real public URL before pasting the command.
  const targetUrl = params.platform_url_override?.trim() || PLATFORM_URL;
  const isLocalhost = /^https?:\/\/(localhost|127\.0\.0\.1|0\.0\.0\.0)(:|\/|$)/.test(targetUrl);
  const warnings: string[] = [];
  if (isLocalhost && !params.platform_url_override) {
    warnings.push(
      `PLATFORM_URL is ${targetUrl} — this only works if the remote agent is on the same machine as the platform. ` +
      `Pass platform_url_override with the agent-reachable URL (e.g. https://your-platform.example.com) before pasting on a different host.`
    );
  }

  const setupCmd = [
    `# Run on the remote machine where the agent will live.`,
    `# Requires Python 3.11+ and bash (the SDK invokes setup.sh via bash).`,
    `pip install molecule-sdk  # (or: pip install -e <molecule-checkout>/sdk/python)`,
    ``,
    `WORKSPACE_ID=${w.id} \\`,
    `PLATFORM_URL=${targetUrl} \\`,
    `python3 -m examples.remote-agent.run`,
    ``,
    `# The agent will register, mint its bearer token (cached at`,
    `# ~/.molecule/${w.id}/.auth_token), pull secrets, then heartbeat.`,
  ].join("\n");
  return {
    content: [{
      type: "text" as const,
      text: JSON.stringify({
        workspace_id: w.id,
        workspace_name: w.name,
        platform_url: targetUrl,
        setup_command: setupCmd,
        ...(warnings.length > 0 ? { warnings } : {}),
      }, null, 2),
    }],
  };
}

export async function handleCheckRemoteAgentFreshness(params: {
  workspace_id: string;
  threshold_seconds?: number;
}) {
  const ws = await apiCall("GET", `/workspaces/${params.workspace_id}`);
  if (ws && typeof ws === "object" && "error" in ws) {
    return { content: [{ type: "text" as const, text: JSON.stringify(ws, null, 2) }] };
  }
  const w = ws as { last_heartbeat_at?: string; status?: string; runtime?: string };
  const threshold = params.threshold_seconds ?? 90;
  const heartbeatStr = w.last_heartbeat_at;
  let secondsSince: number | null = null;
  if (heartbeatStr) {
    const heartbeatMs = Date.parse(heartbeatStr);
    if (!isNaN(heartbeatMs)) {
      secondsSince = Math.floor((Date.now() - heartbeatMs) / 1000);
    }
  }
  const fresh = secondsSince !== null && secondsSince <= threshold;
  return {
    content: [{
      type: "text" as const,
      text: JSON.stringify({
        workspace_id: params.workspace_id,
        status: w.status,
        runtime: w.runtime,
        last_heartbeat_at: heartbeatStr,
        seconds_since_heartbeat: secondsSince,
        threshold_seconds: threshold,
        fresh,
      }, null, 2),
    }],
  };
}

// ============================================================
// MCP Server registration
// ============================================================

export function createServer() {
  const srv = new McpServer({
    name: "molecule",
    version: "1.0.0",
  });

  // === WORKSPACE TOOLS ===

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

  // === CHAT / A2A ===

  srv.tool(
    "chat_with_agent",
    "Send a message to a workspace agent and get a response",
    {
      workspace_id: z.string().describe("Workspace ID"),
      message: z.string().describe("Message to send"),
    },
    handleChatWithAgent
  );

  // === AGENT MANAGEMENT ===

  srv.tool(
    "assign_agent",
    "Assign an AI model to a workspace",
    {
      workspace_id: z.string().describe("Workspace ID"),
      model: z.string().describe("Model string (e.g., openrouter:anthropic/claude-3.5-haiku)"),
    },
    handleAssignAgent
  );

  // === SECRETS ===

  srv.tool(
    "set_secret",
    "Set an API key or environment variable for a workspace",
    {
      workspace_id: z.string().describe("Workspace ID"),
      key: z.string().describe("Secret key (e.g., ANTHROPIC_API_KEY)"),
      value: z.string().describe("Secret value"),
    },
    handleSetSecret
  );

  srv.tool(
    "list_secrets",
    "List secret keys for a workspace (values never exposed)",
    { workspace_id: z.string().describe("Workspace ID") },
    handleListSecrets
  );

  // === FILES ===

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

  // === MEMORY (HMA) ===

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

  // === TEMPLATES ===

  srv.tool("list_templates", "List available workspace templates", {}, handleListTemplates);

  // === TEAM EXPANSION ===

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

  // === APPROVALS ===

  srv.tool(
    "list_pending_approvals",
    "List all pending approval requests across workspaces",
    {},
    handleListPendingApprovals
  );

  srv.tool(
    "decide_approval",
    "Approve or deny a pending approval request",
    {
      workspace_id: z.string().describe("Workspace ID"),
      approval_id: z.string().describe("Approval ID"),
      decision: z.enum(["approved", "denied"]).describe("Decision"),
    },
    handleDecideApproval
  );

  // === MISSING TOOLS — FULL COVERAGE ===

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
    "replace_agent",
    "Replace the model on an existing workspace agent",
    { workspace_id: z.string(), model: z.string() },
    handleReplaceAgent
  );

  srv.tool(
    "remove_agent",
    "Remove the agent from a workspace",
    { workspace_id: z.string() },
    handleRemoveAgent
  );

  srv.tool(
    "move_agent",
    "Move an agent from one workspace to another",
    { workspace_id: z.string(), target_workspace_id: z.string() },
    handleMoveAgent
  );

  srv.tool(
    "delete_secret",
    "Delete a secret from a workspace",
    { workspace_id: z.string(), key: z.string() },
    handleDeleteSecret
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
    "import_template",
    "Import agent files as a new workspace template",
    {
      name: z.string().describe("Template name"),
      files: z.record(z.string()).describe("Map of file path → content"),
    },
    handleImportTemplate
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
    "list_traces",
    "List recent LLM traces from Langfuse for a workspace",
    { workspace_id: z.string() },
    handleListTraces
  );

  srv.tool(
    "list_activity",
    "List activity logs for a workspace (A2A communications, tasks, errors)",
    {
      workspace_id: z.string(),
      type: z
        .enum(["a2a_receive", "a2a_send", "task_update", "agent_log", "error"])
        .optional()
        .describe("Filter by activity type"),
      limit: z.number().optional().describe("Max entries to return (default 100, max 500)"),
    },
    handleListActivity
  );

  srv.tool(
    "delete_memory",
    "Delete a specific memory entry",
    { workspace_id: z.string(), memory_id: z.string() },
    handleDeleteMemory
  );

  srv.tool(
    "get_model",
    "Get current model configuration for a workspace",
    { workspace_id: z.string() },
    handleGetModel
  );

  srv.tool(
    "create_approval",
    "Create an approval request for a workspace",
    {
      workspace_id: z.string(),
      action: z.string().describe("What needs approval"),
      reason: z.string().optional().describe("Why it's needed"),
    },
    handleCreateApproval
  );

  srv.tool(
    "get_workspace_approvals",
    "List approval requests for a specific workspace",
    { workspace_id: z.string() },
    handleGetWorkspaceApprovals
  );

  // === PLUGINS ===

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

  // === GLOBAL SECRETS ===

  srv.tool("list_global_secrets", "List global secret keys (values never exposed)", {}, handleListGlobalSecrets);

  srv.tool(
    "set_global_secret",
    "Set a global secret (available to all workspaces)",
    {
      key: z.string().describe("Secret key (e.g., GITHUB_TOKEN)"),
      value: z.string().describe("Secret value"),
    },
    handleSetGlobalSecret
  );

  srv.tool(
    "delete_global_secret",
    "Delete a global secret",
    { key: z.string().describe("Secret key") },
    handleDeleteGlobalSecret
  );

  // === PAUSE / RESUME ===

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

  // === ORG TEMPLATES ===

  srv.tool("list_org_templates", "List available org templates", {}, handleListOrgTemplates);

  srv.tool(
    "import_org",
    "Import an org template to create an entire workspace hierarchy",
    { dir: z.string().describe("Org template directory name (e.g., 'molecule-dev')") },
    handleImportOrg
  );

  // === SOCIAL CHANNELS ===

  srv.tool("list_channel_adapters", "List available social channel adapters (Telegram, Slack, etc.)", {}, handleListChannelAdapters);

  srv.tool("list_channels", "List social channels connected to a workspace", {
    workspace_id: z.string().describe("Workspace ID"),
  }, handleListChannels);

  srv.tool(
    "add_channel",
    "Connect a social channel (Telegram, Slack, etc.) to a workspace. Messages on the channel will be forwarded to the agent.",
    {
      workspace_id: z.string().describe("Workspace ID"),
      channel_type: z.string().describe("Channel type (e.g., 'telegram')"),
      config: z.string().describe('Channel config as JSON string (e.g., \'{"bot_token":"123:ABC","chat_id":"-100"}\')'),
      allowed_users: z.string().optional().describe("Comma-separated user IDs allowed to message (empty = allow all)"),
    },
    handleAddChannel
  );

  srv.tool(
    "update_channel",
    "Update a social channel's config, enabled state, or allowed users. Triggers hot reload.",
    {
      workspace_id: z.string().describe("Workspace ID"),
      channel_id: z.string().describe("Channel ID"),
      config: z.string().optional().describe("Updated config as JSON string"),
      enabled: z.boolean().optional().describe("Enable or disable the channel"),
      allowed_users: z.string().optional().describe("Comma-separated user IDs (replaces existing list)"),
    },
    handleUpdateChannel
  );

  srv.tool("remove_channel", "Remove a social channel from a workspace", {
    workspace_id: z.string().describe("Workspace ID"),
    channel_id: z.string().describe("Channel ID"),
  }, handleRemoveChannel);

  srv.tool(
    "send_channel_message",
    "Send an outbound message from a workspace to its connected social channel (e.g., proactive Telegram message).",
    {
      workspace_id: z.string().describe("Workspace ID"),
      channel_id: z.string().describe("Channel ID"),
      text: z.string().describe("Message text to send"),
    },
    handleSendChannelMessage
  );

  srv.tool("test_channel", "Send a test message to verify a social channel connection works", {
    workspace_id: z.string().describe("Workspace ID"),
    channel_id: z.string().describe("Channel ID"),
  }, handleTestChannel);

  // === ASYNC DELEGATION ===

  srv.tool(
    "async_delegate",
    "Delegate a task to another workspace (non-blocking). Returns immediately with a delegation_id. The target workspace processes the task in the background. Use check_delegations to poll for results.",
    {
      workspace_id: z.string().describe("Source workspace ID (the delegator)"),
      target_id: z.string().describe("Target workspace ID to delegate to"),
      task: z.string().describe("Task description to send"),
    },
    handleAsyncDelegate
  );

  srv.tool(
    "check_delegations",
    "Check status of delegated tasks for a workspace. Returns recent delegations with their status (pending/completed/failed) and results.",
    { workspace_id: z.string().describe("Workspace ID") },
    handleCheckDelegations
  );

  // ====================================================
  // Coverage completion — 100% platform parity (#64/#65 + gaps)
  // ====================================================

  // Delegations (#64): agents mirror direct-A2A delegations to activity_logs
  srv.tool(
    "record_delegation",
    "Register an agent-initiated delegation with the platform's activity log. Used by agent tooling so GET /delegations sees the same set as check_delegation_status.",
    {
      workspace_id: z.string().describe("Source workspace ID (the delegator)"),
      target_id: z.string().describe("Target workspace ID (the delegate)"),
      task: z.string().describe("Task description sent to the target"),
      delegation_id: z.string().describe("Agent-generated task_id to correlate with local state"),
    },
    handleRecordDelegation,
  );

  srv.tool(
    "update_delegation_status",
    "Mirror an agent-initiated delegation's status to activity_logs (completed or failed).",
    {
      workspace_id: z.string().describe("Source workspace ID"),
      delegation_id: z.string().describe("Delegation ID previously registered via record_delegation"),
      status: z.enum(["completed", "failed"]),
      error: z.string().optional(),
      response_preview: z.string().optional().describe("Response text (truncated to 500 chars server-side)"),
    },
    handleUpdateDelegationStatus,
  );

  // Activity
  srv.tool(
    "report_activity",
    "Write an arbitrary activity log row from an agent (a2a events, tool calls, errors).",
    {
      workspace_id: z.string(),
      activity_type: z.string().describe("a2a_receive / a2a_send / tool_call / task_complete / error / ..."),
      method: z.string().optional(),
      summary: z.string().optional(),
      status: z.string().optional().describe("ok / error / pending"),
      error_detail: z.string().optional(),
      request_body: z.unknown().optional(),
      response_body: z.unknown().optional(),
      duration_ms: z.number().optional(),
    },
    handleReportActivity,
  );

  srv.tool(
    "notify_user",
    "Push a notification from the agent to the canvas via WebSocket — appears as a toast / chat bubble.",
    {
      workspace_id: z.string(),
      type: z.string().describe("Notification category (e.g. 'delegation_complete', 'approval_needed')"),
    },
    handleNotifyUser,
  );

  // Canvas viewport
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

  // Channel platform-level discovery
  srv.tool(
    "discover_channel_chats",
    "Auto-detect chat IDs / channels for a given bot token (e.g. Telegram). Useful before creating a workspace channel.",
    {
      type: z.string().describe("Channel type (telegram, slack, etc.)"),
      config: z.record(z.unknown()).describe("Adapter-specific config (bot_token, etc.)"),
    },
    handleDiscoverChannelChats,
  );

  // Plugins — sources + per-workspace availability + compatibility
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

  // Schedules (cron)
  srv.tool(
    "list_schedules",
    "List cron schedules for a workspace.",
    { workspace_id: z.string() },
    handleListSchedules,
  );

  srv.tool(
    "create_schedule",
    "Create a cron schedule that fires a prompt on a recurring timer.",
    {
      workspace_id: z.string(),
      name: z.string(),
      cron_expr: z.string().describe("5-field cron (e.g. '0 9 * * 1-5')"),
      prompt: z.string(),
      timezone: z.string().optional(),
      enabled: z.boolean().optional(),
    },
    handleCreateSchedule,
  );

  srv.tool(
    "update_schedule",
    "Update fields on an existing schedule.",
    {
      workspace_id: z.string(),
      schedule_id: z.string(),
      name: z.string().optional(),
      cron_expr: z.string().optional(),
      prompt: z.string().optional(),
      timezone: z.string().optional(),
      enabled: z.boolean().optional(),
    },
    handleUpdateSchedule,
  );

  srv.tool(
    "delete_schedule",
    "Delete a schedule.",
    { workspace_id: z.string(), schedule_id: z.string() },
    handleDeleteSchedule,
  );

  srv.tool(
    "run_schedule",
    "Fire a schedule manually, bypassing its cron expression.",
    { workspace_id: z.string(), schedule_id: z.string() },
    handleRunSchedule,
  );

  srv.tool(
    "get_schedule_history",
    "Get past runs of a schedule — status, start/end, output preview.",
    { workspace_id: z.string(), schedule_id: z.string() },
    handleGetScheduleHistory,
  );

  // Session search + shared context
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

  // K/V memory
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

  // ==========================================================
  // Phase 30 — Remote agent management (SaaS surface)
  // ==========================================================
  srv.tool(
    "list_remote_agents",
    "List all workspaces with runtime='external' (Phase 30 remote agents). Returns id, name, status, last_heartbeat_at, url. Useful for spotting offline remote agents from a Claude session.",
    {},
    handleListRemoteAgents,
  );

  srv.tool(
    "get_remote_agent_state",
    "Phase 30.4 lightweight state poll for a remote workspace. Returns {status, paused, deleted}. Faster than get_workspace because it doesn't include config/agent_card. Useful when you only need to know whether a remote agent is alive.",
    { workspace_id: z.string() },
    handleGetRemoteAgentState,
  );

  srv.tool(
    "get_remote_agent_setup_command",
    "Build a one-shot bash command an operator can paste into a remote machine to register an agent against this Molecule AI platform. Returns a string like `WORKSPACE_ID=... PLATFORM_URL=... python3 -m molecule_agent.bootstrap`. Pass platform_url_override when the MCP server's PLATFORM_URL is localhost (the agent will live on a different host and needs the platform's public URL). The workspace must exist and be runtime='external'.",
    {
      workspace_id: z.string(),
      platform_url_override: z.string().optional(),
    },
    handleGetRemoteAgentSetupCommand,
  );

  srv.tool(
    "check_remote_agent_freshness",
    "Compare a remote workspace's last_heartbeat_at against now. Returns {seconds_since_heartbeat, fresh, threshold_seconds} where `fresh` is true if the agent heartbeated within the platform's stale-after window. Useful for pre-flight checks before delegating work.",
    { workspace_id: z.string(), threshold_seconds: z.number().optional() },
    handleCheckRemoteAgentFreshness,
  );

  return srv;
}

// ============================================================
// Main entry point — only runs when executed directly
// ============================================================

async function main() {
  // Validate platform connectivity on startup
  try {
    const res = await fetch(`${PLATFORM_URL}/health`);
    if (res.ok) {
      console.error(`Molecule AI platform connected: ${PLATFORM_URL}`);
    } else {
      console.error(`WARNING: Molecule AI platform at ${PLATFORM_URL} returned ${res.status}. Tools may fail.`);
    }
  } catch {
    console.error(`WARNING: Cannot reach Molecule AI platform at ${PLATFORM_URL}. Start it with: cd platform && go run ./cmd/server`);
  }

  const server = createServer();
  const transport = new StdioServerTransport();
  await server.connect(transport);
  console.error("Molecule AI MCP server running on stdio (61 tools available)");
}

// Only auto-start when run directly (not when imported for testing).
// JEST_WORKER_ID is set automatically by Jest in every worker process.
if (!process.env.JEST_WORKER_ID) {
  main().catch(console.error);
}
