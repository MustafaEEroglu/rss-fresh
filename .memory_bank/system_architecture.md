<!-- memory-bank-schema: v1 -->
# System Architecture

## Topology

```
Browser/PWA --HTTPS--> Cloudflare Access --> Tunnel --localhost:8088--> rss-fresh:3000
                                                              |
                                                              +--> central-pgbouncer:6432 --> central-postgres
                                                              +--> api.telegram.org
```

Network: **`central-postgres-net`**. One Go binary embeds the Svelte SPA (`embed.FS`).

## Notification flow

| Path | Trigger | Channel |
|------|---------|---------|
| Critical push | New articles + `category.is_critical` | Telegram `NotifyCritical` |
| Daily digest | `DIGEST_CRON` (default `0 8 * * *`) | Telegram `SendDigest` — unread counts + saved (24h) |

Both require `TELEGRAM_BOT_TOKEN` + `TELEGRAM_CHAT_ID`. Throttled send queue in `internal/telegram`.

## RSS ingest

Cron `FETCH_CRON` (default `*/15 * * * *`) picks oldest `last_fetched_at` feeds in batches.
On parse, skip items where `published_at < feed.created_at` (nil date allowed). Dedup `(feed_id, guid)`.

## Process layout (single binary)

1. `chi` HTTP `:3000` — REST + embedded SPA
2. `gocron` — RSS fetch
3. `gocron` — Telegram digest (if bot env set)
4. `gocron` — article retention (`RETENTION_DAYS > 0`)

## Retention

Nightly `article-retention` in `internal/retention`: delete rows where `is_read AND NOT is_saved`
and `COALESCE(published_at, fetched_at) < now - RETENTION_DAYS`. Default 30. `RETENTION_DAYS=0` disables.
Dexie cache not purged server-side.

## DB schema

`categories` (incl. `is_critical`), `feeds` (incl. `created_at`), `articles` (incl. `is_read`, `is_saved`).
Dedup: `UNIQUE(feed_id, guid)`.

## API (`/api/v1`, JSON, snake_case)

| Endpoint | Notes |
|----------|-------|
| `GET /healthz` | Public health |
| CRUD `/categories`, `/feeds` | CF Access |
| `POST /feeds/:id/refresh` | Async re-fetch |
| `GET /articles?unread\|read\|saved&limit&cursor` | Mutually exclusive unread/read |
| `PATCH /articles/:id` | Toggle read/saved |
| `POST /articles/mark-read` | Bulk mark read |

No bearer-token endpoints. All UI routes gated by Cloudflare Access.

## Frontend highlights

- `articleFilter`: `'unread' \| 'read' \| 'saved'` — `ArticleFilterBar.svelte`
- Mobile: `mobilePane` sidebar \| list \| detail (no auto-detail `$effect`)
- Offline: Dexie `rss-fresh-cache` v1

## Management plane (host — do not edit from app repo)

`~/projects/management/` — Portainer (`:9000`), Uptime Kuma (`:3001`), Watchtower (label-only pulls).
Watchtower: `WATCHTOWER_LABEL_ENABLE=true`, GHCR creds via `~/.docker/config.json`.

## Deploy flow

`git push main` → shared-workflows CI → GHCR `:latest` → Watchtower restarts labeled `rss-fresh`.
Verify: `./scripts/verify-deploy.sh` against `http://127.0.0.1:8088`.

## Repo layout

```
backend/   Go API + worker + embed
frontend/  Svelte 5 PWA
scripts/   verify-deploy.sh, seed.sh, soak-watch.sh
docker-compose.yml
INFRA_HANDOFF.md
```
