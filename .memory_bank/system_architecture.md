# System Architecture — RSS-Fresh

## Topology

```
Browser/PWA --HTTPS--> Cloudflare Tunnel --localhost:8088--> rss-fresh:3000 (Go)
                                                              |
                                                              +--> central-pgbouncer:6432 --> central-postgres
                                                              +--> api.telegram.org
OpenClaw OS --HTTPS--Bearer--> Cloudflare Tunnel --> rss-fresh:3000 /api/v1/news/summary
```

The Go binary embeds the built Svelte SPA via `embed.FS`. One process. One container.

## Process layout (one binary, three concurrent goroutines)
1. `chi` HTTP server on `:3000` — REST + SPA static.
2. `gocron` worker — `FETCH_CRON` (default `*/15 * * * *`), staggered batch fetch.
3. `gocron` worker — `DIGEST_CRON` (default `0 8 * * *`), Telegram daily digest.
   Critical-category Telegram pushes happen inline at the end of fetch transactions.

## DB schema (rss_fresh database)
See `backend/migrations/init.sql`. Three tables: `categories`, `feeds`, `articles`.
Key indexes: `idx_feeds_active(is_active, last_fetched_at)` (worker uses this to pick
the oldest active feeds), `idx_articles_unread`, `idx_articles_saved`,
`UNIQUE(feed_id, guid)` for dedup.

## API contract — frozen here, both BE and FE conform

### Auth
- Browser/PWA: gated by Cloudflare Access at the edge. App trusts all reaching it.
- OpenClaw endpoint: `Authorization: Bearer $OPENCLAW_GATEWAY_TOKEN`, constant-time
  comparison; missing/wrong → 401, no body.
- All `/api/v1/*` responses: `Content-Type: application/json`, error envelope
  `{ "error": "<message>", "code": "<machine_code>" }`.

### Endpoints

`GET /api/v1/healthz`
→ 200 `{ "status": "ok", "version": "<sha>", "uptime_seconds": <int> }` (no auth)

`GET /api/v1/categories`
→ 200 `{ "items": [{ "id": int, "name": string, "slug": string, "is_critical": bool, "feed_count": int, "unread_count": int }] }`

`POST /api/v1/categories`  body `{ "name": string, "slug"?: string, "is_critical"?: bool }`
→ 201 same item shape (without counts), 409 on slug conflict.

`PATCH /api/v1/categories/:id`  body any of `{ name?, slug?, is_critical? }`
→ 200 item.

`DELETE /api/v1/categories/:id` → 204 (cascades feeds + articles).

`GET /api/v1/feeds[?category_id=int]`
→ 200 `{ "items": [{ "id": int, "category_id": int, "name": string, "url": string, "last_fetched_at": rfc3339|null, "error_count": int, "is_active": bool }] }`

`POST /api/v1/feeds`  body `{ "category_id": int, "name"?: string, "url": string }`
→ 201 item; if `name` omitted, derived from feed `<title>` on first successful fetch.

`PATCH /api/v1/feeds/:id`  body any of `{ category_id?, name?, url?, is_active? }` → 200.

`DELETE /api/v1/feeds/:id` → 204 (cascades articles).

`POST /api/v1/feeds/:id/refresh` → 202; triggers an immediate single-feed fetch.

`GET /api/v1/articles?category_id=&feed_id=&unread=&saved=&limit=&cursor=`
- `unread=1` filters `is_read=false`. `saved=1` filters `is_saved=true`.
- `cursor` is opaque, returned in prior response; pagination by `(published_at desc, id desc)`.
- `limit` default 50, max 200.
→ 200 `{ "items": [Article], "next_cursor": string|null }`
  `Article = { id, feed_id, feed_name, category_id, category_slug, title, url, author?, summary?, content?, published_at, is_read, is_saved }`.

`PATCH /api/v1/articles/:id`  body any of `{ is_read?, is_saved? }` → 200 Article.

`POST /api/v1/articles/mark-read`  body `{ ids: [int, ...] }` → 200 `{ updated: int }` (bulk).

### OpenClaw endpoint (token-gated)
`GET /api/v1/news/summary?since=<rfc3339>&category=<slug>&limit=<int>`
- Default `since` = now-24h. Default limit 100, max 500.
- 401 if `Authorization` missing/wrong.
→ 200 `{ "generated_at": rfc3339, "since": rfc3339, "items": [{ "title", "url", "summary", "category_slug", "feed_name", "published_at" }] }`

## Frontend ↔ Backend contract pinning
- Field names are **snake_case** in JSON across the wire.
- Timestamps are RFC 3339 UTC strings.
- IDs are JSON numbers (BIGSERIAL fits safely under 2^53 for any conceivable feed
  count).
- The PWA mirrors this exactly in Dexie; no field renaming on either side.

## Worker contract
- Tick: `FETCH_CRON` (default `*/15 * * * *`).
- Per tick: `SELECT id FROM feeds WHERE is_active ORDER BY last_fetched_at NULLS FIRST LIMIT FETCH_BATCH_SIZE`.
- Fetch with `If-None-Match` (`etag`) and `If-Modified-Since` (`last_modified`).
  - `304` → bump `last_fetched_at`, no parse, no rows.
  - `2xx` → parse, upsert articles by `(feed_id, guid)` (dedup), reset `error_count=0`,
    update `etag`/`last_modified`, bump `last_fetched_at`.
  - any error → `error_count += 1`; if `error_count >= 10` → `is_active = false`.
- Per fetch HTTP timeout: `FETCH_TIMEOUT_SECONDS` (default 20s).
- After successful fetch, if any inserted articles' category has `is_critical=true`,
  push them (max 5 per message, throttled to 1 message / 5s) to Telegram.

## Digest contract
- `DIGEST_CRON` (default `0 8 * * *`, TZ from container `TZ` env).
- Emits one Telegram message:
  `"📰 RSS-Fresh digest <date>: <category_name> — <unread_count> new"` lines.
- No-op if every category has 0 unread.

## Service worker / Dexie contract (frontend ↔ frontend)
- Dexie DB: `rss-fresh-cache`, version 1.
  - Table `articles` keyed by `id`, indexed on `[category_id+published_at]`, `is_read`, `is_saved`.
  - Table `feeds` keyed by `id`, indexed on `category_id`.
  - Table `categories` keyed by `id`.
  - Table `outbox` for queued mutations (Background Sync fallback).
- SW caches:
  - `app-shell` — precache from Vite manifest.
  - `images` — runtime cache, max 100 entries / 30 days.
  - `api-get` — NetworkFirst, 3s timeout, also writes through to Dexie.
  - mutations — NetworkOnly + Workbox `BackgroundSyncPlugin('rss-fresh-mutations')`.
