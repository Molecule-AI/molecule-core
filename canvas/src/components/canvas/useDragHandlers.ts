"use client";

import { useCallback, useRef, useState } from "react";
import {
  useReactFlow,
  type Node,
  type OnNodeDrag,
} from "@xyflow/react";
import { useCanvasStore, type WorkspaceNodeData } from "@/store/canvas";
import { clampChildIntoParent, shouldDetach } from "./dragUtils";

type WorkspaceNode = Node<WorkspaceNodeData>;

export interface PendingNestState {
  nodeId: string;
  targetId: string | null;
  nodeName: string;
  targetName: string;
}

interface DragHandlers {
  onNodeDragStart: OnNodeDrag<Node<WorkspaceNodeData>>;
  onNodeDrag: OnNodeDrag<Node<WorkspaceNodeData>>;
  onNodeDragStop: OnNodeDrag<Node<WorkspaceNodeData>>;
  pendingNest: PendingNestState | null;
  confirmNest: () => void;
  cancelNest: () => void;
}



/**
 * Encapsulates every drag gesture on the canvas:
 *
 *   - On drag start, snapshot the modifier keys (Alt / Cmd-Meta) and
 *     remember which parent the node lived in so we can detect a
 *     re-parent on release.
 *   - On drag (mousemove), compute the best drop target via an
 *     absolute-bounds hit test and publish it via setDragOverNode so
 *     WorkspaceNode can render the highlight + DropTargetBadge can
 *     render its label + ghost preview.
 *   - On drag stop, decide one of: nest into new parent, un-nest, soft
 *     clamp back inside current parent, or plain move — based on
 *     modifier keys and hysteresis. Persist the absolute position,
 *     then run one commit-on-release grow pass on the parent chain.
 */
export function useDragHandlers(): DragHandlers {
  const setDragOverNode = useCanvasStore((s) => s.setDragOverNode);
  const savePosition = useCanvasStore((s) => s.savePosition);
  const nestNode = useCanvasStore((s) => s.nestNode);
  const batchNest = useCanvasStore((s) => s.batchNest);
  const isDescendant = useCanvasStore((s) => s.isDescendant);
  const { getInternalNode } = useReactFlow();

  const dragModifiersRef = useRef<{ alt: boolean; meta: boolean }>({
    alt: false,
    meta: false,
  });
  // Remember where the dragged node started so we can put it back on
  // cancel. React Flow tracks only the current position during drag;
  // if the user drags out → "Extract?" dialog → Cancel, we want the
  // card to go back inside its parent at its original coords rather
  // than stay dangling at the cancel-time position.
  const dragStartStateRef = useRef<{
    nodeId: string;
    parentId: string | null;
    position: { x: number; y: number };
  } | null>(null);
  const [pendingNest, setPendingNest] = useState<PendingNestState | null>(null);

  // Absolute-bounds hit test. Tiebreakers in order: highest zIndex
  // first (matches what the user sees in front after Cmd+] reorder),
  // deepest tree depth second, smallest area third. Depths are
  // pre-computed once per call so the whole pass stays O(n).
  const findDropTarget = useCallback(
    (draggedId: string, point: { x: number; y: number }): string | null => {
      const all = useCanvasStore.getState().nodes;
      const depthById = new Map<string, number>();
      for (const n of all) {
        depthById.set(
          n.id,
          n.data.parentId ? (depthById.get(n.data.parentId) ?? 0) + 1 : 0,
        );
      }
      let best:
        | { id: string; depth: number; zIndex: number; area: number }
        | null = null;
      for (const n of all) {
        if (n.id === draggedId || isDescendant(draggedId, n.id)) continue;
        const internal = getInternalNode(n.id);
        if (!internal) continue;
        const abs = internal.internals.positionAbsolute;
        const w = internal.measured?.width ?? n.width ?? 220;
        const h = internal.measured?.height ?? n.height ?? 120;
        if (point.x < abs.x || point.x > abs.x + w) continue;
        if (point.y < abs.y || point.y > abs.y + h) continue;
        const depth = depthById.get(n.id) ?? 0;
        const z = n.zIndex ?? 0;
        const area = w * h;
        if (
          !best ||
          z > best.zIndex ||
          (z === best.zIndex && depth > best.depth) ||
          (z === best.zIndex && depth === best.depth && area < best.area)
        ) {
          best = { id: n.id, depth, zIndex: z, area };
        }
      }
      return best?.id ?? null;
    },
    [getInternalNode, isDescendant],
  );

  const onNodeDragStart: OnNodeDrag<WorkspaceNode> = useCallback(
    (event, node) => {
      dragModifiersRef.current = {
        alt: event.altKey,
        meta: event.metaKey || event.ctrlKey,
      };
      dragStartStateRef.current = {
        nodeId: node.id,
        parentId: node.data.parentId,
        position: { x: node.position.x, y: node.position.y },
      };
    },
    [],
  );

  const onNodeDrag: OnNodeDrag<WorkspaceNode> = useCallback(
    (event, node) => {
      dragModifiersRef.current = {
        alt: event.altKey,
        meta: event.metaKey || event.ctrlKey,
      };
      const internal = getInternalNode(node.id);
      if (!internal) {
        setDragOverNode(null);
        return;
      }
      const abs = internal.internals.positionAbsolute;
      const w = internal.measured?.width ?? 220;
      const h = internal.measured?.height ?? 120;
      const center = { x: abs.x + w / 2, y: abs.y + h / 2 };
      setDragOverNode(findDropTarget(node.id, center));
    },
    [findDropTarget, getInternalNode, setDragOverNode],
  );

  const onNodeDragStop: OnNodeDrag<WorkspaceNode> = useCallback(
    (event, node) => {
      const { dragOverNodeId, nodes: allNodes } = useCanvasStore.getState();
      setDragOverNode(null);

      const nodeName = node.data.name;
      const currentParentId = node.data.parentId;
      const forceDetach =
        event.metaKey || event.ctrlKey || dragModifiersRef.current.meta;
      const droppingIntoAnotherParent =
        !!dragOverNodeId && dragOverNodeId !== currentParentId;
      // Past the 20 %-overlap hysteresis? Treat the gesture as a
      // deliberate drag-out. Below that threshold we soft-clamp the
      // child back inside so a twitchy release doesn't un-nest
      // accidentally (same intent as before, just: plain drag works
      // without a modifier now).
      const pastHysteresis =
        !!currentParentId &&
        shouldDetach(node.id, currentParentId, getInternalNode);

      if (droppingIntoAnotherParent) {
        // Explicit drop onto another workspace always wins over
        // clamp/detach — the user pointed at a new target.
        const targetNode = allNodes.find((n) => n.id === dragOverNodeId);
        const targetName = targetNode?.data.name || "Unknown";
        setPendingNest({
          nodeId: node.id,
          targetId: dragOverNodeId,
          nodeName,
          targetName,
        });
      } else if (currentParentId && (forceDetach || pastHysteresis)) {
        // Dragged past the edge (or Cmd-held as a force override): the
        // user wants out of the parent. Confirm the un-nest.
        const parentNode = allNodes.find((n) => n.id === currentParentId);
        const parentName = parentNode?.data.name || "Unknown";
        setPendingNest({
          nodeId: node.id,
          targetId: null,
          nodeName,
          targetName: parentName,
        });
      } else if (currentParentId) {
        // Still inside parent but the drag ended slightly past the
        // edge (under 20 % outside). Snap back in so the card doesn't
        // visually spill — Miro frame behaviour.
        clampChildIntoParent(node.id, currentParentId, getInternalNode);
      }

      const internal = getInternalNode(node.id);
      const abs = internal?.internals.positionAbsolute ?? node.position;
      savePosition(node.id, abs.x, abs.y);
      useCanvasStore.getState().growParentsToFitChildren();
    },
    [getInternalNode, savePosition, setDragOverNode],
  );

  const confirmNest = useCallback(() => {
    if (!pendingNest) return;
    // Close the dialog before dispatching the async store action so a
    // second drag can't kick off a competing batch while this one is
    // still mid-flight. The store actions surface their own errors via
    // showToast, so `void` is the right pattern here.
    const pending = pendingNest;
    setPendingNest(null);
    dragStartStateRef.current = null;
    const state = useCanvasStore.getState();
    if (
      state.selectedNodeIds.size > 1 &&
      state.selectedNodeIds.has(pending.nodeId)
    ) {
      void batchNest(Array.from(state.selectedNodeIds), pending.targetId);
    } else {
      void nestNode(pending.nodeId, pending.targetId);
    }
  }, [pendingNest, nestNode, batchNest]);

  const cancelNest = useCallback(() => {
    // Restore the dragged card to wherever it started. Without this,
    // a user who drags a child out of a parent then clicks Cancel
    // leaves the card stranded outside the parent with no visual
    // parent link — a state that doesn't match any save-backed
    // truth (the DB position was already written on drag-stop).
    const start = dragStartStateRef.current;
    if (start) {
      const { nodes } = useCanvasStore.getState();
      useCanvasStore.setState({
        nodes: nodes.map((n) =>
          n.id === start.nodeId
            ? { ...n, position: start.position }
            : n,
        ),
      });
      // Write the restore back to the DB so a reload shows the same
      // position. Convert the stored relative position back to absolute
      // via the parent's absolute origin before saving.
      const parent = start.parentId
        ? nodes.find((n) => n.id === start.parentId)
        : null;
      const parentInternal = start.parentId
        ? getInternalNode(start.parentId)
        : null;
      const parentAbs = parentInternal?.internals.positionAbsolute ?? {
        x: parent?.position.x ?? 0,
        y: parent?.position.y ?? 0,
      };
      savePosition(
        start.nodeId,
        start.position.x + parentAbs.x,
        start.position.y + parentAbs.y,
      );
    }
    dragStartStateRef.current = null;
    setPendingNest(null);
  }, [getInternalNode, savePosition]);

  return {
    onNodeDragStart,
    onNodeDrag,
    onNodeDragStop,
    pendingNest,
    confirmNest,
    cancelNest,
  };
}
