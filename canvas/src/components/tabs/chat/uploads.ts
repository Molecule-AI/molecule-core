import { PLATFORM_URL } from "@/lib/api";
import { getTenantSlug } from "@/lib/tenant";
import type { ChatAttachment } from "./types";

/** Chat attachments are intentionally uploaded via a direct fetch()
 *  instead of the `api.post` helper — `api.post` JSON-stringifies the
 *  body, which would 500 on a Blob. Mirrors the header plumbing
 *  (tenant slug, admin token, credentials) so SaaS + self-hosted
 *  callers work the same way. */
export async function uploadChatFiles(
  workspaceId: string,
  files: File[],
): Promise<ChatAttachment[]> {
  if (files.length === 0) return [];

  const form = new FormData();
  for (const f of files) form.append("files", f, f.name);

  const headers: Record<string, string> = {};
  const slug = getTenantSlug();
  if (slug) headers["X-Molecule-Org-Slug"] = slug;
  const adminToken = process.env.NEXT_PUBLIC_ADMIN_TOKEN;
  if (adminToken) headers["Authorization"] = `Bearer ${adminToken}`;

  // Uploads legitimately take a while on cold cache (tar write +
  // docker cp into the container). 60s is comfortable for the 25MB/
  // 50MB caps the server enforces.
  const res = await fetch(`${PLATFORM_URL}/workspaces/${workspaceId}/chat/uploads`, {
    method: "POST",
    headers,
    body: form,
    credentials: "include",
    signal: AbortSignal.timeout(60_000),
  });
  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(`upload failed: ${res.status} ${text}`);
  }
  const json = (await res.json()) as { files: ChatAttachment[] };
  return json.files ?? [];
}

/** Resolve a file URI into a browser-downloadable URL. Accepts:
 *    - `workspace:<abs-path>` (our canonical form)
 *    - `file:///workspace/...` (some agents emit this)
 *    - `/workspace/...` (bare absolute path inside the container)
 *  Everything that looks like an allowed-root container path is
 *  rewritten to the authenticated /chat/download endpoint. HTTP(S)
 *  URIs pass through unchanged so we can also render links to
 *  artefacts hosted off-platform. Unknown schemes fall back to the
 *  raw URI — the caller gets to decide how to render it. */
export function resolveAttachmentHref(
  workspaceId: string,
  uri: string,
): string {
  const containerPath = normalizeWorkspaceUri(uri);
  if (containerPath) {
    return `${PLATFORM_URL}/workspaces/${workspaceId}/chat/download?path=${encodeURIComponent(containerPath)}`;
  }
  return uri;
}

/** Extracts the absolute container path from a workspace-scoped URI,
 *  or null if the URI isn't a container path. The matching roots
 *  mirror the server's `allowedRoots` allowlist. */
const ALLOWED_CONTAINER_ROOTS = ["/configs", "/workspace", "/home", "/plugins"];

function normalizeWorkspaceUri(uri: string): string | null {
  let path: string | null = null;
  if (uri.startsWith("workspace:")) {
    path = uri.slice("workspace:".length);
  } else if (uri.startsWith("file:///")) {
    path = uri.slice("file://".length); // keep the leading slash
  } else if (uri.startsWith("/")) {
    path = uri;
  }
  if (!path) return null;
  // Only rewrite when the path lands in an allowed root; otherwise
  // return null so the caller falls through to raw-URI handling
  // (which will open a new tab for HTTP-ish schemes).
  for (const root of ALLOWED_CONTAINER_ROOTS) {
    if (path === root || path.startsWith(root + "/")) return path;
  }
  return null;
}

/** Trigger a browser download for an attachment. Uses fetch+blob
 *  rather than an anchor navigation because the download endpoint
 *  requires workspace auth — and the browser won't attach
 *  `Authorization: Bearer` or `X-Molecule-Org-Slug` to a bare anchor
 *  click. A 25MB per-file cap server-side keeps the blob buffer
 *  bounded. HTTP(S) URIs skip the fetch path and open directly
 *  since they're off-platform artefacts that we don't own auth for. */
export async function downloadChatFile(
  workspaceId: string,
  attachment: ChatAttachment,
): Promise<void> {
  const href = resolveAttachmentHref(workspaceId, attachment.uri);
  const isContainerPath = normalizeWorkspaceUri(attachment.uri) !== null;
  if (!isContainerPath) {
    // External URL — let the browser navigate. Opens in new tab so
    // the canvas context survives a navigation. `href` here is the
    // raw URI (http(s), or anything else the agent sent back).
    window.open(href, "_blank", "noopener,noreferrer");
    return;
  }

  const headers: Record<string, string> = {};
  const slug = getTenantSlug();
  if (slug) headers["X-Molecule-Org-Slug"] = slug;
  const adminToken = process.env.NEXT_PUBLIC_ADMIN_TOKEN;
  if (adminToken) headers["Authorization"] = `Bearer ${adminToken}`;

  const res = await fetch(href, {
    headers,
    credentials: "include",
    signal: AbortSignal.timeout(60_000),
  });
  if (!res.ok) {
    throw new Error(`download failed: ${res.status}`);
  }
  const blob = await res.blob();
  // Revoke the object URL after the click — browsers hold the blob
  // until the URL is either revoked or the document unloads. 30s is
  // plenty of headroom for the click → save dialog round-trip.
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = attachment.name;
  a.rel = "noopener";
  document.body.appendChild(a);
  a.click();
  a.remove();
  setTimeout(() => URL.revokeObjectURL(url), 30_000);
}
