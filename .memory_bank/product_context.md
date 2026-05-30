<!-- memory-bank-schema: v1 -->
# Product Context

## What
Personal, ultra-lightweight RSS / news manager (FreshRSS alternative) for a single
operator on a 4 GB Hetzner VPS (`mustafaeroglu`), using shared `central-postgres` via PgBouncer.

## Why
- Avoid heavy self-hosted readers and duplicate DB stacks.
- Reuse `central-postgres` on `postgres-shared-net`.
- **OpenClaw OS** is the real-time AI consumer (Bearer `/api/v1/news/summary`; planned:
  critical-category push moved here from Telegram).
- **Telegram** is the operator notification channel — daily digest + **saved-article**
  curation feed (planned; not critical-category spam).

## Hard constraints
1. `mem_limit: 256m`; steady-state RAM well under 100 MB.
2. No dedicated Postgres container — shared cluster only.
3. Bind `127.0.0.1:8088`; public access via Cloudflare Tunnel only.
4. Staggered cron fetch, not continuous polling.
5. Single-user: **Cloudflare Access**, not in-app accounts.
6. **PWA on iOS** is a first-class client — 44px touch targets, visible button chrome, safe-area insets.

## Planned product rules (not yet built)

| Rule | Intent |
|------|--------|
| New feed ingest cutoff | Only articles **published on or after feed add date**; no historical backfill |
| Critical → OpenClaw | `is_critical` triggers OpenClaw notification, **not** Telegram |
| Saved → messaging | Operator-saved articles are the curated payload for messaging tools |

## Success criteria
| Criterion | Status |
|-----------|--------|
| Operator reads feeds via tunnel + Access | **Met** |
| Worker + API stable on VPS | **Met** |
| Unread / Read / Saved filters | **Met** (`9491a84`) |
| iOS PWA touch + filter bar | **Shipped** — verify on device |
| Refresh feedback + mobile sidebar | **Shipped** (`65ca785`) — verify on device |
| FeedManager CRUD (Add submit fix) | **Shipped** (`2fce616`) — deploy pending |
| Auto-redeploy via Watchtower | **Blocked** — container not running on VPS |
| OpenClaw summary endpoint | **Built** — verify Access bypass if needed |
| Feed ingest from add-date only | **Pending** — TODO #1 |
| Critical push → OpenClaw (not Telegram) | **Pending** — TODO #2 |
| Saved articles → messaging tool | **Pending** — TODO #3 |
| Telegram digest (saved-based) | **Pending** — env not set; design TBD |
| 24h soak / polite PgBouncer tenant | **Recommended** — not logged |
| Read-article retention (30 days) | **Shipped** — cron `article-retention` |

## Out of scope
- Multi-user / in-app login
- Full-text search beyond filters
- Article scraping beyond RSS payloads
- Native mobile apps (PWA only)
