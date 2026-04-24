# Org-Scoped API Keys — Social Copy
**Campaign:** Org-Scoped API Keys | **Blog:** `docs/blog/2026-04-25-org-scoped-api-keys/index.md`
**Canonical URL:** `docs.molecule.ai/blog/org-scoped-api-keys`
**Status:** APPROVED — URL and asset fixes applied by PMM (2026-04-25 Day 5 pre-publish)
**Owner:** PMM → Social Media Brand | **Launch:** Coordinated with PR #1342 merge

---

## X (140–280 chars)

### Version A — Security framing
```
Every integration. One credential. Zero shared secrets.

Org-scoped API keys: named, revocable, with full audit trail. Rotate without downtime. Attribute every call back to the key that made it.

Your security team called — this is the answer.
```

### Version B — Production use cases
```
Three things that break at scale with a shared ADMIN_TOKEN:

1. You can't rotate without downtime
2. You can't tell which agent called your API
3. Compromised token = everything compromised

Org-scoped keys fix all three.
```

### Version C — Developer angle
```
How to give a CI pipeline its own API key:

1. POST /org/tokens with a name
2. Store the token (shown once)
3. Done.

That's it. Named. Revocable. Audited.
```

### Version D — Enterprise angle
```
Replace your shared ADMIN_TOKEN.

Org-scoped API keys: one per integration, immediate revocation, full audit trail. Rotate without coordinating downtime.

Tiers: Lazy bootstrap → WorkOS session → Org token → ADMIN_TOKEN (break-glass).

Security teams love this architecture.
```

---

## LinkedIn (100–200 words)

```
When your engineering team scales from two agents to twenty, a single ADMIN_TOKEN hardcoded in your environment is a single point of failure.

Org-scoped API keys give every integration its own credential: named, revocable, with full audit trail. Rotate without coordinating downtime across ten agents. Identify exactly which integration called your API. Revoke one key without touching the others.

The security model: tier-based authentication priority (WorkOS session first, org tokens primary for service integrations, ADMIN_TOKEN as break-glass only). When a request arrives, the platform checks in priority order — and every org API key call is attributed in the audit log with its key prefix and creation provenance.

Every call traced. Every key revocable. Every rotation zero-downtime.

Navigate to Settings → Org API Keys in the Canvas, or use the REST API directly.

→ docs.molecule.ai/blog/org-scoped-api-keys
```

---

## Image suggestions

| Post | Image | Source |
|---|---|---|
| X Version A | `before-after-credential-model.png` — shared key vs org-scoped (red/green table) | `campaigns/org-api-keys-launch/` |
| X Version B | 3-item checklist: Rotate without downtime / Attribute every call / Revoke one key | Custom graphic |
| X Version C | `audit-log-terminal.png` — terminal showing token creation and audit attribution | `campaigns/org-api-keys-launch/` |
| X Version D | Auth tier hierarchy: Lazy bootstrap → WorkOS → Org token → ADMIN_TOKEN (break-glass) | Custom graphic |
| LinkedIn | `canvas-org-api-keys-ui.png` — Canvas Settings → Org API Keys tab | `campaigns/org-api-keys-launch/` |

**Do NOT use:** `phase30-fleet-diagram.png` — wrong visual for this campaign.

**CTA URL:** `docs.molecule.ai/blog/org-scoped-api-keys` *(corrected from `moleculesai.app/blog/deploy-anywhere`)*

---

## Hashtags

`#MoleculeAI #APIKeys #EnterpriseSecurity #A2A #DevOps #MultiAgent`

---

## UTM

`?utm_source=linkedin&utm_medium=social&utm_campaign=org-api-keys-launch`
