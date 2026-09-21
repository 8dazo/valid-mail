package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/proxy"
)

const defaultProxySource = "https://api.proxyscrape.com/v4/free-proxy-list/get?request=display_proxies&protocol=socks5&proxy_format=protocolipport&format=text"

type proxyPool struct {
	mu            sync.RWMutex
	healthy       []string
	next          int
	lastRefresh   time.Time
	lastError     string
	sourceURL     string
	maxCandidates int
	maxHealthy    int
	concurrency   int
	probeTimeout  time.Duration
	refreshEvery  time.Duration
	client        *http.Client
	refreshCh     chan struct{}
}

type proxyPoolStats struct {
	Healthy     int       `json:"healthy"`
	LastRefresh time.Time `json:"last_refresh,omitempty"`
	LastError   string    `json:"last_error,omitempty"`
}

func newProxyPool() *proxyPool {
	return &proxyPool{
		sourceURL:     envString("PROXY_SOURCE_URL", defaultProxySource),
		maxCandidates: envInt("PROXY_MAX_CANDIDATES", 40, 1, 250),
		maxHealthy:    envInt("PROXY_MAX_HEALTHY", 8, 1, 50),
		concurrency:   envInt("PROXY_PROBE_CONCURRENCY", 12, 1, 50),
		probeTimeout:  envDuration("PROXY_PROBE_TIMEOUT", 4*time.Second, time.Second, 15*time.Second),
		refreshEvery:  envDuration("PROXY_REFRESH_INTERVAL", 5*time.Minute, 30*time.Second, time.Hour),
		client: &http.Client{
			Timeout: 12 * time.Second,
		},
		refreshCh: make(chan struct{}, 1),
	}
}

func (p *proxyPool) run() {
	p.refresh()
	ticker := time.NewTicker(p.refreshEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			p.refresh()
		case <-p.refreshCh:
			p.refresh()
		}
	}
}

func (p *proxyPool) triggerRefresh() {
	select {
	case p.refreshCh <- struct{}{}:
	default:
	}
}

func (p *proxyPool) nextProxy(exclude map[string]struct{}) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.healthy) == 0 {
		return ""
	}
	for i := 0; i < len(p.healthy); i++ {
		idx := p.next % len(p.healthy)
		p.next = (p.next + 1) % len(p.healthy)
		candidate := p.healthy[idx]
		if _, skipped := exclude[candidate]; !skipped {
			return candidate
		}
	}
	return ""
}

func (p *proxyPool) markBad(proxyURI string) {
	if proxyURI == "" {
		return
	}
	p.mu.Lock()
	filtered := p.healthy[:0]
	for _, candidate := range p.healthy {
		if candidate != proxyURI {
			filtered = append(filtered, candidate)
		}
	}
	p.healthy = append([]string(nil), filtered...)
	if len(p.healthy) == 0 {
		p.next = 0
	}
	empty := len(p.healthy) == 0
	p.mu.Unlock()
	if empty {
		p.triggerRefresh()
	}
}

func (p *proxyPool) stats() proxyPoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return proxyPoolStats{
		Healthy:     len(p.healthy),
		LastRefresh: p.lastRefresh,
		LastError:   p.lastError,
	}
}

func (p *proxyPool) refresh() {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	candidates, err := p.fetchCandidates(ctx)
	if err != nil {
		p.setRefreshResult(nil, err)
		log.Printf("proxy pool refresh failed: %v", err)
		return
	}
	if len(candidates) == 0 {
		err := errors.New("proxy source returned no SOCKS5 candidates")
		p.setRefreshResult(nil, err)
		log.Printf("proxy pool refresh failed: %v", err)
		return
	}

	targets := smtpProbeTargets(ctx)
	if len(targets) == 0 {
		err := errors.New("could not resolve SMTP probe targets")
		p.setRefreshResult(nil, err)
		log.Printf("proxy pool refresh failed: %v", err)
		return
	}

	sem := make(chan struct{}, p.concurrency)
	results := make(chan string, len(candidates))
	var wg sync.WaitGroup

	for _, candidate := range candidates {
		candidate := candidate
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()
			if p.probe(ctx, candidate, targets) {
				select {
				case results <- candidate:
				case <-ctx.Done():
				}
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(results)
		close(done)
	}()

	healthy := make([]string, 0, p.maxHealthy)
	for len(healthy) < p.maxHealthy {
		select {
		case candidate, ok := <-results:
			if !ok {
				p.setRefreshResult(healthy, nil)
				log.Printf("proxy pool refreshed: candidates=%d healthy=%d", len(candidates), len(healthy))
				return
			}
			healthy = append(healthy, candidate)
		case <-ctx.Done():
			p.setRefreshResult(healthy, ctx.Err())
			log.Printf("proxy pool refresh timed out: candidates=%d healthy=%d", len(candidates), len(healthy))
			return
		}
	}

	p.setRefreshResult(healthy, nil)
	log.Printf("proxy pool refreshed: candidates=%d healthy=%d", len(candidates), len(healthy))
	go func() { <-done }()
}

func (p *proxyPool) setRefreshResult(healthy []string, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if healthy != nil {
		p.healthy = append([]string(nil), healthy...)
		if p.next >= len(p.healthy) {
			p.next = 0
		}
	}
	p.lastRefresh = time.Now().UTC()
	if err != nil {
		p.lastError = err.Error()
	} else {
		p.lastError = ""
	}
}

func (p *proxyPool) fetchCandidates(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.sourceURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "valid-mail/1.0")
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("proxy source returned HTTP %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 512*1024)
	seen := make(map[string]struct{})
	out := make([]string, 0, p.maxCandidates)
	for scanner.Scan() {
		candidate, ok := normalizeProxy(scanner.Text())
		if !ok {
			continue
		}
		if _, exists := seen[candidate]; exists {
			continue
		}
		seen[candidate] = struct{}{}
		out = append(out, candidate)
		if len(out) >= p.maxCandidates {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func normalizeProxy(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.HasPrefix(raw, "#") {
		return "", false
	}
	if !strings.Contains(raw, "://") {
		raw = "socks5://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "socks5" && u.Scheme != "socks5h") {
		return "", false
	}
	if u.Hostname() == "" || u.Port() == "" {
		return "", false
	}
	if _, err := net.LookupPort("tcp", u.Port()); err != nil {
		return "", false
	}
	return u.String(), true
}

func smtpProbeTargets(ctx context.Context) []string {
	domains := []string{"gmail.com", "outlook.com"}
	targets := make([]string, 0, len(domains))
	for _, domain := range domains {
		lookupCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		mx, err := net.DefaultResolver.LookupMX(lookupCtx, domain)
		cancel()
		if err != nil || len(mx) == 0 {
			continue
		}
		host := strings.TrimSuffix(mx[0].Host, ".")
		if host != "" {
			targets = append(targets, net.JoinHostPort(host, "25"))
		}
	}
	return targets
}

func (p *proxyPool) probe(parent context.Context, proxyURI string, targets []string) bool {
	u, err := url.Parse(proxyURI)
	if err != nil {
		return false
	}
	forward := &net.Dialer{Timeout: p.probeTimeout}
	dialer, err := proxy.FromURL(u, forward)
	if err != nil {
		return false
	}

	for _, target := range targets {
		ctx, cancel := context.WithTimeout(parent, p.probeTimeout)
		conn, err := dialProxyContext(ctx, dialer, target)
		cancel()
		if err != nil {
			continue
		}
		_ = conn.SetDeadline(time.Now().Add(p.probeTimeout))
		line, readErr := bufio.NewReaderSize(conn, 512).ReadString('\n')
		_ = conn.Close()
		if readErr == nil && strings.HasPrefix(strings.TrimSpace(line), "220") {
			return true
		}
	}
	return false
}

func dialProxyContext(ctx context.Context, dialer proxy.Dialer, target string) (net.Conn, error) {
	if d, ok := dialer.(proxy.ContextDialer); ok {
		return d.DialContext(ctx, "tcp", target)
	}
	type result struct {
		conn net.Conn
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		conn, err := dialer.Dial("tcp", target)
		ch <- result{conn: conn, err: err}
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-ch:
		return result.conn, result.err
	}
}
