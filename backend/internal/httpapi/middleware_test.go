package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mustafaeeroglu/rss-fresh/internal/config"
)

func newTestServer(token string) *Server {
	return &Server{
		cfg: &config.Config{OpenClawToken: token},
		log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func TestRequireOpenClawToken(t *testing.T) {
	const tok = "secret-token-abc-123"
	s := newTestServer(tok)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := s.requireOpenClawToken(next)

	cases := []struct {
		name   string
		header string
		want   int
	}{
		{"no header", "", http.StatusUnauthorized},
		{"wrong scheme", "Token " + tok, http.StatusUnauthorized},
		{"wrong token", "Bearer wrong", http.StatusUnauthorized},
		{"correct token", "Bearer " + tok, http.StatusOK},
		{"empty bearer", "Bearer ", http.StatusUnauthorized},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			if c.header != "" {
				req.Header.Set("Authorization", c.header)
			}
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code != c.want {
				t.Fatalf("got %d want %d", rr.Code, c.want)
			}
		})
	}
}
