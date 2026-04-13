// Prefer MOLECULE_URL (the canonical MCP env var), fall back to PLATFORM_URL
// (what the workspace runtime already injects for heartbeat/register), and
// only then to localhost:8080. Injecting MOLECULE_URL at container provision
// is handled by platform/internal/provisioner/provisioner.go; this fallback
// chain protects older containers and host-side users alike. Fixes #67.
export const PLATFORM_URL =
  process.env.MOLECULE_URL ||
  process.env.PLATFORM_URL ||
  "http://localhost:8080";

export async function apiCall(method: string, path: string, body?: unknown) {
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
      return JSON.parse(text);
    } catch {
      return { raw: text, status: res.status };
    }
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err);
    console.error(`Molecule AI API error (${method} ${path}): ${msg}`);
    return { error: `Platform unreachable at ${PLATFORM_URL}`, detail: msg };
  }
}
