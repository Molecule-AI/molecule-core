// @vitest-environment jsdom
import { describe, it, expect, afterEach } from "vitest";
import { render, screen, cleanup } from "@testing-library/react";
import { WorkspaceUsage } from "../WorkspaceUsage";

afterEach(() => {
  cleanup();
});

describe("WorkspaceUsage", () => {
  it("renders without crashing", () => {
    const { container } = render(
      <WorkspaceUsage workspaceId="ws-test-123" />
    );
    expect(container.firstChild).toBeTruthy();
  });

  it("renders the Usage heading", () => {
    render(<WorkspaceUsage workspaceId="ws-test-123" />);
    expect(screen.getByText("Usage")).toBeTruthy();
  });

  it("renders the pending #593 badge", () => {
    render(<WorkspaceUsage workspaceId="ws-test-123" />);
    const badge = screen.getByTestId("usage-pending-badge");
    expect(badge).toBeTruthy();
    expect(badge.textContent).toBe("pending #593");
  });

  it("renders the outer container and stats container", () => {
    render(<WorkspaceUsage workspaceId="ws-test-123" />);
    expect(screen.getByTestId("workspace-usage")).toBeTruthy();
    expect(screen.getByTestId("usage-stats")).toBeTruthy();
  });

  it("renders Input tokens row with placeholder dash", () => {
    render(<WorkspaceUsage workspaceId="ws-test-123" />);
    const row = screen.getByTestId("usage-input-tokens");
    expect(row).toBeTruthy();
    expect(row.textContent).toContain("Input tokens");
    expect(row.textContent).toContain("—");
  });

  it("renders Output tokens row with placeholder dash", () => {
    render(<WorkspaceUsage workspaceId="ws-test-123" />);
    const row = screen.getByTestId("usage-output-tokens");
    expect(row).toBeTruthy();
    expect(row.textContent).toContain("Output tokens");
    expect(row.textContent).toContain("—");
  });

  it("renders Estimated cost row with placeholder dash", () => {
    render(<WorkspaceUsage workspaceId="ws-test-123" />);
    const row = screen.getByTestId("usage-estimated-cost");
    expect(row).toBeTruthy();
    expect(row.textContent).toContain("Estimated cost");
    expect(row.textContent).toContain("—");
  });

  it("accepts any workspaceId without throwing", () => {
    const ids = ["", "ws-abc", "00000000-0000-0000-0000-000000000000"];
    for (const id of ids) {
      const { unmount } = render(<WorkspaceUsage workspaceId={id} />);
      expect(screen.getByTestId("workspace-usage")).toBeTruthy();
      unmount();
    }
  });

  it("does not display live token counts or dollar amounts", () => {
    render(<WorkspaceUsage workspaceId="ws-test-123" />);
    const stats = screen.getByTestId("usage-stats");
    // Placeholder state must not contain any digit sequences
    expect(stats.textContent).not.toMatch(/\d+/);
  });
});
