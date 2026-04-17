// @vitest-environment jsdom
/**
 * Tests for BudgetSection (issue #541).
 *
 * Covers:
 *  - Loading state
 *  - Stats row: used / limit, "Unlimited" when null
 *  - Progress bar: correct percentage, capped at 100%, absent when no limit
 *  - Budget remaining text
 *  - Input pre-fill (existing limit / blank when null)
 *  - Save: PATCH with number, PATCH with null (blank input)
 *  - 402 on GET → exceeded banner, no fetch-error text
 *  - 402 on PATCH → exceeded banner
 *  - Non-402 fetch error → error text
 *  - Non-402 save error → save error alert
 *  - Section header and subheading
 *  - Fetch error does not show stats
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import {
  render,
  screen,
  fireEvent,
  waitFor,
  cleanup,
  act,
} from "@testing-library/react";

// ── Mock api ──────────────────────────────────────────────────────────────────

vi.mock("@/lib/api", () => ({
  api: {
    get: vi.fn(),
    patch: vi.fn(),
  },
}));

import { api } from "@/lib/api";
import { BudgetSection } from "../tabs/BudgetSection";

const mockGet = vi.mocked(api.get);
const mockPatch = vi.mocked(api.patch);

// ── Helpers ───────────────────────────────────────────────────────────────────

function budgetResponse(overrides: Partial<{
  budget_limit: number | null;
  budget_used: number;
  budget_remaining: number | null;
}> = {}) {
  return {
    budget_limit: 1000,
    budget_used: 250,
    budget_remaining: 750,
    ...overrides,
  };
}

function make402Error(): Error {
  return new Error("API GET /workspaces/ws-1/budget: 402 Payment Required");
}

function make402PatchError(): Error {
  return new Error("API PATCH /workspaces/ws-1/budget: 402 Payment Required");
}

function makeGenericError(msg = "network timeout"): Error {
  return new Error(`API GET /workspaces/ws-1/budget: 500 ${msg}`);
}

beforeEach(() => {
  vi.clearAllMocks();
});

afterEach(() => {
  cleanup();
});

// ── Rendering helpers ─────────────────────────────────────────────────────────

async function renderLoaded(budgetData = budgetResponse()) {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  mockGet.mockResolvedValueOnce(budgetData as any);
  render(<BudgetSection workspaceId="ws-1" />);
  // Wait for loading to finish
  await waitFor(() => expect(screen.queryByTestId("budget-loading")).toBeNull());
}

// ── Loading state ─────────────────────────────────────────────────────────────

describe("BudgetSection — loading state", () => {
  it("shows loading indicator while fetch is in flight", () => {
    // Never resolve
    mockGet.mockReturnValue(new Promise(() => {}));
    render(<BudgetSection workspaceId="ws-1" />);
    expect(screen.getByTestId("budget-loading")).toBeTruthy();
    expect(screen.getByText("Loading…")).toBeTruthy();
  });

  it("hides loading indicator after fetch resolves", async () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockGet.mockResolvedValueOnce(budgetResponse() as any);
    render(<BudgetSection workspaceId="ws-1" />);
    await waitFor(() => expect(screen.queryByTestId("budget-loading")).toBeNull());
  });
});

// ── Section header ────────────────────────────────────────────────────────────

describe("BudgetSection — header and subheading", () => {
  it("renders 'Budget' as the section heading", async () => {
    await renderLoaded();
    expect(screen.getByText("Budget")).toBeTruthy();
  });

  it("renders the subheading 'Limit total message credits for this workspace'", async () => {
    await renderLoaded();
    expect(
      screen.getByText("Limit total message credits for this workspace")
    ).toBeTruthy();
  });

  it("renders 'Budget limit (credits)' label for the input", async () => {
    await renderLoaded();
    expect(screen.getByText("Budget limit (credits)")).toBeTruthy();
  });
});

// ── Stats row ─────────────────────────────────────────────────────────────────

describe("BudgetSection — stats row", () => {
  it("shows budget_used in the stats row", async () => {
    await renderLoaded(budgetResponse({ budget_used: 350, budget_limit: 1000 }));
    expect(screen.getByTestId("budget-used-value").textContent).toBe("350");
  });

  it("shows budget_limit in the stats row", async () => {
    await renderLoaded(budgetResponse({ budget_used: 100, budget_limit: 500 }));
    expect(screen.getByTestId("budget-limit-value").textContent).toBe("500");
  });

  it("shows 'Unlimited' when budget_limit is null", async () => {
    await renderLoaded(budgetResponse({ budget_limit: null, budget_remaining: null }));
    expect(screen.getByTestId("budget-limit-value").textContent).toBe("Unlimited");
  });

  it("shows budget_remaining when present", async () => {
    await renderLoaded(budgetResponse({ budget_remaining: 750 }));
    expect(screen.getByTestId("budget-remaining").textContent).toContain("750");
    expect(screen.getByTestId("budget-remaining").textContent).toContain("credits remaining");
  });

  it("hides budget_remaining row when null", async () => {
    await renderLoaded(budgetResponse({ budget_remaining: null }));
    expect(screen.queryByTestId("budget-remaining")).toBeNull();
  });
});

// ── Progress bar ──────────────────────────────────────────────────────────────

describe("BudgetSection — progress bar", () => {
  it("renders the progress bar when budget_limit is set", async () => {
    await renderLoaded(budgetResponse({ budget_used: 250, budget_limit: 1000 }));
    expect(screen.getByRole("progressbar")).toBeTruthy();
  });

  it("does NOT render progress bar when budget_limit is null", async () => {
    await renderLoaded(budgetResponse({ budget_limit: null, budget_remaining: null }));
    expect(screen.queryByRole("progressbar")).toBeNull();
  });

  it("fills to the correct percentage (25%)", async () => {
    await renderLoaded(budgetResponse({ budget_used: 250, budget_limit: 1000 }));
    const fill = screen.getByTestId("budget-progress-fill") as HTMLDivElement;
    expect(fill.style.width).toBe("25%");
  });

  it("fills to the correct percentage (50%)", async () => {
    await renderLoaded(budgetResponse({ budget_used: 500, budget_limit: 1000 }));
    const fill = screen.getByTestId("budget-progress-fill") as HTMLDivElement;
    expect(fill.style.width).toBe("50%");
  });

  it("caps fill at 100% when budget_used exceeds budget_limit", async () => {
    await renderLoaded(budgetResponse({ budget_used: 1500, budget_limit: 1000 }));
    const fill = screen.getByTestId("budget-progress-fill") as HTMLDivElement;
    expect(fill.style.width).toBe("100%");
  });

  it("progress bar has aria-valuenow equal to the calculated percentage", async () => {
    await renderLoaded(budgetResponse({ budget_used: 300, budget_limit: 1000 }));
    const bar = screen.getByRole("progressbar");
    expect(bar.getAttribute("aria-valuenow")).toBe("30");
  });
});

// ── Input pre-fill ────────────────────────────────────────────────────────────

describe("BudgetSection — input pre-fill", () => {
  it("pre-fills input with existing budget_limit", async () => {
    await renderLoaded(budgetResponse({ budget_limit: 500 }));
    const input = screen.getByTestId("budget-limit-input") as HTMLInputElement;
    expect(input.value).toBe("500");
  });

  it("leaves input empty when budget_limit is null", async () => {
    await renderLoaded(budgetResponse({ budget_limit: null, budget_remaining: null }));
    const input = screen.getByTestId("budget-limit-input") as HTMLInputElement;
    expect(input.value).toBe("");
  });
});

// ── Save — PATCH calls ────────────────────────────────────────────────────────

describe("BudgetSection — save", () => {
  it("calls PATCH /workspaces/:id/budget with budget_limit as integer", async () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockPatch.mockResolvedValueOnce(budgetResponse({ budget_limit: 800 }) as any);
    await renderLoaded(budgetResponse({ budget_limit: 1000 }));

    fireEvent.change(screen.getByTestId("budget-limit-input"), {
      target: { value: "800" },
    });
    fireEvent.click(screen.getByTestId("budget-save-btn"));

    await waitFor(() => expect(mockPatch).toHaveBeenCalled());
    expect(mockPatch.mock.calls[0][0]).toBe("/workspaces/ws-1/budget");
    const body = mockPatch.mock.calls[0][1] as Record<string, unknown>;
    expect(body.budget_limit).toBe(800);
  });

  it("sends budget_limit: 0 (not null) when input is '0' — zero-credit budget", async () => {
    // Regression for QA bug report: `parseInt("0") || null` would yield null.
    // The correct form `raw !== "" ? parseInt(raw, 10) : null` must return 0.
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockPatch.mockResolvedValueOnce(budgetResponse({ budget_limit: 0, budget_used: 0, budget_remaining: 0 }) as any);
    await renderLoaded(budgetResponse({ budget_limit: 1000 }));

    fireEvent.change(screen.getByTestId("budget-limit-input"), {
      target: { value: "0" },
    });
    fireEvent.click(screen.getByTestId("budget-save-btn"));

    await waitFor(() => expect(mockPatch).toHaveBeenCalled());
    const body = mockPatch.mock.calls[0][1] as Record<string, unknown>;
    expect(body.budget_limit).toBe(0);
    expect(body.budget_limit).not.toBeNull();
  });

  it("sends budget_limit: null when input is blank", async () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockPatch.mockResolvedValueOnce(budgetResponse({ budget_limit: null, budget_remaining: null }) as any);
    await renderLoaded(budgetResponse({ budget_limit: 1000 }));

    fireEvent.change(screen.getByTestId("budget-limit-input"), {
      target: { value: "" },
    });
    fireEvent.click(screen.getByTestId("budget-save-btn"));

    await waitFor(() => expect(mockPatch).toHaveBeenCalled());
    const body = mockPatch.mock.calls[0][1] as Record<string, unknown>;
    expect(body.budget_limit).toBeNull();
  });

  it("updates displayed stats after successful save", async () => {
    const updated = budgetResponse({ budget_limit: 2000, budget_used: 500, budget_remaining: 1500 });
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockPatch.mockResolvedValueOnce(updated as any);
    await renderLoaded(budgetResponse({ budget_limit: 1000, budget_used: 250 }));

    fireEvent.change(screen.getByTestId("budget-limit-input"), {
      target: { value: "2000" },
    });
    fireEvent.click(screen.getByTestId("budget-save-btn"));

    await waitFor(() =>
      expect(screen.getByTestId("budget-limit-value").textContent).toBe("2,000")
    );
  });

  it("shows save error message on non-402 PATCH failure", async () => {
    mockPatch.mockRejectedValueOnce(
      new Error("API PATCH /workspaces/ws-1/budget: 500 server error")
    );
    await renderLoaded();

    fireEvent.click(screen.getByTestId("budget-save-btn"));

    await waitFor(() =>
      expect(screen.getByTestId("budget-save-error")).toBeTruthy()
    );
    expect(screen.getByTestId("budget-save-error").textContent).toContain("500");
  });
});

// ── 402 handling ──────────────────────────────────────────────────────────────

describe("BudgetSection — 402 handling", () => {
  it("shows exceeded banner when GET returns 402", async () => {
    mockGet.mockRejectedValueOnce(make402Error());
    render(<BudgetSection workspaceId="ws-1" />);

    await waitFor(() =>
      expect(screen.getByTestId("budget-exceeded-banner")).toBeTruthy()
    );
    expect(screen.getByText("Budget exceeded — messages blocked")).toBeTruthy();
  });

  it("does NOT show fetch error text when GET returns 402 (only banner)", async () => {
    mockGet.mockRejectedValueOnce(make402Error());
    render(<BudgetSection workspaceId="ws-1" />);

    await waitFor(() =>
      expect(screen.queryByTestId("budget-loading")).toBeNull()
    );
    expect(screen.queryByTestId("budget-fetch-error")).toBeNull();
    expect(screen.getByTestId("budget-exceeded-banner")).toBeTruthy();
  });

  it("shows exceeded banner when PATCH returns 402", async () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockGet.mockResolvedValueOnce(budgetResponse() as any);
    mockPatch.mockRejectedValueOnce(make402PatchError());
    render(<BudgetSection workspaceId="ws-1" />);
    await waitFor(() => expect(screen.queryByTestId("budget-loading")).toBeNull());

    fireEvent.click(screen.getByTestId("budget-save-btn"));

    await waitFor(() =>
      expect(screen.getByTestId("budget-exceeded-banner")).toBeTruthy()
    );
    // Should NOT also show the save-error alert
    expect(screen.queryByTestId("budget-save-error")).toBeNull();
  });

  it("clears exceeded banner after a successful save", async () => {
    mockGet.mockRejectedValueOnce(make402Error());
    render(<BudgetSection workspaceId="ws-1" />);
    await waitFor(() =>
      expect(screen.getByTestId("budget-exceeded-banner")).toBeTruthy()
    );

    // Now a successful PATCH (limit was raised)
    const updated = budgetResponse({ budget_limit: 5000, budget_used: 250, budget_remaining: 4750 });
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockPatch.mockResolvedValueOnce(updated as any);

    await act(async () => {
      fireEvent.change(screen.getByTestId("budget-limit-input"), {
        target: { value: "5000" },
      });
      fireEvent.click(screen.getByTestId("budget-save-btn"));
    });

    await waitFor(() =>
      expect(screen.queryByTestId("budget-exceeded-banner")).toBeNull()
    );
  });
});

// ── Non-402 fetch error ───────────────────────────────────────────────────────

describe("BudgetSection — non-402 fetch errors", () => {
  it("shows fetch error text on non-402 GET failure", async () => {
    mockGet.mockRejectedValueOnce(makeGenericError("internal server error"));
    render(<BudgetSection workspaceId="ws-1" />);

    await waitFor(() =>
      expect(screen.getByTestId("budget-fetch-error")).toBeTruthy()
    );
    expect(screen.getByTestId("budget-fetch-error").textContent).toContain("500");
  });

  it("does NOT show stats row on fetch error", async () => {
    mockGet.mockRejectedValueOnce(makeGenericError());
    render(<BudgetSection workspaceId="ws-1" />);

    await waitFor(() => expect(screen.queryByTestId("budget-loading")).toBeNull());
    expect(screen.queryByTestId("budget-stats-row")).toBeNull();
  });

  it("does NOT show exceeded banner on non-402 fetch error", async () => {
    mockGet.mockRejectedValueOnce(makeGenericError());
    render(<BudgetSection workspaceId="ws-1" />);

    await waitFor(() => expect(screen.queryByTestId("budget-loading")).toBeNull());
    expect(screen.queryByTestId("budget-exceeded-banner")).toBeNull();
  });
});
