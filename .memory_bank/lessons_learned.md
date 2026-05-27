# Lessons Learned — RSS-Fresh

_New entries appended at the top with date + epic._

## 2026-05-27 — Epic 3 / Epic 4
- **`embed.FS` placeholder pattern.** Go's `//go:embed` errors at compile time
  if the pattern matches no files. Solution: ship a `.gitkeep` (or any
  placeholder) inside the embed directory and use `all:dist` to pick it up;
  then have the runtime check for `index.html` to decide whether the SPA is
  actually populated. Letting `web.FS()` return `nil` lets the server fall
  through to "no SPA" gracefully on local Go runs without a built frontend.
- **Multi-stage Docker SPA injection.** Building the frontend in a node stage
  and copying `dist/` into the Go stage **before** `go build` is the cleanest
  way to embed a JS bundle into a Go binary. The `rm -rf web/dist && mkdir -p
  web/dist` before the COPY guards against the placeholder leaking into the
  embed and against stale caches in CI.
- **Distroless static + read_only + tmpfs.** `gcr.io/distroless/static-debian12:nonroot`
  has no `/tmp`. Combine `read_only: true` with a small `tmpfs:/tmp:size=16m`
  mount to keep the rootfs immutable while still satisfying any library that
  spills to `/tmp`.
- **Healthcheck without curl/wget.** Distroless has neither. Solution: ship a
  `healthcheck` subcommand inside the application binary and call it from the
  Docker `HEALTHCHECK` directive (`["CMD","/app/rss-fresh","healthcheck"]`).
  No extra binaries, no extra image layers.
- **Workbox Background Sync for offline mutations.** Naïvely using
  `NetworkFirst` for PATCH/POST loses user actions when offline. Pairing
  `NetworkOnly` with `BackgroundSyncPlugin` queues the mutation in IndexedDB
  and flushes when connectivity returns — and the UI only needs to apply the
  optimistic update locally. This is the difference between "kind of offline"
  and "actually offline" for a reader.
- **Svelte 5 runes inside a class.** Putting all reactive state into a class
  with `$state`/`$derived`/`$effect` fields gives a single import handle
  (`import { app }`) for every component, keeps the store strongly typed,
  and avoids the boilerplate of writable stores. Files use `.svelte.ts`
  extension to enable rune compilation outside `.svelte` files.

## 2026-05-27 — Epic 0
- **PgBouncer + pgx prepared statements**: in transaction-pooling mode,
  PgBouncer cannot route prepared statements safely. Set
  `default_query_exec_mode=exec` in the connection string (or
  `pgxpool.Config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec`).
  Forgetting this is the most common production-only failure for pgx +
  PgBouncer apps.
- **Cache API vs IndexedDB for offline reader UX**: Cache API is keyed by
  Request URL; it cannot answer "give me unread articles in the AI category,
  sorted by published_at desc". Dexie/IndexedDB is the right primitive for
  the reader's offline read path. The SW writes through API responses to
  Dexie; the UI reads from Dexie first.
- **Bind to 127.0.0.1 not 0.0.0.0** when fronted by Cloudflare Tunnel on the
  same host — prevents accidental exposure if the firewall ever lapses.
