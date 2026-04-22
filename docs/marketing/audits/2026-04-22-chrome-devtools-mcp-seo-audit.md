# Chrome DevTools MCP SEO Audit — 2026-04-22

**File:** `docs/blog/2026-04-20-chrome-devtools-mcp-seo/index.md`
**Audit by:** SEO Analyst
**Status on main:** ❌ NOT LIVE — file exists locally only
**Campaign:** Phase 30 Chrome DevTools MCP SEO launch

---

## Overall Verdict: ⚠️ CONDITIONAL PASS — 2 critical keyword misses

`mcp browser automation` (P0) is well-covered. `chrome devtools mcp` (P0) is **absent entirely**. Fix before publishing.

---

## 1. Title & Meta

| Element | Value | Count | Status |
|---|---|---|---|
| Title | "Give Your AI Agents MCP Browser Automation: Chrome DevTools" | 59 chars | ✅ OK (≤60) |
| Meta description | "Add browser automation to your AI agents with Chrome DevTools and the Model Context Protocol (MCP). Working Python examples — no SaaS dependencies." | 147 chars | ✅ OK (≤160) |
| Slug | `browser-automation-ai-agents-mcp` | — | ✅ Clean |
| OG title | Same as title | 59 chars | ✅ OK |
| OG description | Same as meta | 147 chars | ✅ OK |
| Canonical URL | `https://molecule.ai/blog/browser-automation-ai-agents-mcp` | — | ✅ Present |
| Twitter card | `summary_large_image` | — | ✅ Correct |

---

## 2. P0 Keywords

### ✅ `mcp browser automation` — PASS

| Check | Result |
|---|---|
| In H1 | ✅ Yes (exact match, start of H1) |
| In first 100 words | ✅ Yes — appears in subtitle and opening paragraph |
| Total occurrences | 4 — natural usage throughout |
| Notes | Strong coverage, exact-match H1 |

### ❌ `chrome devtools mcp` — FAIL

| Check | Result |
|---|---|
| In H1 | ❌ No — H1 has both terms separately but not as compound phrase |
| In first 100 words | ❌ No |
| Total occurrences | 0 |
| Notes | **Critical.** This is a P0 keyword from the brief. The compound phrase `chrome devtools mcp` does not appear anywhere in the document. The keyword brief explicitly lists this as a P0 informational/product keyword targeting the blog H2 + meta description. It should appear in the H1 subtitle or intro paragraph. |

**Fix required before publish:** Add `chrome devtools mcp` as a subtitle or H2 beneath the main H1. Suggested: `> **Chrome DevTools MCP** — browser automation for AI agents, powered by the Model Context Protocol`

---

## 3. P1 Keywords

### `ai agent browser control` — ABSENT

| Check | Result |
|---|---|
| In body | ❌ No occurrences |
| Notes | Not used in the tutorial or body sections. This is a P1 informational keyword — should appear in at least one body section heading or paragraph. Suggested injection: a section heading "AI Agent Browser Control via MCP" or natural usage in the use-cases section. |

### `mcp protocol tutorial` — ABSENT

| Check | Result |
|---|---|
| In body | ❌ No occurrences |
| Notes | Not used. P1 tutorial/how-to keyword — should appear in the setup or code example sections. Suggested injection: add "MCP protocol tutorial" to the MCP Server section heading, or a callout box describing the MCP protocol shape. |

---

## 4. Structured Data

| Check | Status |
|---|---|
| JSON-LD Article schema | ✅ Present |
| `og:title` / `og:description` / `og:image` | ✅ All present |
| `twitter:card` | ✅ `summary_large_image` |
| Canonical URL | ✅ Present |
| `keywords` frontmatter | ✅ Present |

---

## 5. Content Quality

| Check | Result |
|---|---|
| Word count | 1,796 words |
| Est. reading time | 9 min |
| Code examples | ✅ 3 complete Python examples + MCP tool schema + shell commands |
| Internal links | 2 (to MCP Server Setup Guide and Quickstart) |
| External links | 2 (GitHub repo) |
| Images / alt text | 0 images — no screenshot/diagram assets |
| FAQ section | ✅ Present — 6 FAQs covering browser compatibility, headless, Playwright comparison, session recovery, multi-tab, cloud tier |

---

## 6. SEO Action Items (Before Publish)

| Priority | Item | Owner |
|---|---|---|
| 🔴 P0 | Add `chrome devtools mcp` compound phrase to H1 subtitle or intro paragraph | Content Marketer |
| 🟡 P1 | Add `ai agent browser control` to body — suggested: use-case section or a subheading | Content Marketer |
| 🟡 P1 | Add `mcp protocol tutorial` to MCP Server section or setup steps | Content Marketer |
| 🟢 P2 | Consider adding 1-2 inline screenshots/diagrams with descriptive alt text for visual variety | DevRel |
| 🟢 P2 | Add OG image asset: `assets/blog/2026-04-20-chrome-devtools-mcp-seo/og.png` | DevRel (PR #1530 already adds `og.png` for non-SEO version — coordinate) |

---

## 7. Blog Post Not Live on Main

The SEO-optimized version at `docs/blog/2026-04-20-chrome-devtools-mcp-seo/index.md` is **not on `origin/main`**. The non-SEO version at `docs/blog/2026-04-20-chrome-devtools-mcp/index.md` is also not on `origin/main`.

Coordinate with DevRel/PMM to determine:
1. Which version is canonical (SEO or non-SEO)?
2. Is the non-SEO version replacing the SEO version, or are they separate posts?
3. Which branch should the SEO version land on?

---

*Audit completed 2026-04-22 by SEO Analyst*
