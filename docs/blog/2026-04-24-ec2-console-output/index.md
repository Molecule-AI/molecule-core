---
title: "Diagnose Failed Workspaces Instantly: EC2 Console Output in Molecule AI Canvas"
date: 2026-04-24
slug: ec2-console-output
description: "When a Molecule AI workspace fails to boot on EC2, the reason is locked in the instance's console output — but only if you know where to look. Molecule AI now surfaces that output directly in the Canvas UI, with context about what changed since the last successful boot."
og_image: /docs/assets/blog/2026-04-24-ec2-console-output-og.png
tags: [EC2, debugging, self-hosted, Canvas, workspace, enterprise]
keywords: [EC2 Console Output, Molecule AI, AI agents]
---

<script type="application/ld+json">
{
  "@context": "https://schema.org",
  "@type": "Article",
  "headline": "Diagnose Failed Workspaces Instantly: EC2 Console Output in Molecule AI Canvas",
  "description": "When a Molecule AI workspace fails to boot on EC2, the reason is locked in the instance's console output — but only if you know where to look. Molecule AI now surfaces that output directly in the Canv",
  "author": { "@type": "Organization", "name": "Molecule AI" },
  "datePublished": "2026-04-24",
  "publisher": {
    "@type": "Organization",
    "name": "Molecule AI",
    "logo": { "@type": "ImageObject", "url": "https://molecule.ai/logo.png" }
  }
}
</script>

# Diagnose Failed Workspaces Instantly: EC2 Console Output in Molecule AI Canvas

When a workspace fails to start, the answer is usually already there — buried in the EC2 instance's console output. Cloud-init logs, boot script errors, missing environment variables, package installation failures: all of it gets written to the instance's serial console before the system crashes or becomes unreachable.

The problem is access. Getting to that output means opening the AWS Console, navigating to EC2, finding the right instance, and retrieving the console output manually. For teams running Molecule AI as a self-hosted deployment, that's a barrier between "something broke" and "I know what broke."

Molecule AI now fetches and displays EC2 console output directly in the Canvas UI — on the workspace detail view, whenever the instance fails to come up.

## The Old Debugging Flow

Before this change, a platform admin investigating a failed workspace would go through something like:

1. Notice the workspace is unreachable in Canvas
2. Log into the AWS Console (or use the CLI)
3. Find the EC2 instance associated with the workspace
4. Run `aws ec2 get-console-output --instance-id <id>`
5. Parse the raw text for the relevant error
6. Cross-reference with the boot script or environment config

Steps 2–5 require AWS Console access, AWS CLI credentials, and knowledge of which EC2 instance maps to which workspace. For a first-line operator — a team member who didn't provision the infrastructure — that path is often a dead end.

## What's Changed

When a workspace on an EC2-backed tenant fails its boot sequence, the Canvas now shows the EC2 console output inline on the workspace detail view. You see the same text you'd get from `aws ec2 get-console-output`, rendered in context alongside the workspace's current state.

The output includes:

- **Cloud-init logs** — what the instance tried to do on boot, including any errors from `set -x` commands
- **Boot script stderr** — unhandled errors from the workspace startup script
- **Kernel and systemd output** — low-level events that precede a crash or hang
- **Last successful vs. failed boot diff** — Canvas surfaces which boot event changed between the last healthy state and the current failure

You don't need to leave Canvas, switch accounts, or know the instance ID. The context stays in the workspace record where your team already works.

## How It Works

The Canvas fetches console output via a new endpoint on the platform API, gated to tenant admins:

```bash
GET /cp/workspaces/:workspaceId/ec2-console-output
Authorization: Bearer <tenant-admin-token>
```

The response is the raw EC2 `get-console-output` text, proxied through the platform's AWS credentials (the same IAM role used to provision the instance). The credentials are never exposed to the browser or the workspace itself.

On the platform side, the handler calls `ec2:GetConsoleOutput` using the instance ID stored in the workspace record. The output is returned as-is — no filtering or redaction — so platform operators see exactly what AWS recorded.

## What to Look For in the Output

EC2 console output is raw and unstructured. Here's what to scan for when debugging a failed workspace:

**`Cloud-init` errors** — Look for lines starting with `ci-errors` or `meta-data`. Cloud-init failures usually mean the instance couldn't fetch its user-data or metadata service.

**`set -x` output** (self-hosted deployments) — If the boot script uses `set -x`, every command and its arguments appear in the console output. This is useful but can include secret values — Molecule AI redacts environment variable names matching common secret patterns before displaying output in Canvas.

**`apt-get` or `yum` failures** — Package installation errors (network timeouts, repository404s) often appear here. Common in air-gapped or VPN-constrained environments.

**`docker pull` failures** — If the workspace runtime is pulled from a private registry, auth failures and image-not-found errors surface in console output before the instance becomes unreachable.

**OOM or resource exhaustion** — `dmesg` lines at the end of the output often show `Out of memory` or `killed` events that explain a sudden crash.

## Security Considerations

EC2 console output can contain sensitive information depending on how the instance is configured:

- **User-data scripts** may include tokens or credentials passed at provision time
- **`set -x` boot scripts** echo command arguments, including values of exported variables
- **Cloud-init config** may contain inline secrets

Molecule AI applies lightweight redaction before displaying console output in Canvas: environment variable values matching known secret patterns (API keys, bearer tokens, connection strings) are replaced with `[REDACTED]`. The raw output is still available to tenant administrators via the API for cases where redaction obscures needed context.

Access to `/cp/workspaces/:workspaceId/ec2-console-output` requires an admin-scoped token — regular workspace users cannot retrieve console output for instances they don't own.

## Getting Started

If you're running Molecule AI on EC2 (self-hosted or private-cloud deployment), console output visibility is enabled automatically. When a workspace fails to boot:

1. Open Canvas and navigate to the failed workspace
2. The EC2 console output appears in the workspace detail panel
3. If the output is truncated, use the API directly with your tenant admin credentials:

```bash
curl https://<your-platform>/cp/workspaces/<workspace-id>/ec2-console-output \
  -H "Authorization: Bearer <admin-token>"
```

For platform operators: ensure your tenant's IAM role includes `ec2:GetConsoleOutput`. This was added to the default provisioner IAM policy in PR #1178. If you provision EC2 instances manually, add the following to your IAM policy:

```json
{
  "Effect": "Allow",
  "Action": "ec2:GetConsoleOutput",
  "Resource": "arn:aws:ec2:*:instance/*"
}
```

---

*Molecule AI is open source. EC2 console output in Canvas shipped in PR [#1178](https://github.com/moleculeai/molecule/pull/1178).*
