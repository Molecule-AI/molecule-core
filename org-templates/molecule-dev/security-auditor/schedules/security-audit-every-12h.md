Recurring security audit. Be thorough and incremental.

1. SETUP:
   cd /workspace/repo && git pull 2>/dev/null || true
   LAST_SHA=$(cat /tmp/last-security-audit-sha 2>/dev/null || git rev-parse HEAD~48 2>/dev/null || echo '')
   CURRENT=$(git rev-parse HEAD)
   CHANGED=$(git diff --name-only $LAST_SHA $CURRENT 2>/dev/null)

2. STATIC ANALYSIS on changed files:
   - Go: gosec -quiet <files>
   - Python: bandit -ll <files>

3. MANUAL REVIEW of every changed file:
   - SQL injection (fmt.Sprintf in DB queries vs $1/$2 params)
   - Path traversal (filepath.Join without validation)
   - Missing auth on new HTTP handlers
   - Secret leakage in logs/errors/responses
   - Command injection (exec.Command with user input)
   - XSS (dangerouslySetInnerHTML, unescaped content in .tsx)
   - #337 class: every secret/token/HMAC comparison MUST use
     `subtle.ConstantTimeCompare` (Go) or `crypto.timingSafeEqual`
     (Node). Flag any `!=` / `==` / `bytes.Equal` against a
     user-supplied value that gates auth or a webhook signature.
   - #319 class: any new channel_config field that holds a
     credential (bot_token, api_key, webhook_secret, oauth_*)
     MUST be added to the `sensitiveFields` slice in
     `platform/internal/channels/secret.go`. Check both
     EncryptSensitiveFields (write path: Create/Update handlers)
     AND DecryptSensitiveFields (read boundary: List, Reload,
     loadChannel, Webhook). Verify the `ec1:` ciphertext prefix
     never leaks into API responses — decryption must happen
     BEFORE masking in list handlers.

4. LIVE API CHECKS against http://host.docker.internal:8080:
   - CanCommunicate bypass: POST /workspaces/<zero-id>/a2a
   - CORS: verify Access-Control-Allow-Origin on a cross-origin request
   - Rate limit headers on /health

4a. DAST TEARDOWN (MANDATORY — prevents test-artifact leak into prod DB):
    Any workspace, secret, or plugin you CREATE during this audit must be
    DELETED before this step exits. Maintain three lists as you go:

      TESTS_WORKSPACES=""   # workspace IDs you POSTed
      TESTS_SECRETS=""      # secret keys you set
      TESTS_PLUGINS=""      # "<ws_id>:<plugin_name>" pairs

    At the end of step 4, iterate each list and DELETE — even if the audit
    aborts, the teardown block must run:

      for ws_id in $TESTS_WORKSPACES; do
        curl -s -X DELETE "http://host.docker.internal:8080/workspaces/$ws_id" \
          -H "Authorization: Bearer $WORKSPACE_AUTH_TOKEN" > /dev/null || true
      done
      for key in $TESTS_SECRETS; do
        curl -s -X DELETE "http://host.docker.internal:8080/admin/secrets/$key" > /dev/null || true
      done
      for pair in $TESTS_PLUGINS; do
        ws="${pair%:*}"; pl="${pair#*:}"
        curl -s -X DELETE "http://host.docker.internal:8080/workspaces/$ws/plugins/$pl" > /dev/null || true
      done

    Prior incident (#17): repeated DAST runs leaked 4 workspaces
    (aaaaaaaa-/bbbbbbbb-/cccccccc-/dddddddd-) into the live DB, each trapped
    in a restart loop on missing config.yaml. This teardown step prevents
    that class of leak regardless of which specific probes you run.

5. SECRETS SCAN: last 20 commits grepped for token patterns
   (sk-ant, sk-or, api_key= etc.) excluding test files.

6. OPEN-PR REVIEW:
   gh pr list --repo Molecule-AI/molecule-monorepo --state open --json number
   For each: gh pr diff | grep '^+' for injection / exec / unsafe patterns.

7. RECORD commit SHA:
   echo $CURRENT > /tmp/last-security-audit-sha

=== FINAL STEP — DELIVERABLE ROUTING (MANDATORY every cycle) ===

a. For each CRITICAL or HIGH finding, FILE A GITHUB ISSUE:
   - Dedupe first: gh issue list --repo Molecule-AI/molecule-monorepo --search "<category>" --state open
   - If not already open: gh issue create --repo Molecule-AI/molecule-monorepo
     --title "security(<category>): <short>"
     --body with severity, file:line, concrete repro (curl or code), proposed fix, related issues
   - Capture the issue number for the PM summary below.

b. delegate_task to PM (workspace id: see `list_peers` for "PM") with a summary:
   - Audit timestamp + SHA range audited
   - Counts by severity (critical / high / medium / low / clean)
   - List of GH issue numbers filed this cycle
   - Top recommendation
   PM decides which dev agent picks up each issue.

c. If NOTHING critical or high this cycle: STILL delegate_task to PM with a
   one-line "clean, audited <SHA_RANGE>, no new findings" so the audit is observable.
   Memory write is a secondary record, not the primary deliverable.

d. Save to memory key 'security-audit-latest' AFTER routing (for cross-session
   recall only — not a substitute for the PM + issue routing above).
