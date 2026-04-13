---
name: Invoice custom steps policy
description: Never use customSteps in invoices unless the item genuinely doesn't exist in the MCP system — always check modifiers/params first, then ask user before adding custom text
type: feedback
---

Custom steps are a LAST RESORT in the invoice MCP system. The whole design is zero AI text generation — all text comes from typed step classes.

**Why:** On 2026-04-09, built the Harmi invoice with 5+ custom steps that should have been handled by existing modifiers (island duplicated, extra drawers already a remark, relocate outlets = EPV modifier, potlights = potlights modifier). This defeats the purpose of the structured system and leads to inconsistent invoices.

**How to apply:**
1. Before adding any customStep, check list_catalog + describe_item for matching modifiers/params
2. If nothing fits, ASK the user: "I can't find [X] in the system — can I add it as a custom line item?"
3. Only legitimate custom items: garbage pull-out, microwave trim kit, window installation — things genuinely not in any model/modifier

Also: vanity size `()''` is intentionally empty in estimates — measured on-site later. Don't fill it from photos.
Also: always cross-reference PDF photos with text notes — demolition list must match actual site fixtures (glass door ≠ shower curtain).
