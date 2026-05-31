// Package telegram is the notification module: critical-category pushes plus
// the daily digest. Both pass through a single throttled send queue.
package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	tgbot "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/mustafaeeroglu/rss-fresh/internal/config"
	"github.com/mustafaeeroglu/rss-fresh/internal/db"
)

// Notifier is the full contract for the Telegram notification module.
// New returns a nopNotifier when the bot token or chat ID is absent, so callers
// never need to nil-check.
type Notifier interface {
	Run(ctx context.Context)
	Close()
	SendDigest(ctx context.Context)
	NotifyCritical(category db.Category, articles []db.Article)
}

// New returns a Notifier. When TELEGRAM_BOT_TOKEN or TELEGRAM_CHAT_ID is
// missing it returns a silent no-op rather than nil.
func New(cfg *config.Config, database *db.DB, log *slog.Logger) (Notifier, error) {
	if cfg.TelegramBotToken == "" || cfg.TelegramChatID == 0 {
		log.Warn("telegram disabled (missing TELEGRAM_BOT_TOKEN or TELEGRAM_CHAT_ID)")
		return nopNotifier{}, nil
	}
	bot, err := tgbot.NewBotAPI(cfg.TelegramBotToken)
	if err != nil {
		return nil, fmt.Errorf("telegram bot init: %w", err)
	}
	bot.Debug = false
	return &notifier{
		cfg:    cfg,
		db:     database,
		log:    log,
		bot:    bot,
		chatID: cfg.TelegramChatID,
		queue:  make(chan tgbot.MessageConfig, 64),
	}, nil
}

type notifier struct {
	cfg    *config.Config
	db     *db.DB
	log    *slog.Logger
	bot    *tgbot.BotAPI
	chatID int64

	queue chan tgbot.MessageConfig
	wg    sync.WaitGroup
}

// Run starts the throttled sender goroutine. Returns on ctx cancellation.
func (n *notifier) Run(ctx context.Context) {
	n.wg.Add(1)
	defer n.wg.Done()
	throttle := n.cfg.CriticalThrottle
	if throttle <= 0 {
		throttle = 5 * time.Second
	}
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-n.queue:
			if _, err := n.bot.Send(msg); err != nil {
				n.log.Warn("telegram send", "err", err)
			}
			// Throttle between sends, cancellable via ctx.
			select {
			case <-ctx.Done():
				return
			case <-time.After(throttle):
			}
		}
	}
}

// Close waits for the Run goroutine to finish.
func (n *notifier) Close() { n.wg.Wait() }

// NotifyCritical batches up to 5 articles per message and enqueues them.
// Best-effort: if the queue is full, drops with a warning (the article is
// still in the DB — the user just doesn't get the push).
func (n *notifier) NotifyCritical(cat db.Category, articles []db.Article) {
	if len(articles) == 0 {
		return
	}
	const perMsg = 5
	for i := 0; i < len(articles); i += perMsg {
		end := i + perMsg
		if end > len(articles) {
			end = len(articles)
		}
		batch := articles[i:end]
		var sb strings.Builder
		fmt.Fprintf(&sb, "🚨 %s — %d new\n", cat.Name, len(batch))
		for _, a := range batch {
			fmt.Fprintf(&sb, "• %s\n%s\n", escapeHTML(a.Title), escapeHTML(a.URL))
		}
		msg := tgbot.NewMessage(n.chatID, sb.String())
		msg.DisableWebPagePreview = true
		msg.ParseMode = "HTML"
		select {
		case n.queue <- msg:
		default:
			n.log.Warn("telegram queue full, dropping critical message")
			return
		}
	}
}

// SendDigest emits one message summarising unread counts per category
// plus any saved articles from the last 24 hours. No-op if nothing to report.
func (n *notifier) SendDigest(ctx context.Context) {
	counts, err := n.db.UnreadCountsByCategory(ctx)
	if err != nil {
		n.log.Warn("digest: unread counts", "err", err)
		return
	}
	cats, err := n.db.ListCategoriesWithCounts(ctx)
	if err != nil {
		n.log.Warn("digest: categories", "err", err)
		return
	}

	since := time.Now().Add(-24 * time.Hour).UTC()
	saved, err := n.db.SavedArticlesSince(ctx, since, 10)
	if err != nil {
		n.log.Warn("digest: saved articles", "err", err)
		saved = nil
	}

	totalUnread := 0
	var sb strings.Builder
	fmt.Fprintf(&sb, "📰 RSS-Fresh digest — %s\n", time.Now().Format("2006-01-02"))
	for _, c := range cats {
		v := counts[c.ID]
		if v == 0 {
			continue
		}
		totalUnread += v
		fmt.Fprintf(&sb, "• %s — %d unread\n", c.Name, v)
	}

	if len(saved) > 0 {
		fmt.Fprintf(&sb, "\n⭐ Saved (%d)\n", len(saved))
		for _, a := range saved {
			fmt.Fprintf(&sb, "• %s\n%s\n", escapeHTML(a.Title), escapeHTML(a.URL))
		}
	}

	if totalUnread == 0 && len(saved) == 0 {
		return
	}

	msg := tgbot.NewMessage(n.chatID, sb.String())
	msg.DisableWebPagePreview = true
	msg.ParseMode = "HTML"
	select {
	case n.queue <- msg:
	default:
		n.log.Warn("telegram queue full, dropping digest")
	}
}

// nopNotifier is the Null Object returned when Telegram is disabled.
type nopNotifier struct{}

func (nopNotifier) Run(context.Context)                      {}
func (nopNotifier) Close()                                   {}
func (nopNotifier) SendDigest(context.Context)               {}
func (nopNotifier) NotifyCritical(db.Category, []db.Article) {}

var htmlReplacer = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

func escapeHTML(s string) string { return htmlReplacer.Replace(s) }
