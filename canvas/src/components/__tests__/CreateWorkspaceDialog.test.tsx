// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor, cleanup } from "@testing-library/react";
import { CreateWorkspaceButton, HERMES_PROVIDERS } from "../CreateWorkspaceDialog";

vi.mock("@/lib/api", () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
  },
}));

import { api } from "@/lib/api";

const mockGet = vi.mocked(api.get);
const mockPost = vi.mocked(api.post);

const SAMPLE_WORKSPACES = [
  { id: "ws-1", name: "Platform Team", tier: 1 },
  { id: "ws-2", name: "Research Agent", tier: 2 },
];

beforeEach(() => {
  vi.clearAllMocks();
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  mockGet.mockResolvedValue(SAMPLE_WORKSPACES as any);
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  mockPost.mockResolvedValue({} as any);
});

afterEach(() => {
  cleanup();
});

async function openDialog() {
  render(<CreateWorkspaceButton />);
  const btn = screen.getAllByRole("button").find((b) => b.textContent?.includes("New Workspace"));
  expect(btn).toBeTruthy();
  fireEvent.click(btn!);
  await waitFor(() => expect(screen.getByText("Create Workspace")).toBeTruthy());
}

async function setTemplate(value: string) {
  fireEvent.change(
    screen.getByPlaceholderText("e.g. seo-agent (from workspace-configs-templates/)"),
    { target: { value } }
  );
}

describe("CreateWorkspaceDialog", () => {
  it("opens the dialog when New Workspace button is clicked", async () => {
    await openDialog();
    expect(screen.getByText("Create Workspace")).toBeTruthy();
  });

  it("renders a <select> for parent workspace — not a text input", async () => {
    await openDialog();
    const selects = document.querySelectorAll("select");
    expect(selects.length).toBeGreaterThanOrEqual(1);
    // The old raw UUID text input is gone
    expect(screen.queryByPlaceholderText("Leave empty for root-level")).toBeNull();
  });

  it('first option is "None (root level)" with empty value', async () => {
    await openDialog();
    const select = document.querySelector("select") as HTMLSelectElement;
    expect(select).toBeTruthy();
    const firstOption = select.options[0];
    expect(firstOption.value).toBe("");
    expect(firstOption.text.trim()).toBe("None (root level)");
  });

  it("populates select with workspace names from GET /workspaces", async () => {
    await openDialog();
    await waitFor(() => {
      const select = document.querySelector("select") as HTMLSelectElement;
      const optionValues = Array.from(select.options).map((o) => o.value);
      expect(optionValues).toContain("ws-1");
      expect(optionValues).toContain("ws-2");
    });
    const select = document.querySelector("select") as HTMLSelectElement;
    const optionTexts = Array.from(select.options).map((o) => o.text.trim());
    expect(optionTexts.some((t) => t.includes("Platform Team"))).toBe(true);
    expect(optionTexts.some((t) => t.includes("Research Agent"))).toBe(true);
  });

  it("sends parent_id in POST body when a workspace is selected", async () => {
    await openDialog();
    await waitFor(() => {
      const select = document.querySelector("select") as HTMLSelectElement;
      expect(select.options.length).toBeGreaterThan(1);
    });

    fireEvent.change(screen.getByPlaceholderText("e.g. SEO Agent"), {
      target: { value: "My Agent" },
    });

    const select = document.querySelector("select") as HTMLSelectElement;
    fireEvent.change(select, { target: { value: "ws-1" } });

    const createBtn = screen.getAllByRole("button").find((b) => b.textContent === "Create");
    fireEvent.click(createBtn!);

    await waitFor(() => expect(mockPost).toHaveBeenCalled());
    const body = mockPost.mock.calls[0][1] as Record<string, unknown>;
    expect(body.parent_id).toBe("ws-1");
  });

  it("sends parent_id as undefined when None (root level) is selected", async () => {
    await openDialog();
    fireEvent.change(screen.getByPlaceholderText("e.g. SEO Agent"), {
      target: { value: "Root Agent" },
    });

    const select = document.querySelector("select") as HTMLSelectElement;
    fireEvent.change(select, { target: { value: "" } });

    const createBtn = screen.getAllByRole("button").find((b) => b.textContent === "Create");
    fireEvent.click(createBtn!);

    await waitFor(() => expect(mockPost).toHaveBeenCalled());
    const body = mockPost.mock.calls[0][1] as Record<string, unknown>;
    expect(body.parent_id).toBeUndefined();
  });

  it("renders gracefully when GET /workspaces fails", async () => {
    mockGet.mockRejectedValueOnce(new Error("Network error"));
    await openDialog();

    // Dialog still renders; select exists with only the root option
    await waitFor(() => {
      const select = document.querySelector("select") as HTMLSelectElement;
      expect(select.options.length).toBe(1);
      expect(select.options[0].value).toBe("");
    });
  });
});

// ---------------------------------------------------------------------------
// Hermes provider picker tests
// ---------------------------------------------------------------------------

describe("CreateWorkspaceDialog — Hermes provider picker", () => {
  it("does NOT show hermes provider section for non-hermes templates", async () => {
    await openDialog();
    await setTemplate("seo-agent");
    expect(document.querySelector("[data-testid='hermes-provider-section']")).toBeNull();
  });

  it("shows hermes provider section when template is 'hermes'", async () => {
    await openDialog();
    await setTemplate("hermes");
    await waitFor(() =>
      expect(document.querySelector("[data-testid='hermes-provider-section']")).toBeTruthy()
    );
  });

  it("shows hermes provider section for template 'HERMES' (case-insensitive)", async () => {
    await openDialog();
    await setTemplate("HERMES");
    await waitFor(() =>
      expect(document.querySelector("[data-testid='hermes-provider-section']")).toBeTruthy()
    );
  });

  it("hermes provider dropdown defaults to 'anthropic'", async () => {
    await openDialog();
    await setTemplate("hermes");
    await waitFor(() =>
      expect(document.querySelector("[data-testid='hermes-provider-section']")).toBeTruthy()
    );
    const providerSelect = document.getElementById("hermes-provider-select") as HTMLSelectElement;
    expect(providerSelect).toBeTruthy();
    expect(providerSelect.value).toBe("anthropic");
  });

  it("hermes provider dropdown lists all 15 providers", async () => {
    await openDialog();
    await setTemplate("hermes");
    await waitFor(() =>
      expect(document.querySelector("[data-testid='hermes-provider-section']")).toBeTruthy()
    );
    const providerSelect = document.getElementById("hermes-provider-select") as HTMLSelectElement;
    expect(providerSelect.options.length).toBe(HERMES_PROVIDERS.length);
    const ids = Array.from(providerSelect.options).map((o) => o.value);
    expect(ids).toContain("anthropic");
    expect(ids).toContain("openai");
    expect(ids).toContain("gemini");
    expect(ids).toContain("deepseek");
    expect(ids).toContain("hermes");
  });

  it("hermes API key field is a password input (masked)", async () => {
    await openDialog();
    await setTemplate("hermes");
    await waitFor(() =>
      expect(document.querySelector("[data-testid='hermes-provider-section']")).toBeTruthy()
    );
    const keyInput = document.getElementById("hermes-api-key-input") as HTMLInputElement;
    expect(keyInput).toBeTruthy();
    expect(keyInput.type).toBe("password");
  });

  it("shows an error if hermes template is set but API key is empty on submit", async () => {
    await openDialog();
    fireEvent.change(screen.getByPlaceholderText("e.g. SEO Agent"), {
      target: { value: "Hermes Agent" },
    });
    await setTemplate("hermes");
    await waitFor(() =>
      expect(document.querySelector("[data-testid='hermes-provider-section']")).toBeTruthy()
    );

    // Submit without API key
    const createBtn = screen.getAllByRole("button").find((b) => b.textContent === "Create");
    fireEvent.click(createBtn!);

    await waitFor(() => {
      const alert = screen.getByRole("alert");
      expect(alert.textContent).toContain("API key");
    });
    expect(mockPost).not.toHaveBeenCalled();
  });

  it("includes secrets in POST body with correct env var for selected provider", async () => {
    await openDialog();
    fireEvent.change(screen.getByPlaceholderText("e.g. SEO Agent"), {
      target: { value: "Hermes Agent" },
    });
    await setTemplate("hermes");
    await waitFor(() =>
      expect(document.querySelector("[data-testid='hermes-provider-section']")).toBeTruthy()
    );

    // Fill in the API key
    const keyInput = document.getElementById("hermes-api-key-input") as HTMLInputElement;
    fireEvent.change(keyInput, { target: { value: "sk-test-anthropic-key" } });

    const createBtn = screen.getAllByRole("button").find((b) => b.textContent === "Create");
    fireEvent.click(createBtn!);

    await waitFor(() => expect(mockPost).toHaveBeenCalled());
    const body = mockPost.mock.calls[0][1] as Record<string, unknown>;
    expect(body.secrets).toEqual({ ANTHROPIC_API_KEY: "sk-test-anthropic-key" });
    expect(body.template).toBe("hermes");
  });

  it("uses the correct env var when a non-default provider is selected", async () => {
    await openDialog();
    fireEvent.change(screen.getByPlaceholderText("e.g. SEO Agent"), {
      target: { value: "Hermes OpenAI" },
    });
    await setTemplate("hermes");
    await waitFor(() =>
      expect(document.querySelector("[data-testid='hermes-provider-section']")).toBeTruthy()
    );

    // Switch to openai
    const providerSelect = document.getElementById("hermes-provider-select") as HTMLSelectElement;
    fireEvent.change(providerSelect, { target: { value: "openai" } });

    const keyInput = document.getElementById("hermes-api-key-input") as HTMLInputElement;
    fireEvent.change(keyInput, { target: { value: "sk-openai-test" } });

    const createBtn = screen.getAllByRole("button").find((b) => b.textContent === "Create");
    fireEvent.click(createBtn!);

    await waitFor(() => expect(mockPost).toHaveBeenCalled());
    const body = mockPost.mock.calls[0][1] as Record<string, unknown>;
    expect(body.secrets).toEqual({ OPENAI_API_KEY: "sk-openai-test" });
  });

  it("does NOT include secrets field when template is not hermes", async () => {
    await openDialog();
    fireEvent.change(screen.getByPlaceholderText("e.g. SEO Agent"), {
      target: { value: "Normal Agent" },
    });
    await setTemplate("seo-agent");

    const createBtn = screen.getAllByRole("button").find((b) => b.textContent === "Create");
    fireEvent.click(createBtn!);

    await waitFor(() => expect(mockPost).toHaveBeenCalled());
    const body = mockPost.mock.calls[0][1] as Record<string, unknown>;
    expect(body.secrets).toBeUndefined();
  });

  it("hides hermes section and resets state when template is cleared", async () => {
    await openDialog();
    await setTemplate("hermes");
    await waitFor(() =>
      expect(document.querySelector("[data-testid='hermes-provider-section']")).toBeTruthy()
    );

    // Clear template
    await setTemplate("");
    await waitFor(() =>
      expect(document.querySelector("[data-testid='hermes-provider-section']")).toBeNull()
    );
  });
});
