// Package retention runs scheduled cleanup of old read articles.
package retention

import (
	"context"
	"log/slog"

	"github.com/mustafaeeroglu/rss-fresh/internal/db"
)

type Purger struct {
	db   *db.DB
	log  *slog.Logger
	days int
}

func New(database *db.DB, log *slog.Logger, retentionDays int) *Purger {
	return &Purger{db: database, log: log, days: retentionDays}
}

// Tick deletes read articles older than the configured retention window.
func (p *Purger) Tick(ctx context.Context) {
	if p.days <= 0 {
		p.log.Debug("retention tick skipped: disabled")
		return
	}
	n, err := p.db.PurgeReadArticles(ctx, p.days)
	if err != nil {
		p.log.Error("retention purge", "err", err, "days", p.days)
		return
	}
	if n > 0 {
		p.log.Info("retention purge", "deleted", n, "days", p.days)
	} else {
		p.log.Debug("retention purge", "deleted", 0, "days", p.days)
	}
}
