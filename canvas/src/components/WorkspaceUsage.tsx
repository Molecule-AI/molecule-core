'use client';

// WorkspaceUsage — Usage panel for a single workspace.
// Currently renders placeholder stat rows.
// TODO: fetch GET /workspaces/:id/metrics when #593 lands and replace
// placeholder values with real token/cost data from the response.

export interface WorkspaceUsageProps {
  workspaceId: string;
}

export function WorkspaceUsage({ workspaceId: _workspaceId }: WorkspaceUsageProps) {
  return (
    <div
      className="rounded-md border border-zinc-700 bg-zinc-900 p-3 space-y-2"
      data-testid="workspace-usage"
    >
      <div className="flex items-center justify-between">
        <h4 className="text-xs font-semibold text-zinc-400 uppercase tracking-wider">
          Usage
        </h4>
        <span
          className="text-[10px] text-zinc-500 bg-zinc-800 border border-zinc-700 rounded px-1.5 py-0.5"
          data-testid="usage-pending-badge"
        >
          pending #593
        </span>
      </div>

      {/* Placeholder stat rows — will be replaced with live data once #593 lands */}
      <div className="space-y-1.5" data-testid="usage-stats">
        <StatRow label="Input tokens" value="—" testId="usage-input-tokens" />
        <StatRow label="Output tokens" value="—" testId="usage-output-tokens" />
        <StatRow label="Estimated cost" value="—" testId="usage-estimated-cost" />
      </div>
    </div>
  );
}

function StatRow({
  label,
  value,
  testId,
}: {
  label: string;
  value: string;
  testId?: string;
}) {
  return (
    <div className="flex justify-between items-center" data-testid={testId}>
      <span className="text-xs text-zinc-500">{label}</span>
      <span className="text-xs text-zinc-400 font-mono">{value}</span>
    </div>
  );
}
