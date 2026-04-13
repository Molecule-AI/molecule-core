"use client";

import { useState, useEffect } from "react";
import { api } from "@/lib/api";
import { useCanvasStore } from "@/store/canvas";
import { type ActivityEntry } from "@/types/activity";
import { useWorkspaceName } from "@/hooks/useWorkspaceName";

interface Props {
  open: boolean;
  workspaceId: string;
  onClose: () => void;
}

function extractMessageText(body: Record<string, unknown> | null): string {
  if (!body) return "";
  try {
    // Simple task format from MCP server: {task: "..."}
    if (body.task && typeof body.task === "string") return body.task;

    // Request: params.message.parts[].text
    const params = body.params as Record<string, unknown> | undefined;
    const message = params?.message as Record<string, unknown> | undefined;
    const parts = (message?.parts || []) as Array<Record<string, unknown>>;
    const text = parts
      .map((p) => (p.text as string) || "")
      .filter(Boolean)
      .join("\n");
    if (text) return text;

    // Response: result.parts[].text or result.parts[].root.text
    const result = body.result as Record<string, unknown> | undefined;
    const rParts = (result?.parts || []) as Array<Record<string, unknown>>;
    const rText = rParts
      .map((p) => {
        if (p.text) return p.text as string;
        const root = p.root as Record<string, unknown> | undefined;
        return (root?.text as string) || "";
      })
      .filter(Boolean)
      .join("\n");
    if (rText) return rText;

    if (typeof body.result === "string") return body.result;
  } catch { /* ignore */ }
  return "";
}

export function ConversationTraceModal({ open, workspaceId, onClose }: Props) {
  const [entries, setEntries] = useState<ActivityEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const nodes = useCanvasStore((s) => s.nodes);
  const resolveName = useWorkspaceName();

  // Fetch activities from all workspaces (including hidden children) and merge
  useEffect(() => {
    if (!open) return;
    setLoading(true);

    const wsIds = nodes.map((n) => n.id);

    Promise.all(
      wsIds.map((id) =>
        api
          .get<ActivityEntry[]>(`/workspaces/${id}/activity?limit=200`)
          .catch(() => [] as ActivityEntry[])
      )
    ).then((results) => {
      // Merge, deduplicate by ID, sort chronologically (oldest first)
      const seen = new Set<string>();
      const all: ActivityEntry[] = [];
      for (const batch of results) {
        for (const entry of batch) {
          if (!seen.has(entry.id)) {
            seen.add(entry.id);
            all.push(entry);
          }
        }
      }
      all.sort((a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime());
      setEntries(all);
      setLoading(false);
    });
  }, [open, nodes]);

  if (!open) return null;

  const isA2A = (e: ActivityEntry) =>
    e.activity_type === "a2a_receive" || e.activity_type === "a2a_send";

  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center">
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/70 backdrop-blur-sm" onClick={onClose} />

      {/* Modal */}
      <div className="relative bg-zinc-900 border border-zinc-700 rounded-xl shadow-2xl max-w-[700px] w-full mx-4 max-h-[85vh] flex flex-col overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-3 border-b border-zinc-800">
          <div>
            <h3 className="text-sm font-semibold text-zinc-100">
              Conversation Trace
            </h3>
            <p className="text-[10px] text-zinc-500 mt-0.5">
              {entries.length} events across all workspaces
            </p>
          </div>
          <button
            onClick={onClose}
            className="text-zinc-500 hover:text-zinc-300 text-lg px-2"
          >
            ✕
          </button>
        </div>

        {/* Timeline */}
        <div className="flex-1 overflow-y-auto px-5 py-4">
          {loading && (
            <div className="text-xs text-zinc-500 text-center py-8">
              Loading trace from all workspaces...
            </div>
          )}

          {!loading && entries.length === 0 && (
            <div className="text-xs text-zinc-500 text-center py-8">
              No activity found
            </div>
          )}

          <div className="space-y-1">
            {entries.map((entry) => {
              const time = new Date(entry.created_at).toLocaleTimeString();
              const wsName = resolveName(entry.workspace_id);
              const sourceName = resolveName(entry.source_id);
              const targetName = resolveName(entry.target_id);
              const requestText = extractMessageText(entry.request_body);
              const responseText = extractMessageText(entry.response_body);
              const isError = entry.status === "error";
              const isSend = entry.activity_type === "a2a_send";
              const isReceive = entry.activity_type === "a2a_receive";

              return (
                <div key={entry.id} className="group">
                  {/* Event header */}
                  <div className="flex items-start gap-3">
                    {/* Timeline dot + line */}
                    <div className="flex flex-col items-center pt-1.5">
                      <div
                        className={`w-2.5 h-2.5 rounded-full shrink-0 ${
                          isError
                            ? "bg-red-500"
                            : isSend
                            ? "bg-cyan-500"
                            : isReceive
                            ? "bg-blue-500"
                            : "bg-zinc-600"
                        }`}
                      />
                      <div className="w-px flex-1 bg-zinc-800 min-h-[8px]" />
                    </div>

                    {/* Content */}
                    <div className="flex-1 pb-3 min-w-0">
                      <div className="flex items-center gap-2 flex-wrap">
                        <span className="text-[9px] text-zinc-600 font-mono">
                          {time}
                        </span>
                        <span
                          className={`text-[9px] font-semibold px-1.5 py-0.5 rounded ${
                            isError
                              ? "bg-red-950/50 text-red-400"
                              : isSend
                              ? "bg-cyan-950/50 text-cyan-400"
                              : isReceive
                              ? "bg-blue-950/50 text-blue-400"
                              : "bg-zinc-800 text-zinc-400"
                          }`}
                        >
                          {isSend
                            ? "SEND"
                            : isReceive
                            ? "RECEIVE"
                            : entry.activity_type.toUpperCase()}
                        </span>
                        {entry.duration_ms != null && entry.duration_ms > 0 && (
                          <span className="text-[9px] text-zinc-600">
                            {entry.duration_ms > 1000
                              ? `${Math.round(entry.duration_ms / 1000)}s`
                              : `${entry.duration_ms}ms`}
                          </span>
                        )}
                      </div>

                      {/* Flow */}
                      {isA2A(entry) && (
                        <div className="text-[11px] mt-1">
                          {isSend ? (
                            <span>
                              <span className="text-cyan-400 font-medium">
                                {sourceName || wsName}
                              </span>
                              <span className="text-zinc-600"> → </span>
                              <span className="text-blue-400 font-medium">
                                {targetName}
                              </span>
                            </span>
                          ) : (
                            <span>
                              <span className="text-blue-400 font-medium">
                                {targetName || wsName}
                              </span>
                              {sourceName && (
                                <>
                                  <span className="text-zinc-600">
                                    {" "}← {" "}
                                  </span>
                                  <span className="text-cyan-400 font-medium">
                                    {sourceName}
                                  </span>
                                </>
                              )}
                            </span>
                          )}
                        </div>
                      )}

                      {/* Summary */}
                      {entry.summary && !isA2A(entry) && (
                        <div className="text-[10px] text-zinc-400 mt-1">
                          <span className="text-zinc-300 font-medium">{wsName}:</span>{" "}
                          {entry.summary}
                        </div>
                      )}

                      {/* Error */}
                      {isError && entry.error_detail && (
                        <div className="text-[10px] text-red-400/80 mt-1 truncate">
                          {entry.error_detail.slice(0, 200)}
                        </div>
                      )}

                      {/* Message content — show request and/or response */}
                      {requestText && (
                        <div className="mt-1.5 bg-zinc-950/60 border border-zinc-800/50 rounded-lg px-3 py-2 max-h-32 overflow-y-auto">
                          <div className="text-[8px] text-zinc-500 uppercase mb-1">
                            {isSend ? "Task" : "Request"}
                          </div>
                          <div className="text-[10px] text-zinc-300 whitespace-pre-wrap break-words leading-relaxed">
                            {requestText.slice(0, 2000)}
                            {requestText.length > 2000 && (
                              <span className="text-zinc-600"> ...({requestText.length} chars)</span>
                            )}
                          </div>
                        </div>
                      )}
                      {responseText && (
                        <div className="mt-1 bg-zinc-950/60 border border-emerald-900/30 rounded-lg px-3 py-2 max-h-32 overflow-y-auto">
                          <div className="text-[8px] text-emerald-500/60 uppercase mb-1">Response</div>
                          <div className="text-[10px] text-zinc-300 whitespace-pre-wrap break-words leading-relaxed">
                            {responseText.slice(0, 2000)}
                            {responseText.length > 2000 && (
                              <span className="text-zinc-600"> ...({responseText.length} chars)</span>
                            )}
                          </div>
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        </div>

        {/* Footer */}
        <div className="px-5 py-3 border-t border-zinc-800 bg-zinc-950/50 flex justify-end">
          <button
            onClick={onClose}
            className="px-4 py-1.5 text-[12px] bg-zinc-800 hover:bg-zinc-700 text-zinc-300 rounded-lg transition-colors"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
}
