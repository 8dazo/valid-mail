package main

import (
	"net"
	"testing"
)

func TestNormalizePublicDomain(t *testing.T) {
	tests := []struct {
		input string
		want  string
		ok    bool
	}{
		{"example.com", "example.com", true},
		{"https://www.example.com/team", "example.com", true},
		{"sub.example.com", "sub.example.com", true},
		{"localhost", "", false},
		{"127.0.0.1", "", false},
		{"bad_domain.com", "", false},
	}
	for _, tt := range tests {
		got, err := normalizePublicDomain(tt.input)
		if tt.ok && err != nil {
			t.Fatalf("normalizePublicDomain(%q) error = %v", tt.input, err)
		}
		if !tt.ok && err == nil {
			t.Fatalf("normalizePublicDomain(%q) expected error", tt.input)
		}
		if got != tt.want {
			t.Fatalf("normalizePublicDomain(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExtractPublicEmails(t *testing.T) {
	body := `
		<a href="mailto:alice.smith@example.com">Alice</a>
		bob [at] example [dot] com
		third@other.com
		alice.smith@example.com
	`
	got := extractPublicEmails(body, "example.com")
	if len(got) != 2 {
		t.Fatalf("got %d emails: %#v", len(got), got)
	}
	if got[0] != "alice.smith@example.com" || got[1] != "bob@example.com" {
		t.Fatalf("unexpected emails: %#v", got)
	}
}

func TestInferObservedPattern(t *testing.T) {
	emails := []string{
		"alice.smith@example.com",
		"bob.jones@example.com",
		"carol.white@example.com",
		"support@example.com",
	}
	pattern, confidence := inferObservedPattern(emails, "example.com")
	if pattern != "first.last" {
		t.Fatalf("pattern = %q, want first.last", pattern)
	}
	if confidence < 80 {
		t.Fatalf("confidence = %d, want >= 80", confidence)
	}
}

func TestPublicIPGuard(t *testing.T) {
	private := []string{"127.0.0.1", "10.0.0.1", "169.254.169.254", "192.168.1.1", "100.64.0.1", "::1"}
	for _, raw := range private {
		if isPublicIP(net.ParseIP(raw)) {
			t.Fatalf("expected %s to be blocked", raw)
		}
	}
	if !isPublicIP(net.ParseIP("8.8.8.8")) {
		t.Fatal("expected 8.8.8.8 to be public")
	}
}

func TestApplyEmailPattern(t *testing.T) {
	if got := applyEmailPattern("first.last", "alice", "smith", "example.com"); got != "alice.smith@example.com" {
		t.Fatalf("got %q", got)
	}
	if got := applyEmailPattern("flast", "alice", "smith", "example.com"); got != "asmith@example.com" {
		t.Fatalf("got %q", got)
	}
}
