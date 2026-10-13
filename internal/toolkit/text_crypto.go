package toolkit

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/ettle/strcase"
	"github.com/google/uuid"
	crontext "github.com/lnquy/cron"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/sethvargo/go-password/password"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func explainCron(values map[string]string) (Result, error) {
	input := strings.TrimSpace(values["input"])
	if len(strings.Fields(input)) < 5 || len(strings.Fields(input)) > 7 {
		return Result{}, errors.New("Use a five-, six-, or seven-field cron expression.")
	}
	descriptor, err := crontext.NewDescriptor(crontext.Use24HourTimeFormat(true), crontext.Verbose(true))
	if err != nil {
		return Result{}, err
	}
	description, err := descriptor.ToDescription(input, crontext.Locale_en)
	if err != nil {
		return Result{}, fmt.Errorf("Invalid cron expression: %w", err)
	}
	return Result{Text: description, Title: "Your schedule, explained", Notice: "Timezone and actual execution depend on your scheduler. Use the schedule planner for five-field forecasts."}, nil
}
func compareText(values map[string]string) (Result, error) {
	first, second := values["input"], values["second"]
	if len(first)+len(second) > 20_000 {
		return Result{}, errors.New("Keep compared texts under 20,000 bytes combined.")
	}
	engine := diffmatchpatch.New()
	engine.DiffTimeout = 2 * time.Second
	diffs := engine.DiffCleanupSemantic(engine.DiffMain(first, second, false))
	parts := []Diff{}
	var text strings.Builder
	changed := false
	for _, diff := range diffs {
		part := Diff{Value: diff.Text}
		prefix := ""
		switch diff.Type {
		case diffmatchpatch.DiffInsert:
			part.Added = true
			prefix = "[+] "
			changed = true
		case diffmatchpatch.DiffDelete:
			part.Removed = true
			prefix = "[-] "
			changed = true
		}
		parts = append(parts, part)
		text.WriteString(prefix + diff.Text)
	}
	title := "These texts are identical"
	if changed {
		title = "Here’s what changed"
	}
	return Result{Text: text.String(), Title: title, Diff: parts, Filename: "text-comparison.txt"}, nil
}
func generateIDs(values map[string]string) (Result, error) {
	count, err := number(values, "count", 5, 1, 100)
	if err != nil {
		return Result{}, err
	}
	ids := make([]string, 0, count)
	for i := 0; i < count; i++ {
		var id string
		if values["mode"] == "nanoid" {
			id, err = gonanoid.New()
		} else {
			idValue, uuidErr := uuid.NewRandom()
			id, err = idValue.String(), uuidErr
		}
		if err != nil {
			return Result{}, errors.New("Cryptographic randomness is unavailable.")
		}
		ids = append(ids, id)
	}
	return Result{Text: strings.Join(ids, "\n"), Title: "Fresh identifiers", Filename: "identifiers.txt", Mime: "text/plain"}, nil
}
func generatePasswords(values map[string]string) (Result, error) {
	length, err := number(values, "length", 24, 8, 128)
	if err != nil {
		return Result{}, err
	}
	count, err := number(values, "count", 3, 1, 20)
	if err != nil {
		return Result{}, err
	}
	generator, err := password.NewGenerator(&password.GeneratorInput{LowerLetters: "abcdefghijkmnopqrstuvwxyz", UpperLetters: "ABCDEFGHJKLMNPQRSTUVWXYZ", Digits: "23456789", Symbols: "!@#$%&*+-=?"})
	if err != nil {
		return Result{}, err
	}
	passwords := make([]string, 0, count)
	digits, symbols := max(1, length/6), max(1, length/6)
	for i := 0; i < count; i++ {
		candidate := ""
		for attempt := 0; attempt < 50; attempt++ {
			candidate, err = generator.Generate(length, digits, symbols, false, true)
			if err != nil {
				return Result{}, errors.New("The secure password generator could not run.")
			}
			if strings.ContainsFunc(candidate, unicode.IsUpper) && strings.ContainsFunc(candidate, unicode.IsLower) {
				break
			}
			candidate = ""
		}
		if candidate == "" {
			return Result{}, errors.New("A password meeting every character requirement could not be generated. Retry.")
		}
		passwords = append(passwords, candidate)
	}
	return Result{Text: strings.Join(passwords, "\n"), Title: "Fresh, strong passwords", Filename: "passwords.txt", Notice: "Generated locally with cryptographic randomness and upper/lowercase, digits and symbols. Store these in a password manager; WeMe does not save them."}, nil
}
func hashText(values map[string]string) (Result, error) {
	input := []byte(values["input"])
	if values["mode"] == "sha512" {
		sum := sha512.Sum512(input)
		return Result{Text: hex.EncodeToString(sum[:]), Title: "SHA-512 fingerprint", Filename: "sha512.txt"}, nil
	}
	sum := sha256.Sum256(input)
	return Result{Text: hex.EncodeToString(sum[:]), Title: "SHA-256 fingerprint", Filename: "sha256.txt"}, nil
}
func handleBase64(values map[string]string) (Result, error) {
	input := values["input"]
	if values["mode"] != "decode" {
		return Result{Text: base64.StdEncoding.EncodeToString([]byte(input)), Filename: "encoded-base64.txt"}, nil
	}
	encoded := strings.Join(strings.Fields(input), "")
	var data []byte
	var err error
	if strings.Contains(encoded, "=") {
		data, err = base64.StdEncoding.Strict().DecodeString(encoded)
	} else {
		data, err = base64.RawStdEncoding.Strict().DecodeString(encoded)
	}
	if err != nil {
		return Result{}, errors.New("That is not valid Base64 text.")
	}
	if !utf8.Valid(data) {
		return Result{}, errors.New("Decoded bytes are not valid UTF-8 text. This tool does not decode binary files.")
	}
	return Result{Text: string(data), Filename: "decoded-text.txt"}, nil
}
func handleURLCodec(values map[string]string) (Result, error) {
	if values["mode"] == "decode" {
		text, err := url.PathUnescape(values["input"])
		if err != nil || !utf8.ValidString(text) {
			return Result{}, errors.New("Invalid percent encoding or Unicode sequence. Check the input and try again.")
		}
		return Result{Text: text}, nil
	}
	return Result{Text: strings.ReplaceAll(url.QueryEscape(values["input"]), "+", "%20")}, nil
}
func convertCase(values map[string]string) (Result, error) {
	input := values["input"]
	words := strings.ReplaceAll(strcase.ToSnake(input), "_", " ")
	sentence := strings.ToLower(words)
	if sentence != "" {
		first, size := utf8.DecodeRuneInString(sentence)
		sentence = string(unicode.ToUpper(first)) + sentence[size:]
	}
	data := map[string]string{"camelCase": strcase.ToCamel(input), "PascalCase": strcase.ToPascal(input), "snake_case": strcase.ToSnake(input), "kebab-case": strcase.ToKebab(input), "CONSTANT_CASE": strcase.ToSNAKE(input), "Title Case": cases.Title(language.Und).String(words), "Sentence case": sentence, "lowercase": strings.ToLower(input), "UPPERCASE": strings.ToUpper(input)}
	result, err := jsonResult(data, "Every naming style", "text-cases.json")
	result.Table = &Table{Columns: []string{"Case", "Result"}, Rows: [][]string{}}
	for _, key := range []string{"camelCase", "PascalCase", "snake_case", "kebab-case", "CONSTANT_CASE", "Title Case", "Sentence case", "lowercase", "UPPERCASE"} {
		result.Table.Rows = append(result.Table.Rows, []string{key, data[key]})
	}
	return result, err
}
func numericClaim(value any) (time.Time, error) {
	var numeric float64
	var err error
	switch number := value.(type) {
	case json.Number:
		numeric, err = number.Float64()
	case float64:
		numeric = number
	default:
		return time.Time{}, errors.New("invalid numeric date")
	}
	if err != nil || math.IsInf(numeric, 0) || math.IsNaN(numeric) {
		return time.Time{}, errors.New("invalid numeric date")
	}
	instant := time.UnixMilli(int64(numeric * 1000))
	if instant.Year() < 0 || instant.Year() > 9999 {
		return time.Time{}, errors.New("unsupported numeric date")
	}
	return instant, nil
}
func convertTimestamp(values map[string]string) (Result, error) {
	input := strings.TrimSpace(values["input"])
	var instant time.Time
	numeric, err := strconv.ParseFloat(input, 64)
	if err == nil {
		if values["unit"] != "milliseconds" {
			numeric *= 1000
		}
		if math.IsNaN(numeric) || math.IsInf(numeric, 0) || math.Abs(numeric) > 8.64e15 {
			return Result{}, errors.New("That timestamp is outside the supported date range.")
		}
		instant = time.UnixMilli(int64(numeric))
	} else {
		instant, err = time.Parse(time.RFC3339Nano, input)
		if err != nil {
			return Result{}, errors.New("That date does not exist or needs a timezone. Use an ISO date such as 2026-01-01T00:00:00Z.")
		}
	}
	if instant.Year() < 0 || instant.Year() > 9999 {
		return Result{}, errors.New("That timestamp is outside the supported ISO date range (years 0000–9999).")
	}
	timezone := values["timezone"]
	if timezone == "" || timezone == "local" {
		timezone = values["localTimezone"]
	}
	if timezone == "" {
		timezone = "UTC"
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return Result{}, errors.New("Choose a valid IANA timezone.")
	}
	iso := instant.UTC().Format("2006-01-02T15:04:05.000Z")
	local := instant.In(location).Format("2006-01-02 15:04:05 MST")
	result, err := jsonResult(map[string]any{"unixSeconds": instant.Unix(), "unixMilliseconds": instant.UnixMilli(), "iso": iso, "utc": instant.UTC().Format(time.RFC1123), "local": local, "localTimezone": timezone}, "One instant, every format", "timestamp-report.json")
	result.Facts = []Fact{{Label: "ISO 8601", Value: iso}, {Label: "Unix seconds", Value: fmt.Sprint(instant.Unix())}, {Label: "Unix milliseconds", Value: fmt.Sprint(instant.UnixMilli())}}
	result.Table = &Table{Columns: []string{"Representation", "Value"}, Rows: [][]string{{"UTC", instant.UTC().Format(time.RFC1123)}, {timezone, local}}}
	return result, err
}
func hmacText(values map[string]string) (Result, error) {
	if values["key"] == "" {
		return Result{}, errors.New("Add a secret key first.")
	}
	var constructor func() hash.Hash = sha256.New
	title := "HMAC-SHA256"
	if values["mode"] == "sha512" {
		constructor = sha512.New
		title = "HMAC-SHA512"
	}
	mac := hmac.New(constructor, []byte(values["key"]))
	mac.Write([]byte(values["input"]))
	return Result{Text: hex.EncodeToString(mac.Sum(nil)), Title: title, Filename: "hmac.txt", Notice: "Message and key stay local. HMAC authenticates data with a shared secret; it is not a public-key signature."}, nil
}
