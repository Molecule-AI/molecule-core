"use client";

import { useState, useRef, useEffect, useCallback } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { api } from "@/lib/api";
import { useCanvasStore, type WorkspaceNodeData } from "@/store/canvas";
import { WS_URL } from "@/store/socket";
import { type ChatMessage, createMessage } from "./chat/types";
import { extractResponseText, extractRequestText } from "./chat/message-parser";
import { AgentCommsPanel } from "./chat/AgentCommsPanel";
import { runtimeDisplayName } from "@/lib/runtime-names";

interface Props {
  workspaceId: string;
  data: WorkspaceNodeData;
}

type ChatSubTab = "my-chat" | "agent-comms";

// A2A response shape (subset). The full schema is in @a2a-js/sdk but we only
// need parts/artifacts text extraction for the synchronous fallback path.
interface A2APart {
  kind: string;
  text: string;
}
interface A2AResponse {
  result?: {
    parts?: A2APart[];
    artifacts?: Array<{ parts: A2APart[] }>;
  };
}

// extractReplyText pulls the agent's text reply out of an A2A response.
// Mirrors the Go-side extractReplyText in platform/internal/channels/manager.go.
function extractReplyText(resp: A2AResponse): string {
  const result = resp?.result;
  if (result?.parts) {
    for (const p of result.parts) {
      if (p.kind === "text") return p.text;
    }
  }
  if (result?.artifacts) {
    for (const a of result.artifacts) {
      for (const p of a.parts || []) {
        if (p.kind === "text") return p.text;
      }
    }
  }
  return "";
}

/**
 * Load chat history from the activity_logs database via the platform API.
 * Uses source=canvas to only get user-initiated messages (not agent-to-agent).
 */
async function loadMessagesFromDB(workspaceId: string): Promise<ChatMessage[]> {
  try {
    const activities = await api.get<Array<{
      activity_type: string;
      status: string;
      created_at: string;
      request_body: Record<string, unknown> | null;
      response_body: Record<string, unknown> | null;
    }>>(`/workspaces/${workspaceId}/activity?type=a2a_receive&source=canvas&limit=50`);

    const messages: ChatMessage[] = [];
    // Activities are newest-first, reverse for chronological order
    for (const a of [...activities].reverse()) {
      // Extract user message from request_body
      const userText = extractRequestText(a.request_body);
      if (userText) {
        messages.push(createMessage("user", userText));
      }

      // Extract agent response
      if (a.response_body) {
        const text = extractResponseText(a.response_body);
        if (text) {
          const role = a.status === "error" || text.toLowerCase().startsWith("agent error") ? "system" : "agent";
          messages.push({ ...createMessage(role, text), timestamp: a.created_at });
        }
      }
    }
    return messages;
  } catch {
    return [];
  }
}

/**
 * ChatTab container — renders sub-tab bar + My Chat or Agent Comms panel.
 */
export function ChatTab({ workspaceId, data }: Props) {
  const [subTab, setSubTab] = useState<ChatSubTab>("my-chat");

  return (
    <div className="flex flex-col h-full">
      {/* Sub-tab bar */}
      <div className="flex border-b border-zinc-800/40 bg-zinc-900/30 px-2 shrink-0">
        <button
          onClick={() => setSubTab("my-chat")}
          className={`px-3 py-1.5 text-[10px] font-medium transition-colors ${
            subTab === "my-chat"
              ? "text-zinc-200 border-b-2 border-blue-500"
              : "text-zinc-500 hover:text-zinc-300"
          }`}
        >
          My Chat
        </button>
        <button
          onClick={() => setSubTab("agent-comms")}
          className={`px-3 py-1.5 text-[10px] font-medium transition-colors ${
            subTab === "agent-comms"
              ? "text-zinc-200 border-b-2 border-blue-500"
              : "text-zinc-500 hover:text-zinc-300"
          }`}
        >
          Agent Comms
        </button>
      </div>
      {/* Content */}
      <div className="flex-1 overflow-hidden flex flex-col">
        {subTab === "my-chat" ? (
          <MyChatPanel workspaceId={workspaceId} data={data} />
        ) : (
          <AgentCommsPanel workspaceId={workspaceId} />
        )}
      </div>
    </div>
  );
}

/**
 * MyChatPanel — user↔agent conversation (extracted from original ChatTab).
 */
function MyChatPanel({ workspaceId, data }: Props) {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState("");
  const [sending, setSending] = useState(!!data.currentTask);
  const [thinkingElapsed, setThinkingElapsed] = useState(0);
  const [activityLog, setActivityLog] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const currentTaskRef = useRef(data.currentTask);
  const sendingFromAPIRef = useRef(false);
  const [agentReachable, setAgentReachable] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const bottomRef = useRef<HTMLDivElement>(null);

  // Load chat history from database on mount
  useEffect(() => {
    setLoading(true);
    loadMessagesFromDB(workspaceId).then((msgs) => {
      setMessages(msgs);
      setLoading(false);
    });
  }, [workspaceId]);

  // Agent reachability
  useEffect(() => {
    const reachable = data.status === "online" || data.status === "degraded";
    setAgentReachable(reachable);
    setError(reachable ? null : `Agent is ${data.status}`);
  }, [data.status]);

  useEffect(() => {
    currentTaskRef.current = data.currentTask;
  }, [data.currentTask]);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  // Consume agent push messages (send_message_to_user) from global store
  const pendingAgentMsgs = useCanvasStore((s) => s.agentMessages[workspaceId]);
  useEffect(() => {
    if (!pendingAgentMsgs || pendingAgentMsgs.length === 0) return;
    const consume = useCanvasStore.getState().consumeAgentMessages;
    const msgs = consume(workspaceId);
    for (const m of msgs) {
      setMessages((prev) => [...prev, createMessage("agent", m.content)]);
    }
  }, [pendingAgentMsgs, workspaceId]);

  // Consume A2A_RESPONSE events from global store (streaming response delivery).
  // Guarded by sendingFromAPIRef to avoid duplicate messages when the
  // synchronous HTTP .then() handler also fires for the same response.
  const pendingA2AResponse = useCanvasStore((s) => s.agentMessages[`a2a:${workspaceId}`]);
  useEffect(() => {
    if (!pendingA2AResponse || pendingA2AResponse.length === 0) return;
    const consume = useCanvasStore.getState().consumeAgentMessages;
    const msgs = consume(`a2a:${workspaceId}`);
    if (!sendingFromAPIRef.current) return; // HTTP .then() already handled this response
    for (const m of msgs) {
      setMessages((prev) => [...prev, createMessage("agent", m.content)]);
    }
    setSending(false);
    sendingFromAPIRef.current = false;
  }, [pendingA2AResponse, workspaceId]);

  // Resolve workspace ID → name for activity display
  const resolveWorkspaceName = useCallback((id: string) => {
    const nodes = useCanvasStore.getState().nodes;
    const node = nodes.find((n) => n.id === id);
    return (node?.data as WorkspaceNodeData)?.name || id.slice(0, 8);
  }, []);

  // Elapsed timer while sending
  useEffect(() => {
    if (!sending) {
      setThinkingElapsed(0);
      return;
    }
    const startTime = Date.now();
    const timer = setInterval(() => {
      setThinkingElapsed(Math.floor((Date.now() - startTime) / 1000));
    }, 1000);
    return () => clearInterval(timer);
  }, [sending]);

  // Live activity feed via WebSocket while sending
  useEffect(() => {
    if (!sending) {
      setActivityLog([]);
      return;
    }
    setActivityLog([`Processing with ${runtimeDisplayName(data.runtime)}...`]);

    const ws = new WebSocket(WS_URL);
    ws.onerror = () => {
      // Don't crash — activity feed is non-essential, just log
      console.warn("ChatTab activity feed WS error");
    };
    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        if (msg.event === "ACTIVITY_LOGGED") {
          const p = msg.payload || {};
          const type = p.activity_type as string;
          const method = (p.method as string) || "";
          const status = (p.status as string) || "";
          const targetId = (p.target_id as string) || "";
          const durationMs = p.duration_ms as number | undefined;

          let line = "";
          if (type === "a2a_receive" && method === "message/send") {
            const targetName = resolveWorkspaceName(targetId || msg.workspace_id);
            if (status === "ok" && durationMs) {
              const sec = Math.round(durationMs / 1000);
              line = `← ${targetName} responded (${sec}s)`;
            } else if (status === "error") {
              line = `⚠ ${targetName} error`;
            }
          } else if (type === "a2a_send") {
            const targetName = resolveWorkspaceName(targetId);
            line = `→ Delegating to ${targetName}...`;
          } else if (type === "task_update") {
            const summary = (p.summary as string) || "";
            if (summary) line = `⟳ ${summary}`;
          }

          if (line) {
            setActivityLog((prev) => [...prev.slice(-8), line]);
          }
        } else if (msg.event === "TASK_UPDATED" && msg.workspace_id === workspaceId) {
          const task = (msg.payload?.current_task as string) || "";
          if (task) {
            setActivityLog((prev) => [...prev.slice(-8), `⟳ ${task}`]);
          }
        }
        // A2A_RESPONSE is handled by the store (pendingA2AResponse effect) — no duplicate here
      } catch { /* ignore */ }
    };

    return () => ws.close();
  }, [sending, workspaceId, resolveWorkspaceName]);

  const sendMessage = async () => {
    const text = input.trim();
    if (!text || !agentReachable || sending) return;

    setInput("");
    setMessages((prev) => [...prev, createMessage("user", text)]);
    setSending(true);
    sendingFromAPIRef.current = true;
    setError(null);

    // Build conversation history from prior messages (last 20)
    const history = messages
      .filter((m) => m.role === "user" || m.role === "agent")
      .slice(-20)
      .map((m) => ({
        role: m.role === "user" ? "user" : "agent",
        parts: [{ kind: "text", text: m.content }],
      }));

    api.post<A2AResponse>(`/workspaces/${workspaceId}/a2a`, {
      method: "message/send",
      params: {
        message: {
          role: "user",
          messageId: crypto.randomUUID(),
          parts: [{ kind: "text", text }],
        },
        metadata: { history },
      },
    })
      .then((resp) => {
        // Skip if the WS A2A_RESPONSE event already handled this response.
        // Both paths (WS + HTTP) check sendingFromAPIRef — whichever clears
        // it first wins, the other becomes a no-op (no duplicate messages).
        if (!sendingFromAPIRef.current) return;
        const replyText = extractReplyText(resp);
        if (replyText) {
          setMessages((prev) => [...prev, createMessage("agent", replyText)]);
        }
        setSending(false);
        sendingFromAPIRef.current = false;
      })
      .catch(() => {
        setSending(false);
        sendingFromAPIRef.current = false;
        setError("Failed to send message — agent may be unreachable");
      });
  };

  const isOnline = data.status === "online" || data.status === "degraded";

  return (
    <div className="flex flex-col h-full">
      {/* Messages */}
      <div className="flex-1 overflow-y-auto p-3 space-y-3">
        {loading && (
          <div className="text-xs text-zinc-500 text-center py-4">Loading chat history...</div>
        )}
        {!loading && messages.length === 0 && (
          <div className="text-xs text-zinc-500 text-center py-8">
            No messages yet. Send a message to start chatting with this agent.
          </div>
        )}
        {messages.map((msg) => (
          <div key={msg.id} className={`flex ${msg.role === "user" ? "justify-end" : "justify-start"}`}>
            <div
              className={`max-w-[85%] rounded-lg px-3 py-2 text-xs ${
                msg.role === "user"
                  ? "bg-blue-600/30 text-blue-100 border border-blue-500/20"
                  : msg.role === "system"
                    ? "bg-red-900/30 text-red-200 border border-red-800/30"
                    : "bg-zinc-800/80 text-zinc-200 border border-zinc-700/30"
              }`}
            >
              <div className="prose prose-sm prose-invert max-w-none [&>p]:mb-1 [&>p:last-child]:mb-0">
                <ReactMarkdown remarkPlugins={[remarkGfm]}>{msg.content}</ReactMarkdown>
              </div>
              <div className="text-[9px] text-zinc-500 mt-1">
                {new Date(msg.timestamp).toLocaleTimeString()}
              </div>
            </div>
          </div>
        ))}

        {/* Thinking indicator */}
        {sending && (
          <div className="flex justify-start">
            <div className="bg-zinc-800/50 border border-zinc-700/30 rounded-lg px-3 py-2 max-w-[85%]">
              <div className="flex items-center gap-2 text-xs text-zinc-400">
                <span className="flex gap-0.5">
                  <span className="w-1.5 h-1.5 bg-zinc-500 rounded-full animate-bounce" style={{ animationDelay: "0ms" }} />
                  <span className="w-1.5 h-1.5 bg-zinc-500 rounded-full animate-bounce" style={{ animationDelay: "150ms" }} />
                  <span className="w-1.5 h-1.5 bg-zinc-500 rounded-full animate-bounce" style={{ animationDelay: "300ms" }} />
                </span>
                {thinkingElapsed}s
              </div>
              {activityLog.length > 0 && (
                <div className="mt-1.5 text-[9px] text-zinc-500 space-y-0.5">
                  <div className="text-zinc-400">Processing with {runtimeDisplayName(data.runtime)}...</div>
                  {activityLog.map((line, i) => (
                    <div key={i} className="pl-2 border-l border-zinc-700">◇ {line}</div>
                  ))}
                </div>
              )}
            </div>
          </div>
        )}
        <div ref={bottomRef} />
      </div>

      {/* Error banner */}
      {error && (
        <div className="px-3 py-2 bg-red-900/20 border-t border-red-800/30">
          <div className="flex items-center justify-between">
            <span className="text-[10px] text-red-400">{error}</span>
            {!isOnline && (
              <button
                onClick={() => {
                  if (confirm("Restart this workspace?")) {
                    useCanvasStore.getState().restartWorkspace(workspaceId);
                  }
                }}
                className="text-[9px] px-2 py-0.5 bg-red-800/40 text-red-300 rounded hover:bg-red-700/50"
              >
                Restart
              </button>
            )}
          </div>
        </div>
      )}

      {/* Input */}
      <div className="p-3 border-t border-zinc-800">
        <div className="flex gap-2">
          <textarea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && !e.shiftKey) {
                e.preventDefault();
                sendMessage();
              }
            }}
            placeholder={agentReachable ? "Send a message... (Shift+Enter for new line)" : `Agent is ${data.status}`}
            disabled={!agentReachable || sending}
            rows={1}
            className="flex-1 bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-xs text-zinc-200 placeholder-zinc-500 focus:outline-none focus:border-blue-500 resize-none disabled:opacity-50"
          />
          <button
            onClick={sendMessage}
            disabled={!input.trim() || !agentReachable || sending}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-500 text-xs font-medium rounded-lg text-white disabled:opacity-30 transition-colors shrink-0"
          >
            Send
          </button>
        </div>
      </div>
    </div>
  );
}
