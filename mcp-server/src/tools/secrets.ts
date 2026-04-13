import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { apiCall } from "../api.js";

export async function handleSetSecret(params: { workspace_id: string; key: string; value: string }) {
  const { workspace_id, key, value } = params;
  const data = await apiCall("POST", `/workspaces/${workspace_id}/secrets`, { key, value });
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleListSecrets(params: { workspace_id: string }) {
  const data = await apiCall("GET", `/workspaces/${params.workspace_id}/secrets`);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

export async function handleDeleteSecret(params: { workspace_id: string; key: string }) {
  const { workspace_id, key } = params;
  const data = await apiCall("DELETE", `/workspaces/${workspace_id}/secrets/${encodeURIComponent(key)}`);
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

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

export function registerSecretTools(srv: McpServer) {
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

  srv.tool(
    "delete_secret",
    "Delete a secret from a workspace",
    { workspace_id: z.string(), key: z.string() },
    handleDeleteSecret
  );

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
}
