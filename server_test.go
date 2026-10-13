package main

import (
	"encoding/json"
	"errors"
	"net"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	emailverifier "github.com/AfterShip/email-verifier"
)

type fakeVerifier struct {
	result *emailverifier.Result
	err    error
}

func (v fakeVerifier) Verify(string) (*emailverifier.Result, error) { return v.result, v.err }

func testApplication() *application {
	return &application{baseVerifier: emailverifier.NewVerifier(), verifySlots: make(chan struct{}, 2), smtpSlots: make(chan struct{}, 2), verifyLimiter: newRequestLimiter(30, time.Minute, 100)}
}

func TestFrontendRoutesAndSecurity(t *testing.T) {
	app := testApplication()
	for _, path := range []string{"/", "/tools/json", "/tools/email", "/finder"} {
		r := httptest.NewRecorder()
		app.routes().ServeHTTP(r, httptest.NewRequest("GET", path, nil))
		if r.Code != 200 || !strings.Contains(r.Body.String(), "WeMe") {
			t.Fatalf("%s: %d %s", path, r.Code, r.Body.String())
		}
		if r.Header().Get("Cache-Control") != "no-cache" {
			t.Fatal("HTML must revalidate")
		}
		csp := r.Header().Get("Content-Security-Policy")
		if !strings.Contains(csp, "script-src 'self';") || !strings.Contains(csp, "frame-ancestors 'none'") {
			t.Fatalf("weak CSP: %s", csp)
		}
	}
	for _, path := range []string{"/assets/", "/missing.js", "/api/unknown", "/.env"} {
		r := httptest.NewRecorder()
		app.routes().ServeHTTP(r, httptest.NewRequest("GET", path, nil))
		if r.Code != 404 {
			t.Fatalf("%s unexpectedly returned %d", path, r.Code)
		}
	}
	r := httptest.NewRecorder()
	app.routes().ServeHTTP(r, httptest.NewRequest("POST", "/tools/json", nil))
	if r.Code != 405 {
		t.Fatal("frontend accepted POST")
	}
	r = httptest.NewRecorder()
	app.routes().ServeHTTP(r, httptest.NewRequest("GET", "/healthz", nil))
	if r.Code != 200 || !strings.Contains(r.Body.String(), `"smtp_enabled":false`) {
		t.Fatal("health failed")
	}
}

func TestEmailAPIValidationBoundaries(t *testing.T) {
	app := testApplication()
	for _, body := range []string{`{}`, `{"email":"x","unexpected":true}`, `{"email":"x"}{"email":"y"}`, `{"email":"` + strings.Repeat("x", 321) + `"}`, strings.Repeat(" ", 8193) + `{}`} {
		r := httptest.NewRecorder()
		app.routes().ServeHTTP(r, httptest.NewRequest("POST", "/api/verify", strings.NewReader(body)))
		if r.Code != 400 {
			t.Fatalf("invalid body accepted: %d", r.Code)
		}
	}
	r := httptest.NewRecorder()
	app.routes().ServeHTTP(r, httptest.NewRequest("POST", "/api/verify", strings.NewReader(`{"email":"not-an-email"}`)))
	if r.Code != 200 || !strings.Contains(r.Body.String(), `"reason":"bad_syntax"`) {
		t.Fatalf("syntax check: %d %s", r.Code, r.Body.String())
	}
	if r.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("email response is cacheable")
	}
}

func TestDNSFailureIsInconclusive(t *testing.T) {
	app := testApplication()
	result := &emailverifier.Result{Email: "hello@example.com", Syntax: emailverifier.Syntax{Valid: true}}
	app.baseVerifier = fakeVerifier{result: result, err: &net.DNSError{Err: "connection refused", Name: "example.com"}}
	r := httptest.NewRecorder()
	app.routes().ServeHTTP(r, httptest.NewRequest("POST", "/api/verify", strings.NewReader(`{"email":"hello@example.com"}`)))
	var response struct {
		Validity     validityAssessment
		Reachability reachabilityAssessment
		Warning      string
	}
	if err := json.Unmarshal(r.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Validity.Status != "unknown" || response.Validity.Reason != "dns_unavailable" || response.Reachability.Status != "unknown" || !response.Reachability.Retryable {
		t.Fatalf("DNS error classified as negative evidence: %s", r.Body.String())
	}
	if strings.Contains(response.Warning, "connection refused") {
		t.Fatal("internal network details leaked")
	}
	if dnsUnavailable(result, &net.DNSError{IsNotFound: true}) {
		t.Fatal("NXDOMAIN incorrectly classified as service failure")
	}
	if !dnsUnavailable(result, errors.New("DNS timeout")) {
		t.Fatal("timeout not handled")
	}
}

func TestRateLimitAndConcurrency(t *testing.T) {
	app := testApplication()
	app.verifyLimiter = newRequestLimiter(1, time.Minute, 10)
	for i, code := range []int{200, 429} {
		r := httptest.NewRecorder()
		app.routes().ServeHTTP(r, httptest.NewRequest("GET", "/api/verify?email=bad", nil))
		if r.Code != code {
			t.Fatalf("request %d: %d wanted %d", i, r.Code, code)
		}
		if code == 429 && r.Header().Get("Retry-After") != "60" {
			t.Fatal("missing retry guidance")
		}
	}
	app.verifyLimiter = nil
	app.verifySlots <- struct{}{}
	app.verifySlots <- struct{}{}
	r := httptest.NewRecorder()
	app.routes().ServeHTTP(r, httptest.NewRequest("GET", "/api/verify?email=bad", nil))
	if r.Code != 503 {
		t.Fatal("concurrency limit failed")
	}
	limiter := newRequestLimiter(2, time.Minute, 1)
	if !limiter.allow("a") || limiter.allow("b") {
		t.Fatal("unbounded limiter key growth")
	}
}

func TestForwardedHeadersRequireExplicitTrust(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "192.0.2.1:1234"
	r.Header.Set("X-Forwarded-For", "203.0.113.1")
	t.Setenv("TRUST_PROXY", "false")
	if requestIP(r) != "192.0.2.1" {
		t.Fatal("untrusted forwarded header used")
	}
	t.Setenv("TRUST_PROXY", "true")
	if requestIP(r) != "203.0.113.1" {
		t.Fatal("trusted forwarded header ignored")
	}
	r.Header.Set("X-Forwarded-For", "spoofed")
	if requestIP(r) != "192.0.2.1" {
		t.Fatal("invalid forwarded IP accepted")
	}
}
