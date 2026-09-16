package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

//go:embed all:dist
var distFS embed.FS

// RegisterWebRoutes registers static file serving with SPA fallback to index.html
func RegisterWebRoutes(r chi.Router) {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		logrus.Fatalf("Failed to create sub filesystem for web/dist: %v", err)
	}

	fileServer := http.FileServer(http.FS(sub))

	r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
		path := strings.TrimPrefix(req.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		// If file exists in embedded filesystem, serve it
		f, err := sub.Open(path)
		if err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(w, req)
			return
		}

		// Fallback to index.html for React Router SPA
		req.URL.Path = "/"
		fileServer.ServeHTTP(w, req)
	})
}
