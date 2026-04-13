"use client";

export function Legend() {
  return (
    <div className="fixed bottom-6 left-4 z-30 bg-zinc-900/95 border border-zinc-700/50 rounded-xl px-4 py-3 shadow-xl shadow-black/30 backdrop-blur-sm max-w-[280px]">
      <div className="text-[10px] font-semibold text-zinc-400 uppercase tracking-wider mb-2">Legend</div>

      {/* Status */}
      <div className="mb-2">
        <div className="text-[9px] text-zinc-500 font-medium mb-1">Status</div>
        <div className="flex flex-wrap gap-x-3 gap-y-1">
          <StatusItem color="bg-emerald-400" label="Online" />
          <StatusItem color="bg-sky-400 animate-pulse" label="Starting" />
          <StatusItem color="bg-amber-400" label="Degraded" />
          <StatusItem color="bg-red-400" label="Failed" />
          <StatusItem color="bg-indigo-400" label="Paused" />
          <StatusItem color="bg-zinc-500" label="Offline" />
        </div>
      </div>

      {/* Tiers */}
      <div className="mb-2">
        <div className="text-[9px] text-zinc-500 font-medium mb-1">Tier</div>
        <div className="flex flex-wrap gap-x-3 gap-y-1">
          <TierItem tier={1} label="Sandboxed" color="text-sky-300 bg-sky-950/40 border-sky-700/30" />
          <TierItem tier={2} label="Standard" color="text-violet-300 bg-violet-950/40 border-violet-700/30" />
          <TierItem tier={3} label="Full Access" color="text-amber-300 bg-amber-950/40 border-amber-700/30" />
        </div>
      </div>

      {/* Communication */}
      <div>
        <div className="text-[9px] text-zinc-500 font-medium mb-1">Communication</div>
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
      <span className="text-[8px] text-zinc-400">{label}</span>
    </div>
  );
}

function TierItem({ tier, label, color }: { tier: number; label: string; color: string }) {
  return (
    <div className="flex items-center gap-1">
      <span className={`text-[8px] font-mono px-1 py-0.5 rounded border ${color}`}>T{tier}</span>
      <span className="text-[8px] text-zinc-400">{label}</span>
    </div>
  );
}

function CommItem({ icon, color, label }: { icon: string; color: string; label: string }) {
  return (
    <div className="flex items-center gap-1">
      <span className={`text-[9px] ${color}`}>{icon}</span>
      <span className="text-[8px] text-zinc-400">{label}</span>
    </div>
  );
}
