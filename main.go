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
	verifier    *emailverifier.Verifier
	smtpEnabled bool
}

type verifyRequest struct {
	Email string `json:"email"`
}

type verifyResponse struct {
	Success     bool                  `json:"success"`
	Verification *emailverifier.Result `json:"verification,omitempty"`
	SMTPEnabled bool                  `json:"smtp_enabled"`
	Warning     string                `json:"warning,omitempty"`
	CheckedAt   time.Time             `json:"checked_at"`
	DurationMS  int64                 `json:"duration_ms"`
}

func main() {
	smtpEnabled := envBool("ENABLE_SMTP", false)
	verifier := emailverifier.NewVerifier().EnableDomainSuggest()

	if envBool("AUTO_UPDATE_DISPOSABLE", false) {
		verifier.EnableAutoUpdateDisposable()
	}
	if smtpEnabled {
		verifier.EnableSMTPCheck()
	}

	app := &application{
		verifier:    verifier,
		smtpEnabled: smtpEnabled,
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

	log.Printf("valid-mail listening on :%s (smtp=%t)", port, smtpEnabled)
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

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":           true,
		"service":      "valid-mail",
		"smtp_enabled": app.smtpEnabled,
		"time":         time.Now().UTC(),
	})
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
	result, err := app.verifier.Verify(email)
	duration := time.Since(started)

	if result == nil {
		message := "verification failed"
		if err != nil {
			message = err.Error()
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
		SMTPEnabled:  app.smtpEnabled,
		CheckedAt:    time.Now().UTC(),
		DurationMS:   duration.Milliseconds(),
	}
	if err != nil {
		// The verifier can return useful partial results alongside DNS/SMTP errors.
		response.Warning = err.Error()
	}

	writeJSON(w, http.StatusOK, response)
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
