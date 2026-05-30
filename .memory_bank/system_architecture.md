<!-- memory-bank-schema: v1 -->
# System Architecture

## Topology (production)

```
Browser/PWA --HTTPS--> Cloudflare Access --> Tunnel --localhost:8088--> rss-fresh:3000
                                                              |
                                                              +--> pgbouncer:5432 --> central-postgres
OpenClaw --HTTPS--> Cloudflare --> /api/v1/news/summary (Bearer)
```

Network: **`postgres-shared-net`**. In-network pooler: **`pgbouncer:5432`** (not host `:6432`).

One Go binary embeds the Svelte SPA (`embed.FS`). One app container.

## Notification flow

### Current (as shipped)

| Path | Trigger | Channel |
|------|---------|---------|
| Critical push | New articles + `category.is_critical` | Telegram (`NotifyCritical`) |
| Daily digest | `DIGEST_CRON`; unread counts per category | Telegram (`SendDigest`) |
| Summary poll | OpenClaw calls `GET /news/summary?since&category&limit` | OpenClaw (Bearer) |

### Planned (TODO backlog)

| Path | Trigger | Channel |
|------|---------|---------|
| Real-time push | New articles + `category.is_critical` | **OpenClaw** (replace Telegram critical) |
| Curated messaging | Articles with `is_saved = TRUE` | Telegram digest and/or OpenClaw summary |
| Ingest cutoff | New feed first fetch | Skip RSS items with `published_at < feed.created_at` |

## Management plane (host — off-limits for app edits)

Lives under **`~/projects/management/`** on VPS user `godeleck`. **Planned (Option A):**
single `docker-compose.yml` with:

| Service | Container | Host bind |
|---------|-----------|-----------|
| Portainer CE | `portainer` | `127.0.0.1:9000` |
| Uptime Kuma | `uptime-kuma` | `127.0.0.1:3001` |
| Watchtower | `watchtower` | none (docker.sock only) |

Watchtower env (planned): `WATCHTOWER_LABEL_ENABLE=true`, `WATCHTOWER_POLL_INTERVAL=300`,
`WATCHTOWER_CLEANUP=true`, mount `/home/godeleck/.docker/config.json` for GHCR pulls.

**As of 2026-05-30:** Portainer and Uptime Kuma run; **Watchtower container does not exist**
(unused `containrrr/watchtower:latest` image only). Auto-redeploy for `rss-fresh` is inactive.

Portainer data: `~/projects/management/portainer/portainer_data/`.

## App folder layout (repo)

```
backend/     Go API + worker + embedded SPA
frontend/    Svelte 5 PWA
.memory_bank/
docker-compose.yml   # app only; watchtower label, not watchtower service
```

## Process layout (rss-fresh binary)
1. `chi` HTTP on `:3000` — REST + SPA.
2. `gocron` — RSS fetch (`FETCH_CRON`, default `*/15 * * * *`).
3. `gocron` — Telegram digest if env set (`DIGEST_CRON`).
4. `gocron` — article retention if `RETENTION_DAYS > 0` (`RETENTION_CRON`, default `0 4 * * *`).

## Retention (articles)

Nightly job `article-retention` in `internal/retention`:

| Rule | Behavior |
|------|----------|
| Age threshold | `RETENTION_DAYS` (default **30**) |
| Eligible rows | `is_read = TRUE` AND `is_saved = FALSE` |
| Age source | `COALESCE(published_at, fetched_at) < now - N days` |
| Protected | Unread articles; saved (`is_saved`) articles |
| Disabled | `RETENTION_DAYS=0` skips cron registration |

Feed/category rows are never deleted by retention — only article bodies in PostgreSQL.
PWA Dexie cache is not purged server-side; may show stale rows until refresh.

## DB schema
`categories`, `feeds`, `articles` — owner `rss_user`. Dedup `UNIQUE(feed_id, guid)`.
Feeds have `created_at` — planned cutoff anchor for first-ingest filter.

## API contract

### Articles list
`GET /api/v1/articles?category_id&feed_id&unread&read&saved&limit&cursor`

| Query | Effect |
|-------|--------|
| `unread=1` | `is_read = false` |
| `read=1` | `is_read = true` |
| `saved=1` | `is_saved = true` |

`unread` + `read` → **400**. UI uses mutually exclusive `articleFilter` modes.

### Other
Categories, feeds, `PATCH /articles/:id`, `POST /articles/mark-read`, `GET /news/summary` (Bearer), `GET /healthz`.

**Planned:** `/news/summary` may gain `saved=1` filter; OpenClaw may receive push/webhook on critical new articles.

## Frontend

### State (`app.svelte.ts`)
- `articleFilter`: `'unread' | 'read' | 'saved'`.
- `refreshing`, `refreshNotice`, `lastRefreshedAt` on manual `refreshAll()`.
- `pruneArticlesToFilter()` when mobile back-to-list from detail.

### UI
- `ArticleFilterBar.svelte` — Unread / Read / Saved (mobile: list header; desktop: sidebar).
- `FeedManager.svelte` — category/feed CRUD; Add buttons use `type="submit"` (`2fce616`).
- Mobile nav: `mobilePane` sidebar | list | detail — **no** `$effect` forcing detail when article selected (fixed `65ca785`).

### Offline
Dexie `rss-fresh-cache` v1; filters mirror API.

## Deploy flow (intended vs actual)

**Intended:** push `main` → CI → GHCR `:latest` → Watchtower pulls labeled containers.

**Actual (2026-05-30):** CI + GHCR work; operator must `docker compose pull/up` until Watchtower runs.
