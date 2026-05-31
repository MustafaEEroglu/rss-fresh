<!-- memory-bank-schema: v1 -->
# Lessons Learned

## 2026-05-31 — OpenClaw removed

- Removed `/api/v1/news/summary`, bearer middleware, `OPENCLAW_*` env, `internal/openclaw`.
- Critical push stays on Telegram. After removing a feature, **purge stale env keys on VPS**
  or the container may fail startup on missing required vars.

## 2026-05-31 — Feed ingest cutoff

- Skip RSS items with `published_at < feed.created_at` — prevents archive backfill on new feeds.
  Nil publish date passes through.

## 2026-05-31 — FeedManager Add buttons

- Submit buttons inside `<form onsubmit=…>` must be `type="submit"`, not `type="button"`.
  Fixed `2fce616`. Surface API errors in UI — silent failures confuse operators.

## 2026-05-30 — Article retention

- Retention deletes **read, non-saved** article rows only — never feeds/categories.
- Age: `COALESCE(published_at, fetched_at)`. `RETENTION_DAYS=0` disables cron.
- Dexie cache is client-side; server purge does not sync offline storage.

## 2026-05-30 — Watchtower / deploy

- Watchtower runs in `~/projects/management/` with `WATCHTOWER_LABEL_ENABLE=true`.
- Label on `rss-fresh` alone is insufficient — the Watchtower **container** must exist.
- Fallback: `docker compose pull && docker compose up -d` in app folder.

## 2026-05-30 — Mobile nav + refresh UX

- Do not `$effect`-force `mobilePane = 'detail'` on article select — breaks iOS sidebar.
- `refreshAll()` must set `refreshing` + `refreshNotice` — no hover feedback on iOS.

## 2026-05-30 — iOS PWA + Read tab

- 44px min touch targets, `100dvh`, safe-area insets.
- Read tab needs `read=1` API param — `unread=0` alone shows mixed list.
- Defer list prune until back navigation after auto-mark-read.

## 2026-05-27 — Production deploy (Hetzner)

- Compose truth: **`central-pgbouncer:6432`**, network **`central-postgres-net`**.
- Table owner **`rss_user`**. Pgx: `default_query_exec_mode=exec` under PgBouncer transaction mode.
- CI: `MustafaEEroglu/shared-workflows/.github/workflows/docker-build.yml@main`.

## 2026-05-27 — Build / embed / PWA

- `embed.FS` needs `.gitkeep` in `web/dist`; distroless healthcheck via subcommand.
- Workbox Background Sync for offline mutations.
