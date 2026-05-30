<!-- memory-bank-schema: v1 -->
# Active Context

_Last updated: 2026-05-30._

## Status

**LIVE** — `9491a84` pushed to `main`; CI → GHCR → Watchtower redeploy expected.
Verify on iOS PWA after image rolls out.

## Immediate next steps

1. Confirm production picked up `9491a84` (Read tab + iOS polish).
2. Align repo `docker-compose.yml` + `INFRA_HANDOFF.md` with production (`pgbouncer:5432`, `postgres-shared-net`).
3. Optional: Telegram env, 24h soak (`scripts/soak-watch.sh`), `scripts/seed.sh`.

## Blockers

None.

## Hand-off pointers

- Ops: [`INFRA_HANDOFF.md`](../INFRA_HANDOFF.md)
- Deploy history: [lessons_learned.md](lessons_learned.md)
