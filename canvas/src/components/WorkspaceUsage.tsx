'use client';

import { useState, useEffect } from "react";
import { api } from "@/lib/api";

export interface WorkspaceUsageProps {
  workspaceId: string;
}

interface WorkspaceMetrics {
  input_tokens: number;
  output_tokens: number;
  total_calls: number;
  estimated_cost_usd: string;
  period_start: string;
  period_end: string;
}

export function WorkspaceUsage({ workspaceId }: WorkspaceUsageProps) {
  const [metrics, setMetrics] = useState<WorkspaceMetrics | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let ignore = false;
    setLoading(true);
    setError(null);

    api
      .get<WorkspaceMetrics>(`/workspaces/${workspaceId}/metrics`)
      .then((data) => {
        if (!ignore) setMetrics(data);
      })
      .catch((e) => {
        if (!ignore)
          setError(e instanceof Error ? e.message : "Failed to load metrics");
      })
      .finally(() => {
        if (!ignore) setLoading(false);
      });

    return () => {
      ignore = true;
    };
  }, [workspaceId]);

  return (
    <div
      className="rounded-md border border-zinc-700 bg-zinc-900 p-3 space-y-2"
      data-testid="workspace-usage"
    >
      <div className="flex items-center justify-between">
        <h4 className="text-xs font-semibold text-zinc-400 uppercase tracking-wider">
          Usage
        </h4>
        {!loading && metrics && (
          <span
            className="text-[10px] text-zinc-600 font-mono"
            data-testid="usage-period"
          >
            {formatPeriod(metrics.period_start, metrics.period_end)}
          </span>
        )}
      </div>

      <div className="space-y-1.5" data-testid="usage-stats">
        {loading ? (
          <>
            <SkeletonRow />
            <SkeletonRow />
            <SkeletonRow />
          </>
        ) : error ? (
          <p className="text-xs text-red-400" data-testid="usage-error">
            {error}
          </p>
        ) : metrics ? (
          <>
            <StatRow
              label="Input tokens"
              value={`${(metrics.input_tokens ?? 0).toLocaleString()} tokens`}
              testId="usage-input-tokens"
            />
            <StatRow
              label="Output tokens"
              value={`${(metrics.output_tokens ?? 0).toLocaleString()} tokens`}
              testId="usage-output-tokens"
            />
            <StatRow
              label="Estimated cost"
              value={`$${parseFloat(metrics.estimated_cost_usd ?? "0").toFixed(6)}`}
              testId="usage-estimated-cost"
            />
          </>
        ) : null}
      </div>
    </div>
  );
}

function formatPeriod(start: string, end: string): string {
  const fmt = (s: string) =>
    new Date(s).toLocaleDateString(undefined, {
      month: "short",
      day: "numeric",
    });
  return `${fmt(start)} – ${fmt(end)}`;
}

function SkeletonRow() {
  return (
    <div
      className="flex justify-between items-center animate-pulse"
      data-testid="usage-skeleton-row"
    >
      <div className="h-3 w-20 rounded bg-zinc-700" />
      <div className="h-3 w-16 rounded bg-zinc-700" />
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
