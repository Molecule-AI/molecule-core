import { useCanvasStore } from "./canvas";
import { deriveWsBaseUrl } from "@/lib/ws-url";

// If explicit WS_URL is set, use it as-is (may include custom path).
// Otherwise derive base + append /ws.
export const WS_URL = process.env.NEXT_PUBLIC_WS_URL || (deriveWsBaseUrl() + "/ws");

export interface WSMessage {
  event: string;
  workspace_id: string;
  timestamp: string;
  payload: Record<string, unknown>;
}

class ReconnectingSocket {
  private ws: WebSocket | null = null;
  private attempt = 0;
  private url: string;
  private lastEventTime = 0;
  private healthCheckTimer: ReturnType<typeof setInterval> | null = null;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  // disposed signals that disconnect() has been called. Any in-flight
  // reconnect / handshake must abort early rather than attach to a
  // socket the caller no longer owns — otherwise React StrictMode's
  // effect double-invoke (and any future intentional disconnect)
  // leaves a zombie WebSocket alive forever.
  private disposed = false;

  constructor(url: string) {
    this.url = url;
  }

  connect() {
    if (this.disposed) return;
    useCanvasStore.getState().setWsStatus("connecting");
    const ws = new WebSocket(this.url);
    this.ws = ws;

    ws.onopen = () => {
      if (this.disposed || this.ws !== ws) {
        // Late-open on an abandoned socket. Close it cleanly; the
        // caller already moved on.
        try { ws.close(); } catch { /* noop */ }
        return;
      }
      this.attempt = 0;
      this.lastEventTime = Date.now();
      useCanvasStore.getState().setWsStatus("connected");
      this.rehydrate();
      this.startHealthCheck();
    };

    ws.onmessage = (event) => {
      if (this.disposed || this.ws !== ws) return;
      this.lastEventTime = Date.now();
      try {
        const msg: WSMessage = JSON.parse(event.data);
        useCanvasStore.getState().applyEvent(msg);
      } catch {
        // Malformed WS message — skip silently
      }
    };

    ws.onclose = () => {
      // Fired on intentional close (disposed) OR server/network drop.
      // Only schedule a reconnect when the socket is still live AND
      // corresponds to the WS we just tore down (prevents a stale
      // onclose from a zombie socket from re-arming the loop).
      if (this.disposed || this.ws !== ws) return;
      this.stopHealthCheck();
      useCanvasStore.getState().setWsStatus("connecting");
      const delay = Math.min(1000 * 2 ** this.attempt, 30000);
      this.attempt++;
      this.reconnectTimer = setTimeout(() => this.connect(), delay);
    };

    ws.onerror = () => {
      // Suppressed — onclose handles reconnection. onerror fires before onclose
      // and the Event object doesn't contain useful info (serializes to {}).
    };
  }

  /** Periodically re-fetch state in case WebSocket events were missed (e.g. agent
   *  status changed while the socket stayed open but no event was emitted). */
  private startHealthCheck() {
    this.stopHealthCheck();
    this.healthCheckTimer = setInterval(() => {
      const silenceSec = (Date.now() - this.lastEventTime) / 1000;
      // If no events for 30s, re-hydrate to catch missed status changes
      if (silenceSec > 30) {
        this.rehydrate();
        this.lastEventTime = Date.now(); // prevent rapid re-fetches
      }
    }, 30_000);
  }

  private stopHealthCheck() {
    if (this.healthCheckTimer) {
      clearInterval(this.healthCheckTimer);
      this.healthCheckTimer = null;
    }
  }

  private async rehydrate() {
    try {
      const { api } = await import("@/lib/api");
      const workspaces = await api.get<WorkspaceData[]>("/workspaces");
      useCanvasStore.getState().hydrate(workspaces);
    } catch {
      // Rehydration failed — will retry on next health check cycle
    }
  }

  disconnect() {
    this.disposed = true;
    this.stopHealthCheck();
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    if (this.ws) {
      // Detach listeners before close() so we don't route the close
      // event through our onclose → scheduleReconnect path. Belt +
      // braces on top of the `disposed` check, because StrictMode
      // cycles through so fast that an attached onclose can fire
      // after disposed=true is set but before this assignment runs.
      this.ws.onopen = null;
      this.ws.onmessage = null;
      this.ws.onclose = null;
      this.ws.onerror = null;
      try { this.ws.close(); } catch { /* noop */ }
      this.ws = null;
    }
    useCanvasStore.getState().setWsStatus("disconnected");
  }
}

export interface WorkspaceData {
  id: string;
  name: string;
  role: string;
  tier: number;
  status: string;
  agent_card: Record<string, unknown> | null;
  url: string;
  parent_id: string | null;
  active_tasks: number;
  last_error_rate: number;
  last_sample_error: string;
  uptime_seconds: number;
  current_task: string;
  runtime: string;
  x: number;
  y: number;
  collapsed: boolean;
  /** USD spend ceiling set by the user; null = unlimited. Added by issue #541. */
  budget_limit: number | null;
  /** Cumulative USD spend for this workspace. Present when the platform tracks spend. */
  budget_used?: number | null;
}

let socket: ReconnectingSocket | null = null;

export function connectSocket() {
  if (!socket) {
    socket = new ReconnectingSocket(WS_URL);
  }
  socket.connect();
}

export function disconnectSocket() {
  if (socket) {
    socket.disconnect();
    socket = null;
  }
}
