# Discord Adapter Storyboard
> Source: PR #1209
> Visual asset spec for design team

---

## Asset 1: discord-molecule-logo-combo.png (1200×800)

**Background:** Discord-dark (#1E1E2E) with subtle radial blurple glow (Molecule brand purple/cyan)
**Center:** Discord logo (official mark, white) + Molecule AI logo (wordmark, white) flanking a small AI connection node graphic
**Bottom third:** Tagline in Inter Bold: "AI agents, live in Discord."
**Bottom edge:** Small molecule-atomic logo in brand cyan

---

## Asset 2: discord-community-signal-flow.png (1200×600)

**Layout:** Left-to-right horizontal flow diagram, dark #1E1E2E background

**Left node:** Discord logo + "User" label in a rounded box (#5865F2 border)
**Arrow (→):** Animated dashed line in brand cyan

**Center node:** "Community Manager" agent card (Molecule AI canvas aesthetic — subtle border, workspace-name in monospace)
**Label below:** "Molecule AI Agent"

**Arrow (→):** Second dashed line

**Right node:** Three outcome icons in a row:
- 🔧 "→ Security Lead" (issue detected)
- ✅ "→ Status response" (query answered)  
- 📋 "→ Support ticket" (ticket created)

**Bottom caption:** "Slash commands. Status updates. Ticket routing. — without leaving Discord."

---

## Asset 3: discord-slack-command-mockup.png (1200×900)

**Layout:** Faux Discord UI screenshot with interaction overlay

**Discord UI (base):**
- Dark theme (#36393F) message area
- Channel name: `#agent-deployments` in sidebar
- Message thread showing agent activity

**Overlay annotations (callout arrows):**
1. **Slash command input** — callout pointing to `/status production` in the message box, labeled "User invokes /status"
2. **Agent response block** — a rendered message block in cyan (#00AFF4 border) showing:
   ```
   🟢 production — 99.7% uptime
   Last deploy: 2h ago (v2.1.4)
   Agents: 3 active, 12 tasks
   ```
   Labeled "Agent responds in <2s"
3. **Metadata badge** — small badge on the agent message: "via Molecule AI • Discord Adapter"

---

## Color Palette Reference

| Element | Value |
|---|---|
| Discord brand | `#5865F2` |
| Background | `#1E1E2E` |
| Surface | `#2F3136` |
| Message bg | `#36393F` |
| Brand cyan | `#00AFF4` |
| Text primary | `#FFFFFF` |
| Text muted | `#B9BBBE` |
| Success green | `#57F287` |
| Warning yellow | `#FEE75C` |
| Error red | `#ED4245` |

---

*Design team: use official Discord brand assets only (discord.design)*