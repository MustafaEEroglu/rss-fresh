package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mustafaeeroglu/rss-fresh/internal/db"
)

type articleUpdateReq struct {
	IsRead  *bool `json:"is_read"`
	IsSaved *bool `json:"is_saved"`
}

type bulkMarkReadReq struct {
	IDs []int64 `json:"ids"`
}

func (s *Server) handleListArticles(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := db.ListArticlesFilter{
		Cursor: q.Get("cursor"),
	}
	if v := q.Get("category_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "bad_query", "category_id must be int")
			return
		}
		filter.CategoryID = &id
	}
	if v := q.Get("feed_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "bad_query", "feed_id must be int")
			return
		}
		filter.FeedID = &id
	}
	if q.Get("unread") == "1" || q.Get("unread") == "true" {
		filter.Unread = true
	}
	if q.Get("read") == "1" || q.Get("read") == "true" {
		filter.Read = true
	}
	if filter.Unread && filter.Read {
		writeError(w, http.StatusBadRequest, "bad_query", "unread and read are mutually exclusive")
		return
	}
	if q.Get("saved") == "1" || q.Get("saved") == "true" {
		filter.Saved = true
	}
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil && n > 0 {
			filter.Limit = n
		}
	}
	if v := q.Get("since"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "bad_query", "since must be an RFC3339 timestamp")
			return
		}
		filter.Since = &t
	}

	items, next, err := s.db.ListArticles(r.Context(), filter)
	if err != nil {
		s.log.Error("list articles", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "list articles failed")
		return
	}
	resp := map[string]any{"items": items}
	if next != "" {
		resp["next_cursor"] = next
	} else {
		resp["next_cursor"] = nil
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleUpdateArticle(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_id", "id must be an integer")
		return
	}
	var req articleUpdateReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	a, err := s.db.UpdateArticle(r.Context(), id, req.IsRead, req.IsSaved)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "article not found")
		return
	}
	if err != nil {
		s.log.Error("update article", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "update article failed")
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (s *Server) handleBulkMarkRead(w http.ResponseWriter, r *http.Request) {
	var req bulkMarkReadReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	if len(req.IDs) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{"updated": 0})
		return
	}
	if len(req.IDs) > 1000 {
		writeError(w, http.StatusBadRequest, "validation", "max 1000 ids per request")
		return
	}
	n, err := s.db.BulkMarkRead(r.Context(), req.IDs)
	if err != nil {
		s.log.Error("bulk mark read", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "bulk mark read failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"updated": n})
}
