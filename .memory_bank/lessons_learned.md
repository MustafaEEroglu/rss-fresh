<!-- memory-bank-schema: v1 -->
# Lessons Learned

## 2026-05-31 — OpenClaw removed

- **OpenClaw was out of scope** — removed `/api/v1/news/summary`, bearer middleware,
  `OPENCLAW_*` env vars, and `internal/openclaw`. Critical push stays on Telegram.
- **`DATABASE_URL` is the only required env** at startup; Telegram remains optional.

## 2026-05-31 — Feed ingest cutoff

- **Skip RSS items with `published_at < feed.created_at`** on first fetch and ongoing
  ticks — prevents backfilling full archives when adding a feed. Nil publish date passes through.

## 2026-05-31 — FeedManager Add buttons

- **Form submit buttons must be `type="submit"`.** Add category/feed buttons were
  `type="button"` inside `<form onsubmit=…>` — clicks did nothing; Enter might work.
  Fixed in `2fce616`. Delete/refresh failures were silent until error banner added.

## 2026-05-30 — Article retention

- **Retention applies to articles, not feeds.** Feed rows stay; old **read** article bodies are
  `DELETE`d after `RETENTION_DAYS` (default 30).
- **Never purge** `is_saved = TRUE` or `is_read = FALSE` rows.
- Age uses `COALESCE(published_at, fetched_at)` so items without RSS publish date still expire.
- **`RETENTION_DAYS=0`** disables the cron entirely (dev/safety valve).
- Dexie offline cache is independent — clients may retain deleted articles until next online sync.

## 2026-05-30 — Watchtower / deploy

- **Watchtower is now running** in `~/projects/management/` with `WATCHTOWER_LABEL_ENABLE=true`.
  Push to `main` → CI → GHCR → auto-redeploy for labeled containers.
- **Label alone does not deploy Watchtower.** `com.centurylinklabs.watchtower.enable=true` on
  `rss-fresh` only marks the app as eligible; the Watchtower **service** must run separately.
- **Use `WATCHTOWER_LABEL_ENABLE=true`** — avoids updating all 12+ host containers.
- **Manual fallback** if Watchtower stops: `docker compose pull && docker compose up -d`.

## 2026-05-30 — Mobile nav + refresh UX

- **`$effect` forcing `mobilePane = 'detail'`** when `selectedArticleId` set broke the ☰ sidebar
  on iOS — remove effect; navigate to detail only via `onPickArticle`. Clear selection on back-to-list.
- **`refreshAll()` must set `refreshing`** — iOS has no hover; users need spinner + "Updated" /
  offline notice (`refreshNotice`), not silent API calls.

## 2026-05-30 — iOS PWA + Read tab

- **No hover on iOS** — use `.btn` with border/background and 44px min height.
- **Safe area + `100dvh`** for standalone PWA.
- **Read tab needs `read=1` API** — disabling unread alone lists everything mixed.
- **Defer list prune** until back navigation so reader keeps content after auto-mark-read.

## 2026-05-27 — Production deploy (Hetzner)

- In-network DB URL: **`pgbouncer:5432`**, not host `:6432`.
- External network: **`postgres-shared-net`**.
- Table owner must be **`rss_user`** if migrations run as `postgres` in Adminer.
- GHCR CI: `MustafaEEroglu/shared-workflows/.github/workflows/docker-build.yml@main`.
- UI gated by Cloudflare Access, not in-app login.

## 2026-05-27 — Build / embed / PWA

- `embed.FS` needs `.gitkeep` in `web/dist`; distroless uses binary healthcheck subcommand.
- Workbox Background Sync for offline mutations; pgx needs `default_query_exec_mode=exec`.
