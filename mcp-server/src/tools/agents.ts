import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { apiCall } from "../api.js";

export async function handleChatWithAgent(params: { workspace_id: string; message: string }) {
  const { workspace_id, message } = params;
  const data = await apiCall("POST", `/workspaces/${workspace_id}/a2a`, {
    method: "message/send",
    params: {
      message: { role: "user", parts: [{ type: "text", text: message }] },
    },
  });
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

export async function handleGetModel(params: { workspace_id: string }) {
  const data = await apiCall("GET", `/workspaces/${params.workspace_id}/model`);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export function registerAgentTools(srv: McpServer) {
  srv.tool(
    "chat_with_agent",
    "Send a message to a workspace agent and get a response",
    {
      workspace_id: z.string().describe("Workspace ID"),
      message: z.string().describe("Message to send"),
    },
    handleChatWithAgent
  );

  srv.tool(
    "assign_agent",
    "Assign an AI model to a workspace",
    {
      workspace_id: z.string().describe("Workspace ID"),
      model: z.string().describe("Model string (e.g., openrouter:anthropic/claude-3.5-haiku)"),
    },
    handleAssignAgent
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
    "get_model",
    "Get current model configuration for a workspace",
    { workspace_id: z.string() },
    handleGetModel
  );
}
