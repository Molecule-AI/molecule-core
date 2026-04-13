export function extractAgentText(task: Record<string, unknown>): string {
  try {
    const directTexts = extractTextsFromParts(task.parts);
    if (directTexts) return directTexts;

    const artifacts = task.artifacts as Array<Record<string, unknown>> | undefined;
    if (artifacts && artifacts.length > 0) {
      const texts = extractTextsFromParts(artifacts[0].parts);
      if (texts) return texts;
    }

    const status = task.status as Record<string, unknown> | undefined;
    if (status?.message) {
      const msg = status.message as Record<string, unknown>;
      const texts = extractTextsFromParts(msg.parts);
      if (texts) return texts;
    }

    if (typeof task === "string") return task;
    return "(Could not extract response text)";
  } catch {
    return "(Failed to parse response)";
  }
}

export function extractTextsFromParts(parts: unknown): string | null {
  if (!Array.isArray(parts)) return null;
  const texts = parts
    .filter((p: Record<string, unknown>) => p.type === "text" || p.kind === "text")
    .map((p: Record<string, unknown>) => String(p.text || ""))
    .filter(Boolean);
  return texts.length > 0 ? texts.join("\n") : null;
}

/** Extract user message text from an activity log request_body */
export function extractRequestText(body: Record<string, unknown> | null): string {
  if (!body) return "";
  const params = body.params as Record<string, unknown> | undefined;
  const msg = params?.message as Record<string, unknown> | undefined;
  const parts = msg?.parts as Array<Record<string, unknown>> | undefined;
  return (parts?.[0]?.text as string) || "";
}

/** Extract text from an activity log response_body (multiple possible formats) */
export function extractResponseText(body: Record<string, unknown>): string {
  try {
    // {result: "text"} — from MCP server delegation logs
    if (typeof body.result === "string") return body.result;

    // A2A JSON-RPC response: {result: {parts: [{kind: "text", text: "..."}]}}
    const result = body.result as Record<string, unknown> | undefined;
    if (result) {
      const parts = (result.parts || []) as Array<Record<string, unknown>>;
      for (const p of parts) {
        const t = (p.text as string) || "";
        if (t) return t;
        const root = p.root as Record<string, unknown> | undefined;
        if (root?.text) return root.text as string;
      }
    }

    // {task: "text"} — request body format, shouldn't be in response but handle it
    if (typeof body.task === "string") return body.task;
  } catch { /* ignore */ }
  return "";
}
