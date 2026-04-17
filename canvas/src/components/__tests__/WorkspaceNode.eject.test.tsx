// @vitest-environment jsdom
/**
 * Tests for issue #854 — TeamMemberChip eject button:
 *   - aria-label must be dynamic: `Extract ${childName} from team`
 *   - title must be dynamic: `Extract ${childName} from team`
 *   - EjectIcon svg must carry aria-hidden="true"
 */
import { describe, it, expect, vi, afterEach } from "vitest";
import { render, cleanup } from "@testing-library/react";
import type { Node } from "@xyflow/react";
import type { WorkspaceNodeData } from "@/store/canvas";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

// ── Mock @xyflow/react ─────────────────────────────────────────────────────────
vi.mock("@xyflow/react", () => ({
  Handle: () => null,
  Position: { Bottom: "bottom", Top: "top" },
  useReactFlow: vi.fn(),
}));

// ── Mock Toaster ───────────────────────────────────────────────────────────────
vi.mock("@/components/Toaster", () => ({ showToast: vi.fn() }));

// ── Mock Tooltip ───────────────────────────────────────────────────────────────
vi.mock("@/components/Tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

// ── Mock design tokens ─────────────────────────────────────────────────────────
vi.mock("@/lib/design-tokens", () => ({
  STATUS_CONFIG: {
    online: { label: "Online", dot: "bg-emerald-400", bar: "from-emerald-500/10" },
    offline: { label: "Offline", dot: "bg-zinc-600", bar: "from-zinc-700/10" },
    provisioning: { label: "Provisioning", dot: "bg-sky-400", bar: "from-sky-500/10" },
    degraded: { label: "Degraded", dot: "bg-amber-400", bar: "from-amber-500/10" },
    failed: { label: "Failed", dot: "bg-red-400", bar: "from-red-500/10" },
    paused: { label: "Paused", dot: "bg-zinc-500", bar: "from-zinc-600/10" },
  },
  TIER_CONFIG: {
    1: { label: "T1", color: "text-zinc-400 bg-zinc-800" },
    2: { label: "T2", color: "text-blue-400 bg-blue-900/40" },
  },
}));

// ── Canvas store mock state ────────────────────────────────────────────────────
const PARENT_ID = "parent-ws";
const CHILD_ID = "child-ws";
const CHILD_NAME = "Child Workspace";

function makeNodeData(overrides: Partial<WorkspaceNodeData> = {}): WorkspaceNodeData {
  return {
    name: "Test WS",
    role: "agent",
    tier: 1,
    status: "online",
    agentCard: null,
    url: "http://localhost:9000",
    parentId: null,
    activeTasks: 0,
    lastErrorRate: 0,
    lastSampleError: "",
    uptimeSeconds: 60,
    currentTask: "",
    collapsed: false,
    runtime: "",
    needsRestart: false,
    budgetLimit: null,
    ...overrides,
  } as WorkspaceNodeData;
}

const parentNodeData = makeNodeData({ name: "Parent WS", parentId: null });
const childNodeData = makeNodeData({ name: CHILD_NAME, parentId: PARENT_ID });

const allNodes: Node<WorkspaceNodeData>[] = [
  { id: PARENT_ID, type: "workspaceNode", position: { x: 0, y: 0 }, data: parentNodeData },
  { id: CHILD_ID, type: "workspaceNode", position: { x: 0, y: 0 }, data: childNodeData, hidden: true },
];

// Build a selector-compatible mock of useCanvasStore
const mockStoreState = {
  nodes: allNodes,
  edges: [],
  selectedNodeId: null,
  panelTab: "chat",
  dragOverNodeId: null,
  contextMenu: null,
  searchOpen: false,
  viewport: { x: 0, y: 0, zoom: 1 },
  selectNode: vi.fn(),
  openContextMenu: vi.fn(),
  nestNode: vi.fn(),
  isDescendant: vi.fn(() => false),
  restartWorkspace: vi.fn(),
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

// ── Mock zustand/react/shallow ─────────────────────────────────────────────────
vi.mock("zustand/react/shallow", () => ({
  useShallow: (fn: (s: typeof mockStoreState) => unknown) => fn,
}));

// ── Import component AFTER mocks ───────────────────────────────────────────────
import { WorkspaceNode } from "../WorkspaceNode";

// ── Helpers ────────────────────────────────────────────────────────────────────
function renderParentNode() {
  return render(
    <WorkspaceNode
      id={PARENT_ID}
      data={parentNodeData}
      // NodeProps — all required fields included; React Flow internals unused in mock env
      type="workspaceNode"
      selected={false}
      isConnectable={true}
      zIndex={0}
      positionAbsoluteX={0}
      positionAbsoluteY={0}
      dragging={false}
      draggable={false}
      selectable={false}
      deletable={false}
    />
  );
}

// ── Tests ──────────────────────────────────────────────────────────────────────

describe("TeamMemberChip eject button — aria-label (issue #854)", () => {
  it("eject button has a dynamic aria-label containing the child workspace name", () => {
    const { container } = renderParentNode();
    const buttons = container.querySelectorAll("button");
    const ejectBtn = Array.from(buttons).find(
      (b) => b.getAttribute("aria-label")?.includes("Extract") && b.getAttribute("aria-label")?.includes("from team")
    );
    expect(ejectBtn).toBeTruthy();
    expect(ejectBtn?.getAttribute("aria-label")).toBe(`Extract ${CHILD_NAME} from team`);
  });
});

describe("TeamMemberChip eject button — title tooltip (issue #854)", () => {
  it("eject button has a dynamic title tooltip containing the child workspace name", () => {
    const { container } = renderParentNode();
    const buttons = container.querySelectorAll("button");
    const ejectBtn = Array.from(buttons).find(
      (b) => b.getAttribute("title")?.includes("Extract") && b.getAttribute("title")?.includes("from team")
    );
    expect(ejectBtn).toBeTruthy();
    expect(ejectBtn?.getAttribute("title")).toBe(`Extract ${CHILD_NAME} from team`);
  });

  it("aria-label and title are identical (both use child workspace name)", () => {
    const { container } = renderParentNode();
    const buttons = container.querySelectorAll("button");
    const ejectBtn = Array.from(buttons).find(
      (b) => b.getAttribute("aria-label")?.startsWith("Extract")
    );
    expect(ejectBtn).toBeTruthy();
    expect(ejectBtn?.getAttribute("aria-label")).toBe(ejectBtn?.getAttribute("title"));
  });
});

describe("TeamMemberChip eject button — aria-hidden on EjectIcon (issue #854)", () => {
  it("EjectIcon svg has aria-hidden='true' to prevent AT double-announcement", () => {
    const { container } = renderParentNode();
    const buttons = container.querySelectorAll("button");
    const ejectBtn = Array.from(buttons).find(
      (b) => b.getAttribute("aria-label")?.startsWith("Extract")
    );
    expect(ejectBtn).toBeTruthy();
    const svg = ejectBtn?.querySelector("svg");
    expect(svg).toBeTruthy();
    expect(svg?.getAttribute("aria-hidden")).toBe("true");
  });
});
