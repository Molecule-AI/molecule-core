import { getTenantSlug } from "./tenant";

// When NEXT_PUBLIC_PLATFORM_URL is set to "" (empty string), the canvas
// uses relative paths — correct for the combined tenant image where Go
// platform + canvas run on the same port via reverse proxy. The `??`
// operator preserves "" as a valid value; `||` would fall through to
// the localhost default.
export const PLATFORM_URL =
  process.env.NEXT_PUBLIC_PLATFORM_URL ?? "http://localhost:8080";

// 15s is long enough for slow CP queries but short enough that a
// hung backend doesn't leave the UI spinning forever. The abort
// propagates through AbortController so React components can observe
// the error and render a retry affordance.
const DEFAULT_TIMEOUT_MS = 15_000;

async function request<T>(
  method: string,
  path: string,
  body?: unknown
): Promise<T> {
  // SaaS cross-origin shape:
  //  - X-Molecule-Org-Slug: derived from window.location.hostname by
  //    getTenantSlug(). Control plane uses it for fly-replay routing.
  //    Empty on localhost / non-tenant hosts — safe to omit.
  //  - credentials:"include": sends the session cookie cross-origin.
  //    Cookie's Domain=.moleculesai.app attribute + cp's CORS allow this.
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  const slug = getTenantSlug();
  if (slug) headers["X-Molecule-Org-Slug"] = slug;

  const res = await fetch(`${PLATFORM_URL}${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
    credentials: "include",
    signal: AbortSignal.timeout(DEFAULT_TIMEOUT_MS),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`API ${method} ${path}: ${res.status} ${text}`);
  }
  return res.json();
}

export const api = {
  get: <T>(path: string) => request<T>("GET", path),
  post: <T>(path: string, body?: unknown) => request<T>("POST", path, body),
  patch: <T>(path: string, body?: unknown) => request<T>("PATCH", path, body),
  put: <T>(path: string, body?: unknown) => request<T>("PUT", path, body),
  del: <T>(path: string) => request<T>("DELETE", path),
};
