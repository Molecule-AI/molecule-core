# Sourcing Gap: "Sub-100ms clone times from anywhere on Cloudflare's edge"

**Filed:** 2026-04-22
**Filed by:** SEO Analyst
**Ruling:** PM (2026-04-22) — Do NOT soften. Flag as unsubstantiated in all published copy. Document as sourcing gap.

---

## Claim

> "Sub-100ms clone times from anywhere on Cloudflare's edge"
> Variant: "Fast edge-based clone times from anywhere on Cloudflare's global network"

## Files where claim appears

| File | Line | Context | Status |
|---|---|---|---|
| `docs/marketing/campaigns/cloudflare-artifacts/social-copy.md` | 25 | X/LinkedIn Post 2 — "fast edge-based clone times from anywhere on Cloudflare's global network" | ⚠️ Flagged as unsubstantiated |
| `docs/marketing/campaigns/cloudflare-artifacts/social-copy.md` | 77 | LinkedIn post — "sub-100ms clone times from anywhere" | ⚠️ Flagged as unsubstantiated |
| `docs/marketing/plans/phase-30-launch-plan.md` | 32 | Launch plan — "sub-100ms softened to fast edge-based clone times" | ⚠️ Outdated — PM ruling supersedes PMM softening |

## PM Ruling (2026-04-22)

- Do **NOT** soften the claim
- Do **NOT** remove the claim
- Flag it as **unsubstantiated** in all published copy
- Document it as a sourcing gap
- Sourcing required before reuse

## What to do

### In draft copy (before publish)
Keep the claim but add an inline flag:
```
> ⚠️ [SOURCING GAP — UNSUBSTANTIATED] The sub-100ms claim is not currently
> sourced to a Cloudflare benchmark or official spec. Flag in all published
> copy. Sourcing required before reuse.
```

### In published copy
- If already published with the claim, do not edit it out
- Add a correction note or editor's note flagging it as unsubstantiated
- Do not repeat the claim in new copy without sourcing

### For sourcing
Required to close this gap:
- [ ] Cloudflare official documentation or benchmark citing sub-100ms global clone times
- [ ] Cloudflare Artifacts SLA or performance spec
- [ ] Third-party benchmark (e.g., independent latency tests across edge regions)
- [ ] Internal performance test results with methodology documented

## Background

The claim appears in Cloudflare Artifacts campaign copy describing the performance benefit of Cloudflare's global edge network for git clone operations. The implied claim is that Cloudflare's edge gives AI agents near-instant access to git repos regardless of geographic location.

This is a plausible architectural claim (Cloudflare does have global edge PoPs) but **no specific benchmark, SLA, or official Cloudflare documentation has been cited to substantiate the specific "sub-100ms" figure**.

## Related

- Cloudflare Artifacts campaign: `docs/marketing/campaigns/cloudflare-artifacts/social-copy.md`
- Phase 30 launch plan (outdated softening note): `docs/marketing/plans/phase-30-launch-plan.md`

---

*SEO Analyst — sourcing gap log. Update this file when sourcing is secured.*
