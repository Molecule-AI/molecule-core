/**
 * Pre-deploy secret check per runtime.
 *
 * Before a workspace is deployed, validates that all required secrets/env vars
 * are configured for the target runtime. Each runtime defines its own set of
 * required keys (derived from each runtime's config.yaml `env.required` field).
 */

import { api } from "./api";

/* ---------- Required keys per runtime ----------
 *
 * A runtime may accept ANY of several provider keys (Hermes speaks
 * OpenRouter or OpenAI or its native Nous API; LangGraph speaks
 * OpenAI or Anthropic; …). Represent that as a list of provider
 * choices — the UI renders a picker when length > 1, and the
 * preflight check treats the runtime as satisfied if *any one* of
 * the listed keys is configured.
 *
 * The first entry is the default / recommended provider for that
 * runtime.
 */

export interface ProviderChoice {
  /** Stable id for the provider. Used as React key + picker value. */
  id: string;
  /** Human label shown in the provider picker. */
  label: string;
  /** Env var name the workspace container reads at runtime. */
  envVar: string;
  /** Short rationale shown under the picker option, optional. */
  note?: string;
}

export const RUNTIME_PROVIDERS: Record<string, ProviderChoice[]> = {
  langgraph: [
    { id: "openai", label: "OpenAI", envVar: "OPENAI_API_KEY" },
    { id: "anthropic", label: "Anthropic", envVar: "ANTHROPIC_API_KEY" },
    { id: "openrouter", label: "OpenRouter (proxy — any model)", envVar: "OPENROUTER_API_KEY", note: "Broadest model coverage incl. Minimax, DeepSeek, Groq" },
  ],
  "claude-code": [
    { id: "anthropic", label: "Anthropic", envVar: "ANTHROPIC_API_KEY" },
  ],
  openclaw: [
    { id: "openai", label: "OpenAI", envVar: "OPENAI_API_KEY" },
    { id: "openrouter", label: "OpenRouter", envVar: "OPENROUTER_API_KEY" },
  ],
  deepagents: [
    { id: "openai", label: "OpenAI", envVar: "OPENAI_API_KEY" },
    { id: "anthropic", label: "Anthropic", envVar: "ANTHROPIC_API_KEY" },
    { id: "openrouter", label: "OpenRouter", envVar: "OPENROUTER_API_KEY" },
  ],
  crewai: [
    { id: "openai", label: "OpenAI", envVar: "OPENAI_API_KEY" },
    { id: "anthropic", label: "Anthropic", envVar: "ANTHROPIC_API_KEY" },
  ],
  autogen: [
    { id: "openai", label: "OpenAI", envVar: "OPENAI_API_KEY" },
    { id: "openrouter", label: "OpenRouter", envVar: "OPENROUTER_API_KEY" },
  ],
  hermes: [
    { id: "openrouter", label: "OpenRouter", envVar: "OPENROUTER_API_KEY", note: "Recommended — widest model coverage (Minimax, DeepSeek, Llama, …)" },
    { id: "openai", label: "OpenAI", envVar: "OPENAI_API_KEY" },
    { id: "hermes-native", label: "Nous Research (Hermes native)", envVar: "HERMES_API_KEY" },
  ],
  "gemini-cli": [
    { id: "google", label: "Google AI", envVar: "GOOGLE_API_KEY" },
  ],
};

/** Back-compat: flat list of the DEFAULT (first) env var per runtime.
 *  Preserved so existing callers keep working; the richer provider-
 *  aware UX consumes RUNTIME_PROVIDERS directly. */
export const RUNTIME_REQUIRED_KEYS: Record<string, string[]> = Object.fromEntries(
  Object.entries(RUNTIME_PROVIDERS).map(([rt, choices]) => [rt, [choices[0].envVar]]),
);

/** Human-readable labels for common secret keys */
export const KEY_LABELS: Record<string, string> = {
  OPENAI_API_KEY: "OpenAI API Key",
  ANTHROPIC_API_KEY: "Anthropic API Key",
  GOOGLE_API_KEY: "Google AI API Key",
  SERP_API_KEY: "SERP API Key",
  OPENROUTER_API_KEY: "OpenRouter API Key",
  HERMES_API_KEY: "Nous Research API Key",
  DEEPSEEK_API_KEY: "DeepSeek API Key",
};

/** Get the provider choices for a runtime. Returns [] for unknown runtimes. */
export function getRuntimeProviders(runtime: string): ProviderChoice[] {
  return RUNTIME_PROVIDERS[runtime] ?? [];
}

/** Returns the first provider choice whose env var is in `configured`,
 *  or null if none are set. Used to auto-skip the picker when the
 *  user has already wired up a supported provider. */
export function findConfiguredProvider(
  runtime: string,
  configured: Set<string>,
): ProviderChoice | null {
  for (const p of getRuntimeProviders(runtime)) {
    if (configured.has(p.envVar)) return p;
  }
  return null;
}

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
  const providers = getRuntimeProviders(runtime);
  if (providers.length === 0) {
    // Unknown runtime — nothing to preflight.
    return { ok: true, missingKeys: [], runtime };
  }

  try {
    const secrets = workspaceId
      ? await api.get<SecretEntry[]>(`/workspaces/${workspaceId}/secrets`)
      : await api.get<SecretEntry[]>("/settings/secrets");

    const configuredKeys = new Set(
      secrets.filter((s) => s.has_value).map((s) => s.key),
    );

    // If ANY supported provider's key is already set we're satisfied —
    // the picker is only for "none yet" cases.
    if (findConfiguredProvider(runtime, configuredKeys)) {
      return { ok: true, missingKeys: [], runtime };
    }

    // Nothing configured — surface every supported provider so the
    // modal can render a picker. The default (first) still renders at
    // the top.
    const missingKeys = providers.map((p) => p.envVar);
    return { ok: false, missingKeys, runtime };
  } catch (error) {
    // Log the error before falling back — aids debugging when the API is down.
    console.error("[deploy-preflight] Failed to check secrets, assuming all missing:", error);
    // If we can't reach the secrets API, assume missing — safer to prompt the user.
    return {
      ok: false,
      missingKeys: providers.map((p) => p.envVar),
      runtime,
    };
  }
}
