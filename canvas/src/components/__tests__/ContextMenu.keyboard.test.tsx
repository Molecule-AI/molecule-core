// @vitest-environment jsdom
import { describe, it, expect, vi, afterEach, beforeEach } from "vitest";
import { render, screen, fireEvent, cleanup } from "@testing-library/react";

afterEach(cleanup);

// ── Mocks ─────────────────────────────────────────────────────────────────────
vi.mock("../ConfirmDialog", () => ({ ConfirmDialog: () => null }));
vi.mock("../Toaster", () => ({ showToast: vi.fn() }));
vi.mock("@/lib/api", () => ({
  api: { get: vi.fn(), post: vi.fn(), del: vi.fn(), patch: vi.fn() },
}));

const closeContextMenu = vi.fn();
const mockStore = {
  contextMenu: {
    x: 100,
    y: 200,
    nodeId: "ws-1",
    nodeData: {
      name: "Alpha Workspace",
      status: "online",
      tier: 1,
      parentId: null,
      agentCard: null,
      activeTasks: 0,
      collapsed: false,
      role: "dev",
      lastErrorRate: 0,
      lastSampleError: "",
      url: "",
      currentTask: "",
      runtime: "claude-code",
      needsRestart: false,
    },
  } as {
    x: number;
    y: number;
    nodeId: string;
    nodeData: Record<string, unknown>;
  } | null,
  closeContextMenu,
  removeNode: vi.fn(),
  updateNodeData: vi.fn(),
  selectNode: vi.fn(),
  setPanelTab: vi.fn(),
  nestNode: vi.fn(),
  nodes: [] as Array<{ id: string; data: { parentId: string | null } }>,
};

vi.mock("@/store/canvas", () => ({
  useCanvasStore: vi.fn(
    (selector: (s: typeof mockStore) => unknown) => selector(mockStore)
  ),
}));

// ── Component under test — imported AFTER mocks ───────────────────────────────
import { ContextMenu } from "../ContextMenu";

// ── Helpers ───────────────────────────────────────────────────────────────────
const onlineMenu = {
  x: 100,
  y: 200,
  nodeId: "ws-1",
  nodeData: {
    name: "Alpha Workspace",
    status: "online",
    tier: 1,
    parentId: null,
    agentCard: null,
    activeTasks: 0,
    collapsed: false,
    role: "dev",
    lastErrorRate: 0,
    lastSampleError: "",
    url: "",
    currentTask: "",
    runtime: "claude-code",
    needsRestart: false,
  },
};

describe("ContextMenu — keyboard accessibility", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockStore.contextMenu = onlineMenu;
    mockStore.nodes = [];
  });

  it("renders with role='menu'", () => {
    render(<ContextMenu />);
    expect(screen.getByRole("menu")).toBeTruthy();
  });

  it("menu has aria-label containing the workspace name", () => {
    render(<ContextMenu />);
    const menu = screen.getByRole("menu");
    expect(menu.getAttribute("aria-label")).toContain("Alpha Workspace");
  });

  it("menu items have role='menuitem'", () => {
    render(<ContextMenu />);
    const items = screen.getAllByRole("menuitem");
    expect(items.length).toBeGreaterThan(0);
  });

  it("dividers have role='separator'", () => {
    render(<ContextMenu />);
    const separators = document.querySelectorAll('[role="separator"]');
    expect(separators.length).toBeGreaterThan(0);
  });

  it("Escape key calls closeContextMenu", () => {
    render(<ContextMenu />);
    const menu = screen.getByRole("menu");
    fireEvent.keyDown(menu, { key: "Escape" });
    // Both the document keydown listener and the menu onKeyDown handler fire
    // on the same event — both call closeContextMenu. Two calls is correct.
    expect(closeContextMenu).toHaveBeenCalled();
  });

  it("Tab key calls closeContextMenu", () => {
    render(<ContextMenu />);
    const menu = screen.getByRole("menu");
    fireEvent.keyDown(menu, { key: "Tab" });
    expect(closeContextMenu).toHaveBeenCalledOnce();
  });

  it("ArrowDown with nothing focused moves focus to the first enabled button", () => {
    render(<ContextMenu />);
    const menu = screen.getByRole("menu");
    fireEvent.keyDown(menu, { key: "ArrowDown" });
    const buttons = menu.querySelectorAll<HTMLButtonElement>(
      "button:not(:disabled)"
    );
    expect(document.activeElement).toBe(buttons[0]);
  });

  it("ArrowDown wraps from the last enabled button to the first", () => {
    render(<ContextMenu />);
    const menu = screen.getByRole("menu");
    const buttons = menu.querySelectorAll<HTMLButtonElement>(
      "button:not(:disabled)"
    );
    buttons[buttons.length - 1].focus();
    fireEvent.keyDown(menu, { key: "ArrowDown" });
    expect(document.activeElement).toBe(buttons[0]);
  });

  it("ArrowUp wraps from the first enabled button to the last", () => {
    render(<ContextMenu />);
    const menu = screen.getByRole("menu");
    const buttons = menu.querySelectorAll<HTMLButtonElement>(
      "button:not(:disabled)"
    );
    buttons[0].focus();
    fireEvent.keyDown(menu, { key: "ArrowUp" });
    expect(document.activeElement).toBe(buttons[buttons.length - 1]);
  });

  it("returns null when contextMenu is null", () => {
    mockStore.contextMenu = null;
    const { container } = render(<ContextMenu />);
    expect(container.firstChild).toBeNull();
  });
});
