# Social Copy — Deploy AI Agents on Fly.io Campaign
## Blog Post: "Deploy AI Agents on Fly.io — or Any Cloud — with One Config Change"
**URL:** /blog/deploy-anywhere
**Date:** 2026-04-17 (published)
**Author:** Content Marketer (draft — for Social Media Brand review + publish)
**Status:** DRAFT — pending Social Media Brand + Marketing Lead review

---

## X / Twitter Thread

**Post 1 (Hook):**
> Your infrastructure choice just got decoupled from your agent platform.

Until this week: Molecule AI workspaces ran on Docker. One backend. One option.

Now there are three. And switching takes one environment variable.

---

**Post 2 (What's new):**
> Molecule AI now ships three production-ready workspace backends:

🐳 Docker — self-hosted, no external deps
🚀 Fly.io Machines — pay-per-use, scale to zero
☁️ Control Plane API — multi-tenant SaaS, credential isolation built in

Same agent code. Same API surface. Just flip a config flag.

---

**Post 3 (The security angle — SaaS teams):**
> If you're building a SaaS product on Molecule AI, you have a Fly API token problem.

Every tenant platform instance that carries a `FLY_API_TOKEN` is one misconfiguration away from a credential exposure.

The fix: `CONTAINER_BACKEND=controlplane`. Fly credentials live in Molecule AI's control plane — never on the tenant.

Architecture: Canvas → Tenant Platform → Control Plane API → Fly Machines API

---

**Post 4 (The indie dev angle):**
> On Fly.io already?

Three env vars and your Molecule AI workspaces are Fly Machines:

```bash
CONTAINER_BACKEND=flyio
FLY_API_TOKEN=<your-token>
FLY_WORKSPACE_APP=<your-app>
```

Pay for what you use. Scale to zero. No idle Docker host.

---

**Post 5 (Comparison table):**
> Quick guide: which backend fits?

| Use case | Backend |
|---|---|
| Self-hosted / local dev | Docker (default) |
| On Fly, small team | flyio |
| SaaS, multi-tenant | controlplane |

Picking your backend → deploying your agents.

Link in bio.

---

## LinkedIn Post

**Single post:**

We just decoupled Molecule AI's infrastructure from its agent platform.

Before this week: one deployment model. Docker. End of story.

Now: three backends — Docker, Fly Machines, and a control plane API for SaaS teams. Same agent code across all three. Switching is a single environment variable.

The two groups who were making compromises they shouldn't have to:

**Indie developers on Fly** — you wanted Fly's economics: pay-per-use, scale to zero, no idle infrastructure. Now you get it. Three env vars and your Molecule AI workspaces are Fly Machines in your own account.

**SaaS builders** — the Fly API token sitting on your tenant platform instance is a structural security problem, not a policy problem. With `CONTAINER_BACKEND=controlplane`, Fly credentials live in the Molecule AI control plane — structurally isolated from your tenants from day one.

Both groups now get the deployment model they need without sacrificing the agent platform they chose.

Full breakdown of all three backends, with env var reference tables, in the blog post.

→ [Read: "Deploy AI Agents on Fly.io — or Any Cloud — with One Config Change"](https://docs.molecule.ai/blog/deploy-anywhere?utm_source=linkedin&utm_medium=social&utm_campaign=fly-deploy-anywhere)

#AIagents #Flyio #SaaS #DeveloperTools #DevOps #MultiTenant

---

## Image / Visual Recommendations

| Platform | Asset | File |
|---|---|---|
| X/LinkedIn | Architecture diagram | Canvas → Tenant Platform → Control Plane API → Fly Machines. Clean, labeled boxes. |
| X/LinkedIn | Comparison table card | `assets/backend-comparison-card.svg` |
| X (thread) | Env var code card | Three env vars, clean syntax highlight. "Three lines. Done." |
| X/LinkedIn | "Before vs After" | Left: one backend (Docker). Right: three backends (Docker + Fly + Control Plane). Shows expansion. |

**Generated assets available in `docs/marketing/campaigns/fly-deploy-anywhere/assets/`:**
- `backend-comparison-card.svg` — 3 backend comparison with env vars, use cases, credential ownership

---

## Hashtag Set
#AIagents #Flyio #SaaS #DeveloperTools #DevOps #MultiTenant #CloudDeployment #SelfHosting

---

## UTM Tags
Append `?utm_source=linkedin&utm_medium=social&utm_campaign=fly-deploy-anywhere` to LinkedIn links.
Append `?utm_source=twitter&utm_medium=social&utm_campaign=fly-deploy-anywhere` to X links.

---

## Publishing Notes
- Published 2026-04-17 — this copy can be used retroactively for ongoing distribution
- Cross-links naturally to the Chrome DevTools MCP blog post (2026-04-20) — consider stacking both in the same social week
- Social Media Brand: coordinate with Chrome DevTools MCP post social push to avoid publishing both on the same day

---

*Draft by Content Marketer 2026-04-20 — for Social Media Brand review before publishing*
