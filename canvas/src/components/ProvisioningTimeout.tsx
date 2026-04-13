"use client";

import { useState, useEffect, useCallback, useRef, useMemo } from "react";
import { useCanvasStore, type WorkspaceNodeData } from "@/store/canvas";
import { api } from "@/lib/api";
import { showToast } from "./Toaster";

/** Default provisioning timeout in milliseconds (2 minutes). */
export const DEFAULT_PROVISION_TIMEOUT_MS = 120_000;

interface TimeoutEntry {
  workspaceId: string;
  workspaceName: string;
  startedAt: number;
}

/**
 * Monitors workspaces in "provisioning" status and shows a timeout banner
 * with recovery actions (Retry, Cancel, View Logs) when provisioning takes
 * too long.
 *
 * Rendered at the top of the canvas (inside Canvas component). Watches the
 * Zustand store for nodes with status === "provisioning" and tracks elapsed
 * time per node.
 */
export function ProvisioningTimeout({
  timeoutMs = DEFAULT_PROVISION_TIMEOUT_MS,
}: {
  timeoutMs?: number;
}) {
  const [timedOut, setTimedOut] = useState<TimeoutEntry[]>([]);
  const [retrying, setRetrying] = useState<Set<string>>(new Set());
  const [cancelling, setCancelling] = useState<Set<string>>(new Set());
  const trackingRef = useRef<Map<string, number>>(new Map());

  // Subscribe to provisioning nodes — use shallow compare to avoid infinite re-render
  // (filter+map creates new array reference on every store update)
  const provisioningNodes = useCanvasStore((s) => {
    const result = s.nodes
      .filter((n) => n.data.status === "provisioning")
      .map((n) => `${n.id}:${n.data.name}`);
    return result.join(",");
  });
  const parsedProvisioningNodes = useMemo(
    () =>
      provisioningNodes
        ? provisioningNodes.split(",").map((entry) => {
            const [id, name] = entry.split(":");
            return { id, name };
          })
        : [],
    [provisioningNodes],
  );

  useEffect(() => {
    const tracking = trackingRef.current;

    // Start tracking new provisioning nodes
    for (const node of parsedProvisioningNodes) {
      if (!tracking.has(node.id)) {
        tracking.set(node.id, Date.now());
      }
    }

    // Remove tracking for nodes that are no longer provisioning
    const activeIds = new Set(parsedProvisioningNodes.map((n) => n.id));
    for (const id of tracking.keys()) {
      if (!activeIds.has(id)) {
        tracking.delete(id);
      }
    }

    // Also remove from timedOut list if no longer provisioning
    setTimedOut((prev) => prev.filter((e) => activeIds.has(e.workspaceId)));

    // Interval to check for timeouts
    const interval = setInterval(() => {
      const now = Date.now();
      const newTimedOut: TimeoutEntry[] = [];

      for (const node of parsedProvisioningNodes) {
        const startedAt = tracking.get(node.id);
        if (startedAt && now - startedAt >= timeoutMs) {
          newTimedOut.push({
            workspaceId: node.id,
            workspaceName: node.name,
            startedAt,
          });
        }
      }

      if (newTimedOut.length > 0) {
        setTimedOut((prev) => {
          const existingIds = new Set(prev.map((e) => e.workspaceId));
          const additions = newTimedOut.filter(
            (e) => !existingIds.has(e.workspaceId),
          );
          return additions.length > 0 ? [...prev, ...additions] : prev;
        });
      }
    }, 5_000); // check every 5s

    return () => clearInterval(interval);
  }, [parsedProvisioningNodes, timeoutMs]);

  const RETRY_COOLDOWN_MS = 5_000;
  const [retryCooldown, setRetryCooldown] = useState<Set<string>>(new Set());

  const handleRetry = useCallback(async (workspaceId: string) => {
    setRetrying((prev) => new Set(prev).add(workspaceId));
    try {
      await api.post(`/workspaces/${workspaceId}/restart`);
      // Remove from timed-out list — tracking will restart when provisioning event comes in
      setTimedOut((prev) => prev.filter((e) => e.workspaceId !== workspaceId));
      trackingRef.current.delete(workspaceId);
      showToast("Retrying deployment...", "info");
    } catch (e) {
      showToast(
        e instanceof Error ? e.message : "Retry failed",
        "error",
      );
    } finally {
      setRetrying((prev) => {
        const next = new Set(prev);
        next.delete(workspaceId);
        return next;
      });
      // Start cooldown — disable retry button for 5s
      setRetryCooldown((prev) => new Set(prev).add(workspaceId));
      setTimeout(() => {
        setRetryCooldown((prev) => {
          const next = new Set(prev);
          next.delete(workspaceId);
          return next;
        });
      }, RETRY_COOLDOWN_MS);
    }
  }, []);

  const [confirmingCancel, setConfirmingCancel] = useState<string | null>(null);

  const handleCancelRequest = useCallback((workspaceId: string) => {
    setConfirmingCancel(workspaceId);
  }, []);

  const handleCancelConfirm = useCallback(async () => {
    if (!confirmingCancel) return;
    const workspaceId = confirmingCancel;
    setConfirmingCancel(null);
    setCancelling((prev) => new Set(prev).add(workspaceId));
    try {
      await api.del(`/workspaces/${workspaceId}`);
      setTimedOut((prev) => prev.filter((e) => e.workspaceId !== workspaceId));
      trackingRef.current.delete(workspaceId);
      showToast("Deployment cancelled", "info");
    } catch (e) {
      showToast(
        e instanceof Error ? e.message : "Cancel failed",
        "error",
      );
    } finally {
      setCancelling((prev) => {
        const next = new Set(prev);
        next.delete(workspaceId);
        return next;
      });
    }
  }, [confirmingCancel]);

  const handleViewLogs = useCallback((workspaceId: string) => {
    // Open the terminal tab for this workspace so user can see logs
    useCanvasStore.getState().selectNode(workspaceId);
    useCanvasStore.getState().setPanelTab("terminal");
  }, []);

  if (timedOut.length === 0) return null;

  return (
    <div role="alert" aria-live="assertive" className="fixed top-14 left-1/2 -translate-x-1/2 z-40 flex flex-col gap-2 max-w-[480px] w-full px-4">
      {timedOut.map((entry) => {
        const elapsed = Math.round((Date.now() - entry.startedAt) / 1000);
        const isRetrying = retrying.has(entry.workspaceId);
        const isCancelling = cancelling.has(entry.workspaceId);

        return (
          <div
            key={entry.workspaceId}
            className="bg-amber-950/90 border border-amber-700/40 rounded-xl px-4 py-3 shadow-2xl shadow-black/40 backdrop-blur-md"
          >
            <div className="flex items-start gap-3">
              {/* Warning icon */}
              <div className="w-8 h-8 rounded-lg bg-amber-600/20 border border-amber-500/30 flex items-center justify-center shrink-0 mt-0.5">
                <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
                  <path
                    d="M8 2L14 13H2L8 2Z"
                    stroke="#fbbf24"
                    strokeWidth="1.3"
                    strokeLinejoin="round"
                  />
                  <path d="M8 7V9.5" stroke="#fbbf24" strokeWidth="1.3" strokeLinecap="round" />
                  <circle cx="8" cy="11" r="0.6" fill="#fbbf24" />
                </svg>
              </div>

              <div className="flex-1 min-w-0">
                <div className="text-[12px] font-semibold text-amber-200 mb-0.5">
                  Provisioning Timeout
                </div>
                <div className="text-[11px] text-amber-300/80 leading-relaxed">
                  <span className="font-medium text-amber-200">{entry.workspaceName}</span>{" "}
                  has been provisioning for{" "}
                  <span className="font-mono text-amber-300">{formatDuration(elapsed)}</span>.
                  It may have encountered an issue.
                </div>

                {/* Action buttons */}
                <div className="flex items-center gap-2 mt-2.5">
                  <button
                    onClick={() => handleRetry(entry.workspaceId)}
                    disabled={isRetrying || isCancelling || retryCooldown.has(entry.workspaceId)}
                    className="px-3 py-1.5 bg-amber-600 hover:bg-amber-500 text-[11px] font-medium rounded-lg text-white disabled:opacity-40 transition-colors"
                  >
                    {isRetrying ? "Retrying..." : retryCooldown.has(entry.workspaceId) ? "Wait..." : "Retry"}
                  </button>
                  <button
                    onClick={() => handleCancelRequest(entry.workspaceId)}
                    disabled={isRetrying || isCancelling}
                    className="px-3 py-1.5 bg-zinc-800 hover:bg-zinc-700 text-[11px] text-zinc-300 rounded-lg border border-zinc-600 disabled:opacity-40 transition-colors"
                  >
                    {isCancelling ? "Cancelling..." : "Cancel"}
                  </button>
                  <button
                    onClick={() => handleViewLogs(entry.workspaceId)}
                    className="px-3 py-1.5 text-[11px] text-amber-400 hover:text-amber-300 transition-colors"
                  >
                    View Logs
                  </button>
                </div>
              </div>
            </div>
          </div>
        );
      })}

      {/* Cancel confirmation dialog */}
      {confirmingCancel && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 bg-black/60" onClick={() => setConfirmingCancel(null)} />
          <div className="relative bg-zinc-900 border border-zinc-700 rounded-xl shadow-2xl p-5 max-w-[340px] w-full mx-4">
            <h3 className="text-sm font-semibold text-zinc-100 mb-2">
              Cancel deployment?
            </h3>
            <p className="text-[12px] text-zinc-400 mb-4 leading-relaxed">
              This will permanently remove the workspace. This action cannot be undone.
            </p>
            <div className="flex justify-end gap-2">
              <button
                onClick={() => setConfirmingCancel(null)}
                className="px-3.5 py-1.5 text-[12px] text-zinc-400 hover:text-zinc-200 bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 rounded-lg transition-colors"
              >
                Keep
              </button>
              <button
                onClick={handleCancelConfirm}
                className="px-3.5 py-1.5 text-[12px] bg-red-600 hover:bg-red-500 text-white rounded-lg transition-colors"
              >
                Remove Workspace
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

/** Format seconds into a human-friendly string like "2m 30s" */
function formatDuration(seconds: number): string {
  if (seconds < 60) return `${seconds}s`;
  const mins = Math.floor(seconds / 60);
  const secs = seconds % 60;
  return secs > 0 ? `${mins}m ${secs}s` : `${mins}m`;
}
