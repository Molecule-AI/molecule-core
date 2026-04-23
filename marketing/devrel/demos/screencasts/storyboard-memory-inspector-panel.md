# Screencast Storyboard — MemoryInspectorPanel
**Feature:** `canvas/src/components/MemoryInspectorPanel.tsx`
**Duration:** 60 seconds | **Format:** Canvas UI-led, dark zinc theme

---

## Pre-roll (0:00–0:04)

**Canvas — workspace panel open**
Sidebar showing `pm-agent [ONLINE]`. User clicks into the Memory tab.

Narration (0:00–0:04):
> "Every agent accumulates knowledge over time — facts, decisions, context. Molecule AI's memory inspector gives you a first-class view of what your agent knows."

**Camera:** Static Canvas panel. Clean frame. No cursor movement in first 3s.

---

## Moment 1 — Memory list loads (0:04–0:14)

**Panel populated:**
Three memory entry cards visible:
- `user-preferences:v3` — blue badge "Similarity: 92%" — "2h ago"
- `project-context:v1` — "4h ago"
- `latest-decision:v5` — "1d ago"

Each card shows: key (blue mono), version counter, similarity badge (if query active), relative timestamp, expand arrow.

**Camera:** Smooth scroll through the list. Hold 2s on the first entry.

Narration (0:05–0:12):
> "The inspector loads all memory entries — keys, versions, freshness. When semantic search is active, it shows a similarity score — how closely each entry matches your query."

**Callout text (bottom-left):**
`Semantic search. Meaning, not just keywords.`

---

## Moment 2 — Semantic search (0:14–0:26)

User types in the search bar: `customer pricing`

**Camera:** Cursor moves to search input. Type-in animation.

Search bar shows: "Semantic search…" placeholder, debounce spinner (300ms), then results update.

List re-sorts:
- `user-preferences:v3` — blue badge "Similarity: 87%" (moved to top)
- `latest-decision:v5` — "Similarity: 34%" (new position)
- `project-context:v1` — "Similarity: 12%" (bottom)

**Camera:** Smooth scroll showing re-sorted results.

Narration (0:16–0:23):
> "Type a query. After 300 milliseconds — no submit button — the list re-sorts by semantic similarity. Entries below 50% fade to a lower contrast. The agent found what it knows about pricing decisions."

**Callout text:**
`300ms debounce. No submit. No page reload.`

---

## Moment 3 — Expand + Edit a memory entry (0:26–0:44)

User clicks `user-preferences:v3`.

**Camera:** Entry expands. Card opens downward.

**Expanded content shown:**
```json
{
  "preferred_tier": "enterprise",
  "pricing_sensitivity": "high",
  "last_interaction": "2026-04-18",
  "notes": "Requested SSO before trial"
}
```

Metadata below: "Updated: 2026-04-20 14:32:11", Edit button, Delete button.

User clicks **Edit**.

**Camera:** Textarea appears, pre-filled with JSON. Cursor blinks.

User edits: changes `"pricing_sensitivity": "high"` → `"medium"`.

User clicks **Save**.

**Camera:** Blue "Saving…" spinner (1s). Then: textarea closes, entry collapses, entry updates in list — `user-preferences:v4` (version increment shown).

Narration (0:28–0:40):
> "Click any entry. See the full JSON — every fact the agent stored. Edit directly in the panel. Save — it's versioned, timestamped, persisted. No API calls to remember."

**Callout text:**
`Version conflict detection. Optimistic updates. Never lose a write.`

---

## Moment 4 — Delete entry (0:44–0:54)

User clicks the red Delete button on `project-context:v1`.

**Delete confirmation dialog appears:**
`Delete key "project-context"? This cannot be undone.`

User clicks **Delete**.

**Camera:** Dialog closes. Entry animates out. List collapses. Count decrements: "2 entries" shown in toolbar.

Narration (0:46–0:52):
> "Delete with confirmation. Entries are removed from the memory store immediately. Canvas updates in real time."

---

## Close (0:54–1:00)

**Panel clean frame.** Two entries remaining.

Narration (0:54–0:58):
> "The memory inspector — semantic search, in-line editing, version history, and full delete. Everything your agent knows, visible and editable."

**End card:**
```
MemoryInspectorPanel
canvas/src/components/MemoryInspectorPanel.tsx
```
**Fade to black.**

---

## Production Spec

| Spec | Value |
|------|-------|
| Theme | Dark zinc, blue accents (`#3B82F6`), SF Mono 11-14pt |
| Canvas | Dev canvas localhost:3000, pre-record workspace with 3+ memory entries |
| Camera | Screenflow / Camtasia, 1440×900 → 1080p export |
| Type-in animation | Realistic cursor blink, natural typing speed |
| Dialog | Center modal with red "Delete" button |
| Callout highlight | Amber ring `#E8A000`, 1s fade-in/out |
| VO voice | en-US-AriaNeural (consistent with other storyboards) |
| Music | None |
| Speed | Moment 1 at 2x playback for log-scroll effect |
