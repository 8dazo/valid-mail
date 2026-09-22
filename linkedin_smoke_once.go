package main

import (
	"context"
	"log"
	"net/url"
	"time"
)

func init() {
	go func() {
		time.Sleep(2 * time.Second)
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		client := newSafeEnrichClient()

		checks := []struct {
			name string
			url  string
		}{
			{"linkedin_directory", "https://www.linkedin.com/pub/dir/patrick/collison"},
			{"linkedin_profile", "https://www.linkedin.com/in/patrickcollison"},
			{"duckduckgo", "https://html.duckduckgo.com/html/?q=" + url.QueryEscape(`site:linkedin.com/in/ "Patrick Collison"`)},
		}
		for _, check := range checks {
			body, contentType, finalURL, err := fetchPublicText(ctx, client, check.url)
			if err != nil {
				log.Printf("LINKEDIN_NETCHECK %s failed: err=%v", check.name, err)
				continue
			}
			log.Printf("LINKEDIN_NETCHECK %s ok: type=%q final=%s bytes=%d", check.name, contentType, finalURL, len(body))
		}

		resolution, err := discoverLinkedInProfileByName(ctx, "Patrick Collison")
		if err != nil {
			log.Printf("LINKEDIN_SMOKETEST failed: err=%v alternatives=%d confidence=%d ambiguous=%t", err, len(resolution.Alternatives), resolution.Confidence, resolution.Ambiguous)
			return
		}
		if resolution.Selected == nil {
			log.Printf("LINKEDIN_SMOKETEST failed: no selected profile alternatives=%d", len(resolution.Alternatives))
			return
		}
		log.Printf("LINKEDIN_SMOKETEST ok: url=%s name=%q company=%q status=%s score=%d alternatives=%d ambiguous=%t", resolution.Selected.URL, resolution.Selected.Name, resolution.Selected.Company, resolution.Selected.Status, resolution.Selected.Score, len(resolution.Alternatives), resolution.Ambiguous)
	}()
}
