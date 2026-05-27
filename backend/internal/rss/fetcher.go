// Package rss is the staggered cron RSS fetcher.
package rss

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/mmcdole/gofeed"

	"github.com/mustafaeeroglu/rss-fresh/internal/config"
	"github.com/mustafaeeroglu/rss-fresh/internal/db"
)

// Notifier is the contract for the Telegram critical-push hook.
// Implemented by internal/telegram.Notifier; nil-safe.
type Notifier interface {
	NotifyCritical(category db.Category, articles []db.Article)
}

type Fetcher struct {
	cfg      *config.Config
	db       *db.DB
	log      *slog.Logger
	notifier Notifier
	parser   *gofeed.Parser
	client   *http.Client
	mu       sync.Mutex // serialises ticks; we never run two ticks concurrently
}

func New(cfg *config.Config, database *db.DB, log *slog.Logger, n Notifier) *Fetcher {
	return &Fetcher{
		cfg:      cfg,
		db:       database,
		log:      log,
		notifier: n,
		parser:   gofeed.NewParser(),
		client: &http.Client{
			Timeout: cfg.FetchTimeout,
		},
	}
}

// Tick processes one batch of feeds. Designed to be called by gocron.
// Internally serialised so overlapping schedules don't double-fetch.
func (f *Fetcher) Tick(ctx context.Context) {
	if !f.mu.TryLock() {
		f.log.Warn("rss tick skipped: previous tick still running")
		return
	}
	defer f.mu.Unlock()

	feeds, err := f.db.PickFeedsForFetch(ctx, f.cfg.FetchBatchSize)
	if err != nil {
		f.log.Error("pick feeds", "err", err)
		return
	}
	if len(feeds) == 0 {
		f.log.Debug("rss tick: no feeds due")
		return
	}
	f.log.Info("rss tick", "batch", len(feeds))

	for _, feed := range feeds {
		if ctx.Err() != nil {
			return
		}
		f.fetchOne(ctx, feed)
	}
}

// RefreshFeed implements httpapi.Refresher: triggered by POST /feeds/:id/refresh.
func (f *Fetcher) RefreshFeed(feedID int64) {
	ctx, cancel := context.WithTimeout(context.Background(), f.cfg.FetchTimeout+5*time.Second)
	defer cancel()
	feed, err := f.db.GetFeed(ctx, feedID)
	if err != nil {
		f.log.Warn("refresh: feed lookup", "id", feedID, "err", err)
		return
	}
	f.fetchOne(ctx, *feed)
}

func (f *Fetcher) fetchOne(ctx context.Context, feed db.Feed) {
	log := f.log.With("feed_id", feed.ID, "url", feed.URL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feed.URL, nil)
	if err != nil {
		log.Error("build request", "err", err)
		_ = f.db.MarkFeedError(ctx, feed.ID, f.cfg.FeedDeactivateAfter)
		return
	}
	req.Header.Set("User-Agent", f.cfg.UserAgent)
	req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml, application/json, text/xml;q=0.9, */*;q=0.5")
	if feed.ETag != nil && *feed.ETag != "" {
		req.Header.Set("If-None-Match", *feed.ETag)
	}
	if feed.LastModified != nil && *feed.LastModified != "" {
		req.Header.Set("If-Modified-Since", *feed.LastModified)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		log.Warn("http error", "err", err)
		_ = f.db.MarkFeedError(ctx, feed.ID, f.cfg.FeedDeactivateAfter)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		log.Debug("not modified")
		_ = f.db.MarkFeedFetchOnly(ctx, feed.ID)
		return
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Warn("non-2xx", "status", resp.StatusCode)
		_ = f.db.MarkFeedError(ctx, feed.ID, f.cfg.FeedDeactivateAfter)
		return
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20)) // 8 MB cap per feed
	if err != nil {
		log.Warn("read body", "err", err)
		_ = f.db.MarkFeedError(ctx, feed.ID, f.cfg.FeedDeactivateAfter)
		return
	}

	parsed, err := f.parser.ParseString(string(body))
	if err != nil {
		log.Warn("parse", "err", err)
		_ = f.db.MarkFeedError(ctx, feed.ID, f.cfg.FeedDeactivateAfter)
		return
	}

	// If feed name was created equal to URL, upgrade to feed title on first parse.
	if feed.Name == feed.URL && parsed.Title != "" {
		_, _ = f.db.UpdateFeed(ctx, feed.ID, nil, &parsed.Title, nil, nil)
	}

	inserted := []db.Article{}
	for _, item := range parsed.Items {
		if item == nil {
			continue
		}
		guid := item.GUID
		if guid == "" {
			guid = item.Link
		}
		if guid == "" {
			continue
		}
		title := strings.TrimSpace(item.Title)
		if title == "" {
			title = "(untitled)"
		}
		var author *string
		if item.Author != nil && item.Author.Name != "" {
			a := item.Author.Name
			author = &a
		}
		var content *string
		if item.Content != "" {
			c := item.Content
			content = &c
		}
		var summary *string
		if item.Description != "" {
			s := item.Description
			summary = &s
		}
		published := item.PublishedParsed
		if published == nil {
			published = item.UpdatedParsed
		}
		id, ok, err := f.db.InsertArticle(ctx, feed.ID, guid, title, item.Link, author, content, summary, published)
		if err != nil {
			log.Warn("insert article", "guid", guid, "err", err)
			continue
		}
		if !ok {
			continue
		}
		// Build Article projection for notifier (lightweight).
		inserted = append(inserted, db.Article{
			ID:          id,
			FeedID:      feed.ID,
			FeedName:    parsed.Title,
			CategoryID:  feed.CategoryID,
			Title:       title,
			URL:         item.Link,
			Author:      author,
			Summary:     summary,
			PublishedAt: published,
		})
	}

	etag := resp.Header.Get("ETag")
	lastMod := resp.Header.Get("Last-Modified")
	var etagPtr, lmPtr *string
	if etag != "" {
		etagPtr = &etag
	}
	if lastMod != "" {
		lmPtr = &lastMod
	}
	if err := f.db.MarkFeedFetched(ctx, feed.ID, etagPtr, lmPtr); err != nil {
		log.Warn("mark fetched", "err", err)
	}

	if len(inserted) == 0 {
		log.Debug("no new articles")
		return
	}
	log.Info("new articles", "count", len(inserted))

	if f.notifier == nil {
		return
	}
	cat, err := f.db.GetCategory(ctx, feed.CategoryID)
	if err != nil {
		if !errors.Is(err, db.ErrNotFound) {
			log.Warn("notifier: category lookup", "err", err)
		}
		return
	}
	if cat.IsCritical {
		f.notifier.NotifyCritical(*cat, inserted)
	}
}
