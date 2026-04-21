# Phase 30 Launch — Community Announcements

> **For:** DevRel / Community Manager | **Status:** ✅ Publish-ready
> **Channels:** Discord, Slack (public channels), relevant forums

---

## Discord — #announcements

**Subject:** Phase 30 is GA — Remote Workspaces are live

```
Phase 30 is generally available as of today.

Remote Workspaces let you run Molecule AI agents on any machine — your laptop, a cloud VM, an on-prem server — and they show up in Canvas like every other workspace. Same auth, same A2A protocol, same audit trail.

Quickstart → https://moleculesai.app/docs/guides/remote-workspaces

Two features that shipped with Phase 30 worth highlighting:
• AGENTS.md auto-generation — peer agents can read each other's manifest without system prompts (AAIF standard)
• Cloudflare Artifacts integration — workspace state can be versioned in a git repo, forked into new agents

Demo walkthroughs → https://moleculesai.app/docs/marketing/demos

Questions? Drop them here or in #support.
```

---

## Discord — #remote-workspaces (new or existing channel)

```
Heads up: Remote Workspaces are now GA in Phase 30.

If you've been waiting for a way to run agents locally (for debugging) or in your own cloud account, this is the release.

What changed:
• Agent runtime: remote (connects via WSS, no inbound ports needed)
• Auth: org-scoped bearer token — same as container workspaces
• Canvas: REMOTE badge shows the runtime type
• A2A: works across container/remote without code changes

Docs → https://moleculesai.app/docs/guides/remote-workspaces
FAQ → https://moleculesai.app/docs/guides/remote-workspaces-faq

Known issues → reply here or ping me.
```

---

## Slack — #general or #launch (public org Slack)

```
Phase 30 is live.

Remote Workspaces are now generally available. You can run Molecule AI agents on your own infrastructure — laptop, cloud VM, on-prem — and they'll register to your org and appear in Canvas.

Key detail for teams evaluating data residency: agent compute can stay on your infrastructure. The platform handles orchestration, auth, and coordination.

Docs: https://moleculesai.app/docs/guides/remote-workspaces
Quickstart: https://moleculesai.app/docs/guides/remote-workspaces#quick-start
Launch post: https://moleculesai.app/blog/remote-workspaces-ga
```

---

## Slack — #devrel / #community (ecosystem channels)

```
Phase 30 is GA.

Two things that shipped that the agent ecosystem community might care about:

1. AGENTS.md is now auto-generated at workspace boot — implements the AAIF / Linux Foundation standard. Peer agents can discover each other's identity and tools without reading system prompts. PR: molecule-core#763

2. Cloudflare Artifacts git integration — every workspace can have a git repo for versioned state snapshots. Fork the repo to bootstrap a new agent from any checkpoint. PR: molecule-core#641

Working demos with full API examples: https://moleculesai.app/docs/marketing/demos

If you're building agent coordination tooling, these two features should make your life easier.
```

---

## Reddit — r/MachineLearning / r/LocalLLaMA (if applicable)

**Post title:** Molecule AI Phase 30: Remote Workspaces are GA — agents that run on your own infrastructure

**Body (adapt from HN submission above)** — keep it technical, no marketing language, short.

---

## Notes

- Post Discord/Slack announcements the morning of launch day (09:00 UTC window)
- Reddit posts should go up after Discord/Slack (don't want to look like spam across channels simultaneously)
- Customize [CHANNEL-WELCOME-TONE] per channel — `#general` should be accessible, `#engineering` can be more technical
- All links assume docs site is live — confirm before posting

---

*Ready to publish. Customize sender name and channel-specific opening lines before posting.*
