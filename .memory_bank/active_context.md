<!-- memory-bank-schema: v1 -->
# Active Context

_Last updated: 2026-05-31._

## Status

**LIVE** — FeedManager Add-button fix shipped (`2fce616`); deploy to VPS via
`docker compose pull/up` (Watchtower still inactive).

## Currently working on

Product backlog — notification + ingestion redesign (see TODO below).

## TODO (priority order)

1. ~~**Feed ingest cutoff**~~ — **Done.** `fetchOne` skips items with
   `published_at < feed.created_at`; nil-date items pass through.
2. ~~**Critical push → OpenClaw**~~ — **Done.** `internal/openclaw.Notifier` POSTs
   to `OPENCLAW_WEBHOOK_URL`; fetcher wired to OpenClaw, Telegram kept for digest only.
3. ~~**Saved → messaging**~~ — **Done.** `/news/summary?saved=1` filter added;
   Telegram digest includes saved articles (last 24h) via `SavedArticlesSince`.

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
