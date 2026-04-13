import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { apiCall, PLATFORM_URL, toMcpResult, isApiError } from "../api.js";

// Fetch the workspace list, filter to runtime='external'. The platform
// has no dedicated /remote-agents endpoint — we filter client-side
// because the workspace list is small (tens to low-hundreds, never
// pagination scale) and adding a server endpoint would be a separate PR.
export async function handleListRemoteAgents() {
  const data = await apiCall("GET", "/workspaces");
  if (!Array.isArray(data)) {
    return toMcpResult(data);
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
  return toMcpResult({ count: remote.length, agents: remote });
}

// Phase 30.4 — token-gated; from MCP we don't have a workspace bearer
// (we're an operator surface), so we hit the lightweight unauthenticated
// /workspaces/:id endpoint and project the same shape. Still useful as
// a focused tool that doesn't dump the full workspace blob.
export async function handleGetRemoteAgentState(params: { workspace_id: string }) {
  const data = await apiCall("GET", `/workspaces/${params.workspace_id}`);
  if (isApiError(data)) {
    return toMcpResult(data);
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
  return toMcpResult(projected);
}

export async function handleGetRemoteAgentSetupCommand(params: {
  workspace_id: string;
  platform_url_override?: string;
}) {
  // Verify the workspace exists and is runtime='external' before generating
  // the command — saves the operator from pasting a bash line that will
  // fail because the workspace was a Docker workspace they typed by mistake.
  const ws = await apiCall("GET", `/workspaces/${params.workspace_id}`);
  if (isApiError(ws)) {
    return toMcpResult(ws);
  }
  const w = ws as { id: string; name: string; runtime?: string };
  if (w.runtime !== "external") {
    return toMcpResult({
      error: "workspace is not external; setup command only applies to runtime='external'",
      workspace_id: w.id,
      actual_runtime: w.runtime,
    });
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
    `python3 -c "from molecule_agent import RemoteAgentClient; \\`,
    `  c = RemoteAgentClient.register_from_env(); \\`,
    `  c.pull_secrets(); \\`,
    `  c.run_heartbeat_loop()"`,
    ``,
    `# For a richer demo (logging, graceful shutdown) see`,
    `# sdk/python/examples/remote-agent/run.py in the molecule-monorepo checkout.`,
    `# The agent will register, mint its bearer token (cached at`,
    `# ~/.molecule/${w.id}/.auth_token), pull secrets, then heartbeat.`,
  ].join("\n");
  return toMcpResult({
    workspace_id: w.id,
    workspace_name: w.name,
    platform_url: targetUrl,
    setup_command: setupCmd,
    ...(warnings.length > 0 ? { warnings } : {}),
  });
}

export async function handleCheckRemoteAgentFreshness(params: {
  workspace_id: string;
  threshold_seconds?: number;
}) {
  const ws = await apiCall("GET", `/workspaces/${params.workspace_id}`);
  if (isApiError(ws)) {
    return toMcpResult(ws);
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
  return toMcpResult({
    workspace_id: params.workspace_id,
    status: w.status,
    runtime: w.runtime,
    last_heartbeat_at: heartbeatStr,
    seconds_since_heartbeat: secondsSince,
    threshold_seconds: threshold,
    fresh,
  });
}

export function registerRemoteAgentTools(srv: McpServer) {
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
}
