# Featured Skills Showcase — HERMES v0.10.0 Counter-Demo
**Issue:** #1415 | **Owner:** DevRel Engineer
**Purpose:** Make Molecule AI's skills architecture tangible for sellers and evaluators. Not "you can install skills" — "here's what 5 minutes of skill installation gets you."
**Format:** Interactive demo + README walkthrough (~5 min live, or self-guided ~10 min)

---

## What This Showcase Demonstrates

A single Molecule AI workspace with 3 agent personas, each with a different skill stack:

| Agent | Skills Installed | What It Does |
|---|---|---|
| `data-agent` | `mcp-filesystem`, `mcp-postgres` | Reads workspace DB, writes query results to filesystem |
| `content-agent` | `mcp-github`, `@molecule/ai/tts`, `mcp-slack` | Summarizes a GitHub PR, converts to audio, posts to Slack |
| `monitoring-agent` | `mcp-aws`, `mcp-cloudflare` | Reads AWS cost report + CF analytics, posts combined dashboard |

This proves: **different agents, different tool stacks, same platform.**

---

## Skills Used

| Skill | Source | Purpose in Demo |
|---|---|---|
| `mcp-filesystem` | MCP registry | Write/read files |
| `mcp-postgres` | MCP registry | Query DB |
| `mcp-github` | MCP registry | Read PR metadata |
| `@molecule/ai/tts` | Molecule AI skills | Convert text to speech |
| `mcp-slack` | MCP registry | Post to Slack channel |
| `mcp-aws` | MCP registry | Read cost explorer |
| `mcp-cloudflare` | MCP registry | Read analytics API |

---

## Demo Flow

### Step 1 — Install skills (30 seconds)
```bash
molecule skills install mcp-filesystem mcp-postgres
molecule skills install mcp-github @molecule/ai/tts mcp-slack
molecule skills install mcp-aws mcp-cloudflare
```

### Step 2 — Verify installations (15 seconds)
```bash
molecule skills list
# Shows 7 skills, each with version, MCP server, status
```

### Step 3 — Run data-agent (60 seconds)
```
Prompt: "Query the production database for the top 10 users by API calls this week. Save results to /reports/weekly-users.csv."
```
- Agent loads `mcp-postgres` + `mcp-filesystem`
- Runs query, formats CSV, writes to workspace filesystem
- Seller notes: "That's the same workspace as our other agents — different skills"

### Step 4 — Run content-agent (90 seconds)
```
Prompt: "Summarize this PR: molecule-ai/molecule-core/pull/1439. Then convert the summary to a 30-second audio clip and post it to #ai-updates."
```
- Agent loads `mcp-github` → reads PR summary
- Loads `@molecule/ai/tts` → converts to audio
- Loads `mcp-slack` → posts to channel
- Seller notes: "Three different skills, three different API integrations, one agent"

### Step 5 — Run monitoring-agent (60 seconds)
```
Prompt: "Show me this week's AWS spend and Cloudflare analytics. Write a one-paragraph summary."
```
- Agent loads `mcp-aws` + `mcp-cloudflare`
- Fetches both, synthesizes into a paragraph
- Seller notes: "Cross-cloud, cross-API — that's the fleet visibility story"

---

## Key Talking Points (for sellers)

1. **Installation is one command** — `molecule skills install <name>` — no SDK work, no code
2. **Skills are per-agent** — same workspace, different tool stack per persona
3. **MCP-compatible** — any MCP server works, including custom ones
4. **No locked pricing** — pay per-call to the tool providers you choose, not a bundled Portal fee
5. **Fleet-wide visibility** — Canvas shows which skills each agent has loaded

---

## What Sellers Can Say When They Hear "Bundled Tools"

> "Bundled tools work great for one agent. When you have five teams running twenty agents, you need different tools for different problems. Molecule AI's skills architecture means every agent has exactly the tools it needs — and nothing it doesn't. Here's what that looks like in five minutes."

---

## README Structure

```
docs/devrel/skills-showcase/
├── README.md          ← main walkthrough (this file)
├── demo-notes.md      ← seller talking points + objection handlers
└── screenshots/      ← skill install output, Canvas skill panel
```

---

## Brand Audio Note

Generate a 15-second TTS clip using `marketing/audio/phase30-announce.mp3` cadence reference: "Skills architecture. Install what you need."
Use `@molecule/ai/tts` skill output as the audio asset.
Brand theme: dark zinc (#0f011), blue-500 (#3b82f6) for highlights.

---

## TTS/Multimedia Directive

**Audio asset:** 15-second brand TTS clip — "Skills architecture. Install what you need. Molecule AI Phase 30."
- Use `marketing/audio/phase30-announce.mp3` cadence as reference
- Output: `docs/devrel/skills-showcase/audio/skills-intro.mp3`
- Include as optional audio in the skills showcase README

---

*Issue #1415 — DevRel Engineer owns demo script + README + screenshots. Marketing Lead reviews for messaging accuracy. Brand audio directive included per CEO directive.*
