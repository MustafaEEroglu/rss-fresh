<!-- memory-bank-schema: v1 -->
# Product Context

## What
Personal, ultra-lightweight RSS / news manager (FreshRSS alternative) for a single
operator on a 4 GB Hetzner VPS (`mustafaeroglu`), using shared `central-postgres` via PgBouncer.

## Why
- Avoid heavy self-hosted readers and duplicate DB stacks.
- Reuse `central-postgres` on `postgres-shared-net`.
- OpenClaw OS consumes `/api/v1/news/summary` with a bearer token.

## Hard constraints
1. `mem_limit: 256m`; steady-state RAM well under 100 MB.
2. No dedicated Postgres container — shared cluster only.
3. Bind `127.0.0.1:8088`; public access via Cloudflare Tunnel only.
4. Staggered cron fetch, not continuous polling.
5. Single-user: **Cloudflare Access**, not in-app accounts.
6. **PWA on iOS** is a first-class client — 44px touch targets, visible button chrome, safe-area insets.

## Success criteria
| Criterion | Status |
|-----------|--------|
| Operator reads feeds via tunnel + Access | **Met** |
| Worker + API stable on VPS | **Met** |
| Unread / Read / Saved filters | **Met** (`9491a84`) |
| iOS PWA touch + filter bar | **Shipped** — verify on device |
| Refresh feedback + mobile sidebar | **Shipped** (`65ca785`) — verify on device |
| Auto-redeploy via Watchtower | **Blocked** — container not running on VPS |
| OpenClaw summary endpoint | **Built** — verify Access bypass if needed |
| Telegram critical + digest | **Pending** — env not set |
| 24h soak / polite PgBouncer tenant | **Recommended** — not logged |

## Out of scope
- Multi-user / in-app login
- Full-text search beyond filters
- Article scraping beyond RSS payloads
- Native mobile apps (PWA only)
