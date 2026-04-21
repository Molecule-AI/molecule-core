# FOR IMMEDIATE RELEASE

## Molecule AI Launches Phase 30: Remote Workspaces Bring AI Agent Fleets to Any Infrastructure

*Platform update enables enterprises to run autonomous AI agents on-premises, in any cloud, or on a developer's laptop — while maintaining single-pane-of-glass orchestration and governance*

**[Date: April 20, 2026] — Molecule AI** today announced the general availability of Phase 30: Remote Workspaces, a platform update that allows AI agents to run on any infrastructure — a developer's laptop, a cloud VM, or an on-premises server — while remaining fully visible and governed within the Molecule AI platform.

Until now, Molecule AI customers who wanted the platform's agent orchestration, A2A coordination, and governance features had to run agents on the platform's infrastructure. Phase 30 removes that constraint. Agents can now register to a Molecule AI org from external machines using a lightweight, outbound-only connection, and appear in Canvas alongside managed (container) workspaces — with no code changes required.

---

### What Phase 30 Ships

Phase 30 is eight bounded improvements packaged as one coherent feature:

- **Remote runtime** — Agent binary connects via WSS. No inbound ports, no VPN. Outbound HTTPS to the platform only.
- **Workspace auth tokens** — Cryptographic 256-bit bearer identities, minted at registration. No shared secrets.
- **Token-gated secrets pull** — Agents pull API keys from the platform at boot. No credentials in container images.
- **Mixed-fleet Canvas** — Container and remote workspaces appear in the same Canvas. Same status, same chat, same audit trail.
- **A2A across runtimes** — Agents on different runtimes communicate via A2A without code changes.
- **AGENTS.md auto-generation** — Every workspace generates a machine-readable agent manifest at boot. Peer agents can discover each other's identity and tools without reading system prompts. (AAIF / Linux Foundation standard.)
- **Cloudflare Artifacts integration** — Every workspace can be linked to a git repo for versioned state snapshots. Agents can fork repos to bootstrap from any checkpoint.
- **`/cp/*` reverse proxy** — Allowlist-based same-origin access for internal APIs. Fail-closed.

---

### Why It Matters

The enterprise AI agent landscape is fragmenting along infrastructure lines. Some teams need agents that run on-premises due to data-residency requirements. Others need agents that run in their own cloud accounts. Many want the ability to debug agents locally before promoting them to production. Phase 30 was designed for all three scenarios simultaneously — without forcing customers to choose between platform convenience and infrastructure control.

"With Phase 30, we made the infrastructure choice optional," said [NAME, TITLE]. "Where the agent runs is now a deployment decision — not an architectural constraint. Customers can run managed agents for standard tasks and remote agents for data-locality or environment-specific requirements, in the same Canvas, with the same governance."

---

### Use Cases

- **Data residency** — Run agent compute on-premises or in a private cloud account. Raw data never touches the Molecule AI platform.
- **Developer iteration** — Run an agent locally for debugging with an IDE, then point the same agent at the org for production tasks.
- **Multi-cloud fleet management** — Run agents across AWS, GCP, and on-premises simultaneously. Visible in one Canvas, governed by one auth system.
- **Existing agent integrations** — Register an existing agent with the org without containerizing and redeploying it.

---

### Availability

Phase 30: Remote Workspaces is generally available as of April 20, 2026. Remote workspaces are priced identically to container workspaces at GA. Self-serve setup takes under five minutes.

- **Docs:** https://moleculesai.app/docs/guides/remote-workspaces
- **Quickstart:** https://moleculesai.app/docs/guides/remote-workspaces#quick-start
- **Launch post:** https://moleculesai.app/blog/remote-workspaces-ga
- **Working demos:** https://moleculesai.app/docs/marketing/demos

---

### About Molecule AI

Molecule AI is an agent orchestration platform for autonomous AI agent fleets. The platform provides A2A task dispatch, multi-workspace Canvas, org-scoped auth, and MCP governance. Used by platform engineering teams, data engineering teams, and enterprise organizations running multi-agent workflows.

---

## Media Contact

[NAME]
[EMAIL]
[moleculesai.app](https://moleculesai.app)

---

## Notes for PR team

- **[Date]** field: replace with actual press release publish date
- **[NAME, TITLE]** field: replace with quote attribution from CEO or CTO
- **[MEDIA CONTACT]** fields: replace with actual PR contact details
- Embargo: confirm whether this should be under embargo until a specific time
- Distribution: wire services (PR Newswire, Business Wire) or direct media outreach
- Follow-up: schedule analyst briefing for enterprise-focused analysts (Gartner, Forrester if applicable)
- Links assume docs site is live — confirm before finalizing

---

*Replace `[BRACKETED]` placeholders before distribution. Check all links for live URLs.*
