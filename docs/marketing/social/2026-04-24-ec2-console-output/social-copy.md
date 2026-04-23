# EC2 Console Output — Social Copy
Campaign: EC2 Console Output | Source: PR #1178
Publish day: 2026-04-24 (Day 4)
Status: ✅ APPROVED — Marketing Lead 2026-04-22 (PM confirmed)
Assets: `ec2-console-output-canvas.png` (1200×800, dark mode)

---

## X (Twitter) — Primary thread (4 posts)

### Post 1 — Hook
Your workspace failed.
You already know that.
What you don't know is *why* — and right now that means switching to the AWS Console, finding the instance, pulling the console output, and switching back.

That's about to get better.

---

### Post 2 — The old workflow
Before this fix:
Click failed workspace → tab switch → AWS Console → log in → find instance → Actions → Get system log.

You're in the right place. You have the output. But you're also outside Canvas — you've lost the context of what the agent was doing, which workspace it was, and what the last_sample_error said.

Still doable. Still a minute of your time. Still a context switch.

---

### Post 3 — The new workflow
After PR #1178:
Click failed workspace → EC2 Console tab → full instance boot log, colorized by level, directly in Canvas.

Same output as AWS Console. Same detail. No tab switch. No context loss.

Thirty seconds to root cause, if that.

---

### Post 4 — CTA
EC2 Console Output is now in Canvas — no AWS Console required.

Works for any workspace: local Docker, remote EC2, on-prem VM.
If Molecule AI manages the instance, the console log is one click away.

→ [See how it works](https://docs.molecule.ai/docs/guides/remote-workspaces)

---

## LinkedIn — Single post

**Title:** The fastest way to debug a failed AI agent workspace

When an AI agent workspace fails in production, the debugging question is always the same: what happened on the instance?

Before this week, the answer required leaving the canvas. Log into AWS. Find the instance. Pull the system log. Cross-reference with the workspace ID. Piece together what the agent was doing.

That workflow just changed.

Molecule AI now surfaces EC2 Console Output directly in the Canvas workspace detail panel. Full instance boot log, colorized by log level — INFO, WARN, ERROR — without leaving your workflow.

The practical difference: root cause in thirty seconds instead of three minutes. No tab switch. No losing the workspace context you were already looking at.

Works for any workspace Molecule AI manages: local Docker, remote EC2, on-prem VM. The console output is there when you need it.

EC2 Console Output ships with Phase 30.

→ [Read the docs](https://docs.molecule.ai/docs/guides/remote-workspaces)
→ [Molecule AI on GitHub](https://github.com/Molecule-AI/molecule-core)

#AIagents #DevOps #AWs #CloudComputing #MoleculeAI

---

## Campaign notes

**Audience:** Platform engineers, DevOps, MLOps (X + LinkedIn)
**Tone:** Operational. Concrete. Shows the workflow, not the feature announcement.
**Differentiation:** EC2 Console Output in Canvas is a canvas/workspace UX differentiator — directly in the operator's workflow, not in a separate AWS tab.
**CTA:** /docs/guides/remote-workspaces — ties back to Phase 30 Remote Workspaces
**Coordinate with:** Day 4 of Phase 30 social campaign. Post after Discord Adapter (Day 2) and Org API Keys (Day 3).

*Draft by Marketing Lead 2026-04-21 — based on PR #1178 + EC2 Console demo storyboard*
