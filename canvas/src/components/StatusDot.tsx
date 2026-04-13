"use client";

export const STATUS_COLORS: Record<string, string> = {
  online: "bg-emerald-400",
  offline: "bg-zinc-500",
  paused: "bg-indigo-400",
  degraded: "bg-amber-400",
  failed: "bg-red-400",
  provisioning: "bg-sky-400 animate-pulse",
};

export function StatusDot({
  status,
  size = "sm",
}: {
  status: string;
  size?: "sm" | "md";
}) {
  const sizeClass = size === "md" ? "w-2.5 h-2.5" : "w-2 h-2";
  const glowClass = status === "online" ? "shadow-sm shadow-emerald-400/50" : "";
  return (
    <div
      className={`${sizeClass} rounded-full shrink-0 ${STATUS_COLORS[status] || "bg-zinc-600"} ${glowClass}`}
    />
  );
}
