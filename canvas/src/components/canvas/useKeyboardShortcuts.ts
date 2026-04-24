"use client";

import { useEffect } from "react";
import { useCanvasStore } from "@/store/canvas";

/**
 * Canvas-wide keyboard shortcuts. All bound to the document window so
 * they work regardless of focused node, except when the user is typing
 * into an input (`inInput` short-circuits handling).
 *
 *   Esc                  — close context menu, clear selection, deselect
 *   Enter                — descend into selected node's first child
 *   Shift+Enter          — ascend to selected node's parent
 *   Cmd/Ctrl+]           — bump selected node forward in z-order
 *   Cmd/Ctrl+[           — bump selected node backward in z-order
 *   Z                    — zoom-to-team if the selected node has children
 */
export function useKeyboardShortcuts() {
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      const tag = (e.target as HTMLElement).tagName;
      const inInput =
        tag === "INPUT" ||
        tag === "TEXTAREA" ||
        tag === "SELECT" ||
        (e.target as HTMLElement).isContentEditable;

      if (e.key === "Escape") {
        const state = useCanvasStore.getState();
        if (state.contextMenu) {
          state.closeContextMenu();
        } else if (state.selectedNodeIds.size > 0) {
          state.clearSelection();
        } else if (state.selectedNodeId) {
          state.selectNode(null);
        }
      }

      // Figma-style hierarchy navigation. Skipped when the user is
      // typing so Enter can still submit forms.
      if (!inInput && (e.key === "Enter" || e.key === "NumpadEnter")) {
        e.preventDefault();
        const state = useCanvasStore.getState();
        const id = state.selectedNodeId;
        if (!id) return;
        if (e.shiftKey) {
          const sel = state.nodes.find((n) => n.id === id);
          const parentId = sel?.data.parentId ?? null;
          if (parentId) state.selectNode(parentId);
        } else {
          const firstChild = state.nodes.find((n) => n.data.parentId === id);
          if (firstChild) state.selectNode(firstChild.id);
        }
      }

      if (
        !inInput &&
        (e.metaKey || e.ctrlKey) &&
        (e.key === "]" || e.key === "[")
      ) {
        e.preventDefault();
        const state = useCanvasStore.getState();
        const id = state.selectedNodeId;
        if (!id) return;
        state.bumpZOrder(id, e.key === "]" ? 1 : -1);
      }

      if (!inInput && (e.key === "z" || e.key === "Z")) {
        const state = useCanvasStore.getState();
        const selectedId = state.selectedNodeId;
        if (!selectedId) return;
        const hasChildren = state.nodes.some(
          (n) => n.data.parentId === selectedId,
        );
        if (hasChildren) {
          window.dispatchEvent(
            new CustomEvent("molecule:zoom-to-team", {
              detail: { nodeId: selectedId },
            }),
          );
        }
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, []);
}
