"use client";

import { useState } from "react";
import { api } from "@/lib/api";

export function CreateWorkspaceButton() {
  const [open, setOpen] = useState(false);

  return (
    <>
      <button
        onClick={() => setOpen(true)}
        className="fixed bottom-6 right-6 z-40 px-5 py-2.5 bg-blue-600 hover:bg-blue-500 active:bg-blue-700 text-sm font-medium rounded-xl text-white shadow-lg shadow-blue-600/20 hover:shadow-xl hover:shadow-blue-500/30 transition-all duration-200 flex items-center gap-2"
      >
        <svg width="14" height="14" viewBox="0 0 14 14" fill="none" className="shrink-0">
          <path d="M7 1v12M1 7h12" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
        </svg>
        New Workspace
      </button>

      {open && <CreateDialog onClose={() => setOpen(false)} />}
    </>
  );
}

function CreateDialog({ onClose }: { onClose: () => void }) {
  const [name, setName] = useState("");
  const [role, setRole] = useState("");
  const [tier, setTier] = useState(1);
  const [template, setTemplate] = useState("");
  const [parentId, setParentId] = useState("");
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleCreate = async () => {
    if (!name.trim()) {
      setError("Name is required");
      return;
    }

    setCreating(true);
    setError(null);

    try {
      await api.post("/workspaces", {
        name: name.trim(),
        role: role.trim() || undefined,
        template: template.trim() || undefined,
        tier,
        parent_id: parentId.trim() || undefined,
        canvas: { x: Math.random() * 400 + 100, y: Math.random() * 300 + 100 },
      });
      onClose();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to create workspace");
    } finally {
      setCreating(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm">
      <div className="bg-zinc-900 border border-zinc-700/60 rounded-2xl shadow-2xl shadow-black/40 w-[400px] p-6">
        <h2 className="text-base font-semibold text-zinc-100 mb-1">Create Workspace</h2>
        <p className="text-xs text-zinc-500 mb-5">Add a new workspace node to the canvas</p>

        <div className="space-y-3.5">
          <InputField label="Name" required value={name} onChange={setName} placeholder="e.g. SEO Agent" autoFocus />
          <InputField label="Role" value={role} onChange={setRole} placeholder="e.g. SEO Specialist" />
          <InputField label="Template" value={template} onChange={setTemplate} placeholder="e.g. seo-agent (from workspace-configs-templates/)" mono />

          <div>
            <label className="text-[11px] text-zinc-400 block mb-1">Tier</label>
            <div className="grid grid-cols-3 gap-1.5">
              {[
                { value: 1, label: "T1", desc: "Sandboxed" },
                { value: 2, label: "T2", desc: "Standard" },
                { value: 3, label: "T3", desc: "Full Access" },
              ].map((t) => (
                <button
                  key={t.value}
                  onClick={() => setTier(t.value)}
                  className={`py-2 rounded-lg text-center transition-colors ${
                    tier === t.value
                      ? "bg-blue-600/20 border border-blue-500/50 text-blue-300"
                      : "bg-zinc-800/60 border border-zinc-700/40 text-zinc-400 hover:text-zinc-300 hover:border-zinc-600"
                  }`}
                >
                  <div className="text-xs font-mono font-semibold">{t.label}</div>
                  <div className="text-[9px] mt-0.5 opacity-70">{t.desc}</div>
                </button>
              ))}
            </div>
          </div>

          <InputField label="Parent Workspace ID" value={parentId} onChange={setParentId} placeholder="Leave empty for root-level" mono />
        </div>

        {error && (
          <div className="mt-4 px-3 py-2 bg-red-950/40 border border-red-800/50 rounded-lg text-xs text-red-400">
            {error}
          </div>
        )}

        <div className="flex justify-end gap-2.5 mt-6">
          <button
            onClick={onClose}
            className="px-4 py-2 bg-zinc-800 hover:bg-zinc-700 text-sm rounded-lg text-zinc-300 transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={handleCreate}
            disabled={creating}
            className="px-5 py-2 bg-blue-600 hover:bg-blue-500 active:bg-blue-700 text-sm rounded-lg text-white disabled:opacity-50 transition-colors"
          >
            {creating ? "Creating..." : "Create"}
          </button>
        </div>
      </div>
    </div>
  );
}

function InputField({
  label,
  value,
  onChange,
  placeholder,
  required,
  autoFocus,
  mono,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
  required?: boolean;
  autoFocus?: boolean;
  mono?: boolean;
}) {
  return (
    <div>
      <label className="text-[11px] text-zinc-400 block mb-1">
        {label} {required && <span className="text-red-400">*</span>}
      </label>
      <input
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        autoFocus={autoFocus}
        className={`w-full bg-zinc-800/60 border border-zinc-700/50 rounded-lg px-3 py-2 text-sm text-zinc-100 placeholder-zinc-600 focus:outline-none focus:border-blue-500/60 focus:ring-1 focus:ring-blue-500/20 transition-colors ${mono ? "font-mono text-xs" : ""}`}
      />
    </div>
  );
}
