package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// Speak real SMTP on a local socket; the production net/smtp client handles commands.
func smtpFixture(t *testing.T, reply string, catchAll, tls bool) (*application, *[]string) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	commands := []string{}
	var mu sync.Mutex
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		fmt.Fprint(conn, "220 fixture ESMTP\r\n")
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			line := scanner.Text()
			mu.Lock()
			commands = append(commands, line)
			mu.Unlock()
			switch {
			case strings.HasPrefix(line, "EHLO"):
				if tls {
					fmt.Fprint(conn, "250-fixture\r\n250 STARTTLS\r\n")
				} else {
					fmt.Fprint(conn, "250 fixture\r\n")
				}
			case line == "STARTTLS":
				fmt.Fprint(conn, "220 Ready\r\n")
				return // No valid TLS certificate: client must refuse.
			case strings.HasPrefix(line, "MAIL"), line == "RSET":
				fmt.Fprint(conn, "250 OK\r\n")
			case strings.HasPrefix(line, "RCPT"):
				if strings.Contains(line, "validmail-nonexistent-") && !catchAll {
					fmt.Fprint(conn, "550 5.1.1 No such user\r\n")
				} else {
					fmt.Fprint(conn, reply+"\r\n")
				}
			case line == "QUIT":
				fmt.Fprint(conn, "221 Bye\r\n")
				return
			default:
				fmt.Fprint(conn, "500 Unexpected command\r\n")
			}
		}
	}()
	t.Cleanup(func() { listener.Close(); <-done })
	app := &application{directSMTP: true, connectTimeout: time.Second, operationTimeout: time.Second, smtpTotalTimeout: 2 * time.Second, catchAllProbes: 2}
	app.smtpLookupMX = func(context.Context, string) ([]*net.MX, error) {
		return []*net.MX{{Host: "fixture.test.", Pref: 10}}, nil
	}
	app.smtpDial = func(ctx context.Context, _, _ string, _ time.Duration) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "tcp", listener.Addr().String())
	}
	// Wait for the handler before examining commands to avoid concurrent reads.
	app.smtpProbe = func(ctx context.Context, email, route string) singleSMTPProbe {
		probe := app.runSMTPProbe(ctx, email, route)
		<-done
		return probe
	}
	return app, &commands
}

func TestSMTPProtocolEvidence(t *testing.T) {
	for _, tt := range []struct {
		name, reply, status, reason string
		confidence                  int
		catchAll, tls               bool
	}{
		{"accepted", "250 2.1.5 Recipient accepted", "valid", "smtp_accepted_non_catch_all", 96, false, false},
		{"catch-all", "250 2.1.5 Recipient accepted", "risky", "smtp_catch_all", 60, true, false},
		{"missing", "550 5.1.1 No such user", "invalid", "smtp_recipient_missing", 90, false, false},
		{"policy", "550 5.7.1 Address rejected by policy: user unknown", "unknown", "smtp_blocked", 0, false, false},
		{"temporary", "450 4.1.1 User unknown temporarily", "unknown", "smtp_temporary_failure", 0, false, false},
		{"tls-refused", "250 OK", "unknown", "smtp_tls_failed", 0, false, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app, commands := smtpFixture(t, tt.reply, tt.catchAll, tt.tls)
			assessment, attempted, via := app.assessReachability(context.Background(), "target@fixture.test")
			if !attempted || via || assessment.Status != tt.status || assessment.Reason != tt.reason || assessment.Confidence != tt.confidence {
				t.Fatalf("unexpected assessment: %+v", assessment)
			}
			for _, command := range *commands {
				if strings.HasPrefix(command, "DATA") || strings.Contains(command, "example.org") {
					t.Fatalf("unsafe command: %s", command)
				}
			}
			if !tt.tls && !strings.Contains(strings.Join(*commands, "\n"), "MAIL FROM:<>") {
				t.Fatal("expected null envelope sender")
			}
		})
	}
}

func TestSMTPDeadlineAndCancellation(t *testing.T) {
	for _, canceled := range []bool{false, true} {
		app := &application{directSMTP: true, connectTimeout: time.Second, operationTimeout: time.Second, smtpTotalTimeout: 40 * time.Millisecond}
		app.smtpLookupMX = func(ctx context.Context, _ string) ([]*net.MX, error) { <-ctx.Done(); return nil, ctx.Err() }
		ctx, cancel := context.WithCancel(context.Background())
		if canceled {
			cancel()
		}
		defer cancel()
		start := time.Now()
		a, _, _ := app.assessReachability(ctx, "target@fixture.test")
		if time.Since(start) > 200*time.Millisecond || a.Status != "unknown" || a.Confidence != 0 {
			t.Fatalf("deadline not honored: %+v", a)
		}
	}
}

func TestProxyNegativeRequiresConsensus(t *testing.T) {
	app := &application{maxSMTPAttempts: 3, negativeConsensus: 2, proxyPool: &proxyPool{healthy: []string{"route-a", "route-b"}}}
	app.smtpProbe = func(context.Context, string, string) singleSMTPProbe {
		return singleSMTPProbe{connected: true, target: smtpReply{Code: 550, Category: "recipient_missing"}}
	}
	a, attempted := app.assessThroughProxies(context.Background(), "target@fixture.test")
	if !attempted || a.Status != "invalid" || a.NegativeConsensus != 2 || a.Attempts != 2 {
		t.Fatalf("bad consensus: %+v", a)
	}
	app.proxyPool.healthy = []string{"route-a"}
	a, _ = app.assessThroughProxies(context.Background(), "target@fixture.test")
	if a.Status != "unknown" || a.Reason != "negative_unconfirmed" {
		t.Fatalf("single rejection treated conclusive: %+v", a)
	}
	app.smtpProbe = func(context.Context, string, string) singleSMTPProbe {
		return singleSMTPProbe{stage: "connect", err: errors.New("blocked")}
	}
	a, _ = app.assessThroughProxies(context.Background(), "target@fixture.test")
	if a.Status != "unknown" || a.Confidence != 0 {
		t.Fatalf("transport failure treated invalid: %+v", a)
	}
}

func TestProxyColdStartRunsInsideRequest(t *testing.T) {
	// An unavailable source is still exercised synchronously, without a run goroutine.
	requests := 0
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests++; w.WriteHeader(http.StatusServiceUnavailable) }))
	defer source.Close()
	pool := newProxyPool()
	pool.sourceURL = source.URL
	pool.ensureReady(context.Background())
	if requests != 1 || pool.stats().LastRefresh.IsZero() {
		t.Fatal("cold start did not refresh during request")
	}
	pool.ensureReady(context.Background())
	if requests != 1 {
		t.Fatal("failed source cooldown not honored")
	}
	if pool.stats().Healthy != 0 {
		t.Fatal("failed refresh manufactured healthy proxies")
	}
}

func TestConfiguredPrivateSMTPRoute(t *testing.T) {
	app := &application{smtpProxyURL: "socks5://private.fixture:1080", smtpTotalTimeout: time.Second}
	app.smtpProbe = func(_ context.Context, _ string, route string) singleSMTPProbe {
		if route != app.smtpProxyURL {
			t.Fatal("configured route ignored")
		}
		noCatchAll := false
		return singleSMTPProbe{connected: true, catchAll: &noCatchAll, target: smtpReply{Code: 250, Category: "accepted"}}
	}
	a, attempted, viaProxy := app.assessReachability(context.Background(), "target@fixture.test")
	if !attempted || !viaProxy || a.Status != "valid" || a.Confidence != 96 {
		t.Fatalf("unexpected private route result: %+v", a)
	}
}

func TestSMTPRejectsPrivateDestinations(t *testing.T) {
	for _, target := range []string{"127.0.0.1:25", "10.1.2.3:25", "169.254.169.254:25", "[::1]:25", "127.0.0.1:80"} {
		conn, err := dialSMTPRoute(context.Background(), "", target, time.Second)
		if err == nil || conn != nil {
			t.Fatalf("private SMTP destination permitted: %s", target)
		}
	}
}
