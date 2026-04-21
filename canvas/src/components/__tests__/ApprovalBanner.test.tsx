// @vitest-environment jsdom
/**
 * ApprovalBanner tests — covers polling, approve/deny actions, and empty state.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, cleanup, fireEvent, act } from "@testing-library/react";

// ── Mocks (hoisted before imports) ────────────────────────────────────────────

const mockGet = vi.fn();
const mockPost = vi.fn();

vi.mock("@/lib/api", () => ({
  api: {
    get: (...args: unknown[]) => mockGet(...args),
    post: (...args: unknown[]) => mockPost(...args),
  },
}));

vi.mock("./Toaster", () => ({
  showToast: vi.fn(),
}));

// ── Imports (after mocks) ─────────────────────────────────────────────────────

import { ApprovalBanner } from "../ApprovalBanner";

// ── Helpers ───────────────────────────────────────────────────────────────────

const makePendingApproval = (overrides: Record<string, unknown> = {}) => ({
  id: "approval-1",
  workspace_id: "ws-1",
  workspace_name: "Research Agent",
  action: "Execute shell command: rm -rf /tmp/cache",
  reason: "Agent wants to clear cache",
  status: "pending",
  created_at: new Date().toISOString(),
  ...overrides,
});

beforeEach(() => {
  vi.useFakeTimers();
  mockGet.mockReset();
  mockPost.mockReset();
});

afterEach(() => {
  cleanup();
  vi.useRealTimers();
  vi.restoreAllMocks();
});

// ── Tests ────────────────────────────────────────────────────────────────────

describe("ApprovalBanner — empty state", () => {
  it("renders nothing when no pending approvals", async () => {
    mockGet.mockResolvedValue([]);

    const { container } = render(<ApprovalBanner />);
    await act(async () => {});

    expect(container.innerHTML).toBe("");
  });

  it("renders nothing when API errors", async () => {
    mockGet.mockRejectedValue(new Error("network error"));

    const { container } = render(<ApprovalBanner />);
    await act(async () => {});

    expect(container.innerHTML).toBe("");
  });
});

describe("ApprovalBanner — with approvals", () => {
  it("renders approval cards with workspace name and action", async () => {
    const approval = makePendingApproval();
    mockGet.mockResolvedValue([approval]);

    render(<ApprovalBanner />);
    await act(async () => {});

    expect(screen.getByText("Research Agent needs approval")).toBeTruthy();
    expect(screen.getByText("Execute shell command: rm -rf /tmp/cache")).toBeTruthy();
    expect(screen.getByText("Agent wants to clear cache")).toBeTruthy();
  });

  it("renders Approve and Deny buttons", async () => {
    mockGet.mockResolvedValue([makePendingApproval()]);

    render(<ApprovalBanner />);
    await act(async () => {});

    expect(screen.getByText("Approve")).toBeTruthy();
    expect(screen.getByText("Deny")).toBeTruthy();
  });

  it("uses role=alert for accessibility", async () => {
    mockGet.mockResolvedValue([makePendingApproval()]);

    render(<ApprovalBanner />);
    await act(async () => {});

    const alerts = screen.getAllByRole("alert");
    expect(alerts.length).toBeGreaterThan(0);
  });
});

describe("ApprovalBanner — approve action", () => {
  it("removes the approval card after approve", async () => {
    const approval = makePendingApproval();
    mockGet.mockResolvedValue([approval]);
    mockPost.mockResolvedValue({});

    render(<ApprovalBanner />);
    await act(async () => {});

    const approveBtn = screen.getByText("Approve");
    await act(async () => {
      fireEvent.click(approveBtn);
    });

    expect(mockPost).toHaveBeenCalledWith(
      "/workspaces/ws-1/approvals/approval-1/decide",
      { decision: "approved", decided_by: "human" }
    );
  });

  it("removes the approval card after deny", async () => {
    const approval = makePendingApproval();
    mockGet.mockResolvedValue([approval]);
    mockPost.mockResolvedValue({});

    render(<ApprovalBanner />);
    await act(async () => {});

    const denyBtn = screen.getByText("Deny");
    await act(async () => {
      fireEvent.click(denyBtn);
    });

    expect(mockPost).toHaveBeenCalledWith(
      "/workspaces/ws-1/approvals/approval-1/decide",
      { decision: "denied", decided_by: "human" }
    );
  });
});

describe("ApprovalBanner — no reason field", () => {
  it("renders without reason when reason is null", async () => {
    const approval = makePendingApproval({ reason: null });
    mockGet.mockResolvedValue([approval]);

    render(<ApprovalBanner />);
    await act(async () => {});

    expect(screen.getByText("Research Agent needs approval")).toBeTruthy();
    // Reason paragraph should not be present
    expect(screen.queryByText("Agent wants to clear cache")).toBeNull();
  });
});
