<!-- memory-bank-schema: v1 -->
# Tech Stack

## Backend (Go 1.23)
| Concern | Choice |
|---|---|
| Router | `github.com/go-chi/chi/v5` |
| DB | `github.com/jackc/pgx/v5` + `pgxpool` (`default_query_exec_mode=exec`) |
| RSS | `github.com/mmcdole/gofeed` |
| Cron | `github.com/go-co-op/gocron/v2` |
| Retention | Daily cron; `RETENTION_DAYS` (default 30), `RETENTION_CRON` (default `0 4 * * *`) |
| Telegram | `go-telegram-bot-api/v5` (optional; digest today, critical push to be removed) |
| OpenClaw | Bearer-gated `GET /api/v1/news/summary`; push TBD |
| Logging | `log/slog` JSON |
| SPA | `embed.FS` in single binary |

## Frontend
| Concern | Choice |
|---|---|
| Bundler | Vite 5 |
| UI | Svelte 5 + TypeScript (runes, `.svelte.ts` store) |
| CSS | Tailwind v4 + `.btn` / `.btn-segment` in `app.css` |
| Offline | Dexie + `vite-plugin-pwa` (Workbox 7) |
| iOS PWA | `100dvh`, safe-area insets, 44px touch, `refreshing` + `refreshNotice` on manual sync |

## Database (production)
| Item | Value |
|------|--------|
| Network | `postgres-shared-net` (external) |
| Pooler | `pgbouncer:5432` (in-network; host map `127.0.0.1:6432`) |
| DB / user | `rss_fresh` / `rss_user` |
| Pool | `pool_max_conns=4`, `default_query_exec_mode=exec` |

## Deploy
| Item | Value |
|------|--------|
| Image | `ghcr.io/mustafaeeroglu/rss-fresh:latest` |
| Base | `gcr.io/distroless/static-debian12:nonroot` |
| CI | `MustafaEEroglu/shared-workflows` → `docker-build.yml@main` |
| Host path | `~/projects/rss-fresh` |
| Bind | `127.0.0.1:8088:3000` |
| Limits | `256m` RAM, `0.5` CPU, `read_only` + tmpfs `/tmp` |
| Edge auth | Cloudflare Access |
| Watchtower label | `com.centurylinklabs.watchtower.enable=true` on `rss-fresh` service |
| Auto-redeploy | **Intended** via Watchtower in `~/projects/management/` — **not active** (2026-05-30) |
| Manual fallback | `docker compose pull && docker compose up -d` in app folder |

## Open items
- **TODO #1:** Feed ingest cutoff at `feed.created_at` in `internal/rss/fetcher.go`.
- **TODO #2:** Move `NotifyCritical` target from Telegram to OpenClaw; drop critical→Telegram.
- **TODO #3:** Messaging tools consume `is_saved` articles (extend summary API / digest query).
- Stand up unified management compose (portainer + uptime-kuma + watchtower).
- Sync committed `docker-compose.yml` / `INFRA_HANDOFF.md` with production DB/network names.
- Telegram env when bot is ready (saved-article digest design first).
- OpenClaw Service Auth if Access blocks machine clients.
