// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor, cleanup, fireEvent } from "@testing-library/react";

vi.mock("@/lib/api", () => ({
  api: { get: vi.fn() },
}));

import { api } from "@/lib/api";
import { ConsoleModal } from "../ConsoleModal";

const mockGet = vi.mocked(api.get);

beforeEach(() => vi.clearAllMocks());
afterEach(cleanup);

describe("ConsoleModal", () => {
  it("returns null when closed — no fetch triggered", () => {
    const { container } = render(
      <ConsoleModal workspaceId="ws-1" open={false} onClose={() => {}} />,
    );
    expect(container.firstChild).toBeNull();
    expect(mockGet).not.toHaveBeenCalled();
  });

  it("fetches console output when opened", async () => {
    mockGet.mockResolvedValueOnce({ output: "boot line 1\nRuntime running (PID 42)\n", instance_id: "i-x" });
    render(<ConsoleModal workspaceId="ws-1" open={true} onClose={() => {}} />);
    await waitFor(() =>
      expect(mockGet).toHaveBeenCalledWith("/workspaces/ws-1/console"),
    );
    await waitFor(() => {
      const out = screen.getByTestId("console-output");
      expect(out.textContent).toContain("Runtime running (PID 42)");
    });
  });

  it("renders a friendly message on 501 (non-CP deploy)", async () => {
    mockGet.mockRejectedValueOnce(new Error("GET /workspaces/ws-1/console: 501 Not Implemented"));
    render(<ConsoleModal workspaceId="ws-1" open={true} onClose={() => {}} />);
    await waitFor(() => {
      const err = screen.getByTestId("console-error");
      expect(err.textContent).toMatch(/only available on cloud/i);
    });
  });

  it("renders a specific message on 404 (instance terminated)", async () => {
    mockGet.mockRejectedValueOnce(new Error("GET /workspaces/ws-1/console: 404 Not Found"));
    render(<ConsoleModal workspaceId="ws-1" open={true} onClose={() => {}} />);
    await waitFor(() => {
      const err = screen.getByTestId("console-error");
      expect(err.textContent).toMatch(/No EC2 instance found/i);
    });
  });

  it("Close button invokes onClose", async () => {
    mockGet.mockResolvedValueOnce({ output: "" });
    const onClose = vi.fn();
    render(<ConsoleModal workspaceId="ws-1" open={true} onClose={onClose} />);
    await waitFor(() => screen.getByText("Close"));
    fireEvent.click(screen.getByText("Close"));
    expect(onClose).toHaveBeenCalled();
  });

  it("Escape key invokes onClose", async () => {
    mockGet.mockResolvedValueOnce({ output: "" });
    const onClose = vi.fn();
    render(<ConsoleModal workspaceId="ws-1" open={true} onClose={onClose} />);
    await waitFor(() => screen.getByText("Close"));
    fireEvent.keyDown(window, { key: "Escape" });
    expect(onClose).toHaveBeenCalled();
  });
});
