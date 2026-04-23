---
title: "How Molecule AI Tracks Active Agent Tasks: Concurrency Control in Production"
date: 2026-04-21
slug: active-tasks-concurrency
description: "Running a fleet of AI agents in production means running many tasks concurrently. Here's how Molecule AI's active_tasks counter tracks concurrency, prevents overscheduling, and keeps your agent fleet from overwhelming your infrastructure."
tags: [architecture, runtime, concurrency, production, platform-engineering]
author: Molecule AI
og_title: "How Molecule AI Tracks Active Agent Tasks"
og_description: "The active_tasks counter is the mechanism Molecule AI uses to track how many agents are running concurrently — preventing overscheduling, enabling graceful concurrency limits, and giving your platform team real-time fleet visibility."
twitter_card: summary_large_image
---

<script type="application/ld+json">
{
  "@context": "https://schema.org",
  "@type": "Article",
  "headline": "How Molecule AI Tracks Active Agent Tasks: Concurrency Control in Production",
  "datePublished": "2026-04-21",
  "dateModified": "2026-04-22",
  "author": {
    "@type": "Organization",
    "name": "Molecule AI"
  },
  "publisher": {
    "@type": "Organization",
    "name": "Molecule AI",
    "logo": {
      "@type": "ImageObject",
      "url": "https://molecule.ai/logo.png"
    }
  },
  "description": "Running a fleet of AI agents in production means running many tasks concurrently. Here's how Molecule AI's active_tasks counter tracks concurrency, prevents overscheduling, and keeps your agent fleet from overwhelming your infrastructure.",
  "keywords": "Running a fleet of AI agents in production means running many tasks concurrently. Here's how Molecul",
  "url": "https://molecule.ai/blog/active-tasks-concurrency"
}
</script>
author: Molecule AI
og_title: "How Molecule AI Tracks Active Agent Tasks: Concurrency Control in Production"
og_description: "Running a fleet of AI agents in production means running many tasks concurrently. Here's how Molecule AI's active_tasks counter tracks concurrency, prevents overscheduling, and keeps your agent fleet from overwhelming your infrastructure."
og_image: /assets/blog/2026-04-21-active-tasks-og.png
twitter_card: summary_large_image
canonical: https://molecule.ai/blog/active-tasks-concurrency
keywords:
  - "AI agent fleet management"
  - "multi-tenant AI agents"
  - "AI agent concurrency control"
  - "AI agent deployment platform"

# How Molecule AI Tracks Active Agent Tasks: Concurrency Control in Production

Every production agent platform eventually faces the same problem: too many agents running at once.

Your infrastructure has limits — CPU, memory, GPU slots, API rate limits. Your agents have variable resource footprints. Without a mechanism to track how many agents are running at any given time, you can't enforce concurrency limits, detect overscheduling, or give your platform team accurate fleet visibility.

Molecule AI's `active_tasks` counter is the answer.

---

## What the active_tasks Counter Tracks

The `active_tasks` counter tracks how many agent tasks are currently running across your deployment. It increments when a task starts and decrements when a task completes — whether it finishes cleanly, errors out, or is cancelled.

This makes it a real-time view of fleet load, not just a scheduled queue depth.

```go
// Simplified model
func (s *Scheduler) IncrementActiveTasks() {
    atomic.AddInt32(&s.activeTasks, 1)
}

func (s *Scheduler) DecrementActiveTasks() {
    atomic.AddInt32(&s.activeTasks, 1)
}

func (s *Scheduler) GetActiveTasks() int32 {
    return atomic.LoadInt32(&s.activeTasks)
}
```

The counter uses atomic operations so it's safe across concurrent goroutines — your scheduler can increment on task start and decrement on task completion without locks.

---

## What You Can Build With It

**Concurrency limits**

Set `max_concurrent` on your deployment. When `active_tasks >= max_concurrent`, the scheduler holds new task submissions until a running task completes. No overscheduling, no resource exhaustion.

```bash
# Example: limit a workspace to 10 concurrent agents
molecule workspace update ws-01 --max-concurrent 10
```

**Overscheduling detection**

If `active_tasks` is consistently at your limit and you're seeing queued tasks pile up, that's a signal to scale — more workers, larger instance types, or a sharded deployment.

**Fleet visibility**

The Canvas fleet view shows `active_tasks` per workspace, per org. Your platform team sees real load, not just scheduled load.

**Incident alerting**

Alert when `active_tasks` drops unexpectedly (agents crashing) or spikes to your limit (scheduling backlog forming).

---

## How It Fits Into the Scheduler

The scheduler checks `active_tasks` before accepting a new task:

```go
func (s *Scheduler) Submit(task *Task) error {
    if atomic.LoadInt32(&s.activeTasks) >= s.maxConcurrent {
        return ErrConcurrencyLimit
    }
    atomic.AddInt32(&s.activeTasks, 1)
    s.schedule(task)
    return nil
}
```

When a task completes, `DecrementActiveTasks()` fires and the queue advances. The scheduler always knows the true concurrency level — not a guess based on queue depth.

---

## Why It Matters for Agent Fleets

Traditional job queues track queue depth. `active_tasks` tracks actual execution. The difference matters when:
- Tasks have variable runtimes (30 seconds vs 30 minutes)
- Agents consume different resources per task
- Your infrastructure has burst vs sustained capacity limits

Queue depth alone can't tell you whether you're at capacity. `active_tasks` can.

---

## Get Started

Set a `max_concurrent` limit on your workspace and watch the fleet view. If you're hitting limits consistently, that's a capacity planning signal — not a platform failure.

→ [Scheduler Architecture Documentation](/docs/architecture/scheduler) | → [Canvas Fleet View Documentation](/docs/guides/canvas) | → [Phase 30 Launch Blog](/docs/blog/remote-workspaces-ga)

---

*active_tasks concurrency shipped in [PR #1413](https://github.com/Molecule-AI/molecule-core/pull/1413) as part of Molecule AI Phase 30.*
