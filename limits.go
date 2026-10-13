package main

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	emailverifier "github.com/AfterShip/email-verifier"
)

type requestLimiter struct {
	sync.Mutex
	hits    map[string][]time.Time
	limit   int
	window  time.Duration
	maxKeys int
}

func newRequestLimiter(limit int, window time.Duration, maxKeys int) *requestLimiter {
	return &requestLimiter{hits: make(map[string][]time.Time), limit: limit, window: window, maxKeys: maxKeys}
}

func (l *requestLimiter) allow(key string) bool {
	now := time.Now()
	cutoff := now.Add(-l.window)
	l.Lock()
	defer l.Unlock()
	if len(l.hits) >= l.maxKeys {
		for k, hits := range l.hits {
			if len(hits) == 0 || hits[len(hits)-1].Before(cutoff) {
				delete(l.hits, k)
			}
		}
		if _, exists := l.hits[key]; !exists && len(l.hits) >= l.maxKeys {
			return false
		}
	}
	hits := l.hits[key]
	kept := hits[:0]
	for _, hit := range hits {
		if hit.After(cutoff) {
			kept = append(kept, hit)
		}
	}
	if len(kept) >= l.limit {
		l.hits[key] = kept
		return false
	}
	l.hits[key] = append(kept, now)
	return true
}

func requireJSONEnd(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("expected end of JSON body")
	}
	return nil
}

func dnsUnavailable(result *emailverifier.Result, err error) bool {
	if err == nil || result == nil || !result.Syntax.Valid || result.Disposable || result.HasMxRecords {
		return false
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) && dnsErr.IsNotFound {
		return false
	}
	return true
}
