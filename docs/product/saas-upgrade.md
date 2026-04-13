# SaaS Upgrade Path

The open-source project has **no auth**. This is intentional — the project follows the n8n Community Edition model.

## How It Works

When productizing as SaaS, a separate `molecule-cloud` repo wraps this project and adds:

| Feature | Technology |
|---------|-----------|
| Authentication | Clerk or Auth.js |
| Multi-tenancy | Org isolation (`org_id` added to schema) |
| Billing | Stripe |
| Managed infrastructure | ECS + Neon + Upstash |
| White-labelled canvas | Custom branding |

## Key Principle

**No changes to this repo are needed.** The SaaS layer is purely additive. The open-source core remains clean and self-hostable.

## Schema Changes

The MVP schema intentionally omits `org_id`. It is added in the SaaS migration for multi-tenancy isolation. This avoids cluttering the open-source schema with fields that only matter for hosted deployments.

## Related Docs

- [Constraints & Rules](../development/constraints-and-rules.md) — Design decisions that enable this
- [Architecture](../architecture/architecture.md) — System overview
- [Database Schema](../architecture/database-schema.md) — MVP schema that `org_id` extends
