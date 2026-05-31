package db

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type Category struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	IsCritical  bool      `json:"is_critical"`
	CreatedAt   time.Time `json:"created_at"`
	FeedCount   int       `json:"feed_count"`
	UnreadCount int       `json:"unread_count"`
}

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("conflict")

func (d *DB) ListCategoriesWithCounts(ctx context.Context) ([]Category, error) {
	const q = `
SELECT c.id, c.name, c.slug, c.is_critical, c.created_at,
       COALESCE(fc.feed_count, 0)   AS feed_count,
       COALESCE(uc.unread_count, 0) AS unread_count
FROM categories c
LEFT JOIN (
  SELECT category_id, COUNT(*) AS feed_count
  FROM feeds GROUP BY category_id
) fc ON fc.category_id = c.id
LEFT JOIN (
  SELECT f.category_id, COUNT(*) AS unread_count
  FROM articles a JOIN feeds f ON f.id = a.feed_id
  WHERE a.is_read = FALSE
  GROUP BY f.category_id
) uc ON uc.category_id = c.id
ORDER BY c.name ASC`
	rows, err := d.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Category{}
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.IsCritical, &c.CreatedAt, &c.FeedCount, &c.UnreadCount); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (d *DB) GetCategory(ctx context.Context, id int64) (*Category, error) {
	const q = `SELECT id, name, slug, is_critical, created_at FROM categories WHERE id = $1`
	var c Category
	err := d.pool.QueryRow(ctx, q, id).Scan(&c.ID, &c.Name, &c.Slug, &c.IsCritical, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (d *DB) CreateCategory(ctx context.Context, name, slug string, isCritical bool) (*Category, error) {
	const q = `
INSERT INTO categories (name, slug, is_critical)
VALUES ($1, $2, $3)
RETURNING id, name, slug, is_critical, created_at`
	var c Category
	err := d.pool.QueryRow(ctx, q, name, slug, isCritical).Scan(&c.ID, &c.Name, &c.Slug, &c.IsCritical, &c.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, err
	}
	return &c, nil
}

func (d *DB) UpdateCategory(ctx context.Context, id int64, name, slug *string, isCritical *bool) (*Category, error) {
	sets := []string{}
	args := []any{}
	i := 1
	if name != nil {
		sets = append(sets, "name = $"+itoa(i))
		args = append(args, *name)
		i++
	}
	if slug != nil {
		sets = append(sets, "slug = $"+itoa(i))
		args = append(args, *slug)
		i++
	}
	if isCritical != nil {
		sets = append(sets, "is_critical = $"+itoa(i))
		args = append(args, *isCritical)
		i++
	}
	if len(sets) == 0 {
		return d.GetCategory(ctx, id)
	}
	args = append(args, id)
	q := "UPDATE categories SET " + strings.Join(sets, ", ") + " WHERE id = $" + itoa(i) +
		" RETURNING id, name, slug, is_critical, created_at"
	var c Category
	err := d.pool.QueryRow(ctx, q, args...).Scan(&c.ID, &c.Name, &c.Slug, &c.IsCritical, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, err
	}
	return &c, nil
}

func (d *DB) DeleteCategory(ctx context.Context, id int64) error {
	tag, err := d.pool.Exec(ctx, `DELETE FROM categories WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "23505")
}

func itoa(n int) string { return strconv.Itoa(n) }
