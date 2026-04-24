# Cloudflare Tunnel Migration — Social Copy
Campaign: Phase 33 direct-connect | Source: PR #1612
Publish day: 2026-04-22
Status: DRAFT — for Social Media Brand review + publish
Blog: `docs.moleculesai.app/blog/cloudflare-tunnel-migration` (pending)

---

## X (Twitter) — 4-post thread

### Post 1 — Hook
Your agent workspace has a new IP address.

In Phase 33, every Molecule AI workspace in your cloud account gets its own public IP — direct-connect, no Cloudflare Tunnel in the path.

That's a real change for how you run production agents.

---

### Post 2 — What changed
Before Phase 33: every workspace connected through Cloudflare Tunnel (cloudflared).
Outbound-only, no firewall rules needed. But: extra latency, egress metered by Cloudflare, single dependency.

After Phase 33: each workspace gets a VPC public IP. The platform connects directly.
Same security model, cleaner path.

---

### Post 3 — The operational wins
Direct-connect workspaces mean:
→ curl the IP directly — no tunnel diagnostic dance
→ no Cloudflare egress costs at agent-fleet scale
→ no single dependency on Cloudflare edge availability
→ platform-controlled inbound rules via AWS security groups

If you're running 10+ production agent workspaces, this compounds.

---

### Post 4 — CTA
Phase 33 is live for all new cloud-hosted workspaces.

Works with existing Molecule AI deployments — no config changes required.
Existing tunnel workspaces continue to work; direct-connect is the default for new provisions.

→ [docs link]

---

## LinkedIn — Single post

**Title:** We replaced Cloudflare Tunnel with direct-connect agent workspaces. Here's what changed.

When you run a cloud-hosted agent workspace, there are two ways it can connect to the platform:

The old way: a lightweight daemon (Cloudflare Tunnel / cloudflared) runs inside the container, maintaining an outbound-only WebSocket to Cloudflare's edge. No inbound firewall rules required. Clean and simple — until you're running 20 agents at scale.

That's the model Phase 33 replaces.

Every new workspace in your cloud account now gets its own public IP from the VPC public subnet. The platform connects directly, with the same authentication and security model. No tunnel in the path.

The operational differences matter at scale:
- **No egress costs** through Cloudflare's metered network — the workspace sends traffic directly from your VPC
- **Lower latency** — one fewer network hop through Cloudflare's edge
- **Direct diagnostics** — curl the IP, run network checks, SSH directly — no tunnel to debug
- **No single dependency** — if Cloudflare has an incident, your agents keep running

For single-agent dev environments, the tunnel model was fine. For production agent fleets, direct-connect is the right trade-off.

Phase 33 is live for all new cloud-hosted workspaces. Existing tunnel workspaces continue to function; direct-connect is the default going forward.

→ [Read the architecture walkthrough](https://docs.molecule.ai/docs/guides/remote-workspaces)
→ [Molecule AI on GitHub](https://github.com/Molecule-AI/molecule-core)

#DevOps #CloudComputing #AIAgents #AWS #MoleculeAI
