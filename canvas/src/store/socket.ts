import { useCanvasStore } from "./canvas";

// Derive WebSocket URL. Priority:
// 1. Explicit NEXT_PUBLIC_WS_URL (non-empty)
// 2. Derived from NEXT_PUBLIC_PLATFORM_URL (http→ws + /ws)
// 3. Derived from window.location (for same-origin tenant image)
// 4. Fallback to localhost
function deriveWsUrl(): string {
  const explicit = process.env.NEXT_PUBLIC_WS_URL;
  if (explicit) return explicit;

  const platform = process.env.NEXT_PUBLIC_PLATFORM_URL;
  if (platform) return platform.replace(/^http/, "ws").concat("/ws");

  // Same-origin tenant: derive from browser location
  if (typeof window !== "undefined") {
    const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
    return `${proto}//${window.location.host}/ws`;
  }

  return "ws://localhost:8080/ws";
}

export const WS_URL = deriveWsUrl();

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

  constructor(url: string) {
    this.url = url;
  }

  connect() {
    useCanvasStore.getState().setWsStatus("connecting");
    this.ws = new WebSocket(this.url);

    this.ws.onopen = () => {
      this.attempt = 0;
      this.lastEventTime = Date.now();
      useCanvasStore.getState().setWsStatus("connected");
      this.rehydrate();
      this.startHealthCheck();
    };

    this.ws.onmessage = (event) => {
      this.lastEventTime = Date.now();
      try {
        const msg: WSMessage = JSON.parse(event.data);
        useCanvasStore.getState().applyEvent(msg);
      } catch {
        // Malformed WS message — skip silently
      }
    };

    this.ws.onclose = () => {
      this.stopHealthCheck();
      useCanvasStore.getState().setWsStatus("connecting");
      const delay = Math.min(1000 * 2 ** this.attempt, 30000);
      this.attempt++;
      setTimeout(() => this.connect(), delay);
    };

    this.ws.onerror = () => {
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
    this.stopHealthCheck();
    if (this.ws) {
      this.ws.close();
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
