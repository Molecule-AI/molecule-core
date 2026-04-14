// @vitest-environment jsdom
import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent, cleanup } from "@testing-library/react";

afterEach(() => {
  cleanup();
});

// ── Mock all tab content components to null ──────────────────────────────────
vi.mock("../tabs/DetailsTab", () => ({ DetailsTab: () => null }));
vi.mock("../tabs/SkillsTab", () => ({ SkillsTab: () => null }));
vi.mock("../tabs/ChatTab", () => ({ ChatTab: () => null }));
vi.mock("../tabs/ConfigTab", () => ({ ConfigTab: () => null }));
vi.mock("../tabs/TerminalTab", () => ({ TerminalTab: () => null }));
vi.mock("../tabs/FilesTab", () => ({ FilesTab: () => null }));
vi.mock("../tabs/MemoryTab", () => ({ MemoryTab: () => null }));
vi.mock("../tabs/TracesTab", () => ({ TracesTab: () => null }));
vi.mock("../tabs/EventsTab", () => ({ EventsTab: () => null }));
vi.mock("../tabs/ActivityTab", () => ({ ActivityTab: () => null }));
vi.mock("../tabs/ScheduleTab", () => ({ ScheduleTab: () => null }));
vi.mock("../tabs/ChannelsTab", () => ({ ChannelsTab: () => null }));

// ── Mock StatusDot and Tooltip ───────────────────────────────────────────────
vi.mock("../StatusDot", () => ({ StatusDot: () => null }));
vi.mock("../Tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));
vi.mock("@/components/Toaster", () => ({ showToast: vi.fn() }));

// ── Mock canvas store ────────────────────────────────────────────────────────
const mockSetPanelTab = vi.fn();

const mockStoreState = {
  selectedNodeId: "ws-1",
  panelTab: "chat",
  setPanelTab: mockSetPanelTab,
  selectNode: vi.fn(),
  nodes: [
    {
      id: "ws-1",
      data: {
        name: "Test WS",
        status: "online",
        tier: 1,
        role: "Engineer",
        parentId: null,
        needsRestart: false,
        currentTask: null,
        agentCard: null,
      },
    },
  ],
};

vi.mock("@/store/canvas", () => ({
  useCanvasStore: Object.assign(
    vi.fn((selector: (s: typeof mockStoreState) => unknown) =>
      selector(mockStoreState)
    ),
    { getState: () => mockStoreState }
  ),
  summarizeWorkspaceCapabilities: () => ({ runtime: "claude-code", skillCount: 0 }),
}));

// ── Import component under test AFTER all mocks ──────────────────────────────
import { SidePanel } from "../SidePanel";

const TABS = [
  "chat", "activity", "details", "skills", "terminal",
  "config", "schedule", "channels", "files", "memory", "traces", "events",
];

describe("SidePanel — ARIA tablist pattern", () => {
  it("renders a tablist with aria-label='Workspace panel tabs'", () => {
    render(<SidePanel />);
    const tablist = screen.getByRole("tablist");
    expect(tablist).toBeTruthy();
    expect(tablist.getAttribute("aria-label")).toBe("Workspace panel tabs");
  });

  it("renders exactly 12 tab buttons", () => {
    render(<SidePanel />);
    const tabs = screen.getAllByRole("tab");
    expect(tabs.length).toBe(12);
  });

  it("active tab (chat) has aria-selected='true'", () => {
    render(<SidePanel />);
    const chatTab = screen.getAllByRole("tab").find(
      (t) => t.id === "tab-chat"
    );
    expect(chatTab?.getAttribute("aria-selected")).toBe("true");
  });

  it("all other 11 tabs have aria-selected='false'", () => {
    render(<SidePanel />);
    const tabs = screen.getAllByRole("tab");
    const inactive = tabs.filter((t) => t.id !== "tab-chat");
    expect(inactive.length).toBe(11);
    for (const tab of inactive) {
      expect(tab.getAttribute("aria-selected")).toBe("false");
    }
  });

  it("active tab has tabIndex=0 and all others have tabIndex=-1 (roving tabIndex)", () => {
    render(<SidePanel />);
    const tabs = screen.getAllByRole("tab");
    const zeros = tabs.filter((t) => t.getAttribute("tabindex") === "0");
    const minusOnes = tabs.filter((t) => t.getAttribute("tabindex") === "-1");
    expect(zeros.length).toBe(1);
    expect(zeros[0].id).toBe("tab-chat");
    expect(minusOnes.length).toBe(11);
  });

  it("active tab has aria-controls='panel-chat' and id='tab-chat'", () => {
    render(<SidePanel />);
    const chatTab = document.getElementById("tab-chat");
    expect(chatTab).toBeTruthy();
    expect(chatTab?.getAttribute("aria-controls")).toBe("panel-chat");
  });

  it("renders a role=tabpanel for the active tab", () => {
    render(<SidePanel />);
    const tabpanel = screen.getByRole("tabpanel");
    expect(tabpanel).toBeTruthy();
  });

  it("tabpanel has id='panel-chat' and aria-labelledby='tab-chat'", () => {
    render(<SidePanel />);
    const panel = document.getElementById("panel-chat");
    expect(panel).toBeTruthy();
    expect(panel?.getAttribute("aria-labelledby")).toBe("tab-chat");
  });

  it("ArrowRight calls setPanelTab with 'activity' (next tab after chat)", () => {
    render(<SidePanel />);
    const tablist = screen.getByRole("tablist");
    fireEvent.keyDown(tablist, { key: "ArrowRight" });
    expect(mockSetPanelTab).toHaveBeenCalledWith("activity");
  });

  it("ArrowLeft from 'chat' (first) wraps to 'events' (last)", () => {
    render(<SidePanel />);
    const tablist = screen.getByRole("tablist");
    fireEvent.keyDown(tablist, { key: "ArrowLeft" });
    expect(mockSetPanelTab).toHaveBeenCalledWith("events");
  });

  it("Home key calls setPanelTab with 'chat' (first tab)", () => {
    render(<SidePanel />);
    const tablist = screen.getByRole("tablist");
    fireEvent.keyDown(tablist, { key: "Home" });
    expect(mockSetPanelTab).toHaveBeenCalledWith("chat");
  });

  it("End key calls setPanelTab with 'events' (last tab)", () => {
    render(<SidePanel />);
    const tablist = screen.getByRole("tablist");
    fireEvent.keyDown(tablist, { key: "End" });
    expect(mockSetPanelTab).toHaveBeenCalledWith("events");
  });
});
