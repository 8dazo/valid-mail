package main

import (
	"context"
	"log"
	"net/url"
	"strings"
	"time"
)

func init() {
	go func() {
		time.Sleep(2 * time.Second)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		client := newSafeEnrichClient()
		query := `site:linkedin.com/in/ "Patrick Collison"`

		checks := []struct {
			name string
			url  string
		}{
			{"linkedin_directory", "https://www.linkedin.com/pub/dir/patrick/collison"},
			{"duckduckgo", "https://html.duckduckgo.com/html/?q=" + url.QueryEscape(query)},
			{"bing_rss", "https://www.bing.com/search?format=rss&q=" + url.QueryEscape(query)},
			{"bing_html", "https://www.bing.com/search?q=" + url.QueryEscape(query)},
			{"brave_html", "https://search.brave.com/search?q=" + url.QueryEscape(query) + "&source=web"},
			{"yahoo_html", "https://search.yahoo.com/search?p=" + url.QueryEscape(query)},
		}
		for _, check := range checks {
			body, contentType, finalURL, err := fetchPublicText(ctx, client, check.url)
			if err != nil {
				log.Printf("LINKEDIN_NETCHECK %s failed: err=%v", check.name, err)
				continue
			}
			log.Printf("LINKEDIN_NETCHECK %s ok: type=%q final=%s bytes=%d linkedin_hits=%d items=%d", check.name, contentType, finalURL, len(body), strings.Count(strings.ToLower(body), "linkedin.com/in"), strings.Count(strings.ToLower(body), "<item>"))
		}

		signal := fetchLinkedInPublicSignalForDiscovery(ctx, "https://www.linkedin.com/in/patrickcollison")
		log.Printf("LINKEDIN_PROFILECHECK status=%s name=%q company=%q headline_len=%d evidence=%v", signal.Status, signal.Name, signal.Company, len(signal.Headline), signal.Evidence)

		resolution, err := discoverLinkedInProfileByName(ctx, "Patrick Collison")
		if err != nil {
			log.Printf("LINKEDIN_SMOKETEST failed: err=%v alternatives=%d confidence=%d ambiguous=%t method=%s", err, len(resolution.Alternatives), resolution.Confidence, resolution.Ambiguous, resolution.Method)
			return
		}
		if resolution.Selected == nil {
			log.Printf("LINKEDIN_SMOKETEST failed: no selected profile alternatives=%d method=%s", len(resolution.Alternatives), resolution.Method)
			return
		}
		log.Printf("LINKEDIN_SMOKETEST ok: url=%s name=%q company=%q status=%s score=%d alternatives=%d ambiguous=%t method=%s", resolution.Selected.URL, resolution.Selected.Name, resolution.Selected.Company, resolution.Selected.Status, resolution.Selected.Score, len(resolution.Alternatives), resolution.Ambiguous, resolution.Method)
	}()
}
