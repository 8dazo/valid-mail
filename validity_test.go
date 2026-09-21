package main

import (
	"testing"

	emailverifier "github.com/AfterShip/email-verifier"
)

func TestAssessValidity(t *testing.T) {
	tests := []struct {
		name   string
		result *emailverifier.Result
		status string
		reason string
	}{
		{
			name: "bad syntax",
			result: &emailverifier.Result{
				Syntax: emailverifier.Syntax{Valid: false},
			},
			status: "invalid",
			reason: "bad_syntax",
		},
		{
			name: "disposable is risky not no mx",
			result: &emailverifier.Result{
				Syntax:     emailverifier.Syntax{Valid: true},
				Disposable: true,
			},
			status: "risky",
			reason: "disposable_domain",
		},
		{
			name: "missing mx",
			result: &emailverifier.Result{
				Syntax: emailverifier.Syntax{Valid: true},
			},
			status: "invalid",
			reason: "no_mx",
		},
		{
			name: "possible typo",
			result: &emailverifier.Result{
				Syntax:       emailverifier.Syntax{Valid: true},
				HasMxRecords: true,
				Suggestion:   "gmail.com",
			},
			status: "risky",
			reason: "possible_domain_typo",
		},
		{
			name: "valid address and domain",
			result: &emailverifier.Result{
				Syntax:       emailverifier.Syntax{Valid: true},
				HasMxRecords: true,
				Free:         true,
			},
			status: "valid",
			reason: "syntax_and_domain_valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := assessValidity(tt.result)
			if got == nil {
				t.Fatal("assessValidity returned nil")
			}
			if got.Status != tt.status {
				t.Fatalf("status = %q, want %q", got.Status, tt.status)
			}
			if got.Reason != tt.reason {
				t.Fatalf("reason = %q, want %q", got.Reason, tt.reason)
			}
		})
	}
}
