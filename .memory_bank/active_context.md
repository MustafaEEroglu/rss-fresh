<!-- memory-bank-schema: v1 -->
# Active Context

_Last updated: 2026-05-30._

## Status

**LIVE** — app on VPS (`rss-fresh` container healthy). Latest app commit on `main`: **`65ca785`**
(refresh UX + mobile sidebar fix). Prior: `9491a84` (Read tab + iOS polish).

**Auto-redeploy broken:** Watchtower container **not running** on the host (only unused
`containrrr/watchtower:latest` image). GHCR pushes do **not** auto-update production until
management stack is fixed.

## Immediate next steps

1. **Deploy Option A management stack** — unified `~/projects/management/docker-compose.yml`
   (portainer + uptime-kuma + watchtower; `WATCHTOWER_LABEL_ENABLE=true`).
2. Until Watchtower runs: manual deploy after each push —
   `cd ~/projects/rss-fresh && docker compose pull && docker compose up -d`.
3. Verify production has `65ca785` UX (refresh spinner, ☰ sidebar, Read tab).
4. Align repo `docker-compose.yml` + `INFRA_HANDOFF.md` with production DB/network names.

## Blockers

- Watchtower absent → no automatic image pull for `rss-fresh`.

## Hand-off

- Ops: [`INFRA_HANDOFF.md`](../INFRA_HANDOFF.md)
- Infra layout: [system_architecture.md](system_architecture.md) § Management plane
