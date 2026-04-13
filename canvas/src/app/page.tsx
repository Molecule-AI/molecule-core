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
      // Initial hydration failed — socket reconnect will retry
      console.error("Canvas: initial hydration failed", err);
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
    </>
  );
}
