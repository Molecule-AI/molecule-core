import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { apiCall } from "../api.js";

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

export async function handleDiscoverChannelChats(params: {
  type: string;
  config: Record<string, unknown>;
}) {
  const data = await apiCall("POST", "/channels/discover", params);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export function registerChannelTools(srv: McpServer) {
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

  srv.tool(
    "discover_channel_chats",
    "Auto-detect chat IDs / channels for a given bot token (e.g. Telegram). Useful before creating a workspace channel.",
    {
      type: z.string().describe("Channel type (telegram, slack, etc.)"),
      config: z.record(z.unknown()).describe("Adapter-specific config (bot_token, etc.)"),
    },
    handleDiscoverChannelChats,
  );
}
