# EC2 Instance Connect SSH — Social Copy
**Feature:** PR #1533 — `feat(terminal): remote path via aws ec2-instance-connect + pty`
**Campaign:** EC2 Instance Connect SSH | **Blog:** `docs/infra/workspace-terminal.md` (shipped in PR #1533)
**Canonical URL:** `moleculesai.app/docs/infra/workspace-terminal`
**Status:** DRAFT — PMM proactive draft; no file existed before this entry
**Owner:** PMM → Social Media Brand | **Day:** Blocked on DevRel code demo (#1545) + Content Marketer blog (#1546)

---

## X (140–280 chars)

### Version A — Infrastructure angle
```
Your SaaS-provisioned EC2 workspace has a terminal tab. No SSH keys needed.

Molecule AI connects via EC2 Instance Connect Endpoint — IAM-authorized, no bastion, no public IP required.

One click. You're in.
```

### Version B — Zero credential overhead
```
Connecting to a cloud VM used to mean: SSH key, bastion host, public IP, and a security review.

EC2 Instance Connect changes that. Your IAM role is the auth layer. No keys on disk. No rotation. No gap.

The terminal just works.
```

### Version C — Developer angle
```
Your agent's EC2 workspace just got a terminal tab.

No pre-configured SSH keys. No bastion. No public IP needed.

Molecule AI handles EC2 Instance Connect for you — IAM-authorized, PTY over WebSocket, in the canvas.

That's the SaaS difference.
```

### Version D — Security / Enterprise
```
SSH key left on a laptop. Former employee. Rotation takes a week.

EC2 Instance Connect: no shared keys, no orphaned credentials, every connection authorized via IAM and logged in CloudTrail.

Security teams notice this architecture.
```

### Version E — Problem → solution
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
| X Version B | Before/after: SSH key config vs "just click terminal" | Custom graphic |
| X Version C | Terminal demo: IAM auth flow → canvas terminal | Custom: DevRel code demo output |
| X Version D | IAM policy diagram: EC2 Instance Connect → CloudTrail log entry | Custom: AWS CloudTrail screenshot |
| X Version E | Problem/solution card: "Before: bastion + keys + public IP" vs "After: one click, canvas terminal" | Custom graphic |
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

*PMM drafted 2026-04-22 — no prior social copy file found anywhere in workspace*
*Positioning brief: `docs/marketing/launches/pr-1533-ec2-instance-connect-ssh.md`*