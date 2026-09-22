package main

import (
	"context"
	"log"
	"time"
)

func init() {
	go func() {
		time.Sleep(2 * time.Second)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		providers := []struct {
			name string
			fn   func(context.Context, string) ([]publicSearchHit, error)
		}{
			{"bing_html", searchBingHTMLLinkedIn},
			{"duckduckgo", searchDuckDuckGoLinkedIn},
		}
		for _, provider := range providers {
			hits, err := provider.fn(ctx, "Patrick Collison")
			if err != nil {
				log.Printf("LINKEDIN_HITS %s failed: %v", provider.name, err)
				continue
			}
			log.Printf("LINKEDIN_HITS %s count=%d", provider.name, len(hits))
			for i, hit := range hits {
				text := hit.Text
				if len(text) > 220 {
					text = text[:220]
				}
				profile := scoreDiscoveredProfile("Patrick Collison", hit, nil)
				log.Printf("LINKEDIN_HIT %s #%d url=%s text=%q company_guess=%q score=%d", provider.name, i+1, hit.URL, text, profile.Company, profile.Score)
			}
		}

		resolution, err := discoverLinkedInProfileByName(ctx, "Patrick Collison")
		if err != nil {
			log.Printf("LINKEDIN_SMOKETEST failed: err=%v alternatives=%d confidence=%d ambiguous=%t method=%s", err, len(resolution.Alternatives), resolution.Confidence, resolution.Ambiguous, resolution.Method)
			for i, alt := range resolution.Alternatives {
				log.Printf("LINKEDIN_ALT #%d url=%s company=%q score=%d text=%q", i+1, alt.URL, alt.Company, alt.Score, alt.SearchText)
			}
			return
		}
		if resolution.Selected == nil {
			log.Printf("LINKEDIN_SMOKETEST failed: no selected profile alternatives=%d method=%s", len(resolution.Alternatives), resolution.Method)
			return
		}
		log.Printf("LINKEDIN_SMOKETEST ok: url=%s company=%q status=%s score=%d alternatives=%d ambiguous=%t method=%s", resolution.Selected.URL, resolution.Selected.Company, resolution.Selected.Status, resolution.Selected.Score, len(resolution.Alternatives), resolution.Ambiguous, resolution.Method)
	}()
}
