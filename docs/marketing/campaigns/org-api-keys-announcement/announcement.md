# Org-Scoped API Keys — Community Announcement Copy

**Canonical hashtag:** #OrgAPIKeys
**Status:** Ready to post — PMM-approved per issue #1116
**Channels:** Forum + Discord (Twitter/X + LinkedIn handled separately via #1115)

---

## FORUM POST

### 🚀 Org-Scoped API Keys Are Live — 2026-04-20

**CrewAI gives you teams. Molecule AI gives you teams you can actually trust in production.**

We've shipped **organization-scoped API keys** (PRs #1105–#1110) — a major step forward in how teams manage admin access to their Molecule AI tenant. Org-scoped keys are built in, not bolted on.

**What's new:**

Every organization can now mint, name, and revoke their own API keys — no more relying on a single shared `ADMIN_TOKEN` env var that nobody can rotate without ops intervention. Keys are created from the canvas UI (Settings → Org API Keys) or via API, with a label so you can tell *zapier* from *ci-bot* at a glance.

- **Named + revocable** — give each integration its own key; revoke individually, instantly
- **Surgical blast-radius control** — rotate one key without touching your whole stack
- **Audit trail** — every request carries `org:keyId` prefix; know exactly which pipeline made which call
- **Full org scope** — manage all workspaces, channels, secrets, templates, and approvals
- **Breaks the ADMIN_TOKEN dependency** — reduces your single point of failure for production deployments
- **Rate-limited minting** — 10 mints/hour per IP to prevent abuse

> *"No ADMIN_TOKEN single point of failure. Org-level key rotation without touching your whole stack."*

📖 **Docs:** `docs/guides/org-api-keys.md` | **UI:** Settings (⌘,) → Org API Keys tab

---

### 📋 FAQ: Org-Scoped Keys for Enterprise Teams

**Q: How are org-scoped keys different from personal/workspace tokens?**
Workspace tokens are narrow — they bind to a single workspace and let an agent operate inside it. Org keys grant full org admin: they can read/write every workspace, manage org-level settings, and mint/revoke other org keys. Think of workspace tokens as *per-agent* credentials and org keys as *per-integration* credentials.

**Q: Can I limit what a key can access?**
Not yet. Currently every org key grants full org admin. Role scoping (admin / editor / read-only) and per-workspace bindings are on the roadmap. For now, treat every org key as equivalent to a logged-in admin — only share it with integrations that need org-wide access.

**Q: What happens if a key is leaked?**
Revoke it immediately from Settings → Org API Keys. Revocation is instant. Mint a replacement key right away. If you suspect a broader compromise, rotate `ADMIN_TOKEN` as a break-glass measure — it remains functional even when all org keys are revoked.

**Q: How do I audit key usage?**
Each key row records a `created_by` field:
- `"session"` — minted from the browser UI
- `"org-token:<prefix>"` — minted by another org key (chain of custody visible)
- `"admin-token"` — minted using `ADMIN_TOKEN` directly

`last_used_at` is updated on every authenticated request. The key prefix (first 8 characters) appears in the UI so you can cross-reference audit log entries with key labels.

**Q: Are there rate limits?**
- **Mint**: 10 requests per hour, per IP (prevents a compromised session from minting unlimited keys)
- **List / Revoke**: standard global rate limiter
- **Use a valid key**: no per-key rate limit; standard request limits apply

**Q: Can a key access other tenants?**
No. Each tenant's `org_api_tokens` table is isolated. A key for org A cannot authenticate to org B.

**Q: Do keys expire?**
Not yet. Tokens live until explicitly revoked. Expiry / TTL is planned but not shipped yet.

**Q: Can I migrate away from `ADMIN_TOKEN`?**
Yes. Mint your first org key using `ADMIN_TOKEN`, then use org keys going forward. `ADMIN_TOKEN` still works as a break-glass fallback.

---

**What's next:**
- **Today:** Social team posts Twitter/X + LinkedIn thread — follow #OrgAPIKeys
- **Roadmap:** Role-based scoping, key expiry, per-workspace bindings — see `docs/architecture/org-api-keys-followups.md`

Questions? Drop them below or [open a GitHub issue](https://github.com/Molecule-AI/molecule-core/issues).

---

## DISCORD POST (3 messages, stay under 2000 chars each)

### Message 1 — Announcement

🚀 **Org-Scoped API Keys Are Live — 2026-04-20**

**CrewAI gives you teams. Molecule AI gives you teams you can actually trust in production.**

We've shipped organization-scoped API keys (PRs #1105–#1110). Org-scoped keys are built in, not bolted on.

Every org can now mint, name, and revoke their own API keys — no more relying on a single shared `ADMIN_TOKEN` that nobody can rotate without ops intervention.

### Message 2 — Key Features

**What you can do now:**
• Give each integration its own named key — revoke individually, instantly
• Rotate one key without touching your whole stack
• Audit trail shows `org:keyId` on every call — know exactly which pipeline made which request
• Manage all workspaces, channels, secrets, templates, and approvals from one key
• Breaks the `ADMIN_TOKEN` single point of failure for production deployments
• Rate-limited minting: 10 mints/hour per IP

**Docs:** `docs/guides/org-api-keys.md` | Settings → Org API Keys tab

### Message 3 — FAQ + CTA

📋 **FAQ for enterprise teams** (see docs for full detail):

Q: Org keys vs workspace tokens? → Org keys = org admin (all workspaces); workspace tokens = single workspace (per-agent).
Q: Can I scope a key to fewer permissions? → Not yet — role scoping on roadmap. Treat every org key as an admin equivalent.
Q: Key leaked? → Revoke instantly from Settings → Org API Keys. `ADMIN_TOKEN` remains as break-glass fallback.
Q: Audit trail? → `created_by` field tracks minting origin (session / org-token / admin-token). `last_used_at` updated on every request.
Q: Rate limits? → Mint: 10/hr/IP. Use key: no per-key limit.

**Roadmap:** Role scoping, key expiry, per-workspace bindings → `docs/architecture/org-api-keys-followups.md`

Questions? Open a GitHub issue or drop it here.

#OrgAPIKeys