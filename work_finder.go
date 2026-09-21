package main

import (
	"context"
	"encoding/json"
	"errors"
	"html"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

type findWorkEmailRequest struct {
	Name    string `json:"name"`
	Company string `json:"company"`
	Domain  string `json:"domain,omitempty"`
}

type companyDomainCandidate struct {
	Domain     string   `json:"domain"`
	Confidence int      `json:"confidence"`
	Reasons    []string `json:"reasons,omitempty"`
}

type companyDomainResolution struct {
	Domain       string                   `json:"domain"`
	Confidence   int                      `json:"confidence"`
	Method       string                   `json:"method"`
	Alternatives []companyDomainCandidate `json:"alternatives,omitempty"`
}

type findWorkEmailResponse struct {
	Success            bool                     `json:"success"`
	Name               string                   `json:"name"`
	Company            string                   `json:"company,omitempty"`
	Domain             string                   `json:"domain"`
	DomainResolution   *companyDomainResolution `json:"domain_resolution,omitempty"`
	ObservedEmailCount int                      `json:"observed_email_count"`
	PagesFetched       int                      `json:"pages_fetched"`
	Pattern            string                   `json:"pattern"`
	PatternConfidence  int                      `json:"pattern_confidence"`
	Candidates         []findEmailCandidate     `json:"candidates"`
	Note               string                   `json:"note"`
	CheckedAt          time.Time                `json:"checked_at"`
}

var companyTitleRE = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)

func (app *application) handleFindWorkEmail(w http.ResponseWriter, r *http.Request) {
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
		writeJSON(w, http.StatusTooManyRequests, map[string]any{
			"success": false,
			"error":   "email discovery rate limit exceeded",
		})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	defer r.Body.Close()

	var req findWorkEmailRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "invalid JSON body"})
		return
	}

	name := strings.TrimSpace(req.Name)
	company := strings.TrimSpace(req.Company)
	if len(name) < 2 || len(name) > 120 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "name must be between 2 and 120 characters"})
		return
	}
	if len(company) > 160 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "company is too long"})
		return
	}

	var resolution *companyDomainResolution
	var domain string
	var err error
	if strings.TrimSpace(req.Domain) != "" {
		domain, err = normalizePublicDomain(req.Domain)
		if err == nil {
			resolution = &companyDomainResolution{Domain: domain, Confidence: 100, Method: "provided_domain"}
		}
	} else {
		if company == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "company or domain is required"})
			return
		}
		resolved, resolveErr := resolveCompanyDomain(r.Context(), company)
		if resolveErr != nil {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
				"success": false,
				"error":   resolveErr.Error(),
				"hint":    "provide the company domain explicitly when automatic resolution is uncertain",
			})
			return
		}
		resolution = &resolved
		domain = resolved.Domain
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": err.Error()})
		return
	}

	harvest, harvestErr := harvestCompanyWebsite(r.Context(), domain)
	if harvestErr != nil && harvest.PagesFetched == 0 {
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"success":          false,
			"error":            "official domain was resolved, but its public pages could not be read",
			"domain":           domain,
			"domain_resolution": resolution,
		})
		return
	}

	pattern, patternConfidence := inferObservedPattern(harvest.Emails, domain)
	candidates := app.buildEmailCandidates(name, domain, pattern, patternConfidence, harvest)

	writeJSON(w, http.StatusOK, findWorkEmailResponse{
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
		Note:               "The finder uses public company pages, official-domain heuristics, observed company email patterns, and SMTP-independent validity checks. A generated candidate is not proof that a mailbox belongs to the named person.",
		CheckedAt:          time.Now().UTC(),
	})
}

func resolveCompanyDomain(ctx context.Context, company string) (companyDomainResolution, error) {
	company = strings.TrimSpace(company)
	if company == "" {
		return companyDomainResolution{}, errors.New("company is required")
	}

	// If the company field itself is a URL/domain, treat it as strong user input.
	if strings.Contains(company, ".") {
		if domain, err := normalizePublicDomain(company); err == nil {
			return companyDomainResolution{Domain: domain, Confidence: 100, Method: "company_as_domain"}, nil
		}
	}

	roots := companyDomainRoots(company)
	if len(roots) == 0 {
		return companyDomainResolution{}, errors.New("could not derive a domain from company name")
	}

	tlds := []string{"com", "ai", "io", "co", "in", "org", "net", "tech"}
	candidateSet := make(map[string]struct{})
	var domains []string
	for _, root := range roots {
		for _, tld := range tlds {
			domain := root + "." + tld
			if _, seen := candidateSet[domain]; seen {
				continue
			}
			candidateSet[domain] = struct{}{}
			domains = append(domains, domain)
			if len(domains) >= 24 {
				break
			}
		}
		if len(domains) >= 24 {
			break
		}
	}

	client := newSafeEnrichClient()
	type scored struct {
		candidate companyDomainCandidate
	}
	results := make(chan scored, len(domains))
	sem := make(chan struct{}, 4)
	var wg sync.WaitGroup

	for _, candidateDomain := range domains {
		candidateDomain := candidateDomain
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()

			candidate, ok := scoreCompanyDomain(ctx, client, company, candidateDomain)
			if ok {
				select {
				case results <- scored{candidate: candidate}:
				case <-ctx.Done():
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	byDomain := make(map[string]companyDomainCandidate)
	for item := range results {
		current, exists := byDomain[item.candidate.Domain]
		if !exists || item.candidate.Confidence > current.Confidence {
			byDomain[item.candidate.Domain] = item.candidate
		}
	}

	candidates := make([]companyDomainCandidate, 0, len(byDomain))
	for _, candidate := range byDomain {
		candidates = append(candidates, candidate)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Confidence == candidates[j].Confidence {
			return candidates[i].Domain < candidates[j].Domain
		}
		return candidates[i].Confidence > candidates[j].Confidence
	})

	if len(candidates) == 0 || candidates[0].Confidence < 55 {
		return companyDomainResolution{}, errors.New("could not resolve the official company domain confidently")
	}

	alternatives := append([]companyDomainCandidate(nil), candidates...)
	if len(alternatives) > 4 {
		alternatives = alternatives[:4]
	}
	return companyDomainResolution{
		Domain:       candidates[0].Domain,
		Confidence:   candidates[0].Confidence,
		Method:       "verified_domain_candidates",
		Alternatives: alternatives,
	}, nil
}

func companyDomainRoots(company string) []string {
	legalSuffixes := map[string]struct{}{
		"inc": {}, "incorporated": {}, "llc": {}, "ltd": {}, "limited": {},
		"pvt": {}, "private": {}, "corp": {}, "corporation": {}, "plc": {},
		"gmbh": {}, "company": {}, "co": {},
	}

	var words []string
	for _, field := range strings.Fields(company) {
		var b strings.Builder
		for _, r := range strings.ToLower(field) {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				b.WriteRune(r)
			}
		}
		word := b.String()
		if word != "" {
			words = append(words, word)
		}
	}
	for len(words) > 1 {
		if _, legal := legalSuffixes[words[len(words)-1]]; !legal {
			break
		}
		words = words[:len(words)-1]
	}
	if len(words) == 0 {
		return nil
	}

	seen := make(map[string]struct{})
	var roots []string
	add := func(root string) {
		root = strings.Trim(root, "-")
		if len(root) < 2 || len(root) > 63 {
			return
		}
		if _, ok := seen[root]; ok {
			return
		}
		seen[root] = struct{}{}
		roots = append(roots, root)
	}

	add(strings.Join(words, ""))
	if len(words) > 1 {
		add(strings.Join(words, "-"))
		add(words[0])
		var acronym strings.Builder
		for _, word := range words {
			if word != "" {
				acronym.WriteByte(word[0])
			}
		}
		add(acronym.String())
	}
	return roots
}

func scoreCompanyDomain(ctx context.Context, client *http.Client, company, candidateDomain string) (companyDomainCandidate, bool) {
	lookupCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := validatePublicHost(lookupCtx, candidateDomain); err != nil {
		return companyDomainCandidate{}, false
	}

	score := 20
	reasons := []string{"public DNS resolves"}
	requestedURL := "https://" + candidateDomain + "/"
	body, _, finalURL, fetchErr := fetchPublicText(ctx, client, requestedURL)
	finalDomain := candidateDomain
	if fetchErr == nil && body != "" {
		score += 12
		reasons = append(reasons, "homepage reachable")
		if parsed, err := url.Parse(finalURL); err == nil {
			if normalized, err := normalizePublicDomain(parsed.Hostname()); err == nil {
				finalDomain = normalized
			}
		}

		companyCompact := compactComparable(company)
		bodySample := body
		if len(bodySample) > 96*1024 {
			bodySample = bodySample[:96*1024]
		}
		pageCompact := compactComparable(html.UnescapeString(bodySample))
		if companyCompact != "" && strings.Contains(pageCompact, companyCompact) {
			score += 38
			reasons = append(reasons, "company name appears on homepage")
		} else if companyTokenCoverage(company, bodySample) >= 0.75 {
			score += 24
			reasons = append(reasons, "company name tokens match homepage")
		}
		if match := companyTitleRE.FindStringSubmatch(body); len(match) == 2 {
			titleCompact := compactComparable(html.UnescapeString(match[1]))
			if companyCompact != "" && strings.Contains(titleCompact, companyCompact) {
				score += 12
				reasons = append(reasons, "company name matches page title")
			}
		}
	}

	roots := companyDomainRoots(company)
	rootLabel := strings.Split(candidateDomain, ".")[0]
	for i, root := range roots {
		if rootLabel == root {
			bonus := 14
			if i == 0 {
				bonus = 20
			}
			score += bonus
			reasons = append(reasons, "domain label matches company name")
			break
		}
	}

	mxCtx, mxCancel := context.WithTimeout(ctx, 3*time.Second)
	mx, mxErr := net.DefaultResolver.LookupMX(mxCtx, finalDomain)
	mxCancel()
	if mxErr == nil && len(mx) > 0 {
		score += 10
		reasons = append(reasons, "domain accepts email")
	}

	if score > 100 {
		score = 100
	}
	return companyDomainCandidate{Domain: finalDomain, Confidence: score, Reasons: reasons}, true
}

func compactComparable(input string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(input) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func companyTokenCoverage(company, page string) float64 {
	pageLower := strings.ToLower(page)
	ignored := map[string]struct{}{
		"inc": {}, "incorporated": {}, "llc": {}, "ltd": {}, "limited": {},
		"pvt": {}, "private": {}, "corp": {}, "corporation": {}, "company": {}, "co": {},
	}
	var tokens []string
	for _, raw := range strings.Fields(company) {
		token := compactComparable(raw)
		if len(token) < 2 {
			continue
		}
		if _, skip := ignored[token]; skip {
			continue
		}
		tokens = append(tokens, token)
	}
	if len(tokens) == 0 {
		return 0
	}
	matched := 0
	for _, token := range tokens {
		if strings.Contains(pageLower, token) {
			matched++
		}
	}
	return float64(matched) / float64(len(tokens))
}
