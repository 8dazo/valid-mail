package main

import (
	_ "embed"
	"net/http"
)

//go:embed web/finder.html
var finderHTML []byte

func (app *application) handleFinderPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/finder" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if r.Method == http.MethodGet {
		_, _ = w.Write(finderHTML)
	}
}
