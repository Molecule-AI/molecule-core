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
  // Hide the Cancel button for single-action info toasts.
  // onCancel is still invoked on Esc / backdrop-click, so when using this
  // dialog as a simple info toast the caller should pass the SAME handler
  // for both `onConfirm` and `onCancel` — otherwise dismissing via Esc /
  // backdrop click will run different logic than clicking the OK button,
  // which is almost never what you want for an info dialog.
  singleButton?: boolean;
}

export function ConfirmDialog({
  open,
  title,
  message,
  confirmLabel = "Confirm",
  confirmVariant = "primary",
  onConfirm,
  onCancel,
  singleButton = false,
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

  // Move focus into the dialog when it opens (WCAG 2.1 SC 2.4.3 / 3.2.2)
  useEffect(() => {
    if (!open || !mounted) return;
    const raf = requestAnimationFrame(() => {
      dialogRef.current?.querySelector<HTMLElement>("button")?.focus();
    });
    return () => cancelAnimationFrame(raf);
  }, [open, mounted]);

  // Keyboard: Escape cancels, Enter confirms, Tab is trapped within the dialog
  useEffect(() => {
    if (!open) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onCancelRef.current();
        return;
      }
      if (e.key === "Enter") {
        onConfirmRef.current();
        return;
      }
      if (e.key === "Tab" && dialogRef.current) {
        const focusable = Array.from(
          dialogRef.current.querySelectorAll<HTMLElement>(
            'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
          )
        ).filter((el) => !el.hasAttribute("disabled"));
        if (focusable.length === 0) { e.preventDefault(); return; }
        const first = focusable[0];
        const last = focusable[focusable.length - 1];
        if (e.shiftKey) {
          if (document.activeElement === first) {
            e.preventDefault();
            last.focus();
          }
        } else {
          if (document.activeElement === last) {
            e.preventDefault();
            first.focus();
          }
        }
      }
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

      {/* Dialog — role="dialog" + aria-modal prevent interaction with background */}
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="confirm-dialog-title"
        className="relative bg-zinc-900 border border-zinc-700 rounded-xl shadow-2xl shadow-black/50 max-w-[380px] w-full mx-4 overflow-hidden"
      >
        <div className="px-5 py-4">
          <h3 id="confirm-dialog-title" className="text-sm font-semibold text-zinc-100 mb-2">{title}</h3>
          <p className="text-[13px] text-zinc-400 leading-relaxed">{message}</p>
        </div>

        <div className="flex items-center justify-end gap-2 px-5 py-3 border-t border-zinc-800 bg-zinc-950/50">
          {!singleButton && (
            <button
              type="button"
              onClick={onCancel}
              className="px-3.5 py-1.5 text-[13px] text-zinc-400 hover:text-zinc-200 bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 rounded-lg transition-colors"
            >
              Cancel
            </button>
          )}
          <button
            type="button"
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
