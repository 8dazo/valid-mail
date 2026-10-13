// Package toolkit wraps reusable Go libraries for local browser and CLI tools.
package toolkit

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type Request struct {
	Tool   string            `json:"tool"`
	Values map[string]string `json:"values"`
}
type Fact struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Tone  string `json:"tone,omitempty"`
}
type Table struct {
	Columns []string   `json:"columns"`
	Rows    [][]string `json:"rows"`
}
type Swatch struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}
type Segment struct {
	Value   string `json:"value"`
	Matched bool   `json:"matched"`
}
type Diff struct {
	Value   string `json:"value"`
	Added   bool   `json:"added,omitempty"`
	Removed bool   `json:"removed,omitempty"`
}
type Binary struct {
	Base64   string `json:"base64"`
	Mime     string `json:"mime"`
	Filename string `json:"filename"`
}
type Result struct {
	Diff     []Diff    `json:"diff,omitempty"`
	Binary   *Binary   `json:"binary,omitempty"`
	Text     string    `json:"text"`
	Title    string    `json:"title,omitempty"`
	Filename string    `json:"filename,omitempty"`
	Mime     string    `json:"mime,omitempty"`
	HTML     string    `json:"html,omitempty"`
	Image    string    `json:"image,omitempty"`
	ImageAlt string    `json:"imageAlt,omitempty"`
	Language string    `json:"language,omitempty"`
	Facts    []Fact    `json:"facts,omitempty"`
	Table    *Table    `json:"table,omitempty"`
	Swatches []Swatch  `json:"swatches,omitempty"`
	Segments []Segment `json:"segments,omitempty"`
	Notice   string    `json:"notice,omitempty"`
}
type Response struct {
	Result *Result `json:"result,omitempty"`
	Error  string  `json:"error,omitempty"`
}

func Run(request Request) (result Result, err error) {
	defer func() {
		if len(result.Text) > 2_000_000 {
			result = Result{}
			err = errors.New("Output exceeds the 2 MB budget. Try a smaller input.")
		}
		if recover() != nil {
			result = Result{}
			err = errors.New("This input could not be processed safely. Try a smaller or simpler example.")
		}
	}()
	total := 0
	for _, value := range request.Values {
		total += len(value)
		if len(value) > 400_000 || total > 600_000 {
			return result, errors.New("Keep each input under 100,000 characters and the total under 600 KB.")
		}
	}
	switch request.Tool {
	case "json":
		return formatJSON(request.Values)
	case "yaml":
		return convertYAML(request.Values)
	case "xml":
		return convertXML(request.Values)
	case "csv":
		return convertCSV(request.Values)
	case "phone":
		return inspectPhone(request.Values)
	case "url":
		return inspectURL(request.Values)
	case "ip":
		return inspectIP(request.Values)
	case "jwt":
		return inspectJWT(request.Values)
	case "cron":
		return explainCron(request.Values)
	case "qr":
		return generateQR(request.Values)
	case "diff":
		return compareText(request.Values)
	case "id":
		return generateIDs(request.Values)
	case "password":
		return generatePasswords(request.Values)
	case "hash":
		return hashText(request.Values)
	case "base64":
		return handleBase64(request.Values)
	case "timestamp":
		return convertTimestamp(request.Values)
	case "case":
		return convertCase(request.Values)
	case "url-codec":
		return handleURLCodec(request.Values)
	case "hmac":
		return hmacText(request.Values)
	case "gzip":
		return handleGzip(request.Values)
	case "chmod":
		return chmodPermissions(request.Values)
	case "markdown":
		return renderMarkdown(request.Values)
	case "toml":
		return convertTOML(request.Values)
	case "ini":
		return convertINI(request.Values)
	case "html-minify", "css-minify", "js-minify", "svg-minify":
		return minifyText(request.Tool, request.Values)
	case "html-markdown":
		return convertHTML(request.Values)
	case "regex":
		return testRegex(request.Values)
	case "barcode":
		return generateBarcode(request.Values)
	case "color":
		return convertColor(request.Values)
	case "ulid":
		return handleULID(request.Values)
	case "bcrypt":
		return handleBcrypt(request.Values)
	case "text-stats":
		return textStats(request.Values)
	case "certificate":
		return inspectCertificate(request.Values)
	case "cron-next":
		return cronNext(request.Values)
	default:
		return result, errors.New("This Go tool is not available.")
	}
}
func Execute(request Request) Response {
	result, err := Run(request)
	if err != nil {
		return Response{Error: err.Error()}
	}
	return Response{Result: &result}
}
func number(values map[string]string, key string, fallback, min, max int) (int, error) {
	value := values[key]
	if value == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(value)
	if err != nil || n < min || n > max {
		return 0, fmt.Errorf("%s must be a whole number between %d and %d", key, min, max)
	}
	return n, nil
}
func jsonResult(value any, title, filename string) (Result, error) {
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return Result{}, errors.New("Some values have no valid JSON representation.")
	}
	return Result{Text: string(body), Title: title, Filename: filename, Mime: "application/json", Language: "json"}, nil
}
func decodeJSON(input string) (any, error) {
	decoder := json.NewDecoder(strings.NewReader(input))
	decoder.UseNumber()
	var data any
	if err := decoder.Decode(&data); err != nil {
		return nil, fmt.Errorf("Invalid JSON: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, errors.New("Provide one JSON value, without trailing content.")
	}
	return normalizeNumbers(data)
}
func normalizeNumbers(value any) (any, error) {
	switch v := value.(type) {
	case json.Number:
		if !strings.ContainsAny(string(v), ".eE") {
			n, err := v.Int64()
			if err != nil {
				return nil, errors.New("Use integers within the signed 64-bit range for this conversion.")
			}
			return n, nil
		}
		return v.Float64()
	case map[string]any:
		for key, item := range v {
			n, err := normalizeNumbers(item)
			if err != nil {
				return nil, err
			}
			v[key] = n
		}
	case []any:
		for i, item := range v {
			n, err := normalizeNumbers(item)
			if err != nil {
				return nil, err
			}
			v[i] = n
		}
	}
	return value, nil
}
func nonEmpty(input string) error {
	if strings.TrimSpace(input) == "" {
		return errors.New("Add your input first, or try the example.")
	}
	return nil
}
