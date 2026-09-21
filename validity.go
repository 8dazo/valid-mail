package main

import (
	"encoding/json"

	emailverifier "github.com/AfterShip/email-verifier"
)

// validityAssessment is deliberately separate from mailbox reachability.
// It only describes what can be established from syntax/domain metadata and
// never depends on SMTP, proxy availability, or recipient probing.
type validityAssessment struct {
	Status   string         `json:"status"`
	Scope    string         `json:"scope"`
	Reason   string         `json:"reason"`
	Evidence string         `json:"evidence"`
	Checks   validityChecks `json:"checks"`
}

type validityChecks struct {
	SyntaxValid bool   `json:"syntax_valid"`
	HasMX       bool   `json:"has_mx"`
	Disposable  bool   `json:"disposable"`
	RoleAccount bool   `json:"role_account"`
	FreeProvider bool  `json:"free_provider"`
	Suggestion  string `json:"suggestion,omitempty"`
}

func assessValidity(result *emailverifier.Result) *validityAssessment {
	if result == nil {
		return nil
	}

	checks := validityChecks{
		SyntaxValid: result.Syntax.Valid,
		HasMX:       result.HasMxRecords,
		Disposable:  result.Disposable,
		RoleAccount: result.RoleAccount,
		FreeProvider: result.Free,
		Suggestion:  result.Suggestion,
	}

	assessment := &validityAssessment{
		Scope:  "address_and_domain",
		Checks: checks,
	}

	switch {
	case !result.Syntax.Valid:
		assessment.Status = "invalid"
		assessment.Reason = "bad_syntax"
		assessment.Evidence = "The address is not syntactically valid."
	case result.Disposable:
		// The upstream verifier intentionally stops before MX/SMTP checks for
		// known disposable domains, so do not mislabel the missing MX field as
		// a DNS failure.
		assessment.Status = "risky"
		assessment.Reason = "disposable_domain"
		assessment.Evidence = "The address uses a known disposable-email domain. It may receive mail, but it should not be treated as a durable identity."
	case !result.HasMxRecords:
		assessment.Status = "invalid"
		assessment.Reason = "no_mx"
		assessment.Evidence = "The domain does not publish MX records for receiving email."
	case result.Suggestion != "":
		assessment.Status = "risky"
		assessment.Reason = "possible_domain_typo"
		assessment.Evidence = "The address is structurally valid and its domain accepts mail, but the domain resembles a commonly mistyped provider. Review the suggestion before relying on it."
	default:
		assessment.Status = "valid"
		assessment.Reason = "syntax_and_domain_valid"
		assessment.Evidence = "The address syntax is valid and the domain publishes MX records. This validates the address/domain structure, not the existence of the individual mailbox."
	}

	return assessment
}

// MarshalJSON adds the independent validity assessment to the existing API
// response without changing the SMTP/reachability implementation.
func (r verifyResponse) MarshalJSON() ([]byte, error) {
	type responseWire verifyResponse
	return json.Marshal(struct {
		responseWire
		Validity *validityAssessment `json:"validity,omitempty"`
	}{
		responseWire: responseWire(r),
		Validity:     assessValidity(r.Verification),
	})
}
