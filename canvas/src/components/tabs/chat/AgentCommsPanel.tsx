"use client";

import { useState, useEffect, useRef } from "react";
import { api } from "@/lib/api";
import { useCanvasStore, type WorkspaceNodeData } from "@/store/canvas";
import { WS_URL } from "@/store/socket";
import { extractResponseText, extractRequestText } from "./message-parser";

interface ActivityEntry {
  id: string;
  activity_type: string;
  source_id: string | null;
  target_id: string | null;
  method: string | null;
  summary: string | null;
  request_body: Record<string, unknown> | null;
  response_body: Record<string, unknown> | null;
  status: string;
  created_at: string;
}

interface CommMessage {
  id: string;
  direction: "in" | "out";
  peerName: string;
  peerId: string;
  text: string;
  responseText: string | null;
  timestamp: string;
}

function resolveName(id: string): string {
  const nodes = useCanvasStore.getState().nodes;
  const node = nodes.find((n) => n.id === id);
  return (node?.data as WorkspaceNodeData)?.name || id.slice(0, 8);
}

function toCommMessage(entry: ActivityEntry, workspaceId: string): CommMessage | null {
  const isOutgoing = entry.activity_type === "a2a_send";
  const peerId = isOutgoing ? (entry.target_id || "") : (entry.source_id || "");
  if (!peerId) return null;

  const text = extractRequestText(entry.request_body) || entry.summary || "";
  const responseText = entry.response_body ? extractResponseText(entry.response_body) : null;

  return {
    id: entry.id,
    direction: isOutgoing ? "out" : "in",
    peerName: resolveName(peerId),
    peerId,
    text,
    responseText,
    timestamp: entry.created_at,
  };
}

export function AgentCommsPanel({ workspaceId }: { workspaceId: string }) {
  const [messages, setMessages] = useState<CommMessage[]>([]);
  const [loading, setLoading] = useState(true);
  // Dedup by timestamp+type+peer to handle API load + WebSocket race
  const seenKeys = useRef(new Set<string>());
  const bottomRef = useRef<HTMLDivElement>(null);

  // Load history
  useEffect(() => {
    setLoading(true);
    api.get<ActivityEntry[]>(`/workspaces/${workspaceId}/activity?source=agent&limit=50`)
      .then((entries) => {
        const filtered = entries
          .filter((e) => e.activity_type === "a2a_send" || e.activity_type === "a2a_receive")
          .reverse();
        const msgs: CommMessage[] = [];
        for (const e of filtered) {
          const m = toCommMessage(e, workspaceId);
          if (m) {
            const key = `${m.timestamp}:${m.direction}:${m.peerId}`;
            msgs.push(m);
            seenKeys.current.add(key);
          }
        }
        setMessages(msgs);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, [workspaceId]);

  // Live updates via WebSocket
  useEffect(() => {
    const ws = new WebSocket(WS_URL);
    ws.onerror = () => {
      console.warn("AgentCommsPanel WS error");
    };
    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        if (msg.event === "ACTIVITY_LOGGED" && msg.workspace_id === workspaceId) {
          const p = msg.payload || {};
          const type = p.activity_type as string;
          const sourceId = p.source_id as string | null;
          if (!sourceId) return; // canvas-initiated, not agent comms
          if (type !== "a2a_send" && type !== "a2a_receive") return;

          const entry: ActivityEntry = {
            id: p.id as string || crypto.randomUUID(),
            activity_type: type,
            source_id: sourceId,
            target_id: p.target_id as string | null,
            method: p.method as string | null,
            summary: p.summary as string | null,
            request_body: p.request_body as Record<string, unknown> | null,
            response_body: p.response_body as Record<string, unknown> | null,
            status: p.status as string || "ok",
            created_at: msg.timestamp || new Date().toISOString(),
          };
          const m = toCommMessage(entry, workspaceId);
          if (m) {
            const key = `${m.timestamp}:${m.direction}:${m.peerId}`;
            if (seenKeys.current.has(key)) return;
            seenKeys.current.add(key);
            setMessages((prev) => [...prev, m]);
          }
        }
      } catch { /* ignore */ }
    };
    return () => ws.close();
  }, [workspaceId]);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  if (loading) {
    return <div className="text-xs text-zinc-500 text-center py-8">Loading agent communications...</div>;
  }

  if (messages.length === 0) {
    return (
      <div className="text-xs text-zinc-500 text-center py-8">
        No agent-to-agent communications yet.
        <br />
        <span className="text-zinc-600">Delegations and peer messages will appear here.</span>
      </div>
    );
  }

  return (
    <div className="flex-1 overflow-y-auto p-3 space-y-2">
      {messages.map((msg) => (
        <div key={msg.id} className={`flex ${msg.direction === "out" ? "justify-end" : "justify-start"}`}>
          <div
            className={`max-w-[85%] rounded-lg px-3 py-2 text-xs ${
              msg.direction === "out"
                ? "bg-cyan-900/30 text-cyan-100 border border-cyan-700/20"
                : "bg-zinc-800/80 text-zinc-200 border border-zinc-700/30"
            }`}
          >
            <div className="text-[9px] text-zinc-500 mb-1">
              {msg.direction === "out" ? `→ To ${msg.peerName}` : `← From ${msg.peerName}`}
            </div>
            <div className="text-zinc-300">{msg.text || "(no message text)"}</div>
            {msg.responseText && (
              <div className="mt-1.5 pt-1.5 border-t border-zinc-700/30 text-zinc-400">
                {msg.responseText}
              </div>
            )}
            <div className="text-[9px] text-zinc-600 mt-1">
              {new Date(msg.timestamp).toLocaleTimeString()}
            </div>
          </div>
        </div>
      ))}
      <div ref={bottomRef} />
    </div>
  );
}
