# SAFE-MCP Advisory — 2026-04-17

**Type:** Internal action advisory (distilled from full audit)
**Full audit:** `docs/security/safe-mcp-audit-2026-04-17.md` (SAFE-MCP, 438 lines)
**Audience:** Engineering leads, platform team
**Prepared by:** Documentation Specialist (pairs with PR #808)

---

## TL;DR — What needs fixing and in what order

| # | Finding | Severity | Owner | Status |
|---|---------|----------|-------|--------|
| 1 | NEW-003: Unpinned npm MCP packages in `.mcp.json` | **HIGH** | Platform | Open — fix in next deploy |
| 2 | VULN-003: No manifest signing on GitHub plugin install | **HIGH** | Platform | Open — Phase 35 |
| 3 | VULN-004: Floating plugin refs (no pinned SHA) | **HIGH** | Platform | Open — Phase 35 |
| 4 | VULN-002: GLOBAL memory prompt injection (partial) | **HIGH** | Platform | Partially mitigated (#767) |
| 5 | VULN-006: No tool output sanitization in MCP server | MEDIUM | DevRel/SDK | Open |
| 6 | NEW-002: subprocess sandbox allows `language=shell` | MEDIUM | Platform | By-design; needs scope review |
| 7 | NEW-001: LangGraph A2A calls missing auth headers | MEDIUM | LangGraph template | Open |
| 8 | VULN-005: GLOBAL memories visible to all workspaces | MEDIUM | Platform | Partially mitigated (#767) |
| 9 | NEW-004: `_maybe_log_skill_promotion` unauthenticated heartbeat | LOW | Platform | Open |

**Already fixed:** VULN-001 (`X-Workspace-ID` system-caller header forge) — confirmed resolved in PR #766.

---

## Immediate action: NEW-003 (HIGH) — Pin npm MCP packages

**File:** `.mcp.json` — change both entries before next developer onboarding or CI run.

Current (unsafe):
```json
"args": ["-y", "@molecule-ai/mcp-server"]
```

Fixed:
```json
"args": ["@molecule-ai/mcp-server@<current-version>"]
```

Steps:
1. Run `npm show @molecule-ai/mcp-server version` and `npm show @awareness-sdk/local version` to get the latest pinnable version.
2. Update `.mcp.json` — remove `-y` flag, add `@<exact-version>` to each package name.
3. Add a `package.json` + `package-lock.json` alongside `.mcp.json` to lock the full dependency tree.
4. Wire `npm audit signatures` into CI (`molecule-ci` pipeline).

**Why this is urgent:** `npx -y` fetches and executes the latest published npm package on every invocation with no integrity check. A compromised `@molecule-ai` npm account or a dependency confusion attack causes arbitrary code execution in the Claude Code developer environment.

---

## Short-term (Phase 35): Plugin supply-chain hardening

VULN-003 and VULN-004 require a Phase 35 track. Recommended scope:

1. **Require pinned refs** — reject `github://org/repo` without `#<40-char-sha>`. Already gated by `PLUGIN_ALLOW_UNPINNED` (PR #775); make `false` the hard default in production.
2. **Add manifest content hash** — add a `sha256:` field to `plugin.yaml` covering the cloned content tree. Verify post-clone before staging.
3. **Consider sigstore/GPG release signing** for first-party plugins (`molecule-ai-plugin-*`).

---

## Medium-term: GLOBAL memory scope hardening

VULN-002 / VULN-005 — delimiter wrapping (PR #767) reduces injection risk but does not prevent a malicious workspace from writing to GLOBAL scope and having the injected prompt read by a different workspace. Proposed additional controls:

- Rate-limit GLOBAL `commit_memory` writes per workspace per hour.
- Add a supervisor/approval flow for GLOBAL writes from untrusted workspaces.
- Consider making GLOBAL scope read-only except for privileged system roles.

---

## References

- Full audit: `docs/security/safe-mcp-audit-2026-04-17.md`
- SAFE-MCP framework: `docs/security/safe-mcp-audit.md`
- Issue tracker: #747 (parent), see follow-on issues linked from PR #808
- Public docs: PR #18 on `Molecule-AI/docs` (covers only customer-visible security notes)
