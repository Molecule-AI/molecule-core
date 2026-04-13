---
name: Write Scripts to Files Before Running
description: Large inline node -e commands can trigger timeouts — always write to temp file first
type: feedback
---

When running large Node.js scripts, always write the script to a temp file first, then run with `node <filepath>`. Clean up the file afterward.

**Why:** OpenClaw's gateway flagged long `-e` strings as obfuscation and timed out even after approval. This also applies in Claude Code — long inline scripts are harder to debug and review.

**How to apply:** Any script longer than ~5 lines should be written to a temp file. Use `/tmp/` or the workspace for the file. Delete after use.
