# Product Context — RSS-Fresh

## What
A personal, ultra-lightweight RSS / news manager (FreshRSS alternative) for a single
operator. Deployed on a 4 GB Hetzner VPS (`mustafaeroglu`) with the existing
`central-postgres` cluster and PgBouncer.

## Why
- Avoid heavy self-hosted readers (FreshRSS/Miniflux) and duplicate DB stacks.
- Reuse `central-postgres` via PgBouncer on `postgres-shared-net`.
- OpenClaw OS consumes `/api/v1/news/summary` with a bearer token.

## Hard constraints (non-negotiable)
1. `mem_limit: 256m`; target steady-state RAM well under 100 MB.
2. No dedicated Postgres container for this app — shared cluster only.
3. Bind `127.0.0.1:8088`; public access only via Cloudflare Tunnel.
4. Staggered cron fetch, not continuous polling.
5. Single-user: **Cloudflare Access**, not app-level accounts.

## Success criteria (status)
| Criterion | Status |
|-----------|--------|
| Operator reads feeds via tunnel + Access | **Met** (2026-05-27) |
| Worker + API stable on VPS | **Met** (healthz 200) |
| OpenClaw summary endpoint | **Built** — verify token + Access bypass as needed |
| Telegram critical + digest | **Pending** — env vars not set (warn-only in logs) |
| 24h soak / PgBouncer polite tenant | **Recommended** — not logged in memory bank yet |

## Out of scope
- Multi-user / in-app login
- Full-text search beyond simple filters
- Article scraping beyond RSS payloads
- Native mobile apps (PWA only)
