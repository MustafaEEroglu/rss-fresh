<!-- memory-bank-schema: v1 -->
# Lessons Learned

## 2026-05-31 — Code quality pass (strict review + refactors applied)

- **`db.DB.pool` is unexported.** Was `Pool`. All call sites in `articles.go`, `feeds.go`, `categories.go`, `migrations.go` updated. External packages must use the method API.
- **Shared `scanner` interface for pgx.** Defined in `db.go`: `type scanner interface { Scan(...any) error }`. Both `pgx.Row` and `pgx.Rows` satisfy it; `scanArticle(scanner)` and `scanFeed(scanner)` serve all query patterns. Never duplicate scan functions again.
- **Column select lists are constants.** `articleSelectCols` and `feedSelectCols` are package-level constants. Feed column list was repeated six times; now one site.
- **`telegram.Notifier` is an interface; `notifier` (lowercase) is the concrete struct.** `New()` returns `(Notifier, error)`, never nil — `nopNotifier{}` Null Object handles disabled-Telegram. No nil-receiver guards on public methods.
- **`Fetcher.RefreshFeed` accepts `context.Context` as first arg.** Call site uses `context.WithoutCancel(r.Context())` — strips the 30 s request timeout but preserves values. `context.Background()` silently detaches from any future shutdown signal.
- **SSRF guard on RSS fetcher.** `privateIPDialer` rejects loopback, private, and link-local IPs at connection time (DNS rebinding defence).
- **`since` bad query param returns 400.** Silently dropping an unparseable date filter was wrong; now consistent with `category_id`/`feed_id` validation.
- **Named constants for magic durations.** `maxJobRuntime = 30 * time.Minute` in `scheduler.go`.
- **No dead `sync.Once` bodies.** Removed empty `once.Do(func() {})` from telegram notifier.
- **`strings.NewReplacer` is a package-level var.** `htmlReplacer` in `telegram/notifier.go` — was recreated per call in a notification hot path.
- **Telegram notifier tests:** old nil-receiver tests replaced by `nopNotifier` tests; `newTestNotifier()` uses `slog.Default()` to prevent nil-log panic when the queue overflows in tests.

## 2026-05-31 — UX / a11y pass

- **`window.confirm()` is silently suppressed in iOS PWA standalone mode** (returns `true` without displaying).
  Always use inline two-step confirmation for destructive actions in this app.
- **Shared error state across multiple forms causes confusion.** Use per-form error variables
  (`catError`, `feedError`, `listError`) and clear on first keystroke in the relevant field.
- **`btn-danger` needs its own `:hover` rule.** The base `.btn:hover` sets `background: #334155` (slate),
  which overrides the red background on hover — making danger buttons look like ghost buttons.
  Any colored button variant must declare its own `:hover:not(:disabled)` rule in `app.css`.
- **`role="status"` vs `role="alert"`:** `role="status"` implies `aria-live="polite"` (waits for
  current utterance). Error banners need `role="alert"` (`aria-live="assertive"`) to interrupt immediately.
- **`aria-pressed` is for independent toggles, not exclusive selections.** Mutually-exclusive view
  switchers (Reader / Feeds) should use `role="tablist"` + `role="tab"` + `aria-selected`, not `aria-pressed`.
- **Svelte 5 scroll reset on content change:** use `$effect` + `tick().then(() => el.scrollTo({top:0, behavior:'instant'}))`.
  The `tick()` ensures the DOM has updated with the new article before scrolling.
- **`prefers-reduced-motion`:** any CSS `animation` that runs continuously (e.g. `icon-spin`) must be
  wrapped in `@media (prefers-reduced-motion: no-preference)` or suppressed in a `reduce` block.

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
