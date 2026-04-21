/**
 * Audit event row — matches the Go auditEventRow struct returned by
 * GET /workspaces/:id/audit.
 */
export interface AuditEntry {
  id: string;
  workspace_id: string;
  timestamp: string;
  agent_id: string;
  session_id: string;
  operation: string;
  input_hash: string | null;
  output_hash: string | null;
  model_used: string | null;
  human_oversight_flag: boolean;
  risk_flag: boolean;
  prev_hmac: string | null;
  hmac: string;
  /** Derived client-side from operation + model_used for display purposes. */
  chain_valid?: boolean;
}

/** Helper: build a human-readable summary from operation + model_used. */
export function auditSummary(entry: AuditEntry): string {
  const base = entry.operation.replace(/_/g, " ");
  if (entry.model_used) return `${base} (${entry.model_used})`;
  return base;
}

/** Paginated response envelope from GET /workspaces/:id/audit */
export interface AuditResponse {
  events: AuditEntry[];
  /** Total matching rows (ignoring limit/offset). */
  total: number;
  /** null when AUDIT_LEDGER_SALT is not configured on the platform side. */
  chain_valid: boolean | null;
}
