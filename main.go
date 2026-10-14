package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	emailverifier "github.com/AfterShip/email-verifier"
)

type addressVerifier interface {
	Verify(string) (*emailverifier.Result, error)
}

type application struct {
	baseVerifier      addressVerifier
	directSMTP        bool
	proxySMTP         bool
	smtpProxyURL      string
	proxyPool         *proxyPool
	fromEmail         string
	helloName         string
	connectTimeout    time.Duration
	operationTimeout  time.Duration
	maxSMTPAttempts   int
	catchAllProbes    int
	negativeConsensus int
	smtpSlots         chan struct{}
	verifySlots       chan struct{}
	verifyLimiter     *requestLimiter
	smtpTotalTimeout  time.Duration
	smtpProbe         func(context.Context, string, string) singleSMTPProbe
	smtpDial          func(context.Context, string, string, time.Duration) (net.Conn, error)
	smtpLookupMX      func(context.Context, string) ([]*net.MX, error)
}

type verifyRequest struct {
	Email string `json:"email"`
	SMTP  *bool  `json:"smtp,omitempty"`
}

type verifyResponse struct {
	Success        bool                    `json:"success"`
	Verification   *emailverifier.Result   `json:"verification,omitempty"`
	Reachability   *reachabilityAssessment `json:"reachability,omitempty"`
	SMTPEnabled    bool                    `json:"smtp_enabled"`
	SMTPAttempted  bool                    `json:"smtp_attempted"`
	SMTPViaProxy   bool                    `json:"smtp_via_proxy"`
	HealthyProxies int                     `json:"healthy_proxies,omitempty"`
	Warning        string                  `json:"warning,omitempty"`
	CheckedAt      time.Time               `json:"checked_at"`
	DurationMS     int64                   `json:"duration_ms"`
	DNSUnavailable bool                    `json:"-"`
}

func main() {
	// Default to the free SOCKS5 pool, matching the Render Free deployment.
	// Direct outbound port 25 is not available on Render Free.
	directSMTP := envBool("ENABLE_SMTP", false)
	proxySMTP := envBool("ENABLE_PROXY_SMTP", true)
	smtpProxyURL := strings.TrimSpace(os.Getenv("SMTP_PROXY_URL"))
	baseVerifier := newDomainVerifier()

	if envBool("AUTO_UPDATE_DISPOSABLE", false) {
		baseVerifier.metadata.EnableAutoUpdateDisposable()
	}

	app := &application{
		baseVerifier:      baseVerifier,
		directSMTP:        directSMTP,
		proxySMTP:         proxySMTP,
		smtpProxyURL:      smtpProxyURL,
		fromEmail:         envString("SMTP_FROM_EMAIL", ""),
		helloName:         envString("SMTP_HELLO_NAME", ""),
		connectTimeout:    envDuration("SMTP_CONNECT_TIMEOUT", 7*time.Second, time.Second, 20*time.Second),
		smtpTotalTimeout:  envDuration("SMTP_TOTAL_TIMEOUT", 25*time.Second, time.Second, 30*time.Second),
		operationTimeout:  envDuration("SMTP_OPERATION_TIMEOUT", 8*time.Second, time.Second, 25*time.Second),
		maxSMTPAttempts:   envInt("SMTP_MAX_ATTEMPTS", 3, 1, 5),
		catchAllProbes:    envInt("SMTP_CATCHALL_PROBES", 2, 1, 3),
		negativeConsensus: envInt("SMTP_NEGATIVE_CONSENSUS", 2, 1, 3),
		smtpSlots:         make(chan struct{}, envInt("SMTP_MAX_CONCURRENCY", 4, 1, 20)),
		verifySlots:       make(chan struct{}, envInt("VERIFY_MAX_CONCURRENCY", 8, 1, 64)),
		verifyLimiter:     newRequestLimiter(30, time.Minute, 10_000),
	}

	if proxySMTP {
		app.proxyPool = newProxyPool()
		go app.proxyPool.run()
	}

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           app.routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      45 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			log.Printf("shutdown: %v", err)
		}
	}()
	log.Printf("WeMe listening on :%s (direct_smtp=%t proxy_smtp=%t)", port, directSMTP, proxySMTP)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", serveFrontend)
	mux.HandleFunc("/healthz", app.handleHealth)
	mux.HandleFunc("/api/verify", app.handleVerify)
	mux.HandleFunc("/api/pdf/inspect", app.handlePDF)
	mux.HandleFunc("/api/pdf/preview", app.handlePDF)
	mux.HandleFunc("/api/pdf/edit", app.handlePDF)
	mux.HandleFunc("/api/find-email", app.handleFindEmail)
	mux.HandleFunc("/api/find-work-email", app.handleFindWorkEmail)
	mux.HandleFunc("/api/find-linkedin-email", app.handleFindLinkedInEmail)
	mux.HandleFunc("/api/find-by-name", app.handleFindByName)
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]any{"success": false, "error": "endpoint not found"})
	})
	return securityHeaders(mux)
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
		"smtp_enabled":       app.directSMTP || app.proxySMTP || app.smtpProxyURL != "",
		"direct_smtp":        app.directSMTP,
		"proxy_smtp_enabled": app.proxySMTP,
		"catch_all_probes":   app.catchAllProbes,
		"negative_consensus": app.negativeConsensus,
		"max_smtp_attempts":  app.maxSMTPAttempts,
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
	w.Header().Set("Cache-Control", "no-store")
	if app.verifyLimiter != nil && !app.verifyLimiter.allow(requestIP(r)) {
		w.Header().Set("Retry-After", "60")
		writeJSON(w, http.StatusTooManyRequests, map[string]any{"success": false, "error": "Too many checks. Please wait a minute and try again."})
		return
	}

	var email string
	smtpRequested := true
	if r.Method == http.MethodGet {
		email = r.URL.Query().Get("email")
		if raw := r.URL.Query().Get("smtp"); raw != "" {
			requested, err := strconv.ParseBool(raw)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "smtp must be true or false"})
				return
			}
			smtpRequested = requested
		}
	} else {
		r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
		defer r.Body.Close()

		var req verifyRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "invalid JSON body"})
			return
		}
		if err := requireJSONEnd(decoder); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "provide one JSON object"})
			return
		}
		email = req.Email
		if req.SMTP != nil {
			smtpRequested = *req.SMTP
		}
	}

	email = strings.TrimSpace(email)
	if email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "email is required"})
		return
	}
	if len(email) > 320 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "email is too long"})
		return
	}

	started := time.Now()
	if app.verifySlots != nil {
		select {
		case app.verifySlots <- struct{}{}:
			defer func() { <-app.verifySlots }()
		default:
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"success": false, "error": "The service is busy. Please try again shortly."})
			return
		}
	}
	result, baseErr := app.baseVerifier.Verify(email)
	if result == nil {
		message := "verification failed"
		if baseErr != nil {
			message = baseErr.Error()
		}
		writeJSON(w, http.StatusBadGateway, map[string]any{"success": false, "error": message})
		return
	}

	response := verifyResponse{
		Success:      true,
		Verification: result,
		SMTPEnabled:  app.directSMTP || app.proxySMTP || app.smtpProxyURL != "",
		CheckedAt:    time.Now().UTC(),
	}
	if app.proxyPool != nil {
		response.HealthyProxies = app.proxyPool.stats().Healthy
	}
	if baseErr != nil {
		if dnsUnavailable(result, baseErr) {
			response.DNSUnavailable = true
			response.Warning = "Domain verification is temporarily unavailable because the DNS lookup failed. Try again later; this is not evidence that the address is invalid."
		} else {
			response.Warning = "The domain could not be verified. Review the address and try again."
		}
	}

	if !result.Syntax.Valid {
		response.Reachability = &reachabilityAssessment{Status: "invalid", Reason: "bad_syntax", Confidence: 100, Evidence: "The email address syntax is invalid."}
	} else if response.DNSUnavailable {
		response.Reachability = &reachabilityAssessment{Status: "unknown", Reason: "dns_unavailable", Confidence: 0, Evidence: "DNS is unavailable, so mail routing and mailbox reachability could not be checked.", Retryable: true}
	} else if !result.HasMxRecords && !result.Disposable {
		response.Reachability = &reachabilityAssessment{Status: "invalid", Reason: "no_mx", Confidence: 100, Evidence: "The domain does not publish MX records for receiving mail."}
	} else if result.Disposable {
		response.Reachability = &reachabilityAssessment{Status: "risky", Reason: "disposable_domain", Confidence: 95, Evidence: "The address belongs to a known disposable-email domain."}
	} else if !smtpRequested {
		response.Reachability = &reachabilityAssessment{Status: "unknown", Reason: "smtp_not_requested", Confidence: 0, Evidence: "Address and domain checks are complete. Mailbox-level SMTP verification was not requested."}
	} else if shouldAttemptSMTP(result) && (app.directSMTP || app.proxySMTP || app.smtpProxyURL != "") {
		select {
		case app.smtpSlots <- struct{}{}:
			defer func() { <-app.smtpSlots }()
		default:
			response.Reachability = &reachabilityAssessment{Status: "unknown", Reason: "smtp_busy", Confidence: 0, Evidence: "All SMTP verification slots are busy. Please retry shortly.", Retryable: true}
			response.Warning = "SMTP verification is busy. Please retry shortly."
			response.DurationMS = time.Since(started).Milliseconds()
			writeJSON(w, http.StatusOK, response)
			return
		}

		assessment, attempted, viaProxy := app.assessReachability(r.Context(), email)
		response.Reachability = assessment
		response.SMTPAttempted = attempted
		response.SMTPViaProxy = viaProxy
		if app.proxyPool != nil {
			response.HealthyProxies = app.proxyPool.stats().Healthy
		}
		applyAssessment(result, assessment)
		response.Warning = warningForAssessment(assessment)
	} else {
		response.Reachability = &reachabilityAssessment{Status: "unknown", Reason: "smtp_unavailable", Confidence: 0, Evidence: "Domain checks pass, but mailbox-level SMTP verification is not enabled.", Retryable: true}
	}

	response.CheckedAt = time.Now().UTC()
	response.DurationMS = time.Since(started).Milliseconds()
	writeJSON(w, http.StatusOK, response)
}

func applyAssessment(result *emailverifier.Result, assessment *reachabilityAssessment) {
	if result == nil || assessment == nil {
		return
	}
	if assessment.SMTPConnected || assessment.Target != nil {
		result.SMTP = &emailverifier.SMTP{
			HostExists:  assessment.SMTPConnected,
			FullInbox:   assessment.Reason == "smtp_mailbox_full",
			CatchAll:    assessment.CatchAll != nil && *assessment.CatchAll,
			Deliverable: assessment.Status == "valid" || assessment.Reason == "smtp_accepted_catch_all_unknown" || assessment.Reason == "smtp_catch_all",
			Disabled:    assessment.Reason == "smtp_disabled",
		}
	}
	switch assessment.Status {
	case "valid":
		result.Reachable = "yes"
	case "invalid":
		result.Reachable = "no"
	default:
		result.Reachable = "unknown"
	}
}

func warningForAssessment(a *reachabilityAssessment) string {
	if a == nil {
		return ""
	}
	switch a.Reason {
	case "smtp_connection_failed", "smtp_tls_failed", "smtp_inconclusive":
		return "The SMTP route could not complete verification. Mailbox existence remains unknown; this does not mean the address is invalid."
	case "smtp_blocked", "smtp_sender_rejected", "smtp_session_rejected":
		return "The mail server rejected the verification route or sender identity, so mailbox existence remains unknown."
	case "smtp_greylisted", "smtp_rate_limited", "smtp_temporary_failure":
		return "The mail server temporarily deferred verification. Retry later for a more conclusive result."
	case "negative_unconfirmed":
		return "One route rejected the recipient, but a second independent SMTP route did not confirm the negative result."
	case "no_healthy_proxy":
		return "No healthy SMTP-capable public SOCKS5 proxy is available right now; showing domain-level checks instead."
	case "smtp_catch_all":
		return "This domain accepts random non-existent recipients, so SMTP cannot prove this specific mailbox exists."
	case "smtp_accepted_catch_all_unknown":
		return "The target was accepted, but catch-all behavior could not be established conclusively."
	}
	return ""
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
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'wasm-unsafe-eval'; worker-src 'self'; connect-src 'self'; img-src 'self' data: https:; font-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
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
