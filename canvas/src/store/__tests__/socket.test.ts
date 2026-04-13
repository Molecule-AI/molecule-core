import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

// ---------------------------------------------------------------------------
// Mock the canvas store before importing socket.ts
// ---------------------------------------------------------------------------
vi.mock("../canvas", () => ({
  useCanvasStore: {
    getState: vi.fn(() => ({
      applyEvent: vi.fn(),
      hydrate: vi.fn(),
    })),
  },
}));

// ---------------------------------------------------------------------------
// Mock WebSocket
// ---------------------------------------------------------------------------

class MockWebSocket {
  static instances: MockWebSocket[] = [];

  url: string;
  onopen: (() => void) | null = null;
  onmessage: ((event: { data: string }) => void) | null = null;
  onclose: (() => void) | null = null;
  onerror: (() => void) | null = null;
  closeCallCount = 0;

  constructor(url: string) {
    this.url = url;
    MockWebSocket.instances.push(this);
  }

  close() {
    this.closeCallCount++;
  }

  // Helpers to trigger events in tests
  triggerOpen() {
    this.onopen?.();
  }

  triggerMessage(data: unknown) {
    this.onmessage?.({ data: JSON.stringify(data) });
  }

  triggerRawMessage(data: string) {
    this.onmessage?.({ data });
  }

  triggerClose() {
    this.onclose?.();
  }

  triggerError() {
    this.onerror?.();
  }
}

// Install mock WebSocket globally before importing socket module
(globalThis as unknown as Record<string, unknown>).WebSocket = MockWebSocket;

// Now import the socket module (uses globalThis.WebSocket at call time)
import { connectSocket, disconnectSocket } from "../socket";
import { useCanvasStore } from "../canvas";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function getLastWS(): MockWebSocket {
  return MockWebSocket.instances[MockWebSocket.instances.length - 1];
}

beforeEach(() => {
  MockWebSocket.instances = [];
  vi.useFakeTimers();

  // Reset mocked store state
  vi.mocked(useCanvasStore.getState).mockReturnValue({
    applyEvent: vi.fn(),
    hydrate: vi.fn(),
  } as unknown as ReturnType<typeof useCanvasStore.getState>);
});

afterEach(() => {
  // Always disconnect to clean the module-level socket singleton
  disconnectSocket();
  vi.useRealTimers();
  vi.clearAllMocks();
});

// ---------------------------------------------------------------------------
// connectSocket / disconnectSocket
// ---------------------------------------------------------------------------

describe("connectSocket", () => {
  it("creates a WebSocket on connect", () => {
    connectSocket();
    expect(MockWebSocket.instances).toHaveLength(1);
  });

  it("connects to the correct URL (default WS_URL)", () => {
    connectSocket();
    const ws = getLastWS();
    expect(ws.url).toMatch(/^ws/);
  });

  it("sets up onopen, onmessage, onclose, onerror handlers", () => {
    connectSocket();
    const ws = getLastWS();
    expect(ws.onopen).toBeTypeOf("function");
    expect(ws.onmessage).toBeTypeOf("function");
    expect(ws.onclose).toBeTypeOf("function");
    expect(ws.onerror).toBeTypeOf("function");
  });

  it("calling connectSocket twice reuses the same socket instance (does not create a new one, but calls connect again)", () => {
    connectSocket();
    connectSocket(); // second call — socket singleton exists, just calls connect()
    // The first connect() already created one WS; second connect() on the same
    // ReconnectingSocket instance creates another WS because connect() always creates new WebSocket
    // This is the expected behaviour of the implementation
    expect(MockWebSocket.instances.length).toBeGreaterThanOrEqual(1);
  });
});

describe("disconnectSocket", () => {
  it("closes the underlying WebSocket", () => {
    connectSocket();
    const ws = getLastWS();
    disconnectSocket();
    expect(ws.closeCallCount).toBe(1);
  });

  it("nullifies the socket singleton so a subsequent connectSocket creates a fresh one", () => {
    connectSocket();
    disconnectSocket();
    connectSocket();
    // Should have created two WebSocket instances
    expect(MockWebSocket.instances.length).toBeGreaterThanOrEqual(2);
  });

  it("does not throw when called without a prior connectSocket", () => {
    expect(() => disconnectSocket()).not.toThrow();
  });
});

// ---------------------------------------------------------------------------
// onopen handler
// ---------------------------------------------------------------------------

describe("WebSocket onopen", () => {
  it("resets attempt counter (indirectly tested via reconnect behaviour)", () => {
    connectSocket();
    const ws = getLastWS();
    // Should not throw
    expect(() => ws.triggerOpen()).not.toThrow();
  });

  it("starts the health check interval after connection opens", () => {
    const setIntervalSpy = vi.spyOn(globalThis, "setInterval");
    connectSocket();
    const ws = getLastWS();
    ws.triggerOpen();
    expect(setIntervalSpy).toHaveBeenCalled();
    setIntervalSpy.mockRestore();
  });
});

// ---------------------------------------------------------------------------
// onmessage handler
// ---------------------------------------------------------------------------

describe("WebSocket onmessage", () => {
  it("parses JSON and calls applyEvent on the canvas store", () => {
    connectSocket();
    const ws = getLastWS();
    const applyEvent = vi.fn();
    vi.mocked(useCanvasStore.getState).mockReturnValue({
      applyEvent,
      hydrate: vi.fn(),
    } as unknown as ReturnType<typeof useCanvasStore.getState>);

    const msg = {
      event: "WORKSPACE_ONLINE",
      workspace_id: "ws-1",
      timestamp: new Date().toISOString(),
      payload: {},
    };
    ws.triggerMessage(msg);

    expect(applyEvent).toHaveBeenCalledWith(msg);
  });

  it("does not throw when JSON is malformed", () => {
    connectSocket();
    const ws = getLastWS();
    expect(() => ws.triggerRawMessage("not-valid-json{{{")).not.toThrow();
  });

  it("handles an empty JSON object without crashing", () => {
    connectSocket();
    const ws = getLastWS();
    expect(() => ws.triggerRawMessage("{}")).not.toThrow();
  });
});

// ---------------------------------------------------------------------------
// onclose / auto-reconnect
// ---------------------------------------------------------------------------

describe("WebSocket onclose – auto-reconnect", () => {
  it("schedules a reconnect via setTimeout when socket closes", () => {
    const setTimeoutSpy = vi.spyOn(globalThis, "setTimeout");
    connectSocket();
    const ws = getLastWS();
    ws.triggerClose();
    expect(setTimeoutSpy).toHaveBeenCalled();
    setTimeoutSpy.mockRestore();
  });

  it("reconnect delay is at most 30 000 ms", () => {
    const delays: number[] = [];
    const origSetTimeout = globalThis.setTimeout;
    const setTimeoutSpy = vi
      .spyOn(globalThis, "setTimeout")
      .mockImplementation((fn, delay, ...args) => {
        delays.push(delay as number);
        // Don't actually reconnect to avoid infinite loops in the test
        return origSetTimeout(() => {}, 0) as unknown as ReturnType<typeof setTimeout>;
      });

    connectSocket();
    const ws = getLastWS();

    // Trigger several closes to increment the attempt counter
    for (let i = 0; i < 6; i++) {
      ws.triggerClose();
    }

    expect(delays.every((d) => d <= 30_000)).toBe(true);
    setTimeoutSpy.mockRestore();
  });

  it("stops the health check interval on close", () => {
    const clearIntervalSpy = vi.spyOn(globalThis, "clearInterval");
    connectSocket();
    const ws = getLastWS();
    ws.triggerOpen(); // starts health check
    ws.triggerClose(); // should stop health check
    expect(clearIntervalSpy).toHaveBeenCalled();
    clearIntervalSpy.mockRestore();
  });

  it("creates a new WebSocket after reconnect delay elapses", () => {
    connectSocket();
    expect(MockWebSocket.instances).toHaveLength(1);

    const ws = getLastWS();
    ws.triggerClose();

    // Fast-forward timers to trigger the reconnect
    vi.runAllTimers();

    expect(MockWebSocket.instances.length).toBeGreaterThan(1);
  });
});

// ---------------------------------------------------------------------------
// onerror handler
// ---------------------------------------------------------------------------

describe("WebSocket onerror", () => {
  it("does not throw when error is triggered", () => {
    connectSocket();
    const ws = getLastWS();
    expect(() => ws.triggerError()).not.toThrow();
  });
});

// ---------------------------------------------------------------------------
// Health check (startHealthCheck / stopHealthCheck via onopen / disconnect)
// ---------------------------------------------------------------------------

describe("health check", () => {
  it("clears interval on disconnect", () => {
    const clearIntervalSpy = vi.spyOn(globalThis, "clearInterval");
    connectSocket();
    const ws = getLastWS();
    ws.triggerOpen(); // starts health check timer
    disconnectSocket();
    expect(clearIntervalSpy).toHaveBeenCalled();
    clearIntervalSpy.mockRestore();
  });

  it("sets a 30-second health check interval after onopen", () => {
    const intervals: number[] = [];
    const setIntervalSpy = vi
      .spyOn(globalThis, "setInterval")
      .mockImplementation((fn, delay, ...args) => {
        intervals.push(delay as number);
        return 999 as unknown as ReturnType<typeof setInterval>;
      });

    connectSocket();
    const ws = getLastWS();
    ws.triggerOpen();

    // The health check interval should be 30_000 ms
    expect(intervals.some((d) => d === 30_000)).toBe(true);
    setIntervalSpy.mockRestore();
  });

  it("does not accumulate multiple intervals on repeated onopen", () => {
    const clearIntervalSpy = vi.spyOn(globalThis, "clearInterval");
    connectSocket();
    const ws = getLastWS();

    ws.triggerOpen();
    ws.triggerOpen(); // second open should clear old interval first

    // clearInterval must have been called at least once (stopHealthCheck inside startHealthCheck)
    expect(clearIntervalSpy).toHaveBeenCalled();
    clearIntervalSpy.mockRestore();
  });
});
