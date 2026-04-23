# EC2 Instance Connect SSH — Social Copy
**Feature:** PR #1533 — `feat(terminal): remote path via aws ec2-instance-connect + pty`
**Campaign:** EC2 Instance Connect SSH | **Blog:** `docs/infra/workspace-terminal.md` (shipped in PR #1533)
**Canonical URL:** `moleculesai.app/docs/infra/workspace-terminal`
**Status:** APPROVED — unblocked for Social Media Brand
**Owner:** PMM → Social Media Brand | **Day:** Blocked on DevRel code demo (#1545) + Content Marketer blog (#1546)
**Positioning approved by:** PMM (GH issue #1637)

---

## Headline Angle: "No SSH keys, no bastion, no public IP"
**Primary security differentiator:** Ephemeral keys (60-second RSA key lifespan via AWS API — no persistent key on disk, no rotation, no orphaned credential risk)

Secondary angle: Zero key rot — the 60-second key window means there's nothing to rotate, nothing to revoke, nothing exposed on developer machines.

---

## X / Twitter (140–280 chars)

### Version A — Infrastructure angle ✅ (ops simplicity, approved primary)
```
Your SaaS-provisioned EC2 workspace has a terminal tab. No SSH keys needed.

Molecule AI connects via EC2 Instance Connect Endpoint — IAM-authorized, no bastion, no public IP required.

One click. You're in.
```

### Version B — Zero credential overhead (ops simplicity)
```
Connecting to a cloud VM used to mean: SSH key, bastion host, public IP, and a security review.

EC2 Instance Connect changes that. Your IAM role is the auth layer. No keys on disk. No rotation. No gap.

The terminal just works.
```

### Version C — Developer angle (DX)
```
Your agent's EC2 workspace just got a terminal tab.

No pre-configured SSH keys. No bastion. No public IP needed.

Molecule AI handles EC2 Instance Connect for you — IAM-authorized, PTY over WebSocket, in the canvas.

That's the SaaS difference.
```

### Version D — Security / Enterprise (zero key rot) ✅
```
SSH key left on a laptop. Former employee. Rotation takes a week.

EC2 Instance Connect: every connection uses an ephemeral key pushed to instance metadata — valid 60 seconds, never touches a developer machine.

No orphaned keys. No rotation SLAs. IAM is the auth layer.

Security teams notice this architecture.
```

### Version E — Ephemeral key story (new — security lead)
```
Traditional SSH: key lives on disk, gets shared, gets forgotten, becomes a liability.

EC2 Instance Connect SSH in Molecule AI: a temporary RSA key appears in instance metadata for 60 seconds, then disappears.

No key on disk. No key rotation. No blast radius when someone leaves.

The terminal just works. The key doesn't outlast the session.
```

### Version F — Problem → solution (ops lead)
```
Problem: SaaS-provisioned EC2 workspaces don't have a terminal tab without SSH keys, a bastion, and a public IP.

Solution: EC2 Instance Connect Endpoint. IAM-authorized. Platform-initiated. No user-side key management.

Your canvas workspace just got a shell.
```

---

## LinkedIn (100–200 words)

```
Getting a terminal into a cloud VM shouldn't require a security review, a bastion host, and an SSH keypair.

For SaaS-provisioned workspaces — the ones running on Fly Machines or EC2 — that was the reality until this week. Connecting to a remote VM meant: pre-configured keys, a jump box, and either a public IP or an SSM agent installed per instance.

EC2 Instance Connect Endpoint changes this. The platform's IAM credentials authorize the connection. A temporary RSA key appears in the instance metadata (valid for 60 seconds), and the session is proxied over WebSocket to the canvas terminal tab. No keys on disk. No bastion. No configuration required.

The terminal tab appears automatically for every CP-provisioned workspace. The connection is IAM-authorized, so every session is attributable in CloudTrail. Revocation is immediate — stop the IAM role, the connection stops.

This is what SaaS terminal access looks like when it's designed for agents, not humans with SSH config files.
```

---

## Image suggestions

| Post | Image | Source |
|---|---|---|
| X Version A | Canvas screenshot: terminal tab open on a REMOTE badge workspace | Custom: needs DevRel code demo screenshot |
| X Version D | Timeline graphic: "Key pushed to metadata → 60s window → key invalidated" | Custom: AWS/EC2 flow diagram |
| X Version E | Before/after: key-on-disk vs ephemeral key lifecycle | Custom graphic |
| X Version F | Problem/solution card: "Before: bastion + keys + public IP" vs "After: one click, canvas terminal" | Custom graphic |
| LinkedIn | Canvas terminal screenshot with REMOTE badge | Custom |

---

## Hashtags

`#MoleculeAI` `#AWS` `#EC2` `#AIInfrastructure` `#AgentPlatform` `#DevOps` `#Security` `#A2A` `#RemoteWorkspaces`

**Note:** `#AgenticAI` removed — does not appear in Phase 30 positioning brief; keep messaging consistent.

---

## CTA

`moleculesai.app/docs/infra/workspace-terminal`

---

## Campaign timing

Dependent on: DevRel code demo (#1545) → Content Marketer blog (#1546) → Social Media Brand launch thread.
Recommended: Coordinate with DevRel screencast; social posts should reference the demo for credibility.

---

*PMM drafted 2026-04-22 — updated 2026-04-22 (GH issue #1637 positioning decision: lead with ops simplicity, highlight ephemeral key property in security-focused posts)*
*Positioning brief: `docs/marketing/launches/pr-1533-ec2-instance-connect-ssh.md`*
