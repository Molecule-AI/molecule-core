"use client";

import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { api } from "@/lib/api";
import { showToast } from "@/components/Toaster";

interface Props {
  workspaceId: string;
  workspaceName?: string;
  open: boolean;
  onClose: () => void;
}

interface ConsoleResponse {
  output: string;
  instance_id?: string;
}

// ConsoleModal renders the EC2 serial console output for a workspace.
// Used by the "View Logs" button on failed/stuck workspaces so operators
// can see the actual cloud-init + runtime startup trace without SSH or
// AWS console access. The tenant platform proxies to the control plane;
// this component just consumes GET /workspaces/:id/console.
export function ConsoleModal({ workspaceId, workspaceName, open, onClose }: Props) {
  const [output, setOutput] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [mounted, setMounted] = useState(false);
  const closeButtonRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    setMounted(true);
  }, []);

  // Focus close button when modal opens
  useEffect(() => {
    if (!open) return;
    const raf = requestAnimationFrame(() => {
      closeButtonRef.current?.focus();
    });
    return () => cancelAnimationFrame(raf);
  }, [open]);

  useEffect(() => {
    if (!open) return;
    let ignore = false;
    setLoading(true);
    setError(null);
    setOutput(null);
    api
      .get<ConsoleResponse>(`/workspaces/${workspaceId}/console`)
      .then((data) => {
        if (ignore) return;
        setOutput(data.output || "");
      })
      .catch((e) => {
        if (ignore) return;
        // 501 = deployment without a control plane (local docker-compose).
        // 404 = EC2 instance has been terminated. Match with word-boundary
        // regex so a status code appearing inside an unrelated number
        // ("15012") doesn't false-match.
        const msg = e instanceof Error ? e.message : "Failed to load console output";
        if (/\b501\b/.test(msg)) {
          setError("Console output is only available on cloud (SaaS) deployments.");
        } else if (/\b404\b/.test(msg)) {
          setError("No EC2 instance found for this workspace — it may have been terminated.");
        } else {
          setError(msg);
        }
      })
      .finally(() => {
        if (!ignore) setLoading(false);
      });
    return () => {
      ignore = true;
    };
  }, [open, workspaceId]);

  useEffect(() => {
    if (!open) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [open, onClose]);

  if (!open || !mounted) return null;

  return createPortal(
    <div className="fixed inset-0 z-[9999] flex items-center justify-center">
      <div aria-hidden="true" className="absolute inset-0 bg-black/70 backdrop-blur-sm" onClick={onClose} />
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="console-modal-title"
        className="relative bg-zinc-950 border border-zinc-800 rounded-xl shadow-2xl w-[min(900px,90vw)] h-[min(70vh,700px)] flex flex-col overflow-hidden"
      >
        <div className="flex items-center justify-between px-4 py-3 border-b border-zinc-800">
          <div>
            <h3 id="console-modal-title" className="text-sm font-semibold text-zinc-100">
              EC2 console output
            </h3>
            {workspaceName && (
              <div className="text-[11px] text-zinc-500 mt-0.5 truncate max-w-[600px]">
                {workspaceName}
              </div>
            )}
          </div>
          <button
            type="button"
            ref={closeButtonRef}
            onClick={onClose}
            aria-label="Close"
            className="text-zinc-400 hover:text-zinc-100 text-sm px-2"
          >
            ✕
          </button>
        </div>

        <div className="flex-1 overflow-auto bg-black/80 p-4">
          {loading && (
            <div className="text-[12px] text-zinc-500" data-testid="console-loading">
              Loading console output…
            </div>
          )}
          {!loading && error && (
            <div
              role="alert"
              aria-live="assertive"
              className="text-[12px] text-amber-300 bg-amber-950/30 border border-amber-900/40 rounded px-3 py-2"
              data-testid="console-error"
            >
              {error}
            </div>
          )}
          {!loading && !error && output !== null && (
            <pre
              className="text-[11px] text-zinc-300 font-mono whitespace-pre-wrap break-all leading-tight"
              data-testid="console-output"
            >
              {output || "(console output is empty — the instance may still be booting)"}
            </pre>
          )}
        </div>

        <div className="flex items-center justify-end gap-2 px-4 py-3 border-t border-zinc-800 bg-zinc-900/40">
          {output && (
            <button
              type="button"
              onClick={() => {
                if (navigator.clipboard) {
                  navigator.clipboard.writeText(output);
                } else {
                  showToast("Copy requires HTTPS — please select and copy manually", "info");
                }
              }}
              className="px-3 py-1.5 text-[11px] text-zinc-400 hover:text-zinc-200 bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 rounded-lg transition-colors"
            >
              Copy
            </button>
          )}
          <button
            type="button"
            onClick={onClose}
            className="px-3 py-1.5 text-[11px] text-zinc-300 bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 rounded-lg transition-colors"
          >
            Close
          </button>
        </div>
      </div>
    </div>,
    document.body,
  );
}
