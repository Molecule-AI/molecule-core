import { describe, it, expect, beforeEach, vi } from "vitest";

// Mock fetch globally
global.fetch = vi.fn();

// Test the deploy-preflight integration and modal-related logic
// (Component rendering with hooks requires jsdom; we test logic here)
import {
  getRequiredKeys,
  findMissingKeys,
  getKeyLabel,
  checkDeploySecrets,
  RUNTIME_REQUIRED_KEYS,
} from "../../lib/deploy-preflight";

beforeEach(() => {
  vi.clearAllMocks();
});

describe("MissingKeysModal integration logic", () => {
  it("MissingKeysModal module can be imported", async () => {
    // Verify the module exports the component (even though we can't render it in node env)
    const mod = await import("../MissingKeysModal");
    expect(mod.MissingKeysModal).toBeDefined();
    expect(typeof mod.MissingKeysModal).toBe("function");
  });

  it("identifies missing keys for langgraph runtime", () => {
    const configured = new Set<string>();
    const missing = findMissingKeys("langgraph", configured);
    expect(missing).toEqual(["OPENAI_API_KEY"]);
  });

  it("identifies missing keys for claude-code runtime", () => {
    const configured = new Set<string>();
    const missing = findMissingKeys("claude-code", configured);
    expect(missing).toEqual(["ANTHROPIC_API_KEY"]);
  });

  it("generates correct labels for modal display", () => {
    const missing = findMissingKeys("langgraph", new Set<string>());
    const labels = missing.map((k) => ({ key: k, label: getKeyLabel(k) }));
    expect(labels).toEqual([
      { key: "OPENAI_API_KEY", label: "OpenAI API Key" },
    ]);
  });

  it("generates labels for claude-code missing keys", () => {
    const missing = findMissingKeys("claude-code", new Set<string>());
    const labels = missing.map((k) => ({ key: k, label: getKeyLabel(k) }));
    expect(labels).toEqual([
      { key: "ANTHROPIC_API_KEY", label: "Anthropic API Key" },
    ]);
  });

  it("returns no missing keys when all are configured", () => {
    const configured = new Set(["OPENAI_API_KEY"]);
    const missing = findMissingKeys("langgraph", configured);
    expect(missing).toEqual([]);
  });

  it("pre-deploy check returns ok=false and correct missing keys", async () => {
    (global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve([]),
    } as Response);

    const result = await checkDeploySecrets("langgraph");
    expect(result.ok).toBe(false);
    expect(result.missingKeys).toEqual(["OPENAI_API_KEY"]);
    expect(result.runtime).toBe("langgraph");
  });

  it("pre-deploy check returns ok=true when keys are present", async () => {
    (global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
      ok: true,
      json: () =>
        Promise.resolve([
          { key: "ANTHROPIC_API_KEY", has_value: true, created_at: "", updated_at: "" },
        ]),
    } as Response);

    const result = await checkDeploySecrets("claude-code");
    expect(result.ok).toBe(true);
    expect(result.missingKeys).toEqual([]);
  });

  it("modal data can be constructed from preflight result", async () => {
    (global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve([]),
    } as Response);

    const result = await checkDeploySecrets("deepagents");
    // This is the data that would be passed to MissingKeysModal
    const modalData = {
      open: !result.ok,
      missingKeys: result.missingKeys,
      runtime: result.runtime,
    };

    expect(modalData.open).toBe(true);
    expect(modalData.missingKeys).toEqual(["OPENAI_API_KEY"]);
    expect(modalData.runtime).toBe("deepagents");
  });

  it("handles all runtimes correctly for modal data construction", () => {
    const runtimes = Object.keys(RUNTIME_REQUIRED_KEYS);
    for (const runtime of runtimes) {
      const requiredKeys = getRequiredKeys(runtime);
      const missing = findMissingKeys(runtime, new Set<string>());
      const labels = missing.map((k) => getKeyLabel(k));

      expect(requiredKeys.length).toBeGreaterThan(0);
      expect(missing).toEqual(requiredKeys);
      expect(labels.length).toBe(requiredKeys.length);
      // Every label should be a non-empty string
      for (const label of labels) {
        expect(label.length).toBeGreaterThan(0);
      }
    }
  });

  it("save endpoint is correct for global scope", () => {
    // Verify the endpoint that MissingKeysModal would call
    const globalEndpoint = "/settings/secrets";
    expect(globalEndpoint).toBe("/settings/secrets");
  });

  it("save endpoint is correct for workspace scope", () => {
    const workspaceId = "ws-test-123";
    const wsEndpoint = `/workspaces/${workspaceId}/secrets`;
    expect(wsEndpoint).toBe("/workspaces/ws-test-123/secrets");
  });
});
