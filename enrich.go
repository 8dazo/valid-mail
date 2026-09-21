package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
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

const (
	enrichCacheTTL     = 30 * time.Minute
	enrichMaxBodyBytes = 512 << 10
	enrichMaxPages     = 14
	enrichRatePerMin   = 10
)

var (
	emailRE = regexp.MustCompile(`(?i)[a-z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)+`)
	hrefRE  = regexp.MustCompile(`(?i)href\s*=\s*["']([^"'#]+)["']`)
	locRE   = regexp.MustCompile(`(?is)<loc>\s*([^<]+?)\s*</loc>`)
)

type findEmailRequest struct {
	Name   string `json:"name"`
	Domain string `json:"domain"`
}

type findEmailCandidate struct {
	Email         string              `json:"email"`
	Confidence    int                 `json:"confidence"`
	Pattern       string              `json:"pattern"`
	FoundPublicly bool                `json:"found_publicly"`
	Sources       []string            `json:"sources,omitempty"`
	Validity      *validityAssessment `json:"validity,omitempty"`
}

type findEmailResponse struct {
	Success            bool                 `json:"success"`
	Name               string               `json:"name"`
	Domain             string               `json:"domain"`
	ObservedEmailCount int                  `json:"observed_email_count"`
	PagesFetched       int                  `json:"pages_fetched"`
	Pattern            string               `json:"pattern"`
	PatternConfidence  int                  `json:"pattern_confidence"`
	Candidates         []findEmailCandidate `json:"candidates"`
	Note               string               `json:"note"`
	CheckedAt          time.Time             `json:"checked_at"`
}

type harvestResult struct {
	Emails       []string
	Sources      map[string][]string
	PagesFetched int
}

type harvestCacheEntry struct {
	Expires time.Time
	Result  harvestResult
}

var publicHarvestCache = struct {
	sync.Mutex
	Items map[string]harvestCacheEntry
}{Items: make(map[string]harvestCacheEntry)}

var enrichRateState = struct {
	sync.Mutex
	Hits map[string][]time.Time
}{Hits: make(map[string][]time.Time)}

var blockedCIDRs = func() []*net.IPNet {
	cidrs := []string{
		"0.0.0.0/8", "100.64.0.0/10", "127.0.0.0/8", "169.254.0.0/16",
		"192.0.0.0/24", "192.0.2.0/24", "198.18.0.0/15", "198.51.100.0/24",
		"203.0.113.0/24", "224.0.0.0/4", "240.0.0.0/4",
		"::/128", "::1/128", "fc00::/7", "fe80::/10", "ff00::/8", "2001:db8::/32",
	}
	out := make([]*net.IPNet, 0, len(cidrs))
	for _, raw := range cidrs {
		_, network, err := net.ParseCIDR(raw)
		if err == nil {
			out = append(out, network)
		}
	}
	return out
}()

var genericLocalParts = map[string]struct{}{
	"admin": {}, "billing": {}, "contact": {}, "hello": {}, "help": {}, "hr": {},
	"info": {}, "jobs": {}, "legal": {}, "marketing": {}, "noreply": {}, "no-reply": {},
	"office": {}, "press": {}, "privacy": {}, "sales": {}, "security": {}, "support": {},
	"team": {}, "webmaster": {},
}

func (app *application) handleFindEmail(w http.ResponseWriter, r *http.Request) {
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

	clientIP := requestIP(r)
	if !allowEnrichRequest(clientIP) {
		w.Header().Set("Retry-After", "60")
		writeJSON(w, http.StatusTooManyRequests, map[string]any{
			"success": false,
			"error":   "email discovery rate limit exceeded",
		})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	defer r.Body.Close()
	var req findEmailRequest
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

	domain, err := normalizePublicDomain(req.Domain)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": err.Error()})
		return
	}

	harvest, err := harvestCompanyWebsite(r.Context(), domain)
	if err != nil && harvest.PagesFetched == 0 {
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"success": false,
			"error":   "could not read public company pages",
		})
		return
	}

	pattern, patternConfidence := inferObservedPattern(harvest.Emails, domain)
	candidates := app.buildEmailCandidates(name, domain, pattern, patternConfidence, harvest)

	writeJSON(w, http.StatusOK, findEmailResponse{
		Success:            true,
		Name:               name,
		Domain:             domain,
		ObservedEmailCount: len(harvest.Emails),
		PagesFetched:       harvest.PagesFetched,
		Pattern:            pattern,
		PatternConfidence:  patternConfidence,
		Candidates:         candidates,
		Note:               "Candidates come from public company pages and company email-pattern inference. Generated candidates are not proof of mailbox ownership; use verification signals separately.",
		CheckedAt:          time.Now().UTC(),
	})
}

func requestIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		if first := strings.TrimSpace(strings.Split(forwarded, ",")[0]); first != "" {
			return first
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func allowEnrichRequest(key string) bool {
	now := time.Now()
	cutoff := now.Add(-time.Minute)
	enrichRateState.Lock()
	defer enrichRateState.Unlock()

	hits := enrichRateState.Hits[key]
	kept := hits[:0]
	for _, hit := range hits {
		if hit.After(cutoff) {
			kept = append(kept, hit)
		}
	}
	if len(kept) >= enrichRatePerMin {
		enrichRateState.Hits[key] = kept
		return false
	}
	kept = append(kept, now)
	enrichRateState.Hits[key] = kept
	return true
}

func normalizePublicDomain(raw string) (string, error) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return "", errors.New("domain is required")
	}

	candidate := raw
	if !strings.Contains(candidate, "://") {
		candidate = "https://" + candidate
	}
	u, err := url.Parse(candidate)
	if err != nil || u.Hostname() == "" {
		return "", errors.New("invalid domain")
	}
	host := strings.TrimSuffix(strings.TrimPrefix(strings.ToLower(u.Hostname()), "www."), ".")
	if net.ParseIP(host) != nil || host == "localhost" || !strings.Contains(host, ".") || len(host) > 253 {
		return "", errors.New("a public DNS domain is required")
	}
	for _, label := range strings.Split(host, ".") {
		if label == "" || len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return "", errors.New("invalid domain")
		}
		for _, r := range label {
			if !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '-' {
				return "", errors.New("invalid domain")
			}
		}
	}
	return host, nil
}

func harvestCompanyWebsite(ctx context.Context, domain string) (harvestResult, error) {
	publicHarvestCache.Lock()
	if entry, ok := publicHarvestCache.Items[domain]; ok && time.Now().Before(entry.Expires) {
		publicHarvestCache.Unlock()
		return cloneHarvest(entry.Result), nil
	}
	publicHarvestCache.Unlock()

	client := newSafeEnrichClient()
	robots := fetchRobotsRules(ctx, client, domain)

	paths := []string{
		"/", "/about", "/about-us", "/team", "/people", "/leadership", "/company",
		"/contact", "/contact-us", "/press", "/newsroom", "/careers",
		"/.well-known/security.txt", "/humans.txt", "/sitemap.xml",
	}

	targets := make([]string, 0, enrichMaxPages)
	for _, path := range paths {
		if len(targets) >= enrichMaxPages {
			break
		}
		if robots.allowed(path) {
			targets = append(targets, "https://"+domain+path)
		}
	}

	result := harvestResult{Sources: make(map[string][]string)}
	seenURLs := make(map[string]struct{})
	pageBodies := make(map[string]string)

	fetchTargets := func(urls []string) {
		sem := make(chan struct{}, 2)
		var wg sync.WaitGroup
		var mu sync.Mutex
		for _, target := range urls {
			if _, exists := seenURLs[target]; exists {
				continue
			}
			seenURLs[target] = struct{}{}
			wg.Add(1)
			go func(target string) {
				defer wg.Done()
				select {
				case sem <- struct{}{}:
				case <-ctx.Done():
					return
				}
				defer func() { <-sem }()
				body, contentType, finalURL, err := fetchPublicText(ctx, client, target)
				if err != nil || body == "" {
					return
				}
				mu.Lock()
				result.PagesFetched++
				pageBodies[finalURL] = body
				for _, email := range extractPublicEmails(body, domain) {
					if !containsString(result.Sources[email], finalURL) {
						result.Sources[email] = append(result.Sources[email], finalURL)
					}
				}
				if strings.Contains(contentType, "text/html") || strings.Contains(contentType, "xml") {
					pageBodies[finalURL] = body
				}
				mu.Unlock()
			}(target)
		}
		wg.Wait()
	}

	fetchTargets(targets)

	additional := discoverRelevantURLs(domain, pageBodies, robots)
	remaining := enrichMaxPages - len(seenURLs)
	if remaining > 0 && len(additional) > remaining {
		additional = additional[:remaining]
	}
	if remaining > 0 {
		fetchTargets(additional)
	}

	for email := range result.Sources {
		result.Emails = append(result.Emails, email)
		sort.Strings(result.Sources[email])
	}
	sort.Strings(result.Emails)

	publicHarvestCache.Lock()
	publicHarvestCache.Items[domain] = harvestCacheEntry{Expires: time.Now().Add(enrichCacheTTL), Result: cloneHarvest(result)}
	publicHarvestCache.Unlock()

	if result.PagesFetched == 0 {
		return result, errors.New("no public pages could be fetched")
	}
	return result, nil
}

func cloneHarvest(in harvestResult) harvestResult {
	out := harvestResult{
		Emails:       append([]string(nil), in.Emails...),
		Sources:      make(map[string][]string, len(in.Sources)),
		PagesFetched: in.PagesFetched,
	}
	for email, sources := range in.Sources {
		out.Sources[email] = append([]string(nil), sources...)
	}
	return out
}

type robotsRules struct {
	Disallow []string
}

func (r robotsRules) allowed(path string) bool {
	for _, blocked := range r.Disallow {
		if blocked != "" && strings.HasPrefix(path, blocked) {
			return false
		}
	}
	return true
}

func fetchRobotsRules(ctx context.Context, client *http.Client, domain string) robotsRules {
	body, _, _, err := fetchPublicText(ctx, client, "https://"+domain+"/robots.txt")
	if err != nil || body == "" {
		return robotsRules{}
	}

	var out robotsRules
	active := false
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(strings.Split(line, "#")[0])
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])
		switch key {
		case "user-agent":
			ua := strings.ToLower(value)
			active = ua == "*" || strings.Contains(ua, "valid-mail")
		case "disallow":
			if active && strings.HasPrefix(value, "/") {
				out.Disallow = append(out.Disallow, value)
			}
		}
	}
	return out
}

func newSafeEnrichClient() *http.Client {
	transport := &http.Transport{
		Proxy:                 nil,
		DialContext:           safePublicDialContext,
		MaxIdleConns:          8,
		MaxIdleConnsPerHost:   2,
		IdleConnTimeout:       20 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 6 * time.Second,
	}
	return &http.Client{
		Transport: transport,
		Timeout:   8 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return errors.New("too many redirects")
			}
			if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
				return errors.New("unsupported redirect scheme")
			}
			return validatePublicHost(req.Context(), req.URL.Hostname())
		},
	}
}

func safePublicDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	var selected net.IP
	for _, item := range ips {
		if isPublicIP(item.IP) {
			selected = item.IP
			break
		}
	}
	if selected == nil {
		return nil, errors.New("destination does not resolve to a public IP")
	}

	dialer := net.Dialer{Timeout: 5 * time.Second}
	return dialer.DialContext(ctx, network, net.JoinHostPort(selected.String(), port))
}

func validatePublicHost(ctx context.Context, host string) error {
	if host == "" || net.ParseIP(host) != nil {
		return errors.New("public hostname required")
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return err
	}
	for _, item := range ips {
		if isPublicIP(item.IP) {
			return nil
		}
	}
	return errors.New("destination is not public")
}

func isPublicIP(ip net.IP) bool {
	if ip == nil || ip.IsUnspecified() || ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() {
		return false
	}
	for _, network := range blockedCIDRs {
		if network.Contains(ip) {
			return false
		}
	}
	return true
}

func fetchPublicText(ctx context.Context, client *http.Client, rawURL string) (string, string, string, error) {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		return "", "", rawURL, errors.New("invalid public url")
	}
	if err := validatePublicHost(ctx, u.Hostname()); err != nil {
		return "", "", rawURL, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", "", rawURL, err
	}
	req.Header.Set("User-Agent", "valid-mail/1.0 public-contact-discovery")
	req.Header.Set("Accept", "text/html,text/plain,application/xml,text/xml;q=0.9,*/*;q=0.1")

	resp, err := client.Do(req)
	if err != nil {
		return "", "", rawURL, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", resp.Request.URL.String(), fmt.Errorf("http %d", resp.StatusCode)
	}
	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	if !(strings.Contains(contentType, "text/html") || strings.Contains(contentType, "text/plain") || strings.Contains(contentType, "xml") || contentType == "") {
		return "", contentType, resp.Request.URL.String(), errors.New("unsupported content type")
	}

	limited := io.LimitReader(resp.Body, enrichMaxBodyBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return "", contentType, resp.Request.URL.String(), err
	}
	if len(body) > enrichMaxBodyBytes {
		return "", contentType, resp.Request.URL.String(), errors.New("response too large")
	}
	return string(body), contentType, resp.Request.URL.String(), nil
}

func extractPublicEmails(body, domain string) []string {
	text := strings.ToLower(html.UnescapeString(body))
	replacements := []struct{ old, new string }{
		{"[at]", "@"}, {"(at)", "@"}, {" at ", "@"},
		{"[dot]", "."}, {"(dot)", "."}, {" dot ", "."},
	}
	for _, replacement := range replacements {
		text = strings.ReplaceAll(text, replacement.old, replacement.new)
	}

	seen := make(map[string]struct{})
	var out []string
	for _, email := range emailRE.FindAllString(text, -1) {
		email = strings.Trim(strings.ToLower(email), ".,;:()<>[]{}\"'")
		parts := strings.Split(email, "@")
		if len(parts) != 2 || strings.TrimPrefix(parts[1], "www.") != domain {
			continue
		}
		if _, exists := seen[email]; exists {
			continue
		}
		seen[email] = struct{}{}
		out = append(out, email)
	}
	sort.Strings(out)
	return out
}

func discoverRelevantURLs(domain string, pages map[string]string, robots robotsRules) []string {
	hints := []string{"about", "team", "people", "leadership", "contact", "press", "news", "company", "career", "staff", "founder", "management"}
	seen := make(map[string]struct{})
	var out []string

	add := func(raw, base string) {
		if len(out) >= 8 {
			return
		}
		parsed, err := url.Parse(raw)
		if err != nil {
			return
		}
		if !parsed.IsAbs() {
			baseURL, err := url.Parse(base)
			if err != nil {
				return
			}
			parsed = baseURL.ResolveReference(parsed)
		}
		if parsed.Scheme != "https" && parsed.Scheme != "http" {
			return
		}
		host := strings.TrimPrefix(strings.ToLower(parsed.Hostname()), "www.")
		if host != domain || !robots.allowed(parsed.Path) {
			return
		}
		lowerPath := strings.ToLower(parsed.Path)
		relevant := false
		for _, hint := range hints {
			if strings.Contains(lowerPath, hint) {
				relevant = true
				break
			}
		}
		if !relevant {
			return
		}
		parsed.RawQuery = ""
		parsed.Fragment = ""
		key := parsed.String()
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}

	for pageURL, body := range pages {
		for _, match := range hrefRE.FindAllStringSubmatch(body, -1) {
			if len(match) == 2 {
				add(strings.TrimSpace(html.UnescapeString(match[1])), pageURL)
			}
		}
		if strings.Contains(strings.ToLower(pageURL), "sitemap") {
			for _, match := range locRE.FindAllStringSubmatch(body, -1) {
				if len(match) == 2 {
					add(strings.TrimSpace(html.UnescapeString(match[1])), pageURL)
				}
			}
		}
	}
	sort.Strings(out)
	return out
}

func inferObservedPattern(emails []string, domain string) (string, int) {
	counts := make(map[string]int)
	personal := 0
	for _, email := range emails {
		parts := strings.Split(strings.ToLower(email), "@")
		if len(parts) != 2 || parts[1] != domain {
			continue
		}
		local := parts[0]
		if _, generic := genericLocalParts[local]; generic {
			continue
		}
		pattern := classifyLocalPattern(local)
		if pattern == "unknown" {
			continue
		}
		personal++
		counts[pattern]++
	}
	if personal == 0 {
		return "unknown", 0
	}

	order := []string{"first.last", "first", "firstlast", "f.last", "flast", "firstl", "last.first", "last"}
	best := "unknown"
	bestCount := 0
	for _, pattern := range order {
		if counts[pattern] > bestCount {
			best = pattern
			bestCount = counts[pattern]
		}
	}
	if bestCount == 0 {
		return "unknown", 0
	}
	confidence := 40
	switch {
	case bestCount >= 5:
		confidence = 95
	case bestCount >= 3:
		confidence = 85
	case bestCount >= 2:
		confidence = 70
	}
	dominance := float64(bestCount) / float64(personal)
	if dominance < 0.5 {
		confidence -= 20
	} else if dominance < 0.7 {
		confidence -= 10
	}
	if confidence < 35 {
		confidence = 35
	}
	return best, confidence
}

func classifyLocalPattern(local string) string {
	if matched, _ := regexp.MatchString(`^[a-z]\.[a-z]+$`, local); matched {
		return "f.last"
	}
	if matched, _ := regexp.MatchString(`^[a-z]+\.[a-z]+$`, local); matched {
		parts := strings.Split(local, ".")
		if len(parts) == 2 && len(parts[1]) <= 2 && len(parts[0]) >= 3 {
			return "last.first"
		}
		return "first.last"
	}
	if matched, _ := regexp.MatchString(`^[a-z]{2,}$`, local); matched {
		switch {
		case len(local) <= 5:
			return "first"
		case len(local) <= 8:
			return "firstl"
		default:
			return "firstlast"
		}
	}
	return "unknown"
}

func (app *application) buildEmailCandidates(name, domain, inferred string, inferredConfidence int, harvest harvestResult) []findEmailCandidate {
	first, last := splitPersonName(name)
	if first == "" {
		return []findEmailCandidate{}
	}

	patterns := []struct {
		Name string
		Base int
	}{
		{"first.last", 60}, {"first", 55}, {"firstlast", 50}, {"f.last", 48},
		{"flast", 45}, {"firstl", 42}, {"last.first", 38}, {"last", 35},
	}

	seen := make(map[string]struct{})
	candidates := make([]findEmailCandidate, 0, 5)
	for _, item := range patterns {
		email := applyEmailPattern(item.Name, first, last, domain)
		if email == "" {
			continue
		}
		if _, exists := seen[email]; exists {
			continue
		}
		seen[email] = struct{}{}

		score := item.Base
		if item.Name == inferred && inferredConfidence > 0 {
			score = 70 + inferredConfidence/4
		}
		sources := append([]string(nil), harvest.Sources[email]...)
		found := len(sources) > 0
		if found {
			score = 100
		}

		result, err := app.baseVerifier.Verify(email)
		if result == nil {
			continue
		}
		validity := assessValidity(result)
		if validity == nil || validity.Status == "invalid" {
			continue
		}
		if err != nil && !result.Syntax.Valid {
			continue
		}
		if validity.Status == "risky" && score < 100 {
			score -= 10
		}
		if score < 0 {
			score = 0
		}

		candidates = append(candidates, findEmailCandidate{
			Email:         email,
			Confidence:    score,
			Pattern:       item.Name,
			FoundPublicly: found,
			Sources:       sources,
			Validity:      validity,
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Confidence == candidates[j].Confidence {
			return candidates[i].Email < candidates[j].Email
		}
		return candidates[i].Confidence > candidates[j].Confidence
	})
	if len(candidates) > 5 {
		candidates = candidates[:5]
	}
	return candidates
}

func splitPersonName(name string) (string, string) {
	fields := strings.Fields(name)
	if len(fields) == 0 {
		return "", ""
	}
	first := normalizeNameToken(fields[0])
	last := ""
	if len(fields) > 1 {
		last = normalizeNameToken(fields[len(fields)-1])
	}
	return first, last
}

func normalizeNameToken(input string) string {
	var builder strings.Builder
	for _, r := range strings.ToLower(input) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func applyEmailPattern(pattern, first, last, domain string) string {
	if first == "" {
		return ""
	}
	initial := first[:1]
	switch pattern {
	case "first.last":
		if last != "" {
			return first + "." + last + "@" + domain
		}
	case "first":
		return first + "@" + domain
	case "firstlast":
		if last != "" {
			return first + last + "@" + domain
		}
	case "f.last":
		if last != "" {
			return initial + "." + last + "@" + domain
		}
	case "flast":
		if last != "" {
			return initial + last + "@" + domain
		}
	case "firstl":
		if last != "" {
			return first + last[:1] + "@" + domain
		}
	case "last.first":
		if last != "" {
			return last + "." + first + "@" + domain
		}
	case "last":
		if last != "" {
			return last + "@" + domain
		}
	}
	return ""
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
