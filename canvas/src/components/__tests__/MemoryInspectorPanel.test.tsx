// @vitest-environment jsdom
/**
 * MemoryInspectorPanel tests — issue #909
 *
 * Covers: loading, empty state, scope tabs, namespace filter,
 * entry list, expand, delete flow, optimistic updates, Refresh, semantic search.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor, cleanup, act } from "@testing-library/react";

// ── Mocks ─────────────────────────────────────────────────────────────────────

vi.mock("@/lib/api", () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
    del: vi.fn(),
  },
}));

vi.mock("@/components/ConfirmDialog", () => ({
  ConfirmDialog: ({
    open,
    title,
    message,
    onConfirm,
    onCancel,
  }: {
    open: boolean;
    title: string;
    message: string;
    confirmLabel?: string;
    confirmVariant?: string;
    onConfirm: () => void;
    onCancel: () => void;
    singleButton?: boolean;
  }) =>
    open ? (
      <div data-testid="confirm-dialog">
        <p data-testid="dialog-title">{title}</p>
        <p data-testid="dialog-message">{message}</p>
        <button onClick={onConfirm}>Confirm Delete</button>
        <button onClick={onCancel}>Cancel Delete</button>
      </div>
    ) : null,
}));

import { api } from "@/lib/api";
import { MemoryInspectorPanel } from "../MemoryInspectorPanel";

// ── Typed mock helpers ────────────────────────────────────────────────────────

const mockGet = vi.mocked(api.get);
const mockDel = vi.mocked(api.del);

// ── Sample fixtures ───────────────────────────────────────────────────────────

const NOW = "2026-04-17T12:00:00.000Z";

const MEMORY_A: import("../MemoryInspectorPanel").MemoryEntry = {
  id: "mem-a",
  workspace_id: "ws-1",
  content: "Remember to review PRs before merging",
  scope: "LOCAL",
  namespace: "general",
  created_at: NOW,
};

const MEMORY_B: import("../MemoryInspectorPanel").MemoryEntry = {
  id: "mem-b",
  workspace_id: "ws-1",
  content: "Team knowledge: deploy happens on Fridays",
  scope: "TEAM",
  namespace: "procedures",
  created_at: NOW,
};

const TWO_MEMORIES = [MEMORY_A, MEMORY_B];

// ── Setup / teardown ──────────────────────────────────────────────────────────

beforeEach(() => {
  vi.clearAllMocks();
});

afterEach(() => {
  cleanup();
});

// ── Helper: flush microtasks + React state updates ─────────────────────────────
async function flushUpdates(): Promise<void> {
  await act(async () => {});
}

// ── Loading & empty state ─────────────────────────────────────────────────────

describe("MemoryInspectorPanel — loading and empty state", () => {
  it("shows loading indicator before data arrives", () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockGet.mockReturnValue(new Promise(() => {}) as any);
    render(<MemoryInspectorPanel workspaceId="ws-1" />);
    expect(screen.getByText(/loading memories/i)).toBeTruthy();
  });

  it("renders empty state when API returns []", async () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockGet.mockResolvedValue([] as any);
    render(<MemoryInspectorPanel workspaceId="ws-1" />);
    await flushUpdates();
    expect(screen.getByText("No LOCAL memories")).toBeTruthy();
  });

  it("fetches from the correct workspace memories endpoint with scope=LOCAL", async () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockGet.mockResolvedValue([] as any);
    render(<MemoryInspectorPanel workspaceId="ws-abc-123" />);
    await flushUpdates();
    expect(mockGet).toHaveBeenCalledWith(
      "/workspaces/ws-abc-123/memories?scope=LOCAL"
    );
  });

  it("shows error banner when fetch throws", async () => {
    mockGet.mockRejectedValue(new Error("Network error"));
    render(<MemoryInspectorPanel workspaceId="ws-1" />);
    await flushUpdates();
    expect(screen.getByText("Network error")).toBeTruthy();
  });
});

// ── Scope tabs ────────────────────────────────────────────────────────────────

describe("MemoryInspectorPanel — scope tabs", () => {
  it("renders LOCAL, TEAM, GLOBAL tabs", async () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockGet.mockResolvedValue([] as any);
    render(<MemoryInspectorPanel workspaceId="ws-1" />);
    await flushUpdates();
    expect(screen.getByRole("button", { name: "LOCAL" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "TEAM" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "GLOBAL" })).toBeTruthy();
  });

  it("LOCAL is active by default", async () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockGet.mockResolvedValue([] as any);
    render(<MemoryInspectorPanel workspaceId="ws-1" />);
    await flushUpdates();
    expect(screen.getByRole("button", { name: "LOCAL" }).getAttribute("aria-pressed")).toBe("true");
  });

  it("clicking TEAM tab re-fetches with scope=TEAM", async () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockGet.mockResolvedValue([] as any);
    render(<MemoryInspectorPanel workspaceId="ws-1" />);
    await flushUpdates();

    mockGet.mockClear();
    fireEvent.click(screen.getByRole("button", { name: "TEAM" }));
    await flushUpdates();
    expect(mockGet).toHaveBeenCalledWith(
      "/workspaces/ws-1/memories?scope=TEAM"
    );
  });

  it("clicking GLOBAL tab re-fetches with scope=GLOBAL", async () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockGet.mockResolvedValue([] as any);
    render(<MemoryInspectorPanel workspaceId="ws-1" />);
    await flushUpdates();

    mockGet.mockClear();
    fireEvent.click(screen.getByRole("button", { name: "GLOBAL" }));
    await flushUpdates();
    expect(mockGet).toHaveBeenCalledWith(
      "/workspaces/ws-1/memories?scope=GLOBAL"
    );
  });

  it("shows scope-specific empty state when switching tabs", async () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockGet.mockResolvedValue([] as any);
    render(<MemoryInspectorPanel workspaceId="ws-1" />);
    await flushUpdates();

    fireEvent.click(screen.getByRole("button", { name: "TEAM" }));
    await flushUpdates();
    expect(screen.getByText("No TEAM memories")).toBeTruthy();
  });
});

// ── Namespace filter ──────────────────────────────────────────────────────────

describe("MemoryInspectorPanel — namespace filter", () => {
  it("renders namespace filter input", async () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockGet.mockResolvedValue([] as any);
    render(<MemoryInspectorPanel workspaceId="ws-1" />);
    await flushUpdates();
    expect(screen.getByLabelText("Filter by namespace")).toBeTruthy();
  });

  it("includes namespace param in API call when set", async () => {
    vi.useFakeTimers();
    try {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      mockGet.mockResolvedValue([] as any);
      render(<MemoryInspectorPanel workspaceId="ws-1" />);
      await flushUpdates();

      mockGet.mockClear();
      fireEvent.change(screen.getByLabelText("Filter by namespace"), {
        target: { value: "facts" },
      });
      // Advance past the 300ms debounce
      act(() => { vi.advanceTimersByTime(350); });
      await flushUpdates();

      expect(mockGet).toHaveBeenCalledWith(
        "/workspaces/ws-1/memories?scope=LOCAL&namespace=facts"
      );
    } finally {
      vi.useRealTimers();
    }
  });
});

// ── Entry list ───────────────────────────────────────────────────────────────

describe("MemoryInspectorPanel — entry list", () => {
  beforeEach(() => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockGet.mockResolvedValue(TWO_MEMORIES as any);
  });

  it("renders a row for every memory", async () => {
    render(<MemoryInspectorPanel workspaceId="ws-1" />);
    await flushUpdates();
    expect(screen.getByText(/Remember to review PRs before merging/)).toBeTruthy();
    expect(screen.getByText(/Team knowledge: deploy happens on Fridays/)).toBeTruthy();
  });

  it("displays memory count in toolbar", async () => {
    render(<MemoryInspectorPanel workspaceId="ws-1" />);
    await flushUpdates();
    expect(screen.getByText("2 memories")).toBeTruthy();
  });

  it("displays scope badge for each entry", async () => {
    render(<MemoryInspectorPanel workspaceId="ws-1" />);
    await flushUpdates();
    expect(screen.getByTitle("Scope: LOCAL")).toBeTruthy();
    expect(screen.getByTitle("Scope: TEAM")).toBeTruthy();
  });

  it("entries are collapsed by default (pre region not visible)", async () => {
    render(<MemoryInspectorPanel workspaceId="ws-1" />);
    await flushUpdates();
    // Expanded region (pre tag) should not exist in DOM yet
    expect(screen.queryByRole("region")).toBeNull();
  });
});

// ── Expand / collapse ─────────────────────────────────────────────────────────

describe("MemoryInspectorPanel — expand/collapse", () => {
  beforeEach(() => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockGet.mockResolvedValue(TWO_MEMORIES as any);
  });

  it("clicking a row header expands it and shows the full content in a pre tag", async () => {
    render(<MemoryInspectorPanel workspaceId="ws-1" />);
    await flushUpdates();

    fireEvent.click(
      screen.getByText(/Remember to review PRs before merging/).closest("button")!
    );
    await flushUpdates();
    // After expand, a region with the full content <pre> should appear
    expect(screen.getByRole("region")).toBeTruthy();
  });

  it("clicking the header again collapses the row (pre region removed)", async () => {
    render(<MemoryInspectorPanel workspaceId="ws-1" />);
    await flushUpdates();

    const headerBtn = screen
      .getByText(/Remember to review PRs before merging/)
      .closest("button")!;
    fireEvent.click(headerBtn); // expand
    await flushUpdates();
    expect(screen.getByRole("region")).toBeTruthy();

    fireEvent.click(headerBtn); // collapse
    await flushUpdates();
    // After collapse, the region (pre) is removed from the DOM
    expect(screen.queryByRole("region")).toBeNull();
  });
});

// ── Delete flow ───────────────────────────────────────────────────────────────

describe("MemoryInspectorPanel — delete flow", () => {
  beforeEach(() => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockGet.mockResolvedValue(TWO_MEMORIES as any);
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockDel.mockResolvedValue({ status: "deleted" } as any);
  });

  /** Helper: expand memory-A and click its Delete button */
  async function openDeleteForMemoryA() {
    render(<MemoryInspectorPanel workspaceId="ws-1" />);
    await flushUpdates();
    fireEvent.click(
      screen.getByText(/Remember to review PRs before merging/).closest("button")!
    );
    await flushUpdates();
    fireEvent.click(screen.getByRole("button", { name: "Delete memory" }));
    await flushUpdates();
  }

  it("opens ConfirmDialog when Delete is clicked", async () => {
    await openDeleteForMemoryA();
    expect(screen.getByTestId("confirm-dialog")).toBeTruthy();
    expect(screen.getByTestId("dialog-title").textContent).toBe("Delete memory");
  });

  it("calls api.del with the correct URL-encoded path on confirm", async () => {
    await openDeleteForMemoryA();
    fireEvent.click(screen.getByText("Confirm Delete"));
    await flushUpdates();
    expect(mockDel).toHaveBeenCalledWith("/workspaces/ws-1/memories/mem-a");
  });

  it("removes the entry optimistically after confirm", async () => {
    await openDeleteForMemoryA();
    fireEvent.click(screen.getByText("Confirm Delete"));
    await flushUpdates();
    expect(screen.queryByText(/Remember to review PRs before merging/)).toBeNull();
    // Sibling entry unaffected
    expect(screen.getByText(/Team knowledge: deploy happens on Fridays/)).toBeTruthy();
  });

  it("closes ConfirmDialog without deleting when Cancel is clicked", async () => {
    await openDeleteForMemoryA();
    fireEvent.click(screen.getByText("Cancel Delete"));
    await flushUpdates();
    expect(screen.queryByTestId("confirm-dialog")).toBeNull();
    expect(mockDel).not.toHaveBeenCalled();
    // Sibling memory entry (MEMORY_B) is still in the list
    expect(screen.getByText(/Team knowledge: deploy happens on Fridays/)).toBeTruthy();
  });
});

// ── Refresh ───────────────────────────────────────────────────────────────────

describe("MemoryInspectorPanel — Refresh button", () => {
  it("re-fetches entries when Refresh is clicked", async () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockGet.mockResolvedValue([] as any);
    render(<MemoryInspectorPanel workspaceId="ws-1" />);
    await flushUpdates();
    expect(screen.getByText("No LOCAL memories")).toBeTruthy();

    expect(mockGet).toHaveBeenCalledTimes(1);
    fireEvent.click(screen.getByRole("button", { name: "Refresh memories" }));
    await flushUpdates();
    expect(mockGet).toHaveBeenCalledTimes(2);
  });
});

// ── role=alert a11y ──────────────────────────────────────────────────────────

describe("MemoryInspectorPanel — error elements have role=alert", () => {
  it("fetch error banner has role='alert'", async () => {
    mockGet.mockRejectedValue(new Error("Network error"));
    render(<MemoryInspectorPanel workspaceId="ws-1" />);
    await flushUpdates();
    const alert = screen.getByRole("alert");
    expect(alert).toBeTruthy();
    expect(alert.textContent).toContain("Network error");
  });
});

// ── Semantic search ──────────────────────────────────────────────────────────

describe("MemoryInspectorPanel — semantic search", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("debounces search input by 300ms before calling API", async () => {
    vi.useFakeTimers();
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockGet.mockResolvedValue([] as any);
    render(<MemoryInspectorPanel workspaceId="ws-1" />);
    await flushUpdates();

    mockGet.mockClear();

    fireEvent.change(screen.getByLabelText("Search memories"), {
      target: { value: "deploy" },
    });

    // 200ms — debounce has NOT fired yet
    act(() => { vi.advanceTimersByTime(200); });
    await flushUpdates();
    expect(mockGet).not.toHaveBeenCalled();

    // 350ms total — debounce fires
    act(() => { vi.advanceTimersByTime(150); });
    await flushUpdates();

    expect(mockGet).toHaveBeenCalledWith(
      "/workspaces/ws-1/memories?scope=LOCAL&q=deploy"
    );
  });

  it("renders similarity-badge when entry has similarity_score", async () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockGet.mockResolvedValue([{ ...MEMORY_A, similarity_score: 0.87 }] as any);
    render(<MemoryInspectorPanel workspaceId="ws-1" />);
    await flushUpdates();
    const badge = document.querySelector('[data-testid="similarity-badge"]');
    expect(badge).toBeTruthy();
    expect(badge?.textContent).toBe("87%");
  });

  it("does not render similarity-badge when entry has no similarity_score", async () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockGet.mockResolvedValue([MEMORY_A] as any);
    render(<MemoryInspectorPanel workspaceId="ws-1" />);
    await flushUpdates();
    expect(
      document.querySelector('[data-testid="similarity-badge"]')
    ).toBeNull();
  });

  it("clear button resets query immediately and re-fetches without ?q=", async () => {
    vi.useFakeTimers();
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockGet.mockResolvedValue([] as any);
    render(<MemoryInspectorPanel workspaceId="ws-1" />);
    await flushUpdates();

    fireEvent.change(screen.getByLabelText("Search memories"), {
      target: { value: "deploy" },
    });

    act(() => { vi.advanceTimersByTime(350); });
    await flushUpdates();

    expect(mockGet).toHaveBeenCalledWith(
      "/workspaces/ws-1/memories?scope=LOCAL&q=deploy"
    );
    mockGet.mockClear();

    fireEvent.click(screen.getByRole("button", { name: "Clear search" }));
    await flushUpdates();

    expect(mockGet).toHaveBeenCalledWith(
      "/workspaces/ws-1/memories?scope=LOCAL"
    );
  });
});
