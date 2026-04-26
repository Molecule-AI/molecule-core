import { describe, it, expect } from "vitest";
import { resolveAttachmentHref } from "../uploads";

describe("resolveAttachmentHref — URI scheme normalisation", () => {
  const wsId = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee";

  it("rewrites the canonical workspace:<path> scheme to /chat/download", () => {
    const url = resolveAttachmentHref(wsId, "workspace:/workspace/report.pdf");
    expect(url).toContain(`/workspaces/${wsId}/chat/download`);
    expect(url).toContain(encodeURIComponent("/workspace/report.pdf"));
  });

  it("accepts bare absolute container paths (some agents omit the scheme)", () => {
    const url = resolveAttachmentHref(wsId, "/workspace/report.pdf");
    expect(url).toContain(`/workspaces/${wsId}/chat/download`);
    expect(url).toContain(encodeURIComponent("/workspace/report.pdf"));
  });

  it("accepts file:/// URIs pointing into an allowed root", () => {
    const url = resolveAttachmentHref(wsId, "file:///workspace/report.pdf");
    expect(url).toContain(`/workspaces/${wsId}/chat/download`);
    expect(url).toContain(encodeURIComponent("/workspace/report.pdf"));
  });

  it("passes through HTTP(S) URIs unchanged so off-platform artefacts still render", () => {
    const external = "https://example.com/static/report.pdf";
    expect(resolveAttachmentHref(wsId, external)).toBe(external);
  });

  it("passes through container paths that are not under any allowed root", () => {
    // /etc/passwd looks like a path but isn't one of the allowed
    // roots — falling back to raw passthrough forces the caller into
    // the external-URL branch, which opens a new tab and lets the
    // browser refuse. Rewriting would 400 anyway server-side.
    expect(resolveAttachmentHref(wsId, "/etc/passwd")).toBe("/etc/passwd");
  });

  it("passes through unknown schemes unchanged", () => {
    expect(resolveAttachmentHref(wsId, "s3://bucket/key")).toBe("s3://bucket/key");
  });
});
