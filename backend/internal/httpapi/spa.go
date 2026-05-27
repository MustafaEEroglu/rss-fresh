package httpapi

import (
	"errors"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// spaHandler serves a built SPA from the given filesystem.
// Strategy: serve real files at their paths (with content hashes those will be
// long-cached); for any path that doesn't resolve to a file and isn't an /api
// route, fall back to index.html so client-side routing works.
func spaHandler(spa fs.FS) http.Handler {
	indexBytes, _ := fs.ReadFile(spa, "index.html")
	hasIndex := len(indexBytes) > 0

	fileServer := http.FileServer(http.FS(spa))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// /api/* should never reach here (chi routes catch them first), but
		// be defensive.
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		// Try the literal asset first. Strip the leading slash to consult fs.FS.
		clean := path.Clean(r.URL.Path)
		if clean == "/" {
			clean = "index.html"
		} else {
			clean = strings.TrimPrefix(clean, "/")
		}

		if f, err := spa.Open(clean); err == nil {
			info, statErr := f.Stat()
			f.Close()
			if statErr == nil && !info.IsDir() {
				// Long cache for hashed assets, short for index.html.
				if strings.HasPrefix(clean, "assets/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				} else {
					w.Header().Set("Cache-Control", "no-cache")
				}
				fileServer.ServeHTTP(w, r)
				return
			}
		} else if !errors.Is(err, fs.ErrNotExist) {
			http.Error(w, "spa fs error", http.StatusInternalServerError)
			return
		}

		if !hasIndex {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(indexBytes)
	})
}
