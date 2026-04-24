"use client";

import { useCallback, useEffect, useRef } from "react";
import { useReactFlow } from "@xyflow/react";
import { useCanvasStore } from "@/store/canvas";
import {
  CHILD_DEFAULT_HEIGHT,
  CHILD_DEFAULT_WIDTH,
} from "@/store/canvas-topology";

/**
 * Wires the two canvas-wide CustomEvent listeners and the viewport
 * save/restore bookkeeping so Canvas.tsx doesn't have to.
 *
 *   - `molecule:pan-to-node` — scroll viewport onto a specific node
 *     without forcing a specific zoom level (fitView adapts to current).
 *   - `molecule:zoom-to-team` — fit the viewport to a parent + its
 *     direct children, with a small padding.
 *
 * Also returns an `onMoveEnd` handler that debounces viewport saves so
 * the backend isn't spammed with pans.
 */
export function useCanvasViewport() {
  const { fitBounds, fitView } = useReactFlow();
  const saveViewport = useCanvasStore((s) => s.saveViewport);
  const saveTimerRef = useRef<ReturnType<typeof setTimeout>>(undefined);
  const panTimerRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  useEffect(() => {
    return () => {
      clearTimeout(saveTimerRef.current);
      clearTimeout(panTimerRef.current);
    };
  }, []);

  // Pan to a newly deployed / targeted workspace. 100ms delay so React
  // Flow has time to measure a just-rendered node.
  useEffect(() => {
    const handler = (e: Event) => {
      const { nodeId } = (e as CustomEvent<{ nodeId: string }>).detail;
      clearTimeout(panTimerRef.current);
      panTimerRef.current = setTimeout(() => {
        fitView({ nodes: [{ id: nodeId }], duration: 400, padding: 0.3 });
      }, 100);
    };
    window.addEventListener("molecule:pan-to-node", handler);
    return () => window.removeEventListener("molecule:pan-to-node", handler);
  }, [fitView]);

  // Zoom to a team: fit the parent + its direct children in view.
  useEffect(() => {
    const handler = (e: Event) => {
      const { nodeId } = (e as CustomEvent).detail;
      const state = useCanvasStore.getState();
      const children = state.nodes.filter((n) => n.data.parentId === nodeId);
      if (children.length === 0) return;
      const parent = state.nodes.find((n) => n.id === nodeId);
      const allNodes = parent ? [parent, ...children] : children;

      let minX = Infinity,
        minY = Infinity,
        maxX = -Infinity,
        maxY = -Infinity;
      for (const n of allNodes) {
        minX = Math.min(minX, n.position.x);
        minY = Math.min(minY, n.position.y);
        maxX = Math.max(maxX, n.position.x + CHILD_DEFAULT_WIDTH);
        maxY = Math.max(maxY, n.position.y + CHILD_DEFAULT_HEIGHT);
      }

      fitBounds(
        {
          x: minX - 50,
          y: minY - 50,
          width: maxX - minX + 100,
          height: maxY - minY + 100,
        },
        { padding: 0.2, duration: 500 },
      );
    };
    window.addEventListener("molecule:zoom-to-team", handler);
    return () => window.removeEventListener("molecule:zoom-to-team", handler);
  }, [fitBounds]);

  const onMoveEnd = useCallback(
    (_event: unknown, vp: { x: number; y: number; zoom: number }) => {
      clearTimeout(saveTimerRef.current);
      saveTimerRef.current = setTimeout(() => {
        saveViewport(vp.x, vp.y, vp.zoom);
      }, 1000);
    },
    [saveViewport],
  );

  return { onMoveEnd };
}
