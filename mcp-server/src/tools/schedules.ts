import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { apiCall, toMcpResult } from "../api.js";

export async function handleListSchedules(params: { workspace_id: string }) {
  const data = await apiCall("GET", `/workspaces/${params.workspace_id}/schedules`);
  return toMcpResult(data);
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
  return toMcpResult(data);
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
  return toMcpResult(data);
}

export async function handleDeleteSchedule(params: {
  workspace_id: string;
  schedule_id: string;
}) {
  const data = await apiCall(
    "DELETE",
    `/workspaces/${params.workspace_id}/schedules/${params.schedule_id}`,
  );
  return toMcpResult(data);
}

export async function handleRunSchedule(params: {
  workspace_id: string;
  schedule_id: string;
}) {
  const data = await apiCall(
    "POST",
    `/workspaces/${params.workspace_id}/schedules/${params.schedule_id}/run`,
  );
  return toMcpResult(data);
}

export async function handleGetScheduleHistory(params: {
  workspace_id: string;
  schedule_id: string;
}) {
  const data = await apiCall(
    "GET",
    `/workspaces/${params.workspace_id}/schedules/${params.schedule_id}/history`,
  );
  return toMcpResult(data);
}

export function registerScheduleTools(srv: McpServer) {
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
}
