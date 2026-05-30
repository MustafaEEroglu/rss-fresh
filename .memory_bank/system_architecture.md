<!-- memory-bank-schema: v1 -->
# System Architecture

## Topology (production)

```
Browser/PWA --HTTPS--> Cloudflare Access --> Tunnel --localhost:8088--> rss-fresh:3000
                                                              |
                                                              +--> pgbouncer:5432 --> central-postgres
OpenClaw --HTTPS--> Cloudflare --> /api/v1/news/summary (Bearer)
```

Network: **`postgres-shared-net`**. Production uses **`pgbouncer:5432`** (not host port 6432).

One Go binary embeds the Svelte SPA (`embed.FS`). One container.

## Process layout
1. `chi` HTTP on `:3000` — REST + SPA.
2. `gocron` — `FETCH_CRON` (default `*/15 * * * *`), staggered RSS fetch.
3. `gocron` — `DIGEST_CRON` (default `0 8 * * *`), Telegram digest (if env set).

## DB schema
`backend/migrations/init.sql`: `categories`, `feeds`, `articles`. Owner must be `rss_user`.
Dedup: `UNIQUE(feed_id, guid)`. Read state: `articles.is_read`.

## API contract

### Auth
- UI routes: Cloudflare Access at edge; app has no in-app login.
- `/api/v1/news/summary`: `Authorization: Bearer $OPENCLAW_GATEWAY_TOKEN`.
- Errors: `{ "error", "code" }`, snake_case JSON, RFC 3339 timestamps.

### Articles (list filters)
`GET /api/v1/articles?category_id&feed_id&unread&read&saved&limit&cursor`

| Query | Effect |
|-------|--------|
| `unread=1` | `is_read = false` |
| `read=1` | `is_read = true` |
| `saved=1` | `is_saved = true` |

`unread` and `read` together → **400**. Filters are mutually exclusive in the UI.

Other endpoints unchanged: categories, feeds, `PATCH /articles/:id`, `POST /articles/mark-read`, `GET /news/summary`, `GET /healthz`.

## Frontend architecture

### State (`app.svelte.ts`)
- `articleFilter`: `'unread' | 'read' | 'saved'` (default `'unread'`).
- Maps to API via `filterToQuery()`.
- `pruneArticlesToFilter()` — on mobile back-to-list, drop rows that no longer match filter (e.g. read items leave Unread list after opening an article).

### UI components
- `ArticleFilterBar.svelte` — segmented Unread / Read / Saved.
- Mobile: filter bar in `ArticleList` header; desktop: same bar in `Sidebar`.
- `ArticleReader` — full-width action buttons on mobile; shared `.btn` classes (44px min).

### Offline (Dexie)
- DB: `rss-fresh-cache` v1.
- `readCachedArticles` supports `unread`, `read`, `saved` filters mirroring API.

## Worker contract
- Batch fetch by oldest `last_fetched_at`; conditional GET; `error_count >= 10` deactivates feed.
- Critical categories → Telegram push (throttled).
