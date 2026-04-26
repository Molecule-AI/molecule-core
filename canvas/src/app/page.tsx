"use client";

import { useEffect, useState } from "react";
import { Canvas } from "@/components/Canvas";
import { Legend } from "@/components/Legend";
import { CommunicationOverlay } from "@/components/CommunicationOverlay";
import { Spinner } from "@/components/Spinner";
import { connectSocket, disconnectSocket } from "@/store/socket";
import { useCanvasStore } from "@/store/canvas";
import { api, PlatformUnavailableError } from "@/lib/api";
import type { WorkspaceData } from "@/store/socket";

export default function Home() {
  const hydrationError = useCanvasStore((s) => s.hydrationError);
  const setHydrationError = useCanvasStore((s) => s.setHydrationError);
  const [hydrating, setHydrating] = useState(true);
  // Distinct from hydrationError: platform-down is its own UX path
  // (different copy, different action — the user's next step is to
  // check local services, not to retry the API call). Tracked
  // separately rather than encoded into hydrationError so the
  // generic-error branch can stay simple.
  const [platformDown, setPlatformDown] = useState(false);

  useEffect(() => {
    connectSocket();

    // Hydrate workspaces and restore viewport in parallel
    Promise.all([
      api.get<WorkspaceData[]>("/workspaces"),
      api.get<{ x: number; y: number; zoom: number }>("/canvas/viewport").catch(() => null),
    ]).then(([workspaces, viewport]) => {
      useCanvasStore.getState().hydrate(workspaces);
      if (viewport) {
        useCanvasStore.getState().setViewport(viewport);
      }
    }).catch((err) => {
      console.error("Canvas: initial hydration failed", err);
      if (err instanceof PlatformUnavailableError) {
        setPlatformDown(true);
        return;
      }
      useCanvasStore.getState().setHydrationError(
        err instanceof Error && err.message ? err.message : "Failed to load canvas"
      );
    }).finally(() => {
      setHydrating(false);
    });

    return () => {
      disconnectSocket();
    };
  }, []);

  if (hydrating) {
    return (
      <div className="fixed inset-0 flex items-center justify-center bg-zinc-950">
        <div className="flex flex-col items-center gap-3">
          <Spinner size="lg" />
          <span className="text-xs text-zinc-500">Loading canvas...</span>
        </div>
      </div>
    );
  }

  if (platformDown) {
    return <PlatformDownDiagnostic />;
  }

  return (
    <>
      <Canvas />
      <Legend />
      <CommunicationOverlay />
      {hydrationError && (
        <div
          role="alert"
          className="fixed inset-0 flex flex-col items-center justify-center bg-zinc-950 text-zinc-300 gap-4 z-[9999]"
        >
          <p className="text-zinc-400 text-sm">{hydrationError}</p>
          <button
            onClick={() => {
              setHydrationError(null);
              window.location.reload();
            }}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-500 text-white rounded-md text-sm"
          >
            Retry
          </button>
        </div>
      )}
    </>
  );
}

/**
 * Dedicated diagnostic for the case where the platform reported its
 * datastore (Postgres / Redis) is unreachable. Distinct from the
 * generic API-error overlay: the user's next action is to check
 * local services, not to retry the API call. Includes the exact
 * commands for the common dev-host setup.
 */
function PlatformDownDiagnostic() {
  return (
    <div
      role="alert"
      className="fixed inset-0 flex flex-col items-center justify-center bg-zinc-950 text-zinc-300 gap-5 z-[9999] px-6"
    >
      <div className="text-amber-400 text-sm font-semibold uppercase tracking-wider">
        Platform infrastructure unreachable
      </div>
      <p className="text-zinc-400 text-sm max-w-lg text-center leading-relaxed">
        The platform server returned <code className="font-mono text-amber-300">503 platform_unavailable</code>.
        That means it can&apos;t reach Postgres or Redis to validate your session.
        Most common cause on a dev host: one of those services stopped.
      </p>
      <div className="bg-zinc-900/80 border border-zinc-700/50 rounded-lg px-4 py-3 max-w-lg w-full">
        <div className="text-[10px] uppercase tracking-wider text-zinc-500 mb-2">Try first</div>
        <pre className="text-[12px] text-zinc-300 font-mono whitespace-pre-wrap leading-relaxed">{`brew services start postgresql@14
brew services start redis`}</pre>
      </div>
      <p className="text-[11px] text-zinc-500 max-w-lg text-center">
        If both are running, check <code className="font-mono">/tmp/molecule-server.log</code> for
        the underlying error. If you&apos;re on hosted SaaS, this is a platform incident — try again in a moment.
      </p>
      <button
        onClick={() => window.location.reload()}
        className="px-4 py-2 bg-blue-600 hover:bg-blue-500 text-white rounded-md text-sm mt-2"
      >
        Reload
      </button>
    </div>
  );
}
