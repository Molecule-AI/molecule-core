---
name: invoicing
description: Build renovation estimates/invoices from user specs (Telegram messages, PDFs, voice notes). Use when the user sends project scope via Telegram or asks to create an estimate/invoice/quote. Covers the full flow from parsing requirements to publishing on InvoiceSimple.
---

# Invoicing Skill — Reno Stars

## MCP Tools (in order of use)

1. `mcp__reno-stars-invoice__list_catalog` — Call first to see all available models, modifiers, and rules
2. `mcp__reno-stars-invoice__build_section` — Build each section (kitchen, bathroom, painting, flooring, etc.)
3. `mcp__reno-stars-invoice__assemble_invoice` — Combine all sections into a formatted markdown invoice
4. `mcp__reno-stars-invoice__publish_estimate` — Publish to InvoiceSimple (returns URL)

## Input Formats

### Telegram text (Chinese shorthand)
User sends structured Chinese text like:
```
范围：Kitchen
Model：Prefab cabinet
Keep：appliances
Demolish: closet x2
Add：LED light strip, island (72''x24'')
Replace：stone backsplash
```
- 范围 = Scope/Section
- 不拆/保留/Keep = Keep items
- 拆/Demolish = Extra demolition items
- 加/Add = Addons/modifiers
- 换/Replace = Replacements
- **Chinese markers do NOT mean Chinese invoice** — default language is always English unless explicitly requested

### PDF with handwritten notes + photos
- Use `mcp__plugin_telegram_telegram__download_attachment` to get the file
- Convert PDF to images: `pdftoppm -jpeg -r 200 input.pdf /tmp/output`
- Read EACH page image carefully
- **Cross-reference photos with text notes** — photos reveal the ACTUAL current state:
  - Glass door vs shower curtain (look at the tub/shower area)
  - Vanity size (look for green measurement markings)
  - What's being kept (green "keep" annotations)
  - Current fixture types (gold vs chrome, framed vs frameless)
  - Room layout and condition

### Key lesson (learned 2026-04-09):
**Always check photos for current fixture details.** The build_section models default to generic items (e.g. "shower curtain" in demolition) but the site may have something different (e.g. a gold-framed glass sliding door). The demolition list must match what's actually there.

## Build Flow

### 1. Parse the requirements
- Extract: client name, address, language preference
- Identify all sections (wall changes, kitchen, each bathroom, flooring, painting, etc.)
- For each section: model type, keep items, demolish items, addons, replacements, custom items

### 2. Check for ambiguities BEFORE building
The build_section tool description lists what to check:
- Contradictions (keep + demolish same item)
- Unusual quantities
- Missing info (vanity size, cabinet style, stone code)
- If PDF/photo input: verify photo details match text notes

### 3. Build sections
Call `build_section` once per section. Key rules:
- **Each bathroom is a separate section** with its own label (Master Bathroom, Hallway Bathroom, etc.)
- **Section labels should be generic location names** — e.g. "1st Floor Bathroom", "Master Bathroom", "Laundry Room", NOT "1st Floor Bathroom (Tub to Tiled Shower)". Don't include the model type in the label.
- **Cross-section dependencies**: if painting section exists, don't add paint addons to bathrooms. If flooring section exists, don't add floor addons to kitchen.
- **Modifier trigger rules** (from the tool description):
  - Kitchen: "relocate stove/sink" → include `kitchen-electrical-plumbing-venting`
  - Kitchen: "stone backsplash" → include `quartz-backsplash` replacement
  - Bathroom: "relocate vanity light/drainage" → include `bathroom-electrical-plumbing-venting`
  - Bathroom: "niche" → include `bathroom-niche` with size + edge type
  - Bathroom: "bench" → include `bathroom-bench`
  - Bathroom: "LED mirror" → include `bathroom-led-mirror`

### 4. Choose payment schedule
- `70/30` — small jobs (1-2 sections, no plumbing/electrical)
- `milestone-5` — multi-section with plumbing/electrical
- `milestone-large` — large renos with cabinets (kitchen + multiple bathrooms)

### 5. Assemble and publish
- `assemble_invoice` saves markdown locally
- `publish_estimate` pushes to InvoiceSimple and returns a URL
- If publish fails (timeout), retry once. If still fails, send the markdown version.

### 6. Reply on Telegram
Send the InvoiceSimple URL + brief summary of sections to the Telegram chat.

## Common Bathroom Models

| User says | Model ID |
|---|---|
| Tub to tub | `bathroom-tub-to-tub` |
| Tub to tiled shower | `bathroom-tub-to-tiled-shower` |
| Tub to prefab shower | `bathroom-tub-to-prefab-shower` |
| 4 piece / separate tub + shower | `bathroom-4piece` |
| Powder room / laundry vanity | `bathroom-powder-room` |
| Shower only | `bathroom-shower-only-tiled` |

## Common Kitchen Models

| User says | Model ID |
|---|---|
| Prefab cabinet | `kitchen-prefab-cabinet` |
| Custom cabinet | `kitchen-custom-cabinet` |

## Cabinet Styles
White Shaker, Grey Shaker, Navy Blue Shaker, Wood Veins — determine from user prompt. Default: White Shaker for kitchen, varies for bathroom.

## Vanity Styles for Prefab
Same as cabinet styles. The user often specifies per-bathroom (e.g. "wood veins" for bathrooms, "white shaker" for laundry).

## Glass Door Types
- **Prefab Tempered** — standard for tub-to-tub (sliding, frameless)
- **Custom Tempered L-shape** — for tiled shower conversions (hinged)
- Accessories color: Chrome (default), Black (if specified)

## Niche Options
- Standard: metal edge
- Upgrade: miter edge
- LED strip: optional addon inside niche
- Size: usually 12''x20'' — confirm with user

## CRITICAL: Custom Steps Policy

**Custom steps are a LAST RESORT.** The MCP system is designed for zero AI text generation — all invoice text should come from the typed step classes.

Before using `customSteps`:
1. Check `list_catalog` for matching modifiers
2. Check `describe_item` on the base model to see if the feature is already a parameter
3. Check if it can be expressed via `modifierIntents` (e.g. "relocate vanity light" → EPV modifier intent)
4. Check if it's a sub-remark on an existing step (e.g. "garbage pull-out" is a remark on cabinets, not a separate step)
5. **If nothing fits, ASK THE USER** before adding a custom step: "I can't find [X] in the system — can I add it as a custom line item?"

**Common mistakes to avoid:**
- Island is a cabinet parameter (`cabinet.island: true`), NOT a custom step — don't duplicate
- Extra drawers are a remark on the cabinet step ("If need extra drawers $150/Each"), NOT a custom step
- Relocate outlets → use the EPV/electrical modifier, NOT a custom step
- Potlights → use the potlights modifier with intent string, NOT a custom step
- Garbage pull-out, microwave trim kit → these ARE legitimately custom items (not in the system)

## Vanity Size

Leave vanity size empty `()''` in estimates — this is measured on-site later during the detailed measurement visit. Do NOT guess or estimate from photos. The parentheses are intentional placeholders.

## Things That Often Go Wrong

### Source Interpretation
1. **Read source for actual site fixtures** — check transcript/photos/AI summary for what's ACTUALLY at the site. Glass doors vs shower curtains, acrylic vs tiled base, existing fixture types. Don't trust model defaults.
2. **Keep vs demolish default** — if the PDF/user doesn't mention replacing an item, default to KEEP and REINSTALL. Check photos for items in good condition.
3. **Don't overread handwritten notes** — annotations on photos may be observations/measurements, not scope items. Ask if unsure.
4. **"Reinstall" vs "Install"** — when keeping an existing fixture, use "Reinstall" not "Install" (exhaust fans, light fixtures, hardware removed during demo then put back).

### Scope Decisions
5. **Don't add unrequested GFCIs** — only include GFCIs explicitly mentioned in the scope. Don't assume every bathroom needs both vanity + toilet GFCI.
6. **Always check for stairs** — when flooring scope exists, check if stairs are included. The `flooring-stairs` modifier exists.
7. **Laundry is NOT a powder room** — for laundry rooms, use the vanity-only model (just vanity + countertop + sink + faucet). Don't include toilet, mirror, exhaust fan, hardware.
8. **Paint scope after popcorn removal** — means ceiling full repaint + walls TOUCH-UP ONLY (just work areas). Full wall repaint is a separate bigger scope.
9. **Countertop quantities** — always fill `x1` not `x()`. There's always at least 1 countertop and 1 backsplash.

### System Usage
10. **Cabinet add-ons are sub-remarks** — garbage pull, built-in microwave, extra drawers go as sub-remarks on the cabinet step via `cabinetOptions` parameter. NOT as separate custom steps.
11. **Use EPV modifier intents for relocations** — "relocate outlets x3" → use `modifierIntents` on the EPV modifier. NOT a custom step.
12. **Prefab glass door is default** — don't assume Custom+L just because it's a tiled shower. Custom only when explicitly requested or layout requires it (e.g. L-shape enclosure).
13. **Glass door is default in tub-to-tub demolition** — the system now defaults to "glass door" in demolition. Only specify "shower curtain" if that's what's actually there.
14. **Niche edge: metal is default, miter is substitute** — only specify miter when client explicitly requests it.

### Structure
15. **Paint section conflicts** — if painting section exists, do NOT add paint addons to individual bathrooms
16. **Duplicate items** — island in cabinet params AND as custom step = double-counted
17. **Unused default steps from models** — rough-in model includes 7 default steps. Strip irrelevant defaults.
18. **Freestyle text instead of modifiers** — ALWAYS use the modifier system first. Custom steps should be rare.
19. **Floor protection** — always include for any renovation project.
20. **Electrical in its own section** — potlights, switches go in "Others" section, NOT inside painting or bathroom.
21. **Flooring thickness** — default is 6-7mm, user may specify 9mm
22. **Popcorn ceiling** — "keep popcorn" means paint over it, "popcorn removal" is an addon
