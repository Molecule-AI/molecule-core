# Phase 30 Launch — Hacker News Submission Guide

> **For:** DevRel / whoever submits | **Status:** Draft — submit when ready
> **Trigger:** After blog post is live on docs site

---

## Why HN?

Hacker News has a large developer and technical audience that overlaps with Molecule AI's target users: platform engineers, indie developers building with AI, and technical evaluators. A well-crafted HN post can drive significant docs traffic and signups.

---

## What to Submit

**URL:** The Phase 30 launch blog post at `https://moleculesai.app/blog/remote-workspaces-ga`

**Title options:**

| Option | Title | When to use |
|---|---|---|
| A | Show HN — Phase 30: run AI agents on your laptop, your cloud, anywhere | Standard launch |
| B | Show HN — Molecule AI launches Remote Workspaces (GA) | If the "Show HN" prefix is too meta |
| C | Show HN — We built a fleet management layer for AI agents | Developer-heavy audience, less marketing |

**Recommended:** Option A — HN readers respond well to technical products with a clear "what it does" title.

---

## What to Write in the HN Post Body

The blog post is the destination. The HN post body is a 2–3 paragraph pitch that earns the click. Write it yourself — don't paste the full blog post.

**Template:**

```
We just shipped Phase 30 — Remote Workspaces is now GA.

Most AI agent platforms assume all agents run inside the platform's infrastructure. Phase 30 lets agents run anywhere: your laptop, a VM in your own cloud account, an on-prem server. They register to the same org, appear in the same Canvas, and communicate via the same A2A protocol.

The governance is the same. The auth contract is the same. The only visible difference is a REMOTE badge on the workspace card.

Quickstart is under 5 minutes:
  pip install molecule-sdk
  python3 run.py --runtime remote

Docs, demo, and quickstart guide in the link.

(I'm [NAME] from the Molecule AI team — AMA.)
```

**Key HN-specific rules:**
- Don't use "I" too many times — but the "(I'm ... AMA)" close is expected and encouraged
- Don't hard-sell or use marketing language — just describe the product
- Be specific about what it does ("A2A protocol", "workspace auth tokens") — that signals technical depth
- Keep it short — 2–3 paragraphs, not an essay

---

## When to Submit

**Timing matters:**

- Submit when HN traffic is high but not oversaturated
- **Best window:** Tuesday–Thursday, 10:00–13:00 UTC (roughly when US East Coast is morning and Europe is mid-day)
- **Avoid:** Mondays (low traffic), Fridays (weekend readers don't upvote), major news events
- **Recommended day:** Wednesday of launch week, 11:00 UTC

---

## What Happens After Submitting

1. **Monitor for 2–4 hours** after submission — respond to comments, answer technical questions
2. **Don't be defensive** if criticism comes — acknowledge legitimate issues, don't argue
3. **Upvote your own post once** — this is normal and expected on HN
4. **If it hits the front page:** brace for volume — keep at least one team member monitoring

---

## Comment Templates for Common Questions

**"How is this different from Modal / Railway?"**
> Modal and Railway run your code on their infrastructure. Molecule AI Remote Workspaces run on yours — you own the compute, the data stays on your machine. We're an orchestration layer, not an inference platform.

**"How is this different from Cursor / Copilot?"**
> Cursor and Copilot are individual developer tools — one human, one AI. Molecule AI is an agent orchestration platform — multiple autonomous agents coordinating with each other. Remote Workspaces are about running *agents* that collaborate, not just one developer and one AI pairing.

**"Why would I want agents on my laptop?"**
> Local iteration + debugging with your IDE, while the agent still participates in your org's task pipeline. Also useful for data-residency requirements — agent compute on your infrastructure while orchestration stays on the platform.

**"Is this production-ready?"**
> Yes — Phase 30 is generally available. Remote Workspaces are in the same GA release as container workspaces.

---

## Alternate: "Ask HN"

If the team prefers an "Ask HN" format (more engagement, more questions):

**Title:** Ask HN — What would you build with a remote AI agent that runs on your own infrastructure?

**Body:** Short framing paragraph + question. This format tends to get high comment volume. Risk: less control over the narrative.

**Recommended format for launch:** Standard URL submission. More traffic, cleaner signal.

---

*Replace [NAME] with actual submitter name before posting.*
