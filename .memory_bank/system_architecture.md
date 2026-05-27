# System Architecture — RSS-Fresh

## Topology (production — Hetzner)

```
Browser/PWA --HTTPS--> Cloudflare Access --> Cloudflare Tunnel --localhost:8088--> rss-fresh:3000
                                                                                          |
                                                                                          +--> pgbouncer:5432 --> central-postgres
                                                                                          +--> api.telegram.org (optional)
OpenClaw OS --HTTPS--> Cloudflare (Access and/or Service Auth) --> /api/v1/news/summary (Bearer token)
```

Docker network: **`postgres-shared-net`** (external). Peers include `pgbouncer`,
`central-postgres`, `postgres-adminer`, `rss-fresh`.

> **Note:** Early docs used `central-pgbouncer:6432` and `central-postgres-net`.
> Production uses **`pgbouncer:5432`** and **`postgres-shared-net`**. See
> [active_context.md](active_context.md).

The Go binary embeds the built Svelte SPA via `embed.FS`. One process. One container.

## Process layout (one binary, three concurrent goroutines)
1. `chi` HTTP server on `:3000` — REST + SPA static.
2. `gocron` worker — `FETCH_CRON` (default `*/15 * * * *`), staggered batch fetch.
3. `gocron` worker — `DIGEST_CRON` (default `0 8 * * *`), Telegram daily digest.
   Critical-category Telegram pushes happen inline at the end of fetch transactions.

## DB schema (rss_fresh database)
See `backend/migrations/init.sql`. Three tables: `categories`, `feeds`, `articles`.
**Owner must be `rss_user`** if migrations were applied manually in Adminer.

Key indexes: `idx_feeds_active`, `idx_articles_unread`, `idx_articles_saved`,
`UNIQUE(feed_id, guid)` for dedup.

## API contract — frozen here, both BE and FE conform

### Auth
- Browser/PWA: **Cloudflare Access** at the edge (Allow policy — operator email only).
  App trusts all requests that reach it; no in-app login.
- OpenClaw endpoint: `Authorization: Bearer $OPENCLAW_GATEWAY_TOKEN`, constant-time
  comparison; missing/wrong → 401.
- All `/api/v1/*` responses: `Content-Type: application/json`, error envelope
  `{ "error": "<message>", "code": "<machine_code>" }`.

### Endpoints

`GET /api/v1/healthz`
→ 200 `{ "status": "ok", "version": "<sha>", "uptime_seconds": <int> }` (no in-app auth; may still sit behind Access on public hostname)

`GET /api/v1/categories`
→ 200 `{ "items": [{ "id", "name", "slug", "is_critical", "feed_count", "unread_count" }] }`

`POST /api/v1/categories`  body `{ "name", "slug"?, "is_critical"? }` → 201 / 409

`PATCH /api/v1/categories/:id` → 200

`DELETE /api/v1/categories/:id` → 204

`GET /api/v1/feeds[?category_id]` → 200 items list

`POST /api/v1/feeds`  body `{ category_id, name?, url }` → 201

`PATCH /api/v1/feeds/:id` → 200

`DELETE /api/v1/feeds/:id` → 204

`POST /api/v1/feeds/:id/refresh` → 202

`GET /api/v1/articles?category_id&feed_id&unread&saved&limit&cursor` → 200 + `next_cursor`

`PATCH /api/v1/articles/:id`  body `{ is_read?, is_saved? }` → 200

`POST /api/v1/articles/mark-read`  body `{ ids: [...] }` → 200 `{ updated }`

`GET /api/v1/news/summary?since&category&limit` — **Bearer token required** → 200 summary JSON

## Frontend ↔ Backend contract pinning
- Field names are **snake_case** in JSON.
- Timestamps are RFC 3339 UTC strings.
- PWA mirrors server shapes in Dexie (`rss-fresh-cache`).

## Worker contract
- Tick: `FETCH_CRON` (default `*/15 * * * *`).
- Batch: oldest `last_fetched_at` among active feeds, size `FETCH_BATCH_SIZE`.
- Conditional GET via `etag` / `last_modified`; dedup `(feed_id, guid)`.
- `error_count >= 10` → `is_active = false`.
- Critical categories → Telegram push (throttled); daily digest at `DIGEST_CRON`.

## Service worker / Dexie contract
- Dexie DB: `rss-fresh-cache`, version 1.
- SW: precache app shell; NetworkFirst API GET (3s); Background Sync for mutations; SWR for images.
