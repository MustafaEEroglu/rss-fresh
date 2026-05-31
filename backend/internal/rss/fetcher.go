// Package rss is the staggered cron RSS fetcher.
package rss

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/mmcdole/gofeed"

	"github.com/mustafaeeroglu/rss-fresh/internal/config"
	"github.com/mustafaeeroglu/rss-fresh/internal/db"
)

// privateIPDialer returns a DialContext function that rejects connections to
// loopback, private, and link-local addresses. This is a defence-in-depth
// guard against DNS rebinding: even if the URL passes the creation-time
// validation, a rebind cannot redirect the fetcher to an internal host.
func privateIPDialer(timeout time.Duration) func(ctx context.Context, network, addr string) (net.Conn, error) {
	d := &net.Dialer{Timeout: timeout}
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		ips, err := net.DefaultResolver.LookupHost(ctx, host)
		if err != nil {
			return nil, err
		}
		for _, ipStr := range ips {
			ip := net.ParseIP(ipStr)
			if ip == nil {
				continue
			}
			if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
				return nil, fmt.Errorf("ssrf: host %s resolves to disallowed address %s", host, ip)
			}
		}
		return d.DialContext(ctx, network, net.JoinHostPort(ips[0], port))
	}
}

// Notifier is the contract for the Telegram critical-push hook.
// Implemented by internal/telegram; the nopNotifier satisfies it when Telegram is disabled.
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
	dialTimeout := 10 * time.Second
	return &Fetcher{
		cfg:      cfg,
		db:       database,
		log:      log,
		notifier: n,
		parser:   gofeed.NewParser(),
		client: &http.Client{
			Timeout: cfg.FetchTimeout,
			Transport: &http.Transport{
				DialContext:           privateIPDialer(dialTimeout),
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: cfg.FetchTimeout,
				MaxIdleConnsPerHost:   2,
			},
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
// ctx should be derived from the root context (not the request context) so that
// the fetch outlives the HTTP response but still respects the shutdown signal.
func (f *Fetcher) RefreshFeed(ctx context.Context, feedID int64) {
	ctx, cancel := context.WithTimeout(ctx, f.cfg.FetchTimeout+5*time.Second)
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
	skippedOld := 0
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
		// Skip articles published before the feed was added to avoid backfilling
		// the full RSS archive. Items without a publish date are allowed through.
		if published != nil && published.Before(feed.CreatedAt) {
			skippedOld++
			continue
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

	if skippedOld > 0 {
		log.Debug("skipped pre-subscription articles", "count", skippedOld, "feed_created", feed.CreatedAt)
	}

	if len(inserted) == 0 {
		log.Debug("no new articles")
		return
	}
	log.Info("new articles", "count", len(inserted))

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
