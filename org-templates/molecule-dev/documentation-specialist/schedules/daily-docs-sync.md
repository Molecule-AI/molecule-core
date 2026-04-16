Daily documentation maintenance. Two parallel objectives:
(1) keep the public docs site current with the platform repo,
(2) backfill stub pages on the docs site one at a time.

SETUP:
  cd /workspace/repo && git pull 2>/dev/null || true
  cd /workspace/docs && git pull 2>/dev/null || true
  cd /workspace/controlplane && git pull 2>/dev/null || true

1a. PAIR RECENT PLATFORM PRS (last 24h):
   cd /workspace/repo
   gh pr list --repo Molecule-AI/molecule-monorepo --state merged \
     --search "merged:>$(date -u -d '24 hours ago' +%Y-%m-%dT%H:%M:%SZ)" \
     --json number,title,files
   For each merged PR that touches a public surface
   (platform/internal/handlers/, plugins/*, org-templates/*,
   docs/architecture.md, README.md, workspace-template/adapters/*):
   - Identify which docs page(s) on the public site cover that surface.
   - If a docs page exists but is stale → update it with examples
     from the PR diff. Open a PR to Molecule-AI/docs with the change.
   - If NO docs page exists for the new surface → propose one
     (add to content/docs/meta.json + new .mdx file). Open a PR.
   - Always close PRs with `Closes platform PR #N` so the link is durable.

1b. PAIR RECENT CONTROLPLANE PRS (last 24h):
   cd /workspace/controlplane
   gh pr list --repo Molecule-AI/molecule-controlplane --state merged \
     --search "merged:>$(date -u -d '24 hours ago' +%Y-%m-%dT%H:%M:%SZ)" \
     --json number,title,files
   ⚠️  PRIVATE REPO. Two cases:
   (i) Internal-only change (handler, schema, infra, fly.toml,
       billing logic): update README.md + PLAN.md + any
       docs/internal/*.md inside molecule-controlplane itself.
       Open the PR against Molecule-AI/molecule-controlplane.
       NEVER mention these changes in /workspace/docs.
   (ii) Customer-facing change (new tier, new region, new SLA,
       pricing change, signup flow change): write a sanitized
       description for the PUBLIC docs site (e.g. "We now offer
       EU-region tenants" — NOT "controlplane reads FLY_REGION
       from env and passes it to provisioner.go:142"). Open a
       PR against Molecule-AI/docs.
   When unsure which category a change falls into: default to
   INTERNAL-only and ask PM for explicit approval before publishing.

2. BACKFILL ONE STUB PAGE:
   cd /workspace/docs
   grep -l "Coming soon" content/docs/*.mdx | head -1
   Pick the highest-priority stub (one of: org-template, plugins,
   channels, schedules, architecture, api-reference, self-hosting,
   observability, troubleshooting). Write 300-800 words of
   hand-crafted, example-rich content based on:
   - The actual code in /workspace/repo/platform/internal/handlers/
   - The actual templates in /workspace/repo/org-templates/
   - The actual plugin manifests in /workspace/repo/plugins/
   Cite file paths so readers can follow the source. Open a PR.

3. LINK + ANCHOR CHECK:
   Use the browser-automation plugin to crawl
   https://doc.moleculesai.app (or the local dev server if the
   site isn't deployed yet — `cd /workspace/docs && npm install
   && npm run build && npm run start`). Report broken links and
   missing anchors back to PM.

4. ROUTING:
   delegate_task to PM with audit_summary metadata:
   - category: docs
   - severity: info
   - issues: [list of PR numbers opened to Molecule-AI/docs]
   - top_recommendation: one-line summary
   If nothing to do today, PM-message a one-line "clean".

5. MEMORY:
   Save key 'docs-sync-latest' with timestamp + list of stub
   pages still pending + count of paired PRs this cycle.
