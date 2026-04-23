# Positioning Brief: EC2 Instance ID Persistence
**PR:** [#1531](https://github.com/Molecule-AI/molecule-core/pull/1531) — `feat(workspace): persist CP-returned EC2 instance_id on provision`
**Merged:** 2026-04-22T01:40Z (~21h ago)
**Owner:** PMM | **Status:** DRAFT — pending Marketing Lead review

---

## Situation

Control Plane workspace provisioning (SaaS / Phase 30 infrastructure) runs on EC2. The CP returns an `instance_id` when a workspace is provisioned, but previously this was not stored — the platform couldn't distinguish a CP-provisioned workspace from a Docker workspace once running.

PR #1531 persists the `instance_id` returned by the CP into the workspaces table, enabling downstream features that require knowing which EC2 instance backs a workspace.

---

## Problem Statement

Downstream features — notably browser-based terminal (EC2 Instance Connect SSH, PR #1533) and audit attribution — require a reliable `instance_id` field on the workspace record. Without it:
- Terminal tab can't determine which EC2 instance to connect to
- Audit log can't cross-reference workspace events with actual EC2 activity in CloudTrail
- Cost attribution by instance can't work reliably

The CP already returns `instance_id`; the platform just wasn't storing it.

---

## Core Claims

### Claim 1: Platform now knows which EC2 instance backs each workspace

The `instance_id` is stored at provision time and available on every subsequent workspace API response. This is a prerequisite for several Phase 30 features — not visible to end users directly, but enables the features that are.

### Claim 2: Browser-based terminal is now possible for all CP-provisioned workspaces

EICE (PR #1533) uses `instance_id` to initiate the SSH session. Without #1531, EICE can't know which instance to target. Together, #1531 + #1533 = SaaS users get a terminal tab with no SSH keys.

### Claim 3: Audit trail is now attributable to specific EC2 instances

Workspace-level CloudTrail events can now be correlated to the actual EC2 instance via `instance_id`. Compliance teams get more complete audit data.

---

## Target Audience

**Primary:** DevOps and platform engineers managing SaaS-provisioned workspaces. The `instance_id` is invisible to them unless they look at the API — but the features it enables (terminal, audit) are visible.

**Secondary:** Enterprise security/compliance reviewers evaluating Molecule AI SaaS. `instance_id` persistence + CloudTrail attribution is a governance signal.

---

## Positioning Alignment

- **Phase 30 remote workspaces**: `instance_id` is prerequisite infrastructure for the SaaS-side remote workspace UX (terminal + audit)
- **Per-workspace auth tokens**: Platform-level resource identification supports token-scoped access decisions
- **Immutable audit trail**: `instance_id` cross-reference makes CloudTrail events attributable to specific workspaces

This is a **prerequisite PR** — it ships the data layer for features in PR #1533 and future CP-provisioned workspace capabilities. Not a standalone launch.

---

## Channel Coverage

| Channel | Asset | Owner | Notes |
|---------|-------|-------|-------|
| Release notes | Mention in Phase 30 release notes | DevRel | Brief entry — "EC2 instance_id now stored on provision" |
| Phase 30 blog | Call out in remote workspaces blog | Content Marketer | One sentence — "CP-provisioned workspaces now store their EC2 instance ID" |
| No standalone blog or social | Not warranted — prerequisite PR | — | |

**This is not a standalone campaign.** The value is in enabling other features.

---

## Relationship to PR #1533 (EC2 Instance Connect SSH)

PR #1531 + #1533 together deliver: SaaS workspace gets a browser-based terminal tab, no SSH keys required.

- **PR #1531**: Store the `instance_id` (data layer) ✅ **this brief**
- **PR #1533**: Connect via EICE using `instance_id` (UX layer) — brief exists at `pr-1533-ec2-instance-connect-ssh.md`

Route both to DevRel together. Content Marketer uses #1531 as one sentence in the EC2 Instance Connect SSH blog post.

---

## Sign-off

- [x] PMM positioning: approved
- [ ] Marketing Lead: pending
- [ ] DevRel: note in release notes + coordinate with #1533

---

*PMM — this PR is a prerequisite. Coordinate release note entry with #1533. Close when routed.*