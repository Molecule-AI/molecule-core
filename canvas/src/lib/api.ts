import { getTenantSlug } from "./tenant";

export const PLATFORM_URL =
  process.env.NEXT_PUBLIC_PLATFORM_URL || "http://localhost:8080";

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
