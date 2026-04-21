// @vitest-environment jsdom
/**
 * OnboardingWizard tests — covers step progression, localStorage, and dismiss.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, cleanup, fireEvent, act } from "@testing-library/react";

// ── Mocks ────────────────────────────────────────────────────────────────────

// Mock the canvas store with a minimal Zustand-like interface
const mockStoreState = {
  nodes: [] as { id: string }[],
  selectedNodeId: null as string | null,
  panelTab: "chat" as string,
  agentMessages: {} as Record<string, unknown[]>,
  setPanelTab: vi.fn(),
  getState: () => mockStoreState,
};

vi.mock("@/store/canvas", () => ({
  useCanvasStore: Object.assign(
    (selector: (s: typeof mockStoreState) => unknown) => selector(mockStoreState),
    {
      getState: () => mockStoreState,
    }
  ),
}));

// ── Imports ──────────────────────────────────────────────────────────────────

import { OnboardingWizard } from "../OnboardingWizard";

// ── Helpers ──────────────────────────────────────────────────────────────────

const STORAGE_KEY = "molecule-onboarding-complete";

beforeEach(() => {
  localStorage.clear();
  mockStoreState.nodes = [];
  mockStoreState.selectedNodeId = null;
  mockStoreState.panelTab = "chat";
  mockStoreState.agentMessages = {};
  mockStoreState.setPanelTab.mockClear();
});

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

// ── Tests ────────────────────────────────────────────────────────────────────

describe("OnboardingWizard — first-time user", () => {
  it("renders the wizard when no localStorage flag is set", () => {
    render(<OnboardingWizard />);

    expect(screen.getByText("Welcome to Molecule AI")).toBeTruthy();
    expect(screen.getByText("Step 1 of 4")).toBeTruthy();
  });

  it("shows the welcome step when no workspaces exist", () => {
    render(<OnboardingWizard />);

    expect(screen.getByText("Create Workspace")).toBeTruthy();
  });

  it("has proper ARIA role", () => {
    render(<OnboardingWizard />);

    const guide = screen.getByRole("complementary");
    expect(guide.getAttribute("aria-label")).toBe("Onboarding guide");
  });
});

describe("OnboardingWizard — returning user", () => {
  it("renders nothing when onboarding was completed", () => {
    localStorage.setItem(STORAGE_KEY, "true");

    const { container } = render(<OnboardingWizard />);

    expect(container.innerHTML).toBe("");
  });
});

describe("OnboardingWizard — dismiss", () => {
  it("dismisses and sets localStorage when Skip is clicked", () => {
    render(<OnboardingWizard />);

    const skipBtn = screen.getByText("Skip guide");
    fireEvent.click(skipBtn);

    expect(localStorage.getItem(STORAGE_KEY)).toBe("true");
    // Component should now render nothing
    expect(screen.queryByText("Welcome to Molecule AI")).toBeNull();
  });
});

describe("OnboardingWizard — step navigation", () => {
  it("advances to next step when Next is clicked", () => {
    render(<OnboardingWizard />);

    // Start at welcome (step 1)
    expect(screen.getByText("Step 1 of 4")).toBeTruthy();

    const nextBtn = screen.getByText("Next");
    fireEvent.click(nextBtn);

    // Should advance to step 2
    expect(screen.getByText("Step 2 of 4")).toBeTruthy();
    expect(screen.getByText("Set your API key")).toBeTruthy();
  });

  it("shows all 4 steps when stepping through", () => {
    render(<OnboardingWizard />);

    // Step 1
    expect(screen.getByText("Welcome to Molecule AI")).toBeTruthy();

    fireEvent.click(screen.getByText("Next"));
    // Step 2
    expect(screen.getByText("Set your API key")).toBeTruthy();

    fireEvent.click(screen.getByText("Next"));
    // Step 3
    expect(screen.getByText("Send your first message")).toBeTruthy();

    fireEvent.click(screen.getByText("Next"));
    // Step 4 — no Next button, only "Get Started"
    expect(screen.getByText(/You.*re all set/)).toBeTruthy();
    expect(screen.queryByText("Next")).toBeNull();
  });
});

describe("OnboardingWizard — auto-advance", () => {
  it("auto-advances from welcome to api-key when nodes appear", () => {
    const { rerender } = render(<OnboardingWizard />);

    expect(screen.getByText("Step 1 of 4")).toBeTruthy();

    // Simulate workspace creation
    mockStoreState.nodes = [{ id: "ws-1" }];
    rerender(<OnboardingWizard />);

    // Should have advanced to step 2
    expect(screen.getByText("Step 2 of 4")).toBeTruthy();
  });
});

describe("OnboardingWizard — screen reader support", () => {
  it("has a polite live region for step announcements", () => {
    render(<OnboardingWizard />);

    const liveRegion = screen.getByRole("status");
    expect(liveRegion.getAttribute("aria-live")).toBe("polite");
    expect(liveRegion.textContent).toContain("Onboarding step 1 of 4");
  });

  it("has a skip button with descriptive text", () => {
    render(<OnboardingWizard />);

    expect(screen.getByLabelText("Skip onboarding guide")).toBeTruthy();
  });
});
