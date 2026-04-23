"use client";

import { useState, useEffect, useCallback } from "react";
import { api } from "@/lib/api";

interface Props {
  workspaceId: string;
}

interface EventEntry {
  id: string;
  event_type: string;
  workspace_id: string | null;
  payload: Record<string, unknown>;
  created_at: string;
}

const EVENT_COLORS: Record<string, string> = {
  WORKSPACE_ONLINE: "text-green-400",
  WORKSPACE_OFFLINE: "text-zinc-400",
  WORKSPACE_DEGRADED: "text-yellow-400",
  WORKSPACE_PROVISIONING: "text-blue-400",
  WORKSPACE_REMOVED: "text-red-400",
  WORKSPACE_PROVISION_FAILED: "text-red-400",
  AGENT_CARD_UPDATED: "text-purple-400",
};

export function EventsTab({ workspaceId }: Props) {
  const [events, setEvents] = useState<EventEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [expanded, setExpanded] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const loadEvents = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await api.get<EventEntry[]>(`/events/${workspaceId}`);
      setEvents(data);
    } catch (e) {
      setEvents([]);
      setError(e instanceof Error ? e.message : "Failed to load events");
    } finally {
      setLoading(false);
    }
  }, [workspaceId]);

  useEffect(() => {
    loadEvents();
  }, [loadEvents]);

  // Auto-refresh every 10s
  useEffect(() => {
    const interval = setInterval(loadEvents, 10000);
    return () => clearInterval(interval);
  }, [loadEvents]);

  if (loading && events.length === 0) {
    return <div className="p-4 text-xs text-zinc-500">Loading events...</div>;
  }

  return (
    <div className="p-4 space-y-2">
      <div className="flex items-center justify-between mb-2">
        <span className="text-xs text-zinc-400">{events.length} events</span>
        <button
          onClick={loadEvents}
          className="px-2 py-1 bg-zinc-700 hover:bg-zinc-600 text-[10px] rounded text-zinc-300"
        >
          Refresh
        </button>
      </div>

      {error && (
        <div className="px-3 py-1.5 bg-red-900/30 border border-red-800 rounded text-xs text-red-400">
          {error}
        </div>
      )}

      {!error && events.length === 0 ? (
        <p className="text-xs text-zinc-500 text-center py-4">No events yet</p>
      ) : (
        <div className="space-y-1">
          {events.map((event) => (
            <div key={event.id} className="bg-zinc-800 rounded border border-zinc-700">
              <button
                onClick={() => setExpanded(expanded === event.id ? null : event.id)}
                className="w-full flex items-center gap-2 px-3 py-2 text-left"
              >
                <span
                  className={`text-xs font-mono ${
                    EVENT_COLORS[event.event_type] || "text-zinc-300"
                  }`}
                >
                  {event.event_type}
                </span>
                <span className="text-[9px] text-zinc-500 ml-auto">
                  {formatTime(event.created_at)}
                </span>
                <span className="text-[10px] text-zinc-500">
                  {expanded === event.id ? "▼" : "▶"}
                </span>
              </button>

              {expanded === event.id && (
                <div className="px-3 pb-2">
                  <pre className="text-[10px] text-zinc-300 bg-zinc-900 rounded p-2 overflow-x-auto max-h-40">
                    {JSON.stringify(event.payload, null, 2)}
                  </pre>
                  <div className="mt-1 text-[9px] text-zinc-500 font-mono">
                    ID: {event.id}
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function formatTime(iso: string): string {
  const d = new Date(iso);
  const now = new Date();
  const diff = now.getTime() - d.getTime();

  if (diff < 60_000) return `${Math.floor(diff / 1000)}s ago`;
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m ago`;
  if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)}h ago`;
  return d.toLocaleDateString();
}
