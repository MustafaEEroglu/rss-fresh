# Lessons Learned — RSS-Fresh

_New entries appended at the top with date + epic._

## 2026-05-27 — Production deploy (Hetzner)

- **Docker internal port ≠ host port.** `docker port pgbouncer` showed `5432/tcp -> 127.0.0.1:6432`.
  Container-to-container URLs must use **`pgbouncer:5432`**. Using `:6432` in `DATABASE_URL`
  produced `connection refused` even when DNS and network were correct.
- **External network name must match the live stack.** Planned name `central-postgres-net` did
  not exist; the postgres stack uses **`postgres-shared-net`**. Compose fix:
  `networks.central-postgres-net.external: true` + `name: postgres-shared-net`.
- **Table owner must match app user.** Running `init.sql` in Adminer as `postgres` left
  `feeds` (etc.) owned by `postgres`. App user `rss_user` then failed on boot with
  `must be owner of table feeds (SQLSTATE 42501)` during `EnsureSchema` (index creation).
  Fix: `ALTER TABLE … OWNER TO rss_user` or recreate DB with `OWNER rss_user` and run
  migrations as `rss_user`.
- **GHCR reusable workflow name.** Repo planned `deploy.yml@v1`; actual shared workflow is
  **`MustafaEEroglu/shared-workflows/.github/workflows/docker-build.yml@main`** (no `v1` tag).
- **UI auth is Cloudflare Access, not in-app login.** Public tunnel URL without an Access
  policy is world-readable; single-email Allow policy is the intended gate.
- **Verify with:** `docker compose config | grep DATABASE_URL`, `docker compose logs --tail 5`,
  `curl http://127.0.0.1:8088/api/v1/healthz`, and `SELECT tablename, tableowner FROM pg_tables`.

## 2026-05-27 — Epic 3 / Epic 4
- **`embed.FS` placeholder pattern.** Go's `//go:embed` errors at compile time
  if the pattern matches no files. Solution: ship a `.gitkeep` inside `web/dist` and use
  `all:dist`; runtime checks for `index.html` before serving SPA.
- **Multi-stage Docker SPA injection.** Copy Vite `dist/` into `web/dist/` before `go build`.
- **Distroless + read_only + tmpfs.** Mount `tmpfs:/tmp:size=16m` when `read_only: true`.
- **Healthcheck subcommand** in the binary — no curl/wget in distroless.
- **Workbox Background Sync** for offline mutations (NetworkOnly + queue).
- **Svelte 5 runes in `.svelte.ts` class** — single `app` store for components.

## 2026-05-27 — Epic 0
- **PgBouncer + pgx:** `default_query_exec_mode=exec` in the connection string.
- **Dexie over Cache API** for offline article queries.
- **Bind `127.0.0.1:8088`** when fronted by Cloudflare Tunnel on the same host.
