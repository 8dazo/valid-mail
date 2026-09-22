package main

import "testing"

func TestDecodeSearchResultURL(t *testing.T) {
	direct := "https://www.linkedin.com/in/jane-smith/"
	if got := decodeSearchResultURL(direct); got != direct {
		t.Fatalf("direct = %q", got)
	}

	redirected := "https://duckduckgo.com/l/?uddg=https%3A%2F%2Fwww.linkedin.com%2Fin%2Fjane-smith%2F"
	if got := decodeSearchResultURL(redirected); got != direct {
		t.Fatalf("redirected = %q", got)
	}
}

func TestNameMatchScore(t *testing.T) {
	cases := []struct {
		wanted, candidate string
		min int
	}{
		{"Jane Smith", "Jane Smith", 90},
		{"Jane Smith", "Jane Smith - LinkedIn", 84},
		{"Jane Smith", "Jane A Smith", 80},
		{"Jane Smith", "John Smith", 50},
	}
	for _, tc := range cases {
		if got := nameMatchScore(tc.wanted, tc.candidate); got < tc.min {
			t.Fatalf("score(%q,%q)=%d want >=%d", tc.wanted, tc.candidate, got, tc.min)
		}
	}
}

func TestScoreDiscoveredProfile(t *testing.T) {
	signal := &linkedInPublicSignal{
		URL: "https://www.linkedin.com/in/jane-smith/",
		Status: "fetched",
		Name: "Jane Smith",
		Headline: "Engineer at Acme",
		Company: "Acme",
	}
	profile := scoreDiscoveredProfile("Jane Smith", publicSearchHit{URL: signal.URL, Text: "Jane Smith - LinkedIn"}, signal)
	if profile.Score < 90 {
		t.Fatalf("score=%d", profile.Score)
	}
	if profile.Company != "Acme" {
		t.Fatalf("company=%q", profile.Company)
	}
}
