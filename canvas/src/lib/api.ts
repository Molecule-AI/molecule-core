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
// the error and render a retry affordance. Callers that know the
// endpoint is intentionally slow (org import walks a tree of
// workspaces with server-side pacing) can pass `timeoutMs` to
// override.
const DEFAULT_TIMEOUT_MS = 15_000;

export interface RequestOptions {
  timeoutMs?: number;
}

async function request<T>(
  method: string,
  path: string,
  body?: unknown,
  retryCount = 0,
  options?: RequestOptions,
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
  const adminToken = process.env.NEXT_PUBLIC_ADMIN_TOKEN;
  if (adminToken) headers["Authorization"] = `Bearer ${adminToken}`;

  const res = await fetch(`${PLATFORM_URL}${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
    credentials: "include",
    signal: AbortSignal.timeout(options?.timeoutMs ?? DEFAULT_TIMEOUT_MS),
  });
  // Transient rate-limit recovery. A single IP bucket can momentarily
  // spike on page load (several panels hydrate simultaneously). Instead
  // of bubbling up a 429 that blanks the Canvas, wait the
  // Retry-After window and try once — any further 429 surfaces normally.
  // GET / idempotent methods only; never auto-retry mutations.
  if (res.status === 429 && retryCount === 0 && method === "GET") {
    const retryAfterHeader = res.headers.get("Retry-After");
    const retryAfter = retryAfterHeader ? parseInt(retryAfterHeader, 10) : NaN;
    const delayMs = Number.isFinite(retryAfter) ? Math.min(retryAfter, 20) * 1000 : 2000;
    await new Promise((resolve) => setTimeout(resolve, delayMs));
    return request<T>(method, path, body, retryCount + 1, options);
  }
  if (res.status === 401) {
    // Session expired or credentials lost. On SaaS (tenant subdomain)
    // the login page lives at /cp/auth/login and is mounted by the
    // control-plane reverse proxy — redirect. On self-hosted / local
    // dev / Vercel preview there IS no /cp/* mount, so redirecting
    // would navigate to a 404 ("404 page not found") instead of the
    // real error the user should see. In that case, throw instead
    // and let the caller render a meaningful failure (retry button,
    // error banner, etc.).
    if (slug) {
      const { redirectToLogin } = await import("./auth");
      redirectToLogin("sign-in");
      throw new Error("Session expired — redirecting to login");
    }
    throw new Error(`API ${method} ${path}: 401 ${await res.text()}`);
  }
  if (!res.ok) {
    const text = await res.text();
    // Recognise the platform's structured "datastore unreachable"
    // shape (returned by wsauth_middleware.abortAuthLookupError when
    // Postgres/Redis is down). Surface as a typed error so callers
    // can render a dedicated diagnostic instead of a generic toast.
    if (res.status === 503 && text) {
      try {
        const parsed = JSON.parse(text) as { code?: string; error?: string };
        if (parsed.code === "platform_unavailable") {
          throw new PlatformUnavailableError(parsed.error || "platform datastore unavailable");
        }
      } catch (err) {
        // Re-throw the typed error if that's what we just constructed.
        // JSON.parse failures fall through to the generic Error below.
        if (err instanceof PlatformUnavailableError) throw err;
      }
    }
    throw new Error(`API ${method} ${path}: ${res.status} ${text}`);
  }
  return res.json();
}

/** Thrown when the platform reports its datastore (Postgres/Redis) is
 *  unreachable. Surface with a dedicated diagnostic UI rather than a
 *  generic API-error toast — the user's next action is to check local
 *  services, not to retry the API call. */
export class PlatformUnavailableError extends Error {
  readonly code = "platform_unavailable" as const;
  constructor(message: string) {
    super(message);
    this.name = "PlatformUnavailableError";
  }
}

export const api = {
  get: <T>(path: string, options?: RequestOptions) => request<T>("GET", path, undefined, 0, options),
  post: <T>(path: string, body?: unknown, options?: RequestOptions) => request<T>("POST", path, body, 0, options),
  patch: <T>(path: string, body?: unknown, options?: RequestOptions) => request<T>("PATCH", path, body, 0, options),
  put: <T>(path: string, body?: unknown, options?: RequestOptions) => request<T>("PUT", path, body, 0, options),
  del: <T>(path: string, options?: RequestOptions) => request<T>("DELETE", path, undefined, 0, options),
};
