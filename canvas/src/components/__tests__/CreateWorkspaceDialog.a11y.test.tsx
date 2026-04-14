// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor, cleanup } from "@testing-library/react";

afterEach(() => {
  cleanup();
});

vi.mock("@/lib/api", () => ({
  api: {
    get: vi.fn().mockResolvedValue([]),
    post: vi.fn().mockResolvedValue({}),
  },
}));

// Import component AFTER mocks
import { CreateWorkspaceButton } from "../CreateWorkspaceDialog";

async function openDialog() {
  render(<CreateWorkspaceButton />);
  const trigger = screen
    .getAllByRole("button")
    .find((b) => b.textContent?.includes("New Workspace"));
  expect(trigger).toBeTruthy();
  fireEvent.click(trigger!);
  await waitFor(() =>
    expect(screen.queryByRole("dialog")).toBeTruthy()
  );
}

describe("CreateWorkspaceDialog — accessibility", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("dialog is absent before the trigger is clicked", () => {
    render(<CreateWorkspaceButton />);
    expect(screen.queryByRole("dialog")).toBeNull();
  });

  it("clicking the trigger renders a role=dialog", async () => {
    await openDialog();
    expect(screen.getByRole("dialog")).toBeTruthy();
  });

  it("dialog has aria-labelledby pointing to the 'Create Workspace' title", async () => {
    await openDialog();
    const dialog = screen.getByRole("dialog");
    const labelledBy = dialog.getAttribute("aria-labelledby");
    expect(labelledBy).toBeTruthy();
    const titleEl = document.getElementById(labelledBy!);
    expect(titleEl?.textContent?.trim()).toBe("Create Workspace");
  });

  it("dialog has data-state='open' when visible (Radix modal state)", async () => {
    await openDialog();
    const dialog = screen.getByRole("dialog");
    expect(dialog.getAttribute("data-state")).toBe("open");
  });

  it("Cancel button closes the dialog", async () => {
    await openDialog();
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
  });

  it("empty-name submit renders a role=alert error message", async () => {
    await openDialog();
    // Click Create without filling in Name
    fireEvent.click(screen.getByRole("button", { name: "Create" }));
    await waitFor(() =>
      expect(screen.getByRole("alert")).toBeTruthy()
    );
    expect(screen.getByRole("alert").textContent).toContain("required");
  });

  it("tier buttons have role=radio and aria-checked reflects selection", async () => {
    await openDialog();
    const radios = screen.getAllByRole("radio");
    expect(radios.length).toBe(3);
    // T1 is default selection
    const t1 = radios.find((r) => r.textContent?.includes("T1"));
    const t2 = radios.find((r) => r.textContent?.includes("T2"));
    expect(t1?.getAttribute("aria-checked")).toBe("true");
    expect(t2?.getAttribute("aria-checked")).toBe("false");
    // Click T2 and verify aria-checked flips
    fireEvent.click(t2!);
    await waitFor(() =>
      expect(t2?.getAttribute("aria-checked")).toBe("true")
    );
  });
});
