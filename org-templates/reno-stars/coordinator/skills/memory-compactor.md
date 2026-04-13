You are performing memory maintenance for the Claude Code persistent memory system.

## Config
Read /Users/renostars/reno-star-business-intelligent/config/env.json for paths.

## Memory Locations
- Memory (source of truth): /Users/renostars/reno-star-business-intelligent/memory/
- Memory index: /Users/renostars/reno-star-business-intelligent/memory/MEMORY.md
- OpenClaw legacy memory: /Users/renostars/.openclaw/workspace/memory/

## STEPS
1. Read the global memory index (MEMORY.md)
2. Read each memory file referenced in the index
3. Check for:
   - Outdated information that needs updating
   - Duplicate or conflicting memories
   - Missing context from recent work (check git logs of active projects)
4. If any memory needs updating, update it
5. If new significant facts were discovered, create new memory files and add to index
6. Keep MEMORY.md under 200 lines

## What to Capture
- Durable facts, decisions, user preferences
- Project status changes
- New tools, services, or infrastructure
- Lessons learned from errors

## What NOT to Capture
- Transient task details
- Raw secrets or tokens
- Anything already in CLAUDE.md
- Code patterns derivable from reading the code

## Log
Append one JSON line to /Users/renostars/reno-star-business-intelligent/data/cron-logs/memory-compactor.jsonl:
{"ts": "<ISO>", "job": "memory-compactor", "status": "success"|"error", "summary": "<what changed>", "error": null}
