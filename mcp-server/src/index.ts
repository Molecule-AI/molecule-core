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

import { PLATFORM_URL, apiCall } from "./api.js";
import { registerWorkspaceTools } from "./tools/workspaces.js";
import { registerAgentTools } from "./tools/agents.js";
import { registerSecretTools } from "./tools/secrets.js";
import { registerFileTools } from "./tools/files.js";
import { registerMemoryTools } from "./tools/memory.js";
import { registerPluginTools } from "./tools/plugins.js";
import { registerChannelTools } from "./tools/channels.js";
import { registerDelegationTools } from "./tools/delegation.js";
import { registerScheduleTools } from "./tools/schedules.js";
import { registerApprovalTools } from "./tools/approvals.js";
import { registerDiscoveryTools } from "./tools/discovery.js";
import { registerRemoteAgentTools } from "./tools/remote_agents.js";

// Re-exports so existing importers (tests, SDK consumers) keep working.
// Explicit names (not `export *`) so tree-shakers and TS readers can see
// exactly which handlers are part of the public surface, and a missing
// export triggers a compile error instead of a silent undefined at import.
export { PLATFORM_URL, apiCall, isApiError, toMcpResult, toMcpText } from "./api.js";
export type { ApiError } from "./api.js";

export {
  registerWorkspaceTools,
  handleListWorkspaces,
  handleCreateWorkspace,
  handleGetWorkspace,
  handleDeleteWorkspace,
  handleRestartWorkspace,
  handleUpdateWorkspace,
  handlePauseWorkspace,
  handleResumeWorkspace,
} from "./tools/workspaces.js";

export {
  registerAgentTools,
  handleChatWithAgent,
  handleAssignAgent,
  handleReplaceAgent,
  handleRemoveAgent,
  handleMoveAgent,
  handleGetModel,
} from "./tools/agents.js";

export {
  registerSecretTools,
  handleSetSecret,
  handleListSecrets,
  handleDeleteSecret,
  handleListGlobalSecrets,
  handleSetGlobalSecret,
  handleDeleteGlobalSecret,
} from "./tools/secrets.js";

export {
  registerFileTools,
  handleListFiles,
  handleReadFile,
  handleWriteFile,
  handleDeleteFile,
  handleReplaceAllFiles,
  handleGetConfig,
  handleUpdateConfig,
} from "./tools/files.js";

export {
  registerMemoryTools,
  handleCommitMemory,
  handleSearchMemory,
  handleDeleteMemory,
  handleSessionSearch,
  handleGetSharedContext,
  handleSetKV,
  handleGetKV,
  handleListKV,
  handleDeleteKV,
} from "./tools/memory.js";

export {
  registerPluginTools,
  handleListPluginRegistry,
  handleListInstalledPlugins,
  handleInstallPlugin,
  handleUninstallPlugin,
  handleListPluginSources,
  handleListAvailablePlugins,
  handleCheckPluginCompatibility,
} from "./tools/plugins.js";

export {
  registerChannelTools,
  handleListChannelAdapters,
  handleListChannels,
  handleAddChannel,
  handleUpdateChannel,
  handleRemoveChannel,
  handleSendChannelMessage,
  handleTestChannel,
  handleDiscoverChannelChats,
} from "./tools/channels.js";

export {
  registerDelegationTools,
  handleAsyncDelegate,
  handleCheckDelegations,
  handleRecordDelegation,
  handleUpdateDelegationStatus,
  handleReportActivity,
  handleListActivity,
  handleNotifyUser,
  handleListTraces,
} from "./tools/delegation.js";

export {
  registerScheduleTools,
  handleListSchedules,
  handleCreateSchedule,
  handleUpdateSchedule,
  handleDeleteSchedule,
  handleRunSchedule,
  handleGetScheduleHistory,
} from "./tools/schedules.js";

export {
  registerApprovalTools,
  handleListPendingApprovals,
  handleDecideApproval,
  handleCreateApproval,
  handleGetWorkspaceApprovals,
} from "./tools/approvals.js";

export {
  registerDiscoveryTools,
  handleListPeers,
  handleDiscoverWorkspace,
  handleCheckAccess,
  handleListEvents,
  handleListTemplates,
  handleListOrgTemplates,
  handleImportOrg,
  handleImportTemplate,
  handleExportBundle,
  handleImportBundle,
  handleGetViewport,
  handleSetViewport,
  handleExpandTeam,
  handleCollapseTeam,
} from "./tools/discovery.js";

export {
  registerRemoteAgentTools,
  handleListRemoteAgents,
  handleGetRemoteAgentState,
  handleGetRemoteAgentSetupCommand,
  handleCheckRemoteAgentFreshness,
} from "./tools/remote_agents.js";

export function createServer() {
  const srv = new McpServer({
    name: "molecule",
    version: "1.0.0",
  });

  registerWorkspaceTools(srv);
  registerAgentTools(srv);
  registerSecretTools(srv);
  registerFileTools(srv);
  registerMemoryTools(srv);
  registerPluginTools(srv);
  registerChannelTools(srv);
  registerDelegationTools(srv);
  registerScheduleTools(srv);
  registerApprovalTools(srv);
  registerDiscoveryTools(srv);
  registerRemoteAgentTools(srv);

  return srv;
}

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
  console.error("Molecule AI MCP server running on stdio (87 tools available)");
}

// Only auto-start when run directly (not when imported for testing).
// JEST_WORKER_ID is set automatically by Jest in every worker process.
if (!process.env.JEST_WORKER_ID) {
  main().catch(console.error);
}
