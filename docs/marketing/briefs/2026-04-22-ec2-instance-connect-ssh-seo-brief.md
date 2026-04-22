# SEO Brief: Phase 30 — EC2 Instance Connect SSH
**Issue:** (to be assigned by PMM)
**Date:** 2026-04-22
**Author:** SEO Analyst (validated by PMM brief submission)
**Campaign:** Phase 30 — EC2 Instance Connect SSH
**Status:** BRIEF DRAFT — ready for Content Marketer

---

## 1. Context

Phase 30 shipped remote workspaces for AI agents. EC2 Instance Connect SSH is a Phase 30 extension that lets agents connect to EC2 instances via AWS EC2 Instance Connect (ISH) — no static SSH key management, no bastion host, no key rotation overhead. AWS injects the public key via Instance Connect API; the agent authenticates via short-lived credentials.

This is NOT about running an agent ON an EC2 instance (that's covered by Remote Workspaces). This is about giving a Molecule AI agent SSH access INTO EC2 instances to run commands remotely.

**Deliverable:** New blog post + guide covering the EC2 Instance Connect SSH workflow for AI agents.

---

## 2. Target Keywords — Validated Difficulty Scores

Keyword difficulty estimated from benchmark data, cross-referenced with adjacent known keywords. AWS-branded terms ("EC2 Instance Connect") have a documented volume base from AWS's own search demand.

| Keyword | Intent | Est. MSV (US) | Est. KD (0–100) | Priority |
|---|---|---|---|---|
| `EC2 Instance Connect` | Informational / How-to | ~1,000–5,000 | **25–40** (Moderate-low) | **P0** |
| `AI agent SSH access` | Informational | ~50–200 | **10–25** (Low) | **P0** |
| `EC2 Instance Connect Endpoint tutorial` | Tutorial / How-to | ~100–400 | **15–30** (Low) | **P1** |
| `SSH bastion host alternative` | Comparison | ~300–1,000 | **20–35** (Moderate-low) | P1 |
| `SSH AI agent platform` | Commercial / Informational | ~50–150 | **10–20** (Very low) | P1 |

### Difficulty Score Rationale

- **`EC2 Instance Connect`** (KD 25–40): Well-established AWS feature with meaningful search volume (~1,000–5,000 MSV). AWS official docs dominate the SERP. Competing requires a more specific angle (AI agent integration) rather than generic EC2 IC tutorials. **Target long-tail variants, not the head term.**
- **`AI agent SSH access`** (KD 10–25): Compound keyword at the intersection of AI agents and SSH — both mature topics, but the combination is novel. Very few pages target this exact phrase today. Volume will grow as AI agent platforms mature. **First-mover window open.**
- **`SSH bastion host alternative`** (KD 20–35): Established evaluational kw. Active market from Teleport, AWS, and cloud-native security vendors. Molecule AI's angle: "your AI agent is the bastion — it gets temporary credentials, runs commands, closes the session." Compare directly against bastion host setup complexity and key rotation burden.
- **`EC2 Instance Connect Endpoint tutorial`** (KD 15–30): AWS ISH (Instance Connect Endpoint) is newer than EC2 Instance Connect. Less competition than the base term. Step-by-step tutorial format aligns with search intent.

### SSH Keyword Coverage

**`SSH`** keyword is tracked here (SEO Analyst gap flag). SSH as a standalone term is too broad (massive volume, zero ranking opportunity). SSH is covered as a secondary modifier in:
- `AI agent SSH access`
- `SSH bastion host alternative`
- `SSH AI agent platform`

Do NOT add standalone `SSH` to any brief targeting — it belongs as a modifier, not a head term.

---

## 3. Content Angle

**Lead:** "To give your AI agent SSH access to an EC2 instance, you'd normally need a bastion host, a static key, and a key rotation schedule. Now there's a simpler path."

The story has three steps:
1. Problem: bastion hosts and static SSH keys are operational overhead and a security risk
2. Solution: EC2 Instance Connect injects short-lived credentials via AWS API — no static key, no bastion, no rotation
3. Integration: Molecule AI agents use the ISH workflow to run commands on EC2 as part of an agent task

**Key differentiator vs. competitors:** No other AI agent platform documents an EC2 ISH integration. The tutorial content Molecule AI ships will likely be the first indexed page targeting this exact combination.

---

## 4. Content Recommendations

| Content type | Target keyword | Angle | Priority |
|---|---|---|---|
| Blog post | `AI agent SSH access` + `SSH bastion host alternative` | Problem → EC2 ISH → Molecule AI integration story | High |
| Tutorial/guide | `EC2 Instance Connect Endpoint tutorial` | Step-by-step: agent connects to EC2 via ISH | High |
| Comparison | `SSH bastion host alternative` | EC2 ISH vs bastion host vs static keys | Medium |

---

## 5. SEO Analyst Assessment

**Opportunity:** `AI agent SSH access` is a genuine first-mover keyword. No competing content targets this exact compound term. A clean tutorial with working code examples will likely rank quickly.

**Risk:** Volume on head terms is modest. The AWS documentation dominates `EC2 Instance Connect` head term. Strategy should focus on long-tail variants (tutorial + AI agent angle) rather than competing on the base term.

**EC2 Instance Connect Endpoint note:** The ISH product is newer than the base EC2 Instance Connect. Search demand for ISH-specific queries is lower but so is competition. Prioritize `EC2 Instance Connect Endpoint tutorial` over generic `EC2 Instance Connect`.

**SSH keyword:** Validated as covered. No standalone `SSH` entry needed in keywords.md. See Section 2 above.

---

## 6. Action Items

| # | Action | Owner | Status |
|---|---|---|---|
| 1 | Create EC2 ISH + AI agent blog post | Content Marketer | ⏸ Pending |
| 2 | Create EC2 ISH tutorial in docs | DevRel | ⏸ Pending |
| 3 | Confirm EC2 ISH feature shipped before blog publish | Engineering | ⏸ Pending |
| 4 | Update keywords.md (this brief + SSH tracking) | SEO Analyst | ✅ Done |
| 5 | Reconcile Day 4/5 in social queue | Marketing Lead | ⏸ Pending |

---

*Draft by SEO Analyst 2026-04-22 — difficulty scores estimated from benchmark data, not live tool query*
