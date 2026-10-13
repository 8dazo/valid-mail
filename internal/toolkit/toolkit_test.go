package toolkit

import (
	"encoding/base64"
	"encoding/json"
	"image/png"
	"strings"
	"testing"
	"time"
)

func run(t *testing.T, tool string, values map[string]string) Result {
	t.Helper()
	result, err := Run(Request{Tool: tool, Values: values})
	if err != nil {
		t.Fatal(err)
	}
	return result
}
func TestMarkdownReadmeStructureAndSafety(t *testing.T) {
	input := "# Document\n\n```go\nfmt.Println(\"hello\")\n\nreturn nil\n```\n\n| Name | Value |\n| --- | --- |\n| WeMe | 36 |\n\n- [x] Ready\n\n<details><summary>More</summary><strong>Details</strong></details>\n\n![Logo](images/logo.png)\n\n<script>window.attacked=true</script><svg onload=alert(1)></svg><a href=javascript:alert(1)>unsafe</a>"
	result := run(t, "markdown", map[string]string{"input": input})
	for _, want := range []string{"language-go", "fmt.Println(&#34;hello&#34;)", "\n\nreturn nil", "<table>", "<details>", "<summary>", "type=\"checkbox\"", "Image: Logo"} {
		if !strings.Contains(result.HTML, want) {
			t.Errorf("missing %q in %s", want, result.HTML)
		}
	}
	for _, bad := range []string{"<script", "<svg", "onload", "javascript:", "<img"} {
		if strings.Contains(result.HTML, bad) {
			t.Errorf("unsafe output contains %q", bad)
		}
	}
	if !strings.HasPrefix(result.Text, "<!doctype html>") || !strings.Contains(result.Text, "white-space:pre") {
		t.Fatal("HTML download should include standalone code-block styling")
	}
	enabled := run(t, "markdown", map[string]string{"input": "![Logo](images/logo.png)", "images": "allow", "base": "https://example.test/repo/"})
	if !strings.Contains(enabled.HTML, "https://example.test/repo/images/logo.png") {
		t.Fatal("relative image resolution failed")
	}
	blocked := run(t, "markdown", map[string]string{"input": "![Logo](http://example.test/logo.png)", "images": "allow"})
	if strings.Contains(blocked.HTML, "<img") {
		t.Fatal("HTTP image allowed")
	}
	if _, err := Run(Request{Tool: "markdown", Values: map[string]string{"input": "ok", "base": "javascript:alert(1)"}}); err == nil {
		t.Fatal("unsafe base URL accepted")
	}
}
func TestConfigurationPrecisionAndRoundTrips(t *testing.T) {
	input := `{"server":{"count":9007199254740993,"enabled":true,"names":["one","two"]}}`
	toml := run(t, "toml", map[string]string{"input": input, "mode": "json-toml"})
	if !strings.Contains(toml.Text, "9007199254740993") {
		t.Fatal("integer lost precision")
	}
	converted := run(t, "toml", map[string]string{"input": toml.Text})
	if !strings.Contains(converted.Text, "9007199254740993") {
		t.Fatal("round trip lost integer precision")
	}
	ini := run(t, "ini", map[string]string{"input": "theme = teal\n[server]\nport = 8080"})
	back := run(t, "ini", map[string]string{"input": ini.Text, "mode": "json-ini"})
	again := run(t, "ini", map[string]string{"input": back.Text})
	var initial, final any
	json.Unmarshal([]byte(ini.Text), &initial)
	json.Unmarshal([]byte(again.Text), &final)
	if ini.Text != again.Text {
		t.Fatalf("INI round trip changed values: %s != %s", ini.Text, again.Text)
	}
	for _, request := range []Request{{Tool: "toml", Values: map[string]string{"input": "{\"number\":9223372036854775808}", "mode": "json-toml"}}, {Tool: "toml", Values: map[string]string{"input": "[]", "mode": "json-toml"}}, {Tool: "ini", Values: map[string]string{"input": "{\"x\":null}", "mode": "json-ini"}}, {Tool: "ini", Values: map[string]string{"input": "{\"a\":{\"key\":null}}", "mode": "json-ini"}}} {
		if _, err := Run(request); err == nil {
			t.Fatalf("unsupported configuration accepted: %s", request.Tool)
		}
	}
}
func TestMinifiersAndHTMLConversion(t *testing.T) {
	cases := []struct{ tool, input, want string }{{"html-minify", "<p>  Hello  world </p>", "Hello world"}, {"css-minify", "/* test */ .a { color: #ffffff; margin: 0px; }", ".a{"}, {"js-minify", "function sum(a, b) { return a + b; }", "function sum"}, {"svg-minify", `<svg xmlns="http://www.w3.org/2000/svg"><rect width="10" height="10"/></svg>`, "<svg"}}
	for _, tt := range cases {
		t.Run(tt.tool, func(t *testing.T) {
			result := run(t, tt.tool, map[string]string{"input": tt.input})
			if !strings.Contains(result.Text, tt.want) || result.HTML != "" {
				t.Fatalf("unexpected minifier output: %+v", result)
			}
		})
	}
	converted := run(t, "html-markdown", map[string]string{"input": "<h1>WeMe</h1><p>Some <strong>bold</strong> text.</p>"})
	if !strings.Contains(converted.Text, "# WeMe") || !strings.Contains(converted.Text, "**bold**") {
		t.Fatal("HTML formatting was lost")
	}
}
func TestRegexUnicodeGroupsAndBudget(t *testing.T) {
	input := "👩🏽‍💻 café café"
	result := run(t, "regex", map[string]string{"input": input, "pattern": "(?<word>café)"})
	if len(result.Table.Rows) != 2 || result.Table.Rows[0][1] != "5" || !strings.Contains(result.Table.Rows[0][3], "word") {
		t.Fatalf("wrong Unicode offsets or groups: %+v", result.Table)
	}
	joined := ""
	for _, part := range result.Segments {
		joined += part.Value
	}
	if joined != input {
		t.Fatal("highlight segments corrupt the source")
	}
	start := time.Now()
	_, err := Run(Request{Tool: "regex", Values: map[string]string{"input": strings.Repeat("a", 4000) + "!", "pattern": "(a+)+$"}})
	if err == nil || !strings.Contains(err.Error(), "budget") || time.Since(start) > 2*time.Second {
		t.Fatal("pathological regex did not respect its budget")
	}
	zero := run(t, "regex", map[string]string{"input": "ab", "pattern": "(?=.)"})
	if len(zero.Table.Rows) != 2 {
		t.Fatal("zero-length matches failed")
	}
}
func TestBarcodePNGAndQuietZones(t *testing.T) {
	for _, mode := range []string{"code128", "code39", "ean"} {
		t.Run(mode, func(t *testing.T) {
			input := "WEME-1001"
			if mode == "ean" {
				input = "5901234123457"
			}
			result := run(t, "barcode", map[string]string{"input": input, "mode": mode})
			raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(result.Image, "data:image/png;base64,"))
			if err != nil {
				t.Fatal(err)
			}
			image, err := png.Decode(strings.NewReader(string(raw)))
			if err != nil {
				t.Fatal(err)
			}
			if image.Bounds().Dx() != 600 || image.Bounds().Dy() != 160 {
				t.Fatal("wrong PNG dimensions")
			}
			for _, point := range [][2]int{{0, 80}, {599, 80}, {300, 0}} {
				r, g, b, _ := image.At(point[0], point[1]).RGBA()
				if r != 65535 || g != 65535 || b != 65535 {
					t.Fatal("barcode lacks a white quiet zone")
				}
			}
		})
	}
	if _, err := Run(Request{Tool: "barcode", Values: map[string]string{"input": "123", "mode": "ean"}}); err == nil {
		t.Fatal("invalid EAN accepted")
	}
}
func TestBcryptRoundTripAndLongPasswordRejection(t *testing.T) {
	hash := run(t, "bcrypt", map[string]string{"input": " secret ", "cost": "8"})
	match := run(t, "bcrypt", map[string]string{"input": " secret ", "mode": "verify", "hash": hash.Text})
	if !strings.Contains(match.Text, `"matched": true`) {
		t.Fatal("password whitespace was not preserved")
	}
	mismatch := run(t, "bcrypt", map[string]string{"input": "secret", "mode": "verify", "hash": hash.Text})
	if !strings.Contains(mismatch.Text, `"matched": false`) {
		t.Fatal("different password accepted")
	}
	for _, values := range []map[string]string{{"input": strings.Repeat("é", 37)}, {"input": "password", "cost": "30"}, {"input": "password", "mode": "verify", "hash": "not-a-hash"}} {
		if _, err := Run(Request{Tool: "bcrypt", Values: values}); err == nil {
			t.Fatal("invalid bcrypt parameters accepted")
		}
	}
}
func TestULIDColorAndGraphemes(t *testing.T) {
	ids := run(t, "ulid", map[string]string{"count": "50"})
	seen := map[string]bool{}
	for _, id := range strings.Split(ids.Text, "\n") {
		if len(id) != 26 || seen[id] {
			t.Fatal("invalid or repeated ULID")
		}
		seen[id] = true
		run(t, "ulid", map[string]string{"input": id, "mode": "decode"})
	}
	color := run(t, "color", map[string]string{"input": "#fff"})
	if !strings.Contains(color.Text, "rgb(255, 255, 255)") || color.Swatches[0].Color != "#ffffff" {
		t.Fatal("white color conversion failed")
	}
	text := run(t, "text-stats", map[string]string{"input": "👩🏽‍💻"})
	if !strings.Contains(text.Text, `"graphemes": 1`) {
		t.Fatal("combined emoji counted incorrectly")
	}
	if _, err := Run(Request{Tool: "ulid", Values: map[string]string{"input": "ZZZZZZZZZZZZZZZZZZZZZZZZZZ", "mode": "decode"}}); err == nil {
		t.Fatal("ULID overflow accepted")
	}
}
func TestCronTimezoneAndInvalidSchedule(t *testing.T) {
	result := run(t, "cron-next", map[string]string{"input": "0 9 * * 1-5", "timezone": "Asia/Kolkata", "start": "2026-01-01T00:00:00Z"})
	if result.Table.Rows[0][2] != "2026-01-01T03:30:00Z" || len(result.Table.Rows) != 5 {
		t.Fatalf("wrong forecast: %+v", result.Table)
	}
	for _, input := range []string{"invalid", "0 0 31 2 *"} {
		if _, err := Run(Request{Tool: "cron-next", Values: map[string]string{"input": input}}); err == nil {
			t.Fatal("invalid/impossible schedule accepted")
		}
	}
}
func TestCertificateRejectsPrivateKeysAndInvalidInput(t *testing.T) {
	for _, input := range []string{"not a cert", "-----BEGIN PRIVATE KEY-----\nYWJj\n-----END PRIVATE KEY-----"} {
		if _, err := Run(Request{Tool: "certificate", Values: map[string]string{"input": input}}); err == nil {
			t.Fatal("invalid certificate input accepted")
		}
	}
}
