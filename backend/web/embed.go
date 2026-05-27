// Package web embeds the built Svelte SPA into the binary at compile time.
// The Dockerfile copies the built dist/ here before `go build`. For local Go
// builds without a built SPA, the embed pattern matches the placeholder
// dist/.gitkeep so go:embed succeeds.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// FS returns the rooted /dist filesystem; nil if the embedded dist is empty.
func FS() fs.FS {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil
	}
	// Detect "no files but the .gitkeep placeholder" — caller should fall back.
	if !hasIndexHTML(sub) {
		return nil
	}
	return sub
}

func hasIndexHTML(f fs.FS) bool {
	if _, err := fs.Stat(f, "index.html"); err == nil {
		return true
	}
	return false
}
