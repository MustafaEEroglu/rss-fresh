<!-- memory-bank-schema: v1 -->
# Active Context

_Last updated: 2026-05-31._

## Status

**LIVE & COMPLETE** — Production on VPS (`7adc729`). Watchtower auto-redeploy. Telegram
(critical push + daily digest with saved articles). No open backlog.

## Currently working on

Nothing.

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
