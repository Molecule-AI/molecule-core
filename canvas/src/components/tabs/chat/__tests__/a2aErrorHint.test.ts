import { describe, it, expect } from "vitest";
import { inferA2AErrorHint } from "../a2aErrorHint";

// Pure logic. Pin every named pattern so a future contributor adding a
// new symptom doesn't accidentally collapse the buckets — and so the
// "most specific first" ordering can't drift without a test failing.

describe("inferA2AErrorHint", () => {
  it("matches the Claude Code SDK init wedge specifically", () => {
    const hint = inferA2AErrorHint("Control request timeout: initialize");
    expect(hint).toMatch(/Claude Code SDK is wedged/);
  });

  it("does NOT misfire on user tasks containing 'initialize' generally", () => {
    // Regression: an earlier bare-`initialize` pattern would have
    // false-positived "failed to initialize database" into the SDK
    // wedge hint. Confirm the full-phrase guard holds.
    const hint = inferA2AErrorHint("failed to initialize database connection");
    expect(hint).not.toMatch(/Claude Code SDK/);
  });

  it("recognises httpx ReadTimeout / ConnectTimeout class names", () => {
    expect(inferA2AErrorHint("ReadTimeout: timeout")).toMatch(/proxy timeout/);
    expect(inferA2AErrorHint("ConnectTimeout: ...")).toMatch(/proxy timeout/);
  });

  it("recognises generic timeout / deadline-exceeded language", () => {
    expect(inferA2AErrorHint("deadline exceeded after 300s")).toMatch(/proxy timeout/);
    expect(inferA2AErrorHint("Operation timeout")).toMatch(/proxy timeout/);
  });

  it("handles connection-reset family (RemoteProtocolError, ConnectionReset, no-message)", () => {
    expect(inferA2AErrorHint("RemoteProtocolError: ...")).toMatch(/connection.*dropped/);
    expect(inferA2AErrorHint("ConnectionResetError")).toMatch(/connection.*dropped/);
    expect(inferA2AErrorHint("connection reset by peer")).toMatch(/connection.*dropped/);
    expect(inferA2AErrorHint("RemoteProtocolError (no message — likely connection reset)")).toMatch(/connection.*dropped/);
  });

  it("recognises agent-runtime exceptions", () => {
    expect(inferA2AErrorHint("Agent error: ValueError raised")).toMatch(/runtime threw an exception/);
    expect(inferA2AErrorHint("RuntimeException in tool call")).toMatch(/runtime threw an exception/);
  });

  it("recognises peer-unreachable cases (Activity-tab originals)", () => {
    expect(inferA2AErrorHint("workspace not found")).toMatch(/can't be reached/);
    expect(inferA2AErrorHint("not accessible")).toMatch(/can't be reached/);
    expect(inferA2AErrorHint("workspace is offline")).toMatch(/can't be reached/);
  });

  it("returns the empty-detail-specific hint when input is exactly empty", () => {
    expect(inferA2AErrorHint("")).toMatch(/no error detail/);
  });

  it("returns a generic fallback for unrecognised text", () => {
    const hint = inferA2AErrorHint("some completely novel error nobody has matched yet");
    expect(hint).toMatch(/Check the workspace logs|delivery failure/);
  });

  it("Claude SDK wedge wins over the more general timeout pattern", () => {
    // Both 'control request timeout' and 'timeout' match the same
    // input. The SDK wedge hint is more actionable; the ordering in
    // the function must keep it first. Lock that priority in.
    const hint = inferA2AErrorHint("Control request timeout: initialize");
    expect(hint).toMatch(/Claude Code SDK/);
    expect(hint).not.toMatch(/proxy timeout/);
  });
});
