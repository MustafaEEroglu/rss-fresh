<!-- memory-bank-schema: v1 -->
# Product Context

## What
Personal, ultra-lightweight RSS / news manager (FreshRSS alternative) for a single
operator on a 4 GB Hetzner VPS (`mustafaeroglu`), using shared `central-postgres` via PgBouncer.

## Why
- Avoid heavy self-hosted readers and duplicate DB stacks.
- Reuse `central-postgres` on `postgres-shared-net`.
- **Telegram** is the operator notification channel — critical push for `is_critical`
  categories, daily digest with unread counts + saved-article highlights.

## Hard constraints
1. `mem_limit: 256m`; steady-state RAM well under 100 MB.
2. No dedicated Postgres container — shared cluster only.
3. Bind `127.0.0.1:8088`; public access via Cloudflare Tunnel only.
4. Staggered cron fetch, not continuous polling.
5. Single-user: **Cloudflare Access**, not in-app accounts.
6. **PWA on iOS** is a first-class client — 44px touch targets, visible button chrome, safe-area insets.

## Product rules

| Rule | Status |
|------|--------|
| New feed ingest cutoff | **Shipped** — skip `published_at < feed.created_at` |
| Critical → Telegram | **Shipped** — `is_critical` → `NotifyCritical` |
| Saved → digest | **Shipped** — daily digest includes saved articles (24h) |

## Success criteria
| Criterion | Status |
|-----------|--------|
| Operator reads feeds via tunnel + Access | **Met** |
| Worker + API stable on VPS | **Met** |
| Unread / Read / Saved filters | **Met** |
| iOS PWA touch + filter bar | **Met** |
| Feed ingest from add-date only | **Shipped** |
| Critical push via Telegram | **Shipped** (requires bot env) |
| Saved articles in Telegram digest | **Shipped** (requires bot env) |
| Auto-redeploy via Watchtower | **Met** (operator confirmed) |
| Read-article retention (30 days) | **Shipped** — cron `article-retention` |

## Out of scope
- Multi-user / in-app login
- Full-text search beyond filters
- Article scraping beyond RSS payloads
- Native mobile apps (PWA only)
- External AI / summary integrations (OpenClaw removed)
