<!-- memory-bank-schema: v1 -->
# Active Context

_Last updated: 2026-05-31._

## Status

**LIVE** — FeedManager Add-button fix shipped (`2fce616`); deploy to VPS via
`docker compose pull/up` (Watchtower still inactive).

## Currently working on

Product backlog — notification + ingestion redesign (see TODO below).

## TODO (priority order)

1. **Feed ingest cutoff** — when a new RSS feed is added, import articles from
   **feed `created_at` forward only**; do not backfill the full RSS archive.
2. **Reassign critical push to OpenClaw** — stop using `is_critical` → Telegram
   immediate push; wire that real-time new-article path to **OpenClaw OS** instead.
3. **Saved articles → messaging** — articles the operator marks **saved**
   (`is_saved = TRUE`) become the input set for the messaging tool (Telegram digest
   and/or OpenClaw summary — TBD per channel).

## Immediate next steps

1. Deploy `2fce616` to VPS after CI image is ready.
2. Implement TODO #1 (feed add date cutoff in fetcher / insert filter).
3. Design OpenClaw push contract (webhook vs poll extension of `/news/summary`).

## Blockers

- Watchtower not running → no automatic image pull.
- OpenClaw Access bypass for machine clients not verified.

## Hand-off

- Ops: [`INFRA_HANDOFF.md`](../INFRA_HANDOFF.md)
- Retention rules: [system_architecture.md](system_architecture.md) § Retention
