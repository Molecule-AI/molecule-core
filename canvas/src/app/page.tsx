"use client";

import { useEffect } from "react";
import { Canvas } from "@/components/Canvas";
import { Legend } from "@/components/Legend";
import { CommunicationOverlay } from "@/components/CommunicationOverlay";
import { connectSocket, disconnectSocket } from "@/store/socket";
import { useCanvasStore } from "@/store/canvas";
import { api } from "@/lib/api";
import type { WorkspaceData } from "@/store/socket";

export default function Home() {
  const hydrationError = useCanvasStore((s) => s.hydrationError);
  const setHydrationError = useCanvasStore((s) => s.setHydrationError);

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
      // Initial hydration failed — show error banner to user
      console.error("Canvas: initial hydration failed", err);
      useCanvasStore.getState().setHydrationError(
        err instanceof Error && err.message ? err.message : "Failed to load canvas"
      );
    });

    return () => {
      disconnectSocket();
    };
  }, []);

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
