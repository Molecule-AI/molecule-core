---
name: Email AI Handle Service — Project Context
description: Railway-hosted email service that auto-replies and tracks leads to Google Sheets
type: project
---

## What It Does
- Receives emails via Gmail API polling
- Classifies emails and generates LLM-powered acknowledgment replies
- Extracts lead info (name, phone, city, property type, renovation type)
- Writes leads to Google Sheet: "邮件客人" tab in RS 2021 销售表
- Sheet layout is transposed: headers in column A, each lead is a new column

## Infrastructure
- Hosted on Railway (project: considerate-enchantment)
- Services: Node.js app + Postgres + Redis (BullMQ)
- Repo: Reno-Stars/reno-star-email-ai-handle-service
- Local: ~/.openclaw/workspace/reno-star-email-ai-handle-service

## Google Sheets Integration
- Service account: reno-sheets-writer@${GCP_PROJECT_ID}.iam.gserviceaccount.com
- Spreadsheet ID: 19votqeqJ1lO2pZ3eXzRMm7e-YKKGxv3BvCc0dVJivY4
- SA key base64-encoded in Railway env var GOOGLE_SHEETS_SA_KEY

## Recent Updates (2026-04-08)
- backfill-lead endpoint now runs full needs-reply pipeline with email forwarding
- Uncertain email classification default changed from info-only → needs-reply (fewer missed leads)

**Why:** This is a critical business service — email leads from the website flow through here. Downtime = missed leads.

**How to apply:** Be careful with changes to this service. Test thoroughly before pushing. Railway CLI auth doesn't work — use Playwright+Chrome for Railway operations.
