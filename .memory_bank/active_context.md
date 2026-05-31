<!-- memory-bank-schema: v1 -->
# Active Context

_Last updated: 2026-05-31._

## Status

**LIVE & COMPLETE** — Production on VPS. Code quality pass completed this session (strict review + all fixes applied, all tests green).

## Currently working on

Nothing active.

## Known open issues (non-blocking)

- `markAllReadInView` only marks the ≤50 loaded articles — server-side "mark all" endpoint needed.
- Pagination cursor dropped silently when last article has `null published_at`.
- Dexie never evicts articles purged server-side by retention (stale offline cache).
- `escapeHTML` does not escape `"` — safe now, watch if URLs go into `<a href>` attributes.
- Telegram queue cap 64 — bursts of 330+ articles silently drop messages.

## Production snapshot

| Area | State |
|------|--------|
| App | `ghcr.io/mustafaeeroglu/rss-fresh:latest` on `127.0.0.1:8088` |
| DB | `central-pgbouncer:6432` / `rss_fresh` via `central-postgres-net` |
| Notifications | Telegram env set on VPS |
| Deploy | Push `main` → CI → GHCR → Watchtower |
| Smoke test | `./scripts/verify-deploy.sh` |

## Hand-off

- Ops: [`INFRA_HANDOFF.md`](../INFRA_HANDOFF.md)
- Retention: [system_architecture.md](system_architecture.md) § Retention
