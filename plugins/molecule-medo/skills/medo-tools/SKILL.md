---
name: MeDo Tools
description: >
  Create, update, and publish applications on Baidu MeDo (摩搭), a no-code AI
  application builder. Used in the Molecule AI hackathon integration (May 2026).
tags: [hackathon, baidu, medo, china, no-code]
examples:
  - "Create a chatbot app on MeDo called 'Customer Support'"
  - "Update the content of my MeDo app abc123"
  - "Publish my MeDo app to production"
---

# MeDo Tools

Provides three tools for interacting with the Baidu MeDo no-code platform:

- **create_medo_app** — Scaffold a new application from a template (blank, chatbot, form, dashboard).
- **update_medo_app** — Push content or configuration changes to an existing application.
- **publish_medo_app** — Publish a draft application to production or staging.

## Setup

Set `MEDO_API_KEY` as a workspace secret. Optionally override the base URL via `MEDO_BASE_URL`
(default: `https://api.moda.baidu.com/v1`).

When `MEDO_API_KEY` is absent the tools run in mock mode and return stub responses — safe for
local development and testing.
