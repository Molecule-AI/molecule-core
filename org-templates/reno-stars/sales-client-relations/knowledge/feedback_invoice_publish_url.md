---
name: Invoice Publish URL Bug
description: publish_estimate returns generic URL instead of actual estimate URL
type: feedback
---

The invoice MCP's publish_estimate tool returns `https://app.invoicesimple.com/estimate/new` instead of the actual estimate URL (e.g. `https://app.invoicesimple.com/estimate/OfUguhA7Jz`).

**Why:** The Playwright automation creates the estimate but doesn't capture the final URL after save. It returns the creation page URL.

**How to apply:** After publishing, manually check InvoiceSimple for the actual URL until this is fixed in the reno-star-invoice-automation repo. The estimate number (e.g. EST221344) is correct — use it to find the estimate.
