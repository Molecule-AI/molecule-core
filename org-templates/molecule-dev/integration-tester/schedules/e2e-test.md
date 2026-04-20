IMPORTANT: Check Molecule-AI/internal repo for roadmap (PLAN.md), known issues (known-issues.md), runbooks before starting work.

Cross-repo E2E test cycle. Run every 30 minutes.

1. SETUP: Pull latest from molecule-core, molecule-controlplane, molecule-tenant-proxy, molecule-app, molecule-ai-workspace-runtime.

2. SMOKE TESTS: Health check all service endpoints, API connectivity, WebSocket upgrade.

3. E2E FLOW TESTS: User signup -> org -> workspace provision -> task execution. Billing flow. Admin console.

4. CONTRACT TESTS: API schema compatibility, WebSocket protocol, A2A message format.

5. REPORT: File issues for failures. delegate_task to Dev Lead with summary.
