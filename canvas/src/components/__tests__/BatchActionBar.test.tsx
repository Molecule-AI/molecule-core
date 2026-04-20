// @vitest-environment jsdom
/**
 * BatchActionBar tests — Phase 20.3
 *
 * Covers:
 *   - Not rendered when fewer than 2 nodes selected
 *   - Renders with correct count badge when 2+ selected
 *   - Restart/Pause/Delete buttons exist with correct labels
 *   - Clear selection button exists
 *   - ConfirmDialog appears on destructive action click
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, cleanup } from "@testing-library/react";

afterEach(() => {
  cleanup();
});

// ── Mocks ────────────────────────────────────────────────────────────────────

vi.mock("@/components/Toaster", () => ({
  showToast: vi.fn(),
}));

const mockClearSelection = vi.fn();
const mockBatchRestart = vi.fn(() => Promise.resolve());
const mockBatchPause = vi.fn(() => Promise.resolve());
const mockBatchDelete = vi.fn(() => Promise.resolve());

let mockSelectedNodeIds = new Set<string>();

vi.mock("@/store/canvas", () => ({
  useCanvasStore: vi.fn((selector: (s: Record<string, unknown>) => unknown) =>
    selector({
      selectedNodeIds: mockSelectedNodeIds,
      clearSelection: mockClearSelection,
      batchRestart: mockBatchRestart,
      batchPause: mockBatchPause,
      batchDelete: mockBatchDelete,
    })
  ),
}));

// Mock ConfirmDialog to just render buttons for testing
vi.mock("@/components/ConfirmDialog", () => ({
  ConfirmDialog: ({
    open,
    title,
    onConfirm,
    onCancel,
  }: {
    open: boolean;
    title: string;
    confirmLabel: string;
    message: string;
    confirmVariant: string;
    onConfirm: () => void;
    onCancel: () => void;
  }) =>
    open ? (
      <div data-testid="confirm-dialog">
        <span>{title}</span>
        <button onClick={onConfirm}>confirm</button>
        <button onClick={onCancel}>cancel</button>
      </div>
    ) : null,
}));

// Import after mocks
import { BatchActionBar } from "../BatchActionBar";

// ── Tests ────────────────────────────────────────────────────────────────────

describe("BatchActionBar", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockSelectedNodeIds = new Set<string>();
  });

  it("does not render when fewer than 2 nodes selected", () => {
    mockSelectedNodeIds = new Set(["ws-1"]);
    const { container } = render(<BatchActionBar />);
    expect(container.innerHTML).toBe("");
  });

  it("renders count badge when 2+ nodes selected", () => {
    mockSelectedNodeIds = new Set(["ws-1", "ws-2", "ws-3"]);
    render(<BatchActionBar />);
    expect(screen.getByText("3 selected")).toBeTruthy();
  });

  it("renders Restart All, Pause All, Delete All buttons", () => {
    mockSelectedNodeIds = new Set(["ws-1", "ws-2"]);
    render(<BatchActionBar />);
    expect(screen.getByText("Restart All")).toBeTruthy();
    expect(screen.getByText("Pause All")).toBeTruthy();
    expect(screen.getByText("Delete All")).toBeTruthy();
  });

  it("renders clear selection button with aria-label", () => {
    mockSelectedNodeIds = new Set(["ws-1", "ws-2"]);
    render(<BatchActionBar />);
    const clearBtn = screen.getByRole("button", { name: "Clear selection" });
    expect(clearBtn).toBeTruthy();
  });

  it("clicking clear selection calls clearSelection", () => {
    mockSelectedNodeIds = new Set(["ws-1", "ws-2"]);
    render(<BatchActionBar />);
    fireEvent.click(screen.getByRole("button", { name: "Clear selection" }));
    expect(mockClearSelection).toHaveBeenCalled();
  });

  it("clicking Delete All opens ConfirmDialog", () => {
    mockSelectedNodeIds = new Set(["ws-1", "ws-2"]);
    render(<BatchActionBar />);
    fireEvent.click(screen.getByText("Delete All"));
    expect(screen.getByTestId("confirm-dialog")).toBeTruthy();
  });

  it("has role=toolbar with aria-label", () => {
    mockSelectedNodeIds = new Set(["ws-1", "ws-2"]);
    render(<BatchActionBar />);
    const toolbar = screen.getByRole("toolbar");
    expect(toolbar.getAttribute("aria-label")).toBe("Batch workspace actions");
  });
});
