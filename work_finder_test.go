package main

import "testing"

func TestCompanyDomainRoots(t *testing.T) {
	got := companyDomainRoots("Acme Labs Pvt Ltd")
	wantFirst := "acmelabs"
	if len(got) == 0 || got[0] != wantFirst {
		t.Fatalf("companyDomainRoots first = %v, want %q", got, wantFirst)
	}
	if !containsString(got, "acme-labs") {
		t.Fatalf("expected hyphenated root in %v", got)
	}
	if !containsString(got, "acme") {
		t.Fatalf("expected first-word root in %v", got)
	}
}

func TestCompactComparable(t *testing.T) {
	got := compactComparable("OpenAI, Inc.")
	if got != "openaiinc" {
		t.Fatalf("compactComparable = %q", got)
	}
}

func TestCompanyTokenCoverage(t *testing.T) {
	page := `<html><title>Acme Labs</title><body>Welcome to ACME Labs.</body></html>`
	got := companyTokenCoverage("Acme Labs Pvt Ltd", page)
	if got != 1 {
		t.Fatalf("companyTokenCoverage = %v, want 1", got)
	}
}
