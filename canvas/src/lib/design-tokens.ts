export const STATUS_CONFIG: Record<string, { dot: string; glow: string; label: string; bar: string }> = {
  online: { dot: "bg-emerald-400", glow: "shadow-emerald-400/50", label: "Online", bar: "from-emerald-500/20 to-transparent" },
  offline: { dot: "bg-zinc-500", glow: "", label: "Offline", bar: "from-zinc-600/10 to-transparent" },
  paused: { dot: "bg-indigo-400", glow: "", label: "Paused", bar: "from-indigo-500/10 to-transparent" },
  degraded: { dot: "bg-amber-400", glow: "shadow-amber-400/50", label: "Degraded", bar: "from-amber-500/20 to-transparent" },
  failed: { dot: "bg-red-400", glow: "shadow-red-400/50", label: "Failed", bar: "from-red-500/20 to-transparent" },
  provisioning: { dot: "bg-sky-400 motion-safe:animate-pulse", glow: "shadow-sky-400/50", label: "Starting", bar: "from-sky-500/20 to-transparent" },
};

export const TIER_CONFIG: Record<number, { label: string; color: string }> = {
  1: { label: "T1", color: "text-zinc-500 bg-zinc-800/80" },
  2: { label: "T2", color: "text-sky-400 bg-sky-950/50" },
  3: { label: "T3", color: "text-violet-400 bg-violet-950/50" },
  4: { label: "T4", color: "text-amber-400 bg-amber-950/50" },
};

export const TIER_COLORS: Record<number, string> = {
  1: "text-zinc-400 border-zinc-700/60",
  2: "text-sky-400 border-sky-500/30",
  3: "text-violet-400 border-violet-500/30",
  4: "text-amber-400 border-amber-500/30",
};
