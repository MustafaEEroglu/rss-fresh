package telegram

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	tgbot "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/mustafaeeroglu/rss-fresh/internal/db"
)

func TestEscapeHTML_Ampersand(t *testing.T) {
	if got := escapeHTML("Cats & Dogs"); got != "Cats &amp; Dogs" {
		t.Errorf("got %q", got)
	}
}

func TestEscapeHTML_LessThan(t *testing.T) {
	if got := escapeHTML("<script>"); got != "&lt;script&gt;" {
		t.Errorf("got %q", got)
	}
}

func TestEscapeHTML_XSSTitle(t *testing.T) {
	in := `<img src=x onerror=alert(1)>`
	out := escapeHTML(in)
	if strings.Contains(out, "<") || strings.Contains(out, ">") {
		t.Errorf("unescaped angle brackets in output: %q", out)
	}
}

func TestEscapeHTML_Empty(t *testing.T) {
	if got := escapeHTML(""); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestEscapeHTML_NoSpecialChars(t *testing.T) {
	plain := "Hello World 123"
	if got := escapeHTML(plain); got != plain {
		t.Errorf("plain text should be unchanged, got %q", got)
	}
}

func TestEscapeHTML_MultipleAmpersands(t *testing.T) {
	if got := escapeHTML("A&B&C"); got != "A&amp;B&amp;C" {
		t.Errorf("got %q", got)
	}
}

func TestEscapeHTML_AllThreeSpecialChars(t *testing.T) {
	in := "< & >"
	want := "&lt; &amp; &gt;"
	if got := escapeHTML(in); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestNopNotifier_AllMethodsAreNoop(t *testing.T) {
	var n Notifier = nopNotifier{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	n.Run(ctx)
	n.Close()
	n.SendDigest(ctx)
	n.NotifyCritical(db.Category{Name: "Test"}, []db.Article{{ID: 1, Title: "T", URL: "https://x"}})
}

func newTestNotifier() *notifier {
	return &notifier{
		log:   slog.Default(),
		queue: make(chan tgbot.MessageConfig, 64),
	}
}

func TestNotifyCritical_EmptyArticlesProducesNoMessages(t *testing.T) {
	n := newTestNotifier()
	n.NotifyCritical(db.Category{Name: "Tech"}, nil)
	n.NotifyCritical(db.Category{Name: "Tech"}, []db.Article{})
	if got := len(n.queue); got != 0 {
		t.Errorf("expected 0 queued messages, got %d", got)
	}
}

func TestNotifyCritical_SingleArticleProducesOneMessage(t *testing.T) {
	n := newTestNotifier()
	n.NotifyCritical(db.Category{Name: "Breaking"}, []db.Article{
		{ID: 1, Title: "Big News", URL: "https://example.com/1"},
	})
	if got := len(n.queue); got != 1 {
		t.Errorf("expected 1 message, got %d", got)
	}
}

func TestNotifyCritical_BatchesOf5(t *testing.T) {
	n := newTestNotifier()
	articles := make([]db.Article, 11)
	for i := range articles {
		articles[i] = db.Article{ID: int64(i + 1), Title: "Title", URL: "https://x.com"}
	}
	n.NotifyCritical(db.Category{Name: "Breaking"}, articles)
	if got := len(n.queue); got != 3 {
		t.Errorf("11 articles: expected 3 batched messages, got %d", got)
	}
}

func TestNotifyCritical_ExactlyFiveArticlesProducesOneMessage(t *testing.T) {
	n := newTestNotifier()
	articles := make([]db.Article, 5)
	for i := range articles {
		articles[i] = db.Article{ID: int64(i + 1), Title: "T", URL: "https://x.com"}
	}
	n.NotifyCritical(db.Category{Name: "Critical"}, articles)
	if got := len(n.queue); got != 1 {
		t.Errorf("5 articles: expected 1 message, got %d", got)
	}
}

func TestNotifyCritical_QueueFullDropsGracefully(t *testing.T) {
	n := newTestNotifier()
	articles := make([]db.Article, 330)
	for i := range articles {
		articles[i] = db.Article{ID: int64(i + 1), Title: "Alert", URL: "https://x.com"}
	}
	done := make(chan struct{})
	go func() {
		n.NotifyCritical(db.Category{Name: "Critical"}, articles)
		close(done)
	}()
	<-done
	if q := len(n.queue); q > 64 {
		t.Errorf("queue overflow: %d > 64", q)
	}
}

func TestNotifyCritical_MessageContainsCategoryName(t *testing.T) {
	n := newTestNotifier()
	n.NotifyCritical(db.Category{Name: "Security Alerts"}, []db.Article{
		{Title: "Vuln found", URL: "https://example.com"},
	})
	msg := <-n.queue
	if !strings.Contains(msg.Text, "Security Alerts") {
		t.Errorf("expected category name in message, got: %q", msg.Text)
	}
}

func TestNotifyCritical_TitleIsHTMLEscaped(t *testing.T) {
	n := newTestNotifier()
	n.NotifyCritical(db.Category{Name: "Cat"}, []db.Article{
		{Title: "<b>Bold</b> & more", URL: "https://example.com"},
	})
	msg := <-n.queue
	if strings.Contains(msg.Text, "<b>Bold</b>") {
		t.Errorf("title HTML was not escaped in Telegram message: %q", msg.Text)
	}
	if !strings.Contains(msg.Text, "&lt;b&gt;") {
		t.Errorf("expected escaped title in message: %q", msg.Text)
	}
}

func TestNotifyCritical_ParseModeIsHTML(t *testing.T) {
	n := newTestNotifier()
	n.NotifyCritical(db.Category{Name: "Cat"}, []db.Article{
		{Title: "Test", URL: "https://example.com"},
	})
	msg := <-n.queue
	if msg.ParseMode != "HTML" {
		t.Errorf("expected ParseMode HTML, got %q", msg.ParseMode)
	}
}
