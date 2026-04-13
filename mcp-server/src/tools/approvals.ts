import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { apiCall, toMcpResult } from "../api.js";

export async function handleListPendingApprovals() {
  const data = await apiCall("GET", "/approvals/pending");
  return toMcpResult(data);
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
  return toMcpResult(data);
}

export async function handleCreateApproval(params: {
  workspace_id: string;
  action: string;
  reason?: string;
}) {
  const { workspace_id, action, reason } = params;
  const data = await apiCall("POST", `/workspaces/${workspace_id}/approvals`, { action, reason });
  return toMcpResult(data);
}

export async function handleGetWorkspaceApprovals(params: { workspace_id: string }) {
  const data = await apiCall("GET", `/workspaces/${params.workspace_id}/approvals`);
  return toMcpResult(data);
}

export function registerApprovalTools(srv: McpServer) {
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
}
