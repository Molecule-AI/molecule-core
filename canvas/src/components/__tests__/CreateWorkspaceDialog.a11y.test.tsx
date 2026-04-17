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

  // ── Arrow-key navigation (WCAG 2.1 radio group) — Issue #556 ──────────────

  it("selected radio has tabIndex=0, others have tabIndex=-1 (roving tabIndex)", async () => {
    await openDialog();
    const radios = screen.getAllByRole("radio");
    const t1 = radios.find((r) => r.textContent?.includes("T1"))!;
    const t2 = radios.find((r) => r.textContent?.includes("T2"))!;
    const t3 = radios.find((r) => r.textContent?.includes("T3"))!;
    // T1 is default selected
    expect(t1.getAttribute("tabindex")).toBe("0");
    expect(t2.getAttribute("tabindex")).toBe("-1");
    expect(t3.getAttribute("tabindex")).toBe("-1");
  });

  it("ArrowDown moves selection from T1 to T2", async () => {
    await openDialog();
    const radios = screen.getAllByRole("radio");
    const t1 = radios.find((r) => r.textContent?.includes("T1"))!;
    const t2 = radios.find((r) => r.textContent?.includes("T2"))!;
    t1.focus();
    fireEvent.keyDown(t1, { key: "ArrowDown" });
    await waitFor(() => expect(t2.getAttribute("aria-checked")).toBe("true"));
    expect(t1.getAttribute("aria-checked")).toBe("false");
  });

  it("ArrowRight moves selection from T2 to T3", async () => {
    await openDialog();
    const radios = screen.getAllByRole("radio");
    const t2 = radios.find((r) => r.textContent?.includes("T2"))!;
    const t3 = radios.find((r) => r.textContent?.includes("T3"))!;
    fireEvent.click(t2); // select T2 first
    await waitFor(() => expect(t2.getAttribute("aria-checked")).toBe("true"));
    t2.focus();
    fireEvent.keyDown(t2, { key: "ArrowRight" });
    await waitFor(() => expect(t3.getAttribute("aria-checked")).toBe("true"));
  });

  it("ArrowDown wraps from T3 back to T1", async () => {
    await openDialog();
    const radios = screen.getAllByRole("radio");
    const t1 = radios.find((r) => r.textContent?.includes("T1"))!;
    const t3 = radios.find((r) => r.textContent?.includes("T3"))!;
    fireEvent.click(t3); // select T3 first
    await waitFor(() => expect(t3.getAttribute("aria-checked")).toBe("true"));
    t3.focus();
    fireEvent.keyDown(t3, { key: "ArrowDown" });
    await waitFor(() => expect(t1.getAttribute("aria-checked")).toBe("true"));
  });

  it("ArrowUp moves selection from T2 to T1", async () => {
    await openDialog();
    const radios = screen.getAllByRole("radio");
    const t1 = radios.find((r) => r.textContent?.includes("T1"))!;
    const t2 = radios.find((r) => r.textContent?.includes("T2"))!;
    fireEvent.click(t2);
    await waitFor(() => expect(t2.getAttribute("aria-checked")).toBe("true"));
    t2.focus();
    fireEvent.keyDown(t2, { key: "ArrowUp" });
    await waitFor(() => expect(t1.getAttribute("aria-checked")).toBe("true"));
  });

  it("ArrowLeft wraps from T1 back to T3", async () => {
    await openDialog();
    const radios = screen.getAllByRole("radio");
    const t1 = radios.find((r) => r.textContent?.includes("T1"))!;
    const t3 = radios.find((r) => r.textContent?.includes("T3"))!;
    t1.focus();
    fireEvent.keyDown(t1, { key: "ArrowLeft" });
    await waitFor(() => expect(t3.getAttribute("aria-checked")).toBe("true"));
  });
});
