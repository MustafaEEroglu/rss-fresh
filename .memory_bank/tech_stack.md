<!-- memory-bank-schema: v1 -->
# Tech Stack

## Backend (Go 1.23)
| Concern | Choice |
|---|---|
| Router | `github.com/go-chi/chi/v5` |
| DB | `github.com/jackc/pgx/v5` + `pgxpool` (`default_query_exec_mode=exec`) |
| RSS | `github.com/mmcdole/gofeed` |
| Cron | `github.com/go-co-op/gocron/v2` |
| Retention | `RETENTION_DAYS` (default 30), `RETENTION_CRON` (`0 4 * * *`) |
| Telegram | `go-telegram-bot-api/v5` — `Notifier` interface + `nopNotifier` null-object |
| Logging | `log/slog` JSON |
| SPA | `embed.FS` in single binary |
| SSRF | `privateIPDialer` in `internal/rss` rejects loopback/private/link-local IPs |

## Frontend
| Concern | Choice |
|---|---|
| Bundler | Vite 5 |
| UI | Svelte 5 + TypeScript (runes, `.svelte.ts` store) |
| CSS | Tailwind v4 |
| Offline | Dexie + `vite-plugin-pwa` (Workbox 7) |
| iOS PWA | `100dvh`, safe-area insets, 44px touch, refresh feedback |
| Tests | Vitest + jsdom (`npm test`) |

## Database (production)
| Item | Value |
|------|--------|
| Network | `central-postgres-net` (external) |
| Pooler | `central-pgbouncer:6432` |
| DB / user | `rss_fresh` / `rss_user` |
| Pool | `pool_max_conns=4`, `default_query_exec_mode=exec` |

## Deploy
| Item | Value |
|------|--------|
| Image | `ghcr.io/mustafaeeroglu/rss-fresh:latest` |
| Base | `gcr.io/distroless/static-debian12:nonroot` |
| CI | `MustafaEEroglu/shared-workflows` → `docker-build.yml@main` |
| Host | `~/projects/rss-fresh` · bind `127.0.0.1:8088:3000` |
| Limits | `256m` RAM, `0.5` CPU, read-only + tmpfs `/tmp` |
| Edge | Cloudflare Access |
| Redeploy | Watchtower (`~/projects/management/`, label-enabled) |

## Environment
| Variable | Required | Notes |
|----------|----------|-------|
| `DB_PASSWORD` | yes | Substituted into compose `DATABASE_URL` |
| `TELEGRAM_BOT_TOKEN` | prod | Set on VPS |
| `TELEGRAM_CHAT_ID` | prod | Set on VPS |

Dev: only `DATABASE_URL` required at startup. Telegram disabled (nopNotifier) if token empty.
