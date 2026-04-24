"use client";

import { useCallback, useEffect, useRef } from "react";
import { useReactFlow } from "@xyflow/react";
import { useCanvasStore } from "@/store/canvas";
import { appendClass, removeClass } from "@/store/classNames";
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
  const autoFitTimerRef = useRef<ReturnType<typeof setTimeout>>(undefined);
  // Tracks whether any workspace was provisioning on the previous
  // render so we can detect the boundary when the last one finishes
  // and auto-fit the viewport around the whole tree.
  const hadProvisioningRef = useRef(false);
  // Respect-user-pan gate for the deploy-time auto-fit. Earlier
  // revisions tried to detect user pans via `onMoveEnd`, but React
  // Flow v12 fires that callback with a truthy event at the END of
  // a programmatic fitView animation — so the first auto-fit we
  // triggered would immediately look like a user pan and block
  // every subsequent fit for the rest of the deploy, leaving the
  // viewport stuck wherever the first fit landed. Now we stamp
  // this ref ONLY on wheel / pointerdown / touchstart on the
  // React Flow pane itself (see the effect below), which are
  // unambiguous user-gesture signals.
  const userPannedAtRef = useRef<number | null>(null);
  const lastAutoFitAtRef = useRef(0);

  useEffect(() => {
    return () => {
      clearTimeout(saveTimerRef.current);
      clearTimeout(panTimerRef.current);
      clearTimeout(autoFitTimerRef.current);
    };
  }, []);

  // User-gesture listeners for the respect-user-pan gate. Listens on
  // `document` with capture phase and filters to events whose target
  // lies inside the React Flow pane — this avoids a mount-order race
  // (`.react-flow__pane` may not exist when the hook first runs if
  // RF is behind a Suspense boundary) AND keeps clicks on the
  // toolbar / modals / side panel from stamping user-pan-intent.
  // Capture phase runs before target-phase `stopPropagation` so a
  // handler elsewhere can't swallow the signal. `pointerdown` covers
  // mouse + touch + pen on every modern browser — no separate
  // `touchstart` listener needed (would double-stamp on mobile).
  useEffect(() => {
    if (typeof window === "undefined") return;
    const stamp = (e: Event) => {
      const target = e.target as HTMLElement | null;
      if (!target?.closest?.(".react-flow__pane")) return;
      userPannedAtRef.current = Date.now();
    };
    const opts: AddEventListenerOptions = { passive: true, capture: true };
    const targets: Array<keyof DocumentEventMap> = ["wheel", "pointerdown"];
    for (const ev of targets) {
      document.addEventListener(ev, stamp, opts);
    }
    return () => {
      for (const ev of targets) {
        document.removeEventListener(ev, stamp, opts);
      }
    };
  }, []);

  // Auto-fit the viewport once all workspaces finish provisioning. Org
  // imports land dozens of new nodes off-screen; without a follow-up
  // fit, the user has to manually pan + zoom to find what they just
  // created. Only fires when TRANSITIONING from some-provisioning to
  // zero-provisioning — not on every re-render.
  const provisioningCount = useCanvasStore(
    (s) => s.nodes.filter((n) => n.data.status === "provisioning").length,
  );
  const nodeCount = useCanvasStore((s) => s.nodes.length);

  useEffect(() => {
    const hasProvisioning = provisioningCount > 0;
    const wasProvisioning = hadProvisioningRef.current;
    hadProvisioningRef.current = hasProvisioning;

    if (wasProvisioning && !hasProvisioning && nodeCount > 0) {
      // Root-complete moment — every root that has children just
      // finished deploying. Pop + glow once (mol-deploy-root-complete)
      // then auto-fit the viewport around the whole org. Leaf-only
      // roots (single workspaces with no children) are skipped so the
      // effect reads as "your org landed" not "random card flickered".
      const state = useCanvasStore.getState();
      const rootsWithChildren = new Set<string>();
      for (const n of state.nodes) {
        if (n.data.parentId) continue;
        if (state.nodes.some((c) => c.data.parentId === n.id)) {
          rootsWithChildren.add(n.id);
        }
      }
      if (rootsWithChildren.size > 0) {
        useCanvasStore.setState({
          nodes: state.nodes.map((n) =>
            rootsWithChildren.has(n.id)
              ? { ...n, className: appendClass(n.className, "mol-deploy-root-complete") }
              : n,
          ),
        });
        // Strip the one-shot class after the keyframe ends so a later
        // deploy on the same node can fire it again.
        window.setTimeout(() => {
          const s = useCanvasStore.getState();
          useCanvasStore.setState({
            nodes: s.nodes.map((n) =>
              rootsWithChildren.has(n.id)
                ? { ...n, className: removeClass(n.className, "mol-deploy-root-complete") }
                : n,
            ),
          });
        }, 800);
      }

      clearTimeout(autoFitTimerRef.current);
      // 1200ms settle delay: lets React Flow's DOM measurement pass
      // resize newly-online parents before we compute bounds.
      // Measuring too early gives us the pre-render skeleton bbox and
      // fitView zooms to that smaller-than-real rectangle.
      autoFitTimerRef.current = setTimeout(() => {
        fitView({
          // Deliberately SLOWER than the in-flight tracking fits
          // (400ms). The asymmetry reads as "settling" on the
          // finished org rather than "tracking" another arrival,
          // which is the intended UX for the "deploy done" moment.
          // Don't normalize these two durations to the same value.
          duration: 1200,
          // Match the deploy-time fit padding (0.45) so end-state
          // and in-flight state use the same framing — otherwise
          // the final zoom-out "jumps" relative to the intermediate
          // fits and looks like a mis-layout.
          padding: 0.45,
          // Cap zoom-in: a small tree (2-3 nodes) would otherwise end
          // up at the 2x maxZoom, visually implying "something is
          // wrong". 0.65 reads like "here's your whole org" even when
          // the tree is small — matches deploy-time cap.
          maxZoom: 0.65,
          // Cap zoom-out: fitView would fall back to the component's
          // minZoom=0.1 on a sparse/outlier layout, leaving the user
          // staring at a postage-stamp canvas. 0.25 is the floor.
          minZoom: 0.25,
        });
      }, 1200);
    }
  }, [provisioningCount, nodeCount, fitView]);

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

  // Auto pan+zoom to the whole deploying org after each child
  // arrival — DEBOUNCED. Firing fitView on every event with a
  // 600ms animation meant rapid sibling arrivals (server paces 2s
  // apart, HMR bursts can land faster) made the viewport lurch
  // continuously, which the user read as "parent flashing around".
  // We now wait until the arrivals GO QUIET for 500ms, then run
  // exactly one fit. The rootId we captured on the most recent
  // event drives the fit bounds. Respect-user-pan still short-
  // circuits: if the user moved after our last auto-fit, we never
  // fit again this deploy.
  const pendingFitRootRef = useRef<string | null>(null);
  useEffect(() => {
    const runFit = () => {
      const rootCandidate = pendingFitRootRef.current;
      pendingFitRootRef.current = null;
      if (!rootCandidate) return;
      if (
        userPannedAtRef.current !== null &&
        userPannedAtRef.current > lastAutoFitAtRef.current
      ) {
        return;
      }
      const state = useCanvasStore.getState();
      // Climb to the true root — the event's rootId is the just-
      // landed child's direct parent, which may itself be nested.
      let topId = rootCandidate;
      let cursor = state.nodes.find((n) => n.id === topId);
      while (cursor?.data.parentId) {
        const up = state.nodes.find((n) => n.id === cursor!.data.parentId);
        if (!up) break;
        cursor = up;
        topId = up.id;
      }
      const subtree: string[] = [];
      const stack = [topId];
      while (stack.length) {
        const id = stack.pop()!;
        subtree.push(id);
        for (const n of state.nodes) {
          if (n.data.parentId === id) stack.push(n.id);
        }
      }
      if (subtree.length === 0) return;
      fitView({
        nodes: subtree.map((id) => ({ id })),
        // Short animation — server paces children ~2s apart, so a
        // 400ms fit animation reads as "smoothly tracked" rather
        // than "constantly lurching". Longer durations (the earlier
        // 600ms) start to overlap if the user re-triggers deploys.
        duration: 400,
        // Generous padding so the right-hand Communications panel,
        // bottom-left Legend, and bottom-right "New Workspace"
        // button don't cover the outer cards. React Flow padding
        // is a fraction of viewport dims, so 0.45 ≈ ~430px of
        // margin on a 960-wide canvas — enough clearance for the
        // two side panels (~300px + ~280px).
        padding: 0.45,
        // Lower maxZoom so small orgs (2-3 cards) still zoom out
        // enough to show the parent frame + children clearly with
        // the padded margins. 0.65 reads as "here's the whole org"
        // without getting dragged to the maxZoom by fitView's
        // "fill the viewport" default.
        maxZoom: 0.65,
        minZoom: 0.25,
      });
      lastAutoFitAtRef.current = Date.now();
    };
    const handler = (e: Event) => {
      const { rootId } = (e as CustomEvent<{ rootId: string }>).detail;
      // Keep the most recently-requested root. Back-to-back imports
      // on two different orgs (rare — user would have to click
      // Import twice within 500ms) "later wins" the viewport rather
      // than ping-ponging between them. If this becomes a real
      // pattern we'd flush the pending fit synchronously when
      // `rootId` changes, rather than resetting the timer.
      pendingFitRootRef.current = rootId;
      clearTimeout(autoFitTimerRef.current);
      autoFitTimerRef.current = setTimeout(runFit, 500);
    };
    window.addEventListener("molecule:fit-deploying-org", handler);
    return () => window.removeEventListener("molecule:fit-deploying-org", handler);
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
      // User-pan detection moved to the wheel/pointerdown listener
      // above — onMoveEnd fires for programmatic fitView too, which
      // made this callback an unreliable source for user-intent
      // tracking. This now only handles the debounced viewport
      // save so a reload lands the user back where they were.
      clearTimeout(saveTimerRef.current);
      saveTimerRef.current = setTimeout(() => {
        saveViewport(vp.x, vp.y, vp.zoom);
      }, 1000);
    },
    [saveViewport],
  );

  return { onMoveEnd };
}
