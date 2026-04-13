"use client";

import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";

interface Props {
  open: boolean;
  title: string;
  message: string;
  confirmLabel?: string;
  confirmVariant?: "danger" | "primary" | "warning";
  onConfirm: () => void;
  onCancel: () => void;
}

export function ConfirmDialog({
  open,
  title,
  message,
  confirmLabel = "Confirm",
  confirmVariant = "primary",
  onConfirm,
  onCancel,
}: Props) {
  const dialogRef = useRef<HTMLDivElement>(null);
  const [mounted, setMounted] = useState(false);
  // Refs avoid re-binding the keydown handler on every parent render
  const onConfirmRef = useRef(onConfirm);
  const onCancelRef = useRef(onCancel);
  onConfirmRef.current = onConfirm;
  onCancelRef.current = onCancel;

  useEffect(() => {
    setMounted(true);
  }, []);

  useEffect(() => {
    if (!open) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") onCancelRef.current();
      if (e.key === "Enter") onConfirmRef.current();
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [open]);

  if (!open || !mounted) return null;

  const confirmColors =
    confirmVariant === "danger"
      ? "bg-red-600 hover:bg-red-500 text-white"
      : confirmVariant === "warning"
        ? "bg-amber-600 hover:bg-amber-500 text-white"
        : "bg-blue-600 hover:bg-blue-500 text-white";

  // Render via Portal so the fixed-position dialog escapes any containing block
  // (e.g. parents with transform, filter, will-change that break position:fixed).
  return createPortal(
    <div className="fixed inset-0 z-[9999] flex items-center justify-center">
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={onCancel} />

      {/* Dialog */}
      <div
        ref={dialogRef}
        className="relative bg-zinc-900 border border-zinc-700 rounded-xl shadow-2xl shadow-black/50 max-w-[380px] w-full mx-4 overflow-hidden"
      >
        <div className="px-5 py-4">
          <h3 className="text-sm font-semibold text-zinc-100 mb-2">{title}</h3>
          <p className="text-[13px] text-zinc-400 leading-relaxed">{message}</p>
        </div>

        <div className="flex items-center justify-end gap-2 px-5 py-3 border-t border-zinc-800 bg-zinc-950/50">
          <button
            onClick={onCancel}
            className="px-3.5 py-1.5 text-[13px] text-zinc-400 hover:text-zinc-200 bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 rounded-lg transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={onConfirm}
            className={`px-3.5 py-1.5 text-[13px] rounded-lg transition-colors ${confirmColors}`}
          >
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>,
    document.body
  );
}
