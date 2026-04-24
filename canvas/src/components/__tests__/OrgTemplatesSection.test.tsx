// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor, fireEvent, cleanup } from "@testing-library/react";

// Tests for the default-collapsed + expand-on-click behavior of the
// org templates drawer. Before this change the section rendered all
// org cards inline, which pushed the individual workspace templates
// off-screen when there were ≥3 orgs on disk. Collapsed-by-default
// keeps the scroll focused on the primary deploy path.

vi.mock("@/lib/api", () => ({
  api: {
    get: vi.fn().mockResolvedValue([
      { dir: "free-beats-all", name: "Free Beats All", description: "d1", workspaces: 3 },
      { dir: "medo-smoke", name: "MeDo Smoke Test", description: "d2", workspaces: 1 },
    ]),
    post: vi.fn().mockResolvedValue({}),
  },
}));

vi.mock("../Spinner", () => ({ Spinner: () => null }));
vi.mock("../MissingKeysModal", () => ({ MissingKeysModal: () => null }));
vi.mock("../ConfirmDialog", () => ({ ConfirmDialog: () => null }));
vi.mock("@/lib/deploy-preflight", () => ({ checkDeploySecrets: vi.fn() }));

import { OrgTemplatesSection } from "../TemplatePalette";

beforeEach(() => {
  vi.clearAllMocks();
});

afterEach(() => {
  cleanup();
});

describe("OrgTemplatesSection — collapse/expand", () => {
  it("renders collapsed by default — org cards are NOT in the DOM", async () => {
    render(<OrgTemplatesSection />);
    // The header toggle is visible immediately…
    // Two buttons match "Org Templates" (toggle + refresh) — pick the
    // toggle by its aria-controls binding.
    const toggle = (await screen.findAllByRole("button")).find((b) =>
      b.getAttribute("aria-controls") === "org-templates-body"
    )!;
    expect(toggle).toBeTruthy();
    expect(toggle.getAttribute("aria-expanded")).toBe("false");

    // …and the count appears after loadOrgs resolves.
    await waitFor(() => {
      expect(toggle.textContent).toContain("(2)");
    });

    // But none of the individual org cards should be rendered yet.
    expect(screen.queryByText("Free Beats All")).toBeNull();
    expect(screen.queryByText("MeDo Smoke Test")).toBeNull();
  });

  it("clicking the header reveals the org cards", async () => {
    render(<OrgTemplatesSection />);

    // Wait for the count so we know loadOrgs finished.
    // Two buttons match "Org Templates" (toggle + refresh) — pick the
    // toggle by its aria-controls binding.
    const toggle = (await screen.findAllByRole("button")).find((b) =>
      b.getAttribute("aria-controls") === "org-templates-body"
    )!;
    await waitFor(() => {
      expect(toggle.textContent).toContain("(2)");
    });

    // Expand.
    fireEvent.click(toggle);
    await waitFor(() => {
      expect(toggle.getAttribute("aria-expanded")).toBe("true");
    });

    // Org cards now visible.
    expect(screen.getByText("Free Beats All")).toBeTruthy();
    expect(screen.getByText("MeDo Smoke Test")).toBeTruthy();
  });

  it("clicking the header again collapses back", async () => {
    render(<OrgTemplatesSection />);
    // Two buttons match "Org Templates" (toggle + refresh) — pick the
    // toggle by its aria-controls binding.
    const toggle = (await screen.findAllByRole("button")).find((b) =>
      b.getAttribute("aria-controls") === "org-templates-body"
    )!;
    await waitFor(() => {
      expect(toggle.textContent).toContain("(2)");
    });

    fireEvent.click(toggle); // expand
    expect(screen.getByText("Free Beats All")).toBeTruthy();

    fireEvent.click(toggle); // collapse
    await waitFor(() => {
      expect(toggle.getAttribute("aria-expanded")).toBe("false");
    });
    expect(screen.queryByText("Free Beats All")).toBeNull();
  });
});
