package main

import (
	_ "embed"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	emailverifier "github.com/AfterShip/email-verifier"
)

//go:embed web/index.html
var indexHTML []byte

type application struct {
	baseVerifier     *emailverifier.Verifier
	directSMTP       bool
	proxySMTP        bool
	proxyPool        *proxyPool
	fromEmail        string
	helloName        string
	connectTimeout   time.Duration
	operationTimeout time.Duration
	maxSMTPAttempts  int
	smtpSlots        chan struct{}
}

type verifyRequest struct {
	Email string `json:"email"`
}

type verifyResponse struct {
	Success        bool                  `json:"success"`
	Verification   *emailverifier.Result `json:"verification,omitempty"`
	SMTPEnabled    bool                  `json:"smtp_enabled"`
	SMTPAttempted  bool                  `json:"smtp_attempted"`
	SMTPViaProxy   bool                  `json:"smtp_via_proxy"`
	HealthyProxies int                   `json:"healthy_proxies,omitempty"`
	Warning        string                `json:"warning,omitempty"`
	CheckedAt      time.Time             `json:"checked_at"`
	DurationMS     int64                 `json:"duration_ms"`
}

func main() {
	directSMTP := envBool("ENABLE_SMTP", false)
	proxySMTP := envBool("ENABLE_PROXY_SMTP", true)
	baseVerifier := emailverifier.NewVerifier().EnableDomainSuggest()

	if envBool("AUTO_UPDATE_DISPOSABLE", false) {
		baseVerifier.EnableAutoUpdateDisposable()
	}

	app := &application{
		baseVerifier:     baseVerifier,
		directSMTP:       directSMTP,
		proxySMTP:        proxySMTP,
		fromEmail:        envString("SMTP_FROM_EMAIL", ""),
		helloName:        envString("SMTP_HELLO_NAME", ""),
		connectTimeout:   envDuration("SMTP_CONNECT_TIMEOUT", 7*time.Second, time.Second, 20*time.Second),
		operationTimeout: envDuration("SMTP_OPERATION_TIMEOUT", 7*time.Second, time.Second, 20*time.Second),
		maxSMTPAttempts:  envInt("SMTP_MAX_ATTEMPTS", 2, 1, 4),
		smtpSlots:        make(chan struct{}, envInt("SMTP_MAX_CONCURRENCY", 4, 1, 20)),
	}

	if proxySMTP {
		app.proxyPool = newProxyPool()
		go app.proxyPool.run()
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", app.handleHome)
	mux.HandleFunc("/healthz", app.handleHealth)
	mux.HandleFunc("/api/verify", app.handleVerify)

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           securityHeaders(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      35 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("valid-mail listening on :%s (direct_smtp=%t proxy_smtp=%t)", port, directSMTP, proxySMTP)
	log.Fatal(server.ListenAndServe())
}

func (app *application) handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if r.Method == http.MethodGet {
		_, _ = w.Write(indexHTML)
	}
}

func (app *application) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	payload := map[string]any{
		"ok":                 true,
		"service":            "valid-mail",
		"smtp_enabled":       app.directSMTP || app.proxySMTP,
		"direct_smtp":        app.directSMTP,
		"proxy_smtp_enabled": app.proxySMTP,
		"time":               time.Now().UTC(),
	}
	if app.proxyPool != nil {
		payload["proxy_pool"] = app.proxyPool.stats()
	}
	writeJSON(w, http.StatusOK, payload)
}

func (app *application) handleVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST, OPTIONS")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var email string
	if r.Method == http.MethodGet {
		email = r.URL.Query().Get("email")
	} else {
		r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
		defer r.Body.Close()

		var req verifyRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"success": false,
				"error":   "invalid JSON body",
			})
			return
		}
		email = req.Email
	}

	email = strings.TrimSpace(email)
	if email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"success": false,
			"error":   "email is required",
		})
		return
	}
	if len(email) > 320 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"success": false,
			"error":   "email is too long",
		})
		return
	}

	started := time.Now()
	result, baseErr := app.baseVerifier.Verify(email)
	if result == nil {
		message := "verification failed"
		if baseErr != nil {
			message = baseErr.Error()
		}
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"success": false,
			"error":   message,
		})
		return
	}

	response := verifyResponse{
		Success:      true,
		Verification: result,
		SMTPEnabled:  app.directSMTP || app.proxySMTP,
		CheckedAt:    time.Now().UTC(),
	}
	if app.proxyPool != nil {
		response.HealthyProxies = app.proxyPool.stats().Healthy
	}
	if baseErr != nil {
		response.Warning = baseErr.Error()
	}

	if shouldAttemptSMTP(result) && (app.directSMTP || app.proxySMTP) {
		select {
		case app.smtpSlots <- struct{}{}:
			defer func() { <-app.smtpSlots }()
		case <-r.Context().Done():
			response.Warning = "request cancelled before SMTP verification"
			response.DurationMS = time.Since(started).Milliseconds()
			writeJSON(w, http.StatusOK, response)
			return
		}

		if app.proxySMTP && app.proxyPool != nil {
			smtpResult, attempted, err := app.verifyThroughProxy(email)
			response.SMTPAttempted = attempted
			response.SMTPViaProxy = attempted
			response.HealthyProxies = app.proxyPool.stats().Healthy
			if smtpResult != nil && err == nil {
				response.Verification = smtpResult
				response.Warning = ""
			} else if attempted {
				response.Warning = "SMTP through public SOCKS5 proxies was inconclusive; showing domain-level checks instead."
			} else if !app.directSMTP {
				response.Warning = "No healthy SMTP-capable public SOCKS5 proxy is available right now; showing domain-level checks instead."
				app.proxyPool.triggerRefresh()
			}
		}

		if !response.SMTPAttempted && app.directSMTP {
			response.SMTPAttempted = true
			smtpResult, err := app.smtpVerifier("").Verify(email)
			if smtpResult != nil && err == nil {
				response.Verification = smtpResult
				response.Warning = ""
			} else {
				response.Warning = "Direct SMTP verification failed; showing domain-level checks instead."
			}
		}
	}

	response.CheckedAt = time.Now().UTC()
	response.DurationMS = time.Since(started).Milliseconds()
	writeJSON(w, http.StatusOK, response)
}

func (app *application) verifyThroughProxy(email string) (*emailverifier.Result, bool, error) {
	exclude := make(map[string]struct{})
	var lastResult *emailverifier.Result
	var lastErr error

	for attempt := 0; attempt < app.maxSMTPAttempts; attempt++ {
		proxyURI := app.proxyPool.nextProxy(exclude)
		if proxyURI == "" {
			break
		}
		exclude[proxyURI] = struct{}{}

		result, err := app.smtpVerifier(proxyURI).Verify(email)
		if result != nil {
			lastResult = result
		}
		if err == nil {
			return result, true, nil
		}
		lastErr = err
		if result == nil || result.SMTP == nil {
			app.proxyPool.markBad(proxyURI)
		}
	}

	return lastResult, len(exclude) > 0, lastErr
}

func (app *application) smtpVerifier(proxyURI string) *emailverifier.Verifier {
	verifier := emailverifier.NewVerifier().
		EnableDomainSuggest().
		EnableSMTPCheck().
		ConnectTimeout(app.connectTimeout).
		OperationTimeout(app.operationTimeout)
	if proxyURI != "" {
		verifier.Proxy(proxyURI)
	}
	if app.fromEmail != "" {
		verifier.FromEmail(app.fromEmail)
	}
	if app.helloName != "" {
		verifier.HelloName(app.helloName)
	}
	return verifier
}

func shouldAttemptSMTP(result *emailverifier.Result) bool {
	return result != nil && result.Syntax.Valid && result.HasMxRecords && !result.Disposable
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'; connect-src 'self'")
		next.ServeHTTP(w, r)
	})
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envString(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envInt(key string, fallback, min, max int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < min || parsed > max {
		return fallback
	}
	return parsed
}

func envDuration(key string, fallback, min, max time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed < min || parsed > max {
		return fallback
	}
	return parsed
}
