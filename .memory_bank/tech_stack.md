<!-- memory-bank-schema: v1 -->
# Tech Stack

## Backend (Go 1.23)
| Concern | Choice |
|---|---|
| Router | `github.com/go-chi/chi/v5` |
| DB | `github.com/jackc/pgx/v5` + `pgxpool` (`default_query_exec_mode=exec`) |
| RSS | `github.com/mmcdole/gofeed` |
| Cron | `github.com/go-co-op/gocron/v2` |
| Telegram | `go-telegram-bot-api/v5` (optional) |
| Logging | `log/slog` JSON |
| SPA | `embed.FS` in single binary |

## Frontend
| Concern | Choice |
|---|---|
| Bundler | Vite 5 |
| UI | Svelte 5 + TypeScript (runes, `.svelte.ts` store) |
| CSS | Tailwind CSS v4 + shared `.btn` / `.btn-segment` components in `app.css` |
| Offline | Dexie + `vite-plugin-pwa` (Workbox 7) |
| iOS PWA | `100dvh` / `-webkit-fill-available`, `env(safe-area-inset-*)`, 44px min touch |

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
| Redeploy | Watchtower |
| Bind | `127.0.0.1:8088:3000` |
| Limits | `256m` RAM, `0.5` CPU, `read_only` + tmpfs `/tmp` |
| Edge auth | Cloudflare Access |

## Open items
- Sync `docker-compose.yml` / `INFRA_HANDOFF.md` with production host/port/network names.
- Telegram env when bot is ready.
- OpenClaw Service Auth if Access blocks machine clients.
