# PR #1870 — A2A Queue-on-Busy: Priority Queue Phase 1
**Date:** 2026-04-24 | **Owner:** PMM | **PR:** #1870 (merged 2026-04-23)
**Feature:** `feat(a2a): queue-on-busy — Phase 1 of priority queue`
**Brief status:** DRAFT — needs PM confirmation of positioning

---

## Problem

When a Molecule AI agent is busy processing a long-running task, inbound A2A messages have nowhere to go. Without a queue, messages arriving during a busy period are dropped, returned as errors, or cause task conflicts. For production multi-agent workflows, this means unreliable task routing — especially when a supervisor agent delegates to a busy subordinate.

## Solution

Phase 1 implements queue-on-busy behavior for A2A task routing. When an agent is busy, incoming tasks are queued rather than rejected or dropped. The queue is priority-ordered, so higher-priority tasks are processed first when the agent becomes available.

**Phase 1 scope (PR #1870):**
- `ON CONFLICT` syntax fix in a2a-queue handler (`#1893`)
- Queue structure: messages queued when agent reports `busy` status
- Priority ordering: queue is priority-sorted (Phase 1 = foundation)

**Phase 2+ (not yet merged):** Full priority levels, queue management UI, TTL/expiry.

---

## Three Claims (confirm with PM before publishing)

1. **No dropped tasks.** When an agent is busy, inbound tasks are queued rather than rejected — nothing falls through.
2. **Priority ordering.** High-priority tasks ahead in the queue are delivered first when the agent frees up.
3. **Production reliability.** A2A task routing becomes reliable under concurrent load, not best-effort.

---

## Target Developer

Platform engineers running multi-agent supervisor/subordinate workflows. Anyone with two or more agents that delegate to each other and need reliable task delivery under load.

---

## CTA

"Deploy multi-agent workflows with confidence — A2A queues ensure no task is lost when an agent is busy."

---

## Language to Avoid

- "Guaranteed delivery" — Phase 1 queues to memory/disk but Phase 2+ has TTL
- "Full priority queue" — Phase 1 is the foundation, not the complete implementation
- "queue management UI" — not in Phase 1

---

## Do We Need a Standalone Campaign?

**No.** A2A Queue Phase 1 is a reliability underpinning, not a headline feature. Recommend: brief as a supporting proof point in the observability/governance narrative (Phase 34 Tool Trace + Platform Instructions story), or as a footnote in the A2A enterprise deep-dive.

**Decision for Marketing Lead:** File this brief. Do not create standalone social copy unless Phase 2 ships with a full priority management UI.

---

*PMM brief 2026-04-24 — PR #1892 merged 2026-04-23, no prior brief found. Needs PM confirmation of claims before external use.*
