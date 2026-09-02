package handlers

import (
	"net/http"

	"github.com/highercomve/couchness/web/static"
)

func (h *Handler) root(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/shows", http.StatusFound)
}

// staticFiles serves embedded static assets with a long cache header.
func staticFiles() http.Handler {
	fs := http.FileServer(http.FS(static.FS))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=86400")
		fs.ServeHTTP(w, r)
	})
}
