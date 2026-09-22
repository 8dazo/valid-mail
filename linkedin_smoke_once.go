package main

import (
	"context"
	"html"
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
			{"google", "https://www.google.com/search?q=" + url.QueryEscape(query) + "&num=10"},
			{"mojeek", "https://www.mojeek.com/search?q=" + url.QueryEscape(query)},
			{"ddg_lite", "https://lite.duckduckgo.com/lite/?q=" + url.QueryEscape(query)},
		}
		for _, check := range checks {
			body, contentType, finalURL, err := fetchPublicText(ctx, client, check.url)
			if err != nil {
				log.Printf("SEARCH_PROBE %s failed: %v", check.name, err)
				continue
			}
			lower := strings.ToLower(body)
			log.Printf("SEARCH_PROBE %s ok: type=%q final=%s bytes=%d linkedin_mentions=%d patrick_mentions=%d", check.name, contentType, finalURL, len(body), strings.Count(lower, "linkedin.com/in"), strings.Count(lower, "patrick collison"))
			hits := extractLinkedInHits(finalURL, body, maxLinkedInDiscoveryCandidates)
			if len(hits) == 0 { hits = extractLinkedInRawHits(body, maxLinkedInDiscoveryCandidates) }
			log.Printf("SEARCH_HITS %s count=%d", check.name, len(hits))
			for i, hit := range hits {
				text := strings.Join(strings.Fields(html.UnescapeString(hit.Text)), " ")
				if len(text) > 220 { text = text[:220] }
				profile := scoreDiscoveredProfile("Patrick Collison", hit, nil)
				log.Printf("SEARCH_HIT %s #%d url=%s text=%q company=%q score=%d", check.name, i+1, hit.URL, text, profile.Company, profile.Score)
			}
		}
	}()
}
