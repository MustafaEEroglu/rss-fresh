package db

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type Feed struct {
	ID            int64      `json:"id"`
	CategoryID    int64      `json:"category_id"`
	Name          string     `json:"name"`
	URL           string     `json:"url"`
	ETag          *string    `json:"-"`
	LastModified  *string    `json:"-"`
	LastFetchedAt *time.Time `json:"last_fetched_at"`
	ErrorCount    int        `json:"error_count"`
	IsActive      bool       `json:"is_active"`
	CreatedAt     time.Time  `json:"created_at"`
}

const feedSelectCols = `id, category_id, name, url, etag, last_modified, last_fetched_at, error_count, is_active, created_at`

func scanFeed(s scanner) (Feed, error) {
	var f Feed
	err := s.Scan(&f.ID, &f.CategoryID, &f.Name, &f.URL, &f.ETag, &f.LastModified,
		&f.LastFetchedAt, &f.ErrorCount, &f.IsActive, &f.CreatedAt)
	return f, err
}

func (d *DB) ListFeeds(ctx context.Context, categoryID *int64) ([]Feed, error) {
	q := `SELECT ` + feedSelectCols + ` FROM feeds`
	args := []any{}
	if categoryID != nil {
		q += ` WHERE category_id = $1`
		args = append(args, *categoryID)
	}
	q += ` ORDER BY name ASC`

	rows, err := d.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Feed{}
	for rows.Next() {
		f, err := scanFeed(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (d *DB) GetFeed(ctx context.Context, id int64) (*Feed, error) {
	f, err := scanFeed(d.pool.QueryRow(ctx,
		`SELECT `+feedSelectCols+` FROM feeds WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (d *DB) CreateFeed(ctx context.Context, categoryID int64, name, url string) (*Feed, error) {
	if name == "" {
		name = url
	}
	f, err := scanFeed(d.pool.QueryRow(ctx,
		`INSERT INTO feeds (category_id, name, url) VALUES ($1, $2, $3) RETURNING `+feedSelectCols,
		categoryID, name, url))
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, err
	}
	return &f, nil
}

func (d *DB) UpdateFeed(ctx context.Context, id int64, categoryID *int64, name, url *string, isActive *bool) (*Feed, error) {
	sets := []string{}
	args := []any{}
	i := 1
	if categoryID != nil {
		sets = append(sets, "category_id = $"+itoa(i))
		args = append(args, *categoryID)
		i++
	}
	if name != nil {
		sets = append(sets, "name = $"+itoa(i))
		args = append(args, *name)
		i++
	}
	if url != nil {
		sets = append(sets, "url = $"+itoa(i))
		args = append(args, *url)
		i++
	}
	if isActive != nil {
		sets = append(sets, "is_active = $"+itoa(i))
		args = append(args, *isActive)
		i++
	}
	if len(sets) == 0 {
		return d.GetFeed(ctx, id)
	}
	args = append(args, id)
	q := "UPDATE feeds SET " + strings.Join(sets, ", ") + " WHERE id = $" + itoa(i) +
		" RETURNING " + feedSelectCols
	f, err := scanFeed(d.pool.QueryRow(ctx, q, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, err
	}
	return &f, nil
}

func (d *DB) DeleteFeed(ctx context.Context, id int64) error {
	tag, err := d.pool.Exec(ctx, `DELETE FROM feeds WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// PickFeedsForFetch returns the n active feeds with the oldest last_fetched_at.
// NULLs come first so brand-new feeds are fetched immediately on the next tick.
func (d *DB) PickFeedsForFetch(ctx context.Context, n int) ([]Feed, error) {
	rows, err := d.pool.Query(ctx,
		`SELECT `+feedSelectCols+`
		 FROM feeds
		 WHERE is_active = TRUE
		 ORDER BY last_fetched_at NULLS FIRST
		 LIMIT $1`, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Feed{}
	for rows.Next() {
		f, err := scanFeed(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// MarkFeedFetched updates the etag/last-modified/last_fetched_at on a successful fetch
// and resets error_count.
func (d *DB) MarkFeedFetched(ctx context.Context, id int64, etag, lastModified *string) error {
	_, err := d.pool.Exec(ctx,
		`UPDATE feeds
		 SET etag = $2, last_modified = $3, last_fetched_at = NOW(), error_count = 0
		 WHERE id = $1`, id, etag, lastModified)
	return err
}

// MarkFeedFetchOnly bumps last_fetched_at without changing etag/last_modified
// (used after 304 Not Modified, where headers may or may not change).
func (d *DB) MarkFeedFetchOnly(ctx context.Context, id int64) error {
	_, err := d.pool.Exec(ctx,
		`UPDATE feeds SET last_fetched_at = NOW(), error_count = 0 WHERE id = $1`, id)
	return err
}

// MarkFeedError increments error_count and deactivates the feed if it crosses the threshold.
func (d *DB) MarkFeedError(ctx context.Context, id int64, deactivateAfter int) error {
	_, err := d.pool.Exec(ctx,
		`UPDATE feeds
		 SET error_count = error_count + 1,
		     is_active = CASE WHEN error_count + 1 >= $2 THEN FALSE ELSE is_active END,
		     last_fetched_at = NOW()
		 WHERE id = $1`, id, deactivateAfter)
	return err
}
