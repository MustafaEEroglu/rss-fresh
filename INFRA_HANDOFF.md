# INFRA_HANDOFF — rss-fresh

This document is the contract between the application (Go binary + SPA) and the
deployment infrastructure (Hetzner VPS + Cloudflare Tunnel + Docker + Watchtower
+ GHCR). Treat the infra as immutable; only the items below are app-specific.

## 1. Container shape

| Field | Value |
|---|---|
| Image (built by CI) | `ghcr.io/mustafaeeroglu/rss-fresh:latest` |
| Container name | `rss-fresh` |
| Internal port | `3000` (HTTP) |
| Host bind | `127.0.0.1:8088` (loopback only — Cloudflare Tunnel reaches it) |
| Health probe (in-image) | `/app/rss-fresh healthcheck` (returns 0 if `/api/v1/healthz` is 200) |
| User | `nonroot` (distroless) |
| Filesystem | `read_only: true` |
| Memory limit | `256m`, reservation `64m` |
| CPU limit | `0.5` |
| Caps dropped | `ALL`, `no-new-privileges` |
| Auto-redeploy label | `com.centurylinklabs.watchtower.enable=true` |
| Log driver | `json-file`, 10 MB × 3 files |

## 2. Environment variables

| Variable | Required | Notes |
|---|---|---|
| `DATABASE_URL` | yes | Must include `default_query_exec_mode=exec` for PgBouncer transaction mode. |
| `OPENCLAW_GATEWAY_TOKEN` | yes | 64+ random chars; gates `/api/v1/news/summary`. |
| `OPENCLAW_WEBHOOK_URL` | optional | OpenClaw webhook endpoint for critical-category push. Empty = push disabled. |
| `DB_PASSWORD` | yes | Substituted into `DATABASE_URL` from `.env`. |
| `TELEGRAM_BOT_TOKEN` | optional | Telegram used for daily digest only; critical push moved to OpenClaw. |
| `TELEGRAM_CHAT_ID` | optional | int64; required if bot token is set. |
| `FETCH_CRON` | optional | default `*/15 * * * *`. |
| `FETCH_BATCH_SIZE` | optional | default `10`. |
| `FETCH_TIMEOUT_SECONDS` | optional | per-feed HTTP timeout, default `20`. |
| `DIGEST_CRON` | optional | default `0 8 * * *` in `TZ`. |
| `RETENTION_DAYS` | optional | default `30`. Read, non-saved articles older than this are deleted. Set `0` to disable. |
| `RETENTION_CRON` | optional | default `0 4 * * *` in `TZ`. |
| `TZ` | optional | default `Europe/Istanbul`. |
| `TRUSTED_PROXIES` | optional | default `0.0.0.0/0` (we sit behind Cloudflare Tunnel). |
| `LOG_LEVEL` | optional | `debug`, `info`, `warn`, `error`. |

## 3. External dependencies

- **PostgreSQL**: `central-pgbouncer:6432`, database `rss_fresh`, user `rss_user`.
  Network: `central-postgres-net` (external, pre-existing).
- **Telegram API** (egress): `https://api.telegram.org`.
- **RSS source feeds** (egress): operator-supplied via the UI; arbitrary HTTPS hosts.

## 4. Pre-deployment checklist

1. Provision the database user and database (idempotent — skip if exists):

   ```sh
   psql -h central-pgbouncer -p 6432 -U postgres <<'SQL'
   CREATE USER rss_user WITH PASSWORD '<strong-password>';
   CREATE DATABASE rss_fresh OWNER rss_user;
   GRANT ALL PRIVILEGES ON DATABASE rss_fresh TO rss_user;
   SQL
   ```

2. Apply schema (idempotent — the binary also self-applies on boot, but doing
   it manually first is safer for verification):

   ```sh
   psql -h central-pgbouncer -p 6432 -U rss_user -d rss_fresh \
     -f backend/migrations/init.sql
   ```

3. Cloudflare Tunnel route — in the Cloudflare Zero Trust dashboard:
   **Networks → Tunnels → (your existing tunnel) → Public Hostnames →
   Add a hostname**. Set:

   - Subdomain: `rss-fresh` (or your preferred name)
   - Domain: `mustafaeroglu.me`
   - Type: HTTP
   - URL: `localhost:8088`

   Then under **Access → Applications**, attach a self-hosted application
   policy to `rss-fresh.mustafaeroglu.me` that allows only your identity
   email (`mustafaeeroglu@icloud.com`).

   The OpenClaw OS endpoint is reached at the **same** hostname and is gated
   by the bearer token; if your Access policy blocks bot-style requests for
   `/api/v1/news/summary`, add a path bypass or a Service Token under
   Access → Service Auth so OpenClaw can call it server-to-server.

4. Drop a populated `.env` next to `docker-compose.yml` on the server (use
   `.env.example` as the template).

5. Add an **Uptime Kuma monitor** at `https://status.mustafaeroglu.me`
   pointing at `https://rss-fresh.mustafaeroglu.me/api/v1/healthz`
   (HTTP keyword match: `"status":"ok"`).

## 5. First boot

```sh
cd /opt/rss-fresh
docker compose pull
docker compose up -d
docker compose logs -f rss-fresh
```

Within ~30 seconds the healthcheck should pass. Verify:

```sh
curl -s http://127.0.0.1:8088/api/v1/healthz | jq .
# → { "status": "ok", "version": "<sha>", "uptime_seconds": ... }
```

## 6. Smoke tests

```sh
# Categories
curl -s http://127.0.0.1:8088/api/v1/categories
curl -s -X POST http://127.0.0.1:8088/api/v1/categories \
  -H 'Content-Type: application/json' \
  -d '{"name":"AI","is_critical":false}'

# Feed
curl -s -X POST http://127.0.0.1:8088/api/v1/feeds \
  -H 'Content-Type: application/json' \
  -d '{"category_id":1,"url":"https://news.ycombinator.com/rss"}'

# OpenClaw endpoint (unauth — must 401)
curl -s -o /dev/null -w '%{http_code}\n' \
  http://127.0.0.1:8088/api/v1/news/summary  # → 401

# OpenClaw endpoint (auth — must 200)
curl -s http://127.0.0.1:8088/api/v1/news/summary \
  -H "Authorization: Bearer $OPENCLAW_GATEWAY_TOKEN" | jq .
```

## 7. Auto-redeploy

Watchtower watches the image label `com.centurylinklabs.watchtower.enable=true`
and pulls + restarts on every new GHCR push. CI (`.github/workflows/deploy.yml`)
pushes `:latest` after green tests on `main`.

## 8. Rollback

```sh
docker pull ghcr.io/mustafaeeroglu/rss-fresh:<previous-sha>
docker tag ghcr.io/mustafaeeroglu/rss-fresh:<previous-sha> \
           ghcr.io/mustafaeeroglu/rss-fresh:latest
docker compose up -d
```

(Or pin the image tag in `docker-compose.yml` to the known-good SHA and
`docker compose up -d`.)

## 9. Resource expectations

- **Idle RAM**: 30–50 MB (Go binary + pgxpool + ~100 cached articles in memory
  at most during a fetch tick).
- **Peak RAM** during a 10-feed fetch tick: < 100 MB.
- **CPU**: < 5% on the 256 MB / 0.5 CPU cap, except briefly during a tick.
- **Image size**: ~25 MB (distroless static + Go binary + embedded SPA).

If RSS climbs past 200 MB or PgBouncer logs `pool size exceeded`, that's an
alert: lower `FETCH_BATCH_SIZE` or `pool_max_conns`.
