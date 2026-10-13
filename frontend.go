package main

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// The frontend build is embedded so production needs only the Go binary.
// Run npm ci && npm run build in frontend before compiling the server.
//
//go:embed all:frontend/dist
var frontendBundle embed.FS

func serveFrontend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	root, err := fs.Sub(frontendBundle, "frontend/dist")
	if err != nil {
		http.Error(w, "frontend is unavailable", http.StatusServiceUnavailable)
		return
	}
	name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if r.URL.Path == "/" || r.URL.Path == "/finder" || strings.HasPrefix(r.URL.Path, "/tools/") {
		data, err := fs.ReadFile(root, "index.html")
		if err != nil {
			http.Error(w, "frontend is unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		if r.Method == http.MethodGet {
			_, _ = w.Write(data)
		}
		return
	}
	info, err := fs.Stat(root, name)
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}
	if strings.HasPrefix(name, "assets/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=3600")
	}
	http.FileServer(http.FS(root)).ServeHTTP(w, r)
}
