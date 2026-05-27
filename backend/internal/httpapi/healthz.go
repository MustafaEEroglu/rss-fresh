package httpapi

import (
	"net/http"
	"time"
)

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":         "ok",
		"version":        s.cfg.Version,
		"uptime_seconds": int(time.Since(s.startedAt).Seconds()),
	})
}
