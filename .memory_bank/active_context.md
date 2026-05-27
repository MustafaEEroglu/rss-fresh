# Active Context — RSS-Fresh

_Last updated: 2026-05-27 — production deploy verified on Hetzner._

## Status

**LIVE** — container running on `mustafaeroglu` VPS. UI gated by **Cloudflare Access**
(single-operator email). Health endpoint returns 200 behind the tunnel.

## Production snapshot (source of truth for ops)

| Item | Value |
|------|--------|
| Server path | `~/projects/rss-fresh` |
| Container | `rss-fresh` |
| Image | `ghcr.io/mustafaeeroglu/rss-fresh:latest` (built via `MustafaEEroglu/shared-workflows` → `docker-build.yml@main`) |
| Host bind | `127.0.0.1:8088` → container `:3000` |
| Docker network | External **`postgres-shared-net`** (compose key `central-postgres-net`, `name: postgres-shared-net`) |
| DB host (in-network) | **`pgbouncer:5432`** — not `:6432` (6432 is host→container map only) |
| DB name / user | `rss_fresh` / `rss_user` |
| Postgres container | `central-postgres` (same network) |
| Adminer | `postgres-adminer` |
| Auth (UI) | Cloudflare Zero Trust Access — allow policy for operator email only |
| Auth (OpenClaw) | `OPENCLAW_GATEWAY_TOKEN` on `GET /api/v1/news/summary` |
| Telegram | Disabled until `TELEGRAM_BOT_TOKEN` + `TELEGRAM_CHAT_ID` set (warn-only) |

### `DATABASE_URL` (production shape)

```text
postgres://rss_user:${DB_PASSWORD}@pgbouncer:5432/rss_fresh?sslmode=disable&pool_max_conns=4&default_query_exec_mode=exec
```

### Repo vs server drift

The committed [`docker-compose.yml`](../docker-compose.yml) still documents
`central-pgbouncer:6432` and `central-postgres-net` without `name:` override.
**The running server compose was corrected manually.** Next code change should align
the repo file with the production table above so a fresh `docker compose up` matches
what works on the VPS.

## Epic checklist (all complete)

### Epic 0 — Foundation [DONE]
### Epic 1 — Backend & worker [DONE]
### Epic 2 — Frontend PWA [DONE]
### Epic 3 — Deploy [DONE]
### Epic 4 — Polish & verify [DONE]

**Production verification (operator, 2026-05-27):**
- [x] GitHub repo `MustafaEEroglu/rss-fresh`, CI + GHCR image push.
- [x] `docker compose up` on VPS after network + port + ownership fixes.
- [x] `GET /api/v1/healthz` → 200.
- [x] Cloudflare Access restricts UI to operator.
- [ ] Telegram notifications (optional — env not set).
- [ ] 24h RAM soak via `scripts/soak-watch.sh` (recommended, not yet recorded).
- [ ] Run `scripts/seed.sh` for starter feeds (optional).

## Active blockers

None.

## Active errors

None.

## Next session — sensible follow-ups

1. **Align repo `docker-compose.yml`** with production (`pgbouncer:5432`, `postgres-shared-net`).
2. **Align `INFRA_HANDOFF.md`** with same names (replace `central-pgbouncer` / `central-postgres-net` placeholders).
3. Commit **`backend/go.sum`** if not already on `main` (CI requires it).
4. Add Telegram env vars when bot is ready.
5. OpenClaw: confirm Access **Service Auth** or path bypass if machine clients cannot pass Access JWT.

## Hand-off

- Ops runbook: [`INFRA_HANDOFF.md`](../INFRA_HANDOFF.md) (update host/port names when compose is synced).
- Plan file (historical): `rss-fresh_personal_reader_38cf5e8d.plan.md`.
- Deploy troubleshooting history: [lessons_learned.md](lessons_learned.md) § Production deploy 2026-05-27.
