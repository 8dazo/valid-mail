package main

import (
	"context"
	"encoding/json"
	"errors"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const maxLinkedInDiscoveryCandidates = 6

type findByNameRequest struct {
	Name string `json:"name"`
}

type discoveredLinkedInProfile struct {
	URL        string                `json:"url"`
	Score      int                   `json:"score"`
	Name       string                `json:"name,omitempty"`
	Headline   string                `json:"headline,omitempty"`
	Company    string                `json:"company,omitempty"`
	Status     string                `json:"status"`
	SearchText string                `json:"search_text,omitempty"`
	Signal     *linkedInPublicSignal `json:"signal,omitempty"`
}

type linkedInProfileResolution struct {
	Selected     *discoveredLinkedInProfile  `json:"selected,omitempty"`
	Alternatives []discoveredLinkedInProfile `json:"alternatives,omitempty"`
	Confidence   int                         `json:"confidence"`
	Ambiguous    bool                        `json:"ambiguous"`
	Method       string                      `json:"method"`
}

type findByNameResponse struct {
	Success   bool                       `json:"success"`
	Profile   linkedInProfileResolution  `json:"profile_resolution"`
	Email     *findLinkedInEmailResponse `json:"email,omitempty"`
	Note      string                     `json:"note"`
	CheckedAt time.Time                  `json:"checked_at"`
}

type publicSearchHit struct {
	URL  string
	Text string
}

var resultAnchorRE = regexp.MustCompile(`(?is)<a\b[^>]*href\s*=\s*["']([^"']+)["'][^>]*>(.*?)</a>`)
var stripTagRE = regexp.MustCompile(`(?is)<[^>]+>`)

func (app *application) handleFindByName(w http.ResponseWriter, r *http.Request) {
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
	var req findByNameRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "invalid JSON body"})
		return
	}
	name := strings.TrimSpace(req.Name)
	if len(name) < 2 || len(name) > 120 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "name must be between 2 and 120 characters"})
		return
	}

	resolution, err := discoverLinkedInProfileByName(r.Context(), name)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"success":            false,
			"error":              err.Error(),
			"profile_resolution": resolution,
		})
		return
	}
	if resolution.Selected == nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"success":            false,
			"error":              "no sufficiently strong public LinkedIn profile match was found",
			"profile_resolution": resolution,
		})
		return
	}

	selected := resolution.Selected
	if resolution.Ambiguous {
		writeJSON(w, http.StatusConflict, map[string]any{
			"success":            false,
			"error":              "multiple LinkedIn profiles matched this name too closely; provide a company, domain, or LinkedIn URL to disambiguate",
			"profile_resolution": resolution,
		})
		return
	}
	if selected.Company == "" {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"success":            false,
			"error":              "the matched public LinkedIn profile did not expose a usable current company",
			"profile_resolution": resolution,
		})
		return
	}

	emailResult, enrichErr := app.enrichDiscoveredLinkedIn(r.Context(), name, *selected)
	if enrichErr != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"success":            false,
			"error":              enrichErr.Error(),
			"profile_resolution": resolution,
		})
		return
	}

	writeJSON(w, http.StatusOK, findByNameResponse{
		Success: true,
		Profile: resolution,
		Email:   emailResult,
		Note: "The service first discovers public LinkedIn profile URLs from LinkedIn's public people directory, with public web search as a fallback. It then uses only publicly accessible profile metadata as an identity/company signal before running the existing company-domain and email-pattern enrichment pipeline. Login walls or access challenges are not bypassed.",
		CheckedAt: time.Now().UTC(),
	})
}

func discoverLinkedInProfileByName(ctx context.Context, name string) (linkedInProfileResolution, error) {
	hits, method, err := searchPublicLinkedInProfiles(ctx, name)
	if err != nil {
		return linkedInProfileResolution{Method: method}, err
	}
	if len(hits) == 0 {
		return linkedInProfileResolution{Method: method}, errors.New("no public LinkedIn profile results were found for this name")
	}

	profiles := make([]discoveredLinkedInProfile, len(hits))
	sem := make(chan struct{}, 3)
	var wg sync.WaitGroup
	for i, hit := range hits {
		i, hit := i, hit
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()

			signal := fetchLinkedInPublicSignal(ctx, hit.URL)
			profiles[i] = scoreDiscoveredProfile(name, hit, signal)
		}()
	}
	wg.Wait()

	filtered := profiles[:0]
	for _, profile := range profiles {
		if profile.URL != "" {
			filtered = append(filtered, profile)
		}
	}
	profiles = filtered
	sort.SliceStable(profiles, func(i, j int) bool {
		if profiles[i].Score == profiles[j].Score {
			return profiles[i].URL < profiles[j].URL
		}
		return profiles[i].Score > profiles[j].Score
	})

	resolution := linkedInProfileResolution{Method: method}
	if len(profiles) == 0 {
		return resolution, errors.New("public LinkedIn candidates could not be evaluated")
	}
	if len(profiles) > 5 {
		profiles = profiles[:5]
	}
	resolution.Alternatives = append([]discoveredLinkedInProfile(nil), profiles...)
	if profiles[0].Score < 72 {
		resolution.Confidence = profiles[0].Score
		return resolution, errors.New("no sufficiently strong LinkedIn identity match was found")
	}

	resolution.Selected = &profiles[0]
	resolution.Confidence = profiles[0].Score
	if len(profiles) > 1 && profiles[1].Score >= profiles[0].Score-8 && profiles[1].Score >= 72 {
		resolution.Ambiguous = true
	}
	return resolution, nil
}

func searchPublicLinkedInProfiles(ctx context.Context, name string) ([]publicSearchHit, string, error) {
	if hits, err := searchLinkedInPublicDirectory(ctx, name); err == nil && len(hits) > 0 {
		return hits, "linkedin_public_directory", nil
	}
	if hits, err := searchDuckDuckGoLinkedIn(ctx, name); err == nil && len(hits) > 0 {
		return hits, "public_web_search", nil
	}
	return nil, "linkedin_public_directory", errors.New("public LinkedIn profile discovery was unavailable")
}

func searchLinkedInPublicDirectory(ctx context.Context, name string) ([]publicSearchHit, error) {
	tokens := normalizedNameTokens(name)
	if len(tokens) < 2 {
		return nil, errors.New("public LinkedIn directory needs a first and last name")
	}
	first := tokens[0]
	last := tokens[len(tokens)-1]
	directoryURL := "https://www.linkedin.com/pub/dir/" + url.PathEscape(first) + "/" + url.PathEscape(last)
	client := newSafeEnrichClient()
	body, contentType, finalURL, err := fetchPublicText(ctx, client, directoryURL)
	if err != nil {
		return nil, err
	}
	if !strings.Contains(contentType, "text/html") && contentType != "" {
		return nil, errors.New("LinkedIn directory returned an unsupported response")
	}
	return extractLinkedInHits(finalURL, body, maxLinkedInDiscoveryCandidates), nil
}

func searchDuckDuckGoLinkedIn(ctx context.Context, name string) ([]publicSearchHit, error) {
	client := newSafeEnrichClient()
	query := `site:linkedin.com/in/ "` + name + `"`
	searchURL := "https://html.duckduckgo.com/html/?q=" + url.QueryEscape(query)
	body, contentType, finalURL, err := fetchPublicText(ctx, client, searchURL)
	if err != nil {
		return nil, errors.New("public profile search was unavailable")
	}
	if !strings.Contains(contentType, "text/html") && contentType != "" {
		return nil, errors.New("public profile search returned an unsupported response")
	}
	return extractLinkedInHits(finalURL, body, maxLinkedInDiscoveryCandidates), nil
}

func extractLinkedInHits(baseURL, body string, limit int) []publicSearchHit {
	base, _ := url.Parse(baseURL)
	seen := make(map[string]struct{})
	var hits []publicSearchHit
	for _, match := range resultAnchorRE.FindAllStringSubmatch(body, -1) {
		if len(match) != 3 {
			continue
		}
		rawTarget := html.UnescapeString(strings.TrimSpace(match[1]))
		target := decodeSearchResultURL(rawTarget)
		if target == "" && base != nil {
			if relative, err := url.Parse(rawTarget); err == nil {
				target = base.ResolveReference(relative).String()
			}
		}
		if target == "" {
			continue
		}
		normalized, err := normalizeLinkedInProfileURL(target)
		if err != nil {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		text := stripTagRE.ReplaceAllString(match[2], " ")
		text = strings.Join(strings.Fields(html.UnescapeString(text)), " ")
		hits = append(hits, publicSearchHit{URL: normalized, Text: text})
		if len(hits) >= limit {
			break
		}
	}
	return hits
}

func decodeSearchResultURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "//") {
		raw = "https:" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	if host == "linkedin.com" || host == "www.linkedin.com" {
		return raw
	}
	if strings.Contains(host, "duckduckgo.com") {
		if redirected := strings.TrimSpace(u.Query().Get("uddg")); redirected != "" {
			return redirected
		}
	}
	return ""
}

func scoreDiscoveredProfile(wantedName string, hit publicSearchHit, signal *linkedInPublicSignal) discoveredLinkedInProfile {
	profile := discoveredLinkedInProfile{URL: hit.URL, Status: "unavailable", SearchText: hit.Text, Signal: signal}
	if signal != nil {
		profile.Status = signal.Status
		profile.Name = signal.Name
		profile.Headline = signal.Headline
		profile.Company = signal.Company
	}

	score := nameMatchScore(wantedName, firstNonEmpty(profile.Name, hit.Text))
	if profile.Status == "fetched" {
		score += 8
	}
	if profile.Company != "" {
		score += 5
	}
	if profile.Headline != "" {
		score += 2
	}
	if score > 100 {
		score = 100
	}
	profile.Score = score
	return profile
}

func nameMatchScore(wanted, candidate string) int {
	wantTokens := normalizedNameTokens(wanted)
	candidateTokens := normalizedNameTokens(candidate)
	if len(wantTokens) == 0 || len(candidateTokens) == 0 {
		return 0
	}
	wantJoined := strings.Join(wantTokens, " ")
	candidateJoined := strings.Join(candidateTokens, " ")
	if candidateJoined == wantJoined {
		return 90
	}
	if strings.Contains(candidateJoined, wantJoined) {
		return 84
	}

	wantSet := make(map[string]struct{}, len(wantTokens))
	for _, token := range wantTokens {
		wantSet[token] = struct{}{}
	}
	matched := 0
	for _, token := range candidateTokens {
		if _, ok := wantSet[token]; ok {
			matched++
		}
	}
	coverage := float64(matched) / float64(len(wantTokens))
	switch {
	case coverage >= 1:
		return 80
	case coverage >= 0.75:
		return 68
	case coverage >= 0.5:
		return 52
	default:
		return 20
	}
}

func normalizedNameTokens(value string) []string {
	value = strings.ToLower(html.UnescapeString(stripTagRE.ReplaceAllString(value, " ")))
	var tokens []string
	for _, field := range strings.Fields(value) {
		cleaned := normalizeNameToken(field)
		if cleaned != "" && cleaned != "linkedin" {
			tokens = append(tokens, cleaned)
		}
	}
	return tokens
}

func (app *application) enrichDiscoveredLinkedIn(ctx context.Context, requestedName string, profile discoveredLinkedInProfile) (*findLinkedInEmailResponse, error) {
	company := strings.TrimSpace(profile.Company)
	if company == "" {
		return nil, errors.New("matched profile did not expose a current company")
	}
	name := strings.TrimSpace(profile.Name)
	if name == "" {
		name = strings.TrimSpace(requestedName)
	}

	resolution, err := resolveCompanyDomain(ctx, company)
	if err != nil {
		return nil, err
	}
	harvest, harvestErr := harvestCompanyWebsite(ctx, resolution.Domain)
	if harvestErr != nil && harvest.PagesFetched == 0 {
		return nil, errors.New("official company domain was resolved, but its public pages could not be read")
	}
	pattern, patternConfidence := inferObservedPattern(harvest.Emails, resolution.Domain)
	candidates := app.buildEmailCandidates(name, resolution.Domain, pattern, patternConfidence, harvest)

	base := findWorkEmailResponse{
		Success:            true,
		Name:               name,
		Company:            company,
		Domain:             resolution.Domain,
		DomainResolution:   &resolution,
		ObservedEmailCount: len(harvest.Emails),
		PagesFetched:       harvest.PagesFetched,
		Pattern:            pattern,
		PatternConfidence:  patternConfidence,
		Candidates:         candidates,
		Note:               "The selected public LinkedIn profile is used only for identity and company context. Email candidates come from public company pages, domain resolution, observed company email patterns, and SMTP-independent validity checks.",
		CheckedAt:          time.Now().UTC(),
	}
	return &findLinkedInEmailResponse{findWorkEmailResponse: base, LinkedIn: profile.Signal}, nil
}
