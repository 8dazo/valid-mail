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
		searchURL := "https://www.bing.com/search?q=" + url.QueryEscape(query)
		body, _, finalURL, err := fetchPublicText(ctx, client, searchURL)
		if err != nil {
			log.Printf("BING_HREF fetch failed: %v", err)
		} else {
			logged := 0
			for _, match := range resultAnchorRE.FindAllStringSubmatch(body, -1) {
				if len(match) != 3 {
					continue
				}
				href := html.UnescapeString(strings.TrimSpace(match[1]))
				text := strings.Join(strings.Fields(html.UnescapeString(stripTagRE.ReplaceAllString(match[2], " "))), " ")
				lower := strings.ToLower(href + " " + text)
				if !strings.Contains(lower, "ck/a") && !strings.Contains(lower, "linkedin") && !strings.Contains(lower, "u=a1") && !strings.Contains(strings.ToLower(text), "patrick") {
					continue
				}
				decoded := decodeSearchResultURL(href)
				if len(href) > 500 { href = href[:500] }
				if len(text) > 220 { text = text[:220] }
				log.Printf("BING_HREF #%d href=%q text=%q decoded=%q", logged+1, href, text, decoded)
				logged++
				if logged >= 15 { break }
			}
			log.Printf("BING_HREF_SUMMARY final=%s anchors_logged=%d", finalURL, logged)
		}

		hits, err := searchBingHTMLLinkedIn(ctx, "Patrick Collison")
		if err != nil {
			log.Printf("LINKEDIN_HITS bing_html failed: %v", err)
		} else {
			log.Printf("LINKEDIN_HITS bing_html count=%d", len(hits))
			for i, hit := range hits {
				text := hit.Text
				if len(text) > 220 { text = text[:220] }
				profile := scoreDiscoveredProfile("Patrick Collison", hit, nil)
				log.Printf("LINKEDIN_HIT bing_html #%d url=%s text=%q company_guess=%q score=%d", i+1, hit.URL, text, profile.Company, profile.Score)
			}
		}

		resolution, err := discoverLinkedInProfileByName(ctx, "Patrick Collison")
		if err != nil {
			log.Printf("LINKEDIN_SMOKETEST failed: err=%v alternatives=%d confidence=%d ambiguous=%t method=%s", err, len(resolution.Alternatives), resolution.Confidence, resolution.Ambiguous, resolution.Method)
			return
		}
		if resolution.Selected == nil {
			log.Printf("LINKEDIN_SMOKETEST failed: no selected profile alternatives=%d method=%s", len(resolution.Alternatives), resolution.Method)
			return
		}
		log.Printf("LINKEDIN_SMOKETEST ok: url=%s company=%q status=%s score=%d alternatives=%d ambiguous=%t method=%s", resolution.Selected.URL, resolution.Selected.Company, resolution.Selected.Status, resolution.Selected.Score, len(resolution.Alternatives), resolution.Ambiguous, resolution.Method)
	}()
}
