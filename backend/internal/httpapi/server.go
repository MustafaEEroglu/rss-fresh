// Package httpapi wires the chi router and exposes Server.
package httpapi

import (
	"context"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/mustafaeeroglu/rss-fresh/internal/config"
	"github.com/mustafaeeroglu/rss-fresh/internal/db"
)

type Refresher interface {
	RefreshFeed(ctx context.Context, feedID int64) // async-fire trigger for /feeds/:id/refresh
}

type Server struct {
	cfg       *config.Config
	db        *db.DB
	log       *slog.Logger
	refresher Refresher
	startedAt time.Time
	spaFS     fs.FS
}

func NewServer(cfg *config.Config, database *db.DB, log *slog.Logger, refresher Refresher, spaFS fs.FS) *Server {
	return &Server{
		cfg:       cfg,
		db:        database,
		log:       log,
		refresher: refresher,
		startedAt: time.Now(),
		spaFS:     spaFS,
	}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()

	// Trust proxy headers from Cloudflare Tunnel (we sit behind it). The whole
	// 0.0.0.0/0 trust is fine because the host firewall only allows the tunnel
	// daemon to reach :3000 on loopback.
	r.Use(securityHeaders)
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(slogRequestLogger(s.log))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(corsSameOrigin)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/healthz", s.handleHealthz)

		r.Get("/categories", s.handleListCategories)
		r.Post("/categories", s.handleCreateCategory)
		r.Patch("/categories/{id}", s.handleUpdateCategory)
		r.Delete("/categories/{id}", s.handleDeleteCategory)

		r.Get("/feeds", s.handleListFeeds)
		r.Post("/feeds", s.handleCreateFeed)
		r.Patch("/feeds/{id}", s.handleUpdateFeed)
		r.Delete("/feeds/{id}", s.handleDeleteFeed)
		r.Post("/feeds/{id}/refresh", s.handleRefreshFeed)

		r.Get("/articles", s.handleListArticles)
		r.Patch("/articles/{id}", s.handleUpdateArticle)
		r.Post("/articles/mark-read", s.handleBulkMarkRead)
	})

	if s.spaFS != nil {
		r.Mount("/", spaHandler(s.spaFS))
	}

	return r
}

// helpers shared by handlers ----------------------------------------------

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if body == nil {
		return
	}
	enc := json.NewEncoder(w)
	_ = enc.Encode(body)
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]string{"error": msg, "code": code})
}

func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}
