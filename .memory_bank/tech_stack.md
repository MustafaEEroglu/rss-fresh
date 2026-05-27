# Tech Stack — RSS-Fresh

## Backend (Go 1.23)
| Concern | Choice | Why |
|---|---|---|
| Router | `github.com/go-chi/chi/v5` | Stdlib-flavoured, no reflection, ~zero overhead. |
| DB driver | `github.com/jackc/pgx/v5` + `pgxpool` | Best-in-class Postgres driver, PgBouncer-safe with `default_query_exec_mode=exec`. |
| RSS parser | `github.com/mmcdole/gofeed` | Handles RSS 1.0/2.0, Atom, JSON Feed; battle-tested. |
| Cron | `github.com/go-co-op/gocron/v2` | In-process scheduler, no external timer container. |
| Telegram | `github.com/go-telegram-bot-api/telegram-bot-api/v5` | Stable, dependency-light. |
| Logging | `log/slog` (stdlib) | Structured logs without extra deps. |
| Env loading | `github.com/joho/godotenv` (dev only) | Loads `.env` for local runs. |
| SPA embed | `embed.FS` (stdlib) | Built SPA shipped inside the Go binary; one container, one image. |

## Frontend
| Concern | Choice | Why |
|---|---|---|
| Bundler | Vite 5 | Smallest dev/build footprint for Svelte. |
| Framework | Svelte 5 + TypeScript | No virtual DOM; ~5 KB runtime; fits "lightweight" rule. |
| Styling | Tailwind CSS v4 | JIT, no runtime, instant cold start. |
| Routing | `svelte-spa-router` | Client-side hash-free routing without SvelteKit. |
| Offline DB | `dexie` (IndexedDB) | Indexed offline queries (filter by category, unread, saved). |
| Service Worker | `vite-plugin-pwa` (Workbox 7) | Precaching + Background Sync out of the box. |

## Database (existing, do not provision)
- Host: `central-pgbouncer:6432` (transaction-pooling mode)
- DB name: `rss_fresh`
- App user: `rss_user` (created manually before deploy)
- Pool size from app: `pool_max_conns=4`
- Schema: see `migrations/init.sql` and `system_architecture.md`.

## Deploy stack (immutable, see mustafaeroglu-infra-scaffold)
- Image: `ghcr.io/mustafaeeroglu/rss-fresh:latest`
- Base: `gcr.io/distroless/static-debian12:nonroot` (Go static binary, ~25 MB image)
- Auto-redeploy: Watchtower watches the GHCR tag.
- CI: reuses `mustafaeeroglu/shared-workflows/.github/workflows/deploy.yml@v1`.
- Runtime cap: `mem_limit: 256m`, bound to `127.0.0.1:8088`.
- Network: `central-postgres-net` (external).

## Open questions
- Which subdomain will Cloudflare Tunnel route to (e.g. `rss.<domain>`)? — confirmed
  during Epic 3 deploy by operator; not blocking earlier work.
- Should categories support a colour/icon for the UI? — deferred; Tailwind tokens for
  now, no DB column.
