# SaaS Workspaces Now Support Full File API — SSH-Backed Writes Land Today

**Status:** Live — merged 2026-04-23
**PR:** [#1702](https://github.com/Molecule-AI/molecule-core/pull/1702)

---

One gap was blocking SaaS customers from doing something fundamental: writing files programmatically.

When you called `PUT /workspaces/:id/files/config.yaml` from a SaaS (EC2-backed) workspace, you got a 500. `failed to write file: docker not available`. The file API existed, but only for self-hosted Docker deployments. SaaS workspaces — the ones running on real EC2 VMs — had no path to write.

That changes today.

## What Was Wrong

Molecule AI supports two workspace compute models: self-hosted (Docker containers) and SaaS (EC2 VMs). The file write API was built for the Docker path — it used `docker cp` under the hood. SaaS workspaces don't have Docker. There was no fallback, so every API write failed silently.

This wasn't a permissions issue or a timeout. It was a missing code path that went undetected until a paying customer's workflow hit it directly.

## What's Fixed

The file write API now detects which compute model is in use and routes accordingly:

- **Self-hosted (Docker):** Unchanged — `docker cp` path still used
- **SaaS (EC2):** Routes through EC2 Instance Connect (EIC) — the same ephemeral-keypair SSH flow that powers the Terminal tab in the Canvas

The remote write uses `install -m 0644 /dev/stdin <path>` for an atomic write that creates missing parent directories. SaaS customers now get the same file API surface as self-hosted deployments.

## Why It Matters

Your file API workflow shouldn't break depending on where Molecule AI runs. Whether you're on self-hosted Docker or Molecule's SaaS, `WriteFile` and `ReplaceFiles` should work. They do now.

**Try it:**
```bash
curl -X PUT https://your-workspace.moleculesai.app/workspaces/:id/files/config.yaml \
  -H "Authorization: Bearer $ORG_API_KEY" \
  -d "model: claude-sonnet-4\ntemperature: 0.7"
```

File API. Now everywhere Molecule AI runs.

---

*Found a bug or have a feature request? Open an issue at [github.com/Molecule-AI/molecule-core](https://github.com/Molecule-AI/molecule-core).*
