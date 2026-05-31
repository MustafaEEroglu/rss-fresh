# RSS-Fresh

A personal, ultra-lightweight RSS / news manager. Single Go binary, embedded
Svelte 5 PWA, talks to the existing `central-postgres` cluster through
PgBouncer, and sends Telegram notifications (critical push + daily digest).
Designed to run inside `mem_limit: 256m` behind Cloudflare Zero Trust.

## Why one more RSS reader

FreshRSS / Miniflux / Tiny Tiny RSS each ship their own database and want a
PHP/MySQL or Python/Postgres stack. On a 4 GB box already running a
centralised PgBouncer, that's wasteful. RSS-Fresh is one container, one
binary, one connection-string pointing at the shared cluster.

## Architecture in one paragraph

```
Browser/PWA --HTTPS--> Cloudflare Tunnel --localhost:8088--> rss-fresh:3000 (Go)
                                                              |
                                                              +--> central-pgbouncer:6432 -> central-postgres
                                                              +--> api.telegram.org
```

The Go binary embeds the built Svelte SPA via `embed.FS`, runs an in-process
`gocron` scheduler that fetches RSS in staggered batches, dedups by
`(feed_id, guid)`, optimistically pushes to Telegram for `is_critical`
categories, and emits one daily digest message.

## Repo layout

```
backend/
  cmd/rss-fresh/         entrypoint
  internal/config/       env loading
  internal/db/           pgxpool repositories (PgBouncer-safe)
  internal/httpapi/      chi router, handlers, middleware
  internal/rss/          gocron-driven fetcher
  internal/telegram/     throttled notifier (critical push + daily digest)
  internal/scheduler/    gocron wiring
  migrations/init.sql    canonical schema (idempotent)
  web/embed.go           embeds the built SPA
frontend/
  src/                   Svelte 5 + Tailwind v4 + Dexie + vite-plugin-pwa
.github/workflows/       CI/CD via shared-workflows
.memory_bank/            project state for AI sessions
```

Full deployment + ops runbook: see [`INFRA_HANDOFF.md`](INFRA_HANDOFF.md).

## Local development

You don't need Docker for the inner loop:

```sh
# 1) Frontend dev server (proxies /api -> http://127.0.0.1:3000)
cd frontend
npm install
npm run dev

# 2) Backend (in a second terminal)
cd backend
cp .env.example .env  # fill DATABASE_URL at minimum
go run ./cmd/rss-fresh
```

Open `http://localhost:5173`. The Vite dev server proxies `/api/*` through to
the Go server on `:3000`.

For a full Docker build:

```sh
docker build -t rss-fresh:dev .
docker run --rm -p 8088:3000 \
  --env-file .env \
  --network central-postgres-net \
  rss-fresh:dev
```

## Tech stack

- **Backend**: Go 1.23, `chi`, `pgx/v5` + `pgxpool`, `gofeed`, `gocron/v2`,
  `telegram-bot-api/v5`, stdlib `slog`.
- **Frontend**: Svelte 5 (runes), Vite, TypeScript, Tailwind CSS v4, Dexie
  (IndexedDB), `vite-plugin-pwa` (Workbox 7).
- **Database**: existing PostgreSQL via PgBouncer (transaction pooling).
  Pgx pool runs with `default_query_exec_mode=exec` (mandatory under
  PgBouncer transaction mode).
- **Deploy**: distroless static, `127.0.0.1:8088`, GHCR + Watchtower,
  Cloudflare Zero Trust.

## API at a glance

All `/api/v1/*` JSON, snake_case, RFC 3339 timestamps. Cloudflare Access
gates the UI.

| Method | Path | Auth |
|---|---|---|
| GET | `/api/v1/healthz` | none |
| GET / POST / PATCH / DELETE | `/api/v1/categories[/:id]` | CF Access |
| GET / POST / PATCH / DELETE | `/api/v1/feeds[/:id]` | CF Access |
| POST | `/api/v1/feeds/:id/refresh` | CF Access |
| GET | `/api/v1/articles?category_id&unread&saved&limit&cursor` | CF Access |
| PATCH | `/api/v1/articles/:id` | CF Access |
| POST | `/api/v1/articles/mark-read` | CF Access |

## License

MIT — personal project, use at your own risk.
