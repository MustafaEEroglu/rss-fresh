package db

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type Article struct {
	ID            int64      `json:"id"`
	FeedID        int64      `json:"feed_id"`
	FeedName      string     `json:"feed_name"`
	CategoryID    int64      `json:"category_id"`
	CategorySlug  string     `json:"category_slug"`
	GUID          string     `json:"guid"`
	Title         string     `json:"title"`
	URL           string     `json:"url"`
	Author        *string    `json:"author,omitempty"`
	Content       *string    `json:"content,omitempty"`
	Summary       *string    `json:"summary,omitempty"`
	PublishedAt   *time.Time `json:"published_at"`
	FetchedAt     time.Time  `json:"fetched_at"`
	IsRead        bool       `json:"is_read"`
	IsSaved       bool       `json:"is_saved"`
}

type ListArticlesFilter struct {
	CategoryID *int64
	FeedID     *int64
	Unread     bool
	Read       bool
	Saved      bool
	Limit      int
	Cursor     string
	Since      *time.Time
}

const articleSelectCols = `
a.id, a.feed_id, f.name AS feed_name, f.category_id, c.slug AS category_slug,
a.guid, a.title, a.url, a.author, a.content, a.summary,
a.published_at, a.fetched_at, a.is_read, a.is_saved`

func (d *DB) ListArticles(ctx context.Context, filter ListArticlesFilter) ([]Article, string, error) {
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Limit > 200 {
		filter.Limit = 200
	}

	conds := []string{"1=1"}
	args := []any{}
	i := 1

	if filter.CategoryID != nil {
		conds = append(conds, "f.category_id = $"+strconv.Itoa(i))
		args = append(args, *filter.CategoryID)
		i++
	}
	if filter.FeedID != nil {
		conds = append(conds, "a.feed_id = $"+strconv.Itoa(i))
		args = append(args, *filter.FeedID)
		i++
	}
	if filter.Unread {
		conds = append(conds, "a.is_read = FALSE")
	}
	if filter.Read {
		conds = append(conds, "a.is_read = TRUE")
	}
	if filter.Saved {
		conds = append(conds, "a.is_saved = TRUE")
	}
	if filter.Since != nil {
		conds = append(conds, "a.published_at >= $"+strconv.Itoa(i))
		args = append(args, *filter.Since)
		i++
	}

	if filter.Cursor != "" {
		t, id, err := decodeCursor(filter.Cursor)
		if err == nil {
			conds = append(conds,
				"(a.published_at, a.id) < ($"+strconv.Itoa(i)+", $"+strconv.Itoa(i+1)+")")
			args = append(args, t, id)
			i += 2
		}
	}

	args = append(args, filter.Limit+1)
	q := fmt.Sprintf(`
SELECT %s
FROM articles a
JOIN feeds f ON f.id = a.feed_id
JOIN categories c ON c.id = f.category_id
WHERE %s
ORDER BY a.published_at DESC NULLS LAST, a.id DESC
LIMIT $%d`, articleSelectCols, strings.Join(conds, " AND "), i)

	rows, err := d.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	out := []Article{}
	for rows.Next() {
		a, err := scanArticle(rows)
		if err != nil {
			return nil, "", err
		}
		out = append(out, *a)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}

	next := ""
	if len(out) > filter.Limit {
		last := out[filter.Limit-1]
		out = out[:filter.Limit]
		if last.PublishedAt != nil {
			next = encodeCursor(*last.PublishedAt, last.ID)
		}
	}
	return out, next, nil
}

func (d *DB) GetArticle(ctx context.Context, id int64) (*Article, error) {
	q := `SELECT ` + articleSelectCols + `
		FROM articles a
		JOIN feeds f ON f.id = a.feed_id
		JOIN categories c ON c.id = f.category_id
		WHERE a.id = $1`
	row := d.Pool.QueryRow(ctx, q, id)
	a, err := scanArticleRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return a, err
}

func (d *DB) UpdateArticle(ctx context.Context, id int64, isRead, isSaved *bool) (*Article, error) {
	sets := []string{}
	args := []any{}
	i := 1
	if isRead != nil {
		sets = append(sets, "is_read = $"+strconv.Itoa(i))
		args = append(args, *isRead)
		i++
	}
	if isSaved != nil {
		sets = append(sets, "is_saved = $"+strconv.Itoa(i))
		args = append(args, *isSaved)
		i++
	}
	if len(sets) == 0 {
		return d.GetArticle(ctx, id)
	}
	args = append(args, id)
	_, err := d.Pool.Exec(ctx,
		"UPDATE articles SET "+strings.Join(sets, ", ")+" WHERE id = $"+strconv.Itoa(i),
		args...)
	if err != nil {
		return nil, err
	}
	return d.GetArticle(ctx, id)
}

func (d *DB) BulkMarkRead(ctx context.Context, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	tag, err := d.Pool.Exec(ctx,
		`UPDATE articles SET is_read = TRUE WHERE id = ANY($1)`, ids)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// InsertArticle dedups on (feed_id, guid). Returns the new ID and whether a row was
// actually inserted (false on conflict).
func (d *DB) InsertArticle(ctx context.Context, feedID int64, guid, title, url string,
	author, content, summary *string, publishedAt *time.Time) (int64, bool, error) {
	const q = `
INSERT INTO articles (feed_id, guid, title, url, author, content, summary, published_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (feed_id, guid) DO NOTHING
RETURNING id`
	var id int64
	err := d.Pool.QueryRow(ctx, q, feedID, guid, title, url, author, content, summary, publishedAt).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return id, true, nil
}

// DigestItem is a lightweight article projection for the Telegram digest.
type DigestItem struct {
	Title        string     `json:"title"`
	URL          string     `json:"url"`
	Summary      *string    `json:"summary,omitempty"`
	CategorySlug string     `json:"category_slug"`
	FeedName     string     `json:"feed_name"`
	PublishedAt  *time.Time `json:"published_at"`
}

// PurgeReadArticles deletes read, non-saved articles older than the retention window.
// Age is measured from COALESCE(published_at, fetched_at). Unread and saved rows are kept.
func (d *DB) PurgeReadArticles(ctx context.Context, olderThanDays int) (int64, error) {
	if olderThanDays <= 0 {
		return 0, nil
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -olderThanDays)
	tag, err := d.Pool.Exec(ctx, `
DELETE FROM articles
WHERE is_read = TRUE
  AND is_saved = FALSE
  AND COALESCE(published_at, fetched_at) < $1`, cutoff)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// SavedArticlesSince returns saved articles from the given time forward, for digest inclusion.
func (d *DB) SavedArticlesSince(ctx context.Context, since time.Time, limit int) ([]DigestItem, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := d.Pool.Query(ctx, `
SELECT a.title, a.url, a.summary, c.slug, f.name, a.published_at
FROM articles a
JOIN feeds f ON f.id = a.feed_id
JOIN categories c ON c.id = f.category_id
WHERE a.is_saved = TRUE
  AND COALESCE(a.published_at, a.fetched_at) >= $1
ORDER BY a.published_at DESC NULLS LAST
LIMIT $2`, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DigestItem{}
	for rows.Next() {
		var s DigestItem
		if err := rows.Scan(&s.Title, &s.URL, &s.Summary, &s.CategorySlug, &s.FeedName, &s.PublishedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// UnreadCountsByCategory returns a map of category_id -> unread article count.
func (d *DB) UnreadCountsByCategory(ctx context.Context) (map[int64]int, error) {
	rows, err := d.Pool.Query(ctx, `
SELECT f.category_id, COUNT(*)
FROM articles a JOIN feeds f ON f.id = a.feed_id
WHERE a.is_read = FALSE
GROUP BY f.category_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]int{}
	for rows.Next() {
		var k int64
		var v int
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, rows.Err()
}

func scanArticle(rows pgx.Rows) (*Article, error) {
	var a Article
	if err := rows.Scan(
		&a.ID, &a.FeedID, &a.FeedName, &a.CategoryID, &a.CategorySlug,
		&a.GUID, &a.Title, &a.URL, &a.Author, &a.Content, &a.Summary,
		&a.PublishedAt, &a.FetchedAt, &a.IsRead, &a.IsSaved,
	); err != nil {
		return nil, err
	}
	return &a, nil
}

func scanArticleRow(row pgx.Row) (*Article, error) {
	var a Article
	if err := row.Scan(
		&a.ID, &a.FeedID, &a.FeedName, &a.CategoryID, &a.CategorySlug,
		&a.GUID, &a.Title, &a.URL, &a.Author, &a.Content, &a.Summary,
		&a.PublishedAt, &a.FetchedAt, &a.IsRead, &a.IsSaved,
	); err != nil {
		return nil, err
	}
	return &a, nil
}

func encodeCursor(t time.Time, id int64) string {
	s := t.UTC().Format(time.RFC3339Nano) + "|" + strconv.FormatInt(id, 10)
	return base64.RawURLEncoding.EncodeToString([]byte(s))
}

func decodeCursor(c string) (time.Time, int64, error) {
	raw, err := base64.RawURLEncoding.DecodeString(c)
	if err != nil {
		return time.Time{}, 0, err
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, 0, errors.New("bad cursor")
	}
	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, 0, err
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return time.Time{}, 0, err
	}
	return t, id, nil
}
