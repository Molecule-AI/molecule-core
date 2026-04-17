// @vitest-environment jsdom
/**
 * Tests for the budget_limit field in DetailsTab (issue #541).
 * Covers: display in read view, editing + PATCH, exceeded badge,
 * null/unlimited states, and cancel-revert.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor, cleanup } from "@testing-library/react";

// ── Mocks ─────────────────────────────────────────────────────────────────────

vi.mock("@/lib/api", () => ({
  api: {
    get: vi.fn(),
    patch: vi.fn(),
    del: vi.fn(),
    post: vi.fn(),
  },
}));

vi.mock("@/store/canvas", () => ({
  useCanvasStore: vi.fn((selector: (s: unknown) => unknown) =>
    selector({
      updateNodeData: mockUpdateNodeData,
      removeNode: vi.fn(),
      selectNode: vi.fn(),
    })
  ),
}));

vi.mock("../StatusDot", () => ({ StatusDot: () => null }));

import { api } from "@/lib/api";
import { DetailsTab } from "../tabs/DetailsTab";

const mockPatch = vi.mocked(api.patch);
const mockGet = vi.mocked(api.get);
const mockUpdateNodeData = vi.fn();

// ── Base workspace data ────────────────────────────────────────────────────────

function makeData(overrides: Record<string, unknown> = {}) {
  return {
    name: "Test Agent",
    role: "Researcher",
    tier: 1,
    status: "online",
    agentCard: null,
    activeTasks: 0,
    collapsed: false,
    lastErrorRate: 0,
    lastSampleError: "",
    url: "http://localhost:8080",
    parentId: null,
    currentTask: "",
    runtime: "langgraph",
    needsRestart: false,
    budgetLimit: null,
    budgetUsed: null,
    ...overrides,
  };
}

beforeEach(() => {
  vi.clearAllMocks();
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  mockGet.mockResolvedValue([] as any);
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  mockPatch.mockResolvedValue({} as any);
});

afterEach(() => {
  cleanup();
});

// ── Read view ─────────────────────────────────────────────────────────────────

describe("DetailsTab — budget_limit read view", () => {
  it("shows 'Unlimited' when budgetLimit is null", () => {
    render(<DetailsTab workspaceId="ws-1" data={makeData({ budgetLimit: null })} />);
    expect(screen.getByText("Unlimited")).toBeTruthy();
  });

  it("shows formatted dollar amount when budgetLimit is set", () => {
    render(<DetailsTab workspaceId="ws-1" data={makeData({ budgetLimit: 100 })} />);
    expect(screen.getByText("$100.00")).toBeTruthy();
  });

  it("shows budget used row when budgetUsed is present", () => {
    render(
      <DetailsTab
        workspaceId="ws-1"
        data={makeData({ budgetLimit: 100, budgetUsed: 42.5 })}
      />
    );
    expect(screen.getByText("$42.50")).toBeTruthy();
  });

  it("does NOT show budget used row when budgetUsed is null", () => {
    render(
      <DetailsTab
        workspaceId="ws-1"
        data={makeData({ budgetLimit: 100, budgetUsed: null })}
      />
    );
    // "Budget used" label should not appear
    expect(screen.queryByText("Budget used")).toBeNull();
  });
});

// ── Budget exceeded badge ─────────────────────────────────────────────────────

describe("DetailsTab — budget exceeded badge", () => {
  it("shows exceeded badge when budgetUsed > budgetLimit", () => {
    render(
      <DetailsTab
        workspaceId="ws-1"
        data={makeData({ budgetLimit: 50, budgetUsed: 75 })}
      />
    );
    expect(screen.getByTestId("budget-exceeded-badge")).toBeTruthy();
    expect(screen.getByText("Budget limit exceeded")).toBeTruthy();
  });

  it("does NOT show exceeded badge when budgetUsed equals budgetLimit", () => {
    render(
      <DetailsTab
        workspaceId="ws-1"
        data={makeData({ budgetLimit: 100, budgetUsed: 100 })}
      />
    );
    expect(screen.queryByTestId("budget-exceeded-badge")).toBeNull();
  });

  it("does NOT show exceeded badge when budgetUsed < budgetLimit", () => {
    render(
      <DetailsTab
        workspaceId="ws-1"
        data={makeData({ budgetLimit: 200, budgetUsed: 50 })}
      />
    );
    expect(screen.queryByTestId("budget-exceeded-badge")).toBeNull();
  });

  it("does NOT show exceeded badge when budgetLimit is null (unlimited)", () => {
    render(
      <DetailsTab
        workspaceId="ws-1"
        data={makeData({ budgetLimit: null, budgetUsed: 999 })}
      />
    );
    expect(screen.queryByTestId("budget-exceeded-badge")).toBeNull();
  });

  it("does NOT show exceeded badge when budgetUsed is null", () => {
    render(
      <DetailsTab
        workspaceId="ws-1"
        data={makeData({ budgetLimit: 50, budgetUsed: null })}
      />
    );
    expect(screen.queryByTestId("budget-exceeded-badge")).toBeNull();
  });

  it("exceeded badge has role='status' for accessible announcement", () => {
    render(
      <DetailsTab
        workspaceId="ws-1"
        data={makeData({ budgetLimit: 10, budgetUsed: 20 })}
      />
    );
    const badge = screen.getByTestId("budget-exceeded-badge");
    expect(badge.getAttribute("role")).toBe("status");
  });
});

// ── Edit + PATCH ──────────────────────────────────────────────────────────────

describe("DetailsTab — budget_limit editing", () => {
  async function openEdit() {
    const editBtn = screen.getAllByRole("button").find((b) => b.textContent === "Edit");
    fireEvent.click(editBtn!);
    await waitFor(() => expect(screen.getByPlaceholderText("Leave blank for unlimited")).toBeTruthy());
  }

  it("shows budget_limit input with placeholder 'Leave blank for unlimited' when editing", async () => {
    render(<DetailsTab workspaceId="ws-1" data={makeData({ budgetLimit: null })} />);
    await openEdit();
    const input = screen.getByPlaceholderText("Leave blank for unlimited") as HTMLInputElement;
    expect(input).toBeTruthy();
    expect(input.value).toBe("");
  });

  it("pre-fills input with existing budgetLimit value", async () => {
    render(<DetailsTab workspaceId="ws-1" data={makeData({ budgetLimit: 150 })} />);
    await openEdit();
    const input = screen.getByPlaceholderText("Leave blank for unlimited") as HTMLInputElement;
    expect(input.value).toBe("150");
  });

  it("sends budget_limit as a number in PATCH body", async () => {
    render(<DetailsTab workspaceId="ws-1" data={makeData({ budgetLimit: null })} />);
    await openEdit();

    fireEvent.change(screen.getByPlaceholderText("Leave blank for unlimited"), {
      target: { value: "300" },
    });

    const saveBtn = screen.getAllByRole("button").find((b) => b.textContent === "Save");
    fireEvent.click(saveBtn!);

    await waitFor(() => expect(mockPatch).toHaveBeenCalled());
    const body = mockPatch.mock.calls[0][1] as Record<string, unknown>;
    expect(body.budget_limit).toBe(300);
  });

  it("sends budget_limit as null when field is cleared", async () => {
    render(<DetailsTab workspaceId="ws-1" data={makeData({ budgetLimit: 100 })} />);
    await openEdit();

    fireEvent.change(screen.getByPlaceholderText("Leave blank for unlimited"), {
      target: { value: "" },
    });

    const saveBtn = screen.getAllByRole("button").find((b) => b.textContent === "Save");
    fireEvent.click(saveBtn!);

    await waitFor(() => expect(mockPatch).toHaveBeenCalled());
    const body = mockPatch.mock.calls[0][1] as Record<string, unknown>;
    expect(body.budget_limit).toBeNull();
  });

  it("calls updateNodeData with the new budgetLimit on successful save", async () => {
    render(<DetailsTab workspaceId="ws-1" data={makeData({ budgetLimit: null })} />);
    await openEdit();

    fireEvent.change(screen.getByPlaceholderText("Leave blank for unlimited"), {
      target: { value: "500" },
    });

    const saveBtn = screen.getAllByRole("button").find((b) => b.textContent === "Save");
    fireEvent.click(saveBtn!);

    await waitFor(() => expect(mockUpdateNodeData).toHaveBeenCalled());
    const updateArgs = mockUpdateNodeData.mock.calls[0][1] as Record<string, unknown>;
    expect(updateArgs.budgetLimit).toBe(500);
  });

  it("restores original budgetLimit when Cancel is clicked", async () => {
    render(<DetailsTab workspaceId="ws-1" data={makeData({ budgetLimit: 75 })} />);
    await openEdit();

    // Change the value
    fireEvent.change(screen.getByPlaceholderText("Leave blank for unlimited"), {
      target: { value: "9999" },
    });

    // Cancel
    const cancelBtn = screen.getAllByRole("button").find((b) => b.textContent === "Cancel");
    fireEvent.click(cancelBtn!);

    // Re-enter edit mode — should show original value
    await openEdit();
    const input = screen.getByPlaceholderText("Leave blank for unlimited") as HTMLInputElement;
    expect(input.value).toBe("75");
  });
});
