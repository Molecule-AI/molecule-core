/**
 * Comprehensive unit tests for the Molecule AI MCP Server
 *
 * Tests the apiCall() helper and all tool handler functions.
 * fetch is mocked globally so no real HTTP requests are made.
 */

// Jest hoists these mock calls before imports, so the MCP SDK is
// mocked before index.ts is loaded (preventing stdio/server side-effects).
jest.mock("@modelcontextprotocol/sdk/server/mcp.js", () => ({
  McpServer: class {
    tool() {}
    connect() { return Promise.resolve(); }
  },
}));
jest.mock("@modelcontextprotocol/sdk/server/stdio.js", () => ({
  StdioServerTransport: class {},
}));

import {
  apiCall,
  PLATFORM_URL,
  handleListWorkspaces,
  handleCreateWorkspace,
  handleGetWorkspace,
  handleDeleteWorkspace,
  handleRestartWorkspace,
  handleChatWithAgent,
  handleAssignAgent,
  handleSetSecret,
  handleListSecrets,
  handleListFiles,
  handleReadFile,
  handleWriteFile,
  handleDeleteFile,
  handleCommitMemory,
  handleSearchMemory,
  handleListTemplates,
  handleExpandTeam,
  handleCollapseTeam,
  handleListPendingApprovals,
  handleDecideApproval,
  handleUpdateWorkspace,
  handleReplaceAgent,
  handleRemoveAgent,
  handleMoveAgent,
  handleDeleteSecret,
  handleGetConfig,
  handleUpdateConfig,
  handleListPeers,
  handleDiscoverWorkspace,
  handleCheckAccess,
  handleListEvents,
  handleExportBundle,
  handleImportBundle,
  handleImportTemplate,
  handleReplaceAllFiles,
  handleListTraces,
  handleListActivity,
  handleDeleteMemory,
  handleGetModel,
  handleCreateApproval,
  handleGetWorkspaceApprovals,
  handleListPluginRegistry,
  handleListInstalledPlugins,
  handleInstallPlugin,
  handleUninstallPlugin,
  handleListGlobalSecrets,
  handleSetGlobalSecret,
  handleDeleteGlobalSecret,
  handlePauseWorkspace,
  handleResumeWorkspace,
  handleListOrgTemplates,
  handleImportOrg,
  handleListRemoteAgents,
  handleGetRemoteAgentState,
  handleGetRemoteAgentSetupCommand,
  handleCheckRemoteAgentFreshness,
  createServer,
} from "../index.js";

// ============================================================
// Helpers
// ============================================================

/** Build a minimal fetch mock that returns a JSON-serialisable payload. */
function mockFetch(payload: unknown, ok = true, status = 200) {
  const body = JSON.stringify(payload);
  return jest.fn().mockResolvedValue({
    ok,
    status,
    text: jest.fn().mockResolvedValue(body),
  });
}

/** Build a fetch mock whose .text() returns a non-JSON string. */
function mockFetchText(text: string, ok = true, status = 200) {
  return jest.fn().mockResolvedValue({
    ok,
    status,
    text: jest.fn().mockResolvedValue(text),
  });
}

/** Build a fetch mock that throws a network error. */
function mockFetchThrow(message = "Network error") {
  return jest.fn().mockRejectedValue(new Error(message));
}

/** Verify the content array has exactly one text entry matching expected JSON. */
function expectJsonContent(result: { content: Array<{ type: string; text: string }> }, expected: unknown) {
  expect(result.content).toHaveLength(1);
  expect(result.content[0].type).toBe("text");
  const parsed = JSON.parse(result.content[0].text);
  expect(parsed).toEqual(expected);
}

// ============================================================
// apiCall() tests
// ============================================================

describe("apiCall()", () => {
  beforeEach(() => {
    jest.resetAllMocks();
  });

  test("returns parsed JSON on successful response", async () => {
    global.fetch = mockFetch({ workspaces: [] });
    const result = await apiCall("GET", "/workspaces");
    expect(result).toEqual({ workspaces: [] });
  });

  test("sends correct method, URL and Content-Type header", async () => {
    global.fetch = mockFetch({ id: "ws-1" });
    await apiCall("POST", "/workspaces", { name: "test" });
    expect(global.fetch).toHaveBeenCalledWith(
      `${PLATFORM_URL}/workspaces`,
      expect.objectContaining({
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: "test" }),
      })
    );
  });

  test("omits body when none provided (GET requests)", async () => {
    global.fetch = mockFetch([]);
    await apiCall("GET", "/workspaces");
    expect(global.fetch).toHaveBeenCalledWith(
      `${PLATFORM_URL}/workspaces`,
      expect.objectContaining({ body: undefined })
    );
  });

  test("returns error object on non-OK HTTP response (404)", async () => {
    global.fetch = mockFetchText("Not Found", false, 404);
    const result = await apiCall("GET", "/workspaces/missing");
    expect(result).toMatchObject({ error: expect.stringContaining("404") });
  });

  test("returns error object on non-OK HTTP response (500)", async () => {
    global.fetch = mockFetchText("Internal Server Error", false, 500);
    const result = await apiCall("GET", "/workspaces");
    expect(result).toMatchObject({ error: expect.stringContaining("500") });
  });

  test("returns error object when fetch throws (network error)", async () => {
    const consoleSpy = jest.spyOn(console, "error").mockImplementation(() => {});
    global.fetch = mockFetchThrow("ECONNREFUSED");
    const result = await apiCall("GET", "/workspaces");
    expect(result).toMatchObject({
      error: expect.stringContaining("Platform unreachable"),
      detail: "ECONNREFUSED",
    });
    consoleSpy.mockRestore();
  });

  test("falls back to { raw, status } when response body is not valid JSON", async () => {
    global.fetch = mockFetchText("plain text response");
    const result = await apiCall("GET", "/some-endpoint");
    expect(result).toMatchObject({ raw: "plain text response", status: 200 });
  });

  test("stringifies body correctly for nested objects", async () => {
    global.fetch = mockFetch({ ok: true });
    const body = { nested: { deep: [1, 2, 3] } };
    await apiCall("PUT", "/test", body);
    const callArgs = (global.fetch as jest.Mock).mock.calls[0][1];
    expect(JSON.parse(callArgs.body)).toEqual(body);
  });
});

// ============================================================
// Workspace tool handlers
// ============================================================

describe("handleListWorkspaces()", () => {
  test("calls GET /workspaces and returns formatted content", async () => {
    const wsData = [{ id: "ws-1", name: "Alpha" }];
    global.fetch = mockFetch(wsData);
    const result = await handleListWorkspaces();
    expect(global.fetch).toHaveBeenCalledWith(
      `${PLATFORM_URL}/workspaces`,
      expect.objectContaining({ method: "GET" })
    );
    expectJsonContent(result, wsData);
  });
});

describe("handleCreateWorkspace()", () => {
  test("calls POST /workspaces with name, role, template, tier, parent_id", async () => {
    global.fetch = mockFetch({ id: "ws-new", name: "Beta" });
    const result = await handleCreateWorkspace({
      name: "Beta",
      role: "researcher",
      template: "basic",
      tier: 2,
      parent_id: "ws-root",
    });
    const callArgs = (global.fetch as jest.Mock).mock.calls[0];
    expect(callArgs[0]).toBe(`${PLATFORM_URL}/workspaces`);
    expect(callArgs[1].method).toBe("POST");
    const sentBody = JSON.parse(callArgs[1].body);
    expect(sentBody.name).toBe("Beta");
    expect(sentBody.role).toBe("researcher");
    expect(sentBody.tier).toBe(2);
    expect(sentBody.parent_id).toBe("ws-root");
    expect(sentBody.canvas).toBeDefined();
    expect(result.content[0].type).toBe("text");
  });

  test("works with minimal params (name only)", async () => {
    global.fetch = mockFetch({ id: "ws-min" });
    await handleCreateWorkspace({ name: "Minimal" });
    const sentBody = JSON.parse((global.fetch as jest.Mock).mock.calls[0][1].body);
    expect(sentBody.name).toBe("Minimal");
    expect(sentBody.canvas).toBeDefined();
  });
});

describe("handleGetWorkspace()", () => {
  test("calls GET /workspaces/:id with correct path", async () => {
    const ws = { id: "ws-abc", name: "Test" };
    global.fetch = mockFetch(ws);
    const result = await handleGetWorkspace({ workspace_id: "ws-abc" });
    expect(global.fetch).toHaveBeenCalledWith(
      `${PLATFORM_URL}/workspaces/ws-abc`,
      expect.objectContaining({ method: "GET" })
    );
    expectJsonContent(result, ws);
  });
});

describe("handleDeleteWorkspace()", () => {
  test("calls DELETE /workspaces/:id?confirm=true", async () => {
    global.fetch = mockFetch({ deleted: true });
    await handleDeleteWorkspace({ workspace_id: "ws-del" });
    expect(global.fetch).toHaveBeenCalledWith(
      `${PLATFORM_URL}/workspaces/ws-del?confirm=true`,
      expect.objectContaining({ method: "DELETE" })
    );
  });
});

describe("handleRestartWorkspace()", () => {
  test("calls POST /workspaces/:id/restart with empty body", async () => {
    global.fetch = mockFetch({ restarted: true });
    await handleRestartWorkspace({ workspace_id: "ws-r" });
    expect(global.fetch).toHaveBeenCalledWith(
      `${PLATFORM_URL}/workspaces/ws-r/restart`,
      expect.objectContaining({ method: "POST" })
    );
  });
});

// ============================================================
// Chat / A2A
// ============================================================

describe("handleChatWithAgent()", () => {
  test("POSTs to /workspaces/:id/a2a with correct message structure", async () => {
    const a2aResponse = {
      result: {
        parts: [
          { kind: "text", text: "Hello from agent" },
          { kind: "text", text: "Second line" },
        ],
      },
    };
    global.fetch = mockFetch(a2aResponse);
    const result = await handleChatWithAgent({ workspace_id: "ws-chat", message: "Hi there" });

    const callArgs = (global.fetch as jest.Mock).mock.calls[0];
    expect(callArgs[0]).toBe(`${PLATFORM_URL}/workspaces/ws-chat/a2a`);
    const sent = JSON.parse(callArgs[1].body);
    expect(sent.method).toBe("message/send");
    expect(sent.params.message.role).toBe("user");
    expect(sent.params.message.parts[0].text).toBe("Hi there");

    // Text parts should be extracted and joined
    expect(result.content[0].text).toBe("Hello from agent\nSecond line");
  });

  test("falls back to raw JSON when no text parts in response", async () => {
    const a2aResponse = { result: { parts: [{ kind: "data", data: {} }] } };
    global.fetch = mockFetch(a2aResponse);
    const result = await handleChatWithAgent({ workspace_id: "ws-chat", message: "Hi" });
    // No text parts → JSON fallback
    expect(result.content[0].text).toContain("result");
  });

  test("falls back to raw JSON when result is empty", async () => {
    global.fetch = mockFetch({ error: "agent not running" });
    const result = await handleChatWithAgent({ workspace_id: "ws-chat", message: "Hi" });
    expect(result.content[0].text).toContain("agent not running");
  });
});

// ============================================================
// Agent Management
// ============================================================

describe("handleAssignAgent()", () => {
  test("POSTs to /workspaces/:id/agent with model", async () => {
    global.fetch = mockFetch({ agent: "assigned" });
    const result = await handleAssignAgent({ workspace_id: "ws-1", model: "openrouter:anthropic/claude-3.5-haiku" });
    const callArgs = (global.fetch as jest.Mock).mock.calls[0];
    expect(callArgs[0]).toBe(`${PLATFORM_URL}/workspaces/ws-1/agent`);
    expect(callArgs[1].method).toBe("POST");
    const sent = JSON.parse(callArgs[1].body);
    expect(sent.model).toBe("openrouter:anthropic/claude-3.5-haiku");
    expectJsonContent(result, { agent: "assigned" });
  });
});

describe("handleReplaceAgent()", () => {
  test("PATCHes /workspaces/:id/agent with new model", async () => {
    global.fetch = mockFetch({ updated: true });
    await handleReplaceAgent({ workspace_id: "ws-2", model: "openrouter:gpt-4o" });
    const callArgs = (global.fetch as jest.Mock).mock.calls[0];
    expect(callArgs[1].method).toBe("PATCH");
    expect(callArgs[0]).toContain("/workspaces/ws-2/agent");
  });
});

describe("handleRemoveAgent()", () => {
  test("DELETEs /workspaces/:id/agent", async () => {
    global.fetch = mockFetch({ removed: true });
    await handleRemoveAgent({ workspace_id: "ws-3" });
    expect(global.fetch).toHaveBeenCalledWith(
      `${PLATFORM_URL}/workspaces/ws-3/agent`,
      expect.objectContaining({ method: "DELETE" })
    );
  });
});

describe("handleMoveAgent()", () => {
  test("POSTs to /workspaces/:id/agent/move with target id", async () => {
    global.fetch = mockFetch({ moved: true });
    await handleMoveAgent({ workspace_id: "ws-src", target_workspace_id: "ws-dst" });
    const callArgs = (global.fetch as jest.Mock).mock.calls[0];
    expect(callArgs[0]).toBe(`${PLATFORM_URL}/workspaces/ws-src/agent/move`);
    const sent = JSON.parse(callArgs[1].body);
    expect(sent.target_workspace_id).toBe("ws-dst");
  });
});

// ============================================================
// Secrets
// ============================================================

describe("handleSetSecret()", () => {
  test("POSTs to /workspaces/:id/secrets with key and value", async () => {
    global.fetch = mockFetch({ set: true });
    const result = await handleSetSecret({ workspace_id: "ws-s", key: "ANTHROPIC_API_KEY", value: "sk-test" });
    const callArgs = (global.fetch as jest.Mock).mock.calls[0];
    expect(callArgs[0]).toBe(`${PLATFORM_URL}/workspaces/ws-s/secrets`);
    expect(callArgs[1].method).toBe("POST");
    const sent = JSON.parse(callArgs[1].body);
    expect(sent.key).toBe("ANTHROPIC_API_KEY");
    expect(sent.value).toBe("sk-test");
    expectJsonContent(result, { set: true });
  });
});

describe("handleListSecrets()", () => {
  test("GETs /workspaces/:id/secrets", async () => {
    global.fetch = mockFetch({ secrets: ["ANTHROPIC_API_KEY"] });
    const result = await handleListSecrets({ workspace_id: "ws-s" });
    expect(global.fetch).toHaveBeenCalledWith(
      `${PLATFORM_URL}/workspaces/ws-s/secrets`,
      expect.objectContaining({ method: "GET" })
    );
    expectJsonContent(result, { secrets: ["ANTHROPIC_API_KEY"] });
  });
});

describe("handleDeleteSecret()", () => {
  test("DELETEs /workspaces/:id/secrets/:key (URL-encoded)", async () => {
    global.fetch = mockFetch({ deleted: true });
    await handleDeleteSecret({ workspace_id: "ws-s", key: "MY KEY" });
    expect(global.fetch).toHaveBeenCalledWith(
      `${PLATFORM_URL}/workspaces/ws-s/secrets/MY%20KEY`,
      expect.objectContaining({ method: "DELETE" })
    );
  });
});

// ============================================================
// Files
// ============================================================

describe("handleListFiles()", () => {
  test("GETs /workspaces/:id/files", async () => {
    global.fetch = mockFetch(["system-prompt.md"]);
    await handleListFiles({ workspace_id: "ws-f" });
    expect(global.fetch).toHaveBeenCalledWith(
      `${PLATFORM_URL}/workspaces/ws-f/files`,
      expect.objectContaining({ method: "GET" })
    );
  });
});

describe("handleReadFile()", () => {
  test("GETs /workspaces/:id/files/:path and extracts content field", async () => {
    global.fetch = mockFetch({ content: "# Hello World" });
    const result = await handleReadFile({ workspace_id: "ws-f", path: "system-prompt.md" });
    expect(global.fetch).toHaveBeenCalledWith(
      `${PLATFORM_URL}/workspaces/ws-f/files/system-prompt.md`,
      expect.objectContaining({ method: "GET" })
    );
    expect(result.content[0].text).toBe("# Hello World");
  });

  test("falls back to JSON.stringify when no content field", async () => {
    global.fetch = mockFetch({ raw: "data" });
    const result = await handleReadFile({ workspace_id: "ws-f", path: "other.yaml" });
    expect(result.content[0].text).toContain("raw");
  });
});

describe("handleWriteFile()", () => {
  test("PUTs to /workspaces/:id/files/:path with content", async () => {
    global.fetch = mockFetch({ written: true });
    await handleWriteFile({ workspace_id: "ws-f", path: "system-prompt.md", content: "# New" });
    const callArgs = (global.fetch as jest.Mock).mock.calls[0];
    expect(callArgs[0]).toBe(`${PLATFORM_URL}/workspaces/ws-f/files/system-prompt.md`);
    expect(callArgs[1].method).toBe("PUT");
  });
});

describe("handleDeleteFile()", () => {
  test("DELETEs /workspaces/:id/files/:path", async () => {
    global.fetch = mockFetch({ deleted: true });
    await handleDeleteFile({ workspace_id: "ws-f", path: "old.md" });
    expect(global.fetch).toHaveBeenCalledWith(
      `${PLATFORM_URL}/workspaces/ws-f/files/old.md`,
      expect.objectContaining({ method: "DELETE" })
    );
  });
});

describe("handleReplaceAllFiles()", () => {
  test("PUTs to /workspaces/:id/files with files map", async () => {
    global.fetch = mockFetch({ replaced: true });
    await handleReplaceAllFiles({
      workspace_id: "ws-f",
      files: { "system-prompt.md": "# Content", "config.yaml": "key: val" },
    });
    const callArgs = (global.fetch as jest.Mock).mock.calls[0];
    expect(callArgs[1].method).toBe("PUT");
    const sent = JSON.parse(callArgs[1].body);
    expect(sent.files["system-prompt.md"]).toBe("# Content");
  });
});

// ============================================================
// Memory (HMA)
// ============================================================

describe("handleCommitMemory()", () => {
  test("POSTs to /workspaces/:id/memories with content and scope", async () => {
    global.fetch = mockFetch({ id: "mem-1" });
    const result = await handleCommitMemory({
      workspace_id: "ws-m",
      content: "Important fact",
      scope: "GLOBAL",
    });
    const callArgs = (global.fetch as jest.Mock).mock.calls[0];
    expect(callArgs[0]).toBe(`${PLATFORM_URL}/workspaces/ws-m/memories`);
    expect(callArgs[1].method).toBe("POST");
    const sent = JSON.parse(callArgs[1].body);
    expect(sent.content).toBe("Important fact");
    expect(sent.scope).toBe("GLOBAL");
    expectJsonContent(result, { id: "mem-1" });
  });

  test("supports LOCAL scope", async () => {
    global.fetch = mockFetch({ id: "mem-2" });
    await handleCommitMemory({ workspace_id: "ws-m", content: "Local fact", scope: "LOCAL" });
    const sent = JSON.parse((global.fetch as jest.Mock).mock.calls[0][1].body);
    expect(sent.scope).toBe("LOCAL");
  });
});

describe("handleSearchMemory()", () => {
  test("GETs /workspaces/:id/memories with query params", async () => {
    global.fetch = mockFetch([{ id: "mem-1", content: "fact" }]);
    await handleSearchMemory({ workspace_id: "ws-m", query: "important", scope: "GLOBAL" });
    const callUrl: string = (global.fetch as jest.Mock).mock.calls[0][0];
    expect(callUrl).toContain("/workspaces/ws-m/memories");
    expect(callUrl).toContain("q=important");
    expect(callUrl).toContain("scope=GLOBAL");
    expect((global.fetch as jest.Mock).mock.calls[0][1].method).toBe("GET");
  });

  test("omits query params when not provided", async () => {
    global.fetch = mockFetch([]);
    await handleSearchMemory({ workspace_id: "ws-m" });
    const callUrl: string = (global.fetch as jest.Mock).mock.calls[0][0];
    expect(callUrl).not.toContain("q=");
    expect(callUrl).not.toContain("scope=");
  });
});

describe("handleDeleteMemory()", () => {
  test("DELETEs /workspaces/:id/memories/:memory_id", async () => {
    global.fetch = mockFetch({ deleted: true });
    await handleDeleteMemory({ workspace_id: "ws-m", memory_id: "mem-42" });
    expect(global.fetch).toHaveBeenCalledWith(
      `${PLATFORM_URL}/workspaces/ws-m/memories/mem-42`,
      expect.objectContaining({ method: "DELETE" })
    );
  });
});

// ============================================================
// Templates
// ============================================================

describe("handleListTemplates()", () => {
  test("GETs /templates", async () => {
    global.fetch = mockFetch(["basic", "browser"]);
    const result = await handleListTemplates();
    expect(global.fetch).toHaveBeenCalledWith(
      `${PLATFORM_URL}/templates`,
      expect.objectContaining({ method: "GET" })
    );
    expectJsonContent(result, ["basic", "browser"]);
  });
});

describe("handleImportTemplate()", () => {
  test("POSTs to /templates/import with name and files", async () => {
    global.fetch = mockFetch({ imported: "my-template" });
    await handleImportTemplate({ name: "my-template", files: { "SKILL.md": "# Skill" } });
    const callArgs = (global.fetch as jest.Mock).mock.calls[0];
    expect(callArgs[0]).toBe(`${PLATFORM_URL}/templates/import`);
    expect(callArgs[1].method).toBe("POST");
    const sent = JSON.parse(callArgs[1].body);
    expect(sent.name).toBe("my-template");
  });
});

// ============================================================
// Team Expansion
// ============================================================

describe("handleExpandTeam()", () => {
  test("POSTs to /workspaces/:id/expand", async () => {
    global.fetch = mockFetch({ expanded: true });
    await handleExpandTeam({ workspace_id: "ws-team" });
    expect(global.fetch).toHaveBeenCalledWith(
      `${PLATFORM_URL}/workspaces/ws-team/expand`,
      expect.objectContaining({ method: "POST" })
    );
  });
});

describe("handleCollapseTeam()", () => {
  test("POSTs to /workspaces/:id/collapse", async () => {
    global.fetch = mockFetch({ collapsed: true });
    await handleCollapseTeam({ workspace_id: "ws-team" });
    expect(global.fetch).toHaveBeenCalledWith(
      `${PLATFORM_URL}/workspaces/ws-team/collapse`,
      expect.objectContaining({ method: "POST" })
    );
  });
});

// ============================================================
// Approvals
// ============================================================

describe("handleListPendingApprovals()", () => {
  test("GETs /approvals/pending", async () => {
    global.fetch = mockFetch([{ id: "ap-1" }]);
    const result = await handleListPendingApprovals();
    expect(global.fetch).toHaveBeenCalledWith(
      `${PLATFORM_URL}/approvals/pending`,
      expect.objectContaining({ method: "GET" })
    );
    expectJsonContent(result, [{ id: "ap-1" }]);
  });
});

describe("handleDecideApproval()", () => {
  test("POSTs to /workspaces/:id/approvals/:ap_id/decide with approved decision", async () => {
    global.fetch = mockFetch({ decided: true });
    const result = await handleDecideApproval({
      workspace_id: "ws-1",
      approval_id: "ap-42",
      decision: "approved",
    });
    const callArgs = (global.fetch as jest.Mock).mock.calls[0];
    expect(callArgs[0]).toBe(`${PLATFORM_URL}/workspaces/ws-1/approvals/ap-42/decide`);
    expect(callArgs[1].method).toBe("POST");
    const sent = JSON.parse(callArgs[1].body);
    expect(sent.decision).toBe("approved");
    expect(sent.decided_by).toBe("mcp-client");
    expectJsonContent(result, { decided: true });
  });

  test("POSTs with denied decision", async () => {
    global.fetch = mockFetch({ decided: true });
    await handleDecideApproval({ workspace_id: "ws-1", approval_id: "ap-99", decision: "denied" });
    const sent = JSON.parse((global.fetch as jest.Mock).mock.calls[0][1].body);
    expect(sent.decision).toBe("denied");
  });
});

describe("handleCreateApproval()", () => {
  test("POSTs to /workspaces/:id/approvals with action and reason", async () => {
    global.fetch = mockFetch({ id: "ap-new" });
    await handleCreateApproval({ workspace_id: "ws-1", action: "deploy", reason: "prod release" });
    const callArgs = (global.fetch as jest.Mock).mock.calls[0];
    expect(callArgs[0]).toBe(`${PLATFORM_URL}/workspaces/ws-1/approvals`);
    const sent = JSON.parse(callArgs[1].body);
    expect(sent.action).toBe("deploy");
    expect(sent.reason).toBe("prod release");
  });
});

describe("handleGetWorkspaceApprovals()", () => {
  test("GETs /workspaces/:id/approvals", async () => {
    global.fetch = mockFetch([{ id: "ap-1" }]);
    await handleGetWorkspaceApprovals({ workspace_id: "ws-1" });
    expect(global.fetch).toHaveBeenCalledWith(
      `${PLATFORM_URL}/workspaces/ws-1/approvals`,
      expect.objectContaining({ method: "GET" })
    );
  });
});

// ============================================================
// Workspace update
// ============================================================

describe("handleUpdateWorkspace()", () => {
  test("PATCHes /workspaces/:id with provided fields", async () => {
    global.fetch = mockFetch({ updated: true });
    await handleUpdateWorkspace({ workspace_id: "ws-1", name: "New Name", tier: 3 });
    const callArgs = (global.fetch as jest.Mock).mock.calls[0];
    expect(callArgs[0]).toBe(`${PLATFORM_URL}/workspaces/ws-1`);
    expect(callArgs[1].method).toBe("PATCH");
    const sent = JSON.parse(callArgs[1].body);
    expect(sent.name).toBe("New Name");
    expect(sent.tier).toBe(3);
    expect(sent.workspace_id).toBeUndefined();
  });
});

// ============================================================
// Config
// ============================================================

describe("handleGetConfig()", () => {
  test("GETs /workspaces/:id/config", async () => {
    const config = { maxTokens: 4096, temperature: 0.7 };
    global.fetch = mockFetch(config);
    const result = await handleGetConfig({ workspace_id: "ws-cfg" });
    expect(global.fetch).toHaveBeenCalledWith(
      `${PLATFORM_URL}/workspaces/ws-cfg/config`,
      expect.objectContaining({ method: "GET" })
    );
    expectJsonContent(result, config);
  });
});

describe("handleUpdateConfig()", () => {
  test("PATCHes /workspaces/:id/config with config fields", async () => {
    global.fetch = mockFetch({ updated: true });
    await handleUpdateConfig({ workspace_id: "ws-cfg", config: { temperature: 0.5 } });
    const callArgs = (global.fetch as jest.Mock).mock.calls[0];
    expect(callArgs[1].method).toBe("PATCH");
    const sent = JSON.parse(callArgs[1].body);
    expect(sent.temperature).toBe(0.5);
  });
});

// ============================================================
// Peers / Registry
// ============================================================

describe("handleListPeers()", () => {
  test("GETs /registry/:id/peers", async () => {
    const peers = [{ id: "ws-peer" }];
    global.fetch = mockFetch(peers);
    const result = await handleListPeers({ workspace_id: "ws-main" });
    expect(global.fetch).toHaveBeenCalledWith(
      `${PLATFORM_URL}/registry/ws-main/peers`,
      expect.objectContaining({ method: "GET" })
    );
    expectJsonContent(result, peers);
  });
});

describe("handleDiscoverWorkspace()", () => {
  test("GETs /registry/discover/:id", async () => {
    global.fetch = mockFetch({ url: "http://ws-abc:8080" });
    await handleDiscoverWorkspace({ workspace_id: "ws-abc" });
    expect(global.fetch).toHaveBeenCalledWith(
      `${PLATFORM_URL}/registry/discover/ws-abc`,
      expect.objectContaining({ method: "GET" })
    );
  });
});

describe("handleCheckAccess()", () => {
  test("POSTs to /registry/check-access with caller and target ids", async () => {
    global.fetch = mockFetch({ allowed: true });
    const result = await handleCheckAccess({ caller_id: "ws-caller", target_id: "ws-target" });
    const callArgs = (global.fetch as jest.Mock).mock.calls[0];
    expect(callArgs[0]).toBe(`${PLATFORM_URL}/registry/check-access`);
    const sent = JSON.parse(callArgs[1].body);
    expect(sent.caller_id).toBe("ws-caller");
    expect(sent.target_id).toBe("ws-target");
    expectJsonContent(result, { allowed: true });
  });
});

// ============================================================
// Events
// ============================================================

describe("handleListEvents()", () => {
  test("GETs /events when no workspace_id provided", async () => {
    global.fetch = mockFetch([]);
    await handleListEvents({});
    expect(global.fetch).toHaveBeenCalledWith(
      `${PLATFORM_URL}/events`,
      expect.objectContaining({ method: "GET" })
    );
  });

  test("GETs /events/:id when workspace_id provided", async () => {
    global.fetch = mockFetch([]);
    await handleListEvents({ workspace_id: "ws-ev" });
    expect(global.fetch).toHaveBeenCalledWith(
      `${PLATFORM_URL}/events/ws-ev`,
      expect.objectContaining({ method: "GET" })
    );
  });
});

// ============================================================
// Bundles
// ============================================================

describe("handleExportBundle()", () => {
  test("GETs /bundles/export/:id", async () => {
    const bundle = { id: "ws-1", files: {} };
    global.fetch = mockFetch(bundle);
    const result = await handleExportBundle({ workspace_id: "ws-1" });
    expect(global.fetch).toHaveBeenCalledWith(
      `${PLATFORM_URL}/bundles/export/ws-1`,
      expect.objectContaining({ method: "GET" })
    );
    expectJsonContent(result, bundle);
  });
});

describe("handleImportBundle()", () => {
  test("POSTs to /bundles/import with bundle data", async () => {
    global.fetch = mockFetch({ imported: "ws-new" });
    await handleImportBundle({ bundle: { id: "old-ws", name: "Imported" } });
    const callArgs = (global.fetch as jest.Mock).mock.calls[0];
    expect(callArgs[0]).toBe(`${PLATFORM_URL}/bundles/import`);
    expect(callArgs[1].method).toBe("POST");
    const sent = JSON.parse(callArgs[1].body);
    expect(sent.name).toBe("Imported");
  });
});

// ============================================================
// Traces / Activity
// ============================================================

describe("handleListTraces()", () => {
  test("GETs /workspaces/:id/traces", async () => {
    global.fetch = mockFetch([{ traceId: "t-1" }]);
    await handleListTraces({ workspace_id: "ws-tr" });
    expect(global.fetch).toHaveBeenCalledWith(
      `${PLATFORM_URL}/workspaces/ws-tr/traces`,
      expect.objectContaining({ method: "GET" })
    );
  });
});

describe("handleListActivity()", () => {
  test("GETs /workspaces/:id/activity without params when none given", async () => {
    global.fetch = mockFetch([]);
    await handleListActivity({ workspace_id: "ws-act" });
    const callUrl: string = (global.fetch as jest.Mock).mock.calls[0][0];
    expect(callUrl).toBe(`${PLATFORM_URL}/workspaces/ws-act/activity`);
  });

  test("appends type and limit query params when provided", async () => {
    global.fetch = mockFetch([]);
    await handleListActivity({ workspace_id: "ws-act", type: "error", limit: 50 });
    const callUrl: string = (global.fetch as jest.Mock).mock.calls[0][0];
    expect(callUrl).toContain("type=error");
    expect(callUrl).toContain("limit=50");
  });
});

// ============================================================
// Model
// ============================================================

describe("handleGetModel()", () => {
  test("GETs /workspaces/:id/model", async () => {
    global.fetch = mockFetch({ model: "claude-3-sonnet" });
    const result = await handleGetModel({ workspace_id: "ws-m" });
    expect(global.fetch).toHaveBeenCalledWith(
      `${PLATFORM_URL}/workspaces/ws-m/model`,
      expect.objectContaining({ method: "GET" })
    );
    expectJsonContent(result, { model: "claude-3-sonnet" });
  });
});

// ============================================================
// createServer()
// ============================================================

describe("createServer()", () => {
  test("returns an McpServer instance", () => {
    const server = createServer();
    expect(server).toBeDefined();
    expect(typeof server.connect).toBe("function");
  });
});

// ============================================================
// Response format invariants
// ============================================================

describe("Response format invariants", () => {
  beforeEach(() => {
    global.fetch = mockFetch({ ok: true });
  });

  const cases: Array<[string, () => Promise<{ content: Array<{ type: string; text: string }> }>]> = [
    ["handleListWorkspaces", () => handleListWorkspaces()],
    ["handleGetWorkspace", () => handleGetWorkspace({ workspace_id: "x" })],
    ["handleDeleteWorkspace", () => handleDeleteWorkspace({ workspace_id: "x" })],
    ["handleListSecrets", () => handleListSecrets({ workspace_id: "x" })],
    ["handleListPendingApprovals", () => handleListPendingApprovals()],
    ["handleGetConfig", () => handleGetConfig({ workspace_id: "x" })],
    ["handleListPeers", () => handleListPeers({ workspace_id: "x" })],
    ["handleExportBundle", () => handleExportBundle({ workspace_id: "x" })],
    // New tools — plugins, global secrets, pause/resume, org
    ["handleListPluginRegistry", () => handleListPluginRegistry()],
    ["handleListInstalledPlugins", () => handleListInstalledPlugins({ workspace_id: "x" })],
    ["handleInstallPlugin", () => handleInstallPlugin({ workspace_id: "x", source: "local://ecc" })],
    ["handleUninstallPlugin", () => handleUninstallPlugin({ workspace_id: "x", name: "ecc" })],
    ["handleListGlobalSecrets", () => handleListGlobalSecrets()],
    ["handleSetGlobalSecret", () => handleSetGlobalSecret({ key: "K", value: "V" })],
    ["handleDeleteGlobalSecret", () => handleDeleteGlobalSecret({ key: "K" })],
    ["handlePauseWorkspace", () => handlePauseWorkspace({ workspace_id: "x" })],
    ["handleResumeWorkspace", () => handleResumeWorkspace({ workspace_id: "x" })],
    ["handleListOrgTemplates", () => handleListOrgTemplates()],
    ["handleImportOrg", () => handleImportOrg({ dir: "molecule-dev" })],
  ];

  test.each(cases)("%s returns content array with type=text", async (_name, fn) => {
    const result = await fn();
    expect(Array.isArray(result.content)).toBe(true);
    expect(result.content.length).toBeGreaterThan(0);
    expect(result.content[0].type).toBe("text");
    expect(typeof result.content[0].text).toBe("string");
  });
});

// ============================================================
// Plugin handler tests
// ============================================================

describe("Plugin handlers", () => {
  beforeEach(() => {
    jest.resetAllMocks();
    global.fetch = mockFetch([{ name: "ecc", version: "1.0.0", skills: ["coding-standards"] }]);
  });

  test("handleListPluginRegistry calls GET /plugins", async () => {
    const result = await handleListPluginRegistry();
    expect(global.fetch).toHaveBeenCalledWith(
      `${PLATFORM_URL}/plugins`,
      expect.objectContaining({ method: "GET" })
    );
    expectJsonContent(result, [{ name: "ecc", version: "1.0.0", skills: ["coding-standards"] }]);
  });

  test("handleInstallPlugin sends source URL to POST /workspaces/:id/plugins", async () => {
    global.fetch = mockFetch({ status: "installed", plugin: "ecc", source: "local://ecc" });
    const result = await handleInstallPlugin({ workspace_id: "ws-1", source: "local://ecc" });
    expect(global.fetch).toHaveBeenCalledWith(
      `${PLATFORM_URL}/workspaces/ws-1/plugins`,
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ source: "local://ecc" }),
      })
    );
    expectJsonContent(result, { status: "installed", plugin: "ecc", source: "local://ecc" });
  });

  test("handleInstallPlugin supports github:// source", async () => {
    global.fetch = mockFetch({ status: "installed", plugin: "my-plugin", source: "github://org/my-plugin#v1.0" });
    await handleInstallPlugin({ workspace_id: "ws-1", source: "github://org/my-plugin#v1.0" });
    expect(global.fetch).toHaveBeenCalledWith(
      `${PLATFORM_URL}/workspaces/ws-1/plugins`,
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ source: "github://org/my-plugin#v1.0" }),
      })
    );
  });

  test("handleUninstallPlugin calls DELETE /workspaces/:id/plugins/:name", async () => {
    global.fetch = mockFetch({ status: "uninstalled", plugin: "ecc" });
    await handleUninstallPlugin({ workspace_id: "ws-1", name: "ecc" });
    expect(global.fetch).toHaveBeenCalledWith(
      `${PLATFORM_URL}/workspaces/ws-1/plugins/ecc`,
      expect.objectContaining({ method: "DELETE" })
    );
  });
});

// ============================================================
// Global secrets handler tests
// ============================================================

describe("Global secrets handlers", () => {
  beforeEach(() => {
    jest.resetAllMocks();
    global.fetch = mockFetch([{ key: "GITHUB_TOKEN", scope: "global" }]);
  });

  test("handleListGlobalSecrets calls GET /settings/secrets", async () => {
    await handleListGlobalSecrets();
    expect(global.fetch).toHaveBeenCalledWith(
      `${PLATFORM_URL}/settings/secrets`,
      expect.objectContaining({ method: "GET" })
    );
  });

  test("handleSetGlobalSecret calls PUT /settings/secrets", async () => {
    global.fetch = mockFetch({ status: "saved" });
    await handleSetGlobalSecret({ key: "MY_KEY", value: "secret" });
    expect(global.fetch).toHaveBeenCalledWith(
      `${PLATFORM_URL}/settings/secrets`,
      expect.objectContaining({
        method: "PUT",
        body: JSON.stringify({ key: "MY_KEY", value: "secret" }),
      })
    );
  });

  test("handleDeleteGlobalSecret calls DELETE /settings/secrets/:key", async () => {
    global.fetch = mockFetch({ status: "deleted" });
    await handleDeleteGlobalSecret({ key: "OLD_KEY" });
    expect(global.fetch).toHaveBeenCalledWith(
      `${PLATFORM_URL}/settings/secrets/OLD_KEY`,
      expect.objectContaining({ method: "DELETE" })
    );
  });
});

// ============================================================
// Pause/resume and org handler tests
// ============================================================

describe("Pause/resume and org handlers", () => {
  beforeEach(() => {
    jest.resetAllMocks();
    global.fetch = mockFetch({ status: "paused" });
  });

  test("handlePauseWorkspace calls POST /workspaces/:id/pause", async () => {
    await handlePauseWorkspace({ workspace_id: "ws-1" });
    expect(global.fetch).toHaveBeenCalledWith(
      `${PLATFORM_URL}/workspaces/ws-1/pause`,
      expect.objectContaining({ method: "POST" })
    );
  });

  test("handleResumeWorkspace calls POST /workspaces/:id/resume", async () => {
    global.fetch = mockFetch({ status: "provisioning" });
    await handleResumeWorkspace({ workspace_id: "ws-1" });
    expect(global.fetch).toHaveBeenCalledWith(
      `${PLATFORM_URL}/workspaces/ws-1/resume`,
      expect.objectContaining({ method: "POST" })
    );
  });

  test("handleImportOrg calls POST /org/import with dir", async () => {
    global.fetch = mockFetch({ org: "test", count: 5 });
    await handleImportOrg({ dir: "molecule-dev" });
    expect(global.fetch).toHaveBeenCalledWith(
      `${PLATFORM_URL}/org/import`,
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ dir: "molecule-dev" }),
      })
    );
  });
});

// ============================================================
// Phase 30 — Remote agent management tools
// ============================================================
describe("Phase 30 remote-agent tools", () => {
  test("handleListRemoteAgents filters runtime='external'", async () => {
    global.fetch = mockFetch([
      { id: "ws-1", name: "local", runtime: "claude-code", status: "online" },
      { id: "ws-2", name: "remote-a", runtime: "external", status: "online", url: "remote://a", last_heartbeat_at: "2026-04-13T12:00:00Z" },
      { id: "ws-3", name: "remote-b", runtime: "external", status: "offline", url: "remote://b" },
    ]);
    const res = await handleListRemoteAgents();
    const body = JSON.parse(res.content[0].text);
    expect(body.count).toBe(2);
    expect(body.agents.map((a: { id: string }) => a.id).sort()).toEqual(["ws-2", "ws-3"]);
    // Local workspace excluded
    expect(body.agents.find((a: { id: string }) => a.id === "ws-1")).toBeUndefined();
  });

  test("handleListRemoteAgents handles non-array response gracefully", async () => {
    global.fetch = mockFetch({ error: "boom" }, false, 500);
    const res = await handleListRemoteAgents();
    expect(res.content[0].text).toContain("HTTP 500");
  });

  test("handleGetRemoteAgentState projects the right fields", async () => {
    global.fetch = mockFetch({
      id: "ws-x", status: "paused", runtime: "external",
      last_heartbeat_at: "2026-04-13T12:00:00Z",
      agent_card: { name: "noisy" },  // deliberately omitted from projection
    });
    const res = await handleGetRemoteAgentState({ workspace_id: "ws-x" });
    const body = JSON.parse(res.content[0].text);
    expect(body.workspace_id).toBe("ws-x");
    expect(body.status).toBe("paused");
    expect(body.paused).toBe(true);
    expect(body.deleted).toBe(false);
    expect(body.runtime).toBe("external");
    expect(body.agent_card).toBeUndefined();
  });

  test("handleGetRemoteAgentSetupCommand requires runtime='external'", async () => {
    global.fetch = mockFetch({ id: "ws-local", name: "n", runtime: "claude-code" });
    const res = await handleGetRemoteAgentSetupCommand({ workspace_id: "ws-local" });
    const body = JSON.parse(res.content[0].text);
    expect(body.error).toContain("not external");
    expect(body.actual_runtime).toBe("claude-code");
  });

  test("handleGetRemoteAgentSetupCommand emits bash for external workspace", async () => {
    global.fetch = mockFetch({ id: "ws-ext", name: "remote-1", runtime: "external" });
    const res = await handleGetRemoteAgentSetupCommand({ workspace_id: "ws-ext" });
    const body = JSON.parse(res.content[0].text);
    expect(body.workspace_id).toBe("ws-ext");
    expect(body.workspace_name).toBe("remote-1");
    expect(body.setup_command).toContain("WORKSPACE_ID=ws-ext");
    expect(body.setup_command).toContain("PLATFORM_URL=");
    expect(body.setup_command).toContain("molecule-agent");
  });

  test("handleCheckRemoteAgentFreshness fresh when heartbeat is recent", async () => {
    const now = new Date();
    const recent = new Date(now.getTime() - 30_000).toISOString();
    global.fetch = mockFetch({
      id: "ws-fresh", status: "online", runtime: "external",
      last_heartbeat_at: recent,
    });
    const res = await handleCheckRemoteAgentFreshness({ workspace_id: "ws-fresh" });
    const body = JSON.parse(res.content[0].text);
    expect(body.fresh).toBe(true);
    expect(body.seconds_since_heartbeat).toBeLessThan(35);
    expect(body.threshold_seconds).toBe(90);
  });

  test("handleCheckRemoteAgentFreshness stale when past threshold", async () => {
    const now = new Date();
    const old = new Date(now.getTime() - 300_000).toISOString();
    global.fetch = mockFetch({
      id: "ws-stale", status: "online", runtime: "external",
      last_heartbeat_at: old,
    });
    const res = await handleCheckRemoteAgentFreshness({
      workspace_id: "ws-stale", threshold_seconds: 60,
    });
    const body = JSON.parse(res.content[0].text);
    expect(body.fresh).toBe(false);
    expect(body.seconds_since_heartbeat).toBeGreaterThan(60);
  });

  test("handleCheckRemoteAgentFreshness handles missing heartbeat", async () => {
    global.fetch = mockFetch({
      id: "ws-new", status: "online", runtime: "external",
      // last_heartbeat_at omitted entirely (just-registered agent)
    });
    const res = await handleCheckRemoteAgentFreshness({ workspace_id: "ws-new" });
    const body = JSON.parse(res.content[0].text);
    expect(body.fresh).toBe(false);
    expect(body.seconds_since_heartbeat).toBeNull();
  });
});
