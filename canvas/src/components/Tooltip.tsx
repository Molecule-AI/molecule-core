"use client";

import { useState, useRef, useEffect, useCallback, type ReactNode } from "react";
import { createPortal } from "react-dom";

interface Props {
  text: string;
  children: ReactNode;
}

export function Tooltip({ text, children }: Props) {
  const [show, setShow] = useState(false);
  const [pos, setPos] = useState({ x: 0, y: 0 });
  const timerRef = useRef<ReturnType<typeof setTimeout>>(undefined);
  const triggerRef = useRef<HTMLDivElement>(null);

  useEffect(() => () => clearTimeout(timerRef.current), []);

  const enter = useCallback(() => {
    timerRef.current = setTimeout(() => {
      if (triggerRef.current) {
        const rect = triggerRef.current.getBoundingClientRect();
        setPos({ x: rect.left, y: rect.top });
      }
      setShow(true);
    }, 400);
  }, []);

  const leave = useCallback(() => {
    clearTimeout(timerRef.current);
    setShow(false);
  }, []);

  return (
    <div ref={triggerRef} onMouseEnter={enter} onMouseLeave={leave}>
      {children}
      {show && text && createPortal(
        <div
          className="fixed z-[9999] max-w-[400px] max-h-[300px] overflow-y-auto px-3 py-2 bg-zinc-800 border border-zinc-600 rounded-lg shadow-2xl shadow-black/60 pointer-events-none"
          style={{ left: pos.x, top: Math.max(8, pos.y - 8), transform: "translateY(-100%)" }}
        >
          <div className="text-[11px] text-zinc-200 whitespace-pre-wrap break-words leading-relaxed">
            {text}
          </div>
        </div>,
        document.body
      )}
    </div>
  );
}
