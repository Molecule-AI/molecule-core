---
name: Publish InvoiceSimple estimates directly — don't ask for review first
description: For InvoiceSimple estimates (not final invoices), publish via the invoice MCP without a separate scope-review step, then send the link
type: feedback
---

When generating a quote for InvoiceSimple as an Estimate, publish it directly via `mcp__reno-stars-invoice__publish_estimate` and send the link. Do NOT save the markdown locally first and ask the user to confirm scope before publishing.

**Why:** On 2026-04-08 the user (Hongming) corrected me after I built the Harmi quote, saved it locally, and asked for review before publishing. They said: "remember that you can just publish and then send the link, because its an estimate anyway, so we dont have to take a extra step." Estimates in InvoiceSimple are inherently draft state — they can be edited, re-sent, declined, or deleted. There's no consequence to publishing them as-is, and the user prefers to review/edit on the InvoiceSimple side directly rather than through a Telegram round-trip.

**How to apply:**
1. Build all sections via `build_section` as usual.
2. Call `publish_estimate` directly (skip `assemble_invoice` if local markdown isn't needed for some other reason — optional for record-keeping).
3. If pricing isn't known, publish with `rate: 0` per section. The user fills in prices on the InvoiceSimple UI.
4. Send the InvoiceSimple URL via Telegram. Surface any flags/decisions the user should know about (defaults you picked, modifiers you guessed, sections that didn't fit a standard model) AFTER the link, not before.
5. This ONLY applies to **estimates**. If the workflow is ever for a real invoice (final, accounting-of-record), revert to scope-review-first.

**Caveat:** Publishing requires `INVOICE_SIMPLE_ACC` and `INVOICE_SIMPLE_SECRET` env vars, and uses Chrome CDP via Playwright (Verisoul bot detection blocks pure headless — see `feedback_invoicesimple_login.md`). If publish fails, fall back to scope-review-via-Telegram and tell the user the publish path is broken.
