// @vitest-environment node
/**
 * MissingKeysModal preflight logic tests.
 * Component rendering tested in MissingKeysModal.component.test.tsx.
 */
import { describe, it, expect, beforeEach, vi } from "vitest";

global.fetch = vi.fn();

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

describe("MissingKeysModal preflight logic", () => {
  it("identifies missing keys for langgraph runtime", () => {
    const missing = findMissingKeys("langgraph", new Set<string>());
    expect(missing).toEqual(["OPENAI_API_KEY"]);
  });

  it("identifies missing keys for claude-code runtime", () => {
    const missing = findMissingKeys("claude-code", new Set<string>());
    expect(missing).toEqual(["ANTHROPIC_API_KEY"]);
  });

  it("generates correct labels for modal display", () => {
    const missing = findMissingKeys("langgraph", new Set<string>());
    const labels = missing.map((k) => ({ key: k, label: getKeyLabel(k) }));
    expect(labels).toEqual([{ key: "OPENAI_API_KEY", label: "OpenAI API Key" }]);
  });

  it("returns no missing keys when all are configured", () => {
    const missing = findMissingKeys("langgraph", new Set(["OPENAI_API_KEY"]));
    expect(missing).toEqual([]);
  });

  it("pre-deploy check returns ok=false and correct missing keys", async () => {
    (global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve([]),
    } as Response);

    const result = await checkDeploySecrets("langgraph");
    expect(result.ok).toBe(false);
    // langgraph accepts OpenAI, Anthropic, or OpenRouter — when none are
    // configured we surface all three so the picker modal can offer a choice.
    expect(result.missingKeys).toEqual([
      "OPENAI_API_KEY",
      "ANTHROPIC_API_KEY",
      "OPENROUTER_API_KEY",
    ]);
    expect(result.runtime).toBe("langgraph");
  });

  it("pre-deploy check returns ok=true when keys are present", async () => {
    (global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce({
      ok: true,
      json: () =>
        Promise.resolve([{ key: "ANTHROPIC_API_KEY", has_value: true, created_at: "", updated_at: "" }]),
    } as Response);

    const result = await checkDeploySecrets("claude-code");
    expect(result.ok).toBe(true);
    expect(result.missingKeys).toEqual([]);
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
      for (const label of labels) {
        expect(label.length).toBeGreaterThan(0);
      }
    }
  });
});