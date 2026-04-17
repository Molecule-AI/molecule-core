// @vitest-environment jsdom
/**
 * MemoryInspectorPanel tests — issue #730
 *
 * Covers: loading, empty state, entry list, expand, edit flow (happy path,
 * invalid JSON, cancel), delete flow (confirm, cancel), optimistic updates,
 * and Refresh.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor, cleanup } from "@testing-library/react";

// ── Mocks (must be hoisted before any imports) ────────────────────────────────

vi.mock("@/lib/api", () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
    del: vi.fn(),
  },
}));

// ConfirmDialog uses createPortal + a `mounted` state guard that requires
// useEffect to fire. We mock it to a simple inline rendering so tests are
// synchronous and don't depend on document.body portal availability.
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

// ── Imports (after mocks) ─────────────────────────────────────────────────────

import { api } from "@/lib/api";
import { MemoryInspectorPanel } from "../MemoryInspectorPanel";

// ── Typed mock helpers ────────────────────────────────────────────────────────

const mockGet = vi.mocked(api.get);
const mockPost = vi.mocked(api.post);
const mockDel = vi.mocked(api.del);

// ── Sample fixtures ───────────────────────────────────────────────────────────

const NOW = new Date("2026-04-17T12:00:00.000Z").toISOString();
const LATER = new Date("2026-04-17T13:00:00.000Z").toISOString();

const ENTRY_A = {
  key: "task-queue",
  value: { pending: ["t-1", "t-2"], done: [] },
  version: 3,
  updated_at: NOW,
};

const ENTRY_B = {
  key: "session-token",
  value: "abc123",
  version: 1,
  expires_at: LATER,
  updated_at: NOW,
};

const TWO_ENTRIES = [ENTRY_A, ENTRY_B];

// ── Setup / teardown ──────────────────────────────────────────────────────────

beforeEach(() => {
  vi.clearAllMocks();
});

afterEach(() => {
  cleanup();
});

// ── Loading & empty state ─────────────────────────────────────────────────────

describe("MemoryInspectorPanel — loading and empty state", () => {
  it("shows loading indicator before data arrives", () => {
    // Never resolves within this test — just checks the loading UI
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockGet.mockReturnValue(new Promise(() => {}) as any);
    render(<MemoryInspectorPanel workspaceId="ws-1" />);
    expect(screen.getByText(/loading memory/i)).toBeTruthy();
  });

  it("renders empty state when API returns []", async () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockGet.mockResolvedValue([] as any);
    render(<MemoryInspectorPanel workspaceId="ws-1" />);
    await waitFor(() =>
      expect(screen.getByText("No memory entries yet")).toBeTruthy()
    );
  });

  it("fetches from the correct workspace memory endpoint", async () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockGet.mockResolvedValue([] as any);
    render(<MemoryInspectorPanel workspaceId="ws-abc-123" />);
    await waitFor(() =>
      expect(mockGet).toHaveBeenCalledWith("/workspaces/ws-abc-123/memory")
    );
  });

  it("shows error banner when fetch throws", async () => {
    mockGet.mockRejectedValue(new Error("Network error"));
    render(<MemoryInspectorPanel workspaceId="ws-1" />);
    await waitFor(() =>
      expect(screen.getByText("Network error")).toBeTruthy()
    );
  });
});

// ── Entry list ────────────────────────────────────────────────────────────────

describe("MemoryInspectorPanel — entry list", () => {
  beforeEach(() => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockGet.mockResolvedValue(TWO_ENTRIES as any);
  });

  it("renders a row for every entry key", async () => {
    render(<MemoryInspectorPanel workspaceId="ws-1" />);
    await waitFor(() => screen.getByText("task-queue"));
    expect(screen.getByText("session-token")).toBeTruthy();
  });

  it("displays '2 entries' count in the toolbar", async () => {
    render(<MemoryInspectorPanel workspaceId="ws-1" />);
    await waitFor(() => expect(screen.getByText("2 entries")).toBeTruthy());
  });

  it("displays '1 entry' (singular) when there is one entry", async () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockGet.mockResolvedValue([ENTRY_A] as any);
    render(<MemoryInspectorPanel workspaceId="ws-1" />);
    await waitFor(() => expect(screen.getByText("1 entry")).toBeTruthy());
  });

  it("shows version badge for each entry", async () => {
    render(<MemoryInspectorPanel workspaceId="ws-1" />);
    await waitFor(() => screen.getByText("task-queue"));
    expect(screen.getByText("v3")).toBeTruthy();
    expect(screen.getByText("v1")).toBeTruthy();
  });

  it("entries are collapsed by default (value not visible)", async () => {
    render(<MemoryInspectorPanel workspaceId="ws-1" />);
    await waitFor(() => screen.getByText("task-queue"));
    // The JSON value should NOT be rendered while collapsed
    expect(screen.queryByText(/"pending"/)).toBeNull();
  });
});

// ── Expand / collapse ─────────────────────────────────────────────────────────

describe("MemoryInspectorPanel — expand/collapse", () => {
  beforeEach(() => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockGet.mockResolvedValue(TWO_ENTRIES as any);
  });

  it("clicking a row header expands it and shows the JSON value", async () => {
    render(<MemoryInspectorPanel workspaceId="ws-1" />);
    await waitFor(() => screen.getByText("task-queue"));

    // Click to expand
    fireEvent.click(
      screen.getByText("task-queue").closest("button")!
    );

    await waitFor(() =>
      expect(screen.getByText(/"pending"/)).toBeTruthy()
    );
  });

  it("clicking the header again collapses the row", async () => {
    render(<MemoryInspectorPanel workspaceId="ws-1" />);
    await waitFor(() => screen.getByText("task-queue"));

    const headerBtn = screen.getByText("task-queue").closest("button")!;
    fireEvent.click(headerBtn); // expand
    await waitFor(() => screen.getByText(/"pending"/));

    fireEvent.click(headerBtn); // collapse
    await waitFor(() =>
      expect(screen.queryByText(/"pending"/)).toBeNull()
    );
  });

  it("shows expires_at when present", async () => {
    render(<MemoryInspectorPanel workspaceId="ws-1" />);
    await waitFor(() => screen.getByText("session-token"));
    fireEvent.click(
      screen.getByText("session-token").closest("button")!
    );
    await waitFor(() =>
      expect(screen.getByText(/expires/i)).toBeTruthy()
    );
  });
});

// ── Edit flow ─────────────────────────────────────────────────────────────────

describe("MemoryInspectorPanel — edit flow", () => {
  beforeEach(() => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockGet.mockResolvedValue(TWO_ENTRIES as any);
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockPost.mockResolvedValue({ status: "ok", key: "task-queue", version: 4 } as any);
  });

  /** Helper: expand entry-A and click its Edit button */
  async function openEditForEntryA() {
    render(<MemoryInspectorPanel workspaceId="ws-1" />);
    await waitFor(() => screen.getByText("task-queue"));
    fireEvent.click(screen.getByText("task-queue").closest("button")!);
    await waitFor(() =>
      screen.getByRole("button", { name: "Edit task-queue" })
    );
    fireEvent.click(screen.getByRole("button", { name: "Edit task-queue" }));
  }

  it("shows a textarea pre-filled with the entry value after clicking Edit", async () => {
    await openEditForEntryA();
    const ta = screen.getByRole("textbox", { name: "Edit memory value" });
    expect(ta).toBeTruthy();
    expect((ta as HTMLTextAreaElement).value).toContain("pending");
  });

  it("shows Save and Cancel buttons in edit mode", async () => {
    await openEditForEntryA();
    expect(screen.getByRole("button", { name: /^save$/i })).toBeTruthy();
    expect(screen.getByRole("button", { name: /^cancel$/i })).toBeTruthy();
  });

  it("POSTs to correct path with key, parsed value, and if_match_version", async () => {
    await openEditForEntryA();
    fireEvent.change(
      screen.getByRole("textbox", { name: "Edit memory value" }),
      { target: { value: '{"updated":true}' } }
    );
    fireEvent.click(screen.getByRole("button", { name: /^save$/i }));

    await waitFor(() => expect(mockPost).toHaveBeenCalled());

    const [path, body] = mockPost.mock.calls[0] as [
      string,
      { key: string; value: unknown; if_match_version: number }
    ];
    expect(path).toBe("/workspaces/ws-1/memory");
    expect(body.key).toBe("task-queue");
    expect(body.if_match_version).toBe(3); // ENTRY_A.version
    expect(body.value).toEqual({ updated: true });
  });

  it("closes the edit form on successful save", async () => {
    await openEditForEntryA();
    fireEvent.change(
      screen.getByRole("textbox", { name: "Edit memory value" }),
      { target: { value: '"new-value"' } }
    );
    fireEvent.click(screen.getByRole("button", { name: /^save$/i }));

    await waitFor(() =>
      expect(
        screen.queryByRole("textbox", { name: "Edit memory value" })
      ).toBeNull()
    );
  });

  it("shows an inline error (no API call) for syntactically invalid JSON", async () => {
    await openEditForEntryA();
    fireEvent.change(
      screen.getByRole("textbox", { name: "Edit memory value" }),
      { target: { value: "{{bad json" } }
    );
    fireEvent.click(screen.getByRole("button", { name: /^save$/i }));

    // Error message appears, textarea stays open, api.post NOT called
    await waitFor(() =>
      expect(screen.getByText(/invalid json/i)).toBeTruthy()
    );
    expect(mockPost).not.toHaveBeenCalled();
    expect(screen.getByRole("textbox", { name: "Edit memory value" })).toBeTruthy();
  });

  it("Cancel closes the edit form without calling api.post", async () => {
    await openEditForEntryA();
    fireEvent.click(screen.getByRole("button", { name: /^cancel$/i }));

    await waitFor(() =>
      expect(
        screen.queryByRole("textbox", { name: "Edit memory value" })
      ).toBeNull()
    );
    expect(mockPost).not.toHaveBeenCalled();
  });
});

// ── Delete flow ───────────────────────────────────────────────────────────────

describe("MemoryInspectorPanel — delete flow", () => {
  beforeEach(() => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockGet.mockResolvedValue(TWO_ENTRIES as any);
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockDel.mockResolvedValue({ status: "deleted" } as any);
  });

  /** Helper: expand entry-A and click its Delete button */
  async function openDeleteForEntryA() {
    render(<MemoryInspectorPanel workspaceId="ws-1" />);
    await waitFor(() => screen.getByText("task-queue"));
    fireEvent.click(screen.getByText("task-queue").closest("button")!);
    await waitFor(() =>
      screen.getByRole("button", { name: "Delete task-queue" })
    );
    fireEvent.click(screen.getByRole("button", { name: "Delete task-queue" }));
  }

  it("opens the ConfirmDialog when Delete is clicked", async () => {
    await openDeleteForEntryA();
    expect(screen.getByTestId("confirm-dialog")).toBeTruthy();
    expect(screen.getByTestId("dialog-title").textContent).toBe(
      "Delete memory entry"
    );
  });

  it("includes the key in the dialog message", async () => {
    await openDeleteForEntryA();
    expect(screen.getByTestId("dialog-message").textContent).toContain(
      "task-queue"
    );
  });

  it("calls api.del with the correct URL-encoded path on confirm", async () => {
    await openDeleteForEntryA();
    fireEvent.click(screen.getByText("Confirm Delete"));
    await waitFor(() =>
      expect(mockDel).toHaveBeenCalledWith(
        "/workspaces/ws-1/memory/task-queue"
      )
    );
  });

  it("removes the entry from the list optimistically after confirm", async () => {
    await openDeleteForEntryA();
    fireEvent.click(screen.getByText("Confirm Delete"));
    await waitFor(() =>
      expect(screen.queryByText("task-queue")).toBeNull()
    );
    // Sibling entry unaffected
    expect(screen.getByText("session-token")).toBeTruthy();
  });

  it("closes the ConfirmDialog without deleting when Cancel is clicked", async () => {
    await openDeleteForEntryA();
    fireEvent.click(screen.getByText("Cancel Delete"));
    await waitFor(() =>
      expect(screen.queryByTestId("confirm-dialog")).toBeNull()
    );
    expect(mockDel).not.toHaveBeenCalled();
    // Entry still present
    expect(screen.getByText("task-queue")).toBeTruthy();
  });
});

// ── Refresh ───────────────────────────────────────────────────────────────────

describe("MemoryInspectorPanel — Refresh button", () => {
  it("re-fetches entries when the Refresh button is clicked", async () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockGet.mockResolvedValue([] as any);
    render(<MemoryInspectorPanel workspaceId="ws-1" />);
    await waitFor(() => screen.getByText("No memory entries yet"));

    expect(mockGet).toHaveBeenCalledTimes(1);

    fireEvent.click(screen.getByRole("button", { name: "Refresh memory entries" }));
    await waitFor(() => expect(mockGet).toHaveBeenCalledTimes(2));
  });
});
