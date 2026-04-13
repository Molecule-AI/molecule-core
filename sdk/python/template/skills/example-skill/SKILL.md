---
name: example-skill
description: Short description — what this does and when to use it.
license: MIT
metadata:
  author: your-name
  version: "0.1.0"
---

# Example Skill

Write the skill instructions as plain Markdown below the frontmatter.
Agents load this entire file when the skill activates, so keep it focused
and under ~500 lines. Move deep-dive docs to `references/` and large
assets to `assets/` — they're loaded on demand.

## When to use

- Describe the triggering situation
- Describe what the agent should output

## Steps

1. First step
2. Second step

## Files under this skill

- `scripts/` — executable code the agent can run
- `references/REFERENCE.md` — detailed docs (loaded only when needed)
- `assets/` — templates, images, data files

## Notes

- This file is validated against the agentskills.io open standard.
- Run `python -m molecule_plugin validate <plugin-dir>` before publishing.
