// Package openclaw pushes critical-category articles to an OpenClaw OS webhook.
package openclaw

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/mustafaeeroglu/rss-fresh/internal/db"
)

type pushPayload struct {
	Category string        `json:"category"`
	Articles []pushArticle `json:"articles"`
}

type pushArticle struct {
	ID          int64      `json:"id"`
	Title       string     `json:"title"`
	URL         string     `json:"url"`
	FeedName    string     `json:"feed_name"`
	Author      *string    `json:"author,omitempty"`
	Summary     *string    `json:"summary,omitempty"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
}

// Notifier sends critical-category articles to an OpenClaw webhook endpoint.
// Nil-safe: callers can hold a nil *Notifier and call methods without panicking.
type Notifier struct {
	webhookURL string
	log        *slog.Logger
	client     *http.Client
}

// New returns a Notifier. If webhookURL is empty, returns nil (critical push disabled).
func New(webhookURL string, log *slog.Logger) *Notifier {
	if webhookURL == "" {
		log.Warn("openclaw critical push disabled (OPENCLAW_WEBHOOK_URL not set)")
		return nil
	}
	return &Notifier{
		webhookURL: webhookURL,
		log:        log,
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

// NotifyCritical POSTs the articles to the OpenClaw webhook. Best-effort: logs
// errors but never blocks the fetcher.
func (n *Notifier) NotifyCritical(cat db.Category, articles []db.Article) {
	if n == nil || len(articles) == 0 {
		return
	}

	payload := pushPayload{
		Category: cat.Name,
		Articles: make([]pushArticle, len(articles)),
	}
	for i, a := range articles {
		payload.Articles[i] = pushArticle{
			ID:          a.ID,
			Title:       a.Title,
			URL:         a.URL,
			FeedName:    a.FeedName,
			Author:      a.Author,
			Summary:     a.Summary,
			PublishedAt: a.PublishedAt,
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		n.log.Error("openclaw: marshal payload", "err", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.webhookURL, bytes.NewReader(body))
	if err != nil {
		n.log.Error("openclaw: build request", "err", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		n.log.Warn("openclaw: push failed", "err", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		n.log.Warn("openclaw: push non-2xx", "status", resp.StatusCode)
		return
	}

	n.log.Info("openclaw: pushed critical articles",
		"category", cat.Name,
		"count", len(articles),
	)
}

// Close is a no-op; satisfies symmetry with telegram.Notifier.
func (n *Notifier) Close() {}
