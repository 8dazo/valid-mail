package main

import (
	"context"
	"encoding/json"
	"errors"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type linkedInPublicRequest struct {
	LinkedInURL string `json:"linkedin_url"`
	Name        string `json:"name,omitempty"`
	Company     string `json:"company,omitempty"`
	Domain      string `json:"domain,omitempty"`
}

type linkedInPublicSignal struct {
	URL      string   `json:"url"`
	Status   string   `json:"status"`
	Name     string   `json:"name,omitempty"`
	Headline string   `json:"headline,omitempty"`
	Company  string   `json:"company,omitempty"`
	Evidence []string `json:"evidence,omitempty"`
}

type findLinkedInEmailResponse struct {
	findWorkEmailResponse
	LinkedIn *linkedInPublicSignal `json:"linkedin,omitempty"`
}

var (
	metaTagRE   = regexp.MustCompile(`(?is)<meta\b[^>]*>`)
	metaAttrRE  = regexp.MustCompile(`(?is)([a-zA-Z_:.-]+)\s*=\s*["']([^"']*)["']`)
	jsonLDRE    = regexp.MustCompile(`(?is)<script\b[^>]*type\s*=\s*["']application/ld\+json["'][^>]*>(.*?)</script>`)
	linkedinTitleRE = regexp.MustCompile(`(?i)^\s*(.*?)\s*(?:[-|·]\s*LinkedIn)?\s*$`)
)

func (app *application) handleFindLinkedInEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST, OPTIONS")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !allowEnrichRequest(requestIP(r)) {
		w.Header().Set("Retry-After", "60")
		writeJSON(w, http.StatusTooManyRequests, map[string]any{"success": false, "error": "email discovery rate limit exceeded"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	defer r.Body.Close()
	var req linkedInPublicRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "invalid JSON body"})
		return
	}

	linkedinURL, err := normalizeLinkedInProfileURL(req.LinkedInURL)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": err.Error()})
		return
	}

	signal := fetchLinkedInPublicSignal(r.Context(), linkedinURL)
	name := strings.TrimSpace(req.Name)
	company := strings.TrimSpace(req.Company)
	if name == "" {
		name = strings.TrimSpace(signal.Name)
	}
	if company == "" {
		company = strings.TrimSpace(signal.Company)
	}
	if len(name) < 2 || len(name) > 120 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"success":  false,
			"error":    "a usable person name was not available from the public LinkedIn page; provide name explicitly",
			"linkedin": signal,
		})
		return
	}
	if len(company) > 160 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "company is too long", "linkedin": signal})
		return
	}

	var resolution *companyDomainResolution
	var domain string
	if strings.TrimSpace(req.Domain) != "" {
		domain, err = normalizePublicDomain(req.Domain)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": err.Error(), "linkedin": signal})
			return
		}
		resolution = &companyDomainResolution{Domain: domain, Confidence: 100, Method: "provided_domain"}
	} else {
		if company == "" {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
				"success":  false,
				"error":    "a usable company was not available from the public LinkedIn page; provide company or domain explicitly",
				"linkedin": signal,
			})
			return
		}
		resolved, resolveErr := resolveCompanyDomain(r.Context(), company)
		if resolveErr != nil {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
				"success":  false,
				"error":    resolveErr.Error(),
				"hint":     "provide the company domain explicitly when automatic resolution is uncertain",
				"linkedin": signal,
			})
			return
		}
		resolution = &resolved
		domain = resolved.Domain
	}

	harvest, harvestErr := harvestCompanyWebsite(r.Context(), domain)
	if harvestErr != nil && harvest.PagesFetched == 0 {
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"success":           false,
			"error":             "official domain was resolved, but its public pages could not be read",
			"domain":            domain,
			"domain_resolution": resolution,
			"linkedin":          signal,
		})
		return
	}

	pattern, patternConfidence := inferObservedPattern(harvest.Emails, domain)
	candidates := app.buildEmailCandidates(name, domain, pattern, patternConfidence, harvest)
	base := findWorkEmailResponse{
		Success:            true,
		Name:               name,
		Company:            company,
		Domain:             domain,
		DomainResolution:   resolution,
		ObservedEmailCount: len(harvest.Emails),
		PagesFetched:       harvest.PagesFetched,
		Pattern:            pattern,
		PatternConfidence:  patternConfidence,
		Candidates:         candidates,
		Note:               "LinkedIn is used only as an optional public identity signal. Email candidates still come from public company pages, official-domain heuristics, observed company patterns, and SMTP-independent validity checks. Login walls, challenges, and blocked LinkedIn pages are not bypassed.",
		CheckedAt:          time.Now().UTC(),
	}
	writeJSON(w, http.StatusOK, findLinkedInEmailResponse{findWorkEmailResponse: base, LinkedIn: signal})
}

func normalizeLinkedInProfileURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("linkedin_url is required")
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return "", errors.New("invalid LinkedIn URL")
	}
	host := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	if host != "linkedin.com" && host != "www.linkedin.com" {
		return "", errors.New("linkedin_url must point to linkedin.com")
	}
	if u.Scheme != "https" {
		return "", errors.New("linkedin_url must use https")
	}
	if !strings.HasPrefix(strings.ToLower(u.Path), "/in/") {
		return "", errors.New("linkedin_url must be a public profile path under /in/")
	}
	u.Host = "www.linkedin.com"
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

func fetchLinkedInPublicSignal(ctx context.Context, profileURL string) *linkedInPublicSignal {
	signal := &linkedInPublicSignal{URL: profileURL, Status: "unavailable"}
	client := newSafeEnrichClient()
	body, contentType, finalURL, err := fetchPublicText(ctx, client, profileURL)
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

func extractMetaValues(body string) map[string]string {
	out := make(map[string]string)
	for _, tag := range metaTagRE.FindAllString(body, -1) {
		attrs := make(map[string]string)
		for _, match := range metaAttrRE.FindAllStringSubmatch(tag, -1) {
			if len(match) == 3 {
				attrs[strings.ToLower(strings.TrimSpace(match[1]))] = html.UnescapeString(strings.TrimSpace(match[2]))
			}
		}
		key := strings.ToLower(firstNonEmpty(attrs["property"], attrs["name"]))
		if key == "og:title" || key == "og:description" || key == "description" {
			if value := strings.TrimSpace(attrs["content"]); value != "" {
				out[key] = value
			}
		}
	}
	return out
}

func extractLinkedInJSONLD(body string) (string, string, string) {
	for _, match := range jsonLDRE.FindAllStringSubmatch(body, -1) {
		if len(match) != 2 {
			continue
		}
		var value any
		if err := json.Unmarshal([]byte(html.UnescapeString(strings.TrimSpace(match[1]))), &value); err != nil {
			continue
		}
		name := findJSONLDString(value, "name")
		headline := firstNonEmpty(findJSONLDString(value, "jobTitle"), findJSONLDString(value, "headline"))
		company := findJSONLDWorksFor(value)
		if name != "" || headline != "" || company != "" {
			return strings.TrimSpace(name), strings.TrimSpace(headline), strings.TrimSpace(company)
		}
	}
	return "", "", ""
}

func findJSONLDString(value any, key string) string {
	switch typed := value.(type) {
	case map[string]any:
		if raw, ok := typed[key]; ok {
			if text, ok := raw.(string); ok && strings.TrimSpace(text) != "" {
				return text
			}
		}
		for _, child := range typed {
			if found := findJSONLDString(child, key); found != "" {
				return found
			}
		}
	case []any:
		for _, child := range typed {
			if found := findJSONLDString(child, key); found != "" {
				return found
			}
		}
	}
	return ""
}

func findJSONLDWorksFor(value any) string {
	switch typed := value.(type) {
	case map[string]any:
		if raw, ok := typed["worksFor"]; ok {
			switch work := raw.(type) {
			case map[string]any:
				if name, ok := work["name"].(string); ok {
					return strings.TrimSpace(name)
				}
			case []any:
				for _, item := range work {
					if found := findJSONLDWorksFor(map[string]any{"worksFor": item}); found != "" {
						return found
					}
				}
			}
		}
		for _, child := range typed {
			if found := findJSONLDWorksFor(child); found != "" {
				return found
			}
		}
	case []any:
		for _, child := range typed {
			if found := findJSONLDWorksFor(child); found != "" {
				return found
			}
		}
	}
	return ""
}

func cleanLinkedInName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if match := linkedinTitleRE.FindStringSubmatch(value); len(match) == 2 {
		value = strings.TrimSpace(match[1])
	}
	value = strings.TrimSuffix(value, " | LinkedIn")
	value = strings.TrimSuffix(value, " - LinkedIn")
	return strings.TrimSpace(value)
}

func companyFromHeadline(headline string) string {
	headline = strings.TrimSpace(headline)
	if headline == "" {
		return ""
	}
	lower := strings.ToLower(headline)
	for _, marker := range []string{" at ", " @ "} {
		if idx := strings.LastIndex(lower, marker); idx >= 0 {
			company := strings.TrimSpace(headline[idx+len(marker):])
			if cut := strings.IndexAny(company, "|·\n"); cut >= 0 {
				company = strings.TrimSpace(company[:cut])
			}
			if len(company) >= 2 && len(company) <= 160 {
				return company
			}
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
