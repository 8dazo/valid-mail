package main

import (
	"context"
	"errors"
	"html"
	"io"
	"net/http"
	"strings"
)

const linkedInMaxBodyBytes = 2 << 20

func fetchLinkedInPublicSignalForDiscovery(ctx context.Context, profileURL string) *linkedInPublicSignal {
	signal := &linkedInPublicSignal{URL: profileURL, Status: "unavailable"}
	body, contentType, finalURL, err := fetchLinkedInPublicHTML(ctx, profileURL)
	if err != nil || body == "" {
		signal.Evidence = append(signal.Evidence, "public profile fetch was unavailable")
		return signal
	}
	if !strings.Contains(contentType, "text/html") && contentType != "" {
		signal.Evidence = append(signal.Evidence, "LinkedIn response was not an HTML profile")
		return signal
	}

	lowerBody := strings.ToLower(body)
	lowerFinal := strings.ToLower(finalURL)
	if strings.Contains(lowerFinal, "/login") || strings.Contains(lowerFinal, "/authwall") || strings.Contains(lowerFinal, "/checkpoint") ||
		strings.Contains(lowerBody, "authwall") || strings.Contains(lowerBody, "sign in to linkedin") || strings.Contains(lowerBody, "join linkedin") {
		signal.Status = "blocked"
		signal.Evidence = append(signal.Evidence, "LinkedIn returned a login wall or access challenge; no bypass was attempted")
		return signal
	}

	meta := extractMetaValues(body)
	name, headline, company := extractLinkedInJSONLD(body)
	if name == "" {
		name = cleanLinkedInName(meta["og:title"])
	}
	if headline == "" {
		headline = strings.TrimSpace(firstNonEmpty(meta["og:description"], meta["description"]))
	}
	if company == "" {
		company = companyFromHeadline(headline)
	}

	signal.Name = name
	signal.Headline = headline
	signal.Company = company
	if name != "" || headline != "" || company != "" {
		signal.Status = "fetched"
		if name != "" {
			signal.Evidence = append(signal.Evidence, "public profile name metadata found")
		}
		if company != "" {
			signal.Evidence = append(signal.Evidence, "public company metadata found")
		}
	} else {
		signal.Evidence = append(signal.Evidence, "page was public but did not expose usable profile metadata")
	}
	return signal
}

func fetchLinkedInPublicHTML(ctx context.Context, rawURL string) (string, string, string, error) {
	client := newSafeEnrichClient()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", "", rawURL, err
	}
	if err := validatePublicHost(ctx, req.URL.Hostname()); err != nil {
		return "", "", rawURL, err
	}
	req.Header.Set("User-Agent", "valid-mail/1.0 public-profile-metadata")
	req.Header.Set("Accept", "text/html,application/xhtml+xml;q=0.9,*/*;q=0.1")

	resp, err := client.Do(req)
	if err != nil {
		return "", "", rawURL, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", resp.Request.URL.String(), errors.New("linkedin http status outside 2xx")
	}
	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	if !(strings.Contains(contentType, "text/html") || contentType == "") {
		return "", contentType, resp.Request.URL.String(), errors.New("unsupported LinkedIn content type")
	}

	limited := io.LimitReader(resp.Body, linkedInMaxBodyBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return "", contentType, resp.Request.URL.String(), err
	}
	if len(body) > linkedInMaxBodyBytes {
		return "", contentType, resp.Request.URL.String(), errors.New("LinkedIn profile response too large")
	}
	return html.UnescapeString(string(body)), contentType, resp.Request.URL.String(), nil
}
