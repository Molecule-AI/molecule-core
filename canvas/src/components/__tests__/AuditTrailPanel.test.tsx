// @vitest-environment jsdom
/**
 * AuditTrailPanel tests — issue #753
 *
 * Split into three suites:
 *  1. formatAuditRelativeTime — pure helper (no mocks needed)
 *  2. AuditEntryRow — entry renderer: badges, tamper flag, timestamp, summary
 *  3. AuditTrailPanel — component integration: loading, empty state, entries,
 *                        filter bar, pagination, error handling
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, cleanup, fireEvent, act } from "@testing-library/react";

// ── Mocks (hoisted before imports) ────────────────────────────────────────────

vi.mock("@/lib/api", () => ({
  api: { get: vi.fn() },
}));

// ── Imports (after mocks) ─────────────────────────────────────────────────────

import { api } from "@/lib/api";
import {
  formatAuditRelativeTime,
  AuditEntryRow,
  AuditTrailPanel,
} from "../AuditTrailPanel";
import type { AuditEntry } from "@/types/audit";

const mockGet = vi.mocked(api.get);

// ── Helpers ───────────────────────────────────────────────────────────────────

const NOW = 1_745_000_000_000; // fixed "now" for deterministic tests

function makeEntry(overrides: Partial<AuditEntry> = {}): AuditEntry {
  return {
    id: "entry-1",
    workspace_id: "ws-a",
    timestamp: new Date(NOW - 120_000).toISOString(), // 2 min ago
    agent_id: "research-agent",
    session_id: "sess-1",
    operation: "task_start",
    input_hash: null,
    output_hash: null,
    model_used: null,
    human_oversight_flag: false,
    risk_flag: false,
    prev_hmac: null,
    hmac: "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
    chain_valid: true,
    ...overrides,
  };
}

function makeResponse(
  events: AuditEntry[],
  total?: number
) {
  return { events, total: total ?? events.length, chain_valid: true };
}

// ── Suite 1: formatAuditRelativeTime ─────────────────────────────────────────

describe("formatAuditRelativeTime", () => {
  it("returns 'just now' when diff < 60 s", () => {
    expect(formatAuditRelativeTime(new Date(NOW - 30_000).toISOString(), NOW)).toBe("just now");
  });

  it("returns 'Xm ago' for minute-scale diffs", () => {
    expect(formatAuditRelativeTime(new Date(NOW - 3 * 60_000).toISOString(), NOW)).toBe("3m ago");
  });

  it("returns 'Xh ago' for hour-scale diffs", () => {
    expect(formatAuditRelativeTime(new Date(NOW - 2 * 3_600_000).toISOString(), NOW)).toBe("2h ago");
  });

  it("returns a locale date string for diffs >= 24 h", () => {
    const ts = new Date(NOW - 25 * 3_600_000).toISOString();
    const result = formatAuditRelativeTime(ts, NOW);
    // Should be a locale-formatted date, not "Xh ago"
    expect(result).not.toMatch(/ago/);
    expect(result.length).toBeGreaterThan(0);
  });
});

// ── Suite 2: AuditEntryRow ────────────────────────────────────────────────────

describe("AuditEntryRow — badge colors", () => {
  afterEach(() => cleanup());

  it("renders the task_start badge", () => {
    render(<AuditEntryRow entry={makeEntry({ operation: "task_start" })} now={NOW} />);
    expect(screen.getByText("task_start")).toBeTruthy();
  });

  it("renders the task_end badge", () => {
    render(<AuditEntryRow entry={makeEntry({ operation: "task_end" })} now={NOW} />);
    expect(screen.getByText("task_end")).toBeTruthy();
  });

  it("renders the gate badge", () => {
    render(<AuditEntryRow entry={makeEntry({ operation: "gate" })} now={NOW} />);
    expect(screen.getByLabelText("Operation: gate")).toBeTruthy();
  });

  it("renders the hitl badge", () => {
    render(<AuditEntryRow entry={makeEntry({ operation: "hitl" })} now={NOW} />);
    expect(screen.getByLabelText("Operation: hitl")).toBeTruthy();
  });
});

describe("AuditEntryRow — content", () => {
  afterEach(() => cleanup());

  it("displays agent_id", () => {
    render(<AuditEntryRow entry={makeEntry({ agent_id: "my-research-agent" })} now={NOW} />);
    expect(screen.getByText("my-research-agent")).toBeTruthy();
  });

  it("displays summary derived from operation", () => {
    render(<AuditEntryRow entry={makeEntry({ operation: "task_start" })} now={NOW} />);
    expect(screen.getByText("task start")).toBeTruthy();
  });

  it("displays summary with model_used when present", () => {
    render(<AuditEntryRow entry={makeEntry({ operation: "task_start", model_used: "claude-3" })} now={NOW} />);
    expect(screen.getByText("task start (claude-3)")).toBeTruthy();
  });

  it("shows relative timestamp", () => {
    render(<AuditEntryRow entry={makeEntry({ timestamp: new Date(NOW - 2 * 60_000).toISOString() })} now={NOW} />);
    expect(screen.getByText("2m ago")).toBeTruthy();
  });

  it("does NOT render tamper warning when chain_valid is true", () => {
    render(<AuditEntryRow entry={makeEntry({ chain_valid: true })} now={NOW} />);
    expect(screen.queryByRole("img", { name: /tamper/i })).toBeNull();
  });

  it("renders tamper warning when chain_valid is false", () => {
    render(<AuditEntryRow entry={makeEntry({ chain_valid: false })} now={NOW} />);
    const warning = screen.getByRole("img", { name: /tamper/i });
    expect(warning).toBeTruthy();
    expect(warning.textContent).toContain("\u26A0");
  });
});

// ── Suite 3: AuditTrailPanel component ───────────────────────────────────────

describe("AuditTrailPanel — loading and empty state", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
    cleanup();
  });

  it("shows loading state while fetch is in-flight", async () => {
    // Never resolve to keep loading state
    mockGet.mockReturnValue(new Promise(() => {}));
    render(<AuditTrailPanel workspaceId="ws-a" />);
    expect(screen.getByText("Loading audit trail\u2026")).toBeTruthy();
  });

  it("shows empty state when events array is empty", async () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockGet.mockResolvedValue(makeResponse([]) as any);
    render(<AuditTrailPanel workspaceId="ws-a" />);
    await act(async () => { await Promise.resolve(); });
    expect(screen.getByText("No audit events yet")).toBeTruthy();
  });

  it("shows descriptive empty state copy", async () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockGet.mockResolvedValue(makeResponse([]) as any);
    render(<AuditTrailPanel workspaceId="ws-a" />);
    await act(async () => { await Promise.resolve(); });
    expect(screen.getByText(/Delegation, decision, gate/i)).toBeTruthy();
  });
});

describe("AuditTrailPanel — entries", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
    cleanup();
  });

  it("renders all returned entries", async () => {
    const entries = [
      makeEntry({ id: "e1", agent_id: "agent-alpha" }),
      makeEntry({ id: "e2", agent_id: "agent-beta" }),
    ];
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockGet.mockResolvedValue(makeResponse(entries) as any);
    render(<AuditTrailPanel workspaceId="ws-a" />);
    await act(async () => { await Promise.resolve(); });
    expect(screen.getByText("agent-alpha")).toBeTruthy();
    expect(screen.getByText("agent-beta")).toBeTruthy();
  });

  it("renders tamper warning for chain_valid=false entry", async () => {
    const entries = [makeEntry({ id: "e1", chain_valid: false })];
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockGet.mockResolvedValue(makeResponse(entries) as any);
    render(<AuditTrailPanel workspaceId="ws-a" />);
    await act(async () => { await Promise.resolve(); });
    expect(screen.getByRole("img", { name: /tamper/i })).toBeTruthy();
  });

  it("shows entry count footer", async () => {
    const entries = [makeEntry({ id: "e1" }), makeEntry({ id: "e2" })];
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockGet.mockResolvedValue(makeResponse(entries) as any);
    render(<AuditTrailPanel workspaceId="ws-a" />);
    await act(async () => { await Promise.resolve(); });
    expect(screen.getByText(/2 events loaded/)).toBeTruthy();
  });

  it("shows 'all loaded' when all entries are loaded", async () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockGet.mockResolvedValue(makeResponse([makeEntry()], 1) as any);
    render(<AuditTrailPanel workspaceId="ws-a" />);
    await act(async () => { await Promise.resolve(); });
    expect(screen.getByText(/all loaded/)).toBeTruthy();
  });
});

describe("AuditTrailPanel — pagination", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
    cleanup();
  });

  it("shows 'Load more' button when total > loaded count", async () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockGet.mockResolvedValue(makeResponse([makeEntry()], 100) as any);
    render(<AuditTrailPanel workspaceId="ws-a" />);
    await act(async () => { await Promise.resolve(); });
    expect(screen.getByRole("button", { name: /load more/i })).toBeTruthy();
  });

  it("does NOT show 'Load more' when all entries are loaded", async () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockGet.mockResolvedValue(makeResponse([makeEntry()], 1) as any);
    render(<AuditTrailPanel workspaceId="ws-a" />);
    await act(async () => { await Promise.resolve(); });
    expect(screen.queryByRole("button", { name: /load more/i })).toBeNull();
  });

  it("appends entries when 'Load more' is clicked", async () => {
    const page1 = [makeEntry({ id: "e1", agent_id: "alpha" })];
    const page2 = [makeEntry({ id: "e2", agent_id: "beta" })];
    mockGet
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      .mockResolvedValueOnce(makeResponse(page1, 2) as any)
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      .mockResolvedValueOnce(makeResponse(page2, 2) as any);

    render(<AuditTrailPanel workspaceId="ws-a" />);
    await act(async () => { await Promise.resolve(); });

    expect(screen.getByText("alpha")).toBeTruthy();
    expect(screen.queryByText("beta")).toBeNull();

    const loadMoreBtn = screen.getByRole("button", { name: /load more/i });
    fireEvent.click(loadMoreBtn);
    await act(async () => { await Promise.resolve(); });

    expect(screen.getByText("alpha")).toBeTruthy();
    expect(screen.getByText("beta")).toBeTruthy();
  });

  it("second page request includes offset param", async () => {
    const page1 = [makeEntry({ id: "e1" })];
    mockGet
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      .mockResolvedValueOnce(makeResponse(page1, 5) as any)
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      .mockResolvedValueOnce(makeResponse([], 5) as any);

    render(<AuditTrailPanel workspaceId="ws-a" />);
    await act(async () => { await Promise.resolve(); });

    fireEvent.click(screen.getByRole("button", { name: /load more/i }));
    await act(async () => { await Promise.resolve(); });

    // Second call should include offset=1 (one entry loaded from page 1)
    const secondCallPath = mockGet.mock.calls[1][0] as string;
    expect(secondCallPath).toContain("offset=1");
  });
});

describe("AuditTrailPanel — filter bar", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
    cleanup();
  });

  it("renders all five filter buttons", async () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockGet.mockResolvedValue(makeResponse([]) as any);
    render(<AuditTrailPanel workspaceId="ws-a" />);
    await act(async () => { await Promise.resolve(); });
    expect(screen.getByRole("button", { name: /^All$/i })).toBeTruthy();
    expect(screen.getByRole("button", { name: /^Task Start$/i })).toBeTruthy();
    expect(screen.getByRole("button", { name: /^Task End$/i })).toBeTruthy();
    expect(screen.getByRole("button", { name: /^Gate$/i })).toBeTruthy();
    expect(screen.getByRole("button", { name: /^HITL$/i })).toBeTruthy();
  });

  it("includes operation param when a type filter is active", async () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockGet.mockResolvedValue(makeResponse([]) as any);
    render(<AuditTrailPanel workspaceId="ws-a" />);
    await act(async () => { await Promise.resolve(); });

    const taskStartBtn = screen.getByRole("button", { name: /^Task Start$/i });
    fireEvent.click(taskStartBtn);
    await act(async () => { await Promise.resolve(); });

    // Second API call should include operation=task_start
    const lastCallPath = mockGet.mock.calls[mockGet.mock.calls.length - 1][0] as string;
    expect(lastCallPath).toContain("operation=task_start");
  });

  it("omits operation param when 'All' filter is active", async () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockGet.mockResolvedValue(makeResponse([]) as any);
    render(<AuditTrailPanel workspaceId="ws-a" />);
    await act(async () => { await Promise.resolve(); });

    const firstCallPath = mockGet.mock.calls[0][0] as string;
    expect(firstCallPath).not.toContain("operation=");
  });
});

describe("AuditTrailPanel — error handling", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
    cleanup();
  });

  it("shows error banner when fetch fails", async () => {
    mockGet.mockRejectedValue(new Error("Network timeout"));
    render(<AuditTrailPanel workspaceId="ws-a" />);
    await act(async () => { await Promise.resolve(); });
    expect(screen.getByText("Network timeout")).toBeTruthy();
  });

  it("still renders empty state (not error) on successful empty response", async () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockGet.mockResolvedValue(makeResponse([]) as any);
    render(<AuditTrailPanel workspaceId="ws-a" />);
    await act(async () => { await Promise.resolve(); });
    expect(screen.queryByText(/Network/)).toBeNull();
    expect(screen.getByText("No audit events yet")).toBeTruthy();
  });
});
