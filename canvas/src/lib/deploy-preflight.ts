/**
 * Pre-deploy secret check per runtime.
 *
 * Before a workspace is deployed, validates that all required secrets/env vars
 * are configured for the target runtime. Each runtime defines its own set of
 * required keys (derived from each runtime's config.yaml `env.required` field).
 */

import { api } from "./api";

/* ---------- Required keys per runtime ---------- */

export const RUNTIME_REQUIRED_KEYS: Record<string, string[]> = {
  langgraph: ["OPENAI_API_KEY"],
  "claude-code": ["ANTHROPIC_API_KEY"],
  openclaw: ["OPENAI_API_KEY"],
  deepagents: ["OPENAI_API_KEY"],
  crewai: ["OPENAI_API_KEY"],
  autogen: ["OPENAI_API_KEY"],
};

/** Human-readable labels for common secret keys */
export const KEY_LABELS: Record<string, string> = {
  OPENAI_API_KEY: "OpenAI API Key",
  ANTHROPIC_API_KEY: "Anthropic API Key",
  GOOGLE_API_KEY: "Google AI API Key",
  SERP_API_KEY: "SERP API Key",
  OPENROUTER_API_KEY: "OpenRouter API Key",
};

/* ---------- Types ---------- */

export interface SecretEntry {
  key: string;
  has_value: boolean;
  created_at: string;
  updated_at: string;
  scope?: "global" | "workspace";
}

export interface PreflightResult {
  ok: boolean;
  missingKeys: string[];
  runtime: string;
}

/* ---------- Pure helpers (easily testable) ---------- */

/** Get required env keys for a given runtime. Returns empty array for unknown runtimes. */
export function getRequiredKeys(runtime: string): string[] {
  return RUNTIME_REQUIRED_KEYS[runtime] ?? [];
}

/** Given a runtime and a set of configured key names, return which keys are missing. */
export function findMissingKeys(
  runtime: string,
  configuredKeys: Set<string>,
): string[] {
  return getRequiredKeys(runtime).filter((k) => !configuredKeys.has(k));
}

/** Get human-readable label for a key, or fall back to the key itself. */
export function getKeyLabel(key: string): string {
  return KEY_LABELS[key] ?? key;
}

/* ---------- API-calling preflight check ---------- */

/**
 * Fetch configured secrets from the platform and check whether all required
 * keys for the target runtime are present.
 *
 * If `workspaceId` is provided, fetches the merged (global + workspace) secret
 * list for that workspace. Otherwise falls back to global secrets only.
 */
export async function checkDeploySecrets(
  runtime: string,
  workspaceId?: string,
): Promise<PreflightResult> {
  const requiredKeys = getRequiredKeys(runtime);
  if (requiredKeys.length === 0) {
    return { ok: true, missingKeys: [], runtime };
  }

  try {
    const secrets = workspaceId
      ? await api.get<SecretEntry[]>(`/workspaces/${workspaceId}/secrets`)
      : await api.get<SecretEntry[]>("/settings/secrets");

    const configuredKeys = new Set(
      secrets.filter((s) => s.has_value).map((s) => s.key),
    );

    const missingKeys = findMissingKeys(runtime, configuredKeys);
    return { ok: missingKeys.length === 0, missingKeys, runtime };
  } catch (error) {
    // Log the error before falling back — aids debugging when the API is down.
    console.error("[deploy-preflight] Failed to check secrets, assuming all missing:", error);
    // If we can't reach the secrets API, assume missing — safer to prompt the user.
    return { ok: false, missingKeys: requiredKeys, runtime };
  }
}
