"use client";

import { useCallback, useState, type ReactNode } from "react";
import { api } from "@/lib/api";
import {
  checkDeploySecrets,
  resolveRuntime,
  type PreflightResult,
  type Template,
} from "@/lib/deploy-preflight";
import { MissingKeysModal } from "@/components/MissingKeysModal";

/**
 * useTemplateDeploy — shared preflight + POST + modal wiring for
 * every surface that deploys a workspace from a template.
 *
 * Owns: `checkDeploySecrets` call, `MissingKeysModal` render, the
 * `POST /workspaces` that follows, and per-template `deploying`
 * state. Returns `modal` as a `ReactNode` ready to place inline.
 *
 * Why a hook rather than two copies: the runtime-fallback table
 * (`resolveRuntime`) and the preflight wiring were previously
 * copy-pasted between TemplatePalette and EmptyState. When the
 * copies drifted (palette had the full id-to-runtime map,
 * empty-state had only the `-default` strip), the two surfaces
 * could silently disagree on future templates that need a
 * non-identity mapping. Single owner closes the drift surface.
 */
export interface UseTemplateDeployOptions {
  /** Compute canvas coords for the new workspace. Called once per
   *  successful deploy. Defaults to random coords in the [100, 500] ×
   *  [100, 400] band, matching the sidebar palette's historical
   *  placement. Override for surfaces that want deterministic
   *  placement (e.g. EmptyState's first-deploy "center-ish" target). */
  canvasCoords?: () => { x: number; y: number };

  /** Optional post-deploy side effect — passed the id of the new
   *  workspace. EmptyState uses this to auto-select the node and
   *  flip the side panel to Chat so a fresh tenant sees something
   *  useful. */
  onDeployed?: (workspaceId: string) => void;
}

/** Paired template + preflight result carried through the "user
 *  clicked deploy → modal opens → keys saved → retry" loop. Named
 *  so the `useState` generic and any future signature change have
 *  a single place to track. */
interface MissingKeysInfo {
  template: Template;
  preflight: PreflightResult;
}

export interface UseTemplateDeployResult {
  /** Template id currently being deployed (incl. the preflight
   *  network call), or null when idle. Callers pass this to disable
   *  the relevant button and show a spinner. */
  deploying: string | null;

  /** Last deploy error message, or null. Cleared on next `deploy`
   *  call. */
  error: string | null;

  /** Kick off a deploy. Opens the missing-keys modal if preflight
   *  returns not-ok; otherwise fires POST /workspaces directly. */
  deploy: (template: Template) => Promise<void>;

  /** The missing-keys modal, ready to place inline. Always non-null
   *  (the underlying component self-gates on `open`), so the caller
   *  can drop `{modal}` anywhere without conditionals. */
  modal: ReactNode;
}

export function useTemplateDeploy(
  options: UseTemplateDeployOptions = {},
): UseTemplateDeployResult {
  const [deploying, setDeploying] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [missingKeysInfo, setMissingKeysInfo] = useState<MissingKeysInfo | null>(null);

  const { canvasCoords, onDeployed } = options;

  /** Actually execute the POST /workspaces call. Split from `deploy`
   *  so the "modal → keys added → retry" path can reuse it without
   *  re-running preflight (the user just proved the keys are now set). */
  const executeDeploy = useCallback(
    async (template: Template) => {
      setDeploying(template.id);
      setError(null);
      try {
        const coords = canvasCoords
          ? canvasCoords()
          : {
              x: Math.random() * 400 + 100,
              y: Math.random() * 300 + 100,
            };
        const ws = await api.post<{ id: string }>("/workspaces", {
          name: template.name,
          template: template.id,
          tier: template.tier,
          canvas: coords,
        });
        onDeployed?.(ws.id);
      } catch (e) {
        setError(e instanceof Error ? e.message : "Deploy failed");
      } finally {
        setDeploying(null);
      }
    },
    [canvasCoords, onDeployed],
  );

  const deploy = useCallback(
    async (template: Template) => {
      setDeploying(template.id);
      setError(null);
      let preflight: PreflightResult;
      try {
        const runtime = template.runtime ?? resolveRuntime(template.id);
        preflight = await checkDeploySecrets({
          runtime,
          models: template.models,
          required_env: template.required_env,
        });
      } catch (e) {
        // Preflight network failure used to strand `deploying` — the
        // button stayed disabled forever because the throw bypassed
        // the setDeploying(null) in the non-ok branch below. Any
        // future refactor that drops this try block will regress the
        // same way; keep it narrow around just the preflight call
        // so a successful preflight still lets executeDeploy own
        // its own error path.
        setError(e instanceof Error ? e.message : "Preflight check failed");
        setDeploying(null);
        return;
      }
      if (!preflight.ok) {
        setMissingKeysInfo({ template, preflight });
        setDeploying(null);
        return;
      }
      await executeDeploy(template);
    },
    [executeDeploy],
  );

  // No useCallback here — consumers call this on every render anyway
  // (it's placed inline in JSX), and useCallback's deps would
  // invalidate on every state change, making the memoisation a wash.
  // Plain ReactNode is simpler and equally performant.
  const modal: ReactNode = (
    <MissingKeysModal
      open={!!missingKeysInfo}
      missingKeys={missingKeysInfo?.preflight.missingKeys ?? []}
      providers={missingKeysInfo?.preflight.providers ?? []}
      runtime={missingKeysInfo?.preflight.runtime ?? ""}
      onKeysAdded={() => {
        if (missingKeysInfo) {
          const template = missingKeysInfo.template;
          setMissingKeysInfo(null);
          // Intentional fire-and-forget — executeDeploy manages
          // its own error state via setError.
          void executeDeploy(template);
        }
      }}
      onCancel={() => setMissingKeysInfo(null)}
    />
  );

  return { deploying, error, deploy, modal };
}
