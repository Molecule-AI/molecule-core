"use client";

import { useState, useEffect, useRef } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { api } from "@/lib/api";
import { useCanvasStore, type WorkspaceNodeData } from "@/store/canvas";
import { WS_URL } from "@/store/socket";
import { closeWebSocketGracefully } from "@/lib/ws-close";
import { showToast } from "../../Toaster";
import { extractResponseText, extractRequestText } from "./message-parser";
import { inferA2AErrorHint } from "./a2aErrorHint";

export interface ActivityEntry {
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
  /** UI-facing flow from THIS workspace's point of view:
   *
   *    "out" — this workspace either initiated the call (a2a_send)
   *            OR self-logged the reply from a peer it had called
   *            (a2a_receive with source_id == workspaceId).
   *    "in"  — a peer initiated the call to us (a2a_receive with
   *            source_id != workspaceId).
   *
   *  Distinct from activity_type because the agent runtime self-
   *  logs its outbound calls' replies as `a2a_receive` rows; without
   *  this normalisation the UI labels would render those as
   *  incoming ("← From X") and right-justify them on the wrong
   *  side, even though from the user's perspective the call WAS
   *  outgoing. See toCommMessage for the resolution rules. */
  flow: "in" | "out";
  peerName: string;
  peerId: string;
  text: string;
  responseText: string | null;
  /** "ok" | "error" — surfaces failed deliveries with their own
   *  visual treatment + recovery actions instead of an opaque
   *  "[A2A_ERROR]" body the user can't act on. */
  status: string;
  timestamp: string;
}

function resolveName(id: string): string {
  const nodes = useCanvasStore.getState().nodes;
  const node = nodes.find((n) => n.id === id);
  return (node?.data as WorkspaceNodeData)?.name || id.slice(0, 8);
}

export function toCommMessage(entry: ActivityEntry, workspaceId: string): CommMessage | null {
  // delegation activity rows are written by the platform's /delegate
  // handler. They're always outbound from this workspace's POV (the
  // platform proxies the A2A on our behalf). Two methods:
  //   - "delegate"        — the initial outbound; status pending/dispatched
  //   - "delegate_result" — the eventual reply; status completed/queued/failed
  // We surface them in Agent Comms because they ARE agent-to-agent
  // calls; without this branch they'd be dropped by the activity_type
  // filter and the user would see "No agent-to-agent communications yet"
  // even when the director made delegations.
  if (entry.activity_type === "delegation") {
    const peerId = entry.target_id || "";
    if (!peerId) return null;
    return {
      id: entry.id,
      flow: "out",
      peerName: resolveName(peerId),
      peerId,
      // Prefer summary (set by the platform with a human-readable
      // string like "Delegating to X" or "Delegation queued — target
      // at capacity"). Fall back to request body for older rows that
      // pre-date the summary column being populated.
      text: entry.summary || extractRequestText(entry.request_body) || "(delegation)",
      responseText: entry.response_body ? extractResponseText(entry.response_body) : null,
      status: entry.status || "ok",
      timestamp: entry.created_at,
    };
  }

  // a2a_receive activity rows come in two shapes:
  //
  //   1. Real incoming call (a peer called us): source_id = the peer,
  //      target_id = us. peerId is source_id, flow is "in".
  //
  //   2. Self-logged response to an outbound call (the workspace's own
  //      runtime calls report_activity("a2a_receive", ...) after
  //      delegating; see workspace/a2a_tools.py:181). source_id =
  //      our own workspace_id, target_id = the peer that replied.
  //      peerId must come from target_id (otherwise the peer-name
  //      resolves to "us" and Restart would target THIS workspace),
  //      and flow is "out" — from the user's perspective this row
  //      belongs to the outbound thread, not an incoming one.
  //
  // a2a_send rows are always outbound from us: source_id = us,
  // target_id = the peer.
  const isSendActivity = entry.activity_type === "a2a_send";
  const isSelfLoggedReceive =
    entry.activity_type === "a2a_receive" && entry.source_id === workspaceId;
  const flow: "in" | "out" = isSendActivity || isSelfLoggedReceive ? "out" : "in";
  const peerId =
    isSendActivity || isSelfLoggedReceive
      ? entry.target_id || ""
      : entry.source_id || "";
  if (!peerId) return null;

  const text = extractRequestText(entry.request_body) || entry.summary || "";
  const responseText = entry.response_body ? extractResponseText(entry.response_body) : null;

  return {
    id: entry.id,
    flow,
    peerName: resolveName(peerId),
    peerId,
    text,
    responseText,
    status: entry.status || "ok",
    timestamp: entry.created_at,
  };
}

/** Strip the [A2A_ERROR] sentinel prefix the workspace runtime adds
 *  to failed delegation responses, so the UI can render the underlying
 *  message (or fall back to a generic explanation when the inner text
 *  is empty — currently common because httpx exceptions often
 *  stringify as ""). */
const A2A_ERROR_PREFIX = "[A2A_ERROR]";

function unwrapErrorText(raw: string | null): string {
  if (!raw) return "";
  const trimmed = raw.trim();
  if (trimmed.startsWith(A2A_ERROR_PREFIX)) {
    return trimmed.slice(A2A_ERROR_PREFIX.length).trim();
  }
  return trimmed;
}

// inferA2AErrorHint moved to ./a2aErrorHint so the Activity tab and
// this panel render identical hints for the same symptom.

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
        const filtered = (entries ?? [])
          .filter((e) =>
            e.activity_type === "a2a_send" ||
            e.activity_type === "a2a_receive" ||
            e.activity_type === "delegation",
          )
          .reverse();
        const msgs: CommMessage[] = [];
        for (const e of filtered) {
          // Per-row try/catch so a single malformed activity row
          // (e.g. unexpected request_body shape) doesn't kill the
          // batch — the previous code threw out of the for-loop and
          // setMessages([3 items]) never ran, leaving the panel
          // stuck on the empty state with no diagnostic in the
          // console because the outer .catch silently swallowed
          // everything.
          try {
            const m = toCommMessage(e, workspaceId);
            if (m) {
              const key = `${m.timestamp}:${m.flow}:${m.peerId}`;
              msgs.push(m);
              seenKeys.current.add(key);
            }
          } catch (rowErr) {
            console.warn(
              "AgentCommsPanel: failed to map activity row",
              { id: e.id, type: e.activity_type, err: rowErr },
            );
          }
        }
        setMessages(msgs);
        setLoading(false);
      })
      .catch((err) => {
        // Surface the failure in the console so a stuck panel is
        // diagnosable without a debugger. Previous bare
        // `.catch(() => setLoading(false))` swallowed every load
        // failure (network errors, JSON parse errors, throws inside
        // the .then body) — the panel just sat on the empty state
        // with zero signal.
        console.warn("AgentCommsPanel: load activity failed", err);
        setLoading(false);
      });
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
        if (msg.workspace_id !== workspaceId) return;

        // Two live-update paths:
        //   1. ACTIVITY_LOGGED — fired by the LogActivity helper for
        //      a2a_send / a2a_receive (and delegation rows IF the
        //      delegation handler is ever refactored to use it). Today
        //      the platform's delegation handlers do direct INSERT
        //      INTO activity_logs WITHOUT firing ACTIVITY_LOGGED, so
        //      the delegation branch here is defensive — it'd light
        //      up automatically the day delegation handlers are
        //      refactored to call LogActivity.
        //   2. DELEGATION_SENT / DELEGATION_STATUS / DELEGATION_COMPLETE
        //      / DELEGATION_FAILED — fired by the platform's delegation
        //      handlers directly. These are the ONLY live signals the
        //      panel currently has for delegation rows; the GET on
        //      mount (which reads from activity_logs) shows them, but
        //      without this branch, nothing surfaced live until the
        //      next remount. Synthesise an ActivityEntry from the
        //      payload so toCommMessage's existing delegation branch
        //      handles them identically to the GET path.
        let entry: ActivityEntry | null = null;
        if (msg.event === "ACTIVITY_LOGGED") {
          const p = msg.payload || {};
          const type = p.activity_type as string;
          const sourceId = p.source_id as string | null;
          if (!sourceId) return; // canvas-initiated, not agent comms
          if (type !== "a2a_send" && type !== "a2a_receive" && type !== "delegation") return;
          entry = {
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
        } else if (
          msg.event === "DELEGATION_SENT" ||
          msg.event === "DELEGATION_STATUS" ||
          msg.event === "DELEGATION_COMPLETE" ||
          msg.event === "DELEGATION_FAILED"
        ) {
          const p = msg.payload || {};
          const targetId = (p.target_id as string) || "";
          if (!targetId) return;
          // Map event → status. DELEGATION_STATUS payload includes its
          // own `status` field (queued / dispatched). Other events have
          // implicit status: SENT → pending, COMPLETE → completed,
          // FAILED → failed.
          let status: string;
          let summary: string;
          if (msg.event === "DELEGATION_STATUS") {
            status = (p.status as string) || "queued";
            summary = `Delegation ${status}`;
          } else if (msg.event === "DELEGATION_COMPLETE") {
            status = "completed";
            summary = `Delegation completed (${(p.response_preview as string)?.slice(0, 60) || ""})`;
          } else if (msg.event === "DELEGATION_FAILED") {
            status = "failed";
            summary = `Delegation failed: ${(p.error as string) || "unknown"}`;
          } else {
            status = "pending";
            summary = `Delegating to ${(p.target_id as string)?.slice(0, 8) || "peer"}`;
          }
          entry = {
            id: (p.delegation_id as string) || crypto.randomUUID(),
            activity_type: "delegation",
            source_id: workspaceId,
            target_id: targetId,
            method: msg.event === "DELEGATION_SENT" ? "delegate" : "delegate_result",
            summary,
            request_body: null,
            response_body: null,
            status,
            created_at: msg.timestamp || new Date().toISOString(),
          };
        } else {
          return;
        }

        const m = toCommMessage(entry, workspaceId);
        if (m) {
          const key = `${m.timestamp}:${m.flow}:${m.peerId}`;
          if (seenKeys.current.has(key)) return;
          seenKeys.current.add(key);
          setMessages((prev) => [...prev, m]);
        }
      } catch { /* ignore */ }
    };
    return () => {
      closeWebSocketGracefully(ws);
    };
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
      {messages.map((msg) =>
        msg.status === "error" ? (
          <ErrorMessage key={msg.id} msg={msg} />
        ) : (
          <NormalMessage key={msg.id} msg={msg} />
        ),
      )}
      <div ref={bottomRef} />
    </div>
  );
}

function NormalMessage({ msg }: { msg: CommMessage }) {
  return (
    <div className={`flex ${msg.flow === "out" ? "justify-end" : "justify-start"}`}>
      <div
        className={`max-w-[85%] rounded-lg px-3 py-2 text-xs ${
          msg.flow === "out"
            ? "bg-cyan-900/30 text-cyan-100 border border-cyan-700/20"
            : "bg-zinc-800/80 text-zinc-200 border border-zinc-700/30"
        }`}
      >
        <div className="text-[9px] text-zinc-500 mb-1">
          {msg.flow === "out" ? `→ To ${msg.peerName}` : `← From ${msg.peerName}`}
        </div>
        {msg.text ? (
          <MarkdownBody className="text-zinc-300">{msg.text}</MarkdownBody>
        ) : (
          <div className="text-zinc-300">(no message text)</div>
        )}
        {msg.responseText && (
          <MarkdownBody className="mt-1.5 pt-1.5 border-t border-zinc-700/30 text-zinc-400">
            {msg.responseText}
          </MarkdownBody>
        )}
        <div className="text-[9px] text-zinc-500 mt-1">
          {new Date(msg.timestamp).toLocaleTimeString()}
        </div>
      </div>
    </div>
  );
}

/** Failure-state row. Replaces the unactionable "X failed [A2A_ERROR]"
 *  bubble with: a clear banner naming the peer, the underlying
 *  error text (if any), an inferred cause hint, and recovery
 *  actions — Restart workspace, Open workspace.
 *
 *  Recovery actions show on BOTH directions because both target the
 *  same peer (toCommMessage now resolves peerId to the peer in
 *  either case): an outbound delivery failure ("we called X and it
 *  errored"), an inbound runtime failure ("X called us and our
 *  reply errored" — rare), or the agent-self-logged "I called X and
 *  got an error back" pattern that is the most common shape. The
 *  user always wants to restart or inspect the failing peer. */
function ErrorMessage({ msg }: { msg: CommMessage }) {
  const selectNode = useCanvasStore((s) => s.selectNode);
  const [restarting, setRestarting] = useState(false);
  const errorText = unwrapErrorText(msg.responseText);
  const hint = inferA2AErrorHint(errorText);

  // Guard against acting on a peer whose workspace has been deleted
  // since this row was logged. Without the guard, restart 404s
  // surface as a generic toast and Open silently sets a dangling
  // selection that renders nothing in the side panel.
  const peerExists = (): boolean => {
    return useCanvasStore.getState().nodes.some((n) => n.id === msg.peerId);
  };

  const handleRestart = async () => {
    if (restarting) return;
    if (!peerExists()) {
      showToast(`${msg.peerName} no longer exists`, "error");
      return;
    }
    setRestarting(true);
    try {
      await api.post(`/workspaces/${msg.peerId}/restart`, {});
      showToast(`Restarting ${msg.peerName}…`, "success");
    } catch (e) {
      showToast(
        `Restart failed: ${e instanceof Error ? e.message : "unknown error"}`,
        "error",
      );
    } finally {
      setRestarting(false);
    }
  };

  const handleOpen = () => {
    if (!peerExists()) {
      showToast(`${msg.peerName} no longer exists`, "error");
      return;
    }
    selectNode(msg.peerId);
  };

  return (
    <div className={`flex ${msg.flow === "out" ? "justify-end" : "justify-start"}`}>
      <div className="max-w-[85%] rounded-lg border border-red-800/50 bg-red-950/30 px-3 py-2 text-xs">
        <div className="flex items-center gap-1.5 text-[10px] text-red-300 font-semibold uppercase tracking-wide mb-1.5">
          <span aria-hidden="true">⚠</span>
          {msg.flow === "out"
            ? `Failed to deliver to ${msg.peerName}`
            : `${msg.peerName} returned an error`}
        </div>

        {msg.text && (
          <div className="text-[10px] text-zinc-500 mb-1.5">
            <span className="uppercase tracking-wide">Task</span>
            <MarkdownBody className="text-zinc-400">{msg.text}</MarkdownBody>
          </div>
        )}

        <div className="rounded bg-zinc-950/60 border border-red-900/40 px-2 py-1.5 mb-1.5">
          <div className="text-[9px] uppercase tracking-wide text-red-400 mb-0.5">
            Underlying error
          </div>
          <code className="text-[11px] font-mono text-red-200 whitespace-pre-wrap break-words">
            {errorText || "(no detail returned)"}
          </code>
        </div>

        <p className="text-[10px] text-zinc-400 leading-snug mb-2">{hint}</p>

        {msg.peerId && (
          <div className="flex flex-wrap items-center gap-1.5">
            <button
              type="button"
              onClick={handleRestart}
              disabled={restarting}
              className="px-2 py-0.5 rounded bg-red-900/50 hover:bg-red-800/60 border border-red-700/40 text-[10px] text-red-200 disabled:opacity-50 transition-colors"
            >
              {restarting ? "Restarting…" : `Restart ${msg.peerName}`}
            </button>
            <button
              type="button"
              onClick={handleOpen}
              className="px-2 py-0.5 rounded bg-zinc-800 hover:bg-zinc-700 border border-zinc-700/50 text-[10px] text-zinc-300 transition-colors"
            >
              Open {msg.peerName}
            </button>
          </div>
        )}

        <div className="text-[9px] text-zinc-500 mt-1.5">
          {new Date(msg.timestamp).toLocaleTimeString()}
        </div>
      </div>
    </div>
  );
}

/** Tiny markdown wrapper matching ChatTab's My Chat styling. Same
 *  remark-gfm pipeline (tables, strikethrough, task lists) plus the
 *  prose tweaks that keep paragraphs tight inside a small bubble.
 *  Code blocks get an `overflow-x-auto` so a long line of code doesn't
 *  blow out the bubble's max-width — agent-to-agent replies routinely
 *  ship code samples and JSON. */
function MarkdownBody({
  children,
  className,
}: {
  children: string;
  className?: string;
}) {
  return (
    <div
      className={`prose prose-sm prose-invert max-w-none [&>p]:mb-1 [&>p:last-child]:mb-0 [&_pre]:overflow-x-auto [&_table]:block [&_table]:overflow-x-auto ${className ?? ""}`}
    >
      <ReactMarkdown remarkPlugins={[remarkGfm]}>{children}</ReactMarkdown>
    </div>
  );
}
