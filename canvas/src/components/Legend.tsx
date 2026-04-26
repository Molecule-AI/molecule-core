"use client";

import { useEffect, useState } from "react";
import { STATUS_CONFIG } from "@/lib/design-tokens";
import { useCanvasStore } from "@/store/canvas";

const LEGEND_STATUSES = ["online", "provisioning", "degraded", "failed", "paused", "offline"] as const;

// Persist the user's choice across sessions. Default is "open" so
// first-time users still see the symbol key; once dismissed we
// respect that until they explicitly reopen via the floating pill.
const STORAGE_KEY = "molecule.legend.open";

function readStoredOpen(): boolean {
  if (typeof window === "undefined") return true;
  try {
    const v = window.localStorage.getItem(STORAGE_KEY);
    if (v === null) return true;
    return v === "1";
  } catch {
    return true;
  }
}

function writeStoredOpen(open: boolean) {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(STORAGE_KEY, open ? "1" : "0");
  } catch {
    // localStorage can throw in private mode / quota / disabled
    // contexts. Silent fallback — the in-memory state still works
    // for the current session.
  }
}

export function Legend() {
  // TemplatePalette (when open) is fixed top-0 left-0 w-[280px] — the
  // default bottom-6 left-4 position of this legend would sit under it.
  // Shift past the 280 px palette + a 16 px gap when the palette is open.
  const paletteOpen = useCanvasStore((s) => s.templatePaletteOpen);
  const leftClass = paletteOpen ? "left-[296px]" : "left-4";

  // SSR-safe pattern: mount with the default (true) so first paint
  // matches the server output, then hydrate the persisted value
  // after mount. Avoids a hydration mismatch warning when the user
  // had previously closed the legend.
  const [open, setOpen] = useState(true);
  useEffect(() => {
    setOpen(readStoredOpen());
  }, []);

  const closeLegend = () => {
    setOpen(false);
    writeStoredOpen(false);
  };
  const openLegend = () => {
    setOpen(true);
    writeStoredOpen(true);
  };

  if (!open) {
    return (
      <button
        type="button"
        onClick={openLegend}
        aria-label="Show legend"
        title="Show legend"
        className={`fixed bottom-6 ${leftClass} z-30 flex items-center gap-1.5 rounded-full bg-zinc-900/95 border border-zinc-700/50 px-3 py-1.5 text-[11px] font-semibold text-zinc-400 uppercase tracking-wider shadow-xl shadow-black/30 backdrop-blur-sm hover:text-zinc-200 hover:border-zinc-600 transition-[left,colors] duration-200`}
      >
        <span aria-hidden="true" className="text-[10px]">ⓘ</span>
        Legend
      </button>
    );
  }

  return (
    <div className={`fixed bottom-6 ${leftClass} z-30 bg-zinc-900/95 border border-zinc-700/50 rounded-xl px-4 py-3 shadow-xl shadow-black/30 backdrop-blur-sm max-w-[280px] transition-[left] duration-200`}>
      <div className="flex items-start justify-between mb-2">
        <div className="text-[11px] font-semibold text-zinc-400 uppercase tracking-wider">Legend</div>
        <button
          type="button"
          onClick={closeLegend}
          aria-label="Hide legend"
          title="Hide legend"
          className="-mt-0.5 -mr-1 px-1.5 text-[14px] leading-none text-zinc-500 hover:text-zinc-200 transition-colors"
        >
          ×
        </button>
      </div>

      {/* Status */}
      <div className="mb-2">
        <div className="text-[11px] text-zinc-500 font-medium mb-1">Status</div>
        <div className="flex flex-wrap gap-x-3 gap-y-1">
          {LEGEND_STATUSES.map((s) => (
            <StatusItem key={s} color={STATUS_CONFIG[s].dot} label={STATUS_CONFIG[s].label} />
          ))}
        </div>
      </div>

      {/* Tiers */}
      <div className="mb-2">
        <div className="text-[11px] text-zinc-500 font-medium mb-1">Tier</div>
        <div className="flex flex-wrap gap-x-3 gap-y-1">
          <TierItem tier={1} label="Sandboxed" color="text-sky-300 bg-sky-950/40 border-sky-700/30" />
          <TierItem tier={2} label="Standard" color="text-violet-300 bg-violet-950/40 border-violet-700/30" />
          <TierItem tier={3} label="Full Access" color="text-amber-300 bg-amber-950/40 border-amber-700/30" />
        </div>
      </div>

      {/* Communication */}
      <div>
        <div className="text-[11px] text-zinc-500 font-medium mb-1">Communication</div>
        <div className="flex flex-wrap gap-x-3 gap-y-1">
          <CommItem icon="↗" color="text-cyan-400" label="A2A Out" />
          <CommItem icon="↙" color="text-blue-400" label="A2A In" />
          <CommItem icon="◆" color="text-amber-400" label="Task" />
          <CommItem icon="!" color="text-red-400" label="Error" />
        </div>
      </div>
    </div>
  );
}

function StatusItem({ color, label }: { color: string; label: string }) {
  return (
    <div className="flex items-center gap-1">
      <div className={`w-1.5 h-1.5 rounded-full ${color}`} />
      <span className="text-[11px] text-zinc-400">{label}</span>
    </div>
  );
}

function TierItem({ tier, label, color }: { tier: number; label: string; color: string }) {
  return (
    <div className="flex items-center gap-1">
      <span className={`text-[11px] font-mono px-1 py-0.5 rounded border ${color}`}>T{tier}</span>
      <span className="text-[11px] text-zinc-400">{label}</span>
    </div>
  );
}

function CommItem({ icon, color, label }: { icon: string; color: string; label: string }) {
  return (
    <div className="flex items-center gap-1">
      <span className={`text-[11px] ${color}`}>{icon}</span>
      <span className="text-[11px] text-zinc-400">{label}</span>
    </div>
  );
}
