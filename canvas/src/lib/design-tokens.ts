export const STATUS_CONFIG: Record<string, { dot: string; glow: string; label: string; bar: string }> = {
  online: { dot: "bg-emerald-400", glow: "shadow-[0_0_10px_rgba(57,229,140,0.5)]", label: "Online", bar: "from-emerald-500/25 to-transparent" },
  offline: { dot: "bg-slate-500", glow: "", label: "Offline", bar: "from-slate-600/10 to-transparent" },
  paused: { dot: "bg-indigo-400", glow: "", label: "Paused", bar: "from-indigo-500/15 to-transparent" },
  degraded: { dot: "bg-amber-400", glow: "shadow-[0_0_10px_rgba(251,191,36,0.4)]", label: "Degraded", bar: "from-amber-500/25 to-transparent" },
  failed: { dot: "bg-red-400", glow: "shadow-[0_0_10px_rgba(248,113,113,0.4)]", label: "Failed", bar: "from-red-500/25 to-transparent" },
  provisioning: { dot: "bg-cyan-400 motion-safe:animate-pulse", glow: "shadow-[0_0_10px_rgba(34,209,238,0.4)]", label: "Starting", bar: "from-cyan-500/25 to-transparent" },
};

export function statusDotClass(status: string): string {
  return STATUS_CONFIG[status]?.dot ?? "bg-zinc-500";
}

export const TIER_CONFIG: Record<number, { label: string; color: string; border: string }> = {
  1: { label: "T1", color: "text-slate-400 bg-slate-800/60", border: "text-slate-400 border-slate-600/50" },
  2: { label: "T2", color: "text-cyan-400 bg-cyan-950/40 border border-cyan-500/20", border: "text-cyan-400 border-cyan-500/30" },
  3: { label: "T3", color: "text-violet-400 bg-violet-950/40 border border-violet-500/20", border: "text-violet-400 border-violet-500/30" },
  4: { label: "T4", color: "text-amber-400 bg-amber-950/40 border border-amber-500/20", border: "text-amber-400 border-amber-500/30" },
};

export const COMM_TYPE_LABELS: Record<string, string> = {
  a2a_send: "sent",
  a2a_receive: "received",
  task_update: "task update",
};
