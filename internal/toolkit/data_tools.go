package toolkit

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"regexp"
	"sort"
	"strings"

	"github.com/clbanning/mxj/v2"
	jsonrepair "github.com/kaptinlin/jsonrepair"
	"github.com/tidwall/pretty"
	yaml "go.yaml.in/yaml/v3"
)

func formatJSON(values map[string]string) (Result, error) {
	input := []byte(values["input"])
	if err := nonEmpty(string(input)); err != nil {
		return Result{}, err
	}
	repaired := false
	if values["mode"] == "repair" && !json.Valid(input) {
		output, err := jsonrepair.Repair(string(input))
		if err != nil {
			return Result{}, fmt.Errorf("Could not repair this JSON: %w", err)
		}
		input = []byte(output)
		repaired = true
	}
	if !json.Valid(input) {
		return Result{}, errors.New("Invalid JSON. Check quotes and commas, or choose repair mode.")
	}
	var output []byte
	if values["mode"] == "minify" {
		output = pretty.Ugly(input)
	} else {
		output = pretty.PrettyOptions(input, &pretty.Options{Indent: "  ", Width: 0})
	}
	result := Result{Text: strings.TrimRight(string(output), "\n"), Title: "JSON, ready to use", Filename: "formatted.json", Mime: "application/json", Language: "json"}
	if repaired {
		result.Notice = "Review repaired output before using it. Repairs can change how your input is interpreted."
	}
	return result, nil
}
func convertYAML(values map[string]string) (Result, error) {
	input := values["input"]
	if err := nonEmpty(input); err != nil {
		return Result{}, err
	}
	if values["mode"] == "json-yaml" {
		data, err := decodeJSONRaw(input)
		if err != nil {
			return Result{}, err
		}
		node, err := jsonYAMLNode(data)
		if err != nil {
			return Result{}, err
		}
		var buffer bytes.Buffer
		encoder := yaml.NewEncoder(&buffer)
		encoder.SetIndent(2)
		if err := encoder.Encode(node); err != nil {
			return Result{}, err
		}
		encoder.Close()
		return Result{Text: buffer.String(), Filename: "converted.yaml", Mime: "application/yaml", Language: "yaml", Notice: "Comments and original YAML formatting are not retained by conversion."}, nil
	}
	decoder := yaml.NewDecoder(strings.NewReader(input))
	var document yaml.Node
	if err := decoder.Decode(&document); err != nil {
		return Result{}, fmt.Errorf("Invalid YAML: %w", err)
	}
	if err := yamlNodeBudget(&document); err != nil {
		return Result{}, err
	}
	var data any
	if err := document.Decode(&data); err != nil {
		return Result{}, fmt.Errorf("Invalid YAML: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return Result{}, errors.New("Convert one YAML document at a time.")
	}
	normalized, err := jsonCompatibleYAML(data, 0)
	if err != nil {
		return Result{}, err
	}
	return jsonResult(normalized, "Ready as JSON", "converted.json")
}
func jsonYAMLNode(value any) (*yaml.Node, error) {
	node := &yaml.Node{}
	switch item := value.(type) {
	case json.Number:
		node.Kind = yaml.ScalarNode
		node.Value = item.String()
		node.Tag = "!!int"
		if strings.ContainsAny(item.String(), ".eE") {
			node.Tag = "!!float"
		}
	case map[string]any:
		node.Kind = yaml.MappingNode
		node.Tag = "!!map"
		keys := make([]string, 0, len(item))
		for key := range item {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			child, err := jsonYAMLNode(item[key])
			if err != nil {
				return nil, err
			}
			node.Content = append(node.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, child)
		}
	case []any:
		node.Kind = yaml.SequenceNode
		node.Tag = "!!seq"
		for _, entry := range item {
			child, err := jsonYAMLNode(entry)
			if err != nil {
				return nil, err
			}
			node.Content = append(node.Content, child)
		}
	default:
		if err := node.Encode(item); err != nil {
			return nil, err
		}
	}
	return node, nil
}
func jsonCompatibleYAML(value any, depth int) (any, error) {
	if depth > 100 {
		return nil, errors.New("YAML nesting exceeds the 100-level conversion budget.")
	}
	switch item := value.(type) {
	case map[any]any:
		out := map[string]any{}
		for key, entry := range item {
			str, ok := key.(string)
			if !ok {
				return nil, errors.New("JSON requires string mapping keys. Quote non-string YAML keys before converting.")
			}
			converted, err := jsonCompatibleYAML(entry, depth+1)
			if err != nil {
				return nil, err
			}
			out[str] = converted
		}
		return out, nil
	case map[string]any:
		for key, entry := range item {
			converted, err := jsonCompatibleYAML(entry, depth+1)
			if err != nil {
				return nil, err
			}
			item[key] = converted
		}
	case []any:
		for index, entry := range item {
			converted, err := jsonCompatibleYAML(entry, depth+1)
			if err != nil {
				return nil, err
			}
			item[index] = converted
		}
	case float64:
		if math.IsNaN(item) || math.IsInf(item, 0) {
			return nil, errors.New("JSON cannot represent YAML NaN or infinity values.")
		}
	}
	return value, nil
}

var xmlEntityDeclaration = regexp.MustCompile(`(?i)<!\s*(DOCTYPE|ENTITY)\b`)

func convertXML(values map[string]string) (Result, error) {
	input := values["input"]
	if err := nonEmpty(input); err != nil {
		return Result{}, err
	}
	if values["mode"] == "json-xml" {
		data, err := decodeJSONRaw(input)
		if err != nil {
			return Result{}, err
		}
		root, ok := data.(map[string]any)
		if !ok || len(root) != 1 {
			return Result{}, errors.New("Use a JSON object with one XML root element.")
		}
		output, err := mxj.Map(root).XmlIndent("", "  ")
		if err != nil {
			return Result{}, fmt.Errorf("Could not write XML: %w", err)
		}
		if err := validateXML(output); err != nil {
			return Result{}, err
		}
		return Result{Text: string(output), Filename: "converted.xml", Mime: "application/xml", Language: "xml"}, nil
	}
	if xmlEntityDeclaration.MatchString(input) {
		return Result{}, errors.New("DTD and entity declarations are not supported. Remove them before converting.")
	}
	if err := validateXML([]byte(input)); err != nil {
		return Result{}, err
	}
	data, err := mxj.NewMapXml([]byte(input))
	if err != nil {
		return Result{}, fmt.Errorf("Invalid XML: %w", err)
	}
	result, err := jsonResult(data, "Ready as JSON", "converted.json")
	result.Notice = "XML text values remain strings. Attribute keys use a '-' prefix; mixed text uses '#text'. Comments are not retained."
	return result, err
}
func validateXML(input []byte) error {
	decoder := xml.NewDecoder(bytes.NewReader(input))
	depth, roots := 0, 0
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("Invalid XML: %w", err)
		}
		switch token.(type) {
		case xml.StartElement:
			if depth == 0 {
				roots++
			}
			depth++
			if depth > 100 {
				return errors.New("XML nesting exceeds the 100-level conversion budget.")
			}
		case xml.EndElement:
			depth--
		}
	}
	if roots != 1 {
		return errors.New("XML must contain exactly one root element.")
	}
	return nil
}
func convertCSV(values map[string]string) (Result, error) {
	input := values["input"]
	if err := nonEmpty(input); err != nil {
		return Result{}, err
	}
	if values["mode"] == "json-csv" {
		data, err := decodeJSONRaw(input)
		if err != nil {
			return Result{}, err
		}
		rows, ok := data.([]any)
		if !ok || len(rows) == 0 {
			return Result{}, errors.New("Use a non-empty JSON array of objects.")
		}
		keys := map[string]bool{}
		for _, row := range rows {
			object, ok := row.(map[string]any)
			if !ok {
				return Result{}, errors.New("Use a JSON array of objects.")
			}
			for key := range object {
				keys[key] = true
			}
		}
		columns := make([]string, 0, len(keys))
		for key := range keys {
			columns = append(columns, key)
		}
		sort.Strings(columns)
		if len(columns) > 256 {
			return Result{}, errors.New("Use at most 256 CSV columns.")
		}
		var buffer bytes.Buffer
		writer := csv.NewWriter(&buffer)
		writer.UseCRLF = true
		writer.Write(protectCSV(columns))
		for _, row := range rows {
			object := row.(map[string]any)
			cells := make([]string, len(columns))
			for i, column := range columns {
				switch item := object[column].(type) {
				case nil:
				case string:
					cells[i] = item
				case json.Number:
					cells[i] = item.String()
				default:
					bytes, err := json.Marshal(item)
					if err != nil {
						return Result{}, err
					}
					cells[i] = string(bytes)
				}
			}
			writer.Write(protectCSV(cells))
		}
		writer.Flush()
		if writer.Error() != nil {
			return Result{}, writer.Error()
		}
		return Result{Text: strings.TrimSuffix(buffer.String(), "\r\n"), Filename: "converted.csv", Mime: "text/csv", Language: "text", Notice: "Spreadsheet formula prefixes are escaped with an apostrophe. JSON property names determine the CSV columns."}, nil
	}
	reader := csv.NewReader(strings.NewReader(strings.TrimPrefix(input, "\ufeff")))
	records, err := reader.ReadAll()
	if err != nil {
		return Result{}, fmt.Errorf("CSV could not be parsed: %w", err)
	}
	if len(records) == 0 {
		return Result{}, errors.New("Add a CSV header row first.")
	}
	columns := records[0]
	if len(columns) > 256 {
		return Result{}, errors.New("Use at most 256 CSV columns.")
	}
	seen := map[string]bool{}
	for _, column := range columns {
		if column == "" || seen[column] {
			return Result{}, errors.New("CSV column names must be non-empty and unique.")
		}
		seen[column] = true
	}
	data := []map[string]string{}
	tableRows := [][]string{}
	for _, record := range records[1:] {
		row := map[string]string{}
		for i, column := range columns {
			row[column] = record[i]
		}
		data = append(data, row)
		tableRows = append(tableRows, record)
	}
	result, err := jsonResult(data, "Your data, organized", "converted.json")
	result.Table = &Table{Columns: columns, Rows: tableRows}
	return result, err
}
func protectCSV(values []string) []string {
	out := append([]string(nil), values...)
	for i, value := range out {
		trimmed := strings.TrimLeft(value, " \t\r\n")
		if strings.HasPrefix(value, "\t") || strings.HasPrefix(value, "\r") || (len(trimmed) > 0 && strings.ContainsRune("=+-@", rune(trimmed[0]))) {
			out[i] = "'" + value
		}
	}
	return out
}
func decodeJSONRaw(input string) (any, error) {
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
	return data, nil
}

// Check the parsed graph before expanding aliases, including repeated nested references.
func yamlNodeBudget(root *yaml.Node) error {
	aliases, nodes := 0, 0
	active := map[*yaml.Node]bool{}
	var walk func(*yaml.Node, int) error
	walk = func(node *yaml.Node, depth int) error {
		if node == nil {
			return nil
		}
		nodes++
		if depth > 100 || nodes > 20000 {
			return errors.New("YAML exceeds the nesting or expanded-node budget.")
		}
		if active[node] {
			return errors.New("Cyclic YAML aliases are not supported.")
		}
		active[node] = true
		defer delete(active, node)
		if node.Kind == yaml.AliasNode {
			aliases++
			if aliases > 50 {
				return errors.New("YAML alias expansion exceeds the 50-reference budget.")
			}
			return walk(node.Alias, depth+1)
		}
		if node.Kind == yaml.ScalarNode && (node.Tag == "!!int" || (node.Tag == "!!float" && !strings.ContainsAny(node.Value, ".eE"))) {
			integer, ok := new(big.Int).SetString(strings.ReplaceAll(node.Value, "_", ""), 0)
			if ok && (integer.BitLen() > 64 || (integer.Sign() < 0 && integer.Cmp(new(big.Int).Neg(new(big.Int).Lsh(big.NewInt(1), 63))) < 0)) {
				return errors.New("This YAML integer exceeds the 64-bit conversion range. Quote it as a string to preserve it.")
			}
		}
		for _, child := range node.Content {
			if err := walk(child, depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	return walk(root, 0)
}
