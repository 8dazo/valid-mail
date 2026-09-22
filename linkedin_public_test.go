package main

import "testing"

func TestNormalizeLinkedInProfileURL(t *testing.T) {
	got, err := normalizeLinkedInProfileURL("linkedin.com/in/jane-doe-123/?trk=foo#bar")
	if err != nil {
		t.Fatal(err)
	}
	want := "https://www.linkedin.com/in/jane-doe-123/"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}

	for _, bad := range []string{"https://example.com/in/jane", "https://www.linkedin.com/company/openai", "http://www.linkedin.com/in/jane"} {
		if _, err := normalizeLinkedInProfileURL(bad); err == nil {
			t.Fatalf("expected error for %q", bad)
		}
	}
}

func TestExtractLinkedInPublicMetadata(t *testing.T) {
	html := `<!doctype html><html><head>
	<meta property="og:title" content="Jane Doe | LinkedIn">
	<meta property="og:description" content="Staff Engineer at Acme Labs">
	<script type="application/ld+json">{"@type":"Person","name":"Jane Doe","jobTitle":"Staff Engineer","worksFor":{"@type":"Organization","name":"Acme Labs"}}</script>
	</head></html>`

	meta := extractMetaValues(html)
	if meta["og:title"] != "Jane Doe | LinkedIn" {
		t.Fatalf("unexpected title %q", meta["og:title"])
	}
	name, headline, company := extractLinkedInJSONLD(html)
	if name != "Jane Doe" || headline != "Staff Engineer" || company != "Acme Labs" {
		t.Fatalf("unexpected JSON-LD values: %q %q %q", name, headline, company)
	}
	if got := cleanLinkedInName(meta["og:title"]); got != "Jane Doe" {
		t.Fatalf("clean name = %q", got)
	}
	if got := companyFromHeadline(meta["og:description"]); got != "Acme Labs" {
		t.Fatalf("headline company = %q", got)
	}
}
