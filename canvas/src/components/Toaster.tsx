"use client";

import { useEffect, useState } from "react";

interface Toast {
  id: string;
  message: string;
  type: "success" | "error" | "info";
}

let addToastFn: ((message: string, type?: Toast["type"]) => void) | null = null;

/** Call from anywhere to show a toast */
export function showToast(message: string, type: Toast["type"] = "info") {
  addToastFn?.(message, type);
}

export function Toaster() {
  const [toasts, setToasts] = useState<Toast[]>([]);

  useEffect(() => {
    addToastFn = (message, type = "info") => {
      const id = Math.random().toString(36).slice(2);
      setToasts((prev) => [...prev.slice(-4), { id, message, type }]);
      setTimeout(() => {
        setToasts((prev) => prev.filter((t) => t.id !== id));
      }, 4000);
    };
    return () => { addToastFn = null; };
  }, []);

  if (toasts.length === 0) return null;

  return (
    <div className="fixed bottom-16 left-1/2 -translate-x-1/2 z-[80] flex flex-col gap-2 items-center">
      {toasts.map((toast) => (
        <div
          key={toast.id}
          className={`px-4 py-2.5 rounded-xl shadow-2xl shadow-black/40 text-sm backdrop-blur-md animate-in slide-in-from-bottom duration-200 ${
            toast.type === "success"
              ? "bg-emerald-950/90 border border-emerald-700/40 text-emerald-200"
              : toast.type === "error"
              ? "bg-red-950/90 border border-red-700/40 text-red-200"
              : "bg-zinc-900/90 border border-zinc-700/40 text-zinc-200"
          }`}
        >
          {toast.message}
        </div>
      ))}
    </div>
  );
}
