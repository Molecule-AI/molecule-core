// @vitest-environment jsdom
/**
 * WorkspaceNode a11y tests — issue #831
 *
 * Covers the TeamMemberChip sub-component (rendered inside a parent workspace
 * node when that node has children):
 *   - role="button" is present
 *   - aria-label="Select <name>" is present
 *   - pressing Enter triggers onSelect with the child's id
 *   - pressing Space triggers onSelect with the child's id
 *   - the eject button has aria-label="Extract from team"
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, cleanup } from "@testing-library/react";

afterEach(() => {
  cleanup();
});

// ── Mock @xyflow/react (Handles) ──────────────────────────────────────────────
vi.mock("@xyflow/react", () => ({
  Handle: () => null,
  Position: { Top: "top", Bottom: "bottom" },
}));

// ── Mock Tooltip (passthrough) ────────────────────────────────────────────────
vi.mock("@/components/Tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

// ── Mock Toaster ──────────────────────────────────────────────────────────────
vi.mock("@/components/Toaster", () => ({
  showToast: vi.fn(),
}));

// ── Mock design tokens ────────────────────────────────────────────────────────
vi.mock("@/lib/design-tokens", () => ({
  STATUS_CONFIG: {
    online: {
      dot: "bg-emerald-400",
      glow: "",
      bar: "from-emerald-950/30",
      label: "Online",
    },
    offline: {
      dot: "bg-zinc-500",
      glow: "",
      bar: "from-zinc-900",
      label: "Offline",
    },
    degraded: {
      dot: "bg-amber-400",
      glow: "",
      bar: "from-amber-950/30",
      label: "Degraded",
    },
    provisioning: {
      dot: "bg-sky-400",
      glow: "",
      bar: "from-sky-950/30",
      label: "Provisioning",
    },
    failed: {
      dot: "bg-red-400",
      glow: "",
      bar: "from-red-950/30",
      label: "Failed",
    },
  },
  TIER_CONFIG: {
    1: { label: "T1", color: "text-zinc-400 bg-zinc-800" },
    2: { label: "T2", color: "text-zinc-400 bg-zinc-800" },
    3: { label: "T3", color: "text-zinc-400 bg-zinc-800" },
  },
}));

// ── Store state with a parent + one child ────────────────────────────────────

const mockSelectNode = vi.fn();
const mockOpenContextMenu = vi.fn();
const mockNestNode = vi.fn();

const PARENT_ID = "ws-parent";
const CHILD_ID = "ws-child";

const PARENT_DATA = {
  name: "Parent Workspace",
  status: "online",
  tier: 1 as const,
  role: "Manager",
  parentId: null,
  needsRestart: false,
  currentTask: null,
  activeTasks: 0,
  agentCard: null,
  runtime: "langgraph",
  lastSampleError: null,
};

const CHILD_DATA = {
  name: "Child Workspace",
  status: "online",
  tier: 1 as const,
  role: "Worker",
  parentId: PARENT_ID,
  needsRestart: false,
  currentTask: null,
  activeTasks: 0,
  agentCard: null,
  runtime: "langgraph",
  lastSampleError: null,
};

const ALL_NODES = [
  { id: PARENT_ID, position: { x: 0, y: 0 }, data: PARENT_DATA },
  { id: CHILD_ID, position: { x: 0, y: 0 }, data: CHILD_DATA },
];

const mockStoreState = {
  nodes: ALL_NODES,
  selectedNodeId: null,
  dragOverNodeId: null,
  selectNode: mockSelectNode,
  openContextMenu: mockOpenContextMenu,
  nestNode: mockNestNode,
  restartWorkspace: vi.fn(() => Promise.resolve()),
  setPanelTab: vi.fn(),
};

vi.mock("@/store/canvas", () => ({
  useCanvasStore: Object.assign(
    vi.fn((selector: (s: typeof mockStoreState) => unknown) =>
      selector(mockStoreState)
    ),
    { getState: () => mockStoreState }
  ),
}));

// ── Import component AFTER mocks ──────────────────────────────────────────────
import { WorkspaceNode } from "../WorkspaceNode";

// ── Helper ────────────────────────────────────────────────────────────────────

function renderParentNode() {
  // WorkspaceNode's full NodeProps has many optional fields; we only need id+data
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return render(<WorkspaceNode id={PARENT_ID} data={PARENT_DATA as any} />);
}

// ── Tests ─────────────────────────────────────────────────────────────────────

describe("WorkspaceNode — TeamMemberChip a11y (issue #831)", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("TeamMemberChip renders with role='button'", () => {
    renderParentNode();
    // The parent WorkspaceNode div is role=button (aria-label contains the name),
    // and the chip is a separate role=button with aria-label starting with "Select"
    const chip = screen.getByRole("button", {
      name: "Select Child Workspace",
    });
    expect(chip).toBeTruthy();
  });

  it("TeamMemberChip has aria-label='Select <name>'", () => {
    renderParentNode();
    const chip = screen.getByRole("button", {
      name: "Select Child Workspace",
    });
    expect(chip.getAttribute("aria-label")).toBe("Select Child Workspace");
  });

  it("pressing Enter on TeamMemberChip calls selectNode with the child's id", () => {
    renderParentNode();
    const chip = screen.getByRole("button", {
      name: "Select Child Workspace",
    });
    fireEvent.keyDown(chip, { key: "Enter" });
    expect(mockSelectNode).toHaveBeenCalledWith(CHILD_ID);
  });

  it("pressing Space on TeamMemberChip calls selectNode with the child's id", () => {
    renderParentNode();
    const chip = screen.getByRole("button", {
      name: "Select Child Workspace",
    });
    fireEvent.keyDown(chip, { key: " " });
    expect(mockSelectNode).toHaveBeenCalledWith(CHILD_ID);
  });

  it("eject button has aria-label='Extract <name> from team'", () => {
    renderParentNode();
    const ejectBtn = screen.getByRole("button", {
      name: "Extract Child Workspace from team",
    });
    expect(ejectBtn).toBeTruthy();
  });
});
