package httpapi

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/mustafaeeroglu/rss-fresh/internal/db"
)

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

type categoryCreateReq struct {
	Name       string `json:"name"`
	Slug       string `json:"slug"`
	IsCritical *bool  `json:"is_critical"`
}

type categoryUpdateReq struct {
	Name       *string `json:"name"`
	Slug       *string `json:"slug"`
	IsCritical *bool   `json:"is_critical"`
}

func (s *Server) handleListCategories(w http.ResponseWriter, r *http.Request) {
	items, err := s.db.ListCategoriesWithCounts(r.Context())
	if err != nil {
		s.log.Error("list categories", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "list categories failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleCreateCategory(w http.ResponseWriter, r *http.Request) {
	var req categoryCreateReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "validation", "name is required")
		return
	}
	if req.Slug == "" {
		req.Slug = slugify(req.Name)
	} else {
		req.Slug = slugify(req.Slug)
	}
	if req.Slug == "" {
		writeError(w, http.StatusBadRequest, "validation", "slug is empty after normalisation")
		return
	}
	critical := false
	if req.IsCritical != nil {
		critical = *req.IsCritical
	}
	c, err := s.db.CreateCategory(r.Context(), req.Name, req.Slug, critical)
	if errors.Is(err, db.ErrConflict) {
		writeError(w, http.StatusConflict, "conflict", "slug already exists")
		return
	}
	if err != nil {
		s.log.Error("create category", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "create category failed")
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (s *Server) handleUpdateCategory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_id", "id must be an integer")
		return
	}
	var req categoryUpdateReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	if req.Slug != nil {
		ns := slugify(*req.Slug)
		req.Slug = &ns
	}
	c, err := s.db.UpdateCategory(r.Context(), id, req.Name, req.Slug, req.IsCritical)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "category not found")
		return
	}
	if errors.Is(err, db.ErrConflict) {
		writeError(w, http.StatusConflict, "conflict", "slug already exists")
		return
	}
	if err != nil {
		s.log.Error("update category", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "update category failed")
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) handleDeleteCategory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_id", "id must be an integer")
		return
	}
	err = s.db.DeleteCategory(r.Context(), id)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "category not found")
		return
	}
	if err != nil {
		s.log.Error("delete category", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "delete category failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
