"use client";

import { useState, useEffect, useCallback } from "react";
import { useCanvasStore } from "@/store/canvas";

const STORAGE_KEY = "molecule-onboarding-complete";

type Step = "welcome" | "api-key" | "send-message" | "done";

const STEPS: { id: Step; title: string; description: string }[] = [
  {
    id: "welcome",
    title: "Welcome to Molecule AI",
    description:
      "Create your first workspace to deploy an agent. Pick a template from the center panel or create a blank workspace.",
  },
  {
    id: "api-key",
    title: "Set your API key",
    description:
      "Your agent needs an API key to respond. Open the Config tab and add your Anthropic API key under Secrets.",
  },
  {
    id: "send-message",
    title: "Send your first message",
    description:
      'Switch to the Chat tab and say hello! Try: "What can you help me with?"',
  },
  {
    id: "done",
    title: "You're all set!",
    description:
      "Your agent is ready. Explore skills, nest workspaces into teams, or deploy more agents from templates.",
  },
];

/**
 * OnboardingWizard — guides first-time users through setup.
 * Step 1: Welcome + create workspace (shown when canvas is empty)
 * Step 2: API key setup (shown after first workspace created)
 * Step 3: First message
 * Step 4: Done
 *
 * Renders as a floating card in the bottom-left corner.
 * Dismissible at any time. Progress tracked via localStorage.
 */
export function OnboardingWizard() {
  const nodes = useCanvasStore((s) => s.nodes);
  const selectedNodeId = useCanvasStore((s) => s.selectedNodeId);
  const panelTab = useCanvasStore((s) => s.panelTab);

  const [dismissed, setDismissed] = useState(true); // default hidden until we check
  const [step, setStep] = useState<Step>("welcome");

  // Check localStorage on mount
  useEffect(() => {
    const done = localStorage.getItem(STORAGE_KEY);
    if (done) {
      setDismissed(true);
      return;
    }
    // First-time user — show wizard
    const currentNodes = useCanvasStore.getState().nodes;
    setDismissed(false);
    // Start at welcome if no workspaces, otherwise at api-key
    setStep(currentNodes.length === 0 ? "welcome" : "api-key");
  }, []);

  // Auto-advance from "welcome" to "api-key" when first workspace appears
  useEffect(() => {
    if (step === "welcome" && nodes.length > 0) {
      setStep("api-key");
    }
  }, [step, nodes.length]);

  // Auto-advance steps based on user actions
  useEffect(() => {
    if (dismissed) return;

    if (step === "api-key" && panelTab === "config" && selectedNodeId) {
      // User navigated to config — they'll set the key. Advance after a moment.
      const timer = setTimeout(() => setStep("send-message"), 3000);
      return () => clearTimeout(timer);
    }
  }, [step, panelTab, selectedNodeId, dismissed]);

  // Listen for agent messages to auto-advance to "done"
  const agentMessages = useCanvasStore((s) =>
    selectedNodeId ? s.agentMessages[selectedNodeId] : undefined
  );
  useEffect(() => {
    if (step === "send-message" && agentMessages && agentMessages.length > 0) {
      setStep("done");
    }
  }, [step, agentMessages]);

  const dismiss = useCallback(() => {
    setDismissed(true);
    localStorage.setItem(STORAGE_KEY, "true");
  }, []);

  const handleAction = useCallback(() => {
    if (step === "welcome") {
      // No action needed — EmptyState handles workspace creation.
      // If there are already nodes somehow, advance.
      if (useCanvasStore.getState().nodes.length > 0) {
        setStep("api-key");
      }
    } else if (step === "api-key" && selectedNodeId) {
      useCanvasStore.getState().setPanelTab("config");
    } else if (step === "send-message" && selectedNodeId) {
      useCanvasStore.getState().setPanelTab("chat");
    } else if (step === "done") {
      dismiss();
    }
  }, [step, selectedNodeId, dismiss]);

  if (dismissed) return null;

  const currentStepIdx = STEPS.findIndex((s) => s.id === step);
  const currentStep = STEPS[currentStepIdx];

  // Screen-reader labels for each step (announced on step transitions)
  const stepLabels: Record<string, string> = {
    welcome: "Onboarding step 1 of 4: Welcome",
    "api-key": "Onboarding step 2 of 4: Configure your workspace",
    "send-message": "Onboarding step 3 of 4: Send your first message",
    done: "Onboarding complete",
  };

  return (
    <div
      role="complementary"
      aria-label="Onboarding guide"
      className="fixed bottom-20 left-4 z-50 w-80 rounded-2xl border border-zinc-700/60 bg-zinc-900/95 backdrop-blur-xl shadow-2xl shadow-black/40 overflow-hidden"
    >
      {/* Progress bar */}
      <div className="h-1 bg-zinc-800">
        <div
          className="h-full bg-gradient-to-r from-blue-500 to-sky-400 transition-all duration-500"
          style={{ width: `${((currentStepIdx + 1) / STEPS.length) * 100}%` }}
        />
      </div>

      {/* Polite live region — announces step transitions to screen readers */}
      <div
        role="status"
        aria-live="polite"
        aria-atomic="true"
        className="sr-only"
      >
        {stepLabels[step] ?? currentStep.title}
      </div>

      <div className="p-4">
        {/* Step indicator */}
        <div className="flex items-center justify-between mb-2">
          <span className="text-[9px] font-semibold uppercase tracking-widest text-sky-400/80">
            Step {currentStepIdx + 1} of {STEPS.length}
          </span>
          <button
            type="button"
            onClick={dismiss}
            aria-label="Skip onboarding guide"
            className="text-[10px] text-zinc-400 hover:text-zinc-200 transition-colors"
          >
            Skip guide
          </button>
        </div>

        {/* Content */}
        <h3 className="text-sm font-medium text-zinc-100 mb-1">
          {currentStep.title}
        </h3>
        <p className="text-[11px] text-zinc-400 leading-relaxed mb-3">
          {currentStep.description}
        </p>

        {/* Action button */}
        <div className="flex gap-2">
          <button
            type="button"
            onClick={handleAction}
            className="flex-1 px-3 py-1.5 bg-blue-600/90 hover:bg-blue-500 rounded-lg text-[11px] font-medium text-white transition-colors"
          >
            {step === "welcome"
              ? "Create Workspace"
              : step === "api-key"
              ? "Open Config"
              : step === "send-message"
              ? "Open Chat"
              : "Get Started"}
          </button>
          {step !== "done" && (
            <button
              type="button"
              onClick={() => {
                const next = STEPS[currentStepIdx + 1];
                if (next) setStep(next.id);
                else dismiss();
              }}
              className="px-3 py-1.5 bg-zinc-800 hover:bg-zinc-700 rounded-lg text-[11px] text-zinc-400 transition-colors"
            >
              Next
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
