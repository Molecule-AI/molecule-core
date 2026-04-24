import { describe, it, expect, beforeEach, vi } from "vitest";

// Mock fetch globally
global.fetch = vi.fn(() =>
  Promise.resolve({ ok: true, json: () => Promise.resolve({}) } as Response),
);

import { useCanvasStore } from "../../store/canvas";
import type { WorkspaceData } from "../../store/socket";
import {
  DEFAULT_PROVISION_TIMEOUT_MS,
  RUNTIME_TIMEOUT_OVERRIDES_MS,
  timeoutForRuntime,
} from "../ProvisioningTimeout";

// Helper to build a WorkspaceData object
function makeWS(overrides: Partial<WorkspaceData> & { id: string }): WorkspaceData {
  return {
    name: "WS",
    role: "agent",
    tier: 1,
    status: "online",
    agent_card: null,
    url: "http://localhost:9000",
    parent_id: null,
    active_tasks: 0,
    last_error_rate: 0,
    last_sample_error: "",
    uptime_seconds: 60,
    current_task: "",
    x: 0,
    y: 0,
    collapsed: false,
    runtime: "",
    budget_limit: null,
    ...overrides,
  };
}

beforeEach(() => {
  useCanvasStore.setState({
    nodes: [],
    edges: [],
    selectedNodeId: null,
    panelTab: "details",
    dragOverNodeId: null,
    contextMenu: null,
    searchOpen: false,
    viewport: { x: 0, y: 0, zoom: 1 },
  });
  vi.clearAllMocks();
});

describe("ProvisioningTimeout", () => {
  it("exports the default timeout constant", () => {
    expect(DEFAULT_PROVISION_TIMEOUT_MS).toBe(120_000);
  });

  it("can detect provisioning nodes in the store", () => {
    useCanvasStore.getState().hydrate([
      makeWS({ id: "ws-1", name: "Agent 1", status: "provisioning" }),
      makeWS({ id: "ws-2", name: "Agent 2", status: "online" }),
      makeWS({ id: "ws-3", name: "Agent 3", status: "provisioning" }),
    ]);

    const nodes = useCanvasStore.getState().nodes;
    const provisioning = nodes.filter((n) => n.data.status === "provisioning");
    expect(provisioning).toHaveLength(2);
    expect(provisioning.map((n) => n.id)).toEqual(["ws-1", "ws-3"]);
  });

  it("transitions node from provisioning to online on event", () => {
    useCanvasStore.getState().hydrate([
      makeWS({ id: "ws-1", name: "Agent 1", status: "provisioning" }),
    ]);

    useCanvasStore.getState().applyEvent({
      event: "WORKSPACE_ONLINE",
      workspace_id: "ws-1",
      timestamp: new Date().toISOString(),
      payload: {},
    });

    const node = useCanvasStore.getState().nodes.find((n) => n.id === "ws-1");
    expect(node?.data.status).toBe("online");
  });

  it("transitions node from provisioning to failed on provision_failed event", () => {
    useCanvasStore.getState().hydrate([
      makeWS({ id: "ws-1", name: "Agent 1", status: "provisioning" }),
    ]);

    useCanvasStore.getState().applyEvent({
      event: "WORKSPACE_PROVISION_FAILED",
      workspace_id: "ws-1",
      timestamp: new Date().toISOString(),
      payload: { error: "Docker daemon not running" },
    });

    const node = useCanvasStore.getState().nodes.find((n) => n.id === "ws-1");
    expect(node?.data.status).toBe("failed");
    expect(node?.data.lastSampleError).toBe("Docker daemon not running");
  });

  it("handles WORKSPACE_PROVISION_FAILED with no error message", () => {
    useCanvasStore.getState().hydrate([
      makeWS({ id: "ws-1", name: "Agent 1", status: "provisioning" }),
    ]);

    useCanvasStore.getState().applyEvent({
      event: "WORKSPACE_PROVISION_FAILED",
      workspace_id: "ws-1",
      timestamp: new Date().toISOString(),
      payload: {},
    });

    const node = useCanvasStore.getState().nodes.find((n) => n.id === "ws-1");
    expect(node?.data.status).toBe("failed");
    expect(node?.data.lastSampleError).toBe("Unknown provisioning error");
  });

  it("restart API call can be made for provisioning recovery", async () => {
    (global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({}),
    } as Response);

    useCanvasStore.getState().hydrate([
      makeWS({ id: "ws-1", name: "Agent 1", status: "failed" }),
    ]);

    await useCanvasStore.getState().restartWorkspace("ws-1");

    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/workspaces/ws-1/restart"),
      expect.objectContaining({ method: "POST" }),
    );
  });

  it("node removal works for cancelling a failed deployment", () => {
    useCanvasStore.getState().hydrate([
      makeWS({ id: "ws-1", name: "Agent 1", status: "provisioning" }),
      makeWS({ id: "ws-2", name: "Agent 2", status: "online" }),
    ]);

    expect(useCanvasStore.getState().nodes).toHaveLength(2);

    useCanvasStore.getState().removeNode("ws-1");

    expect(useCanvasStore.getState().nodes).toHaveLength(1);
    expect(useCanvasStore.getState().nodes[0].id).toBe("ws-2");
  });

  it("selectNode and setPanelTab work for view logs action", () => {
    useCanvasStore.getState().hydrate([
      makeWS({ id: "ws-1", name: "Agent 1", status: "provisioning" }),
    ]);

    useCanvasStore.getState().selectNode("ws-1");
    expect(useCanvasStore.getState().selectedNodeId).toBe("ws-1");

    useCanvasStore.getState().setPanelTab("terminal");
    expect(useCanvasStore.getState().panelTab).toBe("terminal");
  });

  it("multiple provisioning nodes can coexist", () => {
    useCanvasStore.getState().hydrate([
      makeWS({ id: "ws-1", name: "Agent 1", status: "provisioning" }),
      makeWS({ id: "ws-2", name: "Agent 2", status: "provisioning" }),
      makeWS({ id: "ws-3", name: "Agent 3", status: "provisioning" }),
    ]);

    const provisioning = useCanvasStore
      .getState()
      .nodes.filter((n) => n.data.status === "provisioning");
    expect(provisioning).toHaveLength(3);

    // First one goes online
    useCanvasStore.getState().applyEvent({
      event: "WORKSPACE_ONLINE",
      workspace_id: "ws-1",
      timestamp: new Date().toISOString(),
      payload: {},
    });

    const stillProvisioning = useCanvasStore
      .getState()
      .nodes.filter((n) => n.data.status === "provisioning");
    expect(stillProvisioning).toHaveLength(2);
  });

  // ── Runtime-aware timeout regression tests (2026-04-24 outage) ────────────
  // Prior to this, a hermes workspace consistently false-alarmed at 2 min
  // into its 8-13 min cold boot, pushing users to retry something that
  // would have come online on its own. The runtime-aware override keeps
  // the 2-min floor for fast docker runtimes while giving hermes its
  // honest 12-min budget.

  describe("timeoutForRuntime", () => {
    it("returns the 2-min default for unknown/missing runtimes", () => {
      expect(timeoutForRuntime(undefined)).toBe(DEFAULT_PROVISION_TIMEOUT_MS);
      expect(timeoutForRuntime("")).toBe(DEFAULT_PROVISION_TIMEOUT_MS);
      expect(timeoutForRuntime("some-future-runtime")).toBe(
        DEFAULT_PROVISION_TIMEOUT_MS,
      );
    });

    it("returns the docker-fast 2-min default for known-fast runtimes", () => {
      // These aren't in the override map so they get the default.
      // If someone ever adds one of them to RUNTIME_TIMEOUT_OVERRIDES_MS,
      // this test catches the accidental regression.
      expect(timeoutForRuntime("claude-code")).toBe(DEFAULT_PROVISION_TIMEOUT_MS);
      expect(timeoutForRuntime("langgraph")).toBe(DEFAULT_PROVISION_TIMEOUT_MS);
      expect(timeoutForRuntime("crewai")).toBe(DEFAULT_PROVISION_TIMEOUT_MS);
    });

    it("returns 12 min for hermes — covers cold-boot install tail", () => {
      expect(timeoutForRuntime("hermes")).toBe(720_000);
      expect(timeoutForRuntime("hermes")).toBe(
        RUNTIME_TIMEOUT_OVERRIDES_MS.hermes,
      );
    });

    it("hermes override is materially longer than the default", () => {
      // Guard against future refactors that accidentally weaken the
      // override (e.g. typo lowering hermes to 72_000 = 72s).
      expect(RUNTIME_TIMEOUT_OVERRIDES_MS.hermes).toBeGreaterThanOrEqual(
        DEFAULT_PROVISION_TIMEOUT_MS * 5,
      );
    });
  });
});
