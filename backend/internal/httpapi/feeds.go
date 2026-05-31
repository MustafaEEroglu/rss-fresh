package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/mustafaeeroglu/rss-fresh/internal/db"
)

// validateFeedURL checks that the URL is http(s) and does not resolve to a
// private, loopback, or link-local address — preventing SSRF against
// internal services reachable from the container (Portainer, PgBouncer, etc.).
func validateFeedURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return errors.New("url must be a http(s) URL")
	}
	host := parsed.Hostname()
	addrs, err := net.LookupHost(host)
	if err != nil {
		// Unresolvable at validation time — allow it; the fetcher will fail gracefully.
		return nil
	}
	for _, addr := range addrs {
		ip := net.ParseIP(addr)
		if ip == nil {
			continue
		}
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
			return fmt.Errorf("url resolves to a disallowed address (%s)", ip)
		}
	}
	return nil
}

type feedCreateReq struct {
	CategoryID int64  `json:"category_id"`
	Name       string `json:"name"`
	URL        string `json:"url"`
}

type feedUpdateReq struct {
	CategoryID *int64  `json:"category_id"`
	Name       *string `json:"name"`
	URL        *string `json:"url"`
	IsActive   *bool   `json:"is_active"`
}

func (s *Server) handleListFeeds(w http.ResponseWriter, r *http.Request) {
	var catID *int64
	if v := r.URL.Query().Get("category_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "bad_query", "category_id must be int")
			return
		}
		catID = &id
	}
	items, err := s.db.ListFeeds(r.Context(), catID)
	if err != nil {
		s.log.Error("list feeds", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "list feeds failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleCreateFeed(w http.ResponseWriter, r *http.Request) {
	var req feedCreateReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	if req.CategoryID == 0 {
		writeError(w, http.StatusBadRequest, "validation", "category_id is required")
		return
	}
	req.URL = strings.TrimSpace(req.URL)
	if err := validateFeedURL(req.URL); err != nil {
		writeError(w, http.StatusBadRequest, "validation", err.Error())
		return
	}
	f, err := s.db.CreateFeed(r.Context(), req.CategoryID, strings.TrimSpace(req.Name), req.URL)
	if errors.Is(err, db.ErrConflict) {
		writeError(w, http.StatusConflict, "conflict", "feed url already exists")
		return
	}
	if err != nil {
		s.log.Error("create feed", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "create feed failed")
		return
	}
	if s.refresher != nil {
		go s.refresher.RefreshFeed(context.WithoutCancel(r.Context()), f.ID)
	}
	writeJSON(w, http.StatusCreated, f)
}

func (s *Server) handleUpdateFeed(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_id", "id must be an integer")
		return
	}
	var req feedUpdateReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	if req.URL != nil {
		u := strings.TrimSpace(*req.URL)
		if err := validateFeedURL(u); err != nil {
			writeError(w, http.StatusBadRequest, "validation", err.Error())
			return
		}
		req.URL = &u
	}
	f, err := s.db.UpdateFeed(r.Context(), id, req.CategoryID, req.Name, req.URL, req.IsActive)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "feed not found")
		return
	}
	if errors.Is(err, db.ErrConflict) {
		writeError(w, http.StatusConflict, "conflict", "feed url already exists")
		return
	}
	if err != nil {
		s.log.Error("update feed", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "update feed failed")
		return
	}
	writeJSON(w, http.StatusOK, f)
}

func (s *Server) handleDeleteFeed(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_id", "id must be an integer")
		return
	}
	err = s.db.DeleteFeed(r.Context(), id)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "feed not found")
		return
	}
	if err != nil {
		s.log.Error("delete feed", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "delete feed failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRefreshFeed(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_id", "id must be an integer")
		return
	}
	if _, err := s.db.GetFeed(r.Context(), id); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "feed not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", "lookup failed")
		return
	}
	if s.refresher != nil {
		go s.refresher.RefreshFeed(context.WithoutCancel(r.Context()), id)
	}
	w.WriteHeader(http.StatusAccepted)
}
