<!-- memory-bank-schema: v1 -->
# Active Context

_Last updated: 2026-05-30._

## Status

**LIVE** — latest app commits include retention cron (`RETENTION_DAYS=30`). Deploy via
manual `docker compose pull/up` until Watchtower runs on management stack.

## Immediate next steps

1. Deploy retention change to VPS after CI push.
2. Stand up Option A management stack (portainer + uptime-kuma + watchtower).
3. Align repo `docker-compose.yml` DB/network names with production.

## Blockers

- Watchtower not running → no automatic image pull.

## Hand-off

- Ops: [`INFRA_HANDOFF.md`](../INFRA_HANDOFF.md)
- Retention rules: [system_architecture.md](system_architecture.md) § Retention
