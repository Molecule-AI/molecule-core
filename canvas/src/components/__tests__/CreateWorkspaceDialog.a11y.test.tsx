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

// ── WCAG 2.1 SC 1.3.1 — Programmatic label association (Issue #558) ──────────
//
// Every <input> rendered by the InputField helper must have a matching <label>
// via htmlFor/id so screen readers announce the field name, not just the
// placeholder.  useId() in InputField generates stable unique IDs per render.

describe("CreateWorkspaceDialog — WCAG SC 1.3.1 label/input association", () => {
  it("Name input has a <label> whose htmlFor matches the input id", async () => {
    await openDialog();
    const nameInput = screen.getByPlaceholderText("e.g. SEO Agent") as HTMLInputElement;
    expect(nameInput.id).toBeTruthy();
    const label = document.querySelector(`label[for="${nameInput.id}"]`);
    expect(label).toBeTruthy();
    expect(label?.textContent).toContain("Name");
  });

  it("Role input has a <label> whose htmlFor matches the input id", async () => {
    await openDialog();
    const roleInput = screen.getByPlaceholderText("e.g. SEO Specialist") as HTMLInputElement;
    expect(roleInput.id).toBeTruthy();
    const label = document.querySelector(`label[for="${roleInput.id}"]`);
    expect(label).toBeTruthy();
    expect(label?.textContent).toContain("Role");
  });

  it("Budget limit input has a <label> whose htmlFor matches the input id", async () => {
    await openDialog();
    const budgetInput = screen.getByPlaceholderText("e.g. 100") as HTMLInputElement;
    expect(budgetInput.id).toBeTruthy();
    const label = document.querySelector(`label[for="${budgetInput.id}"]`);
    expect(label).toBeTruthy();
    expect(label?.textContent).toContain("Budget limit");
  });

  it("Template input has a <label> whose htmlFor matches the input id", async () => {
    await openDialog();
    const templateInput = screen.getByPlaceholderText(
      "e.g. seo-agent (from workspace-configs-templates/)"
    ) as HTMLInputElement;
    expect(templateInput.id).toBeTruthy();
    const label = document.querySelector(`label[for="${templateInput.id}"]`);
    expect(label).toBeTruthy();
    expect(label?.textContent).toContain("Template");
  });

  it("each InputField generates a distinct id (no id collisions)", async () => {
    await openDialog();
    const inputs = [
      screen.getByPlaceholderText("e.g. SEO Agent"),
      screen.getByPlaceholderText("e.g. SEO Specialist"),
      screen.getByPlaceholderText("e.g. 100"),
      screen.getByPlaceholderText("e.g. seo-agent (from workspace-configs-templates/)"),
    ] as HTMLInputElement[];

    const ids = inputs.map((i) => i.id).filter(Boolean);
    const unique = new Set(ids);
    expect(unique.size).toBe(ids.length); // no duplicates
    expect(ids.length).toBe(4);
  });

  it("Name label text contains the required asterisk indicator", async () => {
    await openDialog();
    const nameInput = screen.getByPlaceholderText("e.g. SEO Agent") as HTMLInputElement;
    const label = document.querySelector(`label[for="${nameInput.id}"]`);
    // aria-hidden asterisk * is present for visual required indicator
    expect(label?.querySelector("[aria-hidden='true']")?.textContent).toBe("*");
  });
});
