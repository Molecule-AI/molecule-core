import { describe, it, expect, beforeEach, vi } from "vitest";

// Mock fetch globally before importing the module
global.fetch = vi.fn();

import {
  getRequiredKeys,
  findMissingKeys,
  getKeyLabel,
  checkDeploySecrets,
  RUNTIME_REQUIRED_KEYS,
  KEY_LABELS,
} from "../deploy-preflight";

beforeEach(() => {
  vi.clearAllMocks();
});

/* ---------- getRequiredKeys ---------- */

describe("getRequiredKeys", () => {
  it("returns OPENAI_API_KEY for langgraph", () => {
    expect(getRequiredKeys("langgraph")).toEqual(["OPENAI_API_KEY"]);
  });

  it("returns ANTHROPIC_API_KEY for claude-code", () => {
    expect(getRequiredKeys("claude-code")).toEqual(["ANTHROPIC_API_KEY"]);
  });

  it("returns OPENAI_API_KEY for crewai", () => {
    expect(getRequiredKeys("crewai")).toEqual(["OPENAI_API_KEY"]);
  });

  it("returns OPENAI_API_KEY for autogen", () => {
    expect(getRequiredKeys("autogen")).toEqual(["OPENAI_API_KEY"]);
  });

  it("returns OPENAI_API_KEY for openclaw", () => {
    expect(getRequiredKeys("openclaw")).toEqual(["OPENAI_API_KEY"]);
  });

  it("returns OPENAI_API_KEY for deepagents", () => {
    expect(getRequiredKeys("deepagents")).toEqual(["OPENAI_API_KEY"]);
  });

  it("returns empty array for unknown runtimes", () => {
    expect(getRequiredKeys("unknown-runtime")).toEqual([]);
    expect(getRequiredKeys("")).toEqual([]);
  });
});

/* ---------- findMissingKeys ---------- */

describe("findMissingKeys", () => {
  it("returns empty array when all keys are configured", () => {
    const configured = new Set(["OPENAI_API_KEY", "OTHER_KEY"]);
    expect(findMissingKeys("langgraph", configured)).toEqual([]);
  });

  it("returns missing keys when not configured", () => {
    const configured = new Set(["OTHER_KEY"]);
    expect(findMissingKeys("langgraph", configured)).toEqual(["OPENAI_API_KEY"]);
  });

  it("returns empty array for runtime with no required keys", () => {
    const configured = new Set<string>();
    expect(findMissingKeys("unknown-runtime", configured)).toEqual([]);
  });

  it("returns all required keys when nothing is configured", () => {
    const configured = new Set<string>();
    expect(findMissingKeys("claude-code", configured)).toEqual(["ANTHROPIC_API_KEY"]);
  });

  it("handles empty configured set for multi-key runtimes", () => {
    const configured = new Set<string>();
    const result = findMissingKeys("langgraph", configured);
    expect(result).toEqual(["OPENAI_API_KEY"]);
  });
});

/* ---------- getKeyLabel ---------- */

describe("getKeyLabel", () => {
  it("returns label for known keys", () => {
    expect(getKeyLabel("OPENAI_API_KEY")).toBe("OpenAI API Key");
    expect(getKeyLabel("ANTHROPIC_API_KEY")).toBe("Anthropic API Key");
  });

  it("returns the key itself for unknown keys", () => {
    expect(getKeyLabel("CUSTOM_SECRET")).toBe("CUSTOM_SECRET");
  });
});

/* ---------- RUNTIME_REQUIRED_KEYS ---------- */

describe("RUNTIME_REQUIRED_KEYS", () => {
  it("covers all six standard runtimes", () => {
    const runtimes = Object.keys(RUNTIME_REQUIRED_KEYS);
    expect(runtimes).toContain("langgraph");
    expect(runtimes).toContain("claude-code");
    expect(runtimes).toContain("openclaw");
    expect(runtimes).toContain("deepagents");
    expect(runtimes).toContain("crewai");
    expect(runtimes).toContain("autogen");
  });

  it("each runtime has at least one required key", () => {
    for (const [runtime, keys] of Object.entries(RUNTIME_REQUIRED_KEYS)) {
      expect(keys.length).toBeGreaterThan(0);
    }
  });
});

/* ---------- checkDeploySecrets ---------- */

describe("checkDeploySecrets", () => {
  it("returns ok=true when all required keys have values", async () => {
    (global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
      ok: true,
      json: () =>
        Promise.resolve([
          { key: "OPENAI_API_KEY", has_value: true, created_at: "", updated_at: "" },
        ]),
    } as Response);

    const result = await checkDeploySecrets("langgraph");
    expect(result.ok).toBe(true);
    expect(result.missingKeys).toEqual([]);
    expect(result.runtime).toBe("langgraph");
  });

  it("returns ok=false when required keys are missing", async () => {
    (global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
      ok: true,
      json: () =>
        Promise.resolve([
          { key: "OTHER_KEY", has_value: true, created_at: "", updated_at: "" },
        ]),
    } as Response);

    const result = await checkDeploySecrets("langgraph");
    expect(result.ok).toBe(false);
    expect(result.missingKeys).toEqual(["OPENAI_API_KEY"]);
  });

  it("returns ok=false when secret exists but has_value is false", async () => {
    (global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
      ok: true,
      json: () =>
        Promise.resolve([
          { key: "OPENAI_API_KEY", has_value: false, created_at: "", updated_at: "" },
        ]),
    } as Response);

    const result = await checkDeploySecrets("langgraph");
    expect(result.ok).toBe(false);
    expect(result.missingKeys).toEqual(["OPENAI_API_KEY"]);
  });

  it("returns ok=true for runtimes with no required keys", async () => {
    const result = await checkDeploySecrets("unknown-runtime");
    expect(result.ok).toBe(true);
    expect(result.missingKeys).toEqual([]);
    // Should not have called fetch
    expect(global.fetch).not.toHaveBeenCalled();
  });

  it("uses workspace-specific endpoint when workspaceId is provided", async () => {
    (global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
      ok: true,
      json: () =>
        Promise.resolve([
          { key: "ANTHROPIC_API_KEY", has_value: true, created_at: "", updated_at: "" },
        ]),
    } as Response);

    const result = await checkDeploySecrets("claude-code", "ws-123");
    expect(result.ok).toBe(true);
    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/workspaces/ws-123/secrets"),
      expect.any(Object),
    );
  });

  it("uses global secrets endpoint when no workspaceId", async () => {
    (global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve([]),
    } as Response);

    await checkDeploySecrets("langgraph");
    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/settings/secrets"),
      expect.any(Object),
    );
  });

  it("treats API failure as all keys missing (safe default)", async () => {
    (global.fetch as ReturnType<typeof vi.fn>).mockRejectedValueOnce(
      new Error("Network error"),
    );

    const result = await checkDeploySecrets("langgraph");
    expect(result.ok).toBe(false);
    expect(result.missingKeys).toEqual(["OPENAI_API_KEY"]);
  });
});
