package toolkit

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

func TestJSONKeepsLargeIntegersAndRepairsSyntax(t *testing.T) {
	for _, mode := range []string{"format", "minify", "repair"} {
		result := run(t, "json", map[string]string{"input": `{"id":9007199254740993,"huge":123456789012345678901234567890}`, "mode": mode})
		if !strings.Contains(result.Text, "9007199254740993") || !strings.Contains(result.Text, "123456789012345678901234567890") {
			t.Fatal("formatter rounded an integer")
		}
	}
	repaired := run(t, "json", map[string]string{"input": "{name: 'WeMe',}", "mode": "repair"})
	if !json.Valid([]byte(repaired.Text)) || !strings.Contains(repaired.Notice, "Review") {
		t.Fatal("repair should produce valid JSON and explain changes")
	}
}
func TestRepairedJSONPreservesIntegerTokens(t *testing.T) {
	result := run(t, "json", map[string]string{"input": "{id: 9007199254740993,}", "mode": "repair"})
	if !strings.Contains(result.Text, "9007199254740993") {
		t.Fatal("JSON repair rounded an integer")
	}
}
func TestYAMLGuardAndPrecision(t *testing.T) {
	for _, input := range []string{"a: 9007199254740993", "a: 18446744073709551615"} {
		result := run(t, "yaml", map[string]string{"input": input})
		if !strings.Contains(result.Text, strings.TrimPrefix(input, "a: ")) {
			t.Fatal("YAML integer rounded")
		}
	}
	simple := run(t, "yaml", map[string]string{"input": "base: &base [1, 2]\ncopy: *base"})
	if !strings.Contains(simple.Text, `"copy"`) {
		t.Fatal("bounded aliases should work")
	}
	bomb := "a: &a [x, x, x, x, x, x, x, x, x, x]\nb: &b [*a, *a, *a, *a, *a, *a, *a, *a, *a, *a]\nc: [*b, *b, *b, *b, *b, *b, *b, *b, *b, *b]"
	if _, err := Run(Request{Tool: "yaml", Values: map[string]string{"input": bomb}}); err == nil {
		t.Fatal("expanded alias budget ignored")
	}
	if _, err := Run(Request{Tool: "yaml", Values: map[string]string{"input": "a: 18446744073709551616"}}); err == nil {
		t.Fatal("out-of-range YAML integer should not be rounded silently")
	}
	yaml := run(t, "yaml", map[string]string{"input": `{"id":9007199254740993}`, "mode": "json-yaml"})
	if !strings.Contains(yaml.Text, "9007199254740993") {
		t.Fatal("JSON/YAML precision lost")
	}
}
func TestCSVFormulaAndDuplicateHeaderGuards(t *testing.T) {
	result := run(t, "csv", map[string]string{"input": `[{"name":"  =SUM(1,1)","id":9007199254740993}]`, "mode": "json-csv"})
	if !strings.Contains(result.Text, "'  =SUM") || !strings.Contains(result.Text, "9007199254740993") {
		t.Fatal("CSV formula or integer preservation failed")
	}
	if _, err := Run(Request{Tool: "csv", Values: map[string]string{"input": "name,name\nJane,John"}}); err == nil {
		t.Fatal("duplicate header data loss allowed")
	}
}
func TestXMLBlocksMultipleRootsAndPreservesTextNumbers(t *testing.T) {
	if _, err := Run(Request{Tool: "xml", Values: map[string]string{"input": "<a/><b/>"}}); err == nil {
		t.Fatal("multiple XML roots accepted")
	}
	result := run(t, "xml", map[string]string{"input": "<root><id>000123</id><big>9007199254740993</big></root>"})
	if !strings.Contains(result.Text, `"000123"`) || !strings.Contains(result.Text, `"9007199254740993"`) {
		t.Fatal("XML text was coerced into lossy numbers")
	}
}
func TestGzipRoundTripAndDecompressionLimit(t *testing.T) {
	source := strings.Repeat("Hello, 世界 👩🏽‍💻\n", 20)
	compressed := run(t, "gzip", map[string]string{"input": source})
	if compressed.Binary == nil || compressed.Binary.Mime != "application/gzip" {
		t.Fatal("gzip should offer a binary download")
	}
	decoded := run(t, "gzip", map[string]string{"input": compressed.Text, "mode": "decompress"})
	if decoded.Text != source {
		t.Fatal("gzip text round trip failed")
	}
	raw, _ := base64.StdEncoding.DecodeString(compressed.Binary.Base64)
	reader, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(reader)
	reader.Close()
	if string(body) != source {
		t.Fatal("binary download differs from source")
	}
	var bomb bytes.Buffer
	writer := gzip.NewWriter(&bomb)
	writer.Write(bytes.Repeat([]byte("x"), (1<<20)+1))
	writer.Close()
	if _, err := Run(Request{Tool: "gzip", Values: map[string]string{"input": base64.StdEncoding.EncodeToString(bomb.Bytes()), "mode": "decompress"}}); err == nil {
		t.Fatal("decompression bomb budget ignored")
	}
}
func TestHMACAndPermissionVectors(t *testing.T) {
	result := run(t, "hmac", map[string]string{"input": "The quick brown fox jumps over the lazy dog", "key": "key"})
	if result.Text != "f7bc83f430538424b13298e6aa6fb143ef4d59a14946175997479dbc2d1a3cd8" {
		t.Fatal("HMAC vector failed")
	}
	permissions := run(t, "chmod", map[string]string{"input": "4755"})
	if !strings.Contains(permissions.Text, "rwsr-xr-x") {
		t.Fatal("setuid bit was not explained")
	}
	if _, err := Run(Request{Tool: "chmod", Values: map[string]string{"input": "888"}}); err == nil {
		t.Fatal("invalid octal mode accepted")
	}
}
func TestPasswordRequirementsAndUnicodeCase(t *testing.T) {
	result := run(t, "password", map[string]string{"length": "24", "count": "20"})
	for _, value := range strings.Split(result.Text, "\n") {
		if len(value) != 24 || !strings.ContainsAny(value, "ABCDEFGHJKLMNPQRSTUVWXYZ") || !strings.ContainsAny(value, "abcdefghijkmnopqrstuvwxyz") || !strings.ContainsAny(value, "23456789") || !strings.ContainsAny(value, "!@#$%&*+-=?") {
			t.Fatal("password character requirements not met")
		}
	}
	unicode := run(t, "case", map[string]string{"input": "été brûlant"})
	if !strings.Contains(unicode.Text, "étéBrûlant") {
		t.Fatal("Unicode case conversion is broken")
	}
}
