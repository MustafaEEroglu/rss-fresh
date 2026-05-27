# Product Context — RSS-Fresh

## What
A personal, ultra-lightweight RSS / news manager (FreshRSS alternative) for a single
operator. Built to live behind Cloudflare Zero Trust on a 4 GB Hetzner box with the
existing `central-postgres` cluster.

## Why
- FreshRSS / Miniflux / Tiny Tiny RSS are heavier than necessary for a one-person feed
  reader and either ship their own DB or want a dedicated PHP/MySQL stack.
- Operator already has central PostgreSQL via PgBouncer; spinning up another DB would
  duplicate state and waste RAM.
- AI integration matters: OpenClaw OS needs to pull a daily news summary through a
  token-gated endpoint.

## Hard constraints (non-negotiable)
1. Must run inside `mem_limit: 256m` and stay below ~100 MB RSS in steady state.
2. No new database container — use `central-pgbouncer:6432` only.
3. No public ports — bind to `127.0.0.1:8088` for Cloudflare Tunnel.
4. Heavy frameworks (Spring, Django, Next.js SSR) forbidden.
5. RSS fetching must be staggered cron, not continuous polling, to keep CPU/RAM flat.

## Success criteria
- Operator opens `rss.<domain>` through Cloudflare Access and reads news offline-capable.
- Worker stays alive 24h+ with no PgBouncer pool exhaustion warnings.
- OpenClaw OS pulls `/api/v1/news/summary` with a bearer token and gets last-24h items
  per category.
- Telegram bot sends a single 08:00 daily digest plus immediate pushes for `is_critical`
  categories only.

## Out of scope (explicitly)
- Multi-user / user accounts — Cloudflare Access is the authenticator.
- Server-side full-text search beyond `ILIKE` — premature for one user.
- Article scraping beyond what RSS gives us (no readability, no headless browser).
- Mobile native apps — the PWA is the mobile story.
