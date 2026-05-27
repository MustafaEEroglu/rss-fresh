# Tech Stack — RSS-Fresh

## Backend (Go 1.23)
| Concern | Choice | Why |
|---|---|---|
| Router | `github.com/go-chi/chi/v5` | Low overhead, stdlib style. |
| DB driver | `github.com/jackc/pgx/v5` + `pgxpool` | PgBouncer-safe with `default_query_exec_mode=exec`. |
| RSS parser | `github.com/mmcdole/gofeed` | RSS/Atom/JSON Feed. |
| Cron | `github.com/go-co-op/gocron/v2` | In-process; no sidecar. |
| Telegram | `go-telegram-bot-api/v5` | Optional via env. |
| Logging | `log/slog` | JSON structured logs. |
| SPA embed | `embed.FS` | Single binary + container. |

## Frontend
| Concern | Choice |
|---|---|
| Bundler | Vite 5 |
| Framework | Svelte 5 + TypeScript (runes, `.svelte.ts` store) |
| Styling | Tailwind CSS v4 |
| Offline | Dexie + `vite-plugin-pwa` (Workbox 7) |

## Database (shared cluster — production)
| Item | Production value |
|------|------------------|
| Network | `postgres-shared-net` (Docker external) |
| Pooler hostname (in-network) | `pgbouncer` |
| Pooler port (in-network) | **5432** |
| Host port (localhost only) | 6432 → pgbouncer:5432 |
| Postgres container | `central-postgres` |
| Database | `rss_fresh` |
| App user | `rss_user` (must own tables) |
| Connection string flag | `default_query_exec_mode=exec` |
| App pool size | `pool_max_conns=4` |

## Deploy stack (production)
| Item | Value |
|------|--------|
| Image | `ghcr.io/mustafaeeroglu/rss-fresh:latest` |
| Base | `gcr.io/distroless/static-debian12:nonroot` |
| CI | `MustafaEEroglu/shared-workflows` → `docker-build.yml@main` |
| Redeploy | Watchtower (`com.centurylinklabs.watchtower.enable=true`) |
| Bind | `127.0.0.1:8088:3000` |
| Limits | `mem_limit: 256m`, `cpus: 0.5`, `read_only` + tmpfs `/tmp` |
| Edge auth | Cloudflare Access (single user) |
| Repo CI gate | `go test` + `npm run check` + `npm run build` |

## Resolved decisions
- **Subdomain:** served via Cloudflare Tunnel + Access (exact hostname in Zero Trust dashboard).
- **No in-app login** — Access is the authenticator.
- **Category colours/icons** — deferred.

## Open items (non-blocking)
- Sync committed `docker-compose.yml` / `INFRA_HANDOFF.md` with production host/port/network names.
- Enable Telegram when bot token + chat ID are ready.
- Optional: OpenClaw Service Auth if Access blocks server-to-server summary calls.
