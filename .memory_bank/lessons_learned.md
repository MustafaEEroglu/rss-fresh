<!-- memory-bank-schema: v1 -->
# Lessons Learned

## 2026-05-30 — iOS PWA + Read tab

- **iOS has no hover.** Buttons styled with `hover:` only look like plain text in standalone PWA.
  Use visible borders/backgrounds (`.btn` in `app.css`) and `min-height: 2.75rem` (44px).
- **Safe area:** `viewport-fit=cover` requires `env(safe-area-inset-*)` on header/footer or
  controls sit under notch/home indicator.
- **Viewport height:** use `100dvh` and `-webkit-fill-available`, not `100vh` alone, in iOS PWA.
- **Unread list UX:** auto-mark-read on open, but defer removing the row from the list until
  `pruneArticlesToFilter()` on back navigation — otherwise the reader pane loses content mid-read.
- **Article list filters:** backend needs explicit `read=1`; turning off `unread` alone returns
  all articles (read + unread mixed) — not a substitute for a Read tab.

## 2026-05-27 — Production deploy (Hetzner)

- **Docker internal port ≠ host port.** In-network URL: **`pgbouncer:5432`**, not `:6432`.
- **Network name:** live stack uses **`postgres-shared-net`**, not `central-postgres-net`.
- **Table owner:** migrations run as `postgres` in Adminer → `rss_user` cannot alter indexes;
  fix with `ALTER TABLE … OWNER TO rss_user`.
- **GHCR workflow:** `MustafaEEroglu/shared-workflows/.github/workflows/docker-build.yml@main`.
- **UI auth:** Cloudflare Access single-email Allow policy; no in-app login.

## 2026-05-27 — Build / embed / PWA

- **`embed.FS`:** ship `.gitkeep` in `web/dist`; runtime check for `index.html`.
- **Distroless healthcheck:** subcommand in binary, not curl.
- **Workbox Background Sync** for offline PATCH/POST mutations.
- **PgBouncer + pgx:** `default_query_exec_mode=exec` mandatory.
