# Phase 30 Launch — Post-Push Execution Checklist

> **For:** DevRel + Marketing Lead | **Trigger:** After GH_TOKEN refreshes + push completes
> **Purpose:** Step-by-step sequencing so nothing gets missed on launch day

---

## Phase 1 — Push & Validation (Do First)

### 1.1 Push the branch

```bash
git -C /workspace/repo push origin content/blog/memory-backup-restore
```

### 1.2 Verify all 11 commits landed

```bash
gh api repos/Molecule-AI/internal/commits --jq '.[0:11] | .[].commit.message' \
  --param per_page=15 2>&1 | head -30
```

Look for the expected commit messages in reverse chronological order.

### 1.3 Post GitHub issue comments

```bash
bash /workspace/repo/marketing/demos/post-issue-comments.sh
```

This posts completion comments on `#1172` and `#1173` using the staged JSON payloads.

### 1.4 Verify comments posted

```bash
gh issue comment list 1172 --repo Molecule-AI/internal 2>&1
gh issue comment list 1173 --repo Molecule-AI/internal 2>&1
```

Confirm both return the DevRel completion text.

---

## Phase 2 — Docs Site Publish

### 2.1 Submit PR from the branch

```bash
gh pr create \
  --repo Molecule-AI/internal \
  --base main \
  --head content/blog/memory-backup-restore \
  --title "docs(marketing): Phase 30 launch — Remote Workspaces GA, demos, and supporting content" \
  --body "$(cat <<'EOF'
## Summary
- Phase 30 Remote Workspaces GA blog post
- Phase 30 user guide and FAQ
- /cp/* same-origin proxy guide
- Chrome DevTools MCP governance blog post
- Container vs Remote decision guide
- Secure by Design blog post (beta auth launch)
- AGENTS.md auto-generation working demo + screencast spec (#1172)
- Cloudflare Artifacts working demo + screencast spec (#1173)
- Phase 30 social copy (X: 4 versions, LinkedIn)
- Chrome DevTools MCP social copy
- Phase 30 video production package (for Video Editor)
- Phase 30 DevRel asset inventory
- Fleet diagram, TTS audio files, VO scripts

## Test plan
- [ ] Review each guide for technical accuracy before merge
- [ ] Confirm all internal links resolve
- [ ] Confirm blog post dates are correct (2026-04-20)
- [ ] Verify TTS audio files play (mp3)
- [ ] Run docs link audit (all 34 links verified on disk)

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

### 2.2 Get PR reviewed and merged

Hand off to whoever can approve — Marketing Lead or a tech lead.

### 2.3 After merge: verify docs site publishes

```bash
curl -s https://moleculesai.app/docs/guides/remote-workspaces.md | head -20
curl -s https://moleculesai.app/docs/guides/remote-workspaces-faq.md | head -10
```

Confirm both return 200 with correct frontmatter.

---

## Phase 3 — Social Posts (After PR Merges)

### 3.1 X (Twitter) — Phase 30 launch

Post all 4 versions from `marketing/devrel/phase30-social-copy.md`, spaced ~3 hours apart:

| # | Version | Angle | Post time |
|---|---|---|---|
| 1 | Version A | Technical | Launch day, 09:00 UTC |
| 2 | Version B | Product | Launch day, 12:00 UTC |
| 3 | Version C | Developer | Launch day, 15:00 UTC |
| 4 | Version D | Enterprise | Launch day, 18:00 UTC |

**Images:** Attach `marketing/assets/phase30-fleet-diagram.png` to Version A and D. For C, use a terminal screenshot.

### 3.2 LinkedIn — Phase 30 launch

Post the enterprise/platform post from `phase30-social-copy.md`. Attach fleet diagram.

### 3.3 X — Chrome DevTools MCP

Post Version A from `marketing/devrel/chrome-devtools-mcp-social-copy.md`. Attach fleet diagram.

### 3.4 LinkedIn — Chrome DevTools MCP

Post the full LinkedIn block from `chrome-devtools-mcp-social-copy.md`. Attach checklist graphic or quote card.

### 3.5 Schedule cadence

Use Buffer/Hootsuite or schedule manually. All copy is pre-written — no drafting needed at post time.

---

## Phase 4 — Email Campaign

After social posts are live, trigger the email drip sequence (see `marketing/drip/phase30-email-drip.md`).

### 3-step sequence:
1. **Day 1 (launch morning):** Announcement — "Phase 30 is GA" + blog link + quickstart guide
2. **Day 3–4:** Feature deep dive — pick the strongest sub-feature (AGENTS.md or CF Artifacts)
3. **Day 7:** Social proof / case study or customer quote (coordinate with Sales)

---

## Phase 5 — Community & Devrel

### 5.1 Hacker News

See `marketing/community/hacker-news-launch.md` — submit when ready, monitor comments for 4–6 hours.

### 5.2 Discord / Slack announcements

Post in relevant channels. Copy is in `marketing/community/community-announcements.md`.

### 5.3 DevRel outreach

If any开发者 advocates or agent ecosystem influencers should know about Phase 30, pre-write outreach DMs now (coordinate with Marketing Lead).

---

## Phase 6 — Verify Live Assets (Day 2+)

```bash
# Blog posts
curl -s -o /dev/null -w "%{http_code}" https://moleculesai.app/blog/remote-workspaces-ga
curl -s -o /dev/null -w "%{http_code}" https://moleculesai.app/blog/chrome-devtools-mcp-governance

# Guides
curl -s -o /dev/null -w "%{http_code}" https://moleculesai.app/docs/guides/remote-workspaces
curl -s -o /dev/null -w "%{http_code}" https://moleculesai.app/docs/guides/remote-workspaces-faq

# Audio (if hosted)
curl -s -o /dev/null -w "%{http_code}" https://moleculesai.app/audio/phase30-announce.mp3
```

All should return 200.

---

## Known Blockers to Communicate

| Blocker | Owner | Status |
|---|---|---|
| GH_TOKEN must refresh before push | CEO | ⏳ Waiting |
| PR must be reviewed and merged before docs go live | Marketing Lead / Tech Lead | ⏳ Waiting |
| Canvas screenshot (REMOTE badge) not yet captured | Design Team | ⏳ Waiting |
| PMM path for `phase30-launch-plan.md` unconfirmed | PMM | ⏳ Waiting |

---

*Update this doc as items complete. Check off each step after execution.*
