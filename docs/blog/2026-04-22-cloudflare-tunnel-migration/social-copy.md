# Social Copy — Phase 33: Cloudflare Tunnel to Direct Connect
Campaign: cloudflare-tunnel-migration | Blog: `docs/blog/2026-04-22-cloudflare-tunnel-migration/`
Slug: `cloudflare-tunnel-migration`
Publish day: 2026-04-22 (Phase 33 rollout day)
Assets: None required — operational post
Hashtags: #MoleculeAI #DevOps #PlatformEngineering #CloudInfra #AIAgents
UTM: `?utm_source=twitter&utm_medium=social&utm_campaign=cloudflare-tunnel-migration`

---

## X Thread — 4 posts

### Post 1 — Hook (the before state)
We were routing every agent workspace through Cloudflare Tunnel.

Outbound WebSocket from each container → Cloudflare edge → your traffic.

For development, it worked. At fleet scale, it added latency, egress costs, and a single dependency that took down every agent when Cloudflare had issues.

Phase 33: workspaces get their own public IPs. Direct WebSocket. No tunnel in the path.

→ https://docs.molecule.ai/blog/cloudflare-tunnel-migration

---

### Post 2 — What changed (the mechanics)
Before: cloudflared daemon in every container, outbound-only tunnel, Cloudflare-assigned hostname.

After: public IP from your VPC subnet, security group rules managed by the platform, direct WebSocket on :443.

The platform still handles auth and routing. The data path is direct.

Latency improvement: ~20–40ms reduction depending on region, from dropping the Cloudflare hop.

→ https://docs.molecule.ai/blog/cloudflare-tunnel-migration

---

### Post 3 — The operational wins
Four things platform engineers actually care about:

**No egress costs** — Cloudflare metered tunnel bandwidth. At fleet scale, that compounded.

**Direct diagnostics** — `curl https://<workspace-ip>` works. No tunnel path required to check a workspace.

**No single dependency** — Cloudflare outage no longer means every agent drops.

**Lower latency** — direct path, no Cloudflare edge in the data path.

Security group rules are platform-managed. Port 443 only, TLS required, JWT validated before any data is served.

→ https://docs.molecule.ai/blog/cloudflare-tunnel-migration

---

### Post 4 — Who this affects / CTA
If you run CP-managed workspaces (MOLECULE_ORG_ID set, AWS backend): Phase 33 transitions automatically. No action needed.

New provisions: workspace subnet gets a public IP automatically. Same provisioning flow, different backend config.

Existing self-hosted or Fly.io: no change.

Check your workspace's current IP in the Canvas detail view. If you're hitting tunnel hostnames for diagnostics, you can now curl the IP directly.

→ https://docs.molecule.ai/blog/cloudflare-tunnel-migration

---

## LinkedIn — Single Post

**Title:** Why we replaced Cloudflare Tunnel with direct-connect public IPs for our agent workspaces — and what it actually changed

We had a Cloudflare Tunnel daemon running in every agent workspace container.

Outbound-only WebSocket to Cloudflare's edge. Your browser traffic routed through their network. No inbound firewall rules required.

For development environments, this was a clean tradeoff. The container opened nothing inbound. Everything worked.

At fleet scale, it stopped working.

The four compounding problems we hit:

**Latency.** Every request from the platform to the workspace traveled through Cloudflare's network — extra hops, extra milliseconds. For agents running real workloads, the overhead was measurable.

**Egress costs.** Cloudflare metered tunnel bandwidth. At fleet scale, with agents pulling models, installing packages, and streaming results, the bandwidth bill compounded.

**Single dependency.** If Cloudflare had an outage, every agent workspace lost its connection path simultaneously. Not a hypothetical — it happened.

**No direct diagnostics.** You couldn't curl a workspace's IP to check if it was up. Network checks required the tunnel path. For platform engineers, that was a real operational friction.

Phase 33 replaces Cloudflare Tunnel with direct-connect. Each workspace gets its own public IP from the VPC subnet. The data path is direct: your browser → platform API → workspace public IP, no Cloudflare in the middle.

What the platform still handles: auth, routing, security group management (port 443 only, TLS, JWT validation before data is served). What it no longer owns: the transport path.

For teams running CP-managed workspaces in AWS: the transition is automatic. New provisions get public IPs from day one. Existing workspaces migrate on their next restart cycle.

For DevOps and platform engineers evaluating agent infrastructure: this is the difference between a platform that works in demos and one that works in production at scale.

The write-up with the full architecture comparison is on the docs site.

→ https://docs.molecule.ai/blog/cloudflare-tunnel-migration

---

**Hashtags:** #PlatformEngineering #DevOps #CloudInfrastructure #AIAgents #CloudComputing #AWS
**CTA:** Bookmark for when you're evaluating or operating agent fleet infrastructure — the architecture details matter at scale.
