import { describe, it, expect } from "vitest";
import {
  extractRequestText,
  extractResponseText,
  extractAgentText,
  extractTextsFromParts,
} from "../message-parser";

describe("extractRequestText", () => {
  it("extracts text from standard A2A request_body", () => {
    const body = {
      params: {
        message: {
          role: "user",
          parts: [{ kind: "text", text: "Hello agent" }],
        },
      },
    };
    expect(extractRequestText(body)).toBe("Hello agent");
  });

  it("returns empty string for null body", () => {
    expect(extractRequestText(null)).toBe("");
  });

  it("returns empty string for empty object", () => {
    expect(extractRequestText({})).toBe("");
  });

  it("returns empty string when params missing", () => {
    expect(extractRequestText({ other: "data" })).toBe("");
  });

  it("returns empty string when message missing", () => {
    expect(extractRequestText({ params: {} })).toBe("");
  });

  it("returns empty string when parts empty", () => {
    expect(extractRequestText({ params: { message: { parts: [] } } })).toBe("");
  });

  it("extracts first part text only", () => {
    const body = {
      params: {
        message: {
          parts: [
            { kind: "text", text: "First" },
            { kind: "text", text: "Second" },
          ],
        },
      },
    };
    expect(extractRequestText(body)).toBe("First");
  });

  it("handles non-text parts gracefully", () => {
    const body = {
      params: {
        message: {
          parts: [{ kind: "image", data: "base64..." }],
        },
      },
    };
    expect(extractRequestText(body)).toBe("");
  });
});

describe("extractResponseText", () => {
  it("extracts from result string", () => {
    expect(extractResponseText({ result: "Hello!" })).toBe("Hello!");
  });

  it("extracts from result.parts[].text", () => {
    const body = {
      result: {
        parts: [{ kind: "text", text: "Response text" }],
      },
    };
    expect(extractResponseText(body)).toBe("Response text");
  });

  it("extracts from result.parts[].root.text", () => {
    const body = {
      result: {
        parts: [{ root: { text: "Root text" } }],
      },
    };
    expect(extractResponseText(body)).toBe("Root text");
  });

  it("extracts from task field", () => {
    expect(extractResponseText({ task: "Task text" })).toBe("Task text");
  });

  it("returns empty for empty object", () => {
    expect(extractResponseText({})).toBe("");
  });

  it("returns empty when result has no parts", () => {
    expect(extractResponseText({ result: { other: true } })).toBe("");
  });
});

describe("extractTextsFromParts", () => {
  it("extracts text parts with kind=text", () => {
    const parts = [
      { kind: "text", text: "Hello" },
      { kind: "text", text: "World" },
    ];
    expect(extractTextsFromParts(parts)).toBe("Hello\nWorld");
  });

  it("extracts text parts with type=text", () => {
    const parts = [{ type: "text", text: "Legacy format" }];
    expect(extractTextsFromParts(parts)).toBe("Legacy format");
  });

  it("returns null for non-array", () => {
    expect(extractTextsFromParts(null)).toBeNull();
    expect(extractTextsFromParts(undefined)).toBeNull();
    expect(extractTextsFromParts("string")).toBeNull();
  });

  it("returns null for empty array", () => {
    expect(extractTextsFromParts([])).toBeNull();
  });

  it("filters out non-text parts", () => {
    const parts = [
      { kind: "image", data: "..." },
      { kind: "text", text: "Only text" },
    ];
    expect(extractTextsFromParts(parts)).toBe("Only text");
  });
});
