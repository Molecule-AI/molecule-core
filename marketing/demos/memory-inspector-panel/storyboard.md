# Screencast Storyboard — MemoryInspectorPanel

> **Feature:** `canvas/src/components/MemoryInspectorPanel.tsx` | **Duration:** 60 seconds
> **Format:** Canvas-led with API/terminal overlay cuts

---

## Pre-roll (0:00–0:04)

**Canvas — full screen**
Single workspace card in Canvas: `data-agent [ONLINE]`. Active tab: `Memory`.

Memory panel visible in right sidebar with an empty state: `◇ No memory entries yet`.

Narration (0:00–0:04):
> "This agent has been running for an hour. It's stored task results, intermediate findings, and context across its sessions. Now there's a way to inspect it."

**Camera:** Static Canvas frame. 4-second hold. No cursor.

---

## Moment 1 — Agent writes memory entries (0:04–0:18)

**Cut to:** Terminal window, dark theme.

Prompt: `agent@data-agent:~$`

```bash
# Agent writes to its KV memory store
curl -s -X POST \
  "https://acme.moleculesai.app/workspaces/ws-data-agent-001/memory" \
  -H "Authorization: Bearer ws-token-xxx" \
  -H "Content-Type: application/json" \
  -d '{
    "key": "pipeline/Q1-revenue-findings",
    "value": {
      "region": "APAC",
      "delta": -0.073,
      "contributors": ["enterprise-churn", "demo-loss"],
      "confidence": 0.91
    }
  }' | jq
```

**Terminal output:**

```json
{
  "status": "ok",
  "key": "pipeline/Q1-revenue-findings",
  "version": 1
}
```

**Camera:** Brief type-in. Cursor moves. JSON response hold on `version: 1` — 1.5s.

Narration (0:06–0:14):
> "One API call writes a memory entry. Structured data — a key, a value, a version counter. The agent stores what it learns, not just what it's told."

**Second entry:**

```bash
curl -s -X POST \
  "https://acme.moleculesai.app/workspaces/ws-data-agent-001/memory" \
  -H "Authorization: Bearer ws-token-xxx" \
  -H "Content-Type: application/json" \
  -d '{
    "key": "pipeline/anomalies-detected",
    "value": ["deals-84-day-stale", "renewal-risk-enterprise-3"]
  }' | jq
```

```json
{
  "status": "ok",
  "key": "pipeline/anomalies-detected",
  "version": 1
}
```

**Camera:** Quick follow. Clean output. Clean finish.

Narration (0:14–0:18):
> "Version one. Each write is tracked. The agent has a memory — observable, queryable, inspectable."

---

## Moment 2 — Canvas Memory panel shows the entries (0:18–0:32)

**Cut to:** Canvas — data-agent workspace, Memory tab active.

Two entry rows visible:

- `pipeline/Q1-revenue-findings` — `v1` — `2m ago` — right chevron
- `pipeline/anomalies-detected` — `v1` — `1m ago` — right chevron

Entry count badge: `2 entries` in toolbar.

**Camera:** Slow scroll through the two entries. Hold on first row. Highlight `v1` badge and `2m ago` timestamp.

Narration (0:20–0:28):
> "Back in Canvas. The Memory tab shows every entry — key, version, how long ago it was written. Click any row to see the full value."

**Expand first entry:**

Click on `pipeline/Q1-revenue-findings` row.

Row expands: JSON body shows `delta: -0.073`, `confidence: 0.91`, `contributors: [...]`.

Below JSON: `Updated: Apr 21, 12:34 PM` — `Edit` button — `Delete` button.

**Camera:** Full expand animation. Hold on JSON body. Press Edit.

Narration (0:28–0:32):
> "One click expands the entry. JSON in view. Edit inline. Version conflict detection built in. That's the Memory Inspector — read, write, version-tracked."

---

## Moment 3 — Semantic search with similarity scores (0:32–0:46)

**Canvas continues:** memory panel still open.

Search bar focused. Type: `revenue trends`

**Camera:** Cursor to search bar. Type animation (playback speed 2x). 300ms debounce, then request fires.

**Terminal overlay (brief):**

```bash
# Semantic query — pgvector backend
curl -s "https://acme.moleculesai.app/workspaces/ws-data-agent-001/memory?q=revenue%20trends" \
  -H "Authorization: Bearer ws-token-xxx" | jq '.'
```

**Canvas updates:** entries re-sort. Similarity badges appear:

- `pipeline/Q1-revenue-findings` — `v2` — `91%` — blue badge
- `pipeline/anomalies-detected` — `v1` — `~34%` — italic dim badge

Top result highlighted with blue ring for 1.5s.

**Camera:** Smooth re-sort. Badge fade-in. Hold on `91%` badge. Brief highlight pulse.

Narration (0:34–0:44):
> "Type a semantic query — 'revenue trends'. The pgvector backend ranks entries by similarity. 91% match on top. Clearest result surfaces first."

**Callout text (bottom-left):**
`Vector search — find related entries without exact key matches.`

**Camera:** Clear search — click × button. Entries return to default order. Similarity badges disappear.

Narration (0:44–0:46):
> "Clear the search — entries return to recency order. Every query, every ranking, visible."

---

## Moment 4 — Edit and version bump (0:46–0:54)

**Canvas:** `pipeline/Q1-revenue-findings` entry expanded.

Click `Edit`. JSON textarea appears in expanded body.

Update `delta` from `-0.073` to `-0.068` — slightly revise the finding.

Click `Save`.

**Terminal overlay:**

```bash
# Edit with optimistic locking
curl -s -X POST \
  "https://acme.moleculesai.app/workspaces/ws-data-agent-001/memory" \
  -H "Authorization: Bearer ws-token-xxx" \
  -H "Content-Type: application/json" \
  -d '{
    "key": "pipeline/Q1-revenue-findings",
    "value": {
      "region": "APAC",
      "delta": -0.068,
      "contributors": ["enterprise-churn", "demo-loss"],
      "confidence": 0.91
    },
    "if_match_version": 2
  }' | jq
```

**Terminal output:**

```json
{
  "status": "ok",
  "key": "pipeline/Q1-revenue-findings",
  "version": 3
}
```

**Canvas:** Entry version badge updates: `v3` (yellow pulse on number change).

Narration (0:47–0:52):
> "Save with `if_match_version: 2`. The entry bumps to version 3. If another agent updated it in the meantime, you'd see a conflict — not a silent overwrite. That's optimistic locking on every write."

---

## Close (0:54–1:00)

**Canvas — full frame.** Memory panel open. Entries visible. Clean state.

Narration (0:54–0:58):
> "Every agent has memory. Every memory entry is versioned, searchable, editable. The Memory Inspector — in Canvas, built into every workspace."

**End card:**

```
MemoryInspectorPanel
canvas/src/components/MemoryInspectorPanel.tsx — molecule-core
```

**Fade to black.**

---

## Production Notes

- **Canvas cutaways (pre-roll + moments 2–4):** Use dev canvas with one workspace in active state and at least 2 pre-populated memory entries. Pre-record before the session. When recording Moment 4, seed `pipeline/Q1-revenue-findings` with version 2 so the edit goes to version 3.
- **Semantic search (Moment 3):** Requires pgvector backend deployed (issue #776). If pgvector is not available in the demo environment, show the empty-state search result ("No memories match your search") and narrate it as: "Without pgvector deployed, semantic search shows a clean empty state — the UI degrades gracefully without errors."
- **Terminal theme:** Same as AGENTS.md + CF Artifacts storyboards — dark zinc, JetBrains Mono 14pt.
- **Camera:** Screenflow / Camtasia. 1440×900 record → 1080p export.
- **Callout text:** Amber ring `#E8A000`, 1s fade-in/out, bottom-left at 90% opacity.
- **Version badge highlight:** On Moment 4 version bump, briefly pulse `v3` badge with blue ring `#3B82F6` — 1s hold.
- **Similarity badges:** Blue `#3B82F6` for ≥80%, gray for 50–79%, italic gray for <50%. 1px rounded pill shape.
- **VO recording:** Consistent pacing with other Phase 30 screencasts. Match voice talent.
- **Music:** No music. Consider a subtle single-tone click at 0:04 (first memory write) and 0:54 (end card).