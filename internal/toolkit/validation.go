package toolkit

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/dlclark/regexp2"
	"github.com/rivo/uniseg"
	"github.com/robfig/cron/v3"
	_ "time/tzdata"
)

func testRegex(values map[string]string) (Result, error) {
	pattern := values["pattern"]
	input := values["input"]
	if len(pattern) > 2000 || len(input) > 20_000 {
		return Result{}, errors.New("Keep patterns under 2,000 bytes and tested text under 20,000 bytes.")
	}
	var options regexp2.RegexOptions = regexp2.None
	for _, flag := range values["flags"] {
		switch flag {
		case 'i':
			options |= regexp2.IgnoreCase
		case 'm':
			options |= regexp2.Multiline
		case 's':
			options |= regexp2.Singleline
		case ' ', ',':
		default:
			return Result{}, errors.New("Supported flags are i (ignore case), m (multiline), and s (dot matches newlines).")
		}
	}
	expression, err := regexp2.Compile(pattern, options)
	if err != nil {
		return Result{}, fmt.Errorf("Invalid regular expression: %w", err)
	}
	expression.MatchTimeout = 100 * time.Millisecond
	match, err := expression.FindStringMatch(input)
	runes := []rune(input)
	cursor := 0
	segments := []Segment{}
	rows := [][]string{}
	matches := []map[string]any{}
	for match != nil && len(matches) < 100 && err == nil {
		groups := map[string]string{}
		for _, group := range match.Groups()[1:] {
			groups[group.Name] = group.String()
		}
		groupsJSON, _ := jsonResult(groups, "", "")
		rows = append(rows, []string{match.String(), fmt.Sprint(match.Index), fmt.Sprint(match.Length), groupsJSON.Text})
		matches = append(matches, map[string]any{"value": match.String(), "start": match.Index, "length": match.Length, "groups": groups})
		if match.Index >= cursor {
			segments = append(segments, Segment{Value: string(runes[cursor:match.Index])}, Segment{Value: match.String(), Matched: true})
			cursor = match.Index + match.Length
		}
		match, err = expression.FindNextMatch(match)
	}
	if err != nil {
		return Result{}, errors.New("This pattern exceeded the 100 ms matching budget. Try a simpler expression or shorter text.")
	}
	segments = append(segments, Segment{Value: string(runes[cursor:])})
	result, err := jsonResult(matches, fmt.Sprintf("%d match(es)", len(matches)), "regex-matches.json")
	result.Table = &Table{Columns: []string{"Match", "Start", "Length", "Capture groups"}, Rows: rows}
	result.Segments = segments
	result.Notice = "Uses the regexp2 .NET-compatible engine. Start and length count Unicode characters; results are limited to 100 matches."
	return result, err
}
func textStats(values map[string]string) (Result, error) {
	input := values["input"]
	words := 0
	rest := input
	state := -1
	for len(rest) > 0 {
		var word string
		word, rest, state = uniseg.FirstWordInString(rest, state)
		if strings.ContainsFunc(word, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsNumber(r) }) {
			words++
		}
	}
	lines := 0
	if input != "" {
		lines = strings.Count(input, "\n") + 1
	}
	data := map[string]any{"graphemes": uniseg.GraphemeClusterCount(input), "unicodeCodePoints": utf8.RuneCountInString(input), "bytesUTF8": len(input), "words": words, "lines": lines, "readingSeconds": (words*60 + 199) / 200}
	result, err := jsonResult(data, "A closer look at your text", "text-statistics.json")
	result.Facts = []Fact{{Label: "Visible characters", Value: fmt.Sprint(data["graphemes"])}, {Label: "Words", Value: fmt.Sprint(words)}, {Label: "Lines", Value: fmt.Sprint(lines)}, {Label: "UTF-8 size", Value: fmt.Sprintf("%d bytes", len(input))}}
	result.Table = &Table{Columns: []string{"Measure", "Value"}, Rows: [][]string{{"Unicode code points", fmt.Sprint(data["unicodeCodePoints"])}, {"Reading time at 200 words/min", fmt.Sprintf("%d seconds", data["readingSeconds"])}}}
	result.Notice = "Visible characters use Unicode grapheme boundaries, so combined emoji count as one. Reading time is an estimate."
	return result, err
}
func inspectCertificate(values map[string]string) (Result, error) {
	input := []byte(values["input"])
	certificates := []map[string]any{}
	rows := [][]string{}
	for len(strings.TrimSpace(string(input))) > 0 {
		block, rest := pem.Decode(input)
		if block == nil {
			return Result{}, errors.New("Use a PEM certificate with BEGIN CERTIFICATE and END CERTIFICATE markers.")
		}
		input = rest
		if block.Type != "CERTIFICATE" {
			return Result{}, errors.New("Only public CERTIFICATE blocks are supported. Never paste private keys.")
		}
		certificate, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return Result{}, fmt.Errorf("Invalid X.509 certificate: %w", err)
		}
		fingerprint := sha256.Sum256(certificate.Raw)
		now := time.Now()
		status := "Within validity dates"
		if now.Before(certificate.NotBefore) {
			status = "Not yet valid"
		}
		if now.After(certificate.NotAfter) {
			status = "Expired"
		}
		data := map[string]any{"subject": certificate.Subject.String(), "issuer": certificate.Issuer.String(), "notBefore": certificate.NotBefore.UTC().Format(time.RFC3339), "notAfter": certificate.NotAfter.UTC().Format(time.RFC3339), "dateStatus": status, "serialHex": certificate.SerialNumber.Text(16), "dnsNames": certificate.DNSNames, "isCA": certificate.IsCA, "signatureAlgorithm": certificate.SignatureAlgorithm.String(), "publicKeyAlgorithm": certificate.PublicKeyAlgorithm.String(), "sha256": hex.EncodeToString(fingerprint[:])}
		certificates = append(certificates, data)
		rows = append(rows, []string{fmt.Sprint(len(certificates)), certificate.Subject.String(), certificate.NotAfter.UTC().Format(time.RFC3339), status})
		if len(certificates) > 16 {
			return Result{}, errors.New("Inspect at most 16 certificates at a time.")
		}
	}
	if len(certificates) == 0 {
		return Result{}, errors.New("Add a PEM certificate first.")
	}
	result, err := jsonResult(certificates, fmt.Sprintf("%d certificate(s)", len(certificates)), "certificate-report.json")
	result.Table = &Table{Columns: []string{"Certificate", "Subject", "Expires (UTC)", "Date status"}, Rows: rows}
	result.Notice = "Inspection only: chain trust, hostname matching, revocation and cryptographic signatures have not been verified."
	return result, err
}
func cronNext(values map[string]string) (Result, error) {
	timezone := values["timezone"]
	if timezone == "" {
		timezone = "UTC"
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return Result{}, errors.New("Use a valid IANA timezone, such as UTC or Asia/Kolkata.")
	}
	input := strings.TrimSpace(values["input"])
	if strings.Contains(input, "TZ=") {
		return Result{}, errors.New("Select the timezone separately instead of adding TZ= to the expression.")
	}
	schedule, err := cron.ParseStandard("CRON_TZ=" + timezone + " " + input)
	if err != nil {
		return Result{}, fmt.Errorf("Use a five-field cron expression or standard descriptor: %w", err)
	}
	start := time.Now()
	if values["start"] != "" {
		start, err = time.Parse(time.RFC3339, values["start"])
		if err != nil {
			return Result{}, errors.New("Start time must be an ISO date with a timezone, such as 2026-01-01T00:00:00Z.")
		}
	}
	dates := []string{}
	rows := [][]string{}
	for i := 0; i < 5; i++ {
		next := schedule.Next(start)
		if next.IsZero() {
			return Result{}, errors.New("This schedule has no occurrence within the scheduler's supported horizon.")
		}
		dates = append(dates, next.UTC().Format(time.RFC3339))
		rows = append(rows, []string{fmt.Sprint(i + 1), next.In(location).Format("Mon, 02 Jan 2006 15:04:05 MST"), next.UTC().Format(time.RFC3339)})
		start = next
	}
	result, err := jsonResult(map[string]any{"timezone": timezone, "nextRunsUTC": dates}, "The next five runs", "cron-schedule.json")
	result.Table = &Table{Columns: []string{"Run", "Selected timezone", "UTC"}, Rows: rows}
	result.Notice = "Forecast only; this tool does not execute or register jobs. Daylight-saving behavior follows the selected IANA timezone."
	return result, err
}
