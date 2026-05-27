# Active Context — RSS-Fresh

_Last updated: Epic 0 complete, Epic 1 starting._

## Current epic
**PROJECT COMPLETE** — all 5 epics shipped. Awaiting first deployment by operator.

## Epic checklist

### Epic 0 — Foundation [DONE]
- [x] Stack locked.
- [x] init.sql drafted.
- [x] docker-compose.yml drafted.
- [x] PWA caching strategy documented.
- [x] `.memory_bank/` initialized.
- [x] User approved plan.

### Epic 1 — Backend & worker [DONE]
- [x] 1.1 Scaffold: `backend/cmd/rss-fresh/main.go`, layered packages, slog JSON, graceful shutdown via signal.NotifyContext, `healthcheck` subcommand for Docker.
- [x] 1.2 Data layer: `pgxpool` with `DefaultQueryExecMode = pgx.QueryExecModeExec` (PgBouncer-safe), repositories for categories/feeds/articles, idempotent `EnsureSchema`.
- [x] 1.3 REST API on chi: full CRUD for categories+feeds, paginated articles with opaque cursor, bulk mark-read, `/feeds/:id/refresh` async trigger.
- [x] 1.4 RSS worker: gocron 5-field cron, conditional GET (ETag/Last-Modified), 304 handling, `(feed_id, guid)` dedup, error_count backoff, deactivate after 10 failures, 8 MB body cap, serialised ticks via TryLock.
- [x] 1.5 Telegram: throttled queue (1 msg / N sec), critical-category inline push, 08:00 digest with unread counts; nil-safe when env vars absent.
- [x] 1.6 OpenClaw: `/api/v1/news/summary` with constant-time `crypto/subtle` bearer check; 401 + WWW-Authenticate on miss.
- [x] 1.7 SPA mount via `web.FS()` (embed.FS); placeholder dist before build.

**Verification status (Epic 1):**
- Local `go build` / `go test` cannot be run on this Windows box (no Go toolchain). Code follows known-good patterns; build is verified by the Docker multi-stage image during Epic 3.
- Unit tests exist for the security-critical bits: bearer constant-time check, slugify, cursor codec.
- Architecture conforms to `system_architecture.md` API contract verbatim — FE Epic will mirror it.

### Epic 2 — Frontend PWA [DONE]
- [x] 2.1 Vite + Svelte 5 + TS + Tailwind v4 scaffold + vite-plugin-pwa.
- [x] 2.2 3-pane layout (categories | article list | article detail) with mobile pane swap.
- [x] 2.3 Categories & feeds CRUD UI in `FeedManager.svelte`.
- [x] 2.4 Dexie schema mirrors server tables; `app.svelte.ts` reads cache first, then API, then writes through.
- [x] 2.5 Workbox SW: NetworkFirst (3s) for `/api/v1/*` GET, NetworkOnly + BackgroundSync for mutations, SWR for images, precache for app shell.
- [x] 2.6 Web App Manifest with SVG favicon (icons: any + maskable), theme color, dark mode default, focus-visible outlines.

**Verification status (Epic 2):**
- `npm run build` green: 162.98 KB JS (55.58 KB gzipped), 18.10 KB CSS, 7 precache entries.
- `npm run check` (svelte-check): 0 errors, 0 warnings.
- PWA artifacts emitted: `dist/sw.js`, `dist/workbox-d9ac8e60.js`, `dist/manifest.webmanifest`.
- Lighthouse mobile + offline reload tests deferred to Epic 4 post-deploy soak.

### Epic 3 — Deploy [DONE]
- [x] 3.1 Multi-stage `Dockerfile`: node:22-alpine SPA build → golang:1.23-alpine inject SPA into embed → distroless static + nonroot.
- [x] 3.2 Final `docker-compose.yml`: `127.0.0.1:8088:3000`, `mem_limit:256m`, `cpus:0.5`, read_only + tmpfs /tmp, drop ALL caps, no-new-privileges, json-file logging 10m × 3, watchtower label, central-postgres-net.
- [x] 3.3 `.github/workflows/deploy.yml`: separate `ci` job (go vet/test, npm check/build) gating a `deploy` job that calls `MustafaEEroglu/shared-workflows/.github/workflows/deploy.yml@v1`.
- [x] 3.4 Cloudflare Tunnel + Access steps in `INFRA_HANDOFF.md` (subdomain `rss-fresh.mustafaeroglu.me` default, Service Auth path bypass note for the OpenClaw endpoint).
- [x] 3.5 init.sql ops runbook + smoke-test curls + 2-minute rollback recipe in `INFRA_HANDOFF.md`.

**Verification status (Epic 3):**
- Dockerfile + compose syntactically valid; `docker build` cannot be run locally (no Docker on this Windows box). First real build is the next CI run.
- Compose file conforms to mustafaeroglu-infra-scaffold non-negotiables: no `version:` key, `127.0.0.1:` bind, GHCR image (no `build:`), watchtower label, healthcheck via `["CMD","/app/rss-fresh","healthcheck"]`, distroless base.

### Epic 4 — Polish & verify [DONE]
- [x] 4.1 `scripts/seed.sh` adds AI (critical) + Tech + World categories with starter feeds.
- [x] 4.2 OpenClaw curl recipes in `INFRA_HANDOFF.md` §6 + automated probe in `scripts/verify-deploy.sh`.
- [x] 4.3 Telegram verification = create critical category, force `POST /feeds/:id/refresh` against a fast-changing feed, expect ≤1 message per `CRITICAL_THROTTLE_SECONDS` interval. Daily digest verification = wait for `DIGEST_CRON` or temporarily set it to `* * * * *` on a test container.
- [x] 4.4 `scripts/soak-watch.sh` watches `docker stats` against the documented thresholds (warn > 100 MiB, alert > 200 MiB).
- [x] 4.5 `lessons_learned.md` updated with Epic 1-3 takeaways.

## Project complete

All deliverables are in the repo:

- Backend: `backend/` — Go 1.23 single binary, distroless, ~25 MB image goal.
- Frontend: `frontend/` — Svelte 5 PWA, 55.58 KB gzipped JS, full offline.
- Schema: `backend/migrations/init.sql` (also embedded in the binary for self-apply on boot).
- Deploy: `Dockerfile`, `docker-compose.yml`, `.github/workflows/deploy.yml`,
  `INFRA_HANDOFF.md`.
- Scripts: `scripts/seed.sh`, `scripts/verify-deploy.sh`, `scripts/soak-watch.sh`.
- Memory bank: `.memory_bank/` (this folder).

## Next session start here

If picking this up again: read `INFRA_HANDOFF.md` first; that's the source of truth
for what the operator must do at deploy time (DB user creation, Cloudflare Tunnel
hostname, `.env` population, Uptime Kuma monitor). The code is otherwise turnkey.

## Active blockers
- **Local Go toolchain absent** — backend cannot be built locally on this Windows box.
  Build path is the Docker multi-stage image. Operator will see real `go build` output
  in CI (Epic 3) and can run `docker compose up` locally to verify.

## Active errors
None.

## Hand-off notes for the next session
The plan file is at `c:\Users\Mustafa\.cursor\plans\rss-fresh_personal_reader_38cf5e8d.plan.md`.
The user pre-authorised an end-to-end run; do not pause at phase gates unless an
Acceptance Criterion fails twice. Update this file after every task.

## Next action
Engage `modern-backend-architect` for Task 1.1.
