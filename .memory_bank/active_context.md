<!-- memory-bank-schema: v1 -->
# Active Context

_Last updated: 2026-05-31._

## Status

**LIVE** — Watchtower auto-redeploy active. Telegram-only notifications (critical + digest).

## Currently working on

Nothing active.

## Shipped (this release)

- Feed ingest cutoff (`published_at < feed.created_at` skipped)
- Saved articles in Telegram daily digest (24h window)
- OpenClaw fully removed — no summary API, no bearer token, no webhook env

## Operator follow-up (VPS)

1. Remove stale `OPENCLAW_*` keys from server `.env` if present.
2. Set `TELEGRAM_BOT_TOKEN` + `TELEGRAM_CHAT_ID` if notifications are wanted.
3. Run `./scripts/verify-deploy.sh` after Watchtower redeploys.

## Hand-off

- Ops: [`INFRA_HANDOFF.md`](../INFRA_HANDOFF.md)
- Retention rules: [system_architecture.md](system_architecture.md) § Retention
