package main

import (
	"context"
	"net"
	"strings"
	"time"

	emailverifier "github.com/AfterShip/email-verifier"
	"golang.org/x/net/idna"
)

// Keep AfterShip's metadata checks while bounding DNS work and honoring null MX.
// Optional mailbox SMTP probing remains the separate confidence-aware layer.
type domainVerifier struct {
	metadata *emailverifier.Verifier
	lookupMX func(context.Context, string) ([]*net.MX, error)
}

func newDomainVerifier() *domainVerifier {
	return &domainVerifier{metadata: emailverifier.NewVerifier().EnableDomainSuggest(), lookupMX: net.DefaultResolver.LookupMX}
}

func (v *domainVerifier) Verify(email string) (*emailverifier.Result, error) {
	result := &emailverifier.Result{Email: email, Reachable: "unknown", Syntax: v.metadata.ParseAddress(email)}
	if !result.Syntax.Valid {
		return result, nil
	}
	result.Free = v.metadata.IsFreeDomain(result.Syntax.Domain)
	result.RoleAccount = v.metadata.IsRoleAccount(result.Syntax.Username)
	result.Disposable = v.metadata.IsDisposable(result.Syntax.Domain)
	result.Suggestion = v.metadata.SuggestDomain(result.Syntax.Domain)
	if result.Disposable {
		return result, nil
	}
	domain, err := idna.Lookup.ToASCII(result.Syntax.Domain)
	if err != nil {
		return result, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	mx, err := v.lookupMX(ctx, domain)
	if err != nil {
		return result, err
	}
	for _, record := range mx {
		if strings.Trim(record.Host, ". ") != "" {
			result.HasMxRecords = true
			break
		}
	}
	return result, nil
}
