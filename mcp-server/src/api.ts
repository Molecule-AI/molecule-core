// Prefer MOLECULE_URL (the canonical MCP env var), fall back to PLATFORM_URL
// (what the workspace runtime already injects for heartbeat/register), and
// only then to localhost:8080. Injecting MOLECULE_URL at container provision
// is handled by platform/internal/provisioner/provisioner.go; this fallback
// chain protects older containers and host-side users alike. Fixes #67.
export const PLATFORM_URL =
  process.env.MOLECULE_URL ||
  process.env.PLATFORM_URL ||
  "http://localhost:8080";

/**
 * Shape returned by apiCall when the request fails (network error, non-2xx,
 * or non-JSON body with no error). Returned-by-value — apiCall never throws.
 */
export type ApiError = { error: string; detail?: string; raw?: string; status?: number };

export function isApiError(v: unknown): v is ApiError {
  return !!v && typeof v === "object" && "error" in (v as object);
}

/**
 * Wrap arbitrary JSON-serialisable data in the MCP content envelope that
 * tool handlers must return. Centralised so every handler uses the exact
 * same shape (and a future switch to e.g. structured content happens once).
 */
export function toMcpResult(data: unknown) {
  return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
}

/**
 * Wrap a plain string (file contents, assistant reply text, error message)
 * in the MCP content envelope without JSON-stringifying it. For the handful
 * of handlers that return raw text rather than a JSON blob.
 */
export function toMcpText(text: string) {
  return { content: [{ type: "text" as const, text }] };
}

export async function apiCall<T = unknown>(
  method: string,
  path: string,
  body?: unknown,
): Promise<T | ApiError> {
  try {
    const res = await fetch(`${PLATFORM_URL}${path}`, {
      method,
      headers: { "Content-Type": "application/json" },
      body: body ? JSON.stringify(body) : undefined,
    });
    if (!res.ok) {
      const text = await res.text();
      return { error: `HTTP ${res.status}`, detail: text };
    }
    const text = await res.text();
    try {
      return JSON.parse(text) as T;
    } catch {
      return { raw: text, status: res.status } as ApiError;
    }
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err);
    // stdio MCP servers must log to stderr; stdout is the protocol channel.
    console.error(`Molecule AI API error (${method} ${path}): ${msg}`);
    return { error: `Platform unreachable at ${PLATFORM_URL}`, detail: msg };
  }
}
