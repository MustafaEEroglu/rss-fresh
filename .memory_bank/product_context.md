<!-- memory-bank-schema: v1 -->
# Product Context

## What
Personal RSS / news manager for a single operator on a 4 GB Hetzner VPS, backed by
shared `central-postgres` through PgBouncer. FreshRSS alternative without duplicate DB stacks.

## Why
- One container, one Go binary, one connection string to the shared cluster.
- **Telegram** for operator alerts: immediate push on `is_critical` categories, daily
  digest with unread counts and saved-article highlights (24h window).

## Hard constraints
1. `mem_limit: 256m`; steady-state RAM well under 100 MB.
2. No dedicated Postgres container.
3. Bind `127.0.0.1:8088`; public access via Cloudflare Tunnel + Access only.
4. Staggered cron fetch (`FETCH_CRON`), not continuous polling.
5. Single-user — Cloudflare Access, not in-app accounts.
6. iOS PWA first-class: 44px touch targets, safe-area insets, visible button chrome.

## Product rules (all shipped)

| Rule | Behavior |
|------|----------|
| Feed ingest cutoff | Skip RSS items with `published_at < feed.created_at` |
| Critical → Telegram | `is_critical` category → `NotifyCritical` on new articles |
| Saved → digest | Daily digest lists saved articles from last 24h |
| Retention | Delete read, non-saved articles after 30 days |

## Success criteria (all met)

Operator reads via tunnel + Access · stable worker/API · Unread/Read/Saved filters ·
iOS PWA UX · FeedManager CRUD · feed add-date cutoff · Telegram push + digest ·
Watchtower auto-redeploy · 30-day retention cron · VPS env configured (no OpenClaw).

## Out of scope
- Multi-user / in-app login
- Full-text search beyond filters
- Article scraping beyond RSS payloads
- Native mobile apps (PWA only)
- External AI integrations (OpenClaw removed `7adc729`)
