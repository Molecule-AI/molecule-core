import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { appendMessageDeduped, createMessage, type ChatMessage } from "../types";

// Unit tests for appendMessageDeduped — the helper that collapses the
// race between the HTTP /a2a .then() handler, the A2A_RESPONSE WS event,
// and the send_message_to_user push. All three paths can deliver the
// same agent reply; without dedupe the user sees 2-3 identical bubbles
// with identical timestamps.

describe("appendMessageDeduped", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    // Pin Date.now so "recently added" windows are deterministic across
    // the dedupe + Date.parse calls inside the helper.
    vi.setSystemTime(new Date("2026-04-23T12:00:00.000Z"));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("appends a new message when the history is empty", () => {
    const msg = createMessage("agent", "hello");
    const next = appendMessageDeduped([], msg);
    expect(next).toHaveLength(1);
    expect(next[0]).toBe(msg);
  });

  it("appends when content differs from the recent tail", () => {
    const first = createMessage("agent", "hello");
    vi.advanceTimersByTime(100);
    const second = createMessage("agent", "world");
    const next = appendMessageDeduped([first], second);
    expect(next).toHaveLength(2);
  });

  it("skips a duplicate (same role+content) within the window", () => {
    const first = createMessage("agent", "Hey! How can I help you today?");
    vi.advanceTimersByTime(500); // well inside the 3s window
    const dup = createMessage("agent", "Hey! How can I help you today?");
    const next = appendMessageDeduped([first], dup);
    expect(next).toHaveLength(1);
    // The array is returned unchanged — not a new reference.
    expect(next[0]).toBe(first);
  });

  it("does NOT dedupe across different roles even if content matches", () => {
    // Agent echoing the user's "hi" is a legitimate two-bubble case.
    const user = createMessage("user", "hi");
    vi.advanceTimersByTime(100);
    const agent = createMessage("agent", "hi");
    const next = appendMessageDeduped([user], agent);
    expect(next).toHaveLength(2);
  });

  it("does NOT dedupe once the window has elapsed", () => {
    // A user legitimately sending "hi" a few seconds apart must render
    // both bubbles. Default window is 3000 ms.
    const first = createMessage("user", "hi");
    vi.advanceTimersByTime(4000);
    const repeat = createMessage("user", "hi");
    const next = appendMessageDeduped([first], repeat);
    expect(next).toHaveLength(2);
  });

  it("only checks the tail's content, not the entire history", () => {
    // Same (role, content) appearing earlier in the conversation but
    // outside the dedupe window is not a duplicate.
    const old = createMessage("agent", "hi");
    vi.advanceTimersByTime(10_000);
    const newer = createMessage("agent", "hi");
    const next = appendMessageDeduped([old], newer);
    expect(next).toHaveLength(2);
  });

  it("handles malformed timestamps without throwing", () => {
    // Defense: a history entry with a bogus timestamp shouldn't nuke
    // the append path. The helper should just treat that entry as
    // "too old to dedupe against" and append the new message.
    const garbled: ChatMessage = {
      id: "x",
      role: "agent",
      content: "hi",
      timestamp: "not-a-real-timestamp",
    };
    const fresh = createMessage("agent", "hi");
    expect(() => appendMessageDeduped([garbled], fresh)).not.toThrow();
    const next = appendMessageDeduped([garbled], fresh);
    expect(next).toHaveLength(2);
  });

  it("accepts a custom dedupe window", () => {
    const first = createMessage("agent", "hello");
    vi.advanceTimersByTime(500);
    // Tight 100 ms window — the 500 ms-old first message falls outside.
    const dup = createMessage("agent", "hello");
    const next = appendMessageDeduped([first], dup, 100);
    expect(next).toHaveLength(2);
  });
});
