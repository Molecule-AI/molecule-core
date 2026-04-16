// @vitest-environment jsdom
/**
 * Tests that Canvas.tsx responds to the "molecule:pan-to-node" custom event
 * (fired by canvas-events.ts on WORKSPACE_PROVISIONING for new nodes) by
 * calling fitView({ nodes: [{ id }] }) instead of setCenter with a forced
 * zoom=1 (which was jarring when the user was zoomed out — issue #426).
 */
import { describe, it, expect, vi, afterEach, beforeEach } from "vitest";
import { render, act, cleanup } from "@testing-library/react";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

// ── Shared fitView spy — must be set up before vi.mock hoisting ──────────────
const mockFitView = vi.fn();
const mockFitBounds = vi.fn();

vi.mock("@xyflow/react", () => {
  const ReactFlow = ({
    children,
    "aria-label": ariaLabel,
  }: {
    children?: React.ReactNode;
    "aria-label"?: string;
  }) => (
    <div role="application" data-testid="react-flow" aria-label={ariaLabel}>
      {children}
    </div>
  );
  return {
    __esModule: true,
    default: ReactFlow,
    ReactFlow,
    ReactFlowProvider: ({ children }: { children?: React.ReactNode }) => (
      <>{children}</>
    ),
    Background: () => null,
    Controls: () => null,
    MiniMap: () => null,
    BackgroundVariant: { Dots: "dots" },
    useReactFlow: () => ({
      fitView: mockFitView,
      fitBounds: mockFitBounds,
      setViewport: vi.fn(),
      getIntersectingNodes: vi.fn(() => []),
      setCenter: vi.fn(),
    }),
    applyNodeChanges: vi.fn((_: unknown, nodes: unknown) => nodes),
    useStore: vi.fn(() => ({ width: 800, height: 600 })),
  };
});

// ── Canvas store mock ─────────────────────────────────────────────────────────
const mockStoreState = {
  nodes: [{ id: "ws-1", position: { x: 100, y: 100 }, data: { name: "WS1" } }],
  edges: [],
  selectedNodeId: null,
  panelTab: "chat",
  dragOverNodeId: null,
  contextMenu: null,
  viewport: { x: 0, y: 0, zoom: 1 },
  searchOpen: false,
  onNodesChange: vi.fn(),
  savePosition: vi.fn(),
  saveViewport: vi.fn(),
  selectNode: vi.fn(),
  openContextMenu: vi.fn(),
  closeContextMenu: vi.fn(),
  setDragOverNode: vi.fn(),
  nestNode: vi.fn(),
  isDescendant: vi.fn(() => false),
  setSearchOpen: vi.fn(),
};

vi.mock("@/store/canvas", () => ({
  useCanvasStore: Object.assign(
    vi.fn((selector: (s: typeof mockStoreState) => unknown) =>
      selector(mockStoreState)
    ),
    { getState: () => mockStoreState }
  ),
}));

vi.mock("@/store/socket", () => ({
  connectSocket: vi.fn(),
  disconnectSocket: vi.fn(),
}));

// ── Stub child components ─────────────────────────────────────────────────────
vi.mock("../Toolbar", () => ({ Toolbar: () => null }));
vi.mock("../SidePanel", () => ({ SidePanel: () => null }));
vi.mock("../EmptyState", () => ({ EmptyState: () => null }));
vi.mock("../ContextMenu", () => ({ ContextMenu: () => null }));
vi.mock("../SearchDialog", () => ({ SearchDialog: () => null }));
vi.mock("../ConfirmDialog", () => ({ ConfirmDialog: () => null }));
vi.mock("../TemplatePalette", () => ({ TemplatePalette: () => null }));
vi.mock("../OnboardingWizard", () => ({ OnboardingWizard: () => null }));
vi.mock("../ApprovalBanner", () => ({ ApprovalBanner: () => null }));
vi.mock("../BundleDropZone", () => ({ BundleDropZone: () => null }));
vi.mock("../CreateWorkspaceDialog", () => ({ CreateWorkspaceButton: () => null }));
vi.mock("../settings", () => ({
  SettingsPanel: () => null,
  DeleteConfirmDialog: () => null,
}));
vi.mock("../Toaster", () => ({ Toaster: () => null }));
vi.mock("../WorkspaceNode", () => ({ WorkspaceNode: () => null }));

import { Canvas } from "../Canvas";

// ─────────────────────────────────────────────────────────────────────────────

describe("Canvas — molecule:pan-to-node event handler", () => {
  beforeEach(() => {
    mockFitView.mockClear();
    mockFitBounds.mockClear();
  });

  it("calls fitView with the provisioned nodeId after a 100ms debounce", async () => {
    vi.useFakeTimers();
    render(<Canvas />);

    // Simulate the custom event fired by canvas-events.ts on WORKSPACE_PROVISIONING
    act(() => {
      window.dispatchEvent(
        new CustomEvent("molecule:pan-to-node", { detail: { nodeId: "ws-1" } })
      );
    });

    // fitView should NOT be called yet (100ms debounce)
    expect(mockFitView).not.toHaveBeenCalled();

    // Advance past the 100ms delay
    await act(async () => {
      vi.advanceTimersByTime(150);
    });

    expect(mockFitView).toHaveBeenCalledOnce();
    const [options] = mockFitView.mock.calls[0];
    expect(options.nodes).toEqual([{ id: "ws-1" }]);
    expect(options.duration).toBe(400);
    expect(typeof options.padding).toBe("number");

    vi.useRealTimers();
  });

  it("debounces rapid successive events — only the last nodeId is fitted", async () => {
    vi.useFakeTimers();
    render(<Canvas />);

    act(() => {
      window.dispatchEvent(
        new CustomEvent("molecule:pan-to-node", { detail: { nodeId: "ws-first" } })
      );
      window.dispatchEvent(
        new CustomEvent("molecule:pan-to-node", { detail: { nodeId: "ws-last" } })
      );
    });

    await act(async () => {
      vi.advanceTimersByTime(150);
    });

    // Only one fitView call — the debounce clears the first timer
    expect(mockFitView).toHaveBeenCalledOnce();
    expect(mockFitView.mock.calls[0][0].nodes).toEqual([{ id: "ws-last" }]);

    vi.useRealTimers();
  });
});
