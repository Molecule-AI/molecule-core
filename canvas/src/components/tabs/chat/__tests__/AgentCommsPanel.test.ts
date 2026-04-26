// @vitest-environment jsdom
import { describe, it, expect, vi } from "vitest";

// Stub the canvas store before importing the SUT — toCommMessage calls
// useCanvasStore.getState() inside resolveName to look up peer names,
// which would otherwise hit the real Zustand store.
vi.mock("@/store/canvas", () => ({
  useCanvasStore: {
    getState: () => ({
      nodes: [
        { id: "ws-self", data: { name: "Self" } },
        { id: "ws-peer", data: { name: "Peer Agent" } },
      ],
    }),
  },
}));

import { toCommMessage, type ActivityEntry } from "../AgentCommsPanel";

const SELF = "ws-self";
const PEER = "ws-peer";

function makeEntry(overrides: Partial<ActivityEntry> = {}): ActivityEntry {
  return {
    id: "act-1",
    activity_type: "a2a_send",
    source_id: SELF,
    target_id: PEER,
    method: "message/send",
    summary: "Delegating to Peer Agent",
    request_body: null,
    response_body: null,
    status: "ok",
    created_at: "2026-04-25T18:00:00Z",
    ...overrides,
  };
}

describe("toCommMessage — flow derivation", () => {
  it("a2a_send is always outbound (flow=out, peer=target)", () => {
    const m = toCommMessage(
      makeEntry({ activity_type: "a2a_send", source_id: SELF, target_id: PEER }),
      SELF,
    );
    expect(m).toBeTruthy();
    expect(m!.flow).toBe("out");
    expect(m!.peerId).toBe(PEER);
    expect(m!.peerName).toBe("Peer Agent");
  });

  it("a2a_receive from a peer (peer-initiated call) is inbound", () => {
    // Real incoming call: source = peer, target = us.
    const m = toCommMessage(
      makeEntry({
        activity_type: "a2a_receive",
        source_id: PEER,
        target_id: SELF,
      }),
      SELF,
    );
    expect(m!.flow).toBe("in");
    expect(m!.peerId).toBe(PEER);
    expect(m!.peerName).toBe("Peer Agent");
  });

  it("a2a_receive self-logged by our runtime AFTER an outbound call is OUTBOUND from the user's POV", () => {
    // workspace/a2a_tools.py:181 self-logs an a2a_receive on the
    // CALLER's workspace_id with source_id=us, target_id=peer.
    // From the user's perspective this row belongs to the outbound
    // delegation thread — render flow=out + peer=target so the
    // bubble right-justifies under "Delegating to peer" and the
    // Restart button targets the actual peer (NOT us). Regression
    // for the bug where these rows rendered as "← From Self" with
    // a Restart button that would have restarted the user's own
    // workspace.
    const m = toCommMessage(
      makeEntry({
        activity_type: "a2a_receive",
        source_id: SELF,
        target_id: PEER,
        summary: "Peer Agent failed",
        status: "error",
      }),
      SELF,
    );
    expect(m!.flow).toBe("out");
    expect(m!.peerId).toBe(PEER);
    expect(m!.peerName).toBe("Peer Agent");
    expect(m!.status).toBe("error");
  });

  it("returns null when no peer can be resolved", () => {
    // a2a_receive with both ids null — discard rather than render a
    // ghost bubble pointing at "Unknown".
    const m = toCommMessage(
      makeEntry({
        activity_type: "a2a_receive",
        source_id: null,
        target_id: null,
      }),
      SELF,
    );
    expect(m).toBeNull();
  });

  it("propagates status through to the message (drives error rendering)", () => {
    const m = toCommMessage(
      makeEntry({ status: "error", activity_type: "a2a_send" }),
      SELF,
    );
    expect(m!.status).toBe("error");
  });

  // --- delegation rows ---
  // The platform's /delegate handler writes activity_type='delegation'
  // for both the initial outbound (method='delegate') and the eventual
  // reply (method='delegate_result', status=queued|completed|failed).
  // Pre-fix the panel filtered these out and showed "no agent comms"
  // even when 6+ delegations existed in the DB.

  it("delegation 'delegate' row maps as outbound to target", () => {
    const m = toCommMessage(
      makeEntry({
        activity_type: "delegation",
        method: "delegate",
        source_id: SELF,
        target_id: PEER,
        summary: "Delegating to ws-peer",
        status: "pending",
      }),
      SELF,
    );
    expect(m).toBeTruthy();
    expect(m!.flow).toBe("out");
    expect(m!.peerId).toBe(PEER);
    expect(m!.peerName).toBe("Peer Agent");
    expect(m!.text).toBe("Delegating to ws-peer");
    expect(m!.status).toBe("pending");
  });

  it("delegation 'delegate_result' queued row preserves status='queued'", () => {
    // The "queued" status is the load-bearing signal the LLM uses to
    // decide whether to wait or fall back. If toCommMessage drops or
    // rewrites it, the UI loses the ability to show the "peer busy,
    // will reply" affordance.
    const m = toCommMessage(
      makeEntry({
        activity_type: "delegation",
        method: "delegate_result",
        source_id: SELF,
        target_id: PEER,
        summary: "Delegation queued — target at capacity",
        status: "queued",
      }),
      SELF,
    );
    expect(m!.status).toBe("queued");
    expect(m!.text).toContain("queued");
  });

  it("delegation row with no target_id returns null", () => {
    // Defensive: a delegation row missing target_id can't be rendered
    // (we wouldn't know which peer to attribute it to). Drop instead
    // of rendering a ghost.
    const m = toCommMessage(
      makeEntry({
        activity_type: "delegation",
        target_id: null,
      }),
      SELF,
    );
    expect(m).toBeNull();
  });
});
