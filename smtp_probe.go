package main

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"net/textproto"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/idna"
	"golang.org/x/net/proxy"
)

type smtpReply struct {
	Code         int    `json:"code,omitempty"`
	EnhancedCode string `json:"enhanced_code,omitempty"`
	Category     string `json:"category"`
	Message      string `json:"message,omitempty"`
}

type reachabilityAssessment struct {
	Status            string      `json:"status"`
	Reason            string      `json:"reason"`
	Confidence        int         `json:"confidence"`
	Evidence          string      `json:"evidence"`
	Provider          string      `json:"provider,omitempty"`
	MXHost            string      `json:"mx_host,omitempty"`
	SMTPConnected     bool        `json:"smtp_connected"`
	Target            *smtpReply  `json:"target,omitempty"`
	CatchAll          *bool       `json:"catch_all,omitempty"`
	CatchAllProbes    []smtpReply `json:"catch_all_probes,omitempty"`
	Attempts          int         `json:"attempts"`
	NegativeConsensus int         `json:"negative_consensus,omitempty"`
	Retryable         bool        `json:"retryable"`
}

type singleSMTPProbe struct {
	provider       string
	mxHost         string
	connected      bool
	target         smtpReply
	catchAll       *bool
	catchAllProbes []smtpReply
	stage          string
	err            error
}

var enhancedCodePattern = regexp.MustCompile(`\b[245]\.\d\.\d{1,3}\b`)

func (app *application) assessReachability(parent context.Context, email string) (*reachabilityAssessment, bool, bool) {
	budget := app.smtpTotalTimeout
	if budget <= 0 {
		budget = 25 * time.Second
	}
	ctx, cancel := context.WithTimeout(parent, budget)
	defer cancel()
	if app.smtpProxyURL != "" {
		probe := app.executeSMTPProbe(ctx, email, app.smtpProxyURL)
		assessment := assessmentFromProbe(probe)
		assessment.Attempts = 1
		return assessment, true, true
	}
	if app.proxySMTP && app.proxyPool != nil {
		app.proxyPool.ensureReady(ctx)
		assessment, attempted := app.assessThroughProxies(ctx, email)
		if attempted || !app.directSMTP {
			return assessment, attempted, attempted
		}
	}

	if app.directSMTP {
		probe := app.executeSMTPProbe(ctx, email, "")
		assessment := assessmentFromProbe(probe)
		assessment.Attempts = 1
		return assessment, true, false
	}

	return &reachabilityAssessment{
		Status:     "unknown",
		Reason:     "smtp_unavailable",
		Confidence: 0,
		Evidence:   "Mailbox-level SMTP verification was not available.",
		Retryable:  true,
	}, false, false
}

func (app *application) assessThroughProxies(ctx context.Context, email string) (*reachabilityAssessment, bool) {
	exclude := make(map[string]struct{})
	var best *reachabilityAssessment
	negativeConsensus := 0
	attempts := 0

	for attempts < app.maxSMTPAttempts && ctx.Err() == nil {
		proxyURI := app.proxyPool.nextProxy(exclude)
		if proxyURI == "" {
			break
		}
		exclude[proxyURI] = struct{}{}
		attempts++

		probe := app.executeSMTPProbe(ctx, email, proxyURI)
		candidate := assessmentFromProbe(probe)
		candidate.Attempts = attempts

		switch candidate.Reason {
		case "smtp_accepted_non_catch_all":
			return candidate, true
		case "smtp_recipient_missing":
			negativeConsensus++
			candidate.NegativeConsensus = negativeConsensus
			if negativeConsensus >= app.negativeConsensus {
				candidate.Status = "invalid"
				candidate.Confidence = 98
				candidate.Evidence = fmt.Sprintf("%d independent SMTP routes explicitly rejected the recipient as missing.", negativeConsensus)
				return candidate, true
			}
			candidate.Status = "unknown"
			candidate.Confidence = 55
			candidate.Reason = "negative_unconfirmed"
			candidate.Evidence = "One SMTP route said the recipient is missing; a second independent route is required before marking it invalid."
			best = chooseAssessment(best, candidate)
		case "smtp_catch_all":
			return candidate, true
		case "smtp_mailbox_full", "smtp_disabled":
			return candidate, true
		case "smtp_connection_failed", "smtp_tls_failed":
			app.proxyPool.markBad(proxyURI)
			best = chooseAssessment(best, candidate)
		case "smtp_blocked", "smtp_sender_rejected", "smtp_session_rejected":
			best = chooseAssessment(best, candidate)
		default:
			best = chooseAssessment(best, candidate)
		}
	}

	if best == nil {
		best = &reachabilityAssessment{
			Status:     "unknown",
			Reason:     "no_healthy_proxy",
			Confidence: 0,
			Evidence:   "No healthy SMTP-capable SOCKS5 proxy was available for a mailbox-level check.",
			Retryable:  true,
		}
		if app.proxyPool != nil {
			app.proxyPool.triggerRefresh()
		}
	}
	best.Attempts = attempts
	best.NegativeConsensus = negativeConsensus
	return best, attempts > 0
}

func chooseAssessment(current, candidate *reachabilityAssessment) *reachabilityAssessment {
	if candidate == nil {
		return current
	}
	if current == nil {
		return candidate
	}
	if assessmentRank(candidate) > assessmentRank(current) {
		return candidate
	}
	return current
}

func assessmentRank(a *reachabilityAssessment) int {
	if a == nil {
		return 0
	}
	switch a.Reason {
	case "smtp_accepted_non_catch_all":
		return 100
	case "smtp_mailbox_full", "smtp_disabled":
		return 90
	case "smtp_catch_all", "smtp_accepted_catch_all_unknown":
		return 80
	case "negative_unconfirmed":
		return 70
	case "smtp_greylisted", "smtp_rate_limited", "smtp_temporary_failure":
		return 50
	case "smtp_blocked", "smtp_sender_rejected", "smtp_session_rejected":
		return 40
	default:
		return 20
	}
}

func assessmentFromProbe(p singleSMTPProbe) *reachabilityAssessment {
	a := &reachabilityAssessment{
		Status:         "unknown",
		Reason:         "smtp_inconclusive",
		Confidence:     0,
		Evidence:       "SMTP did not provide enough recipient-specific evidence.",
		Provider:       p.provider,
		MXHost:         p.mxHost,
		SMTPConnected:  p.connected,
		CatchAll:       p.catchAll,
		CatchAllProbes: p.catchAllProbes,
		Retryable:      true,
	}
	if p.target.Code != 0 || p.target.Category != "" {
		target := p.target
		a.Target = &target
	}

	if p.err != nil {
		switch p.stage {
		case "connect":
			a.Reason = "smtp_connection_failed"
			a.Evidence = "The SMTP server could not be reached through this route."
		case "tls":
			a.Reason = "smtp_tls_failed"
			a.Evidence = "The SMTP server advertised TLS, but the TLS handshake failed."
		case "mail_from":
			a.Reason = "smtp_sender_rejected"
			a.Evidence = "The server rejected the verifier's MAIL FROM identity before checking the target recipient."
		case "hello":
			a.Reason = "smtp_session_rejected"
			a.Evidence = "The server rejected the SMTP greeting before the target recipient could be checked."
		default:
			a.Reason = "smtp_inconclusive"
		}
		return a
	}

	switch p.target.Category {
	case "accepted":
		if p.catchAll != nil && *p.catchAll {
			a.Status = "risky"
			a.Reason = "smtp_catch_all"
			a.Confidence = 60
			a.Evidence = "The target was accepted, but random non-existent addresses were also accepted, so SMTP cannot prove this mailbox exists."
			a.Retryable = false
			return a
		}
		if p.catchAll != nil && !*p.catchAll {
			a.Status = "valid"
			a.Reason = "smtp_accepted_non_catch_all"
			a.Confidence = 96
			a.Evidence = "The target recipient was accepted while random non-existent recipients were rejected in the same SMTP route."
			a.Retryable = false
			return a
		}
		a.Status = "risky"
		a.Reason = "smtp_accepted_catch_all_unknown"
		a.Confidence = 72
		a.Evidence = "The target was accepted, but catch-all behavior could not be determined conclusively."
		a.Retryable = true
		return a
	case "recipient_missing":
		a.Status = "invalid"
		a.Reason = "smtp_recipient_missing"
		a.Confidence = 90
		a.Evidence = "The SMTP server explicitly indicated that the target recipient does not exist."
		a.Retryable = false
		return a
	case "mailbox_full":
		a.Status = "risky"
		a.Reason = "smtp_mailbox_full"
		a.Confidence = 90
		a.Evidence = "The server response indicates that the mailbox likely exists but cannot currently accept mail because it is full."
		a.Retryable = true
		return a
	case "disabled":
		a.Status = "invalid"
		a.Reason = "smtp_disabled"
		a.Confidence = 95
		a.Evidence = "The server explicitly reported that the recipient account is disabled or inactive."
		a.Retryable = false
		return a
	case "blocked":
		a.Reason = "smtp_blocked"
		a.Confidence = 0
		a.Evidence = "The SMTP route was blocked by policy or IP reputation, so the response does not establish mailbox existence."
		return a
	case "greylisted":
		a.Reason = "smtp_greylisted"
		a.Confidence = 0
		a.Evidence = "The SMTP server temporarily deferred the check (greylisting). A later retry may produce a conclusive result."
		return a
	case "rate_limited":
		a.Reason = "smtp_rate_limited"
		a.Confidence = 0
		a.Evidence = "The SMTP server rate-limited the verification route. A later retry may produce a conclusive result."
		return a
	case "temporary":
		a.Reason = "smtp_temporary_failure"
		a.Confidence = 0
		a.Evidence = "The SMTP server returned a temporary 4xx response."
		return a
	case "rejected":
		a.Reason = "smtp_rejected_ambiguous"
		a.Confidence = 0
		a.Evidence = "The SMTP server rejected the transaction, but the response was not recipient-specific enough to mark the mailbox invalid."
		return a
	default:
		return a
	}
}

func (app *application) executeSMTPProbe(ctx context.Context, email, proxyURI string) singleSMTPProbe {
	if app.smtpProbe != nil {
		return app.smtpProbe(ctx, email, proxyURI)
	}
	return app.runSMTPProbe(ctx, email, proxyURI)
}

func (app *application) runSMTPProbe(parent context.Context, email, proxyURI string) singleSMTPProbe {
	result := singleSMTPProbe{stage: "connect"}
	at := strings.LastIndex(email, "@")
	if at <= 0 || at == len(email)-1 {
		result.err = errors.New("invalid email")
		return result
	}
	domain, domainErr := idna.Lookup.ToASCII(strings.TrimSpace(email[at+1:]))
	if domainErr != nil {
		result.err = domainErr
		return result
	}
	domain = strings.ToLower(domain)
	email = email[:at+1] + domain

	ctx, cancel := context.WithTimeout(parent, app.connectTimeout)
	lookup := app.smtpLookupMX
	if lookup == nil {
		lookup = net.DefaultResolver.LookupMX
	}
	mx, err := lookup(ctx, domain)
	cancel()
	if err != nil || len(mx) == 0 {
		if err == nil {
			err = errors.New("no MX records")
		}
		result.err = err
		return result
	}
	sort.Slice(mx, func(i, j int) bool { return mx[i].Pref < mx[j].Pref })
	result.provider = providerFromMX(mx)

	var lastErr error
	for index, record := range mx {
		if parent.Err() != nil {
			lastErr = parent.Err()
			break
		}
		if index >= 3 {
			break
		}
		host := strings.TrimSuffix(record.Host, ".")
		if host == "" {
			continue
		}
		probe, err := app.probeMX(parent, email, domain, host, proxyURI)
		if err == nil || probe.connected {
			probe.provider = result.provider
			return probe
		}
		lastErr = err
	}
	result.err = lastErr
	return result
}

func (app *application) probeMX(parent context.Context, email, domain, host, proxyURI string) (singleSMTPProbe, error) {
	result := singleSMTPProbe{mxHost: host, stage: "connect"}
	ctx, cancel := context.WithTimeout(parent, app.connectTimeout)
	defer cancel()

	dial := app.smtpDial
	if dial == nil {
		dial = dialSMTPRoute
	}
	conn, err := dial(ctx, proxyURI, net.JoinHostPort(host, "25"), app.connectTimeout)
	if err != nil {
		result.err = err
		return result, err
	}
	deadline := time.Now().Add(app.operationTimeout)
	if totalDeadline, ok := parent.Deadline(); ok && totalDeadline.Before(deadline) {
		deadline = totalDeadline
	}
	_ = conn.SetDeadline(deadline)
	stop := context.AfterFunc(parent, func() { _ = conn.Close() })
	defer stop()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		_ = conn.Close()
		result.err = err
		return result, err
	}
	defer client.Close()
	result.connected = true

	helloName := app.helloName
	if helloName == "" {
		helloName = "localhost"
	}
	result.stage = "hello"
	if err := client.Hello(helloName); err != nil {
		result.err = err
		result.target = replyFromError(err, email)
		return result, err
	}

	if ok, _ := client.Extension("STARTTLS"); ok {
		result.stage = "tls"
		if err := client.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
			result.err = err
			result.target = replyFromError(err, email)
			return result, err
		}
	}

	fromEmail := app.fromEmail
	// An empty address uses the standard null envelope sender, never an invented identity.
	result.stage = "mail_from"
	if err := client.Mail(fromEmail); err != nil {
		result.err = err
		result.target = replyFromError(err, email)
		return result, err
	}

	result.stage = "rcpt"
	result.target = rcptReply(client, email)
	if result.target.Category != "accepted" {
		return result, nil
	}

	accepted := 0
	rejected := 0
	probes := make([]smtpReply, 0, app.catchAllProbes)
	for i := 0; i < app.catchAllProbes; i++ {
		if err := client.Reset(); err != nil {
			break
		}
		if err := client.Mail(fromEmail); err != nil {
			break
		}
		fake := randomRecipient(domain)
		reply := rcptReply(client, fake)
		probes = append(probes, reply)
		switch reply.Category {
		case "accepted":
			accepted++
		case "recipient_missing":
			rejected++
		}
	}
	result.catchAllProbes = probes
	if len(probes) == app.catchAllProbes && accepted == app.catchAllProbes {
		v := true
		result.catchAll = &v
	} else if len(probes) == app.catchAllProbes && rejected == app.catchAllProbes {
		v := false
		result.catchAll = &v
	}
	return result, nil
}

func dialSMTPRoute(ctx context.Context, proxyURI, target string, timeout time.Duration) (net.Conn, error) {
	host, port, err := net.SplitHostPort(target)
	if err != nil || port != "25" {
		return nil, errors.New("invalid SMTP destination")
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	targets := publicSMTPDestinations(ips, port)
	if len(targets) == 0 {
		return nil, errors.New("SMTP destination is not public")
	}
	var proxyDialer proxy.Dialer
	if proxyURI != "" {
		u, err := url.Parse(proxyURI)
		if err != nil {
			return nil, err
		}
		proxyDialer, err = proxy.FromURL(u, &net.Dialer{Timeout: timeout})
		if err != nil {
			return nil, err
		}
	}
	budget, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var lastErr error
	for index, target := range targets {
		if budget.Err() != nil {
			return nil, budget.Err()
		}
		deadline, _ := budget.Deadline()
		attemptBudget := time.Until(deadline) / time.Duration(len(targets)-index)
		if attemptBudget > 3*time.Second {
			attemptBudget = 3 * time.Second
		}
		attempt, stop := context.WithTimeout(budget, attemptBudget)
		var conn net.Conn
		if proxyDialer == nil {
			conn, lastErr = (&net.Dialer{Timeout: attemptBudget}).DialContext(attempt, "tcp", target)
		} else {
			conn, lastErr = dialProxyContext(attempt, proxyDialer, target)
		}
		stop()
		if lastErr == nil {
			return conn, nil
		}
	}
	return nil, lastErr
}

// Pin vetted destinations against DNS rebinding and prefer IPv4 on cloud hosts
// without IPv6 egress. Keep alternate public IPs as bounded fallback routes.
func publicSMTPDestinations(ips []net.IPAddr, port string) []string {
	targets := []string{}
	seen := map[string]bool{}
	for _, ipv4 := range []bool{true, false} {
		for _, item := range ips {
			if !isPublicIP(item.IP) || (item.IP.To4() != nil) != ipv4 {
				continue
			}
			address := net.JoinHostPort(item.IP.String(), port)
			if !seen[address] {
				seen[address] = true
				targets = append(targets, address)
				if len(targets) >= 6 {
					return targets
				}
			}
		}
	}
	return targets
}

func rcptReply(client *smtp.Client, address string) smtpReply {
	if err := client.Rcpt(address); err != nil {
		return replyFromError(err, address)
	}
	return smtpReply{Code: 250, Category: "accepted", Message: "Recipient accepted"}
}

func replyFromError(err error, email string) smtpReply {
	if err == nil {
		return smtpReply{Category: "unknown"}
	}
	code := 0
	message := err.Error()
	var protoErr *textproto.Error
	if errors.As(err, &protoErr) {
		code = protoErr.Code
		message = protoErr.Msg
	} else if len(message) >= 3 {
		if parsed, parseErr := strconv.Atoi(message[:3]); parseErr == nil {
			code = parsed
		}
	}
	enhanced := enhancedCodePattern.FindString(message)
	category := classifySMTPReply(code, enhanced, message)
	if code == 0 {
		message = "SMTP transport did not complete."
	}
	message = sanitizeSMTPMessage(message, email)
	return smtpReply{Code: code, EnhancedCode: enhanced, Category: category, Message: message}
}

func classifySMTPReply(code int, enhanced, message string) string {
	m := strings.ToLower(message)
	if code == 250 || code == 251 {
		return "accepted"
	}

	missingHints := []string{
		"does not exist", "doesn't exist", "no such user", "user unknown", "unknown user",
		"invalid recipient", "unknown recipient", "recipient unknown", "recipient not found",
		"mailbox not found", "no mailbox", "no such mailbox", "invalid mailbox",
		"unknown local part",
		"no such recipient", "no such person", "email doesn't exist", "recipient is not exist",
	}
	fullHints := []string{"mailbox full", "over quota", "quota exceeded", "out of storage", "storage space", "too many messages"}
	disabledHints := []string{"account disabled", "account is disabled", "mailbox is inactive", "account inactive", "suspended", "discontinued"}
	blockedHints := []string{"blacklist", "black list", "block list", "spamhaus", "dnsbl", "poor reputation", "banned", "access denied", "administratively denied", "policy", "spam", "relay denied", "reverse dns", "reverse hostname", "not authorized"}
	greyHints := []string{"greylist", "grey list", "try again later", "temporarily deferred", "temporary deferral"}
	rateHints := []string{"rate limit", "rate that", "too many", "throttl", "too fast"}

	// Policy and temporary responses never establish a missing recipient.
	if strings.HasPrefix(enhanced, "5.7.") || strings.HasPrefix(enhanced, "4.7.") || hasAnySMTPHint(m, blockedHints) {
		return "blocked"
	}
	if enhanced == "5.2.2" || enhanced == "4.2.2" || code == 552 || (code >= 400 && hasAnySMTPHint(m, fullHints)) {
		return "mailbox_full"
	}
	if enhanced == "5.2.1" {
		return "disabled"
	}
	if hasAnySMTPHint(m, greyHints) {
		return "greylisted"
	}
	if code == 421 || hasAnySMTPHint(m, rateHints) {
		return "rate_limited"
	}
	if code >= 400 && code < 500 && enhanced != "4.2.2" {
		return "temporary"
	}
	if code >= 500 && code < 600 && (enhanced == "5.1.1" || hasAnySMTPHint(m, missingHints)) {
		return "recipient_missing"
	}
	if enhanced == "5.2.2" || enhanced == "4.2.2" || code == 552 || hasAnySMTPHint(m, fullHints) {
		return "mailbox_full"
	}
	if enhanced == "5.2.1" || hasAnySMTPHint(m, disabledHints) {
		return "disabled"
	}
	if hasAnySMTPHint(m, greyHints) {
		return "greylisted"
	}
	if code == 421 || hasAnySMTPHint(m, rateHints) {
		return "rate_limited"
	}
	if strings.HasPrefix(enhanced, "5.7.") || strings.HasPrefix(enhanced, "4.7.") || hasAnySMTPHint(m, blockedHints) {
		return "blocked"
	}
	if code >= 400 && code < 500 {
		return "temporary"
	}
	if code >= 500 && code < 600 {
		return "rejected"
	}
	return "unknown"
}

func hasAnySMTPHint(text string, hints []string) bool {
	for _, hint := range hints {
		if strings.Contains(text, hint) {
			return true
		}
	}
	return false
}

func sanitizeSMTPMessage(message, email string) string {
	message = strings.TrimSpace(message)
	if email != "" {
		message = strings.ReplaceAll(message, email, "<email>")
	}
	message = strings.Join(strings.Fields(message), " ")
	if len(message) > 280 {
		message = message[:280] + "…"
	}
	return message
}

func randomRecipient(domain string) string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("validmail-nonexistent-%d@%s", time.Now().UnixNano(), domain)
	}
	return "validmail-nonexistent-" + hex.EncodeToString(buf) + "@" + domain
}

func providerFromMX(mx []*net.MX) string {
	for _, record := range mx {
		h := strings.ToLower(record.Host)
		switch {
		case strings.Contains(h, "google.com"), strings.Contains(h, "googlemail.com"):
			return "google"
		case strings.Contains(h, "protection.outlook.com"), strings.Contains(h, "outlook.com"):
			return "microsoft"
		case strings.Contains(h, "yahoodns.net"), strings.Contains(h, "yahoo.com"):
			return "yahoo"
		case strings.Contains(h, "icloud.com"):
			return "icloud"
		case strings.Contains(h, "protonmail.ch"), strings.Contains(h, "proton.me"):
			return "proton"
		case strings.Contains(h, "zoho.com"), strings.Contains(h, "zohomail.com"):
			return "zoho"
		case strings.Contains(h, "messagingengine.com"):
			return "fastmail"
		}
	}
	return "other"
}
