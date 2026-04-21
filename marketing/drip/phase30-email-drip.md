# Phase 30 — Email Drip Sequence

> **For:** Marketing Lead + Email/CRM team | **Status:** Draft
> **Trigger:** After blog post and guides are live

3-email sequence designed to move developers and platform evaluators from "heard of it" to "tried it."

---

## Email 1 — Launch Announcement (Day 1, Launch Morning)

**Subject line:** Your laptop is now a valid Molecule AI runtime
**Subject line alt:** Phase 30: Remote Workspaces is generally available

**Preview text:** Agents can now run on any machine — your laptop, a VM, on-prem — and appear in Canvas like every other workspace.

**Body:**

> Hi {{first_name}},
>
> Most AI agent platforms assume your agents run in the same place as the control plane.
>
> Phase 30 changes that.
>
> Starting today, any Python agent — running on your laptop, a cloud instance, or an on-premises server — can register with your Molecule AI org and appear in Canvas as a first-class workspace. Same auth. Same A2A protocol. Same audit trail.
>
> **The only visible difference: a purple REMOTE badge.**
>
> We call it Remote Workspaces. Here's why it matters:
>
> - **Developers** — run an agent on your laptop, debug it with your IDE, and have it participate in your org's task pipeline simultaneously
> - **Platform teams** — deploy agents in your own cloud account without changing your Molecule AI workflow
> - **Enterprise** — meet data-residency requirements by keeping agent compute on your infrastructure
>
> Phase 30 is generally available today. Self-serve setup in under 5 minutes.
>
> [Get started →](/docs/guides/remote-workspaces)
> [Read the launch post →](/blog/remote-workspaces-ga)
> [Quickstart guide →](/docs/guides/remote-workspaces#quick-start)
>
> — The Molecule AI team

---

## Email 2 — Feature Deep Dive (Day 3–4)

**Subject line:** The AGENTS.md trick that makes multi-agent coordination just work
**Subject line alt:** Two things that make Remote Workspaces different

**Preview text:** Auto-generated agent manifests and versioned workspace snapshots ship with Phase 30.

**Body:**

> Hi {{first_name}},
>
> A quick follow-up on Phase 30. Two things that shipped with Remote Workspaces that deserve their own explanation:
>
> **1. AGENTS.md auto-generation**
>
> Every Molecule AI workspace now generates an `AGENTS.md` file at boot — automatically. It reflects the workspace config: role, A2A endpoint, available tools. Any peer agent can read it to understand what another agent does and how to reach it, without reading system prompts.
>
> This is the AAIF / Linux Foundation AGENTS.md standard, implemented as a first-class platform feature.
>
> **2. Versioned workspace state with Cloudflare Artifacts**
>
> Every workspace can now be linked to a Cloudflare Artifacts git repo. The agent can push snapshots — current task state, memory dumps, config — and other agents can fork the repo to continue from the same point.
>
> Git for agents, built into the platform. No separate dashboard, no external git service setup.
>
> [See the working demos →](/marketing/demos) *(after docs go live, update to public URL)*
> [Phase 30 launch post →](/blog/remote-workspaces-ga)
>
> Questions? Reply to this email — we read them.
>
> — The Molecule AI team

---

## Email 3 — Social Proof / CTA (Day 7)

**Subject line:** What developers are building with Remote Workspaces
**Subject line alt:** One week in: what the community is doing with Phase 30

**Preview text:** Data residency, multi-cloud fleets, and local debugging — the first week of Phase 30.

**Body:**

> Hi {{first_name}},
>
> One week in, here's what we're seeing from teams using Phase 30 Remote Workspaces:
>
> **A data engineering team** is running a pipeline agent on a GPU instance in their own AWS account — keeping raw data on their infrastructure while using the platform for orchestration. Data residency solved.
>
> **A developer relations team** is running a local agent on their laptops for quick iteration — debugging agent behavior in their IDE, then pointing the same agent at the org for production tasks. No switching environments.
>
> **An enterprise platform team** is running agents across three clouds — GCP, AWS, and a private cloud — visible in one Canvas, governed by the same org auth. Multi-cloud fleet, single governance plane.
>
> If you've been evaluating AI agent platforms and hesitated because "my data can't leave my infrastructure," Phase 30 was built for you.
>
> [Talk to our team →](/contact) *(replace with actual sales link)*
> [Read the docs →](/docs/guides/remote-workspaces)
> [See working demos →](/marketing/demos)
>
> — The Molecule AI team

---

## Notes for CRM team

- Send from `team@moleculesai.app` or a named sender (CEO or Marketing Lead name)
- Segment by: existing customers (already on platform) vs. evaluators (visited docs, not yet a customer) — Email 2 + 3 copy can be swapped for evaluators vs. customers
- Unsubscribe link required in every email
- All internal link placeholders (`/docs/...`, `/blog/...`) must be resolved to live URLs before send
- Phase 2 + Phase 3 email body copy can be A/B tested with the alt subject lines

---

*CRM placeholders: `{{first_name}}`, `{{contact}}`, `{{sales_link}}` — resolve before launch.*
