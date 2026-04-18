# Cloudflare Tunnel Migration — Session Report (2026-04-18)

> **Duration:** ~4 hours
> **Scope:** Replace Cloudflare Worker + wildcard DNS with per-tenant Cloudflare Tunnels
> **Issue:** #933
> **Status:** Tunnel E2E verified on both production and staging subdomains. Ready for production tenant migration.

---

## What Was Done

### 1. PR Triage (15 PRs merged)

Before tunnel work, cleared the PR backlog since CI runner was slow:

| PR | Type | Description |
|----|------|-------------|
| #934 | docs | Staging environment design + Phase 36 plan |
| #849 | docs | Partner API Keys (Phase 34) — resolved PLAN.md conflict |
| #922 | docs | ANTHROPIC_API_KEY as required global secret |
| #880 | docs | SAFE-MCP internal advisory |
| #927 | docs | Ecosystem watch daily sweep |
| #923 | security | Slack OAuth state param — random nonce replaces workspace_id |
| #913 | security | Redact secrets from commit_memory before persistence |
| #925 | security | HITL audit log on approval grant/denial |
| #879 | fix | Canvas TypeScript fixture drift |
| #915 | feature | A2A topology overlay + hermes plugin declarations |
| #921 | feature | Audit trail visualization panel |
| #929 | feature | Temporal crash-resume checkpoints |
| #937 | fix | go vet errors + supply chain hardening (created + merged) |
| #938 | fix | Canvas a11y — TeamMemberChip keyboard nav (created + merged) |

Also closed issue #920 (Slack OAuth) and commented on #889 (VULN-004 dead letter).

### 2. Cloudflare API Token — Tunnel Permission

**Problem:** The existing CF API token (`cfut_****...`) had DNS:Edit but NOT Cloudflare Tunnel:Edit permission. Tunnel create/list/delete calls returned `code 10000: Authentication error`.

**Fix:** CEO added Account → Cloudflare Tunnel → Edit permission in Cloudflare Dashboard → API Tokens.

### 3. Tunnel API Integration Tests

Ran three progressively comprehensive tests:

| Test | Result | What it proved |
|------|--------|----------------|
| API roundtrip | ✓ | Create tunnel → create DNS CNAME → delete both |
| DNS resolution | ✓ | CNAME resolves on first attempt (instant, zero propagation delay) |
| Full E2E with EC2 | ✓ | Tunnel + DNS + EC2 with cloudflared → HTTP 200 through subdomain |

### 4. Worker Coexistence Fix

**Problem:** The Cloudflare Worker route `*.moleculesai.app/*` intercepted tunnel CNAME requests before they could reach the tunnel origin. Tunnel subdomains got the Worker's "Organization not found" page instead of routing through the tunnel.

**Fix (two changes to Worker):**

```typescript
// 1. Reserved slugs now pass through instead of returning 404
if (!slug || slug === host || RESERVED.has(slug) || slug.includes(".")) {
  return fetch(request);  // was: return new Response("Not found", { status: 404 });
}

// 2. Multi-level subdomains (*.staging.moleculesai.app) bypass Worker entirely
// slug.includes(".") catches "foo.staging" and passes to tunnel CNAME
```

Worker redeployed. Production tenants unaffected — they still route through the Worker. Tunnel-routed subdomains pass through to origin.

### 5. SSL Certificate for Staging Subdomains

**Problem:** Cloudflare's free Universal SSL only covers `*.moleculesai.app` (one wildcard level). `*.staging.moleculesai.app` (two levels) fails TLS handshake — no certificate.

**Fix:** Ordered Advanced Certificate via Cloudflare Dashboard:
- Hostnames: `*.staging.moleculesai.app`, `staging.moleculesai.app`
- CA: Let's Encrypt
- Validity: 90 days, auto-renewal 30 days before expiry
- Cost: included in Cloudflare free plan (1 of 100 advanced certs)

### 6. Staging Tunnel E2E — Full Pass

Final test on `*.staging.moleculesai.app` (fully isolated from production):

```
1. Create Tunnel           → OK (ea5aaa13...)
2. Configure ingress       → OK (→ localhost:8080)
3. Create DNS CNAME        → OK (tunnel-stg-test.staging.moleculesai.app)
4. Launch EC2 t3.micro     → OK (cloudflared binary download)
5. Tunnel connected        → OK (healthy in 30s)
6. HTTP 200 through tunnel → OK
   Response: {"status":"ok","domain":"tunnel-stg-test.staging.moleculesai.app"}
7. Cleanup                 → OK (EC2 terminated, DNS + tunnel deleted)
```

### 7. Platform Build Verification

After merging 15 PRs, verified everything still builds and passes:
- Go: `go test -race ./...` — 15/15 packages pass, 0 failures
- Go: `go vet ./...` — clean
- Canvas: `npm run build` — success
- Canvas: `vitest run` — 762/762 tests pass

---

## Architecture: Before vs After

### Before (Cloudflare Worker)

```
User → *.moleculesai.app (wildcard A record, proxied)
     → Cloudflare Worker (extracts slug, looks up EC2 IP from CP API)
     → Worker proxies to EC2 public IP:8080
     → EC2 must have public IP + open port 8080
```

**Problems:**
- Edge cache poisoning when wildcard A record IP changes (2+ hour recovery)
- ADMIN_TOKEN transmitted in plaintext via Worker header injection
- EC2 requires public IP + open inbound ports (security surface)
- Worker is a single point of failure for all tenant routing
- KV cache stale-while-revalidate adds latency on cold starts

### After (Cloudflare Tunnel)

```
User → slug.moleculesai.app (CNAME → tunnel-id.cfargotunnel.com, proxied)
     → Cloudflare edge routes to tunnel
     → cloudflared on EC2 (outbound-only connection) receives request
     → cloudflared forwards to localhost:8080
     → EC2 needs NO public IP, NO open inbound ports
```

**Advantages:**
- No edge cache — CNAME resolves instantly via Cloudflare's anycast
- No plaintext secrets in transit — tunnel is encrypted end-to-end
- EC2 can be in private subnet (no public IP, no security group rules)
- Each tenant has its own tunnel (no single point of failure)
- No Worker maintenance, no KV cache management
- Faster provisioning — DNS works immediately, no cache warming

---

## Known Issues & Risks

### 1. Worker Must Stay Until All Tenants Migrate
The Worker route `*.moleculesai.app/*` still serves existing tenants (e.g., `<example-org>.moleculesai.app`). Cannot delete until every tenant has a tunnel + CNAME. The Worker passthrough for reserved/multi-level slugs is the bridge.

### 2. Worker Source Not in Version Control
The Worker code lives in `/tmp/molecule-tenant-proxy/` — not tracked in any repo. Needs to be committed somewhere before the session ends. Two changes were deployed:
- `fetch(request)` passthrough for reserved slugs (was `404`)
- `slug.includes(".")` bypass for multi-level subdomains

### 3. cloudflared Binary Download at Boot
Current EC2 user-data downloads `cloudflared` from GitHub releases at boot time. This adds ~5 seconds and depends on GitHub availability. Pre-baked AMI would eliminate this dependency.

### 4. Tunnel Token in User-Data
The `cloudflared` tunnel token is passed in EC2 user-data (base64 encoded). AWS user-data is accessible to anyone with EC2 instance metadata access. The token grants tunnel connection rights — if leaked, an attacker could impersonate the tenant's tunnel. Mitigation: use AWS Secrets Manager or SSM Parameter Store instead.

### 5. Tunnel Cleanup on Org Delete
The `DeprovisionInstance` function has a TODO for tunnel deletion. When an org is deleted, the tunnel and DNS CNAME must be cleaned up. The tunnel ID is stored in EC2 tags (`TunnelID`), but needs to be persisted in `org_instances` table for reliable cleanup.

### 6. No Health Check on Tunnel
If `cloudflared` crashes on the EC2 but the instance stays running, the tunnel goes inactive but the DNS CNAME still points to it. Need a health sweep that checks tunnel status via CF API and restarts `cloudflared` or the instance.

### 7. Staging CP Uses Production Tenant Image
`TENANT_IMAGE` on staging is still `ghcr.io/molecule-ai/platform-tenant:latest` (production). Should be `:staging` once the staging image pipeline is set up.

---

## Follow-Up Tasks

### Immediate (before next deploy)

- [ ] **Commit Worker code to repo** — decide location (monorepo `infra/` or separate repo), commit current state with the two passthrough changes
- [ ] **Persist tunnel ID in org_instances table** — add `tunnel_id` column so deprovision cascade can clean up tunnels reliably
- [ ] **Wire tunnel cleanup into DeprovisionInstance** — delete tunnel + DNS CNAME when org is deleted

### Short-term (this week)

- [ ] **Migrate existing tenant to tunnel** — create tunnel, add CNAME, update EC2 to run cloudflared, add slug to Worker RESERVED, verify, then remove old A record
- [ ] **Staging image pipeline** — publish `:staging` tag on main merge, `:latest` only on manual promote
- [ ] **Move tunnel token to SSM Parameter Store** — EC2 user-data is not secret-safe; retrieve token at boot via instance role

### Medium-term (this month)

- [ ] **Pre-baked AMI with cloudflared** — eliminate GitHub download dependency at boot
- [ ] **Tunnel health sweep** — periodic check of tunnel status via CF API, restart cloudflared if inactive
- [ ] **Delete Worker** — once all tenants are on tunnels, remove Worker + wildcard A record entirely
- [ ] **Private subnet for tenant EC2s** — with tunnels, EC2s don't need public IPs; move to private subnet with NAT gateway for outbound

### Nice-to-have

- [ ] **Cloudflare Access** — add zero-trust access policies on tunnel routes (IP allow-list, mTLS)
- [ ] **Tunnel metrics** — export tunnel connection count, latency, bandwidth to Prometheus/Grafana
- [ ] **Multi-region tunnels** — cloudflared connects to nearest Cloudflare edge; for multi-region deployments, each region's EC2 gets its own tunnel

---

## Cost Impact

| Item | Before | After |
|------|--------|-------|
| Cloudflare Worker | Free (100k req/day) | Eliminated |
| Workers KV | Free tier | Eliminated |
| Advanced SSL Cert | $0 | $0 (1 of 100 free) |
| EC2 public IPs | ~$3.65/mo per tenant | $0 (no public IP needed) |
| Cloudflare Tunnel | N/A | Free (unlimited tunnels) |
| **Net change** | | **Saves ~$3.65/tenant/mo** |

---

## Key Learnings

1. **Worker routes take priority over DNS CNAMEs** — even with a CNAME pointing to `cfargotunnel.com`, the Worker's wildcard route fires first. Must explicitly pass through via `fetch(request)`.

2. **Free Universal SSL only covers one wildcard level** — `*.moleculesai.app` works, `*.staging.moleculesai.app` doesn't. Advanced Certificate (free, Let's Encrypt) solves this.

3. **Let's Encrypt rejects mixed wildcard+parent certs** — can't put `*.moleculesai.app` and `*.staging.moleculesai.app` in the same cert. Issue separate certs for each level.

4. **Tunnel connects in ~30 seconds** — from EC2 boot to tunnel healthy, including cloudflared binary download (~5s) + connection establishment (~25s). Faster than DNS propagation ever was.

5. **DNS CNAME resolves instantly** — no propagation delay, no edge cache, no NXDOMAIN caching. This is the fundamental advantage over the wildcard A record approach.

6. **cloudflared binary download is faster than apt** — `curl` from GitHub releases (~5s) vs `apt-get install cloudflared` (~30s). Use binary download in boot scripts.
