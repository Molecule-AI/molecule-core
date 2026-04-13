import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { apiCall } from "../api.js";

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

export async function handleNotifyUser(params: {
  workspace_id: string;
  type: string;
  [k: string]: unknown;
}) {
  const { workspace_id, ...body } = params;
  const data = await apiCall("POST", `/workspaces/${workspace_id}/notify`, body);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleListTraces(params: { workspace_id: string }) {
  const data = await apiCall("GET", `/workspaces/${params.workspace_id}/traces`);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export function registerDelegationTools(srv: McpServer) {
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
    "notify_user",
    "Push a notification from the agent to the canvas via WebSocket — appears as a toast / chat bubble.",
    {
      workspace_id: z.string(),
      type: z.string().describe("Notification category (e.g. 'delegation_complete', 'approval_needed')"),
    },
    handleNotifyUser,
  );

  srv.tool(
    "list_traces",
    "List recent LLM traces from Langfuse for a workspace",
    { workspace_id: z.string() },
    handleListTraces
  );
}
