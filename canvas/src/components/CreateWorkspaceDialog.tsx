"use client";

import { useState, useEffect, useRef, useCallback, useId } from "react";
import * as Dialog from "@radix-ui/react-dialog";
import { api } from "@/lib/api";

interface WorkspaceOption {
  id: string;
  name: string;
  tier: number;
}

interface HermesProvider {
  id: string;
  label: string;
  envVar: string;
}

// All providers supported by Hermes runtime via providers.resolve_provider()
export const HERMES_PROVIDERS: HermesProvider[] = [
  { id: "anthropic", label: "Anthropic (Claude)", envVar: "ANTHROPIC_API_KEY" },
  { id: "openai", label: "OpenAI", envVar: "OPENAI_API_KEY" },
  { id: "openrouter", label: "OpenRouter", envVar: "OPENROUTER_API_KEY" },
  { id: "xai", label: "xAI (Grok)", envVar: "XAI_API_KEY" },
  { id: "gemini", label: "Google Gemini", envVar: "GEMINI_API_KEY" },
  { id: "qwen", label: "Qwen (Alibaba)", envVar: "QWEN_API_KEY" },
  { id: "glm", label: "GLM (Zhipu AI)", envVar: "GLM_API_KEY" },
  { id: "kimi", label: "Kimi (Moonshot)", envVar: "KIMI_API_KEY" },
  { id: "minimax", label: "MiniMax", envVar: "MINIMAX_API_KEY" },
  { id: "deepseek", label: "DeepSeek", envVar: "DEEPSEEK_API_KEY" },
  { id: "groq", label: "Groq", envVar: "GROQ_API_KEY" },
  { id: "mistral", label: "Mistral", envVar: "MISTRAL_API_KEY" },
  { id: "together", label: "Together AI", envVar: "TOGETHER_API_KEY" },
  { id: "fireworks", label: "Fireworks AI", envVar: "FIREWORKS_API_KEY" },
  { id: "hermes", label: "Hermes / Nous (legacy)", envVar: "HERMES_API_KEY" },
];

export function CreateWorkspaceButton() {
  const [open, setOpen] = useState(false);
  const [name, setName] = useState("");
  const [role, setRole] = useState("");
  const [tier, setTier] = useState(1);
  const [template, setTemplate] = useState("");
  const [parentId, setParentId] = useState("");
  const [budgetLimit, setBudgetLimit] = useState("");
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [workspaces, setWorkspaces] = useState<WorkspaceOption[]>([]);

  // Hermes-specific state
  const [hermesProvider, setHermesProvider] = useState("anthropic");
  const [hermesApiKey, setHermesApiKey] = useState("");

  // Refs for roving tabIndex on the tier radio group (WCAG 2.1 arrow-key nav)
  const radioRefs = useRef<Array<HTMLButtonElement | null>>([]);
  const TIERS = [
    { value: 1, label: "T1", desc: "Sandboxed" },
    { value: 2, label: "T2", desc: "Standard" },
    { value: 3, label: "T3", desc: "Full Access" },
  ];

  const handleRadioKeyDown = useCallback(
    (e: React.KeyboardEvent, currentIndex: number) => {
      if (e.key === "ArrowDown" || e.key === "ArrowRight") {
        e.preventDefault();
        const next = (currentIndex + 1) % TIERS.length;
        setTier(TIERS[next].value);
        radioRefs.current[next]?.focus();
      } else if (e.key === "ArrowUp" || e.key === "ArrowLeft") {
        e.preventDefault();
        const prev = (currentIndex - 1 + TIERS.length) % TIERS.length;
        setTier(TIERS[prev].value);
        radioRefs.current[prev]?.focus();
      }
    },
    // TIERS is stable (module-level constant pattern), setTier is stable from useState
    // eslint-disable-next-line react-hooks/exhaustive-deps
    []
  );

  const isHermes = template.trim().toLowerCase() === "hermes";

  // Reset form and load workspaces whenever dialog opens
  useEffect(() => {
    if (!open) return;
    setName("");
    setRole("");
    setTier(1);
    setTemplate("");
    setParentId("");
    setBudgetLimit("");
    setError(null);
    setHermesProvider("anthropic");
    setHermesApiKey("");
    api
      .get<WorkspaceOption[]>("/workspaces")
      .then((ws) => setWorkspaces(ws))
      .catch(() => {});
  }, [open]);

  const handleCreate = async () => {
    if (!name.trim()) {
      setError("Name is required");
      return;
    }
    if (isHermes && !hermesApiKey.trim()) {
      setError("API key is required for Hermes workspaces");
      return;
    }
    setCreating(true);
    setError(null);

    const provider = isHermes
      ? HERMES_PROVIDERS.find((p) => p.id === hermesProvider)
      : undefined;

    try {
      const parsedBudget = budgetLimit.trim()
        ? parseFloat(budgetLimit)
        : null;

      await api.post("/workspaces", {
        name: name.trim(),
        role: role.trim() || undefined,
        template: template.trim() || undefined,
        tier,
        parent_id: parentId || undefined,
        budget_limit: parsedBudget,
        canvas: { x: Math.random() * 400 + 100, y: Math.random() * 300 + 100 },
        ...(isHermes && provider
          ? { secrets: { [provider.envVar]: hermesApiKey.trim() } }
          : {}),
      });
      setOpen(false);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to create workspace");
    } finally {
      setCreating(false);
    }
  };

  return (
    <Dialog.Root open={open} onOpenChange={setOpen}>
      <Dialog.Trigger asChild>
        <button className="fixed bottom-6 right-4 sm:right-6 z-40 px-4 sm:px-5 py-2.5 bg-blue-600 hover:bg-blue-500 active:bg-blue-700 text-sm font-medium rounded-xl text-white shadow-lg shadow-blue-600/20 hover:shadow-xl hover:shadow-blue-500/30 transition-all duration-200 flex items-center gap-2">
          <svg
            width="14"
            height="14"
            viewBox="0 0 14 14"
            fill="none"
            className="shrink-0"
            aria-hidden="true"
          >
            <path
              d="M7 1v12M1 7h12"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
            />
          </svg>
          New Workspace
        </button>
      </Dialog.Trigger>

      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm" />
        <Dialog.Content
          className="fixed z-50 left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 bg-zinc-900 border border-zinc-700/60 rounded-2xl shadow-2xl shadow-black/40 w-[calc(100vw-2rem)] sm:w-[400px] max-h-[90vh] overflow-y-auto p-4 sm:p-6"
          aria-describedby={undefined}
        >
          <Dialog.Title className="text-base font-semibold text-zinc-100 mb-1">
            Create Workspace
          </Dialog.Title>
          <p className="text-xs text-zinc-500 mb-5">
            Add a new workspace node to the canvas
          </p>

          <div className="space-y-3.5">
            <InputField
              label="Name"
              required
              value={name}
              onChange={setName}
              placeholder="e.g. SEO Agent"
            />
            <InputField
              label="Role"
              value={role}
              onChange={setRole}
              placeholder="e.g. SEO Specialist"
            />
            <InputField
              label="Budget limit (USD)"
              value={budgetLimit}
              onChange={setBudgetLimit}
              placeholder="e.g. 100"
              type="number"
              helper="Leave blank for unlimited"
            />
            <InputField
              label="Template"
              value={template}
              onChange={setTemplate}
              placeholder="e.g. seo-agent (from workspace-configs-templates/)"
              mono
            />

            <div>
              <div
                role="radiogroup"
                aria-label="Workspace tier"
                className="grid grid-cols-3 gap-1.5"
              >
                <div className="col-span-3 text-[11px] text-zinc-400 mb-1">
                  Tier
                </div>
                {TIERS.map((t, idx) => (
                  <button
                    key={t.value}
                    ref={(el) => { radioRefs.current[idx] = el; }}
                    role="radio"
                    aria-checked={tier === t.value}
                    tabIndex={tier === t.value ? 0 : -1}
                    onClick={() => setTier(t.value)}
                    onKeyDown={(e) => handleRadioKeyDown(e, idx)}
                    className={`py-2 rounded-lg text-center transition-colors ${
                      tier === t.value
                        ? "bg-blue-600/20 border border-blue-500/50 text-blue-300"
                        : "bg-zinc-800/60 border border-zinc-700/40 text-zinc-400 hover:text-zinc-300 hover:border-zinc-600"
                    }`}
                  >
                    <div className="text-xs font-mono font-semibold">
                      {t.label}
                    </div>
                    <div className="text-[10px] mt-0.5 opacity-70">
                      {t.desc}
                    </div>
                  </button>
                ))}
              </div>
            </div>

            <div>
              <label className="text-[11px] text-zinc-400 block mb-1">
                Parent Workspace
              </label>
              <select
                value={parentId}
                onChange={(e) => setParentId(e.target.value)}
                className="w-full bg-zinc-800/60 border border-zinc-700/50 rounded-lg px-3 py-2 text-sm text-zinc-100 focus:outline-none focus:border-blue-500/60 focus:ring-1 focus:ring-blue-500/20 transition-colors"
              >
                <option value="">None (root level)</option>
                {workspaces.map((ws) => (
                  <option key={ws.id} value={ws.id}>
                    T{ws.tier} · {ws.name}
                  </option>
                ))}
              </select>
            </div>
          </div>

          {/* Hermes provider configuration — shown only when template === "hermes" */}
          {isHermes && (
            <div
              className="mt-4 rounded-xl border border-violet-700/40 bg-violet-950/20 p-4 space-y-3"
              data-testid="hermes-provider-section"
            >
              <p className="text-[11px] font-semibold text-violet-400 uppercase tracking-wide">
                Hermes Provider
              </p>
              <p className="text-[11px] text-zinc-500 -mt-1">
                Choose the AI provider and paste your API key. The key is
                stored as an encrypted workspace secret.
              </p>

              <div>
                <label
                  htmlFor="hermes-provider-select"
                  className="text-[11px] text-zinc-400 block mb-1"
                >
                  Provider
                </label>
                <select
                  id="hermes-provider-select"
                  value={hermesProvider}
                  onChange={(e) => setHermesProvider(e.target.value)}
                  aria-label="Hermes provider"
                  className="w-full bg-zinc-800/60 border border-zinc-700/50 rounded-lg px-3 py-2 text-sm text-zinc-100 focus:outline-none focus:border-violet-500/60 focus:ring-1 focus:ring-violet-500/20 transition-colors"
                >
                  {HERMES_PROVIDERS.map((p) => (
                    <option key={p.id} value={p.id}>
                      {p.label}
                    </option>
                  ))}
                </select>
              </div>

              <div>
                <label
                  htmlFor="hermes-api-key-input"
                  className="text-[11px] text-zinc-400 block mb-1"
                >
                  API Key{" "}
                  <span aria-hidden="true" className="text-red-400">
                    *
                  </span>
                  <span className="sr-only"> (required)</span>
                </label>
                <input
                  id="hermes-api-key-input"
                  type="password"
                  value={hermesApiKey}
                  onChange={(e) => setHermesApiKey(e.target.value)}
                  placeholder="sk-…"
                  aria-label="Hermes API key"
                  autoComplete="off"
                  className="w-full bg-zinc-800/60 border border-zinc-700/50 rounded-lg px-3 py-2 text-sm text-zinc-100 placeholder-zinc-600 focus:outline-none focus:border-violet-500/60 focus:ring-1 focus:ring-violet-500/20 transition-colors font-mono"
                />
              </div>
            </div>
          )}

          {error && (
            <div
              role="alert"
              className="mt-4 px-3 py-2 bg-red-950/40 border border-red-800/50 rounded-lg text-xs text-red-400"
            >
              {error}
            </div>
          )}

          <div className="flex justify-end gap-2.5 mt-6">
            <Dialog.Close asChild>
              <button className="px-4 py-2 bg-zinc-800 hover:bg-zinc-700 text-sm rounded-lg text-zinc-300 transition-colors">
                Cancel
              </button>
            </Dialog.Close>
            <button
              onClick={handleCreate}
              disabled={creating}
              className="px-5 py-2 bg-blue-600 hover:bg-blue-500 active:bg-blue-700 text-sm rounded-lg text-white disabled:opacity-50 transition-colors"
            >
              {creating ? "Creating..." : "Create"}
            </button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

function InputField({
  label,
  value,
  onChange,
  placeholder,
  required,
  mono,
  type = "text",
  helper,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
  required?: boolean;
  mono?: boolean;
  type?: string;
  helper?: string;
}) {
  // useId() generates a stable, unique ID for the label↔input association,
  // satisfying WCAG 2.1 SC 1.3.1 (Info and Relationships, Level A).
  const inputId = useId();

  return (
    <div>
      <label htmlFor={inputId} className="text-[11px] text-zinc-400 block mb-1">
        {label}{" "}
        {required && (
          <>
            <span aria-hidden="true" className="text-red-400">
              *
            </span>
            <span className="sr-only"> (required)</span>
          </>
        )}
      </label>
      <input
        id={inputId}
        type={type}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        min={type === "number" ? "0" : undefined}
        step={type === "number" ? "0.01" : undefined}
        className={`w-full bg-zinc-800/60 border border-zinc-700/50 rounded-lg px-3 py-2 text-sm text-zinc-100 placeholder-zinc-500 focus:outline-none focus:border-blue-500/60 focus:ring-1 focus:ring-blue-500/20 transition-colors ${mono ? "font-mono text-xs" : ""}`}
      />
      {helper && (
        <p className="mt-1 text-xs text-zinc-500">{helper}</p>
      )}
    </div>
  );
}
