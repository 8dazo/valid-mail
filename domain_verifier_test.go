package main

import (
	"context"
	"net"
	"testing"
)

func TestDomainVerifierRespectsMailRouting(t *testing.T) {
	for _, test := range []struct {
		name    string
		records []*net.MX
		want    bool
	}{
		{"real MX", []*net.MX{{Host: "mail.example.com.", Pref: 10}}, true},
		{"null MX", []*net.MX{{Host: ".", Pref: 0}}, false},
		{"no MX", nil, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			verifier := newDomainVerifier()
			verifier.lookupMX = func(ctx context.Context, domain string) ([]*net.MX, error) {
				if _, ok := ctx.Deadline(); !ok {
					t.Fatal("DNS lookup has no deadline")
				}
				if domain != "example.com" {
					t.Fatalf("unexpected domain %q", domain)
				}
				return test.records, nil
			}
			result, err := verifier.Verify("hello@example.com")
			if err != nil || result.HasMxRecords != test.want {
				t.Fatalf("MX result: %#v, %v", result, err)
			}
		})
	}
}

func TestInvalidSyntaxDoesNotUseDNS(t *testing.T) {
	verifier := newDomainVerifier()
	verifier.lookupMX = func(context.Context, string) ([]*net.MX, error) {
		t.Fatal("DNS used for invalid address")
		return nil, nil
	}
	result, err := verifier.Verify("not an email")
	if err != nil || result.Syntax.Valid {
		t.Fatal("invalid syntax accepted")
	}
}
