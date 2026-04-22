# A2A Protocol Deep-Dive — Social Queue

## Post Metadata
- **Blog post:** `docs/blog/2026-04-22-a2a-protocol/index.md`
- **Live URL:** `https://molecule.ai/blog/a2a-protocol`
- **Publish date:** 2026-04-22
- **Author:** Molecule AI
- **Tags:** A2A, agent-to-agent, MCP, AI agents, architecture

---

## X / Twitter — Thread Version

**Tweet 1 (hook):**
> Most AI agent platforms bolt on agent-to-agent communication as an afterthought. Molecule AI was designed around A2A from day one. Here's what that actually means in the code.

**Tweet 2:**
> In Molecule AI, the A2A proxy is the ONLY path between the canvas UI and workspace agents. No back doors. No secondary REST path. Every request is an A2A message.

**Tweet 3:**
> Proof: the auth model is baked into the A2A proxy itself. Phase 30.5 caller token binding validates that workspace A's token can't forge requests as workspace B.

**Tweet 4:**
> Agent discovery? A2A. Delegation? A2A. The scheduler? A2A. When everything is A2A, the audit log is complete by construction — no dark corners.

**Tweet 5 (CTA):**
> The A2A protocol is available in every Molecule AI workspace. Open source. Read the proxy implementation here →

**Tweet 5 link:** https://github.com/Molecule-AI/molecule-core/tree/main/workspace-server/internal/handlers/

**Media:** OG image (1200×630) — generate from brand template, dark, tech-forward. Headline: "A2A Protocol: The Foundational Architecture of Molecule AI Agents"

---

## X / Twitter — Single Post Version

> Most AI agent platforms treat agent-to-agent communication as an add-on feature. Molecule AI was built around the A2A protocol from the start — every canvas request, every delegation, every scheduled task is an A2A message. Here's 5 proof points from the codebase. https://molecule.ai/blog/a2a-protocol

**Alt text for OG image:** Diagram showing A2A proxy routing canvas requests to workspace agents, with auth, SSRF, and audit checks inline.

---

## LinkedIn Post Version

**Hook line:**
> We didn't add A2A support to Molecule AI. We built Molecule AI around it.

**Body:**
> The Agent-to-Agent (A2A) protocol is a JSON-RPC 2.0 specification for how AI agents discover each other, exchange tasks, and coordinate across a distributed system.

> Most platforms treat this as a feature. In Molecule AI, it's the only path.

> Here's what "A2A-native" actually means in production code — five proof points from our codebase:

> **1. The proxy is the only path.**
> Every canvas-to-agent request flows through a single function. No back doors. That means the same auth, SSRF, budget, and audit controls apply everywhere, with no gaps.

> **2. Auth is A2A-native.**
> Phase 30.5 caller token binding validates that workspace A's token cannot forge requests as workspace B. The token is bound to the caller, not the target.

> **3. Discovery is access-controlled at the A2A layer.**
> A workspace can only discover peers it is permitted to communicate with. The policy is the same CanCommunicate hierarchy check used by the A2A proxy.

> **4. Delegation is the A2A protocol.**
> When you delegate a task in Molecule AI, you are firing a standard A2A `message/send` request. The full lifecycle — pending → dispatched → completed/failed — is tracked as A2A activity.

> **5. The scheduler is a first-class A2A client.**
> Scheduled tasks use the same execution path as on-demand delegations. Same retry logic, same timeout handling, same error classification.

> Molecule AI is open source. Read the A2A proxy, delegation handler, discovery handler, and scheduler in our public repository.

**CTA:** Read the full deep-dive with code examples → https://molecule.ai/blog/a2a-protocol

**Tags:** #A2A #AgentToAgent #MCP #AIAgents #MoleculeAI #AgentPlatform #EnterpriseAI

---

## Timing
- Publish blog post first
- Post to X/LinkedIn 30 minutes after blog goes live
- Schedule follow-up X thread for 48 hours later (organic reach)

---

## Assets Needed
- OG image: 1200×630px, dark brand theme, headline "A2A Protocol: The Foundational Architecture of Molecule AI Agents"
- Assign to: Social Media Brand
