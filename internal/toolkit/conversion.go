package toolkit

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/pelletier/go-toml/v2"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
	"github.com/tdewolff/minify/v2/svg"
	ini "gopkg.in/ini.v1"
)

func convertTOML(values map[string]string) (Result, error) {
	input := values["input"]
	if err := nonEmpty(input); err != nil {
		return Result{}, err
	}
	if values["mode"] == "json-toml" {
		data, err := decodeJSON(input)
		if err != nil {
			return Result{}, err
		}
		if _, ok := data.(map[string]any); !ok {
			return Result{}, errors.New("TOML needs a JSON object at the root.")
		}
		output, err := toml.Marshal(data)
		if err != nil {
			return Result{}, fmt.Errorf("Cannot represent this JSON as TOML: %w", err)
		}
		return Result{Text: string(output), Title: "Ready as TOML", Filename: "converted.toml", Mime: "application/toml", Language: "ini"}, nil
	}
	var data map[string]any
	if err := toml.Unmarshal([]byte(input), &data); err != nil {
		return Result{}, fmt.Errorf("Invalid TOML: %w", err)
	}
	return jsonResult(data, "Ready as JSON", "converted.json")
}
func convertINI(values map[string]string) (Result, error) {
	input := values["input"]
	if err := nonEmpty(input); err != nil {
		return Result{}, err
	}
	if values["mode"] == "json-ini" {
		data, err := decodeJSON(input)
		if err != nil {
			return Result{}, err
		}
		sections, ok := data.(map[string]any)
		if !ok {
			return Result{}, errors.New("Use a JSON object whose values are section objects.")
		}
		file := ini.Empty()
		names := make([]string, 0, len(sections))
		for name := range sections {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if strings.ContainsAny(name, "\r\n[]") || name == "" {
				return Result{}, errors.New("Section names cannot contain brackets or line breaks.")
			}
			fields, ok := sections[name].(map[string]any)
			if !ok {
				return Result{}, errors.New("Use section objects, such as {\"DEFAULT\":{\"theme\":\"teal\"}}.")
			}
			section, err := file.NewSection(name)
			if err != nil {
				return Result{}, err
			}
			keys := make([]string, 0, len(fields))
			for key := range fields {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				if key == "" || strings.ContainsAny(key, "\r\n=:") {
					return Result{}, errors.New("INI keys cannot contain line breaks, equals signs or colons.")
				}
				var value string
				switch item := fields[key].(type) {
				case string:
					value = item
				case bool, int64, float64:
					value = fmt.Sprint(item)
				default:
					return Result{}, errors.New("INI values must be strings, numbers or booleans; nested objects, arrays and null are not supported.")
				}
				if strings.ContainsAny(value, "\r\n") {
					return Result{}, errors.New("Use single-line INI values.")
				}
				if _, err := section.NewKey(key, value); err != nil {
					return Result{}, err
				}
			}
		}
		var output bytes.Buffer
		if _, err := file.WriteTo(&output); err != nil {
			return Result{}, err
		}
		return Result{Text: output.String(), Title: "Ready as INI", Filename: "converted.ini", Mime: "text/plain", Language: "ini", Notice: "INI stores values as strings. It cannot preserve JSON types."}, nil
	}
	file, err := ini.LoadSources(ini.LoadOptions{IgnoreInlineComment: true, AllowShadows: false}, []byte(input))
	if err != nil {
		return Result{}, fmt.Errorf("Invalid INI: %w", err)
	}
	data := map[string]map[string]string{}
	for _, section := range file.Sections() {
		fields := map[string]string{}
		for _, key := range section.Keys() {
			fields[key.Name()] = key.Value()
		}
		data[section.Name()] = fields
	}
	result, err := jsonResult(data, "Ready as JSON", "converted.json")
	result.Notice = "INI values remain strings; section and key names are case-sensitive."
	return result, err
}
func minifyText(tool string, values map[string]string) (Result, error) {
	input := values["input"]
	if err := nonEmpty(input); err != nil {
		return Result{}, err
	}
	m := minify.New()
	m.AddFunc("text/html", html.Minify)
	m.AddFunc("text/css", css.Minify)
	m.AddFunc("application/javascript", js.Minify)
	m.AddFunc("image/svg+xml", svg.Minify)
	mime, language, filename := "text/html", "html", "minified.html"
	switch tool {
	case "css-minify":
		mime, language, filename = "text/css", "css", "minified.css"
	case "js-minify":
		mime, language, filename = "application/javascript", "javascript", "minified.js"
	case "svg-minify":
		mime, language, filename = "image/svg+xml", "xml", "minified.svg"
	}
	output, err := m.String(mime, input)
	if err != nil {
		return Result{}, fmt.Errorf("Could not minify this input: %w", err)
	}
	saved := len(input) - len(output)
	percent := float64(saved) * 100 / float64(len(input))
	return Result{Text: output, Title: "A little lighter", Filename: filename, Mime: mime, Language: language, Facts: []Fact{{Label: "Original", Value: fmt.Sprintf("%d bytes", len(input))}, {Label: "Minified", Value: fmt.Sprintf("%d bytes", len(output))}, {Label: "Saved", Value: fmt.Sprintf("%.1f%%", percent)}}, Notice: "Review minified output before publishing. HTML, JavaScript and SVG are displayed as code, never executed."}, nil
}
func convertHTML(values map[string]string) (Result, error) {
	if err := nonEmpty(values["input"]); err != nil {
		return Result{}, err
	}
	output, err := htmltomarkdown.ConvertString(values["input"])
	if err != nil {
		return Result{}, fmt.Errorf("Could not convert HTML: %w", err)
	}
	return Result{Text: output, Title: "Clean Markdown", Filename: "converted.md", Mime: "text/markdown", Language: "markdown", Notice: "Converted locally. The input HTML is never executed or fetched."}, nil
}
