<!-- memory-bank-schema: v1 -->
# Lessons Learned

## 2026-05-30 — Watchtower / deploy reality

- **Docker image ≠ running container.** `containrrr/watchtower:latest` can show **Unused** in
  Portainer while no `watchtower` container exists — auto-redeploy is off.
- **Label alone does not deploy Watchtower.** `com.centurylinklabs.watchtower.enable=true` on
  `rss-fresh` only marks the app as eligible; the Watchtower **service** must run separately
  (planned: `~/projects/management/docker-compose.yml`).
- **Repo grep for watchtower** under `~/projects` finds only the rss-fresh label line — not a
  watchtower service definition. Portainer `portainer_data/compose/` was empty on inspection.
- **Until Watchtower runs:** after every GHCR push, manual
  `docker compose pull && docker compose up -d` in `~/projects/rss-fresh`.
- **Use `WATCHTOWER_LABEL_ENABLE=true`** when standing up Watchtower — avoids updating all 12+
  host containers (vikunja, postgres, etc.).

## 2026-05-30 — Mobile nav + refresh UX

- **`$effect` forcing `mobilePane = 'detail'`** when `selectedArticleId` set broke the ☰ sidebar
  on iOS — remove effect; navigate to detail only via `onPickArticle`. Clear selection on back-to-list.
- **`refreshAll()` must set `refreshing`** — iOS has no hover; users need spinner + "Updated" /
  offline notice (`refreshNotice`), not silent API calls.

## 2026-05-30 — iOS PWA + Read tab

- **No hover on iOS** — use `.btn` with border/background and 44px min height.
- **Safe area + `100dvh`** for standalone PWA.
- **Read tab needs `read=1` API** — disabling unread alone lists everything mixed.
- **Defer list prune** until back navigation so reader keeps content after auto-mark-read.

## 2026-05-27 — Production deploy (Hetzner)

- In-network DB URL: **`pgbouncer:5432`**, not host `:6432`.
- External network: **`postgres-shared-net`**.
- Table owner must be **`rss_user`** if migrations run as `postgres` in Adminer.
- GHCR CI: `MustafaEEroglu/shared-workflows/.github/workflows/docker-build.yml@main`.
- UI gated by Cloudflare Access, not in-app login.

## 2026-05-27 — Build / embed / PWA

- `embed.FS` needs `.gitkeep` in `web/dist`; distroless uses binary healthcheck subcommand.
- Workbox Background Sync for offline mutations; pgx needs `default_query_exec_mode=exec`.
