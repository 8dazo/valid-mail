package main

import (
	"bytes"
	"embed"
	"io/fs"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// The frontend build is embedded so production needs only the Go binary.
// Run npm ci && npm run build in frontend before compiling the server.
//
//go:embed all:frontend/dist
var frontendBundle embed.FS

var fingerprintedWasm = regexp.MustCompile(`^wasm/(toolkit|go-runtime)-[a-f0-9]{16}\.(wasm|js)(\.gz)?$`)

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
	if strings.HasPrefix(name, "assets/") || fingerprintedWasm.MatchString(name) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=3600")
	}
	if strings.HasSuffix(name, ".wasm") {
		w.Header().Set("Vary", "Accept-Encoding")
		if acceptsGzip(r.Header.Get("Accept-Encoding")) && r.Header.Get("Range") == "" {
			if compressed, err := fs.ReadFile(root, name+".gz"); err == nil {
				w.Header().Set("Content-Encoding", "gzip")
				w.Header().Set("Content-Type", "application/wasm")
				http.ServeContent(w, r, path.Base(name), time.Time{}, bytes.NewReader(compressed))
				return
			}
		}
	}

	http.FileServer(http.FS(root)).ServeHTTP(w, r)
}

func acceptsGzip(header string) bool {
	for _, entry := range strings.Split(header, ",") {
		parts := strings.Split(strings.TrimSpace(entry), ";")
		if parts[0] != "gzip" {
			continue
		}
		enabled := true
		for _, parameter := range parts[1:] {
			if strings.HasPrefix(strings.TrimSpace(parameter), "q=") {
				quality, err := strconv.ParseFloat(strings.TrimPrefix(strings.TrimSpace(parameter), "q="), 64)
				enabled = err == nil && quality > 0
			}
		}
		if enabled {
			return true
		}
	}
	return false
}
