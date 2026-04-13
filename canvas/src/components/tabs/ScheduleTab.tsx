'use client';

import { useState, useEffect, useCallback } from "react";
import { api } from "@/lib/api";

interface Schedule {
  id: string;
  workspace_id: string;
  name: string;
  cron_expr: string;
  timezone: string;
  prompt: string;
  enabled: boolean;
  last_run_at: string | null;
  next_run_at: string | null;
  run_count: number;
  last_status: string;
  last_error: string;
  created_at: string;
}

interface Props {
  workspaceId: string;
}

function cronToHuman(expr: string): string {
  const parts = expr.trim().split(/\s+/);
  if (parts.length !== 5) return expr;
  const [min, hour, dom, mon, dow] = parts;
  if (min === "*" && hour === "*") return `Every minute`;
  if (min.startsWith("*/")) return `Every ${min.slice(2)} minutes`;
  if (hour.startsWith("*/") && min === "0") return `Every ${hour.slice(2)} hours`;
  if (dom === "*" && mon === "*" && dow === "*" && !hour.startsWith("*/"))
    return `Daily at ${hour.padStart(2, "0")}:${min.padStart(2, "0")} UTC`;
  if (dom === "*" && mon === "*" && dow === "1-5" && !hour.startsWith("*/"))
    return `Weekdays at ${hour.padStart(2, "0")}:${min.padStart(2, "0")} UTC`;
  return expr;
}

function relativeTime(iso: string | null): string {
  if (!iso) return "never";
  const diff = Date.now() - new Date(iso).getTime();
  if (diff < 0) {
    const future = -diff;
    if (future < 60000) return `in ${Math.round(future / 1000)}s`;
    if (future < 3600000) return `in ${Math.round(future / 60000)}m`;
    if (future < 86400000) return `in ${Math.round(future / 3600000)}h`;
    return `in ${Math.round(future / 86400000)}d`;
  }
  if (diff < 60000) return `${Math.round(diff / 1000)}s ago`;
  if (diff < 3600000) return `${Math.round(diff / 60000)}m ago`;
  if (diff < 86400000) return `${Math.round(diff / 3600000)}h ago`;
  return `${Math.round(diff / 86400000)}d ago`;
}

export function ScheduleTab({ workspaceId }: Props) {
  const [schedules, setSchedules] = useState<Schedule[]>([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [editId, setEditId] = useState<string | null>(null);
  const [formName, setFormName] = useState("");
  const [formCron, setFormCron] = useState("0 9 * * *");
  const [formTimezone, setFormTimezone] = useState("UTC");
  const [formPrompt, setFormPrompt] = useState("");
  const [formEnabled, setFormEnabled] = useState(true);
  const [error, setError] = useState("");

  const fetchSchedules = useCallback(async () => {
    try {
      const data = await api.get<Schedule[]>(`/workspaces/${workspaceId}/schedules`);
      setSchedules(data);
    } catch {
      setSchedules([]);
    } finally {
      setLoading(false);
    }
  }, [workspaceId]);

  useEffect(() => {
    fetchSchedules();
    const interval = setInterval(fetchSchedules, 10000);
    return () => clearInterval(interval);
  }, [fetchSchedules]);

  const resetForm = () => {
    setFormName("");
    setFormCron("0 9 * * *");
    setFormTimezone("UTC");
    setFormPrompt("");
    setFormEnabled(true);
    setEditId(null);
    setShowForm(false);
    setError("");
  };

  const handleSubmit = async () => {
    setError("");
    try {
      if (editId) {
        await api.patch(`/workspaces/${workspaceId}/schedules/${editId}`, {
          name: formName,
          cron_expr: formCron,
          timezone: formTimezone,
          prompt: formPrompt,
          enabled: formEnabled,
        });
      } else {
        await api.post(`/workspaces/${workspaceId}/schedules`, {
          name: formName,
          cron_expr: formCron,
          timezone: formTimezone,
          prompt: formPrompt,
          enabled: formEnabled,
        });
      }
      resetForm();
      fetchSchedules();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Failed to save schedule");
    }
  };

  const handleDelete = async (id: string, name: string) => {
    if (!window.confirm(`Delete schedule "${name || "Unnamed"}"? This cannot be undone.`)) return;
    await api.del(`/workspaces/${workspaceId}/schedules/${id}`);
    fetchSchedules();
  };

  const handleToggle = async (sched: Schedule) => {
    await api.patch(`/workspaces/${workspaceId}/schedules/${sched.id}`, {
      enabled: !sched.enabled,
    });
    fetchSchedules();
  };

  const handleEdit = (sched: Schedule) => {
    setFormName(sched.name);
    setFormCron(sched.cron_expr);
    setFormTimezone(sched.timezone);
    setFormPrompt(sched.prompt);
    setFormEnabled(sched.enabled);
    setEditId(sched.id);
    setShowForm(true);
  };

  const handleRunNow = async (sched: Schedule) => {
    try {
      const result = await api.post<{ prompt: string }>(`/workspaces/${workspaceId}/schedules/${sched.id}/run`, {});
      await api.post(`/workspaces/${workspaceId}/a2a`, {
        method: "message/send",
        params: {
          message: {
            role: "user",
            messageId: `manual-cron-${Date.now()}`,
            parts: [{ kind: "text", text: result.prompt }],
          },
        },
      });
      fetchSchedules();
    } catch {
      setError("Failed to run schedule");
    }
  };

  if (loading) {
    return <div className="p-4 text-[10px] text-zinc-500">Loading schedules...</div>;
  }

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="flex items-center justify-between px-3 py-2 border-b border-zinc-800/50">
        <span className="text-[10px] font-semibold text-zinc-400 uppercase tracking-wider">
          Schedules
        </span>
        <button
          onClick={() => { resetForm(); setShowForm(true); }}
          className="text-[9px] px-2 py-0.5 bg-blue-600/20 text-blue-400 rounded hover:bg-blue-600/30 transition-colors"
        >
          + Add Schedule
        </button>
      </div>

      {/* Create/Edit Form */}
      {showForm && (
        <div className="p-3 border-b border-zinc-800/50 bg-zinc-900/50 space-y-2">
          <input
            type="text"
            placeholder="Schedule name (e.g., Daily security scan)"
            value={formName}
            onChange={(e) => setFormName(e.target.value)}
            className="w-full text-[10px] bg-zinc-800 border border-zinc-700 rounded px-2 py-1 text-zinc-200 placeholder:text-zinc-600"
          />
          <div className="flex gap-2">
            <div className="flex-1">
              <label className="text-[8px] text-zinc-500 block mb-0.5">Cron Expression</label>
              <input
                type="text"
                value={formCron}
                onChange={(e) => setFormCron(e.target.value)}
                className="w-full text-[10px] bg-zinc-800 border border-zinc-700 rounded px-2 py-1 text-zinc-200 font-mono"
              />
              <div className="text-[8px] text-zinc-600 mt-0.5">
                {cronToHuman(formCron)}
              </div>
            </div>
            <div className="w-24">
              <label className="text-[8px] text-zinc-500 block mb-0.5">Timezone</label>
              <select
                value={formTimezone}
                onChange={(e) => setFormTimezone(e.target.value)}
                className="w-full text-[9px] bg-zinc-800 border border-zinc-700 rounded px-1 py-1 text-zinc-200"
              >
                <option value="UTC">UTC</option>
                <option value="America/New_York">US Eastern</option>
                <option value="America/Chicago">US Central</option>
                <option value="America/Denver">US Mountain</option>
                <option value="America/Los_Angeles">US Pacific</option>
                <option value="Europe/London">London</option>
                <option value="Europe/Berlin">Berlin</option>
                <option value="Asia/Tokyo">Tokyo</option>
                <option value="Asia/Shanghai">Shanghai</option>
                <option value="Australia/Sydney">Sydney</option>
              </select>
            </div>
          </div>
          <div>
            <label className="text-[8px] text-zinc-500 block mb-0.5">Prompt / Task</label>
            <textarea
              value={formPrompt}
              onChange={(e) => setFormPrompt(e.target.value)}
              placeholder="What should the agent do on this schedule?"
              rows={3}
              className="w-full text-[10px] bg-zinc-800 border border-zinc-700 rounded px-2 py-1 text-zinc-200 placeholder:text-zinc-600 resize-y"
            />
          </div>
          <div className="flex items-center gap-2">
            <label className="flex items-center gap-1.5 text-[9px] text-zinc-400 cursor-pointer">
              <input
                type="checkbox"
                checked={formEnabled}
                onChange={(e) => setFormEnabled(e.target.checked)}
                className="rounded border-zinc-600"
              />
              Enabled
            </label>
          </div>
          {error && <div className="text-[9px] text-red-400">{error}</div>}
          <div className="flex gap-2">
            <button
              onClick={handleSubmit}
              disabled={!formCron || !formPrompt}
              className="text-[9px] px-3 py-1 bg-blue-600 text-white rounded hover:bg-blue-500 disabled:opacity-40 transition-colors"
            >
              {editId ? "Update" : "Create"}
            </button>
            <button
              onClick={resetForm}
              className="text-[9px] px-3 py-1 bg-zinc-800 text-zinc-400 rounded hover:bg-zinc-700 transition-colors"
            >
              Cancel
            </button>
          </div>
          <div className="text-[8px] text-zinc-600 space-y-0.5">
            <div>Common patterns:</div>
            <div className="font-mono">{"0 9 * * *"} — Daily at 9:00 AM</div>
            <div className="font-mono">{"*/30 * * * *"} — Every 30 minutes</div>
            <div className="font-mono">{"0 */4 * * *"} — Every 4 hours</div>
            <div className="font-mono">{"0 9 * * 1-5"} — Weekdays at 9:00 AM</div>
          </div>
        </div>
      )}

      {/* Schedule List */}
      <div className="flex-1 overflow-y-auto">
        {schedules.length === 0 && !showForm ? (
          <div className="p-6 text-center">
            <div className="text-2xl mb-2">⏲</div>
            <div className="text-[10px] text-zinc-400 mb-1">No schedules yet</div>
            <div className="text-[9px] text-zinc-600">
              Add a schedule to run tasks automatically — daily scans, periodic reports, standup reminders.
            </div>
          </div>
        ) : (
          schedules.map((sched) => (
            <div
              key={sched.id}
              className={`px-3 py-2 border-b border-zinc-800/30 ${
                !sched.enabled ? "opacity-50" : ""
              }`}
            >
              <div className="flex items-start justify-between gap-2">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-1.5">
                    <button
                      onClick={() => handleToggle(sched)}
                      className={`w-2 h-2 rounded-full flex-shrink-0 ${
                        sched.last_status === "error"
                          ? "bg-red-400"
                          : sched.last_status === "ok"
                          ? "bg-emerald-400"
                          : "bg-zinc-600"
                      }`}
                      title={sched.enabled ? "Click to disable" : "Click to enable"}
                    />
                    <span className="text-[10px] font-medium text-zinc-200 truncate">
                      {sched.name || "Unnamed schedule"}
                    </span>
                  </div>
                  <div className="text-[9px] text-zinc-500 mt-0.5 font-mono">
                    {cronToHuman(sched.cron_expr)}
                    {sched.timezone !== "UTC" && (
                      <span className="text-zinc-600"> ({sched.timezone})</span>
                    )}
                  </div>
                  <div className="text-[9px] text-zinc-600 mt-0.5 truncate">
                    {sched.prompt.slice(0, 80)}{sched.prompt.length > 80 ? "..." : ""}
                  </div>
                  <div className="flex items-center gap-3 mt-1 text-[8px] text-zinc-600">
                    <span>Last: {relativeTime(sched.last_run_at)}</span>
                    <span>Next: {relativeTime(sched.next_run_at)}</span>
                    <span>Runs: {sched.run_count}</span>
                  </div>
                  {sched.last_error && (
                    <div className="text-[8px] text-red-400/70 mt-0.5 truncate">
                      Error: {sched.last_error}
                    </div>
                  )}
                </div>
                <div className="flex items-center gap-1 flex-shrink-0">
                  <button
                    onClick={() => handleRunNow(sched)}
                    className="text-[8px] px-1.5 py-0.5 text-blue-400 hover:bg-blue-600/20 rounded transition-colors"
                    title="Run now"
                  >
                    ▶
                  </button>
                  <button
                    onClick={() => handleEdit(sched)}
                    className="text-[8px] px-1.5 py-0.5 text-zinc-400 hover:bg-zinc-700 rounded transition-colors"
                    title="Edit"
                  >
                    ✎
                  </button>
                  <button
                    onClick={() => handleDelete(sched.id, sched.name)}
                    className="text-[8px] px-1.5 py-0.5 text-red-400 hover:bg-red-600/20 rounded transition-colors"
                    title="Delete"
                  >
                    ✕
                  </button>
                </div>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
