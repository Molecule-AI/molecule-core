# Screencast Storyboard — Snapshot Secret Scrubber
**PR:** #977 | **Feature:** `workspace/lib/snapshot_scrub.py`
**Duration:** 60 seconds | **Format:** Terminal-led + browser overlay, dark theme

---

## Pre-roll (0:00–0:04)

**Terminal — dark theme**
Prompt: `agent@pm-workspace:~$`

Narration (0:00–0:04):
> "Every agent workspace can hibernate — preserving its memory state to disk. But what if that snapshot contains secrets? That's where the scrubber comes in."

**Camera:** Static terminal frame. 3-second hold. No cursor.

---

## Moment 1 — Before: raw memory snapshot with secrets (0:04–0:18)

**Terminal:**
```bash
# Simulate a raw memory entry before scrubbing
python3 - << 'EOF'
from snapshot_scrub import scrub_snapshot

raw_snapshot = {
    "workspace_id": "ws-pm-001",
    "memories": [
        {
            "key": "api_config",
            "content": "ANTHROPIC_API_KEY=sk-ant-abcd1234wxyz5678",
            "updated_at": "2026-04-20T10:00:00Z"
        },
        {
            "key": "user_context",
            "content": "User asked about enterprise pricing.",
            "updated_at": "2026-04-20T10:01:00Z"
        },
        {
            "key": "sandbox_output",
            "content": "[sandbox_output] Running: pip install requests\nOutput: success",
            "updated_at": "2026-04-20T10:02:00Z"
        }
    ]
}

print(scrub_snapshot(raw_snapshot))
EOF
```

**Terminal output (raw, BEFORE scrub):**
```json
{
  "workspace_id": "ws-pm-001",
  "memories": [
    {"key": "api_config", "content": "ANTHROPIC_API_KEY=sk-ant-abcd1234wxyz5678"},
    {"key": "user_context", "content": "User asked about enterprise pricing."},
    {"key": "sandbox_output", "content": "[sandbox_output] Running: pip install..."}
  ]
}
```

**Camera:** Highlight the raw ANTHROPIC_API_KEY and sandbox output lines — red underline. Hold 2s.

Narration (0:06–0:16):
> "A raw snapshot before scrubbing. The agent stored an API key in memory. It also ran code — and the sandbox output is in there too. Both are about to go to disk when this workspace hibernates."

**Callout text (bottom-left):**
`Before scrubbing: API keys, Bearer tokens, sandbox output — all on disk.`

---

## Moment 2 — Scrubber runs (0:18–0:32)

**Terminal — same session:**
The python script runs.

**Terminal output (AFTER scrub):**
```json
{
  "workspace_id": "ws-pm-001",
  "memories": [
    {
      "key": "api_config",
      "content": "[REDACTED:API_KEY]"
    },
    {
      "key": "user_context",
      "content": "User asked about enterprise pricing."
    }
  ]
}
```

**Camera:** The output appears line by line. Watch:
1. `"api_config"` entry — content replaced with `[REDACTED:API_KEY]`
2. `"sandbox_output"` entry — **absent entirely** (excluded, not scrubbed)
3. `"user_context"` — passes through unchanged

Green checkmark on the `user_context` line.

Narration (0:20–0:28):
> "The scrubber runs — before the snapshot reaches disk. API keys become `[REDACTED:API_KEY]`. Sandbox output is excluded entirely — it's not scrubbed, it's dropped. The agent's actual knowledge passes through unchanged."

**Callout text:**
`API key → [REDACTED:API_KEY]. Sandbox output → excluded entirely. Everything else → passes through.`

---

## Moment 3 — Pattern coverage (0:32–0:44)

**Terminal:**
```bash
python3 - << 'EOF'
from snapshot_scrub import scrub_content

test_cases = [
    ("OPENAI_API_KEY=sk-proj-123456abcdef",      "env-var"),
    ("Bearer eyJhbGciOiJIUzI1NiJ9",              "Bearer token"),
    ("sk-ant-abcd1234wxyz5678",                   "Anthropic key"),
    ("ghp_abc123def456ghi789jkl012mno",           "GitHub PAT"),
    ("AKIAIOSFODNN7EXAMPLE",                      "AWS key"),
    ("YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnp4eXpBQ0N",  "high-entropy base64"),
    ("Everything looks fine",                      "clean content"),
]

for text, label in test_cases:
    result = scrub_content(text)
    print(f"{label:20s} → {result}")
EOF
```

**Terminal output:**
```
env-var            → [REDACTED:API_KEY]
Bearer token       → [REDACTED:BEARER_TOKEN]
Anthropic key      → [REDACTED:SK_TOKEN]
GitHub PAT         → [REDACTED:GITHUB_PAT]
AWS key            → [REDACTED:AWS_ACCESS_KEY]
high-entropy base64 → [REDACTED:BASE64_BLOB]
clean content       → Everything looks fine
```

**Camera:** Scroll through all 7 patterns. Hold 2s on the clean content line — no redaction.

Narration (0:34–0:42):
> "The scrubber catches seven secret patterns — API keys, Bearer tokens, GitHub PATs, AWS keys, Cloudflare tokens, high-entropy blobs. Clean content passes through unaltered."

---

## Moment 4 — Real-world scenario (0:44–0:54)

**Cut to:** Browser — Molecule AI canvas. Workspace `pm-agent` shows `[HIBERNATING]`.

**Terminal:**
```bash
# Workspace hibernating — scrubber runs automatically
curl -s -X POST "$PLATFORM/workspaces/ws-pm-001/hibernate" \
  -H "Authorization: Bearer $AGENT_TOKEN"
```

**Terminal output:**
```
{"status": "hibernating", "snapshot_id": "snap-xyz-789", "scrubbed": true}
```

**Camera:** Focus on `"scrubbed": true`. Green highlight ring `#22C55E`. Hold 1.5s.

Narration (0:46–0:52):
> "When the workspace hibernates, the scrubber runs automatically — before the snapshot touches disk. The result is marked `scrubbed: true`. Admins can trust that snapshots are safe."

---

## Close (0:54–1:00)

**Terminal clean frame.** Cursor at prompt.

Narration (0:54–0:58):
> "Snapshot secret scrubber — API keys, Bearer tokens, sandbox output, all handled before hibernate. Molecule AI writes only what should be written."

**End card:**
```
Snapshot Secret Scrubber
workspace/lib/snapshot_scrub.py — molecule-core#977
```
**Fade to black.**

---

## Production Spec

| Spec | Value |
|------|-------|
| Terminal theme | Dark, SF Mono 14pt / JetBrains Mono 13pt |
| Camera | Screenflow / Camtasia, 1440×900 → 1080p export |
| JSON output | `jq --monochrome-output` |
| Callout highlight | Amber ring `#E8A000`, 1s fade-in/out |
| Red alert | Red underline `#EF4444` on raw secret lines in Moment 1 |
| Green success | Green ring `#22C55E` on `"scrubbed": true` in Moment 4 |
| VO voice | en-US-AriaNeural (consistent across all 4 storyboards) |
| Music | None |
| Playback speed | Moments 1–3 at 2x for terminal typing effect |
| Type-in animation | Realistic cursor blink |
