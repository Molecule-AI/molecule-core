---
name: Invoices Publish Without Prices by Default
description: When publishing invoices to InvoiceSimple, omit prices unless explicitly provided
type: feedback
---

Invoices should be published without prices by default — scope-only estimates.

**Why:** User requested 2026-04-04. Prices are added later after review, not auto-generated.

**How to apply:** When calling publish_estimate, don't include `rate` in sections unless the user explicitly provides prices. The estimate serves as a scope document first.
