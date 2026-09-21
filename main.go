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
	baseVerifier      *emailverifier.Verifier
	directSMTP        bool
	proxySMTP         bool
	proxyPool         *proxyPool
	fromEmail         string
	helloName         string
	connectTimeout    time.Duration
	operationTimeout  time.Duration
	maxSMTPAttempts   int
	catchAllProbes    int
	negativeConsensus int
	smtpSlots         chan struct{}
}

type verifyRequest struct {
	Email string `json:"email"`
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
}

func main() {
	directSMTP := envBool("ENABLE_SMTP", false)
	proxySMTP := envBool("ENABLE_PROXY_SMTP", true)
	baseVerifier := emailverifier.NewVerifier().EnableDomainSuggest()

	if envBool("AUTO_UPDATE_DISPOSABLE", false) {
		baseVerifier.EnableAutoUpdateDisposable()
	}

	app := &application{
		baseVerifier:      baseVerifier,
		directSMTP:        directSMTP,
		proxySMTP:         proxySMTP,
		fromEmail:         envString("SMTP_FROM_EMAIL", ""),
		helloName:         envString("SMTP_HELLO_NAME", ""),
		connectTimeout:    envDuration("SMTP_CONNECT_TIMEOUT", 7*time.Second, time.Second, 20*time.Second),
		operationTimeout:  envDuration("SMTP_OPERATION_TIMEOUT", 8*time.Second, time.Second, 25*time.Second),
		maxSMTPAttempts:   envInt("SMTP_MAX_ATTEMPTS", 3, 1, 5),
		catchAllProbes:    envInt("SMTP_CATCHALL_PROBES", 2, 1, 3),
		negativeConsensus: envInt("SMTP_NEGATIVE_CONSENSUS", 2, 1, 3),
		smtpSlots:         make(chan struct{}, envInt("SMTP_MAX_CONCURRENCY", 4, 1, 20)),
	}

	if proxySMTP {
		app.proxyPool = newProxyPool()
		go app.proxyPool.run()
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", app.handleHome)
	mux.HandleFunc("/finder", app.handleFinderPage)
	mux.HandleFunc("/healthz", app.handleHealth)
	mux.HandleFunc("/api/verify", app.handleVerify)
	mux.HandleFunc("/api/find-email", app.handleFindEmail)
	mux.HandleFunc("/api/find-work-email", app.handleFindWorkEmail)

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           securityHeaders(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      45 * time.Second,
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
			writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "invalid JSON body"})
			return
		}
		email = req.Email
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
		SMTPEnabled:  app.directSMTP || app.proxySMTP,
		CheckedAt:    time.Now().UTC(),
	}
	if app.proxyPool != nil {
		response.HealthyProxies = app.proxyPool.stats().Healthy
	}
	if baseErr != nil {
		response.Warning = baseErr.Error()
	}

	if !result.Syntax.Valid {
		response.Reachability = &reachabilityAssessment{Status: "invalid", Reason: "bad_syntax", Confidence: 100, Evidence: "The email address syntax is invalid."}
	} else if !result.HasMxRecords {
		response.Reachability = &reachabilityAssessment{Status: "invalid", Reason: "no_mx", Confidence: 100, Evidence: "The domain does not publish MX records for receiving mail."}
	} else if result.Disposable {
		response.Reachability = &reachabilityAssessment{Status: "risky", Reason: "disposable_domain", Confidence: 95, Evidence: "The address belongs to a known disposable-email domain."}
	} else if shouldAttemptSMTP(result) && (app.directSMTP || app.proxySMTP) {
		select {
		case app.smtpSlots <- struct{}{}:
			defer func() { <-app.smtpSlots }()
		case <-r.Context().Done():
			response.Reachability = &reachabilityAssessment{Status: "unknown", Reason: "request_cancelled", Confidence: 0, Evidence: "The request ended before SMTP verification could run.", Retryable: true}
			response.Warning = "request cancelled before SMTP verification"
			response.DurationMS = time.Since(started).Milliseconds()
			writeJSON(w, http.StatusOK, response)
			return
		}

		assessment, attempted, viaProxy := app.assessReachability(email)
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
