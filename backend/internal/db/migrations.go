package db

import (
	"context"
	"fmt"
)

// initSchemaSQL is the canonical schema. Kept in lock-step with
// `backend/migrations/init.sql` (which exists for ops `psql -f` runs).
// Idempotent: every CREATE uses IF NOT EXISTS.
const initSchemaSQL = `
CREATE TABLE IF NOT EXISTS categories (
  id          BIGSERIAL PRIMARY KEY,
  name        TEXT NOT NULL,
  slug        TEXT NOT NULL UNIQUE,
  is_critical BOOLEAN NOT NULL DEFAULT FALSE,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS feeds (
  id              BIGSERIAL PRIMARY KEY,
  category_id     BIGINT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
  name            TEXT NOT NULL,
  url             TEXT NOT NULL UNIQUE,
  etag            TEXT,
  last_modified   TEXT,
  last_fetched_at TIMESTAMPTZ,
  error_count     INTEGER NOT NULL DEFAULT 0,
  is_active       BOOLEAN NOT NULL DEFAULT TRUE,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_feeds_category ON feeds(category_id);
CREATE INDEX IF NOT EXISTS idx_feeds_active   ON feeds(is_active, last_fetched_at);

CREATE TABLE IF NOT EXISTS articles (
  id           BIGSERIAL PRIMARY KEY,
  feed_id      BIGINT NOT NULL REFERENCES feeds(id) ON DELETE CASCADE,
  guid         TEXT NOT NULL,
  title        TEXT NOT NULL,
  url          TEXT NOT NULL,
  author       TEXT,
  content      TEXT,
  summary      TEXT,
  published_at TIMESTAMPTZ,
  fetched_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  is_read      BOOLEAN NOT NULL DEFAULT FALSE,
  is_saved     BOOLEAN NOT NULL DEFAULT FALSE,
  UNIQUE (feed_id, guid)
);
CREATE INDEX IF NOT EXISTS idx_articles_feed_published ON articles(feed_id, published_at DESC);
CREATE INDEX IF NOT EXISTS idx_articles_unread ON articles(is_read, published_at DESC) WHERE is_read = FALSE;
CREATE INDEX IF NOT EXISTS idx_articles_saved  ON articles(is_saved, published_at DESC) WHERE is_saved = TRUE;
`

// EnsureSchema applies the schema idempotently. Safe to call on every boot.
func (d *DB) EnsureSchema(ctx context.Context) error {
	if _, err := d.Pool.Exec(ctx, initSchemaSQL); err != nil {
		return fmt.Errorf("ensure schema: %w", err)
	}
	return nil
}
