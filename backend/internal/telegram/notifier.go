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

type Notifier struct {
	cfg    *config.Config
	db     *db.DB
	log    *slog.Logger
	bot    *tgbot.BotAPI
	chatID int64

	queue chan tgbot.MessageConfig
	wg    sync.WaitGroup
	once  sync.Once
}

// New returns a notifier. If TELEGRAM_BOT_TOKEN or TELEGRAM_CHAT_ID is missing,
// it returns a nil notifier — callers MUST nil-check.
func New(cfg *config.Config, database *db.DB, log *slog.Logger) (*Notifier, error) {
	if cfg.TelegramBotToken == "" || cfg.TelegramChatID == 0 {
		log.Warn("telegram disabled (missing TELEGRAM_BOT_TOKEN or TELEGRAM_CHAT_ID)")
		return nil, nil
	}
	bot, err := tgbot.NewBotAPI(cfg.TelegramBotToken)
	if err != nil {
		return nil, fmt.Errorf("telegram bot init: %w", err)
	}
	bot.Debug = false
	n := &Notifier{
		cfg:    cfg,
		db:     database,
		log:    log,
		bot:    bot,
		chatID: cfg.TelegramChatID,
		queue:  make(chan tgbot.MessageConfig, 64),
	}
	return n, nil
}

// Run starts the throttled sender goroutine. Returns on ctx cancellation.
func (n *Notifier) Run(ctx context.Context) {
	if n == nil {
		return
	}
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
			// Throttle by sleeping between sends, cancellable by ctx.
			select {
			case <-ctx.Done():
				return
			case <-time.After(throttle):
			}
		}
	}
}

// Close waits for the Run goroutine to finish. Safe to call multiple times.
func (n *Notifier) Close() {
	if n == nil {
		return
	}
	n.once.Do(func() { /* lifecycle owned by ctx; nothing else to close */ })
	n.wg.Wait()
}

// NotifyCritical batches up to 5 articles per message and enqueues them.
// Best-effort: if the queue is full, drops with a warning (the article is
// still in the DB — the user just doesn't get the push).
func (n *Notifier) NotifyCritical(cat db.Category, articles []db.Article) {
	if n == nil || len(articles) == 0 {
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
			fmt.Fprintf(&sb, "• %s\n%s\n", escapeHTML(a.Title), a.URL)
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

// SendDigest emits one message summarising unread counts per category.
// No-op if all categories are zero.
func (n *Notifier) SendDigest(ctx context.Context) {
	if n == nil {
		return
	}
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
	if totalUnread == 0 {
		return
	}
	msg := tgbot.NewMessage(n.chatID, sb.String())
	msg.DisableWebPagePreview = true
	select {
	case n.queue <- msg:
	default:
		n.log.Warn("telegram queue full, dropping digest")
	}
}

func escapeHTML(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}
